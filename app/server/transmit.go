package server

import (
	"log"
	"net"

	"github.com/chenx-dust/paracat/channel"
	"github.com/chenx-dust/paracat/packet"
	"github.com/chenx-dust/paracat/transport"
)

func (server *Server) handleForward(ch <-chan []*packet.Packet) {
	for packets := range ch {
		connPacketsMap := make(map[uint16][][]byte)
		for _, newPacket := range packets {
			_, ok := server.forwardConns[newPacket.ConnID]
			if !ok {
				conn, err := net.ListenUDP("udp", nil)
				if err != nil {
					log.Println("error dialing relay:", err)
					continue
				}
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
			err := transport.SendUDPPackets(server.forwardConns[connID], remoteAddr, packets)
			if err != nil {
				log.Println("error writing to udp:", err)
			}
		}
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
			continue
		}
		if addr.String() != remoteAddr.String() {
			log.Println("error receiving udp packets: addr mismatch", addr, remoteAddr)
			continue
		}

		packets := make([][]byte, 0, len(rawPackets))
		for _, rawPacket := range rawPackets {
			packetID := channel.NewPacketID(&server.idIncrement)
			newPacket := &packet.Packet{
				Buffer:   rawPacket,
				ConnID:   connID,
				PacketID: packetID,
			}
			packed := newPacket.Pack()
			packets = append(packets, packed)
		}
		server.dispatcher.Dispatch(packets)
	}
}
