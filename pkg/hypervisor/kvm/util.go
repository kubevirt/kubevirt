package kvm

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

type maskType bool

type cpuMask struct {
	mask map[string]maskType
}

const (
	enabled  maskType = true
	disabled maskType = false
)

var (
	// parse CPU Mask expressions
	cpuRangeRegex  = regexp.MustCompile(`^(\d+)-(\d+)$`)
	negateCPURegex = regexp.MustCompile(`^\^(\d+)$`)
	singleCPURegex = regexp.MustCompile(`^(\d+)$`)
)

// setProcessMemoryLockRLimit Adjusts process MEMLOCK
// soft-limit (current) and hard-limit (max) to the given size.
func setProcessMemoryLockRLimit(pid int, size int64) error {
	// standard golang libraries don't provide API to set runtime limits
	// for other processes, so we have to directly call to kernel
	rlimit := unix.Rlimit{
		Cur: uint64(size),
		Max: uint64(size),
	}
	_, _, errno := unix.RawSyscall6(unix.SYS_PRLIMIT64,
		uintptr(pid),
		uintptr(unix.RLIMIT_MEMLOCK),
		uintptr(unsafe.Pointer(&rlimit)), // #nosec used in unix RawSyscall6
		0, 0, 0)
	if errno != 0 {
		return fmt.Errorf("error setting prlimit: %v", errno)
	}

	return nil
}

// Returns the pid of "vmpid" as seen from the first pid namespace the task
// belongs to.
func GetNspid(vmpid int) (int, error) {
	fpath := filepath.Join("proc", strconv.Itoa(vmpid), "status")
	file, err := os.Open(fpath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 6 {
			continue
		}
		if line[0:6] != "NSpid:" {
			continue
		}
		s := strings.Fields(line)
		if len(s) < 2 {
			continue
		}
		val, err := strconv.Atoi(s[2])
		return val, err
	}

	return -1, nil
}

// parseCPUMask parses the mask and maps the results into a structure that contains which
// CPUs are enabled or disabled for the scheduling and priority changes.
// This implementation reimplements the libvirt parsing logic defined here:
// https://github.com/libvirt/libvirt/blob/56de80cb793aa7aedc45572f8b6ec3fc32c99309/src/util/virbitmap.c#L382
// except that in this case it uses a map[string]maskType instead of a bit array.
func parseCPUMask(mask string) (*cpuMask, error) {

	vcpus := cpuMask{}
	if len(mask) == 0 {
		return &vcpus, nil
	}
	vcpus.mask = make(map[string]maskType)

	masks := strings.Split(mask, ",")
	for _, i := range masks {
		m := strings.TrimSpace(i)
		switch {
		case cpuRangeRegex.MatchString(m):
			match := cpuRangeRegex.FindSubmatch([]byte(m))
			startID, err := strconv.Atoi(string(match[1]))
			if err != nil {
				return nil, err
			}
			endID, err := strconv.Atoi(string(match[2]))
			if err != nil {
				return nil, err
			}
			if startID < 0 {
				return nil, fmt.Errorf("invalid vcpu mask start index `%d`", startID)
			}
			if endID < 0 {
				return nil, fmt.Errorf("invalid vcpu mask end index `%d`", endID)
			}
			if startID > endID {
				return nil, fmt.Errorf("invalid mask range `%d-%d`", startID, endID)
			}
			for id := startID; id <= endID; id++ {
				vid := strconv.Itoa(id)
				if !vcpus.has(vid) {
					vcpus.set(vid, enabled)
				}
			}
		case singleCPURegex.MatchString(m):
			match := singleCPURegex.FindSubmatch([]byte(m))
			vid, err := strconv.Atoi(string(match[1]))
			if err != nil {
				return nil, err
			}
			if vid < 0 {
				return nil, fmt.Errorf("invalid vcpu index `%d`", vid)
			}
			if !vcpus.has(string(match[1])) {
				vcpus.set(string(match[1]), enabled)
			}
		case negateCPURegex.MatchString(m):
			match := negateCPURegex.FindSubmatch([]byte(m))
			vid, err := strconv.Atoi(string(match[1]))
			if err != nil {
				return nil, err
			}
			if vid < 0 {
				return nil, fmt.Errorf("invalid vcpu index `%d`", vid)
			}
			vcpus.set(string(match[1]), disabled)
		default:
			return nil, fmt.Errorf("invalid mask value '%s' in '%s'", i, mask)
		}
	}
	return &vcpus, nil
}

func (c cpuMask) isEnabled(vcpuID string) bool {
	if len(c.mask) == 0 {
		return true
	}
	if t, ok := c.mask[vcpuID]; ok {
		return t == enabled
	}
	return false
}

func (c *cpuMask) has(vcpuID string) bool {
	_, ok := c.mask[vcpuID]
	return ok
}

func (c *cpuMask) set(vcpuID string, mtype maskType) {
	c.mask[vcpuID] = mtype
}
