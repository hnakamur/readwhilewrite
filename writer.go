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
	mu      sync.Mutex
	cond    sync.Cond
	written int64
	closed  int32
	err     error
}

// WriteAborted is an error which is returned to Read of readers
// when Abort and then Close is called for a writer.
var WriteAborted = errors.New("write aborted")

// NewWriter creates a notifying writer.
func NewWriter(w io.WriteCloser) *Writer {
	nww := &Writer{WriteCloser: w}
	nww.cond.L = &nww.mu
	return nww
}

// Write implements the io.Writer interface.
// Write notify readers if n > 0.
func (w *Writer) Write(p []byte) (n int, err error) {
	n, err = w.WriteCloser.Write(p)
	if err != nil {
		return
	}
	if n > 0 {
		w.cond.L.Lock()
		w.written += int64(n)
		w.cond.L.Unlock()
		w.cond.Broadcast()
	}
	return
}

// Close closes the underlying writer.
// When readers got EOF after Close is called, it is the real EOF.
func (w *Writer) Close() error {
	err := w.WriteCloser.Close()
	atomic.StoreInt32(&w.closed, 1)
	return err
}

// Abort is used to make readers stop reading when some error
// happens in the Writer. You must call Close after Abort.
func (w *Writer) Abort() {
	w.cond.L.Lock()
	w.err = WriteAborted
	w.cond.L.Unlock()
	w.cond.Broadcast()
}
