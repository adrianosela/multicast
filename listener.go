package multicast

type Listener[T any] struct {
	c chan T
}

func newListener[T any]() *Listener[T] {
	return &Listener[T]{
		c: make(chan T),
	}
}

func (l Listener[T]) C() <-chan T {
	return l.c
}
