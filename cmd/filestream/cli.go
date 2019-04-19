package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/jadr2ddude/filestream"
)

func main() {
	var decode bool
	var stream string
	var sopts filestream.StreamOptions
	var users bool
	var groups bool
	var perms bool
	var base string
	var list bool

	flag.BoolVar(&decode, "d", false, "decode a stream")
	flag.StringVar(&stream, "s", "-", "stream source/destination")
	flag.StringVar(&sopts.Compression, "z", "", "compression algo to use (gzip/lz4)")
	flag.IntVar(&sopts.CompressionLevel, "l", 0, "compression level")
	flag.BoolVar(&users, "permUser", false, "preserve owning user")
	flag.BoolVar(&groups, "permGroup", false, "preserve owning group")
	flag.BoolVar(&perms, "perms", false, "preserve permissions")
	flag.StringVar(&base, "C", ".", "base directory")
	flag.BoolVar(&list, "t", false, "list files & lengths instead of writing")
	flag.Parse()

	if decode {
		var sr io.ReadCloser
		if stream == "-" {
			sr = os.Stdin
		} else {
			u, err := url.Parse(stream)
			if err != nil {
				panic(err)
			}
			switch u.Scheme {
			case "", "file":
				f, err := os.Open(u.Path)
				if err != nil {
					panic(err)
				}
				sr = f
			case "http", "https":
				resp, err := http.Get(u.String())
				if err != nil {
					panic(err)
				}
				if resp.StatusCode != 200 {
					panic(fmt.Errorf("failed to download: %s", resp.Status))
				}
				sr = resp.Body
			default:
				panic(errors.New("unsupported url scheme"))
			}
		}
		defer sr.Close()
		d, err := filestream.NewReader(sr)
		if err != nil {
			panic(err)
		}
		if list {
			for d.Next() {
				f := d.File()
				n, err := io.Copy(ioutil.Discard, f)
				if err != nil {
					panic(err)
				}
				switch {
				case f.Opts().Permissions.IsRegular():
					fmt.Printf("%s (%d bytes)\n", f.Path(), n)
				case f.Opts().Permissions.IsDir():
					fmt.Printf("%s (dir)\n", f.Path())
				default:
					fmt.Printf("%s (special)\n", f.Path())
				}
			}
			if err = d.Err(); err != nil {
				panic(err)
			}
		} else {
			err = filestream.DecodeStream(d, filestream.DecodeOptions{
				Base:                base,
				PreservePermissions: perms,
				PreserveUser:        users,
				PreserveGroup:       groups,
			})
			if err != nil {
				panic(err)
			}
		}
		err = sr.Close()
		if err != nil {
			panic(err)
		}
	} else {
		var sw io.WriteCloser
		if stream == "-" {
			sw = os.Stdout
		} else {
			u, err := url.Parse(stream)
			if err != nil {
				panic(err)
			}
			switch u.Scheme {
			case "", "file":
				f, err := os.OpenFile(u.Path, os.O_CREATE|os.O_WRONLY, 0640)
				if err != nil {
					panic(err)
				}
				sw = f
			default:
				panic(errors.New("unsupported url scheme"))
			}
		}
		defer sw.Close()
		w, err := filestream.NewWriter(sw, sopts)
		if err != nil {
			panic(err)
		}
		for _, v := range flag.Args() {
			err = filestream.EncodeFiles(w, v, filestream.EncodeOptions{
				Base:               base,
				IncludePermissions: perms,
				IncludeUser:        users,
				IncludeGroup:       groups,
			})
			if err != nil {
				panic(err)
			}
		}
		err = w.Close()
		if err != nil {
			panic(err)
		}
		err = sw.Close()
		if err != nil {
			panic(err)
		}
	}
}
