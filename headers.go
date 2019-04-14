package filestream

import "os"

// streamHeader is the header that goes at the beginning of the stream
type streamHeader struct {
	// Version is the version of the filestream format used.
	Version int `json:"version"`

	// Compression is the compression algorithm to use.
	Compression string `json:"compression,omitempty"`
}

// fileHeader is a header which comes before a file
type fileHeader struct {
	// Path is the path relative to the base of the stream.
	// The path "\x00" terminates the file stream.
	Path string `json:"path"`

	// User is the username of the owner.
	User string `json:"user,omitempty"`

	// Group is the owning group.
	Group string `json:"group,omitempty"`

	// Mode is the file permission mode code.
	Mode os.FileMode `json:"mode,omitempty"`
}
