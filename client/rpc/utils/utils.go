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
    "math"
)

const (
	maxRequestSize = 1048576 // 1MB

    varintTwoBytes = 0xfd
    varintFourBytes = 0xfe
    varintEightBytes = 0xff
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
	length64, err := ReadVarUintSimple(reader)
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
	err = encodeVarint(w, uint64(len(bz)))
	if err != nil {
		return
	}
	_, err = w.Write(bz)
	return
}

func encodeVarint(w io.Writer, i uint64) (err error) {
	var buf [10]byte
	n := PutVarUintSimple(buf[:], i)
	_, err = w.Write(buf[0:n])
	return
}

// Bitcoin-style variable uints
// https://learnmeabitcoin.com/guide/varint

func PutVarUintSimple(buffer []byte, value uint64) uint {
    if value < varintTwoBytes {
        buffer[0] = uint8(value)
        return 1
    } else if value <= math.MaxUint16 {
        buffer[0] = uint8(varintTwoBytes)
        binary.BigEndian.PutUint16(buffer[1:], uint16(value))
        return 3
    } else if value <= math.MaxUint32 {
        buffer[0] = uint8(varintFourBytes)
        binary.BigEndian.PutUint32(buffer[1:], uint32(value))
        return 5
    } else {
        buffer[0] = uint8(varintEightBytes)
        binary.BigEndian.PutUint64(buffer[1:], uint64(value))
        return 9
    }
}

func ReadNBytes(reader io.ByteReader, buffer []byte, number_bytes int) error {
    for i := 0; i < number_bytes; i++ {
        value, err := reader.ReadByte()
        if err != nil {
            return err
        }
        buffer[i] = value
    }
    return nil
}

func ReadVarUintSimple(reader io.ByteReader) (uint64, error) {
    value, err := reader.ReadByte()
    if err != nil {
        return 0, err
    }

    switch value {
    case varintEightBytes:
        buffer := make([]byte, 8)
        err = ReadNBytes(reader, buffer, 8)
        if err != nil {
            return 0, err
        }
        return binary.BigEndian.Uint64(buffer), nil
    case varintFourBytes:
        buffer := make([]byte, 4)
        err = ReadNBytes(reader, buffer, 4)
        if err != nil {
            return 0, err
        }
        return uint64(binary.BigEndian.Uint32(buffer)), nil
    case varintTwoBytes:
        buffer := make([]byte, 2)
        err = ReadNBytes(reader, buffer, 2)
        if err != nil {
            return 0, err
        }
        return uint64(binary.BigEndian.Uint16(buffer)), nil
    default:
        return uint64(value), nil
    }
    return 0, nil
}

