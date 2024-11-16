package packet

import (
	"sync"
	"sync/atomic"
)

type PacketFilter struct {
	packetMutex     sync.Mutex
	packetLowMap    [0x8000]bool
	packetLowClear  bool
	packetHighMap   [0x8000]bool
	packetHighClear bool
}

func NewPacketFilter() *PacketFilter {
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
