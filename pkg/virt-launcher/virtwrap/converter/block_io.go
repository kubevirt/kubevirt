//go:build !linux

package converter

import (
	"os"
)

func getBlockIOSizes(_ string, _ *os.File) (int, int, error) {
	return 512, 4096, nil
}

const (
	SYSCALL_O_DIRECT = 0
)
