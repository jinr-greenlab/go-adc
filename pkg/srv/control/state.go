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

package control

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"jinr.ru/greenlab/go-adc/pkg/layers"
	"jinr.ru/greenlab/go-adc/pkg/srv/control/ifc"

	"go.etcd.io/bbolt"

	"jinr.ru/greenlab/go-adc/pkg/config"
	pkgdevice "jinr.ru/greenlab/go-adc/pkg/device"
	"jinr.ru/greenlab/go-adc/pkg/log"
)

const (
	RegBucketPrefix = "reg_"
)

type State struct {
	context.Context
	DB *bbolt.DB
}

var _ ifc.State = &State{}

func NewState(ctx context.Context, cfg *config.Config) (ifc.State, error) {
	// open register database
	db, err := bbolt.Open(cfg.DBPath(), 0600, nil)
	if err != nil {
		return nil, err
	}
	// create buckets in the register database for all devices
	if err = db.Update(func(tx *bbolt.Tx) error {
		for _, device := range cfg.Devices {
			_, err = tx.CreateBucketIfNotExists([]byte(regBucketName(device.Name)))
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &State{
		Context: ctx,
		DB:      db,
	}, nil
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

func uint16ToByte(v uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, v)
	return b
}

func regBucketName(deviceName string) string {
	return fmt.Sprintf("%s%s", RegBucketPrefix, deviceName)
}

// Close ...
func (s *State) Close() {
	s.DB.Close()
}

// SetReg ...
func (s *State) SetReg(reg *layers.Reg, deviceName string) error {
	log.Debug("Setting register: device: %s Addr: 0x%04x Value: 0x%04x", deviceName, reg.Addr, reg.Value)
	if err := s.DB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(regBucketName(deviceName)))
		if b == nil {
			return errors.New(fmt.Sprintf("Bucket not found: %s", regBucketName(deviceName)))
		}
		if err := b.Put(uint16ToByte(reg.Addr), uint16ToByte(reg.Value)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// GetReg ...
func (s *State) GetReg(addr uint16, deviceName string) (*layers.Reg, error) {
	log.Debug("Getting register: device: %s Addr: %x", deviceName, addr)
	var value uint16
	if err := s.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(regBucketName(deviceName)))
		if b == nil {
			return errors.New(fmt.Sprintf("Bucket not found: %s", regBucketName(deviceName)))
		}
		valueBytes := b.Get(uint16ToByte(addr))
		if valueBytes == nil {
			return errors.New(fmt.Sprintf("Key not found: %d", addr))
		}
		value = binary.BigEndian.Uint16(valueBytes)
		return nil
	}); err != nil {
		return nil, err
	}
	return &layers.Reg{
		Addr:  addr,
		Value: value,
	}, nil
}

// GetRegAll ...
func (s *State) GetRegAll(deviceName string) ([]*layers.Reg, error) {
	log.Debug("Getting all registers: device: %s", deviceName)
	var regs []*layers.Reg
	if err := s.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(regBucketName(deviceName)))
		if b == nil {
			return errors.New(fmt.Sprintf("Bucket not found: %s", regBucketName(deviceName)))
		}
		for _, addr := range pkgdevice.RegMap {
			valueBytes := b.Get(uint16ToByte(addr))
			if valueBytes == nil {
				return errors.New(fmt.Sprintf("Key not found: %d", addr))
			}
			regs = append(regs, &layers.Reg{Addr: addr, Value: binary.BigEndian.Uint16(valueBytes)})
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return regs, nil
}
