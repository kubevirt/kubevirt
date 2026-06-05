package errorhandling

import (
	"os"

	"golang.org/x/sys/unix"
)

func SafelyCloseFile(f *os.File) {
	if err := f.Close(); err != nil {
		panicIfNotReadOnlyFile(f)
	}
}

func panicIfNotReadOnlyFile(f *os.File) {
	fileFlags, err := unix.FcntlInt(f.Fd(), unix.F_GETFD, 0)
	if err != nil {
		panic("could not access file's FD flags")
	}
	if fileFlags&(unix.O_WRONLY|unix.O_APPEND|unix.O_RDWR) != 0 {
		panic("this is not a read-only fd")
	}
}
