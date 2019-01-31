package readwhilewrite

import (
	"context"
	"io"
	"net/http"
	"os"
)

type HTTPFileSender struct {
	file *os.File
	w    *Writer
}

func NewHTTPFileSender(file *os.File, w *Writer) *HTTPFileSender {
	return &HTTPFileSender{
		file: file,
		w:    w,
	}
}

func (s *HTTPFileSender) Send(ctx context.Context, w http.ResponseWriter) (n int64, err error) {
	wroteC := s.w.subscribe()
	defer s.w.unsubscribe(wroteC)

	var n1 int64
	for {
		n1, err = io.Copy(w, s.file)
		n += n1
		if err != nil && err != io.EOF {
			return
		}

		if s.w.isClosed() {
			if s.w.err != nil {
				err = s.w.err
				return
			}
			if n < s.w.getWritten() {
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
