package filestream

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
)

const fmtVersion = 0

// Reader is a filestream reader.
type Reader struct {
	// ready is whether we are ready to read another file header
	ready bool

	// closed is whether the reader has completed
	closed bool

	// stream is the decompressed data stream
	stream bufio.Reader

	// closer is the io.Closer used to be closed after read completed
	closer io.Closer
}

// NewReader creates a new Reader which reads from the source.
func NewReader(src io.Reader) (*Reader, error) {
	br := bufio.NewReader(src)

	jd, err := br.ReadString('\x00')
	if err != nil {
		return nil, err
	}
	jd = jd[:len(jd)-1] // remove trailing null character

	var hdr streamHeader
	err = json.Unmarshal([]byte(jd), &hdr)
	if err != nil {
		return nil, err
	}

	if hdr.Version > fmtVersion {
		return nil, fmt.Errorf("filestream v%d format not supported (max supported: v%d)", hdr.Version, fmtVersion)
	}

	var closer io.Closer
	var stream io.Reader = br
	if hdr.Compression != "" {
		zr, err := decompress(hdr.Compression, br)
		if err != nil {
			return nil, err
		}
		stream = zr
		closer = zr
	}

	r := &Reader{
		stream: *bufio.NewReader(stream),
		ready:  true,
		closer: closer,
	}

	return r, nil
}

// Next gets the reader of the next file.
// Returns io.EOF at the end of the stream.
// File must be read completely before calling Next again.
// Directories do not need to be read, and have no body.
func (r *Reader) Next() (fr *FileReader, err error) {
	if r.closed {
		return nil, io.EOF
	}

	if !r.ready {
		return nil, errors.New("requested next file before finishing previous")
	}

	r.ready = false

	jd, err := r.stream.ReadString('\x00')
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	jd = jd[:len(jd)-1] // remove trailing null character

	var hdr fileHeader
	err = json.Unmarshal([]byte(jd), &hdr)
	if err != nil {
		return nil, err
	}

	if hdr.Path == "\x00" {
		r.closed = true
		if r.closer != nil {
			err = r.closer.Close()
			if err != nil {
				if err == io.EOF {
					err = io.ErrUnexpectedEOF
				}
				return nil, err
			}
		}
		return nil, io.EOF
	}

	fr = &FileReader{
		reader: r,
		hdr:    hdr,
	}

	if fr.IsDir() {
		// dir should be zero length - read terminator
		_, err = fr.Read(nil)
		if err == nil {
			return nil, fmt.Errorf("expected empty body for directory %q but got body", hdr.Path)
		}
		if err != io.EOF {
			return nil, err
		}
	}

	return fr, nil
}

// FileReader is a reader of a single file in a stream.
type FileReader struct {
	// reader is the parent
	reader *Reader

	// hdr is the file header
	hdr fileHeader

	// done is whether the end of the file has been reached
	done bool

	// chunkRem is the remaining size of the current chunk
	chunkRem int
}

// Path is the path of the file.
func (fr *FileReader) Path() string {
	return fr.hdr.Path
}

// IsDir returns whether the entry is a directory.
func (fr *FileReader) IsDir() bool {
	return fr.hdr.Mode.IsDir()
}

// Opts are the options of the file.
func (fr *FileReader) Opts() FileOptions {
	return FileOptions{
		Permissions: fr.hdr.Mode,
		User:        fr.hdr.User,
		Group:       fr.hdr.Group,
	}
}

func (fr *FileReader) Read(dst []byte) (n int, err error) {
	if fr.done {
		return 0, io.EOF
	}

	if fr.chunkRem == 0 {
		lstr, err := fr.reader.stream.ReadString('\x00')
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return 0, err
		}
		lstr = lstr[:len(lstr)-1]

		l, err := strconv.Atoi(lstr)
		if err != nil {
			return 0, err
		}

		if l == 0 {
			fr.done = true
			fr.reader.ready = true
			return 0, io.EOF
		}

		fr.chunkRem = l
	}

	n = fr.chunkRem
	if n > len(dst) {
		n = len(dst)
	}

	n, err = fr.reader.stream.Read(dst[:n])

	fr.chunkRem -= n

	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}

	return n, err
}
