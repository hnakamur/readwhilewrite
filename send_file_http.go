package readwhilewrite

import (
	"context"
	"io"
	"net/http"
	"os"
	"sync/atomic"
)

// SendFileHTTP serves a file as a HTTP response while fw is writing to the same file.
//
// Once it gets an EOF, it waits more writes by the writer. If the ctx is done while
// waiting, SendFileHTTP returns. Typically you want to pass r.Context() as ctx for
// r *http.Request.
//
// If you set the Content-Length header before calling SendFileHTTP, the sendfile
// system call is used on Linux.
func SendFileHTTP(ctx context.Context, w http.ResponseWriter, file *os.File, fw *Writer) (n int64, err error) {
	var canceled int32
	done := make(chan struct{})
	ctxErrC := make(chan error, 1)
	defer func() {
		close(done)
		err2 := <-ctxErrC
		if err2 != nil {
			err = err2
		}
	}()

	go func() {
		select {
		case <-ctx.Done():
			ctxErrC <- ctx.Err()
			atomic.StoreInt32(&canceled, 1)
			fw.cond.Broadcast()
			return
		case <-done:
			close(ctxErrC)
		}
	}()

	for {
		var n1 int64
		n1, err = io.Copy(w, file)
		n += n1
		if err != nil && err != io.EOF {
			return
		}
		if atomic.LoadInt32(&canceled) == 1 {
			return
		}
		if atomic.LoadInt32(&fw.canceled) == 1 {
			return n, ErrWriterCanceled
		}
		if atomic.LoadInt32(&fw.closed) == 1 {
			return
		}
		if n1 > 0 {
			continue
		}

		fw.cond.L.Lock()
		written := atomic.LoadInt64(&fw.written)
		for {
			fw.cond.Wait()
			if atomic.LoadInt32(&canceled) == 1 {
				fw.cond.L.Unlock()
				return
			}
			if atomic.LoadInt32(&fw.canceled) == 1 {
				fw.cond.L.Unlock()
				return n, ErrWriterCanceled
			}
			if atomic.LoadInt64(&fw.written) > written {
				break
			}
			if atomic.LoadInt32(&fw.closed) == 1 {
				fw.cond.L.Unlock()
				return
			}
		}
		fw.cond.L.Unlock()
	}
}
