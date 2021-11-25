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

package discover

import (
	"context"
	"errors"
	"fmt"
	"go.etcd.io/bbolt"
	"sigs.k8s.io/yaml"

	"jinr.ru/greenlab/go-adc/pkg/config"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/log"
)


const (
	BucketPrefix = "discover_"
	DeviceDescriptionKey = "device_description"
)

type State struct {
	context.Context
	DB *bbolt.DB
}

func NewState(ctx context.Context, cfg *config.Config) (*State, error) {
	// open discover database
	db, err := bbolt.Open(cfg.DiscoverDBPath(), 0600, nil)
	if err != nil {
		return nil, err
	}
	return &State{
		Context: ctx,
		DB: db,
	}, nil
}

// Close ...
func (s *State) Close() {
	s.DB.Close()
}

// CreateBucket ...
func (s *State) CreateBucket(name string) error {
	if err := s.DB.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(name))
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func BucketName(deviceName string) string {
	return fmt.Sprintf("%s%s", BucketPrefix, deviceName)
}

//func uint64ToByte(v uint64) []byte {
//	b := make([]byte, 8)
//	binary.BigEndian.PutUint64(b, v)
//	return b
//}

// SetDeviceDescription ...
func (s *State) SetDeviceDescription(dd *layers.DeviceDescription) error {
	log.Debug("Setting device description: device: %s", dd.SerialNumber)
	if err := s.DB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketName(dd.SerialNumber)))
		if b == nil {
			return errors.New(fmt.Sprintf("Bucket not found: %s", BucketName(dd.SerialNumber)))
		}
		ddBytes, err := yaml.Marshal(dd)
		if err != nil {
			return err
		}
		if err := b.Put([]byte(DeviceDescriptionKey), ddBytes); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// GetDeviceDescription ...
func (s *State) GetDeviceDescription(serialNumber string) (string, uint64, error) {
	log.Debug("Getting device description: device: %s", serialNumber)
	var description string
	var timestamp uint64
	if err := s.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketName(serialNumber)))
		if b == nil {
			return errors.New(fmt.Sprintf("Bucket not found: %s", BucketName(serialNumber)))
		}
		ddBytes := b.Get([]byte(DeviceDescriptionKey))
		if ddBytes == nil {
			return errors.New(fmt.Sprintf("Description not found", ))
		}
		var dd *layers.DeviceDescription
		if err := yaml.Unmarshal(ddBytes, dd); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return "", 0, err
	}
	return description, timestamp, nil
}

// GetAllDeviceDescriptions ...
func (s *State) GetAllDeviceDescriptions() ([]*layers.DeviceDescription, error) {
	log.Debug("Getting all device descriptions")
	var devices []*layers.DeviceDescription
	if err := s.DB.View(func(tx *bbolt.Tx) error {
		tx.ForEach(func(_ []byte, b *bbolt.Bucket) error {
			ddBytes := b.Get([]byte(DeviceDescriptionKey))
			if ddBytes == nil {
				return errors.New(fmt.Sprintf("Description not found", ))
			}
			dd := &layers.DeviceDescription{}
			if err := yaml.Unmarshal(ddBytes, dd); err != nil {
				log.Error("Error while unmarshalling DeviceDescription %s\n", err)
				return err
			}
			devices = append(devices, dd)
			return nil
		})
		return nil
	}); err != nil {
		return nil, err
	}
	return devices, nil
}

