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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"kubevirt.io/kubevirt/pkg/unsafepath"

	ps "github.com/mitchellh/go-ps"
	mount "github.com/moby/sys/mountinfo"

	"kubevirt.io/kubevirt/pkg/safepath"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
)

// IsolationResult is the result of a successful PodIsolationDetector.Detect
type IsolationResult interface {
	// process ID
	Pid() int
	// parent process ID
	PPid() int
	// full path to the process namespace
	PIDNamespace() string
	// full path to the process root mount
	MountRoot() (*safepath.Path, error)
	// full path to the mount namespace
	MountNamespace() string
	// mounts for the process
	Mounts(mount.FilterFunc) ([]*mount.Info, error)
	// returns the QEMU process
	GetQEMUProcess() (ps.Process, error)
	// returns the KVM PIT pid
	KvmPitPid() (int, error)
}

type RealIsolationResult struct {
	pid  int
	ppid int
}

func NewIsolationResult(pid, ppid int) IsolationResult {
	return &RealIsolationResult{pid: pid, ppid: ppid}
}

func (r *RealIsolationResult) PIDNamespace() string {
	return fmt.Sprintf("/proc/%d/ns/pid", r.pid)
}

func (r *RealIsolationResult) MountNamespace() string {
	return fmt.Sprintf("/proc/%d/ns/mnt", r.pid)
}

