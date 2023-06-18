package multicast

// Listener represents a multicast listener.
type Listener[T any] struct {
	c chan T
}

// newListener returns a new Listener with a given capacity.
func newListener[T any](capacity int) *Listener[T] {
	return &Listener[T]{c: make(chan T, capacity)}
}

// close closes the Listener's channel.
func (l *Listener[T]) close() {
	close(l.c)
}

// C returns the Listener's (receive-only) channel.
func (l *Listener[T]) C() <-chan T {
	return l.c
}
