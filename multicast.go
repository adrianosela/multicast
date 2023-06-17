package multicast

import "sync"

const (
	defaultOutboundQueueBufferSize = 1000
)

// Multicast represents an object capable of broadcasting
// messages from multiple writers to multiple listeners.
type Multicast[T any] struct {
	// lock for listeners list
	mutex sync.Mutex

	// list of all the current message listeners.
	listeners []chan<- T

	// used as a FIFO queue for outbound messages
	// to be forwarded to all the listeners.
	outboundQueue chan T

	// used to signal when the outbound
	// queue is drained after closure
	outboundQueueDrained chan struct{}
}

// New returns a newly initialized Multicast of the specified type.
func New[T any]() *Multicast[T] {
	m := &Multicast[T]{
		// note: mutex requires no initialization
		listeners:            []chan<- T{},
		outboundQueue:        make(chan T, defaultOutboundQueueBufferSize),
		outboundQueueDrained: make(chan struct{}, 0),
	}
	go m.run()
	return m
}

// Close closes the multicast and blocks until all
// messages currently in the outbound queue are sent.
func (m *Multicast[T]) Close() {
	close(m.outboundQueue)
	<-m.outboundQueueDrained
	close(m.outboundQueueDrained)
}

// NewWriter returns a new message Writer for the multicast.
func (m *Multicast[T]) NewWriter() *Writer[T] {
	return newWriter[T](m.outboundQueue)
}

// NewListener returns a new message Listener for the multicast.
func (m *Multicast[T]) NewListener() *Listener[T] {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	l := newListener[T]()
	m.listeners = append(m.listeners, l.c)
	return l
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
}

// broadcast broadcasts a message to all listeners of the multicast.
func (m *Multicast[T]) broadcast(message T) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, listener := range m.listeners {
		go func(l chan<- T) { l <- message }(listener)
	}
}
