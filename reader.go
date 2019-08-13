package readwhilewrite

import (
	"errors"
	"io"
	"sync/atomic"
)

// Reader is a reader which waits writes by the writer
// when the reader gets an EOF. When the reader gets
// an EOF after Close is called for the writer, then
// it is treated as a real EOF.
// Reader implements the io.ReadCloser interface.
type Reader struct {
	r        io.ReadCloser
	w        *Writer
	canceled int32
}

// ErrReaderCanceled is an error which is returned to Read of readers
// when Cancel is called for a reader.
var ErrReaderCanceled = errors.New("reader canceled")

// NewReader creates a new reader which waits writes
// by the writer.
func NewReader(r io.ReadCloser, w *Writer) *Reader {
	return &Reader{
		r: r,
		w: w,
	}
}

// Read implements the io.Reader interface.
//
// When Abort and then Close is called for the writer,
// WriteAborted is returned as err.
//
// When SetWaitContext was called before calling Read and
// the context is done during waiting writes by the writer,
// the error from the context is returned as err.
func (r *Reader) Read(p []byte) (n int, err error) {
	for {
		n, err = r.r.Read(p)
		if err == io.EOF {
			if atomic.LoadInt32(&r.canceled) == 1 {
				return n, ErrReaderCanceled
			}
			if atomic.LoadInt32(&r.w.canceled) == 1 {
				return n, ErrWriterCanceled
			}
			if atomic.LoadInt32(&r.w.closed) == 1 {
				return n, io.EOF
			}

			err = nil
			if n == 0 {
				r.w.cond.L.Lock()
				written := r.w.written
				for {
					r.w.cond.Wait()
					if atomic.LoadInt32(&r.canceled) == 1 {
						return 0, ErrReaderCanceled
					}
					if atomic.LoadInt32(&r.w.canceled) == 1 {
						return 0, ErrWriterCanceled
					}
					if r.w.written > written {
						break
					}
					if atomic.LoadInt32(&r.w.closed) == 1 {
						return 0, io.EOF
					}
				}
				r.w.cond.L.Unlock()
				continue
			}
		}
		return
	}
}

// Close implements the io.Closer interface.
//
// Close closes the underlying reader.
func (r *Reader) Close() error {
	return r.r.Close()
}

// Cancel cancels this reader.
// ErrReaderCanceled is returned from the Read method of the reader.
func (r *Reader) Cancel() {
	atomic.StoreInt32(&r.canceled, 1)
	r.w.cond.Broadcast()
}
