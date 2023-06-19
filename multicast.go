package multicast

import (
	"fmt"
	"sync"
	"time"
)

const (
	defaultOutboundQueueSize = 1000
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

// CloseFunc represents a function to close a multicast Writer.
type CloseFunc func()

// DrainFunc represents a function to wait for a
// multicast Listener to process its inbound queue.
type DrainFunc func(time.Duration)

// config represents configuration for a new Multicast.
type config struct {
	outboundQueueSize int
}

// Option represents a Multicast configuration option.
type Option func(*config)

// WithOutboundQueueSize is a Multicast configuration
// option to set a non default value for the outbound
// queue (channel) buffer size.
func WithOutboundQueueSize(size int) Option {
	return func(c *config) {
		c.outboundQueueSize = size
	}
}

// New returns a newly initialized Multicast of the specified type.
func New[T any](opts ...Option) *Multicast[T] {
	c := &config{
		outboundQueueSize: defaultOutboundQueueSize,
	}
	for _, opt := range opts {
		opt(c)
	}
	m := &Multicast[T]{
		// note: mutex lock doesn't need initialization
		listeners:            []*Listener[T]{},
		outboundQueue:        make(chan T, c.outboundQueueSize),
		outboundQueueDrained: make(chan struct{}),
	}
	go m.run()
	return m
}

// Close closes the Multicast.
func (m *Multicast[T]) Close() {
	m.closeAllWriters()
	// TODO: consider waiting for all listeners to drain (or timeout)
}

// closeAllWriters closes the multicast and blocks until
// all messages currently in the outbound queue are sent.
func (m *Multicast[T]) closeAllWriters() {
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
func (m *Multicast[T]) NewListener(capacity int) (*Listener[T], DrainFunc) {
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
		go func(l *Listener[T]) {

			defer func() {
				if err := recover(); err != nil {
					// If this happens, the listener was closed ungracefully.
					//
					// This happens in a very specific sequence of events:
					// - (1) The listener's queue is completely full
					// - (2) The listener receives more messages (meaning
					//       that there will be a build up of go routines
					//       of this function waiting for queue space in
					//       order to write their message and exit)
					// - (3) The listener's drain function was called (which
					//       removes the listener from the Multicast and no
					//       new messages are enqueued for the listener) AND
					//       the Done() function for the listener was called
					//       without draining the channel (or never called)
					// - (4) The listener's channel was closed with go routines
					//       of this function still trying to write to the
					//       listener's channel -- all of those go routines
					//       will cause a panic (recovered here)
					//
					// There will be one panic for each undelivered message.
					//
					fmt.Println(fmt.Sprintf("Listener was closed ungracefully: %v", err))
				}
			}()

			l.c <- message

			// TODO: implement different strategies to handle a listener being at
			// capacity, selectable via some configuration setting. For example,
			// could choose to drop messages instead of having go routines in-flight.
			// This strategy would eliminate the possibility of the panics described above.

		}(listener)
	}
}

// closeListenerFn returns a function to handle
// the safe closure of a listener in a multicast.
func (m *Multicast[T]) closeListenerFn(listener *Listener[T]) DrainFunc {
	return func(timeout time.Duration) {
		// remove the listener from the list of listeners
		m.removeListener(listener)

		// close the listener's inbound message channel
		listener.close()

		// wait for the listener's inbound channel to drain
		listener.wait(timeout)
	}
}

// removeListener removes a listener from the Multicast's list of listeners.
func (m *Multicast[T]) removeListener(listener *Listener[T]) {
	m.mutexL.Lock()
	defer m.mutexL.Unlock()

	for i, l := range m.listeners {
		if listener.c == l.c {
			copy(m.listeners[i:], m.listeners[i+1:])
			m.listeners[len(m.listeners)-1] = nil
			m.listeners = m.listeners[:len(m.listeners)-1]
			break
		}
	}
}

// closeWriterFn returns a function to handle
// the safe closure of a writer in a multicast.
func (m *Multicast[T]) closeWriterFn(writer *Writer[T]) func() {
	return func() { writer.close() }
}
