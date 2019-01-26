package readwhilewrite_test

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/hnakamur/readwhilewrite"
)

func TestWriter_Abort(t *testing.T) {
	file, err := ioutil.TempFile("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	writerSteps := make([]chan struct{}, 4)
	for i := range writerSteps {
		writerSteps[i] = make(chan struct{})
	}

	w := readwhilewrite.NewWriter(file)

	var wg sync.WaitGroup
	wg.Add(2)

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

			i := 0
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

				<-writerSteps[i]
				i++
				if i == 2 {
					<-writerSteps[i]
					i++
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
			i := 0
			<-writerSteps[i]
			i++

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

				<-writerSteps[i]
				i++
			}
			return nil
		}()
	}()

	err = func() error {
		_, err := w.Write([]byte("hello\n"))
		if err != nil {
			return err
		}
		close(writerSteps[0])

		_, err = w.Write([]byte("world\n"))
		if err != nil {
			return err
		}
		close(writerSteps[1])

		_, err = w.Write([]byte("goodbye\n"))
		if err != nil {
			return err
		}
		close(writerSteps[2])

		_, err = w.Write([]byte("see you\n"))
		if err != nil {
			return err
		}
		close(writerSteps[3])

		return errors.New("intentional error for test")
	}()
	if err != nil {
		// You should log the error in production here.

		w.Abort()
	}
	w.Close()

	wg.Wait()
	if r1err != readwhilewrite.WriteAborted {
		t.Errorf("Unexpected reader1 error result, got=%v, want=%v", r1err, readwhilewrite.WriteAborted)
	}
	if r2err != readwhilewrite.WriteAborted {
		t.Errorf("Unexpected reader2 error result, got=%v, want=%v", r2err, readwhilewrite.WriteAborted)
	}
}
