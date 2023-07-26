package main

import (
	"bytes"
	"fmt"
	m "pConv/models" // Importing the "m" package
)

// NewNiimbotPacket is the constructor for NiimbotPacket.
func NewNiimbotPacket(type_ byte, data []byte) *m.NiimbotPacket {
	return &m.NiimbotPacket{
		Type: type_,
		Data: data,
	}
}

// FromBytes deserializes a byte slice into a NiimbotPacket.
func FromBytes(pkt []byte) (*m.NiimbotPacket, error) {
	if len(pkt) < 6 || pkt[0] != 0x55 || pkt[1] != 0x55 || pkt[len(pkt)-2] != 0xaa || pkt[len(pkt)-1] != 0xaa {
		return nil, fmt.Errorf("invalid packet format")
	}

	type_ := pkt[2]
	len_ := pkt[3]
	if len(pkt) != int(len_)+6 {
		return nil, fmt.Errorf("invalid packet length")
	}
	data := pkt[4 : 4+len_]

	checksum := type_ ^ len_
	for _, i := range data {
		checksum ^= i
	}
	if checksum != pkt[len(pkt)-3] {
		return nil, fmt.Errorf("invalid checksum")
	}

	return NewNiimbotPacket(type_, data), nil
}

// ToBytes serializes a NiimbotPacket into a byte slice.
func ToBytes(np *m.NiimbotPacket) []byte {
	checksum := np.Type ^ byte(len(np.Data))
	for _, i := range np.Data {
		checksum ^= i
	}
	return bytes.Join([][]byte{
		{0x55, 0x55, np.Type, byte(len(np.Data))},
		np.Data,
		{checksum, 0xaa, 0xaa},
	}, []byte{})
}
