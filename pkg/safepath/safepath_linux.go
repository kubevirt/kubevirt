//go:build darwin

package safepath

import (
	"golang.org/x/sys/unix"
)

// openat helps traversing a path without following symlinks
// to ensure safe path references on user-owned paths by privileged processes
// OS-specific: unix.O_PATH
func openat(dirfd int, path string) (fd int, err error) {
	if err := isSingleElement(path); err != nil {
		return -1, err
	}
	return unix.Openat(dirfd, path, unix.O_NOFOLLOW|unix.O_PATH, 0)
}

// OS-specific: unix.Mknodat
func mknodat(dirfd int, path string, mode uint32, dev uint64) (err error) {
	if err := isSingleElement(path); err != nil {
		return err
	}
	return unix.Mknodat(dirfd, path, mode, int(dev))
}

// OS-specific: unix.O_PATH
func open(path string) (fd int, err error) {
	return unix.Open(path, unix.O_PATH, 0)
}

// path returns the file path associated with the given file descriptor by formatting the path as /proc/self/fd/<fd>.
func symlinkPath(fd int) string {
	return fmt.Sprintf("/proc/self/fd/%d", fd)
}
