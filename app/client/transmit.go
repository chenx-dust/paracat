package client

import (
	"log"

	"github.com/chenx-dust/paracat/buffer"
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
		packets := buffer.NewPackedBuffer()
		nowPtr := 0
		for _, slice := range rawPackets.Ptr.SubPackets {
			packetID := channel.NewPacketID(&client.idIncrement)

			newPacket := &packet.Packet{
				Buffer:   rawPackets.Ptr.Buffer[nowPtr : nowPtr+slice],
				ConnID:   connID,
				PacketID: packetID,
			}
			size := newPacket.PackInPlace(packets.Ptr.Buffer[nowPtr:])
			packets.Ptr.SubPackets = append(packets.Ptr.SubPackets, size)
			packets.Ptr.TotalSize += size

			nowPtr += slice
		}
		rawPackets.Release()
		client.scatterer.Scatter(packets.MoveArg())
	}
}

func (client *Client) handleReverse(ch <-chan buffer.WithBufferArg[[]*packet.Packet]) {
	for packets_ := range ch {
		packets := packets_.ToOwned()
		connPacketsMap := make(map[uint16][][]byte)
		client.connMutex.RLock()
		for _, newPacket := range packets.Thing {
			_, ok := client.connIDAddrMap[newPacket.ConnID]
			if !ok {
				log.Println("conn not found:", newPacket.ConnID)
				continue
			}
			connPacketsMap[newPacket.ConnID] = append(connPacketsMap[newPacket.ConnID], newPacket.Buffer)
		}
		client.connMutex.RUnlock()
		for connID, packets := range connPacketsMap {
			pBuffer := buffer.NewPackedBuffer()
			nowPtr := 0
			for _, packet := range packets {
				pBuffer.Ptr.SubPackets = append(pBuffer.Ptr.SubPackets, len(packet))
				copy(pBuffer.Ptr.Buffer[nowPtr:], packet)
				nowPtr += len(packet)
			}
			pBuffer.Ptr.TotalSize = nowPtr
			err := transport.SendUDPPackets(client.udpListener, client.connIDAddrMap[connID], pBuffer.BorrowArg())
			pBuffer.Release()
			if err != nil {
				log.Println("error writing to udp:", err)
			}
		}
		packets.Release()
	}
}
