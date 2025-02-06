# Conventional Pointer with sync.Pool Memory Management

## Overview

This document describes a conventional pointer implementation that intends to better reuse memory. The conventional pointer uses reference counting to manage the lifetime of the buffer, but the reference counting is done manually. Codes in this project is in `buffer/ptr.go`.

## Basic Pointer

A reusable object must implement the `Acquireable` interface:

```go
type Acquireable interface {
	Acquire()
	Release()
}
```

`Acquire()` is called when the object is shared by another identity, which adds a reference to the object. `Release()` is called when the object is no longer needed, which removes a reference from the object.

A normal pointer is defined as:

```go
type Ptr[T Acquireable] struct {
	Ptr *T
    _ noCopy
}
```

All kinds of normal pointers should not be copied. But for some cases, it is necessary to copy the pointer, for example, when the pointer is passed to a function or channel. Then the `ArgPtr` type is used:

```go
type ArgPtr[T Acquireable] struct {
	ptr *T
}
```

`ptr` in `ArgPtr` is private, which prevents direct usage of it. Any `ArgPtr` should be converted to a normal pointer by calling `ToNormal()`-like method:

```go
func (p *ArgPtr[T]) ToNormal() *Ptr[T] {
	return Ptr[T]{Ptr: p.Ptr}
}
```

## Ownership of Pointers

`OwnedPtr` is a normal pointer declaring that the pointer is owned by the current identity. It implements `Borrow()`, `Share()`, `Move()`, and `Release()` methods:

```go
func (p *OwnedPtr[T]) Borrow() *BorrowedPtr[T] {
	return BorrowedPtr[T]{Ptr: p.Ptr}
}

func (p *OwnedPtr[T]) Share() *OwnedPtr[T] {
	p.Ptr.Acquire()
	return OwnedPtr[T]{Ptr: p.Ptr}
}

func (p *OwnedPtr[T]) Move() *OwnedPtr[T] {
	return OwnedPtr[T]{Ptr: p.Ptr}
}

func (p *OwnedPtr[T]) Release() {
	p.Ptr.Release()
}
```

`Share()` adds a reference to the pointer and gets another `OwnedPtr`, which can be passed to another identity.

`Move()` is mean to be used when the pointer is no longer needed by the current identity, but doesn't remove the reference. It should be transfered the ownership to another identity.

`Release()` means the pointer is no longer needed, and the reference should be removed. Usually used by consumers.

`Borrow()` is mean to be used when the pointer is temporarily used by another identity, like a function call. It doesn't add a reference to the pointer, but it can be converted to an `OwnedPtr` by calling `Share()`. Borrowed pointer, a.k.a. `BorrowedPtr`, implements `Share()` but not `Release()` method, because the pointer is borrowed and never be released by the borrower. **Never pass a borrowed pointer to another identity that has different lifetime.**
