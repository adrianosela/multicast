package multicast

import (
	"context"
	"time"
)

// Listener represents a multicast listener.
type Listener[T any] struct {
	// channel where every message written by any of
	// the Multicast's Writer(s) will be forwarded to.
	c chan T

	// used to signal when the inbound
	// queue is drained after closure.
	inboundQueueDrained chan struct{}
}

// newListener returns a new Listener with a given capacity.
func newListener[T any](capacity int) *Listener[T] {
	return &Listener[T]{
		c:                   make(chan T, capacity),
		inboundQueueDrained: make(chan struct{}),
	}
}

// C returns the Listener's (receive-only) channel.
func (l *Listener[T]) C() <-chan T {
	return l.c
}

// Done is used to signal that the Listener
// finished draining its inbound channel.
func (l *Listener[T]) Done() {
	l.inboundQueueDrained <- struct{}{}
	close(l.inboundQueueDrained)
}

// close closes the Listener's channel.
func (l *Listener[T]) close() {
	close(l.c)
}

// wait waits for a listener to be drained or timeout
func (l *Listener[T]) wait(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		break
	case <-l.inboundQueueDrained:
		break
	}
}
