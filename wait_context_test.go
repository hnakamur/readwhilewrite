package readwhilewrite_test

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/hnakamur/readwhilewrite"
)

func TestReader_SetWaitContext(t *testing.T) {
	file, err := ioutil.TempFile("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	writerSteps := make([]chan struct{}, 4)
	for i := range writerSteps {
		writerSteps[i] = make(chan struct{})
	}

	reader1C := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())

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

			r.SetWaitContext(ctx)

			var buf [4096]byte
			for {
				n, err := r.Read(buf[:])
				if err == io.EOF {
					break
				}
				if err != nil {
					close(reader1C)
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

		cancel()
		<-reader1C

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

		return nil
	}()
	if err != nil {
		// You should log the error in production here.

		w.Abort()
	}
	w.Close()

	wg.Wait()

	want1 := "hello\nworld\n"
	want2 := "hello\nworld\ngoodbye\nsee you\n"

	got1 := string(buf1)
	if got1 != want1 {
		t.Errorf("Unexpected reader1 result, got=%s, want=%s", got1, want1)
	}

	got2 := string(buf2)
	if got2 != want2 {
		t.Errorf("Unexpected reader1 result, got=%s, want=%s", got2, want2)
	}

	if r1err != context.Canceled {
		t.Errorf("Unexpected reader1 error, got=%v, want=%v", r1err, context.Canceled)
	}

	if r2err != nil {
		t.Errorf("Unexpected reader1 error=%v", r2err)
	}
}
