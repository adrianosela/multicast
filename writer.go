package multicast

type Writer[T any] struct {
	c chan T
}

func newWriter[T any](target chan<- T) *Writer[T] {
	w := &Writer[T]{c: make(chan T)}
	go func() {
		for message := range w.c {
			target <- message
		}
	}()
	return w
}

func (w Writer[T]) Close() {
	close(w.c)
}

func (w Writer[T]) Write(message T) {
	w.c <- message
}
