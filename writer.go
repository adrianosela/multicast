package multicast

import (
	"errors"
	"sync"
)

// Writer represents a multicast writer.
type Writer[T any] struct {
	// lock for closed variable.
	mutexC sync.RWMutex

	// true when the Writer is closed.
	closed bool

	// write-only channel where each written
	// message is sent to each listener.
	c chan<- T
}

// ErrWriterClosed is returned when Write() is invoked on a closed Writer.
var ErrWriterClosed = errors.New("writer is closed")

// newWriter returns a new Writer with a given channel.
func newWriter[T any](c chan<- T) *Writer[T] {
	return &Writer[T]{closed: false, c: c}
}

// close soft-closes the Writer (marks it as closed). Note that
// the channel is not closed (as it is owned by the Multicast).
func (w *Writer[T]) close() {
	w.mutexC.Lock()
	defer w.mutexC.Unlock()

	w.closed = true
}

// Write writes a message to the Writer.
func (w *Writer[T]) Write(message T) error {
	w.mutexC.RLock()
	defer w.mutexC.RUnlock()

	if w.closed {
		return ErrWriterClosed
	}

	w.c <- message

	return nil
}
