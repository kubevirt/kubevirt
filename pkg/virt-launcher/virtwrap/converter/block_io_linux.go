//go:build linux

package converter

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func getBlockIOSizes(path string, f *os.File) (int, int, error) {
	logicalSize, err := unix.IoctlGetUint32(int(f.Fd()), unix.BLKSSZGET)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to get logical block size from device %v: %v", path, err)
	}
	physicalSize, err := unix.IoctlGetUint32(int(f.Fd()), unix.BLKBSZGET)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to get physical block size from device %v: %v", path, err)
	}

	return int(logicalSize), int(physicalSize), nil
}

const (
	SYSCALL_O_DIRECT = syscall.O_DIRECT
)
