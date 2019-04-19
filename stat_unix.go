// +build linux darwin

package filestream

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
)

// getUser gets the username of the owner of the given file.
func getUser(info os.FileInfo) (string, error) {
	u, err := user.LookupId(strconv.Itoa(int(info.Sys().(*syscall.Stat_t).Uid)))
	if err != nil {
		return "", err
	}
	return u.Username, nil
}

// getGroup gets the group name of the owning group of the given file.
func getGroup(info os.FileInfo) (string, error) {
	g, err := user.LookupGroupId(strconv.Itoa(int(info.Sys().(*syscall.Stat_t).Uid)))
	if err != nil {
		return "", err
	}
	return g.Name, nil
}

var curUID, curGID = os.Getuid(), os.Getgid()

func chown(path string, fo FileOptions) error {
	var uid, gid int = -1, -1
	if fo.User != "" {
		u, err := user.Lookup(fo.User)
		if err != nil {
			return err
		}
		id, err := strconv.Atoi(u.Uid)
		if err != nil {
			return err
		}
		uid = id
	}
	if fo.Group != "" {
		g, err := user.LookupGroup(fo.User)
		if err != nil {
			return err
		}
		id, err := strconv.Atoi(g.Gid)
		if err != nil {
			return err
		}
		gid = id
	}
	return syscall.Chown(path, uid, gid)
}
