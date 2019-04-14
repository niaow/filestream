package filestream

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// StreamOptions are configuration options for a stream.
type StreamOptions struct {
	// Compression is the compression algorithm to use in transit.
	// This package supports "gzip" and "lz4".
	// Defaults to no compresion.
	Compression string

	// CompressionLevel is the level of compresion to use.
	// Uses a sane default if omitted.
	CompressionLevel int
}

// FileOptions are the set of options which can be applied to a file stream.
type FileOptions struct {
	// Permissions are the unix permission code of the file.
	// If the permissions component is 000, this will be converted to sane defaults.
	// Optional.
	Permissions os.FileMode

	// User is the username of the owner.
	// Optional.
	User string

	// Group is the groupname of the owning group.
	// Optional.
	Group string
}

// Writer is an encoder for a filestream.
type Writer struct {
	curFile uint64
	writing bool
	w       bufio.Writer
	closer  io.Closer
	closed  bool
}

// NewWriter creates a new file stream writer.
func NewWriter(dst io.Writer, opts StreamOptions) (*Writer, error) {
	// obtain compressor
	var z io.WriteCloser
	if opts.Compression != "" {
		zr, err := compress(opts.Compression, opts.CompressionLevel, dst)
		if err != nil {
			return nil, err
		}
		z = zr
	}

	// set up writer
	w := new(Writer)
	w.w = *bufio.NewWriter(dst)
	if opts.Compression != "" {
		w.closer = z
	}

	// write header
	err := json.NewEncoder(&w.w).Encode(streamHeader{
		Version:     0,
		Compression: opts.Compression,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to write stream header: %s", err)
	}
	err = w.w.WriteByte('\x00')
	if err != nil {
		return nil, fmt.Errorf("failed to write stream header: %s", err)
	}

	// set destination to compressor
	if opts.Compression != "" {
		err = w.w.Flush()
		if err != nil {
			return nil, fmt.Errorf("failed to write stream header: %s", err)
		}
		w.w.Reset(z)
	}

	return w, nil
}

// File creates a new file stream at the given path.
// Writing multiple files simultaneously is not supported.
func (w *Writer) File(path string, opts FileOptions) io.WriteCloser {
	return &fileWriter{
		stream: w,
		hdr: fileHeader{
			Path:  path,
			Mode:  opts.Permissions,
			User:  opts.User,
			Group: opts.Group,
		},
	}
}

// Directory creates a directory in the stream with the given path.
func (w *Writer) Directory(path string, opts FileOptions) error {
	opts.Permissions |= os.ModeDir
	return w.File(path, opts).Close()
}

// ErrWriteInterrupted indicates that a close operation interrupted a file stream and may have resulted in a corrupted stream.
var ErrWriteInterrupted = errors.New("write interrupted")

// Close ends the stream.
// If a file stream is incomplete, generates a corrupted stream and returns ErrWriteInterrupted.
func (w *Writer) Close() error {
	// mark as closed
	w.closed = true

	// do not terminate incomplete writes
	if w.writing {
		return ErrWriteInterrupted
	}

	// write terminating header
	err := json.NewEncoder(&w.w).Encode(fileHeader{
		Path: "\x00",
	})
	if err != nil {
		return fmt.Errorf("failed to terminate stream: %s", err)
	}
	err = w.w.WriteByte('\x00')
	if err != nil {
		return fmt.Errorf("failed to terminate stream: %s", err)
	}

	// flush stream to compressor
	err = w.w.Flush()
	if err != nil {
		return fmt.Errorf("failed to terminate stream: %s", err)
	}

	// flush compressor
	if w.closer != nil {
		err = w.closer.Close()
		if err != nil {
			return fmt.Errorf("failed to terminate stream: %s", err)
		}
	}

	return nil
}

func (w *Writer) write(file uint64, dat []byte) (int, error) {
	// check that stream is open
	if w.closed {
		return 0, errors.New("filestream closed")
	}

	// check that file is correct
	if file != w.curFile || !w.writing {
		return 0, errors.New("writing to file that has already been closed")
	}

	// write length of chunk
	_, err := w.w.WriteString(strconv.Itoa(len(dat)))
	if err != nil {
		return 0, err
	}
	err = w.w.WriteByte('\x00')
	if err != nil {
		return 0, err
	}

	// write data
	n, err := w.w.Write(dat)
	if err != nil {
		return n, err
	}

	return len(dat), nil
}

func (w *Writer) startFile(hdr fileHeader) (uint64, error) {
	if w.closed {
		return 0, errors.New("filestream closed")
	}

	if w.writing {
		return 0, errors.New("attempted to write another file while one is already streaming")
	}

	if strings.Contains(hdr.Path, "\x00") {
		return 0, errors.New("illegal null character in file path")
	}

	w.curFile++
	w.writing = true

	err := json.NewEncoder(&w.w).Encode(hdr)
	if err != nil {
		return 0, fmt.Errorf("failed to start file stream: %s", err)
	}
	err = w.w.WriteByte('\x00')
	if err != nil {
		return 0, fmt.Errorf("failed to start file stream: %s", err)
	}

	return w.curFile, nil
}

// fileWriter is a stream for writing a file.
type fileWriter struct {
	stream *Writer
	fileNo uint64
	hdr    fileHeader
}

// Write writes the data to the file stream.
func (fw *fileWriter) Write(data []byte) (int, error) {
	if fw.fileNo == 0 {
		no, err := fw.stream.startFile(fw.hdr)
		if err != nil {
			return 0, err
		}
		fw.fileNo = no
	}

	if len(data) == 0 {
		return 0, nil
	}

	return fw.stream.write(fw.fileNo, data)
}

// Close closes a file stream.
func (fw *fileWriter) Close() error {
	// for 0 length files, start the stream
	if fw.fileNo == 0 {
		_, err := fw.Write(nil)
		if err != nil {
			return err
		}
	}

	// write terminating 0 length chunk
	_, err := fw.stream.write(fw.fileNo, nil)
	if err != nil {
		return err
	}

	// mark as no longer writing
	fw.stream.writing = false

	return nil
}
