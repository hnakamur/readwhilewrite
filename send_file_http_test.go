package readwhilewrite_test

import (
	"context"
	"encoding/hex"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/hnakamur/readwhilewrite"
)

func TestSendFileHTTP(t *testing.T) {
	t.Run("normalCase", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			file, err := ioutil.TempFile("", "test")
			if err != nil {
				httpError(w, http.StatusInternalServerError)
				return
			}
			filename := file.Name()
			defer os.Remove(filename)

			w2 := readwhilewrite.NewWriter(file)

			go func() {
				defer w2.Close()
				rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

				buf := make([]byte, 4096)
				hexBuf := make([]byte, len(buf)*2)
				var n int64
				for i := 0; i < 10; i++ {
					rnd.Read(buf)
					hex.Encode(hexBuf, buf)
					n0, err := w2.Write(hexBuf)
					if err != nil {
						httpError(w, http.StatusInternalServerError)
						return
					}
					n += int64(n0)
				}
			}()

			f, err := os.Open(filename)
			if err != nil {
				t.Fatal(err)
				return
			}
			defer f.Close()

			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Length", "81920")

			_, err = readwhilewrite.SendFileHTTP(r.Context(), w, f, w2)
			if err != nil {
				t.Fatal(err)
				return
			}
		}))
		defer ts.Close()

		res, err := http.Get(ts.URL)
		if err != nil {
			t.Errorf("unexpected error for http client, %s", err)
		}
		content, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		wantLen := 4096 * 2 * 10
		if len(content) != wantLen {
			t.Errorf("unexpected content length, got=%d, want=%d", len(content), wantLen)
		}
	})

	t.Run("readerCancel", func(t *testing.T) {
		sendErrC := make(chan error, 1)
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var sendErr error
			defer func() {
				sendErrC <- sendErr
			}()

			file, err := ioutil.TempFile("", "test")
			if err != nil {
				httpError(w, http.StatusInternalServerError)
				return
			}
			filename := file.Name()
			defer os.Remove(filename)

			w2 := readwhilewrite.NewWriter(file)
			ctx, cancel := context.WithCancel(r.Context())
			defer cancel()

			go func() {
				defer w2.Close()
				rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

				buf := make([]byte, 4096)
				hexBuf := make([]byte, len(buf)*2)
				var n int64
				for i := 0; i < 10; i++ {
					rnd.Read(buf)
					hex.Encode(hexBuf, buf)
					n0, err := w2.Write(hexBuf)
					if err != nil {
						httpError(w, http.StatusInternalServerError)
						return
					}
					n += int64(n0)
					if i == 3 {
						cancel()
					}
				}
			}()

			f, err := os.Open(filename)
			if err != nil {
				t.Fatal(err)
				return
			}
			defer f.Close()

			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Length", "81920")

			_, sendErr = readwhilewrite.SendFileHTTP(ctx, w, f, w2)
			if sendErr != nil {
				return
			}
		}))
		defer ts.Close()

		res, err := http.Get(ts.URL)
		if err != nil {
			t.Errorf("unexpected error for http client sending request: %v", err)
		}
		defer res.Body.Close()
		_, err = ioutil.ReadAll(res.Body)
		if err != nil && err != io.ErrUnexpectedEOF {
			t.Errorf("unexpected error for http client reading response body: %v", err)
		}

		sendErr := <-sendErrC
		if sendErr != context.Canceled {
			t.Errorf("unexpected SendFileHTTP, got=%v, want=%v", sendErr, context.Canceled)
		}
	})

	t.Run("writerCancel", func(t *testing.T) {
		const clientCount = 3
		sendErrC := make(chan error, clientCount)
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var err error
			defer func() {
				sendErrC <- err
			}()
			var file *os.File

			file, err = ioutil.TempFile("", "test")
			if err != nil {
				httpError(w, http.StatusInternalServerError)
				return
			}
			filename := file.Name()
			defer os.Remove(filename)

			w2 := readwhilewrite.NewWriter(file)

			go func() {
				defer w2.Close()
				rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

				buf := make([]byte, 4096)
				hexBuf := make([]byte, len(buf)*2)
				var n int64
				for i := 0; i < 10; i++ {
					rnd.Read(buf)
					hex.Encode(hexBuf, buf)
					n0, err := w2.Write(hexBuf)
					if err != nil {
						httpError(w, http.StatusInternalServerError)
						return
					}
					n += int64(n0)
					if i == 6 {
						w2.Cancel()
					}
				}
			}()

			f, err := os.Open(filename)
			if err != nil {
				return
			}
			defer f.Close()

			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Length", "81920")

			_, err = readwhilewrite.SendFileHTTP(r.Context(), w, f, w2)
			if err != nil {
				return
			}
		}))
		defer ts.Close()

		var wg sync.WaitGroup
		for i := 0; i < clientCount; i++ {
			wg.Add(1)
			go func(clientID int) {
				defer wg.Done()
				res, err := http.Get(ts.URL)
				if err != nil {
					t.Errorf("unexpected error for http client #%d sending request: %v", clientID, err)
				}
				defer res.Body.Close()
				_, err = ioutil.ReadAll(res.Body)
				if err != nil && err != io.ErrUnexpectedEOF {
					t.Errorf("unexpected error for http client #%d reading response body: %v", clientID, err)
				}
			}(i)
		}
		wg.Wait()

		for i := 0; i < clientCount; i++ {
			sendErr := <-sendErrC
			if sendErr != readwhilewrite.ErrWriterCanceled {
				t.Errorf("unexpected SendFileHTTP, got=%v, want=%v", sendErr, readwhilewrite.ErrWriterCanceled)
			}
		}
	})
}

func httpError(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
