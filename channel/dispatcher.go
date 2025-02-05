/* Dispatcher is a SIMO channel. */
package channel

import (
	"errors"
	"log"
	"sync"

	"github.com/chenx-dust/paracat/config"
	"github.com/chenx-dust/paracat/packet"
)

type Dispatcher struct {
	connMutex     sync.RWMutex
	outChans      []chan<- [][]byte
	roundRobinIdx int
	mode          config.DispatchType

	StatisticIn  *packet.PacketStatistic
	StatisticOut *packet.PacketStatistic
}

func NewDispatcher(mode config.DispatchType) *Dispatcher {
	if mode == config.NotDefinedDispatchType {
		log.Fatal("dispatcher mode not defined")
	}
	log.Println("new dispatcher with mode:", config.DispatchTypeToString(mode))
	return &Dispatcher{
		outChans:      make([]chan<- [][]byte, 0),
		roundRobinIdx: 0,
		mode:          mode,
		StatisticIn:   packet.NewPacketStatistic(),
		StatisticOut:  packet.NewPacketStatistic(),
	}
}

func (d *Dispatcher) NewOutput(ch chan<- [][]byte) {
	d.connMutex.Lock()
	defer d.connMutex.Unlock()
	d.outChans = append(d.outChans, ch)
}

func (d *Dispatcher) RemoveOutput(ch chan<- [][]byte) error {
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

func (d *Dispatcher) Dispatch(data [][]byte) {
	totalSize := 0
	for _, packet := range data {
		totalSize += len(packet)
	}
	d.connMutex.RLock()
	defer d.connMutex.RUnlock()
	switch d.mode {
	case config.RoundRobinDispatchType:
		d.StatisticIn.CountPacket(uint32(totalSize))
		d.roundRobinIdx = (d.roundRobinIdx + 1) % len(d.outChans)
		select {
		case d.outChans[d.roundRobinIdx] <- data:
			d.StatisticOut.CountPacket(uint32(totalSize))
		default:
		}
	case config.ConcurrentDispatchType:
		d.StatisticIn.CountPacket(uint32(totalSize))
		for _, outChan := range d.outChans {
			select {
			case outChan <- data:
				d.StatisticOut.CountPacket(uint32(totalSize))
			default:
			}
		}
	}
}
