package main

import (
	"fmt"
	"time"

	"github.com/adrianosela/multicast"
)

const (
	listenerCapacity  = 0
	listeners         = 10
	writers           = 10
	messagesPerWriter = 10
)

func main() {
	m := multicast.New[string]()
	defer m.Close()

	for i := 0; i < listeners; i++ {
		l, drain := m.NewListener(listenerCapacity)
		defer drain(time.Second * 5)

		go runListener(i, l)
	}

	// allow listener routines to come up
	time.Sleep(time.Millisecond * 10)

	for i := 0; i < writers; i++ {
		w, close := m.NewWriter()
		defer close()

		go runWriter(i, w)
	}

	// allow writer routines to come up
	time.Sleep(time.Millisecond * 10)
}

func runListener(jobID int, listener *multicast.Listener[string]) {
	defer listener.Done()

	fmt.Println(fmt.Sprintf("[L] [listener %d] starting...", jobID))
	for data := range listener.C() {
		fmt.Println(fmt.Sprintf("[L] [listener %d] got %s", jobID, data))
	}
	fmt.Println(fmt.Sprintf("[L] [listener %d] ...done", jobID))
}

func runWriter(jobID int, writer *multicast.Writer[string]) {
	for i := 0; i < messagesPerWriter; i++ {
		err := writer.Write(fmt.Sprintf("{ \"from\":\"writer %d\", \"body\": \"%d\" }", jobID, i))
		if err != nil {
			fmt.Println(fmt.Sprintf("[W] [writer %d] ERROR - failed to write: %v", jobID, err))
			break
		}
	}
}
