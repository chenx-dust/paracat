package client

import (
	"log"

	"github.com/chenx-dust/paracat/channel"
	"github.com/chenx-dust/paracat/packet"
	"github.com/chenx-dust/paracat/transport"
)

func (client *Client) handleForward() {
	for {
		rawPackets, addr, err := transport.ReceiveUDPRawPackets(client.udpListener)
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
		packets := make([][]byte, 0, len(rawPackets))
		for _, rawPacket := range rawPackets {
			packetID := channel.NewPacketID(&client.idIncrement)

			newPacket := &packet.Packet{
				Buffer:   rawPacket,
				ConnID:   connID,
				PacketID: packetID,
			}
			packed := newPacket.Pack()
			packets = append(packets, packed)
		}
		client.dispatcher.Dispatch(packets)
	}
}

func (client *Client) handleReverse(ch <-chan []*packet.Packet) {
	for packets := range ch {
		connPacketsMap := make(map[uint16][][]byte)
		client.connMutex.RLock()
		for _, newPacket := range packets {
			_, ok := client.connIDAddrMap[newPacket.ConnID]
			if !ok {
				log.Println("conn not found:", newPacket.ConnID)
				continue
			}
			connPacketsMap[newPacket.ConnID] = append(connPacketsMap[newPacket.ConnID], newPacket.Buffer)
		}
		client.connMutex.RUnlock()
		for connID, packets := range connPacketsMap {
			err := transport.SendUDPPackets(client.udpListener, client.connIDAddrMap[connID], packets)
			if err != nil {
				log.Println("error writing to udp:", err)
			}
		}
	}
}
