package readwhilewrite

import (
	"context"
	"io"
)

// Reader is a reader which waits writes by the writer
// when the reader gets an EOF. When the reader gets
// an EOF after Close is called for the writer, then
// it is treated as a real EOF.
type Reader struct {
	io.ReadCloser
	w           *Writer
	updates     <-chan struct{}
	waitContext context.Context
}

// NewReader creates a new reader which waits writes
// by the writer.
func NewReader(r io.ReadCloser, w *Writer) *Reader {
	return &Reader{
		ReadCloser: r,
		w:          w,
		updates:    w.subscribe(),
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
		n, err = r.ReadCloser.Read(p)
		if err == io.EOF {
			err = nil
			if n == 0 {
				var done <-chan struct{}
				if r.waitContext != nil {
					done = r.waitContext.Done()
				}

				select {
				case _, ok := <-r.updates:
					if ok {
						continue
					}

					if r.w.err != nil {
						err = r.w.err
					} else {
						err = io.EOF
					}
					return
				case <-done:
					err = r.waitContext.Err()
					return
				}
			}
		}
		return
	}
}

// Close implements the io.Closer interface.
//
// Close closes the underlying reader.
func (r *Reader) Close() error {
	err := r.ReadCloser.Close()
	r.w.unsubscribe(r.updates)
	return err
}

// SetWaitContext sets the context for waiting writes by the writer
// after the reader received a temporary EOF in Read.
//
// Note SetWaitContext does not set a deadline for Read of the underlying
// reader.  If you want to set a deadline, you need to call an appropriate
// method for the underlying reader yourself, for example SetReadDeadline
// of *os.File.
//
// Also note SetReadDeadline of *os.File is not supported for ordinal files
// on most systems. See document of SetDeadline of *os.File.
func (r *Reader) SetWaitContext(ctx context.Context) {
	r.waitContext = ctx
}
