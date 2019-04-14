package filestream

import (
	"compress/gzip"
	"errors"
	"io"
	"io/ioutil"

	"github.com/pierrec/lz4"
)

func decompress(algo string, src io.Reader) (io.ReadCloser, error) {
	switch algo {
	case "gzip":
		return gzip.NewReader(src)
	case "lz4":
		return ioutil.NopCloser(lz4.NewReader(src)), nil
	default:
		return nil, errors.New("unsupported compression algorithm")
	}
}

func compress(algo string, level int, dst io.Writer) (io.WriteCloser, error) {
	switch algo {
	case "gzip":
		if level == 0 {
			return gzip.NewWriter(dst), nil
		}
		return gzip.NewWriterLevel(dst, level)
	case "lz4":
		w := lz4.NewWriter(dst)
		w.Header.CompressionLevel = level
		return w, nil
	default:
		return nil, errors.New("unsupported compression algorithm")
	}
}
