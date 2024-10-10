//go:build darwin

package safepath

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
)

// openat helps traversing a path without following symlinks
// to ensure safe path references on user-owned paths by privileged processes
func openat(dirfd int, path string) (fd int, err error) {
	if err := isSingleElement(path); err != nil {
		return -1, err
	}
	return unix.Openat(dirfd, path, unix.O_SYMLINK|unix.O_NOFOLLOW, 0)
}

// mknodat creates a filesystem node (file, device special file, or named pipe) at the given directory file descriptor.
// "/dev/fd" is the equivalent for "/proc/self/fd" on Linux (directory containing symbolic links to all open file descriptors of the current process)
func mknodat(dirfd int, path string, mode uint32, dev uint64) (err error) {
	if err := isSingleElement(path); err != nil {
		return err
	}

	// Change directory to dirfd to ensure proper path creation
	cwd, err := os.Open(symlinkPath(dirfd))
	if err != nil {
		return err
	}
	defer cwd.Close()

	// Create the node
	return unix.Mknod(filepath.Join(symlinkPath(dirfd), path), mode, int(dev))
}

// open safely opens a file descriptor to a given path without following symlinks.
func open(path string) (fd int, err error) {
	return unix.Open(path, unix.O_SYMLINK|unix.O_NOFOLLOW, 0)
}

// path returns the file path associated with the given file descriptor by formatting the path as /proc/self/fd/<fd>.
func symlinkPath(fd int) string {
	return fmt.Sprintf("/dev/fd/%d", fd)
}
