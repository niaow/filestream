// +build !linux,!darwin

package filestream

import "os"

func getUser(info os.FileInfo) (string, error) {
	return "", nil
}

func getGroup(info os.FileInfo) (string, error) {
	return "", nil
}

func chown(path string, fo FileOptions) error { return nil }
