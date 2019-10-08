// Copyright 2019 The Loopix-Messaging Authors
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

package utils

import (
	"bufio"
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"io"
)

const (
	maxRequestSize = 1048576 // 1MB
)


func WriteProtoMessage(msg proto.Message, w io.Writer) error {
	b, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	return encodeByteSlice(w, b)
}

func ReadProtoMessage(msg proto.Message, r io.Reader) error {
	// binary.ReadVarint takes an io.ByteReader, eg. a bufio.Reader
	reader, ok := r.(*bufio.Reader)
	if !ok {
		reader = bufio.NewReader(r)
	}
	length64, err := binary.ReadVarint(reader)
	if err != nil {
		return err
	}
	length := int(length64)
	if length < 0 || length > maxRequestSize {
		return io.ErrShortBuffer
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return err
	}
	return proto.Unmarshal(buf, msg)
}

// below two functions were copied from
// https://github.com/tendermint/tendermint/blob/f7f034a8befeeb84a88ae8f0092f9f465d9a2544/abci/types/messages.go
// Apache 2.0 license

func encodeByteSlice(w io.Writer, bz []byte) (err error) {
	err = encodeVarint(w, int64(len(bz)))
	if err != nil {
		return
	}
	_, err = w.Write(bz)
	return
}

func encodeVarint(w io.Writer, i int64) (err error) {
	var buf [10]byte
	n := binary.PutVarint(buf[:], i)
	_, err = w.Write(buf[0:n])
	return
}