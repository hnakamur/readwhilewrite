package readwhilewrite

import (
	"io"
)

// Reader is a reader which waits writes by the writer
// when the reader gets an EOF. When the reader gets
// an EOF after Close is called for the writer, then
// it is treated as a real EOF.
type Reader struct {
	io.Reader
	w       *Writer
	updates <-chan struct{}
}

// NewReader creates a new reader which waits writes
// by the writer.
func NewReader(r io.Reader, w *Writer) *Reader {
	return &Reader{Reader: r, w: w, updates: w.subscribe()}
}

// Read implements the io.Reader interface.
//
// When Abort and then Close is called for the writer,
// WriteAborted is returned as err.
func (r *Reader) Read(p []byte) (n int, err error) {
retry:
	n, err = r.Reader.Read(p)
	if err == io.EOF {
		if r.w.isClosed() {
			if r.w.err != nil {
				err = r.w.err
			}
			return
		}
		err = nil

		if n == 0 {
			<-r.updates
			goto retry
		}
	}
	return
}
