//go:build (linux && amd64) || (linux && arm64)

package virthandler

import (
	"unsafe"

	"golang.org/x/sys/unix"

	// #include <linux/sched.h>
	// #include <linux/sched/types.h>
	// typedef struct sched_param sched_param;
	"C"
)

type schedParam C.sched_param
type policy uint32

const (
	schedFIFO policy = C.SCHED_FIFO
)

func schedSetScheduler(pid int, policy policy, param schedParam) error {
	_, _, e1 := unix.Syscall(unix.SYS_SCHED_SETSCHEDULER, uintptr(pid), uintptr(policy), uintptr(unsafe.Pointer(&param)))
	if e1 != 0 {
		return e1
	}
	return nil
}
