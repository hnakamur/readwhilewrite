package readwhilewrite_test

import (
	"encoding/hex"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/hnakamur/readwhilewrite"
)

func TestSendFileHTTP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := ioutil.TempFile("", "test")
		if err != nil {
			httpError(w, http.StatusInternalServerError)
			return
		}
		filename := file.Name()
		defer os.Remove(filename)

		w2 := readwhilewrite.NewWriter(file)

		rerrC := make(chan error, 1)
		go func() {
			defer close(rerrC)

			f, err := os.Open(filename)
			if err != nil {
				rerrC <- err
				return
			}
			defer f.Close()

			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Length", "81920")

			_, err = readwhilewrite.SendFileHTTP(r.Context(), w, f, w2)
			if err != nil {
				rerrC <- err
				return
			}
		}()

		rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

		buf := make([]byte, 4096)
		hexBuf := make([]byte, len(buf)*2)
		var n int64
		var n0 int
		for i := 0; i < 10; i++ {
			rnd.Read(buf)
			hex.Encode(hexBuf, buf)
			n0, err = w2.Write(hexBuf)
			if err != nil {
				httpError(w, http.StatusInternalServerError)
				return
			}
			n += int64(n0)
		}
		w2.Close()

		rerr := <-rerrC
		if rerr != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	if err != nil {
		t.Errorf("unexpected error for http client, %s", err)
	}
	content, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if len(content) != 81920 {
		t.Errorf("unexpected content length, got=%d, want=%d", len(content), 81920)
	}
}

func httpError(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
