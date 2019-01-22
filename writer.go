package readwhilewrite

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
)

// Writer is a writer which notifies readers of writes.
type Writer struct {
	io.WriteCloser

	closed int32
	err    error

	mu       sync.Mutex
	channels []chan struct{}
}

// WriteAborted is an error which is returned to Read of readers
// when Abort and then Close is called for a writer.
var WriteAborted = errors.New("write aborted")

// NewWriter creates a notifying writer.
func NewWriter(w io.WriteCloser) *Writer {
	return &Writer{WriteCloser: w}
}

// Write implements the io.Writer interface.
// Write notify readers if n > 0.
func (w *Writer) Write(p []byte) (n int, err error) {
	n, err = w.WriteCloser.Write(p)
	if err != nil {
		return
	}
	if n > 0 {
		w.mu.Lock()
		for _, c := range w.channels {
			select {
			case c <- struct{}{}:
			default:
			}
		}
		w.mu.Unlock()
	}
	return
}

// Close closes the underlying writer.
// When readers got EOF after Close is called, it is the real EOF.
func (w *Writer) Close() error {
	err := w.WriteCloser.Close()
	atomic.StoreInt32(&w.closed, 1)

	w.mu.Lock()
	for _, c := range w.channels {
		close(c)
	}
	w.mu.Unlock()

	return err
}

// Abort is used to make readers stop reading when some error
// happens in the Writer. You must call Close after Abort.
func (w *Writer) Abort() {
	w.err = WriteAborted
}

func (w *Writer) subscribe() <-chan struct{} {
	c := make(chan struct{}, 1)
	w.mu.Lock()
	w.channels = append(w.channels, c)
	w.mu.Unlock()
	return c
}
