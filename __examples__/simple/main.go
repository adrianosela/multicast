package main

import (
	"fmt"

	"github.com/adrianosela/multicast"
)

const defaultListenerCapacity = 10

func main() {
	m := multicast.New[string]()
	defer m.Close()

	for i := 0; i < 10; i++ {
		l, cancel := m.NewListener(defaultListenerCapacity)
		defer cancel()

		go func(jobID int, listener *multicast.Listener[string]) {
			fmt.Println(fmt.Sprintf("[L] [job %d] starting...", jobID))

			for data := range listener.C() {
				fmt.Println(fmt.Sprintf("[L] [job %d] got %s", jobID, data))
			}

			fmt.Println(fmt.Sprintf("[L] [job %d] ...done", jobID))
		}(i, l)
	}

	for writer := 0; writer < 1; writer++ {
		w, cancel := m.NewWriter()
		defer cancel()

		for message := 0; message < 1; message++ {
			err := w.Write(fmt.Sprintf("[writer %d] message %d", writer, message))
			if err != nil {
				fmt.Println(err)
				break
			}
		}
	}
}
