/*
 * This file is part of the kubevirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package isolation

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"kubevirt.io/kubevirt/pkg/unsafepath"

	"golang.org/x/sys/unix"

	"kubevirt.io/kubevirt/pkg/safepath"

	"github.com/containernetworking/plugins/pkg/ns"

	ps "github.com/mitchellh/go-ps"
	"k8s.io/apimachinery/pkg/api/resource"

	mount "github.com/moby/sys/mountinfo"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

// PodIsolationDetector helps detecting cgroups, namespaces and PIDs of Pods from outside of them.
// Different strategies may be applied to do that.
type PodIsolationDetector interface {
	// Detect takes a vm, looks up a socket based the VM and detects pid, cgroups and namespaces of the owner of that socket.
	// It returns an IsolationResult containing all isolation information
	Detect(vm *v1.VirtualMachineInstance) (IsolationResult, error)

	DetectForSocket(vm *v1.VirtualMachineInstance, socket string) (IsolationResult, error)

	// Whitelist allows specifying cgroup controller which should be considered to detect the cgroup slice
	// It returns a PodIsolationDetector to allow configuring the PodIsolationDetector via the builder pattern.
	Whitelist(controller []string) PodIsolationDetector

	// Adjust system resources to run the passed VM
	AdjustResources(vm *v1.VirtualMachineInstance) error
}

const isolationDialTimeout = 5

type MountInfo struct {
	DeviceContainingFile string
	Root                 string
	MountPoint           string
}

// The unit test suite overwrites this function
var mountInfoFunc = func(pid int) string {
	return fmt.Sprintf("/proc/%d/mountinfo", pid)
}

type socketBasedIsolationDetector struct {
	socketDir    string
	controller   []string
	cgroupParser cgroup.Parser
}

func (s *socketBasedIsolationDetector) DetectForSocket(vm *v1.VirtualMachineInstance, socket string) (IsolationResult, error) {
	var pid int
	var slice string
	var err error
	var controller []string

	if pid, err = s.getPid(socket); err != nil {
		log.Log.Object(vm).Reason(err).Errorf("Could not get owner Pid of socket %s", socket)
		return nil, err

	}

	// Look up the cgroup slice based on the whitelisted controller
	if controller, slice, err = s.getSlice(pid); err != nil {
		log.Log.Object(vm).Reason(err).Errorf("Could not get cgroup slice for Pid %d", pid)
		return nil, err
	}

	return NewIsolationResult(pid, slice, controller), nil
}

// NewSocketBasedIsolationDetector takes socketDir and creates a socket based IsolationDetector
// It returns a PodIsolationDetector which detects pid, cgroups and namespaces of the socket owner.
func NewSocketBasedIsolationDetector(socketDir string, cgroupParser cgroup.Parser) PodIsolationDetector {
	return &socketBasedIsolationDetector{
		socketDir:    socketDir,
		controller:   []string{"devices"},
		cgroupParser: cgroupParser,
	}
}

func (s *socketBasedIsolationDetector) Whitelist(controller []string) PodIsolationDetector {
	s.controller = controller
	return s
}

func (s *socketBasedIsolationDetector) Detect(vm *v1.VirtualMachineInstance) (IsolationResult, error) {
	// Look up the socket of the virt-launcher Pod which was created for that VM, and extract the PID from it
	socket, err := cmdclient.FindSocketOnHost(vm)
	if err != nil {
		return nil, err
	}

	return s.DetectForSocket(vm, socket)
}

// standard golang libraries don't provide API to set runtime limits
// for other processes, so we have to directly call to kernel
func prLimit(pid int, limit uintptr, rlimit *unix.Rlimit) error {
	_, _, errno := unix.RawSyscall6(unix.SYS_PRLIMIT64,
		uintptr(pid),
		limit,
		uintptr(unsafe.Pointer(rlimit)), // #nosec used in unix RawSyscall6
		0, 0, 0)
	if errno != 0 {
		return fmt.Errorf("Error setting prlimit: %v", errno)
	}
	return nil
}

func (s *socketBasedIsolationDetector) AdjustResources(vm *v1.VirtualMachineInstance) error {
	// only VFIO attached domains require MEMLOCK adjustment
	if !util.IsVFIOVMI(vm) {
		return nil
	}

	// bump memlock ulimit for libvirtd
	res, err := s.Detect(vm)
	if err != nil {
		return err
	}
	launcherPid := res.Pid()

	processes, err := ps.Processes()
	if err != nil {
		return fmt.Errorf("failed to get all processes: %v", err)
	}

	for _, process := range processes {
		// consider all processes that are virt-launcher children
		if process.PPid() != launcherPid {
			continue
		}

		// libvirtd process sets the memory lock limit before fork/exec-ing into qemu
		if process.Executable() != "libvirtd" {
			continue
		}

		// make the best estimate for memory required by libvirt
		memlockSize, err := getMemlockSize(vm)
		if err != nil {
			return err
		}
		rLimit := unix.Rlimit{
			Max: uint64(memlockSize),
			Cur: uint64(memlockSize),
		}
		err = prLimit(process.Pid(), unix.RLIMIT_MEMLOCK, &rLimit)
		if err != nil {
			return fmt.Errorf("failed to set rlimit for memory lock: %v", err)
		}
		// we assume a single process should match
		break
	}
	return nil
}

// consider reusing getMemoryOverhead()
// This is not scientific, but neither what libvirtd does is. See details in:
// https://www.redhat.com/archives/libvirt-users/2019-August/msg00051.html
func getMemlockSize(vm *v1.VirtualMachineInstance) (int64, error) {
	memlockSize := resource.NewQuantity(0, resource.DecimalSI)

	// start with base memory requested for the VM
	vmiMemoryReq := vm.Spec.Domain.Resources.Requests.Memory()
	memlockSize.Add(*resource.NewScaledQuantity(vmiMemoryReq.ScaledValue(resource.Kilo), resource.Kilo))

	// allocate 1Gb for VFIO needs
	memlockSize.Add(resource.MustParse("1G"))

	// add some more memory for NUMA / CPU topology, platform memory alignment and other needs
	memlockSize.Add(resource.MustParse("256M"))

	bytes_, ok := memlockSize.AsInt64()
	if !ok {
		return 0, fmt.Errorf("could not calculate memory lock size")
	}
	return bytes_, nil
}

func NewIsolationResult(pid int, slice string, controller []string) IsolationResult {
	return &realIsolationResult{pid: pid, slice: slice, controller: controller}
}

type IsolationResult interface {
	// cgroup slice
	Slice() string
	// process ID
	Pid() int
	// full path to the process namespace
	PIDNamespace() string
	// full path to the process root mount
	MountRoot() (*safepath.Path, error)
	// retrieve additional information about the process root mount
	MountInfoRoot() (*MountInfo, error)
	// full path to the mount namespace
	MountNamespace() string
	// full path to the network namespace
	NetNamespace() string
	// execute a function in the process network namespace
	DoNetNS(func() error) error
}

type realIsolationResult struct {
	pid        int
	slice      string
	controller []string
}

func (r *realIsolationResult) DoNetNS(f func() error) error {
	netns, err := ns.GetNS(r.NetNamespace())
	if err != nil {
		return fmt.Errorf("failed to get launcher pod network namespace: %v", err)
	}
	return netns.Do(func(_ ns.NetNS) error {
		return f()
	})
}

func (r *realIsolationResult) PIDNamespace() string {
	return fmt.Sprintf("/proc/%d/ns/pid", r.pid)
}

func (r *realIsolationResult) Slice() string {
	return r.slice
}

func (r *realIsolationResult) MountNamespace() string {
	return fmt.Sprintf("/proc/%d/ns/mnt", r.pid)
}

func (r *realIsolationResult) mountInfo() string {
	return mountInfoFunc(r.pid)
}

func forEachRecord(filepath string, f func(record []string) bool) error {
	in, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("could not open file %s: %v", filepath, err)
	}
	defer util.CloseIOAndCheckErr(in, nil)
	c := csv.NewReader(in)
	c.Comma = ' '
	c.LazyQuotes = true
	for {
		record, err := c.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			if e, ok := err.(*csv.ParseError); ok {
				if e.Err != csv.ErrFieldCount {
					return err
				}
			} else {
				return err
			}
		}

		if f(record) {
			break
		}
	}
	return nil
}

// MountInfoRoot returns information about the root entry in /proc/mountinfo
func (r *realIsolationResult) MountInfoRoot() (mountInfo *MountInfo, err error) {
	if err = forEachRecord(r.mountInfo(), func(record []string) bool {
		if record[4] == "/" {
			mountInfo = &MountInfo{
				DeviceContainingFile: record[2],
				Root:                 record[3],
				MountPoint:           record[4],
			}
		}
		return mountInfo != nil
	}); err != nil {
		return nil, err
	}
	if mountInfo == nil {
		//impossible
		err = fmt.Errorf("process has no root entry")
	}
	return
}

// IsMounted checks if the given path is a mount point or not.
func (r *realIsolationResult) IsMounted(mountPoint *safepath.Path) (isMounted bool, err error) {
	// Ensure that the path is still a valid absolute path without symlinks
	f, err := safepath.OpenAtNoFollow(mountPoint)
	if err != nil {
		// treat os.IsNotExist() as error too
		// since the inherent property of a safepath.Path is that the path must
		// have existed at the point of object creation
		return false, err
	}
	defer f.Close()
	if mountPoint.IsRoot() {
		// mount.Mounted has purely string matching based special logic on how to treat "/".
		// Emulating this for safepath here without ever having to call an unsafe method on our
		// safepath.
		return true, nil
	} else {
		// TODO: Unsafe full path is required, and not a fd, since otherwise mount table lookups and such would not work.
		return mount.Mounted(unsafepath.UnsafeAbsolute(mountPoint.Raw()))
	}
}

// IsBlockDevice check if the path given is a block device or not.
func (r *realIsolationResult) IsBlockDevice(path *safepath.Path) (bool, error) {
	fileInfo, err := safepath.StatAtNoFollow(path)
	if err == nil {
		if !fileInfo.IsDir() && (fileInfo.Mode()&os.ModeDevice) != 0 {
			return true, nil
		}
		return false, fmt.Errorf("found %v, but it's not a block device", path)
	}
	return false, fmt.Errorf("error checking for block device: %v", err)
}

// ParentMountInfoFor takes the mount info from a container, and looks the corresponding
// entry in /proc/mountinfo of the isolation result of the given process.
func (r *realIsolationResult) ParentMountInfoFor(mountInfo *MountInfo) (parentMountInfo *MountInfo, err error) {
	if err = forEachRecord(r.mountInfo(), func(record []string) bool {
		if record[2] == mountInfo.DeviceContainingFile {
			parentMountInfo = &MountInfo{
				DeviceContainingFile: record[2],
				Root:                 record[3],
				MountPoint:           record[4],
			}
		}
		return parentMountInfo != nil
	}); err != nil {
		return nil, err
	}
	if parentMountInfo == nil {
		err = fmt.Errorf("no parent entry for %v found in the mount namespace of %d", mountInfo.DeviceContainingFile, r.pid)
	}
	return
}

// FullPath takes the mount info from a container and composes the full path starting from
// the root mount of the given process.
func (r *realIsolationResult) FullPath(mountInfo *MountInfo) (path *safepath.Path, err error) {
	// Handle btrfs subvolumes: mountInfo.Root seems to already provide the needed path
	if strings.HasPrefix(mountInfo.Root, "/@") {
		mountRoot, err := r.MountRoot()
		if err != nil {
			return nil, err
		}
		path, err = mountRoot.AppendAndResolveWithRelativeRoot(strings.TrimPrefix(mountInfo.Root, "/@"))
		if err != nil {
			return nil, err
		}
		return path, nil
	}
	parentMountInfo, err := r.ParentMountInfoFor(mountInfo)
	if err != nil {
		return
	}
	mountRoot, err := r.MountRoot()
	if err != nil {
		return nil, err
	}
	path, err = mountRoot.AppendAndResolveWithRelativeRoot(parentMountInfo.Root, parentMountInfo.MountPoint, mountInfo.Root)
	if err != nil {
		return nil, err
	}
	return path, nil
}

func (r *realIsolationResult) NetNamespace() string {
	return fmt.Sprintf("/proc/%d/ns/net", r.pid)
}

func (r *realIsolationResult) MountRoot() (*safepath.Path, error) {
	return safepath.JoinAndResolveWithRelativeRoot(fmt.Sprintf("/proc/%d/root", r.pid))
}

func (r *realIsolationResult) MountRootRelative(relativePath string) (*safepath.Path, error) {
	mountRoot, err := r.MountRoot()
	if err != nil {
		return nil, err
	}
	return mountRoot.AppendAndResolveWithRelativeRoot(relativePath)
}

func (r *realIsolationResult) Pid() int {
	return r.pid
}

func (r *realIsolationResult) Controller() []string {
	return r.controller
}

func (s *socketBasedIsolationDetector) getPid(socket string) (int, error) {
	sock, err := net.DialTimeout("unix", socket, time.Duration(isolationDialTimeout)*time.Second)
	if err != nil {
		return -1, err
	}
	defer sock.Close()

	ufile, err := sock.(*net.UnixConn).File()
	if err != nil {
		return -1, err
	}
	// This is the tricky part, which will give us the PID of the owning socket
	ucreds, err := syscall.GetsockoptUcred(int(ufile.Fd()), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	if err != nil {
		return -1, err
	}

	if int(ucreds.Pid) == 0 {
		return -1, fmt.Errorf("The detected PID is 0. Is the isolation detector running in the host PID namespace?")
	}

	return int(ucreds.Pid), nil
}

func (s *socketBasedIsolationDetector) getSlice(pid int) (controllers []string, slice string, err error) {
	slices, err := s.cgroupParser.Parse(pid)
	if err != nil {
		return
	}

	// Skip not supported cgroup controller
	for _, c := range s.controller {
		if s, ok := slices[c]; ok {
			// Set and check cgroup slice
			if slice == "" {
				slice = s
			} else if slice != s {
				err = fmt.Errorf("Process is part of more than one slice. Expected %s, found %s", slice, s)
				return
			}
			// Add controller
			controllers = append(controllers, c)
		}
	}

	if slice == "" {
		err = fmt.Errorf("Could not detect slice of whitelisted controllers: %v", s.controller)
	}

	return
}

func NodeIsolationResult() *realIsolationResult {
	return &realIsolationResult{
		pid: 1,
	}
}
