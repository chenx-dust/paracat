package buffer

type Acquireable interface {
	Acquire()
	Release()
}

type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

// Release() needed
type OwnedPtr[T Acquireable] struct {
	Ptr T
	_   noCopy
}

// Release() needed, for passing to function or channel
type ArgPtr[T Acquireable] struct {
	ptr T
}

// Release() not needed
type BorrowedPtr[T Acquireable] struct {
	Ptr T
	_   noCopy
}

// Release() not needed
type BorrowedArgPtr[T Acquireable] struct {
	ptr T
}

func (p *OwnedPtr[T]) Borrow() BorrowedPtr[T] {
	return BorrowedPtr[T]{Ptr: p.Ptr}
}

func (p *OwnedPtr[T]) BorrowArg() BorrowedArgPtr[T] {
	return BorrowedArgPtr[T]{ptr: p.Ptr}
}

func (p *OwnedPtr[T]) Move() OwnedPtr[T] {
	return OwnedPtr[T]{Ptr: p.Ptr}
}

func (p *OwnedPtr[T]) MoveArg() ArgPtr[T] {
	return ArgPtr[T]{ptr: p.Ptr}
}

func (p *OwnedPtr[T]) Share() OwnedPtr[T] {
	p.Ptr.Acquire()
	return OwnedPtr[T]{Ptr: p.Ptr}
}

func (p *OwnedPtr[T]) ShareArg() ArgPtr[T] {
	p.Ptr.Acquire()
	return ArgPtr[T]{ptr: p.Ptr}
}

func (p *OwnedPtr[T]) Release() {
	p.Ptr.Release()
}

func (p *BorrowedPtr[T]) Share() OwnedPtr[T] {
	p.Ptr.Acquire()
	return OwnedPtr[T]{Ptr: p.Ptr}
}

func (p *BorrowedPtr[T]) ShareArg() ArgPtr[T] {
	p.Ptr.Acquire()
	return ArgPtr[T]{ptr: p.Ptr}
}

func (p *ArgPtr[T]) ToOwned() OwnedPtr[T] {
	return OwnedPtr[T]{Ptr: p.ptr}
}

func (p *BorrowedArgPtr[T]) ToBorrowed() BorrowedPtr[T] {
	return BorrowedPtr[T]{Ptr: p.ptr}
}
