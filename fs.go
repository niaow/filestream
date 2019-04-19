package filestream

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// EncodeOptions are a set of options for encoding files from the filesystem into a filestream.
// They may be used with EncodeFiles.
type EncodeOptions struct {
	// Base path which should be used for stream.
	// Paths within this archive will be relative to this path.
	// By default, it will be set to the top level path provided to EncodeFiles.
	// Setting base to "/" will cause the encoding to use absolute paths.
	Base string

	// IncludePermissions is whether or not to include permission codes from the files.
	// Setting this to true will cause permissions to be preserved in the stream.
	// This does not control whether owning group/user is sent.
	IncludePermissions bool

	// IncludeUser is whether or not to include the owning username in the stream.
	// Setting this to true will cause the system to look up the username of the owning user.
	// Failed username lookups will result in errors.
	// This is supported on Linux and Darwin, and may be a no-op on other systems.
	IncludeUser bool

	// IncludeGroup is whether or not to include the owning group name in the stream.
	// Setting this to true will cause the system to look up the group name of the owning group.
	// Failed group name lookups will result in errors.
	// This is supported on Linux and Darwin, and may be a no-op on other systems.
	IncludeGroup bool
}

// EncodeFiles encodes files from a path into a stream.
func EncodeFiles(dst *Writer, path string, opts EncodeOptions) error {
	// fix paths to be appropriate and absolute
	if opts.Base == "" {
		opts.Base = path
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	opts.Base, err = filepath.Abs(opts.Base)
	if err != nil {
		return err
	}

	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		// dont try to handle inaccessible files
		if err != nil {
			return err
		}

		// convert paths to relative when appropriate
		rawpath := path
		if opts.Base != "/" {
			path, err = filepath.Rel(opts.Base, path)
			if err != nil {
				return err
			}
		}

		// load appropriate file options
		var fo FileOptions
		if opts.IncludePermissions {
			fo.Permissions = info.Mode()
		}
		if opts.IncludeUser {
			fo.User, err = getUser(info)
			if err != nil {
				return err
			}
		}
		if opts.IncludeGroup {
			fo.Group, err = getGroup(info)
			if err != nil {
				return err
			}
		}

		switch {
		case info.Mode().IsDir():
			// encode directory
			return dst.Directory(path, fo)
		case info.Mode().IsRegular():
			// open file entry stream
			fw, err := dst.File(path, fo)
			if err != nil {
				return err
			}

			// open file
			f, err := os.Open(rawpath)
			if err != nil {
				return err
			}
			defer f.Close()

			// copy file data to stream
			_, err = io.Copy(fw, f)
			if err != nil {
				return err
			}

			// close file
			err = f.Close()
			if err != nil {
				return err
			}

			// terminate file stream entry
			err = fw.Close()
			if err != nil {
				return err
			}

			return nil
		default:
			// error if we dont know what to do with a special file
			return fmt.Errorf("unsupported special file: %s", rawpath)
		}
	})
}

// DecodeOptions is a set of options for decoding files from a stream into the filesystem.
type DecodeOptions struct {
	// Base is the base directory from which relative paths will be resolved.
	Base string

	// PreservePermissions is whether or not to preserve the perimission codes from the stream.
	PreservePermissions bool

	// PreserveUser is whether or not to preserve the owning user info from the stream.
	PreserveUser bool

	// PreserveGroup is whether or not to preserve the owning group info from the stream.
	PreserveGroup bool

	// DefaultOpts are the default file options.
	// If any given option is not being preserved, the corresponding default will be applied to everything.
	// If any given option is being preserved, the corresponding default will be applied where not present in the stream.
	// Defaults to 640, current user, current group.
	DefaultOpts FileOptions
}

// DecodeStream decodes a filestream to the filesystem.
func DecodeStream(src *Reader, opts DecodeOptions) error {
	if opts.DefaultOpts.Permissions == 0 {
		opts.DefaultOpts.Permissions = 0640
	}
	if opts.Base == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		opts.Base = wd
	}
	for src.Next() {
		fr := src.File()

		path := filepath.Join(opts.Base, fr.Path())

		fo := fr.Opts()
		if !opts.PreservePermissions {
			fo.Permissions = fo.Permissions &^ os.ModePerm
		}
		if !opts.PreserveUser {
			fo.User = ""
		}
		if !opts.PreserveGroup {
			fo.Group = ""
		}
		if fo.Permissions&os.ModePerm == 0 {
			fo.Permissions |= opts.DefaultOpts.Permissions
			if (fo.Permissions & os.ModeDir) != 0 {
				fo.Permissions |= 0100
			}
		}

		switch {
		case fo.Permissions.IsDir():
			err := os.MkdirAll(path, fo.Permissions&os.ModePerm)
			if err != nil {
				return err
			}
		case fo.Permissions.IsRegular():
			f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, fo.Permissions)
			if err != nil {
				return err
			}

			_, err = io.Copy(f, fr)
			if err != nil {
				f.Close()
				return err
			}

			err = f.Close()
			if err != nil {
				return err
			}
		default:
			return errors.New("cannot decode special file")
		}

		if fo.User != "" || fo.Group != "" {
			err := chown(path, fo)
			if err != nil {
				return err
			}
		}
	}
	return src.Err()
}
