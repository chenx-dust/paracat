/* Scatterer is a SIMO channel. */
package channel

import (
	"errors"
	"log"
	"sync"

	"github.com/chenx-dust/paracat/buffer"
	"github.com/chenx-dust/paracat/config"
	"github.com/chenx-dust/paracat/packet"
)

type Scatterer struct {
	connMutex     sync.RWMutex
	outChans      []chan<- buffer.ArgPtr[*buffer.PackedBuffer]
	roundRobinIdx int
	mode          config.ScatterType

	StatisticIn  *packet.PacketStatistic
	StatisticOut *packet.PacketStatistic
}

func NewScatterer(mode config.ScatterType) *Scatterer {
	if mode == config.NotDefinedScatterType {
		log.Fatal("scatterer mode not defined")
	}
	log.Println("new scatterer with mode:", config.ScatterTypeToString(mode))
	return &Scatterer{
		outChans:      make([]chan<- buffer.ArgPtr[*buffer.PackedBuffer], 0),
		roundRobinIdx: 0,
		mode:          mode,
		StatisticIn:   packet.NewPacketStatistic(),
		StatisticOut:  packet.NewPacketStatistic(),
	}
}

func (d *Scatterer) NewOutput(ch chan<- buffer.ArgPtr[*buffer.PackedBuffer]) {
	d.connMutex.Lock()
	defer d.connMutex.Unlock()
	d.outChans = append(d.outChans, ch)
}

func (d *Scatterer) RemoveOutput(ch chan<- buffer.ArgPtr[*buffer.PackedBuffer]) error {
	d.connMutex.Lock()
	defer d.connMutex.Unlock()
	for i := 0; i < len(d.outChans); i++ {
		if d.outChans[i] == ch {
			d.outChans[i] = d.outChans[len(d.outChans)-1]
			d.outChans = d.outChans[:len(d.outChans)-1]
			close(ch)
			return nil
		}
	}
	return errors.New("channel not found")
}

func (d *Scatterer) Scatter(data_ buffer.ArgPtr[*buffer.PackedBuffer]) {
	data := data_.ToOwned()
	defer data.Release()
	d.StatisticIn.CountPacket(uint32(data.Ptr.TotalSize))
	d.connMutex.RLock()
	defer d.connMutex.RUnlock()
	switch d.mode {
	case config.RoundRobinScatterType:
		d.roundRobinIdx = (d.roundRobinIdx + 1) % len(d.outChans)
		sharingData := data.ShareArg()
		select {
		case d.outChans[d.roundRobinIdx] <- sharingData:
			d.StatisticOut.CountPacket(uint32(data.Ptr.TotalSize))
		default:
			sd := sharingData.ToOwned()
			sd.Release()
		}
	case config.ConcurrentScatterType:
		d.StatisticIn.CountPacket(uint32(data.Ptr.TotalSize))
		for _, outChan := range d.outChans {
			sharingData := data.ShareArg()
			select {
			case outChan <- sharingData:
				d.StatisticOut.CountPacket(uint32(data.Ptr.TotalSize))
			default:
				sd := sharingData.ToOwned()
				sd.Release()
			}
		}
	}
}
