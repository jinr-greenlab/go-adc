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
	"encoding/binary"
	"os"
	"path/filepath"

	"jinr.ru/greenlab/go-adc/pkg/log"
)

const (
	MpdStartRunMagic  = 0x72617453
	MpdStopRunMagic   = 0x706F7453
	MpdRunNumberMagic = 0x236E7552
	MpdRunIndexMagic  = 0x78646E49
)

// MpdStartRunHeader ...
func MpdStartRunHeader() []byte {
	buf := make([]byte, 28)
	binary.LittleEndian.PutUint32(buf[0:4], MpdStartRunMagic)   // start run sync
	binary.LittleEndian.PutUint32(buf[4:8], 0x14)               // start run header length
	binary.LittleEndian.PutUint32(buf[8:12], MpdRunNumberMagic) // run number sync
	binary.LittleEndian.PutUint32(buf[12:16], 0x04)             // run number length
	binary.LittleEndian.PutUint32(buf[16:20], 0)                // run number
	binary.LittleEndian.PutUint32(buf[20:24], MpdRunIndexMagic) // run index sync
	binary.LittleEndian.PutUint32(buf[24:28], 0)                // run index
	return buf
}

// MpdStopRunHeader ...
func MpdStopRunHeader() []byte {
	buf := make([]byte, 28)
	binary.LittleEndian.PutUint32(buf[0:4], MpdStopRunMagic)    // stop run sync
	binary.LittleEndian.PutUint32(buf[4:8], 0x14)               // stop run header length
	binary.LittleEndian.PutUint32(buf[8:12], MpdRunNumberMagic) // run number sync
	binary.LittleEndian.PutUint32(buf[12:16], 0x04)             // run number length
	binary.LittleEndian.PutUint32(buf[16:20], 0)                // run number
	binary.LittleEndian.PutUint32(buf[20:24], MpdRunIndexMagic) // run index sync
	binary.LittleEndian.PutUint32(buf[24:28], 0)                // run index
	return buf
}

type Writer struct {
	file *os.File
}

func NewWriter(filename string) (*Writer, error) {
	dir, _ := filepath.Split(filename)

	if dir != "" {
		err := os.MkdirAll(dir, 0755)
		if err != nil && !os.IsExist(err) {
			log.Error("Error while creating directory: %s", dir)
			return nil, err
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		log.Error("Error while creating file: %s", filename)
		return nil, err
	}
	writer := &Writer{
		file: file,
	}
	writer.file.Write(MpdStartRunHeader())
	return writer, nil
}

func (w *Writer) Write(buf []byte) (int, error) {
	return w.file.Write(buf)
}

func (w *Writer) Flush() {
	w.file.Write(MpdStopRunHeader())
	w.file.Sync()
	w.file.Close()
}
