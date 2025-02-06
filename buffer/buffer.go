package buffer

import (
	"log"
	"sync"
	"sync/atomic"
)

const (
	BUFFER_SIZE = 65535
	OOB_SIZE    = 64
)

type PackedBuffer struct {
	Buffer     [BUFFER_SIZE]byte
	SubPackets []int
	TotalSize  int
	RefCnt     atomic.Int32
	// TraceBack  string
}

type WithBufferArg[T any] struct {
	thing  T
	buffer ArgPtr[*PackedBuffer]
}

func (wb *WithBufferArg[T]) ToOwned() WithBuffer[T] {
	return WithBuffer[T]{
		Thing:  wb.thing,
		Buffer: wb.buffer.ToOwned(),
	}
}

type WithBuffer[T any] struct {
	Thing  T
	Buffer OwnedPtr[*PackedBuffer]
}

func (wb *WithBuffer[T]) Release() {
	wb.Buffer.Release()
}

func (wb *WithBuffer[T]) Move() WithBuffer[T] {
	return WithBuffer[T]{
		Thing:  wb.Thing,
		Buffer: wb.Buffer.Move(),
	}
}

func (wb *WithBuffer[T]) MoveArg() WithBufferArg[T] {
	return WithBufferArg[T]{
		thing:  wb.Thing,
		buffer: wb.Buffer.MoveArg(),
	}
}

var packedBufferPool = sync.Pool{
	New: func() interface{} {
		return &PackedBuffer{
			Buffer:     [BUFFER_SIZE]byte{},
			SubPackets: make([]int, 0),
			RefCnt:     atomic.Int32{},
			TotalSize:  0,
		}
	},
}

var ActiveBuffers atomic.Int64

// var BufferTraceBack = struct {
// 	sync.Mutex
// 	TraceBack map[string]int
// }{
// 	TraceBack: make(map[string]int),
// }

func NewPackedBuffer() OwnedPtr[*PackedBuffer] {
	// default refcnt is 1
	buffer := packedBufferPool.Get().(*PackedBuffer)
	buffer.SubPackets = make([]int, 0)
	buffer.TotalSize = 0
	buffer.RefCnt.Store(1)
	// buffer.TraceBack = string(debug.Stack())
	// BufferTraceBack.Lock()
	// BufferTraceBack.TraceBack[buffer.TraceBack]++
	// BufferTraceBack.Unlock()
	// log.Println("buffer new, active buffers:", ActiveBuffers.Add(1))
	return OwnedPtr[*PackedBuffer]{Ptr: buffer}
}

func (p *PackedBuffer) Acquire() {
	// should be called when there is a new owner, called by caller
	p.RefCnt.Add(1)
}

func (p *PackedBuffer) Release() {
	// should be called when not owned by any one, called by callee
	refCnt := p.RefCnt.Add(-1)
	if refCnt == 0 {
		// it's safe because refCnt is atomic
		packedBufferPool.Put(p)
		// BufferTraceBack.Lock()
		// BufferTraceBack.TraceBack[p.TraceBack]--
		// BufferTraceBack.Unlock()
		// log.Println("buffer released, active buffers:", ActiveBuffers.Add(-1))
	}
	if refCnt < 0 {
		log.Println("buffer reference count is negative:", refCnt)
	}
}
