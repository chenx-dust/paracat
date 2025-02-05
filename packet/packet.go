package packet

import (
	"errors"
	"io"
)

const MAGIC_NUMBER = 0xa1

type Packet struct {
	Buffer   []byte
	ConnID   uint16
	PacketID uint16
}

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

func WritePacket(writer io.Writer, p *Packet) (n int, err error) {
	packed := p.Pack()

	n = 0
	for n < len(packed) {
		n_, err := writer.Write(packed[n:])
		if err != nil {
			return n + n_, err
		}
		n += n_
	}
	n -= 7
	return
}

func Unpack(buffer []byte) (packet *Packet, err error) {
	if len(buffer) < 7 {
		return nil, errors.New("invalid packet")
	}
	if buffer[0] != MAGIC_NUMBER {
		return nil, errors.New("invalid magic number")
	}
	length := int(buffer[1]) | int(buffer[2])<<8
	packet = &Packet{
		Buffer:   buffer[7:],
		ConnID:   uint16(buffer[3]) | uint16(buffer[4])<<8,
		PacketID: uint16(buffer[5]) | uint16(buffer[6])<<8,
	}
	if length != len(packet.Buffer) {
		return nil, errors.New("invalid packet length")
	}
	return
}

func ReadPacket(reader io.Reader) (packet *Packet, err error) {
	header := make([]byte, 7)
	n, err := reader.Read(header)
	if err != nil {
		return
	}
	if n < 7 {
		return nil, errors.New("invalid packet")
	}

	if header[0] != MAGIC_NUMBER {
		return nil, errors.New("invalid magic number")
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
