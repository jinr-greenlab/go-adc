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
	"fmt"
	"io"
	"net"
	"path"
	"strings"
	"time"

	"github.com/google/gopacket"

	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/log"
	"jinr.ru/greenlab/go-adc/pkg/srv"
)

const (
	MStreamPort    = 33301
	WriterChSize   = 100
	FragmentChSize = 100
	InChSize       = 100
)

type MStreamServer struct {
	srv.Server
	api            *ApiServer
	packetSources  map[string]*PacketSource
	writers        map[string]io.Writer
	writerChs      map[string]chan []byte
	writerStateChs map[string]chan string
	fragmentChs    map[string]chan *layers.MStreamFragment
}

func NewMStreamServer(ctx context.Context, cfg *config.Config) (*MStreamServer, error) {
	log.Info("Initializing mstream server with address: %s port: %d", cfg.IP, MStreamPort)

	uaddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", cfg.IP, MStreamPort))
	if err != nil {
		return nil, err
	}

	s := &MStreamServer{
		Server: srv.Server{
			Context: ctx,
			Config:  cfg,
			UDPAddr: uaddr,
			ChOut:   make(chan srv.OutPacket),
		},
		packetSources:  make(map[string]*PacketSource),
		writers:        make(map[string]io.Writer),
		writerChs:      make(map[string]chan []byte),
		writerStateChs: make(map[string]chan string),
		fragmentChs:    make(map[string]chan *layers.MStreamFragment),
	}

	for _, device := range cfg.Devices {
		s.packetSources[device.Name] = NewPacketSource()
		s.writers[device.Name] = io.Discard
		s.writerChs[device.Name] = make(chan []byte, WriterChSize)
		s.writerStateChs[device.Name] = make(chan string)
		s.fragmentChs[device.Name] = make(chan *layers.MStreamFragment, FragmentChSize)
	}

	apiServer, err := NewApiServer(ctx, cfg, s)
	if err != nil {
		return nil, err
	}
	s.api = apiServer

	return s, nil
}