// IsMounted checks if the given path is a mount point or not.
func IsMounted(mountPoint *safepath.Path) (isMounted bool, err error) {
	// Ensure that the path is still a valid absolute path without symlinks
	f, err := safepath.OpenAtNoFollow(mountPoint)
	if err != nil {
		// treat ErrNotExist as error too
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

// AreMounted checks if given paths are mounted by calling IsMounted.
// If error occurs, the first error is returned.
func (r *RealIsolationResult) AreMounted(mountPoints ...*safepath.Path) (isMounted bool, err error) {
	for _, mountPoint := range mountPoints {
		if mountPoint != nil {
			isMounted, err = IsMounted(mountPoint)
			if !isMounted || err != nil {
				return
			}
		}
	}

	return true, nil
}

// IsBlockDevice checks if the given path is a block device or not.
func IsBlockDevice(path *safepath.Path) (bool, error) {
	fileInfo, err := safepath.StatAtNoFollow(path)
	if err != nil {
		return false, fmt.Errorf("error checking for block device: %v", err)
	}
	if fileInfo.IsDir() || (fileInfo.Mode()&os.ModeDevice) == 0 {
		return false, nil
	}
	return true, nil
}

func (r *RealIsolationResult) MountRoot() (*safepath.Path, error) {
	return safepath.JoinAndResolveWithRelativeRoot(fmt.Sprintf("/proc/%d/root", r.pid))
}

func (r *RealIsolationResult) MountRootRelative(relativePath string) (*safepath.Path, error) {
	mountRoot, err := r.MountRoot()
	if err != nil {
		return nil, err
	}
	return mountRoot.AppendAndResolveWithRelativeRoot(relativePath)
}

func (r *RealIsolationResult) Pid() int {
	return r.pid
}

func (r *RealIsolationResult) PPid() int {
	return r.ppid
}

// GetQEMUProcess encapsulates and exposes the logic to retrieve the QEMU process ID
func (r *RealIsolationResult) GetQEMUProcess() (ps.Process, error) {
	processes, err := ps.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to get all processes: %v", err)
	}
	qemuProcess, err := findIsolatedQemuProcess(processes, r.PPid())
	if err != nil {
		return nil, err
	}
	return qemuProcess, nil
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

func (r *RealIsolationResult) KvmPitPid() (int, error) {
	qemuprocess, err := r.GetQEMUProcess()
	if err != nil {
		return -1, err
	}
	processes, _ := ps.Processes()
	nspid, err := GetNspid(qemuprocess.Pid())
	if err != nil || nspid == -1 {
		return -1, err
	}
	pitstr := "kvm-pit/" + strconv.Itoa(nspid)

	for _, process := range processes {
		if process.Executable() == pitstr {
			return process.Pid(), nil
		}
	}
	return -1, nil
}

func NodeIsolationResult() *RealIsolationResult {
	return &RealIsolationResult{
		pid: 1,
	}
}

// Mounts returns mounts for the given process based on the supplied filter
func (r *RealIsolationResult) Mounts(filter mount.FilterFunc) ([]*mount.Info, error) {
	in, err := os.Open(fmt.Sprintf("/proc/%d/mountinfo", r.pid))
	if err != nil {
		return nil, fmt.Errorf("could not open file mountinfo for %d: %v", r.pid, err)
	}
	defer util.CloseIOAndCheckErr(in, nil)
	return mount.GetMountsFromReader(in, filter)
}

func mountInfoFor(r IsolationResult, mountPoint string) (mountinfo *mount.Info, err error) {
	mounts, err := r.Mounts(mount.SingleEntryFilter(mountPoint))
	if err != nil {
		return nil, fmt.Errorf("failed to process mountinfo for pid %d: %v", r.Pid(), err)
	}
	if len(mounts) <= 0 {
		return nil, fmt.Errorf("no '%s' mount point entry found for pid %d", mountPoint, r.Pid())
	}
	return mounts[0], nil
}

// MountInfoRoot returns the mount information for the root mount point
func MountInfoRoot(r IsolationResult) (mountinfo *mount.Info, err error) {
	return mountInfoFor(r, "/")
}

func mountsFilter(compare, m *mount.Info, source string) (bool, bool) {
	nfsMatch := false
	if strings.Contains(m.FSType, "nfs") && compare.FSType == m.FSType {
		nfsMatch = m.Source != source
	}

	return m.Major != compare.Major || m.Minor != compare.Minor ||
		!strings.HasPrefix(compare.Root, m.Root) || nfsMatch, false
}

// parentMountInfoFor takes the mountInfo record of a container (child) and
// attempts to locate a mountpoint containing it on the parent.
func parentMountInfoFor(parent IsolationResult, mountInfo *mount.Info, source string) (*mount.Info, error) {
	mounts, err := parent.Mounts(func(m *mount.Info) (bool, bool) {
		return mountsFilter(mountInfo, m, source)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find mount for %v in the mount namespace of pid %d", mountInfo.Root, parent.Pid())
	}

	if len(mounts) <= 0 {
		return nil, fmt.Errorf("no mount containing %v found in the mount namespace of pid %d", mountInfo.Root, parent.Pid())
	} else if len(mounts) > 1 {
		log.Log.Infof("found %d possible mount point candidates for path %v", len(mounts), mountInfo.Root)
		sort.SliceStable(mounts, func(i, j int) bool {
			return len(mounts[i].Root) > len(mounts[j].Root)
		})
	}

	return mounts[0], nil
}

func ParentPathForMount(parent IsolationResult, child IsolationResult, source, target string) (*safepath.Path, error) {
	childMountInfo, err := mountInfoFor(child, target)
	if err != nil {
		return nil, err
	}
	parentMountInfo, err := parentMountInfoFor(parent, childMountInfo, source)
	if err != nil {
		return nil, err
	}
	parentMountRoot, err := parent.MountRoot()
	if err != nil {
		return nil, err
	}
	path := parentMountRoot
	path, err = path.AppendAndResolveWithRelativeRoot(parentMountInfo.Mountpoint)
	if err != nil {
		return nil, err
	}
	return path.AppendAndResolveWithRelativeRoot(strings.TrimPrefix(childMountInfo.Root, parentMountInfo.Root))
}

// ParentPathForRootMount takes a container (child) and composes a path to
// the root mount point in the context of the parent.
func ParentPathForRootMount(parent IsolationResult, child IsolationResult) (*safepath.Path, error) {
	return ParentPathForMount(parent, child, "", "/")
}

func SafeJoin(res IsolationResult, elems ...string) (*safepath.Path, error) {
	mountRoot, err := res.MountRoot()
	if err != nil {
		return nil, err
	}
	return mountRoot.AppendAndResolveWithRelativeRoot(elems...)
}
