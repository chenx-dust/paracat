/* Dispatcher is a SIMO channel. */
package channel

import (
	"errors"
	"io"
	"log"
	"sync"

	"github.com/chenx-dust/paracat/config"
	"github.com/chenx-dust/paracat/packet"
)

type Dispatcher struct {
	connMutex     sync.RWMutex
	outConns      []io.Writer
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
		outConns:      make([]io.Writer, 0),
		roundRobinIdx: 0,
		mode:          mode,
		StatisticIn:   packet.NewPacketStatistic(),
		StatisticOut:  packet.NewPacketStatistic(),
	}
}

func (d *Dispatcher) NewOutput(conn io.Writer) {
	d.connMutex.Lock()
	defer d.connMutex.Unlock()
	d.outConns = append(d.outConns, conn)
}

func (d *Dispatcher) RemoveOutput(conn io.Writer) error {
	d.connMutex.Lock()
	defer d.connMutex.Unlock()
	for i := 0; i < len(d.outConns); i++ {
		if d.outConns[i] == conn {
			d.outConns[i] = d.outConns[len(d.outConns)-1]
			d.outConns = d.outConns[:len(d.outConns)-1]
			return nil
		}
	}
	return errors.New("channel not found")
}

func (d *Dispatcher) Dispatch(data []byte) {
	d.connMutex.RLock()
	defer d.connMutex.RUnlock()
	switch d.mode {
	case config.RoundRobinDispatchType:
		d.StatisticIn.CountPacket(uint32(len(data)))
		d.roundRobinIdx = (d.roundRobinIdx + 1) % len(d.outConns)
		n, err := d.outConns[d.roundRobinIdx].Write(data)
		if err == nil {
			d.StatisticOut.CountPacket(uint32(n))
		} else {
			log.Println("error dispatching packet:", err)
		}
	case config.ConcurrentDispatchType:
		d.StatisticIn.CountPacket(uint32(len(data)))
		for _, outConn := range d.outConns {
			n, err := outConn.Write(data)
			if err == nil {
				d.StatisticOut.CountPacket(uint32(n))
			} else {
				log.Println("error dispatching packet:", err)
			}
		}
	}
}
