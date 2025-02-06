/* Gatherer is a channel with duplicate packet gather. For MISO usage. */
package channel

import (
	"sync"
	"sync/atomic"

	"github.com/chenx-dust/paracat/buffer"
	"github.com/chenx-dust/paracat/packet"
)

type Gatherer struct {
	// outCallback func(packet *packet.Packet) (int, error)
	gather  *PacketFilter
	chanOut chan buffer.WithBufferArg[[]*packet.Packet]

	StatisticIn  *packet.PacketStatistic
	StatisticOut *packet.PacketStatistic
}

func NewGatherer(chanSize int) *Gatherer {
	return &Gatherer{
		gather:       NewPacketGather(),
		chanOut:      make(chan buffer.WithBufferArg[[]*packet.Packet], chanSize),
		StatisticIn:  packet.NewPacketStatistic(),
		StatisticOut: packet.NewPacketStatistic(),
	}
}

func (ch *Gatherer) GetOutChan() <-chan buffer.WithBufferArg[[]*packet.Packet] {
	return ch.chanOut
}

func (ch *Gatherer) Forward(newPackets_ buffer.WithBufferArg[[]*packet.Packet]) {
	newPackets := newPackets_.ToOwned()
	inSize := 0
	outSize := 0
	fwdPackets := make([]*packet.Packet, 0, len(newPackets.Thing))
	for _, newPacket := range newPackets.Thing {
		inSize += len(newPacket.Buffer)
		if ch.gather.CheckDuplicatePacketID(newPacket.PacketID) {
			continue
		}
		outSize += len(newPacket.Buffer)
		fwdPackets = append(fwdPackets, newPacket)
	}
	ch.StatisticIn.CountPacket(uint32(inSize))
	ch.StatisticOut.CountPacket(uint32(outSize))
	data := buffer.WithBuffer[[]*packet.Packet]{
		Thing:  fwdPackets,
		Buffer: newPackets.Buffer.Move(),
	}
	select {
	case ch.chanOut <- data.MoveArg():
	default:
		data.Release()
	}
}

type PacketFilter struct {
	packetMutex     sync.Mutex
	packetLowMap    [0x8000]bool
	packetLowClear  bool
	packetHighMap   [0x8000]bool
	packetHighClear bool
}

func NewPacketGather() *PacketFilter {
	return &PacketFilter{
		packetLowMap:    [0x8000]bool{},
		packetLowClear:  true,
		packetHighMap:   [0x8000]bool{},
		packetHighClear: true,
	}
}

func (pf *PacketFilter) CheckDuplicatePacketID(id uint16) bool {
	/*
		divide packet id into four partitions:
		0x0000 ~ 0x3FFF: low map, allowing high map packet input
		0x4000 ~ 0x7FFF: low map, clearing high map
		0x8000 ~ 0xBFFF: high map, allowing low map packet input
		0xC000 ~ 0xFFFF: high map, clearing low map
	*/
	var ok bool
	pf.packetMutex.Lock()
	defer pf.packetMutex.Unlock()
	if id < 0x8000 {
		ok = pf.packetLowMap[id]
		if !ok {
			pf.packetLowMap[id] = true
		}
		pf.packetLowClear = false
		if id > 0x3FFF && !pf.packetHighClear {
			pf.packetHighMap = [0x8000]bool{}
			pf.packetHighClear = true
		}
	} else {
		ok = pf.packetHighMap[id-0x8000]
		if !ok {
			pf.packetHighMap[id-0x8000] = true
		}
		pf.packetHighClear = false
		if id < 0xC000 && !pf.packetLowClear {
			pf.packetLowMap = [0x8000]bool{}
			pf.packetLowClear = true
		}
	}
	return ok
}

func NewPacketID(idIncrement *atomic.Uint32) uint16 {
	return uint16(idIncrement.Add(1) - 1)
}
