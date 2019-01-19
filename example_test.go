package readwhilewrite_test

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/hnakamur/readwhilewrite"
)

func Example() {
	file, err := ioutil.TempFile("", "test")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())

	w := readwhilewrite.NewWriter(file)

	done := make(chan struct{})

	go func() {
		defer close(done)

		err := func() error {
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
					os.Stdout.Write(buf[:n])
				}
			}
			return nil
		}()
		if err != nil {
			log.Fatalf("Unexpected reader error=%v", err)
		}
	}()

	err = func() error {
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

	<-done

	// Output:
	// hello
	// world
	// goodbye
	// see you
}
