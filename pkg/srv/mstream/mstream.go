/*
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     https://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package mstream

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"path"
	"sync"
	"time"

	"github.com/google/gopacket"

	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/log"
	"jinr.ru/greenlab/go-adc/pkg/srv"
)

const (
	DeviceMStreamPort  = 33301
	ServerMStreamPort  = 0
	WriterChSize       = 100
	FragmentedChSize   = 100
	DefragmentedChSize = 100
	InChSize           = 1
)

const (
	InputBufferSize = 262144 // 65536 * 4
)

type MStreamServer struct {
	srv.Server
	api               *ApiServer
	packetDataSources map[string]*PacketSource
	writerChs         map[string]chan []byte
	writerStateChs    map[string]chan string
	fragmentedChs     map[string]chan *layers.MStreamFragment
	defragmentedChs   map[string]chan *layers.MStreamFragment
	lastEventChs      map[string]chan []byte
	lastEvent         map[string][]byte
	mu                sync.RWMutex
}

func NewMStreamServer(ctx context.Context, cfg *config.Config) (*MStreamServer, error) {
	log.Info("Initializing mstream server with address: %s port: %d", cfg.IP, ServerMStreamPort)

	s := &MStreamServer{
		Server: srv.Server{
			Context: ctx,
			Config:  cfg,
		},
		packetDataSources: make(map[string]*PacketSource),
		writerChs:         make(map[string]chan []byte),
		writerStateChs:    make(map[string]chan string),
		fragmentedChs:     make(map[string]chan *layers.MStreamFragment),
		defragmentedChs:   make(map[string]chan *layers.MStreamFragment),
		lastEventChs:      make(map[string]chan []byte),
		lastEvent:         make(map[string][]byte),
		mu:                sync.RWMutex{},
	}

	for _, device := range cfg.Devices {
		s.packetDataSources[device.Name] = NewPacketSource()
		s.writerChs[device.Name] = make(chan []byte, WriterChSize)
		s.writerStateChs[device.Name] = make(chan string)
		s.fragmentedChs[device.Name] = make(chan *layers.MStreamFragment, FragmentedChSize)
		s.defragmentedChs[device.Name] = make(chan *layers.MStreamFragment, DefragmentedChSize)
		s.lastEventChs[device.Name] = make(chan []byte, 1)
	}

	apiServer, err := NewApiServer(ctx, cfg, s)
	if err != nil {
		return nil, err
	}
	s.api = apiServer

	return s, nil
}

func (s *MStreamServer) Run() error {
	errChan := make(chan error, 1)

	if len(s.Config.Devices) == 0 {
		errChan <- errors.New("No devices in config")
	}

	// Read packets from input queue and handle them properly
	for _, device := range s.Config.Devices {
		uaddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", s.Config.IP, ServerMStreamPort))
		if err != nil {
			return err
		}

		conn, errListen := net.ListenUDP("udp", uaddr)
		if errListen != nil {
			return errListen
		}
		log.Info("Device server listening on %s", conn.LocalAddr().String())
		defer func(conn *net.UDPConn) {
			conn.Close()
		}(conn)

		deviceName := device.Name
		udpAddr, errResolve := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", device.IP, DeviceMStreamPort))
		if errResolve != nil {
			return errResolve
		}
		defragManager := NewDefragManager(deviceName, s.fragmentedChs[deviceName], s.defragmentedChs[deviceName])
		eventBuilder := NewEventBuilder(deviceName, s.defragmentedChs[deviceName], s.writerChs[deviceName], s.lastEventChs[deviceName])

		// Run mpd writers
		go func(writerStateCh <-chan string, writerCh <-chan []byte) {
			currentFilename := ""
			writer := io.Discard
			for {
				select {
				case filename := <-writerStateCh:
					if currentFilename != "" {
						w := writer.(*Writer)
						w.Flush()
					}
					if filename == "" {
						writer = io.Discard
					} else {
						w, newWriterErr := NewWriter(filename)
						if newWriterErr != nil {
							log.Error("Error while creating writer: %s", newWriterErr)
							continue
						}
						writer = w
					}
					currentFilename = filename
				default:
				}
				select {
				case bytes := <-writerCh:
					_, writeErr := writer.Write(bytes)
					if writeErr != nil {
						log.Error("Error while writing to file: %s", writeErr)
					}
				default:
					time.Sleep(10 * time.Millisecond)
				}
			}
		}(s.writerStateChs[deviceName], s.writerChs[deviceName])

		// Run event builders
		go func(eventBuilder *EventBuilder) {
			eventBuilder.Run()
		}(eventBuilder)

		// Run defragmenter manager
		go func(defragManager *DefragManager) {
			defragManager.Run()
		}(defragManager)

		counterCh := make(chan int)
		//
		go func(counterCh <-chan int) {
			//counter := 0
			for {
				<-counterCh
				//counter += 1
				//log.Info("Packet counter: %d", counter)
			}
		}(counterCh)

		// Run parsers
		go func(deviceName string, conn *net.UDPConn, udpAddr *net.UDPAddr, fragmentedCh chan<- *layers.MStreamFragment, counterCh chan<- int) {
			buffer := make([]byte, InputBufferSize)
			decodeOptions := gopacket.DecodeOptions{
				Lazy:   false,
				NoCopy: true,
			}
			for {
				length, _, readErr := conn.ReadFromUDP(buffer)
				if readErr != nil {
					errChan <- readErr
					return
				}
				counterCh <- 1

				data := make([]byte, length)
				copy(data, buffer[:length])

				packet := gopacket.NewPacket(data, layers.MLinkLayerType, decodeOptions)

				var mlSeq uint16
				var mlSrc uint16
				var mlDst uint16
				mlinkLayer := packet.Layer(layers.MLinkLayerType)
				if mlinkLayer != nil {
					ml := mlinkLayer.(*layers.MLinkLayer)
					mlSeq = ml.Seq
					mlSrc = ml.Src
					mlDst = ml.Dst
				}
				mstreamLayer := packet.Layer(layers.MStreamLayerType)
				if mstreamLayer != nil {
					log.Debug("MStream frame successfully parsed")
					ms := mstreamLayer.(*layers.MStreamLayer)

					for _, f := range ms.Fragments {
						log.Debug("Handling fragment: FragmentID: 0x%04x FragmentOffset: 0x%04x LastFragment: %t",
							f.FragmentID, f.FragmentOffset, f.LastFragment())

						fragmentedCh <- f

						ackErr := SendAck(mlDst, mlSrc, mlSeq, f.FragmentID, f.FragmentOffset, udpAddr, conn)
						if ackErr != nil {
							log.Error("Error while sending Ack: udpAddr: %s fragment: ID: %d Offset: %d Length: %d",
								udpAddr, f.FragmentID, f.FragmentOffset, f.FragmentLength)
						}
					}
				}
			}
		}(deviceName, conn, udpAddr, s.fragmentedChs[deviceName], counterCh)

		errAck := SendAck(layers.MLinkDeviceAddr, 1, 0, 0xffff, 0xffff, udpAddr, conn)
		if errAck != nil {
			log.Error("Error while connecting to MStream device: udpAddr: %s", udpAddr)
			return errAck
		}
	}

	go func() {
		s.api.Run()
	}()

	select {
	case <-s.Context.Done():
		return s.Context.Err()
	case err := <-errChan:
		return err
	}
}

func SendAck(mlSrc, mlDst, mlSeq, fragmentID, fragmentOffset uint16, udpAddr *net.UDPAddr, conn *net.UDPConn) error {
	ml := &layers.MLinkLayer{}
	ml.Type = layers.MLinkTypeMStream
	ml.Sync = layers.MLinkSync
	// 3 words for MLink header + 1 word CRC + 2 words for MStream header
	ml.Len = 6
	ml.Seq = mlSeq
	// Since this is ACK message SRC and DST must be reversed.
	ml.Src = mlSrc
	ml.Dst = mlDst
	ml.Crc = 0

	ms := &layers.MStreamLayer{
		Fragments: []*layers.MStreamFragment{
			{
				DeviceID:       1,
				Subtype:        0,
				Flags:          0b00010000,
				FragmentLength: 0,
				FragmentID:     fragmentID,
				FragmentOffset: fragmentOffset,
				Data:           []byte{},
			},
		},
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	err := gopacket.SerializeLayers(buf, opts, ml, ms)
	if err != nil {
		log.Error("Error while serializing layers when sending MStream ack: udpAddr: %s", udpAddr)
		return err
	}

	log.Debug("Send MStream Ack: udpAddr: %s ack: %s", udpAddr, hex.EncodeToString(buf.Bytes()))
	_, sendErr := conn.WriteToUDP(buf.Bytes(), udpAddr)
	if sendErr != nil {
		return sendErr
	}
	return nil
}

func PersistFilename(dir, prefix, name, suffix string) string {
	filename := fmt.Sprintf("%s_%s.data", name, suffix)
	if prefix != "" {
		filename = fmt.Sprintf("%s_%s", prefix, filename)
	}
	return path.Join(dir, filename)
}

func (s *MStreamServer) Flush() {
	for _, device := range s.Config.Devices {
		log.Info("Flush writer: %s", device.Name)
		s.writerStateChs[device.Name] <- ""
	}
}

func (s *MStreamServer) Persist(dir, filePrefix string) {
	timestamp := time.Now().In(time.Local).Format("20060102_150405")
	for _, device := range s.Config.Devices {
		log.Info("Persist writer: %s", device.Name)
		filename := PersistFilename(dir, filePrefix, device.Name, timestamp)
		s.writerStateChs[device.Name] <- filename
	}
}

type PacketSource struct {
	ChIn chan srv.InPacket
}

func NewPacketSource() *PacketSource {
	return &PacketSource{
		ChIn: make(chan srv.InPacket, InChSize),
	}
}

func (ps *PacketSource) ReadPacketData() ([]byte, gopacket.CaptureInfo, error) {
	p := <-ps.ChIn
	return p.Data, p.CaptureInfo, nil
}
