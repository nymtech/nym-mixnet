// Copyright 2018-2019 The Loopix-Messaging Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
	Package helpers implements all useful functions which are used in the code of anonymous messaging system.
*/

package helpers

import (
	"encoding"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// DirExists checks whether a directory exists at the given path.
func DirExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err == nil {
		return true, nil
	}
	return false, err
}

// EnsureDir checks whether a directory exists at the given path. If not, it will be created.
func EnsureDir(dir string, mode os.FileMode) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, mode)
		if err != nil {
			return fmt.Errorf("Could not create directory %v. %v", dir, err)
		}
	}
	return nil
}

func ToPEMFile(o encoding.BinaryMarshaler, f, pemType string) error {
	b, err := o.MarshalBinary()
	if err != nil {
		return err
	}
	blk := &pem.Block{
		Type:  pemType,
		Bytes: b,
	}
	return ioutil.WriteFile(f, pem.EncodeToMemory(blk), 0600)
}

func FromPEMFile(o encoding.BinaryUnmarshaler, f, pemType string) error {
	if buf, err := ioutil.ReadFile(filepath.Clean(f)); err == nil {
		blk, rest := pem.Decode(buf)
		if len(rest) != 0 {
			return fmt.Errorf("trailing garbage after PEM encoded key")
		}
		if blk.Type != pemType {
			return fmt.Errorf("invalid PEM Type: '%v'", blk.Type)
		}
		if o.UnmarshalBinary(blk.Bytes) != nil {
			return errors.New("failed to read key from PEM file")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}
