package multicast

import (
	"sync"
	"time"
)

const (
	defaultOutboundQueueBufferSize = 1000
)

// Multicast represents an object capable of broadcasting
// messages from multiple writers to multiple listeners.
type Multicast[T any] struct {
	// lock for writers list.
	mutexW sync.RWMutex

	// lock for listeners list.
	mutexL sync.RWMutex

	// list of all the current message writers.
	writers []*Writer[T]

	// list of all the current message listeners.
	listeners []*Listener[T]

	// used as a FIFO queue for outbound messages
	// to be forwarded to all the listeners.
	outboundQueue chan T

	// used to signal when the outbound
	// queue is drained after closure
	outboundQueueDrained chan struct{}
}

// CloseFunc represents a function to close
// a multicast Listener or multicast Writer.
type CloseFunc func()

// New returns a newly initialized Multicast of the specified type.
func New[T any]() *Multicast[T] {
	m := &Multicast[T]{
		// note: mutex lock doesn't need initialization
		listeners:            []*Listener[T]{},
		outboundQueue:        make(chan T, defaultOutboundQueueBufferSize),
		outboundQueueDrained: make(chan struct{}),
	}
	go m.run()
	return m
}

// Close closes the multicast and blocks until all
// messages currently in the outbound queue are sent.
func (m *Multicast[T]) Close() {
	m.mutexW.Lock()
	defer m.mutexW.Unlock()

	// close all writers
	for _, writer := range m.writers {
		writer.close()
	}

	// close outbound queue
	close(m.outboundQueue)

	// block until outbound queue is drained
	<-m.outboundQueueDrained
}

// NewWriter returns a new message Writer for the multicast.
func (m *Multicast[T]) NewWriter() (*Writer[T], CloseFunc) {
	m.mutexW.Lock()
	defer m.mutexW.Unlock()

	writer := newWriter[T](m.outboundQueue)

	m.writers = append(m.writers, writer)

	return writer, m.closeWriterFn(writer)
}

// NewListener returns a new message Listener for the multicast.
func (m *Multicast[T]) NewListener(capacity int) (*Listener[T], CloseFunc) {
	m.mutexL.Lock()
	defer m.mutexL.Unlock()

	listener := newListener[T](capacity)

	m.listeners = append(m.listeners, listener)

	return listener, m.closeListenerFn(listener)
}

// run processes messages in the outbound queue
// until the multicast is closed. When the queue
// is closed, it signals a channel when done draining.
func (m *Multicast[T]) run() {
	// process messages in the outbound queue
	// until the outbound queue channel is closed
	for message := range m.outboundQueue {
		m.broadcast(message)
	}
	// signal that the outbound queue channel is drained
	m.outboundQueueDrained <- struct{}{}
	close(m.outboundQueueDrained)
}

// broadcast broadcasts a message to all listeners of the multicast.
func (m *Multicast[T]) broadcast(message T) {
	m.mutexL.RLock()
	defer m.mutexL.RUnlock()

	for _, listener := range m.listeners {
		listener.c <- message
	}
}

// closeListenerFn returns a function to handle
// the safe closure of a listener in a multicast.
func (m *Multicast[T]) closeListenerFn(listener *Listener[T]) func() {
	return func() {
		m.beforeClosingListener()

		m.mutexL.Lock()
		defer m.mutexL.Unlock()

		for i, l := range m.listeners {
			if listener.c == l.c {
				// remove listener from list of listeners
				copy(m.listeners[i:], m.listeners[i+1:])
				m.listeners[len(m.listeners)-1] = nil
				m.listeners = m.listeners[:len(m.listeners)-1]

				// close the listener
				listener.close()
				break
			}
		}
	}
}

// beforeClosingListener sleeps for a time proportional to the ammount
// of listeners in the multicast. The number of listeners are used as a
// proxy for how busy the CPU is... e.g. more listeners means more time
// for the scheduler to dispatch the reader go routine of any listener.
// We do this because even though all messages have already been read by
// the listener's reader at this point, the processing of the very last
// message may still be in progress...
func (m *Multicast[T]) beforeClosingListener() {
	m.mutexL.Lock()
	nListeners := len(m.listeners)
	m.mutexL.Unlock()
	time.Sleep(time.Millisecond * time.Duration(nListeners))
}

// closeWriterFn returns a function to handle
// the safe closure of a writer in a multicast.
func (m *Multicast[T]) closeWriterFn(writer *Writer[T]) func() {
	return func() { writer.close() }
}
