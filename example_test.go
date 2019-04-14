package filestream_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/jadr2ddude/filestream"
)

func Example_minimal() {
	// Write some files to a stream.
	var buf bytes.Buffer
	w, err := filestream.NewWriter(&buf, filestream.StreamOptions{})
	if err != nil {
		log.Fatal(err)
	}
	var files = []struct {
		Name, Body string
	}{
		{"hello.txt", "Hello World!"},
		{"smile.txt", "☺"},
	}
	for _, file := range files {
		fw, err := w.File(file.Name, filestream.FileOptions{})
		if err != nil {
			log.Fatal(err)
		}
		_, err = fw.Write([]byte(file.Body))
		if err != nil {
			log.Fatal(err)
		}
		err = fw.Close()
		if err != nil {
			log.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		log.Fatal(err)
	}

	// Read files back out of stream.
	r, err := filestream.NewReader(&buf)
	if err != nil {
		log.Fatal(err)
	}
	for r.Next() {
		f := r.File()
		dat, err := ioutil.ReadAll(f)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("File %q: %s\n", f.Path(), string(dat))
	}

	// Output:
	// File "hello.txt": Hello World!
	// File "smile.txt": ☺
}
