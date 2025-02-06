package packet

import (
	"errors"
	"io"
)

type Packet struct {
	Buffer   []byte
	ConnID   uint16
	PacketID uint16
}

var (
	ErrPacketTooShort     = errors.New("packet too short")
	ErrInvalidMagicNumber = errors.New("invalid magic number")
)

const (
	MAGIC_NUMBER    = 0xa1
	MAX_PACKET_SIZE = 1500
)

func (p *Packet) Pack() []byte {
	packed := make([]byte, 0, 7+len(p.Buffer))
	packed = append(packed, MAGIC_NUMBER)
	packed = append(packed, byte(len(p.Buffer)))
	packed = append(packed, byte(len(p.Buffer)>>8))
	packed = append(packed, byte(p.ConnID))
	packed = append(packed, byte(p.ConnID>>8))
	packed = append(packed, byte(p.PacketID))
	packed = append(packed, byte(p.PacketID>>8))
	packed = append(packed, p.Buffer...)

	return packed
}

func (p *Packet) PackInPlace(buffer []byte) (length int) {
	buffer[0] = MAGIC_NUMBER
	buffer[1] = byte(len(p.Buffer))
	buffer[2] = byte(len(p.Buffer) >> 8)
	buffer[3] = byte(p.ConnID)
	buffer[4] = byte(p.ConnID >> 8)
	buffer[5] = byte(p.PacketID)
	buffer[6] = byte(p.PacketID >> 8)
	copy(buffer[7:], p.Buffer)
	return 7 + len(p.Buffer)
}

func Unpack(buffer []byte) (packet *Packet, parsed int, err error) {
	if len(buffer) < 7 {
		err = ErrPacketTooShort
		return
	}
	if buffer[0] != MAGIC_NUMBER {
		err = ErrInvalidMagicNumber
		return
	}
	length := int(buffer[1]) | int(buffer[2])<<8
	if length > len(buffer)-7 {
		err = ErrPacketTooShort
		return
	}
	packet = &Packet{
		Buffer:   buffer[7 : 7+length],
		ConnID:   uint16(buffer[3]) | uint16(buffer[4])<<8,
		PacketID: uint16(buffer[5]) | uint16(buffer[6])<<8,
	}
	parsed = 7 + length
	return
}

func ReadPacket(reader io.Reader) (packet *Packet, err error) {

	header := make([]byte, 7)
	n, err := reader.Read(header)
	if err != nil {
		return
	}
	if n < 7 {
		return nil, ErrPacketTooShort
	}

	if header[0] != MAGIC_NUMBER {
		return nil, ErrInvalidMagicNumber
	}

	length := int(header[1]) | int(header[2])<<8
	packet = &Packet{
		Buffer:   make([]byte, length),
		ConnID:   uint16(header[3]) | uint16(header[4])<<8,
		PacketID: uint16(header[5]) | uint16(header[6])<<8,
	}
	pt := 0
	for pt < length {
		n, err = reader.Read(packet.Buffer[pt:length])
		if err != nil {
			return
		}
		pt += n
	}
	return
}

// func ParsePacket(buffer []byte) ([]*Packet, int, error) {
// 	packets := make([]*Packet, 0)
// 	for ptr := 0; ptr < len(buffer); {
// 		packet, parsed, err := Unpack(buffer[ptr:])
// 		switch err {
// 		case nil:
// 			packets = append(packets, packet)
// 			ptr += parsed
// 		case ErrInvalidMagicNumber:
// 			offset := bytes.IndexByte(buffer[ptr+1:], MAGIC_NUMBER)
// 			if offset == -1 {
// 				return packets, 0, nil
// 			}
// 			ptr += offset + 1
// 		case ErrPacketTooShort:
// 			remainingBytes := len(buffer) - ptr
// 			if remainingBytes > MAX_PACKET_SIZE {
// 				offset := bytes.IndexByte(buffer[ptr+1:], MAGIC_NUMBER)
// 				if offset == -1 {
// 					return packets, 0, nil
// 				}
// 				ptr += offset + 1
// 				continue
// 			}
// 			return packets, remainingBytes, nil
// 		default:
// 			return nil, 0, err
// 		}
// 	}
// 	return packets, 0, nil
// }
