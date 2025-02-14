package server

import (
	"log"
	"net"

	"github.com/chenx-dust/paracat/buffer"
	"github.com/chenx-dust/paracat/channel"
	"github.com/chenx-dust/paracat/packet"
	"github.com/chenx-dust/paracat/transport"
)

func (server *Server) handleForward(ch <-chan buffer.WithBufferArg[[]*packet.Packet]) {
	for packets_ := range ch {
		packets := packets_.ToOwned()
		connPacketsMap := make(map[uint16][][]byte)
		for _, newPacket := range packets.Thing {
			_, ok := server.forwardConns[newPacket.ConnID]
			if !ok {
				conn, err := net.ListenUDP("udp", nil)
				if err != nil {
					log.Println("error dialing relay:", err)
					continue
				}
				log.Println("new forward conn:", conn.LocalAddr())
				transport.EnableGRO(conn)
				transport.EnableGSO(conn)
				server.forwardConns[newPacket.ConnID] = conn
				go server.handleReverse(conn, newPacket.ConnID)
			}
			connPacketsMap[newPacket.ConnID] = append(connPacketsMap[newPacket.ConnID], newPacket.Buffer)
		}
		remoteAddr, err := net.ResolveUDPAddr("udp", server.cfg.RemoteAddr)
		if err != nil {
			log.Fatalln("error resolving remote addr:", err)
		}
		for connID, packets := range connPacketsMap {
			pBuffer := buffer.NewPackedBuffer()
			nowPtr := 0
			for _, packet := range packets {
				pBuffer.Ptr.SubPackets = append(pBuffer.Ptr.SubPackets, len(packet))
				copy(pBuffer.Ptr.Buffer[nowPtr:], packet)
				nowPtr += len(packet)
			}
			pBuffer.Ptr.TotalSize = nowPtr
			err := transport.SendUDPPackets(server.forwardConns[connID], remoteAddr, pBuffer.BorrowArg())
			pBuffer.Release()
			if err != nil {
				log.Println("error writing to udp:", err)
			}
		}
		packets.Release()
	}
}

func (server *Server) handleReverse(conn *net.UDPConn, connID uint16) {
	remoteAddr, err := net.ResolveUDPAddr("udp", server.cfg.RemoteAddr)
	if err != nil {
		log.Fatalln("error resolving remote addr:", err)
	}
	for {
		rawPackets, addr, err := transport.ReceiveUDPRawPackets(conn)
		if err != nil {
			log.Println("error receiving udp packets:", err)
			rawPackets.Release()
			continue
		}
		if addr.String() != remoteAddr.String() {
			log.Println("error receiving udp packets: addr mismatch", addr, remoteAddr)
			rawPackets.Release()
			continue
		}

		packets := buffer.NewPackedBuffer()
		nowPtr := 0
		for _, slice := range rawPackets.Ptr.SubPackets {
			packetID := channel.NewPacketID(&server.idIncrement)
			newPacket := &packet.Packet{
				Buffer:   rawPackets.Ptr.Buffer[nowPtr : nowPtr+slice],
				ConnID:   connID,
				PacketID: packetID,
			}
			size := newPacket.Pack(packets.Ptr.Buffer[nowPtr:])
			packets.Ptr.SubPackets = append(packets.Ptr.SubPackets, size)
			packets.Ptr.TotalSize += size

			nowPtr += slice
		}
		rawPackets.Release()
		server.scatterer.Scatter(packets.MoveArg())
	}
}
