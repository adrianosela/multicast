package main

import (
	"fmt"
	"time"

	"github.com/adrianosela/multicast"
)

func main() {
	m := multicast.New[string]()
	defer m.Close()

	for job := 0; job < 5; job++ {
		go func(jobID int) {
			l := m.NewListener()
			for data := range l.C() {
				fmt.Println(fmt.Sprintf("[job %d] got %s", jobID, data))
			}
		}(job)
	}

	time.Sleep(time.Second * 1)

	for writer := 0; writer < 5; writer++ {
		w := m.NewWriter()
		defer w.Close()

		for message := 0; message < 5; message++ {
			w.Write(fmt.Sprintf("[writer %d] message %d", writer, message))
		}
	}

	time.Sleep(time.Second * 1)
}
