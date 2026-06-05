//go:build linux

package safepath

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

const pathSeparator = string(os.PathSeparator)
const pathRoot = string(os.PathSeparator)

// advance will try to add the child to the parent. If it is a relative symlink it will resolve it
// and return the parent with the new symlink. If it is an absolute symlink, parent will be reset to '/'
// and returned together with the absolute symlink. If the joined result is no symlink, the joined result will
// be returned as the new parent.
func advance(rootBase string, parent string, child string) (string, string, error) {
	// Ensure parent is absolute and never empty
	parent = filepath.Clean(parent)
	if !filepath.IsAbs(parent) {
		return "", "", fmt.Errorf("parent path %v must be absolute", parent)
	}

	if strings.Contains(child, pathSeparator) {
		return "", "", fmt.Errorf("child %q must not contain a path separator", child)
	}

	// Deal with relative path elements like '.', '//' and '..'
	// Since parent is absolute, worst case we get '/' as result
	path := filepath.Join(parent, child)

	if path == rootBase {
		// don't evaluate the root itself, since rootBase is allowed to be a symlink
		return path, "", nil
	}

	fi, err := os.Lstat(filepath.Join(rootBase, path))
	if err != nil {
		return "", "", err
	}

	if fi.Mode()&fs.ModeSymlink == 0 {
		// no symlink, we are done, return the joined result of parent and child
		return filepath.Clean(path), "", nil
	}

	link, err := os.Readlink(filepath.Join(rootBase, path))
	if err != nil {
		return "", "", err
	}

	if filepath.IsAbs(link) {
		// the link is absolute, let's reset the parent and the discovered link path
		return pathRoot, filepath.Clean(link), nil
	} else {
		// on relative links, don't advance parent and return the link
		return parent, filepath.Clean(link), nil
	}
}

// openat helps traversing a path without following symlinks
// to ensure safe path references on user-owned paths by privileged processes
func openat(dirfd int, path string) (fd int, err error) {
	if err := isSingleElement(path); err != nil {
		return -1, err
	}
	return unix.Openat(dirfd, path, unix.O_NOFOLLOW|unix.O_PATH, 0)
}

func unlinkat(dirfd int, path string, flags int) error {
	if err := isSingleElement(path); err != nil {
		return err
	}
	return unix.Unlinkat(dirfd, path, flags)
}

func touchat(dirfd int, path string, mode uint32) (fd int, err error) {
	if err := isSingleElement(path); err != nil {
		return -1, err
	}
	return unix.Openat(dirfd, path, unix.O_NOFOLLOW|syscall.O_CREAT|syscall.O_EXCL, mode)
}

func mknodat(dirfd int, path string, mode uint32, dev uint64) (err error) {
	if err := isSingleElement(path); err != nil {
		return err
	}
	return unix.Mknodat(dirfd, path, mode, int(dev))
}

func open(path string) (fd int, err error) {
	return syscall.Open(path, unix.O_PATH, 0)
}

func path(fd int) string {
	return fmt.Sprintf("/proc/self/fd/%d", fd)
}
