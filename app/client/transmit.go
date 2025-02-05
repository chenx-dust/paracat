package client

import (
	"errors"
	"log"

	"github.com/chenx-dust/paracat/channel"
	"github.com/chenx-dust/paracat/packet"
	"github.com/chenx-dust/paracat/transport"
)

func (client *Client) handleForward() {
	for {
		packets, addr, err := transport.ReceiveUDPRawPackets(client.udpListener)
		if err != nil {
			log.Fatalln("error reading from udp conn:", err)
		}

		connID, ok := client.connAddrIDMap[addr.String()]
		if !ok {
			connID = uint16(client.connIncrement.Add(1) - 1)
			client.connMutex.Lock()
			client.connIDAddrMap[connID] = addr
			client.connAddrIDMap[addr.String()] = connID
			client.connMutex.Unlock()
			log.Println("new connection from:", addr.String())
		}
		for _, rawPacket := range packets {
			packetID := channel.NewPacketID(&client.idIncrement)

			newPacket := &packet.Packet{
				Buffer:   rawPacket,
				ConnID:   connID,
				PacketID: packetID,
			}
			packed := newPacket.Pack()
			client.dispatcher.Dispatch(packed)
		}
	}
}

func (client *Client) reverseLoop(ch <-chan []*packet.Packet) {
	for packets := range ch {
		for _, newPacket := range packets {
			client.handleReverse(newPacket)
		}
	}
}

func (client *Client) handleReverse(newPacket *packet.Packet) (n int, err error) {
	client.connMutex.RLock()
	udpAddr, ok := client.connIDAddrMap[newPacket.ConnID]
	client.connMutex.RUnlock()
	if !ok {
		log.Println("conn not found")
		return 0, errors.New("conn not found")
	}
	n, err = client.udpListener.WriteToUDP(newPacket.Buffer, udpAddr)
	if err != nil {
		log.Println("error writing to udp:", err)
	} else if n != len(newPacket.Buffer) {
		log.Println("error writing to udp: wrote", n, "bytes instead of", len(newPacket.Buffer))
	}
	return
}
