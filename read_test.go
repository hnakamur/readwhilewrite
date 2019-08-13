package readwhilewrite_test

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/hnakamur/readwhilewrite"
)

func TestReader_Read(t *testing.T) {
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

		return nil
	}()
	if err != nil {
		// You should log the error in production here.

		w.Cancel()
	}
	w.Close()

	wg.Wait()

	want := "hello\nworld\ngoodbye\nsee you\n"

	got1 := string(buf1)
	if got1 != want {
		t.Errorf("Unexpected reader1 result, got=%s, want=%s", got1, want)
	}

	got2 := string(buf2)
	if got2 != want {
		t.Errorf("Unexpected reader2 result, got=%s, want=%s", got2, want)
	}

	if r1err != nil {
		t.Errorf("Unexpected reader1 error=%v", r1err)
	}

	if r2err != nil {
		t.Errorf("Unexpected reader2 error=%v", r2err)
	}
}

func TestReader_Cancel(t *testing.T) {
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

				if i < len(writerSteps)-1 {
					<-writerSteps[i]
					i++
				}
				if i == 2 {
					if i < len(writerSteps)-1 {
						<-writerSteps[i]
						i++
					}
				}
			}
			return nil
		}()
	}()

	var buf2 []byte
	var r2err error
	r2C := make(chan *readwhilewrite.Reader)
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
			r2C <- r

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

				if i < len(writerSteps)-1 {
					<-writerSteps[i]
					i++
				}
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
		r2 := <-r2C
		r2.Cancel()

		_, err = w.Write([]byte("see you\n"))
		if err != nil {
			return err
		}
		close(writerSteps[3])

		return nil
	}()
	if err != nil {
		// You should log the error in production here.

		w.Cancel()
	}
	w.Close()

	wg.Wait()

	got1 := string(buf1)
	want1 := "hello\nworld\ngoodbye\nsee you\n"
	if got1 != want1 {
		t.Errorf("Unexpected reader1 result, got=%s, want=%s", got1, want1)
	}

	got2 := string(buf2)
	want2 := "hello\nworld\ngoodbye\n"
	if !strings.HasPrefix(got2, want2) {
		t.Errorf("Unexpected reader2 result strings.HasPrefix(got2, want2) must be true, got=%s, want=%s", got2, want2)
	}

	if r1err != nil {
		t.Errorf("Unexpected reader1 error=%v", r1err)
	}

	if r2err != readwhilewrite.ErrReaderCanceled {
		t.Errorf("Unexpected reader2 error result, got=%v, want=%v", r2err, readwhilewrite.ErrReaderCanceled)
	}
}
