package filestream_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jadr2ddude/filestream"
)

type testFile struct {
	Path string
	Dir  bool
	Data string
	Opts filestream.FileOptions
}

func TestRoundTrip(t *testing.T) {
	tbl := []struct {
		StreamOpts filestream.StreamOptions
		Files      []testFile
	}{
		{
			Files: []testFile{
				testFile{
					Path: "/",
					Dir:  true,
				},
				testFile{
					Path: "/hello.txt",
					Data: "hello world",
				},
			},
		},
		{
			Files: []testFile{
				testFile{
					Path: "/",
					Dir:  true,
				},
				testFile{
					Path: "/hello.txt",
					Data: "hello world",
					Opts: filestream.FileOptions{
						User:        "usr",
						Group:       "grp",
						Permissions: 0644,
					},
				},
			},
		},
		{
			StreamOpts: filestream.StreamOptions{
				Compression: "gzip",
			},
			Files: []testFile{
				testFile{
					Path: "/",
					Dir:  true,
				},
				testFile{
					Path: "/hello.txt",
					Data: "hello world",
				},
			},
		},
		{
			StreamOpts: filestream.StreamOptions{
				Compression:      "gzip",
				CompressionLevel: 9,
			},
			Files: []testFile{
				testFile{
					Path: "/",
					Dir:  true,
				},
				testFile{
					Path: "/hello.txt",
					Data: "hello world",
				},
			},
		},
		{
			StreamOpts: filestream.StreamOptions{
				Compression: "lz4",
			},
			Files: []testFile{
				testFile{
					Path: "/",
					Dir:  true,
				},
				testFile{
					Path: "/hello.txt",
					Data: "hello world",
				},
			},
		},
	}
	for _, c := range tbl {
		var wg sync.WaitGroup
		pr, pw := io.Pipe()
		defer pr.Close()
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer pw.Close()

			w, err := filestream.NewWriter(pw, c.StreamOpts)
			if err != nil {
				pw.CloseWithError(err)
				return
			}

			for i, _ := range c.Files {
				v := &c.Files[i]
				if v.Dir {
					err = w.Directory(v.Path, v.Opts)
					if err != nil {
						pw.CloseWithError(err)
						return
					}
					v.Opts.Permissions |= os.ModeDir
				} else {
					fw, err := w.File(v.Path, v.Opts)
					if err != nil {
						pw.CloseWithError(err)
						return
					}
					n, err := fw.Write([]byte(v.Data))
					if err != nil {
						pw.CloseWithError(err)
						return
					}
					if n < len(v.Data) {
						pw.CloseWithError(errors.New("partial write with no error"))
						return
					}
					err = fw.Close()
					if err != nil {
						pw.CloseWithError(err)
						return
					}
				}
			}

			err = w.Close()
			if err != nil {
				pw.CloseWithError(err)
				return
			}
		}()
		r, err := filestream.NewReader(pr)
		if err != nil {
			t.Fatalf("failed to open reader: %s", err)
		}
		res := []testFile{}
		for r.Next() {
			fr := r.File()

			var buf bytes.Buffer
			_, err = io.Copy(&buf, fr)
			if err != nil {
				t.Fatalf("failed to read: %s", err)
			}

			res = append(res, testFile{
				Path: fr.Path(),
				Dir:  fr.IsDir(),
				Data: buf.String(),
				Opts: fr.Opts(),
			})
		}
		if err := r.Err(); err != nil {
			t.Fatalf("failed to get next file reader: %s", err)
		}
		wg.Wait()
		if diff := cmp.Diff(c.Files, res); diff != "" {
			t.Errorf("data corrupted through stream: (-in +out): %s\n", diff)
		}
	}
}
