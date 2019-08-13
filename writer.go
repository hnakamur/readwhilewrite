package readwhilewrite

import (
	"errors"
	"io"
	"log"
	"sync"
	"sync/atomic"
)

// Writer is a writer which notifies readers of writes.
// Writer implements the io.WriteCloser interface.
type Writer struct {
	w        io.WriteCloser
	mu       sync.Mutex
	cond     sync.Cond
	written  int64
	closed   int32
	canceled int32
}

// ErrWriterCanceled is an error which is returned to Read of readers
// when Cancel is called for a writer.
var ErrWriterCanceled = errors.New("writer canceled")

// NewWriter creates a notifying writer.
func NewWriter(w io.WriteCloser) *Writer {
	w2 := &Writer{
		w: w,
	}
	w2.cond.L = &w2.mu
	return w2
}

// Write implements the io.Writeer interface.
// Write notify readers if n > 0.
func (w *Writer) Write(p []byte) (n int, err error) {
	n, err = w.w.Write(p)
	if err != nil {
		return
	}
	if n > 0 {
		atomic.AddInt64(&w.written, int64(n))
		log.Printf("before broadcast writer write")
		w.cond.Broadcast()
	}
	return
}

// Close implements the io.Closer interface.
// Close closes the underlying writer.
// When readers got EOF after Close is called, it is the real EOF.
func (w *Writer) Close() error {
	err := w.w.Close()
	atomic.StoreInt32(&w.closed, 1)
	log.Printf("before broadcast writer close")
	w.cond.Broadcast()
	return err
}

// Cancel is used to make all readers stop reading when some error
// happens in the Writer. You must call Close after Cancel for
// cleanup.
func (w *Writer) Cancel() {
	atomic.StoreInt32(&w.canceled, 1)
	log.Printf("before broadcast writer cancel")
	w.cond.Broadcast()
}
