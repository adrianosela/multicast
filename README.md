# multicast

[![Go Report Card](https://goreportcard.com/badge/github.com/adrianosela/multicast)](https://goreportcard.com/report/github.com/adrianosela/multicast)
[![Documentation](https://godoc.org/github.com/adrianosela/multicast?status.svg)](https://godoc.org/github.com/adrianosela/multicast)
[![license](https://img.shields.io/github/license/adrianosela/multicast)](https://github.com/adrianosela/multicast/blob/master/LICENSE)

Multiple writer, multiple listener message channel in Go.

- Any message from any writer goes to all listeners
- Supports adding and removing writers and listeners at runtime

### Example

#### Initialization

```
m := multicast.New[string]()
defer m.Close()
```

#### Adding a Listener

```
listener, close := m.NewListener(listenerCapacity)
defer close(time.Second * 5)

go func() {
	defer listener.Done()

	for message := range listener.C() {
		// ... do something useful ...
	}
}()
```

#### Adding a Writer

```
writer, close := m.NewWriter()
defer close()

err := writer.Write(payload)
if err != nil {
	// ... handle error ...
}
```