func (s *MStreamServer) Run() error {
	conn, err := net.ListenUDP("udp", s.UDPAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	errChan := make(chan error, 1)
	buffer := make([]byte, 1048576)

	// flush all files before exit
	defer s.Flush()

	// Read packets from wire and put them to input queue
	go func() {
		for {
			length, addr, readErr := conn.ReadFrom(buffer)
			if readErr != nil {
				errChan <- readErr
				return
			}
			udpAddr, readErr := net.ResolveUDPAddr("udp", addr.String())
			if readErr != nil {
				errChan <- readErr
				return
			}
			log.Debug("Received packet from %s", udpAddr)
			ipAddr := net.ParseIP(strings.Split(addr.String(), ":")[0])
			device, getdevErr := s.GetDeviceByIP(ipAddr)
			if getdevErr != nil {
				log.Debug("Drop packet. Device not found for given IP: %s ", ipAddr.String())
				continue
			}

			captureInfo := gopacket.CaptureInfo{
				Length:        length,
				CaptureLength: length,
				Timestamp:     time.Now(),
				AncillaryData: []interface{}{udpAddr, device.Name},
			}
			packet := srv.InPacket{CaptureInfo: captureInfo, Data: make([]byte, length)}
			copy(packet.Data, buffer[:length])
			s.packetSources[device.Name].ChIn <- packet
		}
	}()

	// Read packets from output queue and send them to wire
	go func() {
		for {
			outPacket := <-s.ChOut
			log.Debug("Sending packet to %s data: \n%s", outPacket.UDPAddr, hex.EncodeToString(outPacket.Data))
			_, sendErr := conn.WriteToUDP(outPacket.Data, outPacket.UDPAddr)
			if sendErr != nil {
				log.Error("Error while sending data to %s", outPacket.UDPAddr)
				errChan <- sendErr
				return
			}
		}
	}()

	// Read packets from input queue and handle them properly
	for _, device := range s.Config.Devices {
		deviceName := device.Name
		eventBuilder := NewEventBuilder(
			deviceName,
			s.fragmentChs[deviceName],
			s.writerChs[deviceName],
		)

		// Run mpd writers
		go func() {
			currentFilename := ""
			for {
				select {
				case filename := <-s.writerStateChs[deviceName]:

					if currentFilename != "" {
						w := s.writers[deviceName].(*Writer)
						w.Flush()
					}
					if filename == "" {
						s.writers[deviceName] = io.Discard
					} else {
						w, newWriterErr := NewWriter(filename)
						if newWriterErr != nil {
							log.Error("Error while creating writer: %s", err)
							continue
						}
						s.writers[deviceName] = w
					}
					currentFilename = filename
				default:
				}
				select {
				case bytes := <-s.writerChs[deviceName]:
					_, writeErr := s.writers[deviceName].Write(bytes)
					if writeErr != nil {
						log.Error("Error while writing to file: %s", err)
					}
				default:
					time.Sleep(10 * time.Millisecond)
				}
			}
		}()

		// Run event builders
		go func() {
			eventBuilder.Run()
		}()

		// Run parsers
		go func() {
			fragmentBuilderManager := layers.NewFragmentBuilderManager(deviceName, s.fragmentChs[deviceName])
			fragmentBuilderManager.Init()

			source := gopacket.NewPacketSource(s.packetSources[deviceName], layers.MLinkLayerType)
			for packet := range source.Packets() {
				var mlSeq uint16
				mlinkLayer := packet.Layer(layers.MLinkLayerType)
				if mlinkLayer != nil {
					ml := mlinkLayer.(*layers.MLinkLayer)
					mlSeq = ml.Seq
				}
				mstreamLayer := packet.Layer(layers.MStreamLayerType)
				if mstreamLayer != nil {
					log.Debug("MStream frame successfully parsed")
					ms := mstreamLayer.(*layers.MStreamLayer)

					udpaddr, getAddrErr := srv.GetAddrPort(packet)
					if getAddrErr != nil {
						log.Error("Error while getting udpaddr for a packet from input queue")
						continue
					}

					for _, f := range ms.Fragments {
						log.Debug("Handling fragment: FragmentID: 0x%04x FragmentOffset: 0x%04x LastFragment: %t",
							f.FragmentID, f.FragmentOffset, f.LastFragment())

						ackErr := s.SendAck(0, mlSeq, f.FragmentID, f.FragmentOffset, udpaddr)
						if ackErr != nil {
							log.Error("Error while sending Ack for fragment: ID: %d Offset: %d Length: %d",
								f.FragmentID, f.FragmentOffset, f.FragmentLength)
						}

						if f.Subtype == layers.MStreamTriggerSubtype && !f.LastFragment() {
							log.Error("!!! Something really bad is happening. Trigger data is fragmented.")
							continue
						}
						fragmentBuilderManager.SetFragment(f)
					}
				}
			}
		}()
	}

	go func() {
		s.api.Run()
	}()

	err = s.ConnectToDevices()
	if err != nil {
		return err
	}

	select {
	case <-s.Context.Done():
		return s.Context.Err()
	case err = <-errChan:
		return err
	}
}

func (s *MStreamServer) SendAck(src, seq, fragmentID, fragmentOffset uint16, udpAddr *net.UDPAddr) error {
	ml := &layers.MLinkLayer{}
	ml.Type = layers.MLinkTypeMStream
	ml.Sync = layers.MLinkSync
	// 3 words for MLink header + 1 word CRC + 2 words for MStream header
	ml.Len = 6
	ml.Seq = seq
	// Since this is ACK message SRC and DST are reversed.
	//ml.Src = layers.MLinkDeviceAddr
	ml.Src = src
	ml.Dst = layers.MLinkHostAddr
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
		log.Error("Error while serializing layers when sending MStream ack message to device %s", udpAddr)
		return err
	}

	log.Debug("Put MStream Ack to output queue: udpaddr: %s ack: %s", udpAddr, hex.EncodeToString(buf.Bytes()))
	log.Debug("Put MStream ack to output queue: udpaddr: %s fragment: %d", udpAddr, fragmentID)
	s.ChOut <- srv.OutPacket{
		Data:    buf.Bytes(),
		UDPAddr: udpAddr,
	}
	return nil
}

func (s *MStreamServer) ConnectToDevices() error {
	// to connect to peer devices it is enough to send them an MStream ack
	// message with empty payload and with fragmentID = -1 and fragmentOffset = -1
	for _, device := range s.Config.Devices {
		udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", device.IP, MStreamPort))
		if err != nil {
			return err
		}
		err = s.SendAck(layers.MLinkDeviceAddr, 0, 0xffff, 0xffff, udpAddr)
		if err != nil {
			log.Error("Error while connecting to MStream device %s:%s", device.IP, MStreamPort)
			return err
		}
	}
	return nil
}

func (s *MStreamServer) persistFilename(dir, prefix, name, suffix string) string {
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
	timestamp := time.Now().UTC().Format("20060102_150405")
	for _, device := range s.Config.Devices {
		log.Info("Persist writer: %s", device.Name)
		filename := s.persistFilename(dir, filePrefix, device.Name, timestamp)
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
