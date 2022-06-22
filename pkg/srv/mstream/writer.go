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
	"os"

	"jinr.ru/greenlab/go-adc/pkg/log"
)

type Writer struct {
	file *os.File
}

func NewWriter(filename string) (*Writer, error) {
	file, err := os.Create(filename)
	if err != nil {
		log.Error("Error while creating file: %s", filename)
		return nil, err
	}
	return &Writer{
		file: file,
	}, nil
}

func (w *Writer) Write(buf []byte) (int, error) {
	return w.file.Write(buf)
}

func (w *Writer) Flush() {
	w.file.Sync()
	w.file.Close()
}
