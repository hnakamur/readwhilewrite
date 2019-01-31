package readwhilewrite

import (
	"errors"
	"io"
)

// Writer is a writer which notifies readers of writes.
type Writer struct {
	io.WriteCloser
	err      error
	notifier notifier
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
		w.notifier.Notify()
	}
	return
}

// Close closes the underlying writer.
// When readers got EOF after Close is called, it is the real EOF.
func (w *Writer) Close() error {
	err := w.WriteCloser.Close()
	w.notifier.Close()
	return err
}

// Abort is used to make readers stop reading when some error
// happens in the Writer. You must call Close after Abort.
func (w *Writer) Abort() {
	w.err = WriteAborted
}

func (w *Writer) subscribe() <-chan struct{} {
	return w.notifier.Subscribe()
}

func (w *Writer) unsubscribe(c <-chan struct{}) {
	w.notifier.Unsubscribe(c)
}
