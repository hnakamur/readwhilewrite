package readwhilewrite_test

import (
	"io"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/hnakamur/readwhilewrite"
)

func TestReader_Read(t *testing.T) {
	file, err := ioutil.TempFile("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	w := readwhilewrite.NewWriter(file)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()

		err := func() error {
			_, err := w.Write([]byte("hello\n"))
			if err != nil {
				return err
			}

			time.Sleep(time.Second)
			_, err = w.Write([]byte("world\n"))
			if err != nil {
				return err
			}

			time.Sleep(time.Second)
			_, err = w.Write([]byte("goodbye\n"))
			if err != nil {
				return err
			}

			time.Sleep(time.Second)
			_, err = w.Write([]byte("see you\n"))
			if err != nil {
				return err
			}

			return nil
		}()
		if err != nil {
			// You should log the error in production here.

			w.Abort()
		}
		w.Close()
	}()

	var buf1 []byte
	var r1err error
	go func() {
		defer wg.Done()

		r1err = func() error {
			f, err := os.Open(file.Name())
			if err != nil {
				return err
			}
			defer f.Close()
			r := readwhilewrite.NewReader(f, w)
			var buf [4096]byte
			for {
				n, err := r.Read(buf[:])
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}
				if n > 0 {
					buf1 = append(buf1, buf[:n]...)
				}
			}
			return nil
		}()
	}()

	var buf2 []byte
	var r2err error
	go func() {
		defer wg.Done()

		r2err = func() error {
			time.Sleep(1500 * time.Millisecond)
			f, err := os.Open(file.Name())
			if err != nil {
				return err
			}
			defer f.Close()
			r := readwhilewrite.NewReader(f, w)
			var buf [4096]byte
			for {
				n, err := r.Read(buf[:])
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}
				if n > 0 {
					buf2 = append(buf2, buf[:n]...)
				}
			}
			return nil
		}()
	}()

	wg.Wait()

	want := "hello\nworld\ngoodbye\nsee you\n"

	got1 := string(buf1)
	if got1 != want {
		t.Errorf("Unexpected reader1 result, got=%s, want=%s", got1, want)
	}

	got2 := string(buf2)
	if got2 != want {
		t.Errorf("Unexpected reader1 result, got=%s, want=%s", got2, want)
	}

	if r1err != nil {
		t.Errorf("Unexpected reader1 error=%v", r1err)
	}

	if r2err != nil {
		t.Errorf("Unexpected reader1 error=%v", r2err)
	}
}
