package readwhilewrite_test

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/hnakamur/readwhilewrite"
)

func TestWriter_Abort(t *testing.T) {
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

			return errors.New("intentional error for test")
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
			r := readwhilewrite.NewReader(f, w)
			defer r.Close()
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
			r := readwhilewrite.NewReader(f, w)
			defer r.Close()
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
	if r1err != readwhilewrite.WriteAborted {
		t.Errorf("Unexpected reader1 error result, got=%v, want=%v", r1err, readwhilewrite.WriteAborted)
	}
	if r2err != readwhilewrite.WriteAborted {
		t.Errorf("Unexpected reader2 error result, got=%v, want=%v", r2err, readwhilewrite.WriteAborted)
	}
}
