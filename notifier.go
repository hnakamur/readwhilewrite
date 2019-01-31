package readwhilewrite

import (
	"sync"
)

type notifier struct {
	mu       sync.Mutex
	channels []chan struct{}
	closed   bool
}

func (n *notifier) Subscribe() <-chan struct{} {
	c := make(chan struct{}, 1)
	n.mu.Lock()
	if n.closed {
		close(c)
	}
	n.channels = append(n.channels, c)
	n.mu.Unlock()
	return c
}

func (n *notifier) Unsubscribe(c <-chan struct{}) {
	n.mu.Lock()
	for i, channel := range n.channels {
		if channel == c {
			n.channels = append(n.channels[:i], n.channels[i+1:]...)
			break
		}
	}
	n.mu.Unlock()
}

func (n *notifier) Notify() {
	n.mu.Lock()
	for _, c := range n.channels {
		select {
		case c <- struct{}{}:
		default:
		}
	}
	n.mu.Unlock()
}

func (n *notifier) Close() {
	n.mu.Lock()
	for _, c := range n.channels {
		close(c)
	}
	n.closed = true
	n.mu.Unlock()
}
