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

	length_as_bytes:= make([]byte, 8)
	if _, err := r.Read(length_as_bytes); err != nil {
		return err
	}

	length := binary.BigEndian.Uint64(length_as_bytes)

	if length < 0 || length > maxRequestSize {
		return io.ErrShortBuffer
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return err
	}
	return proto.Unmarshal(buf, msg)
}

func encodeByteSlice(w io.Writer, bz []byte) (err error) {
	err = encodeBigEndianLen(w, uint64(len(bz)))
	if err != nil {
		return
	}
	_, err = w.Write(bz)
	return
}

func encodeBigEndianLen(w io.Writer, i uint64) (err error) {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, i)
	_, err = w.Write(buf)
	return
}