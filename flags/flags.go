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

/*
	Package flags defines struct and methods on flags used in sphinx packets
	as well as general data packets sent between different entities in the system.
*/

package flags

// SphinxFlag represents flag present in all sphinx packages to indicate whether the packet has reached
// its final hop or should be relayed.
type SphinxFlag byte

const (
	// LastHopFlag denotes whether this message has reached its final destination.
	LastHopFlag SphinxFlag = '\xf0'
	// RelayFlag denotes whether this message should continue further along the path of mixes.
	// This is implementation-specific rather than being part of the Loopix protocol design.
	RelayFlag SphinxFlag = '\xf1'
	// InvalidFlag denotes an invalid sphinx flag.
	InvalidSphinxFlag SphinxFlag = '\x00'
)

func (sf SphinxFlag) Bytes() []byte {
	return []byte{byte(sf)}
}

func SphinxFlagFromByte(b byte) SphinxFlag {
	switch b {
	case byte(LastHopFlag):
		return LastHopFlag
	case byte(RelayFlag):
		return RelayFlag
	default:
		return InvalidSphinxFlag
	}
}

func SphinxFlagFromBytes(b []byte) SphinxFlag {
	if len(b) != 1 {
		return InvalidSphinxFlag
	}
	return SphinxFlagFromByte(b[0])
}

// PacketTypeFlag represents flag present in all general data packets exchanged between all entities in the system.
// They are used to indicate type of the packet content, i.e. sphinx packet, pull request, etc.
type PacketTypeFlag byte

const (
	// AssignFlag is used to indicate client request to get registered at a particular provider.
	AssignFlag PacketTypeFlag = '\xa2'
	// CommFlag is used to indicate that the packet contains sphinx payload and should be processed accordingly.
	CommFlag PacketTypeFlag = '\xc6'
	// TokenFlag is used to indicate that the packet contains authentication token from provider
	// that is sent as a result of getting registered.
	TokenFlag PacketTypeFlag = '\xa9'
	// PullFlag is used to indicate client request to obtain all its messages stored at a particular provider.
	PullFlag PacketTypeFlag = '\xff'
	// InvalidFlag is used to indicate an invalid packet type flag.
	InvalidPacketTypeFlag PacketTypeFlag = '\x00'
)

func (pf PacketTypeFlag) Bytes() []byte {
	return []byte{byte(pf)}
}

func PacketTypeFlagFromByte(b byte) PacketTypeFlag {
	switch b {
	case byte(AssignFlag):
		return AssignFlag
	case byte(CommFlag):
		return CommFlag
	case byte(TokenFlag):
		return TokenFlag
	case byte(PullFlag):
		return PullFlag
	default:
		return InvalidPacketTypeFlag
	}
}

func PacketTypeFlagFromBytes(b []byte) PacketTypeFlag {
	if len(b) != 1 {
		return InvalidPacketTypeFlag
	}
	return PacketTypeFlagFromByte(b[0])
}
