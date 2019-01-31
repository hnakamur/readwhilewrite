package readwhilewrite

import (
	"context"
	"io"
	"net/http"
	"os"
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
	wroteC := fw.subscribe()
	defer fw.unsubscribe(wroteC)

	var n1 int64
	for {
		n1, err = io.Copy(w, file)
		n += n1
		if err != nil && err != io.EOF {
			return
		}

		select {
		case _, ok := <-wroteC:
			if ok {
				continue
			}

			if fw.err != nil {
				err = fw.err
				return
			}

			n1, err = io.Copy(w, file)
			n += n1
			return
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}
}
