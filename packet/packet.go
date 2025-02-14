package packet

import (
	"bytes"
	"errors"

	"github.com/sigurn/crc8"
)

type Packet struct {
	Buffer   []byte
	ConnID   uint16
	PacketID uint16
}

var (
	ErrPacketTooShort     = errors.New("packet too short")
	ErrInvalidMagicNumber = errors.New("invalid magic number")
	ErrInvalidCRC         = errors.New("invalid crc")
	table                 = crc8.MakeTable(crc8.CRC8_MAXIM)
)

const (
	MAGIC_NUMBER = 0xa1
	HEADER_SIZE  = 8
)

func (p *Packet) Pack(buffer []byte) (length int) {
	buffer[0] = MAGIC_NUMBER
	buffer[1] = byte(len(p.Buffer))
	buffer[2] = byte(len(p.Buffer) >> 8)
	buffer[3] = byte(p.ConnID)
	buffer[4] = byte(p.ConnID >> 8)
	buffer[5] = byte(p.PacketID)
	buffer[6] = byte(p.PacketID >> 8)
	crc := crc8.Checksum(buffer[:7], table)
	buffer[7] = crc
	copy(buffer[HEADER_SIZE:], p.Buffer)
	return HEADER_SIZE + len(p.Buffer)
}

func Unpack(buffer []byte) (*Packet, int, error) {
	if len(buffer) < HEADER_SIZE {
		return nil, 0, ErrPacketTooShort
	}
	if buffer[0] != MAGIC_NUMBER {
		return nil, 0, ErrInvalidMagicNumber
	}
	crc := crc8.Checksum(buffer[:HEADER_SIZE-1], table)
	if crc != buffer[HEADER_SIZE-1] {
		return nil, 0, ErrInvalidCRC
	}
	length := int(buffer[1]) | int(buffer[2])<<8
	if length > len(buffer)-HEADER_SIZE {
		return nil, 0, ErrPacketTooShort
	}
	packet := &Packet{
		Buffer:   buffer[HEADER_SIZE : HEADER_SIZE+length],
		ConnID:   uint16(buffer[3]) | uint16(buffer[4])<<8,
		PacketID: uint16(buffer[5]) | uint16(buffer[6])<<8,
	}
	parsed := HEADER_SIZE + length
	return packet, parsed, nil
}

func ParsePacket(buffer []byte) ([]*Packet, int, error) {
	packets := make([]*Packet, 0)
	for ptr := 0; ptr < len(buffer); {
		packet, parsed, err := Unpack(buffer[ptr:])
		switch err {
		case nil:
			packets = append(packets, packet)
			ptr += parsed
		case ErrInvalidMagicNumber, ErrInvalidCRC:
			offset := bytes.IndexByte(buffer[ptr+1:], MAGIC_NUMBER)
			if offset == -1 {
				return packets, 0, nil
			}
			ptr += offset + 1
		case ErrPacketTooShort:
			remainingBytes := len(buffer) - ptr
			if remainingBytes >= HEADER_SIZE {
				offset := bytes.IndexByte(buffer[ptr+1:], MAGIC_NUMBER)
				if offset == -1 {
					return packets, 0, nil
				}
				ptr += offset + 1
				continue
			}
			return packets, remainingBytes, nil
		default:
			return nil, 0, err
		}
	}
	return packets, 0, nil
}
