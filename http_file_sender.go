package readwhilewrite

import (
	"context"
	"io"
	"net/http"
	"os"
)

// SendFileHTTP serve a file as a HTTP response while fw is writing to the same file.
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

		if fw.isClosed() {
			if fw.err != nil {
				err = fw.err
				return
			}
			if n < fw.getWritten() {
				continue
			} else {
				return
			}
		}

		select {
		case <-wroteC:
			continue
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}
}
