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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	mount "github.com/moby/sys/mountinfo"

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
	MountRoot() string
	// full path to the mount namespace
	MountNamespace() string
	// mounts for the process
	Mounts(mount.FilterFunc) ([]*mount.Info, error)
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

// IsMounted checks if the given path is a mount point or not. Works with symlinks.
func (r *RealIsolationResult) IsMounted(mountPoint string) (isMounted bool, err error) {
	mountPoint, err = filepath.Abs(mountPoint)
	if err != nil {
		return false, fmt.Errorf("failed to resolve %v to an absolute path: %v", mountPoint, err)
	}
	mountPoint, err = filepath.EvalSymlinks(mountPoint)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("could not resolve symlinks in path %v: %v", mountPoint, err)
	}
	return mount.Mounted(mountPoint)
}

// AreMounted checks if given paths are mounted by calling IsMounted.
// If error occurs, the first error is returned.
func (r *RealIsolationResult) AreMounted(mountPoints ...string) (isMounted bool, err error) {
	for _, mountPoint := range mountPoints {
		isMounted, err = r.IsMounted(mountPoint)
		if !isMounted || err != nil {
			return
		}
	}

	return true, nil
}

// IsBlockDevice checks if the given path is a block device or not.
func (r *RealIsolationResult) IsBlockDevice(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("error checking for block device: %v", err)
	}
	if fileInfo.IsDir() || (fileInfo.Mode()&os.ModeDevice) == 0 {
		return false, fmt.Errorf("found %v, but it's not a block device", path)
	}
	return true, nil
}

func (r *RealIsolationResult) MountRoot() string {
	return fmt.Sprintf("/proc/%d/root", r.pid)
}

func (r *RealIsolationResult) Pid() int {
	return r.pid
}

func (r *RealIsolationResult) PPid() int {
	return r.ppid
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

// MountInfoRoot returns the mount information for the root mount point
func MountInfoRoot(r IsolationResult) (mountInfo *mount.Info, err error) {
	mounts, err := r.Mounts(mount.SingleEntryFilter("/"))
	if err != nil {
		return nil, fmt.Errorf("failed to process mountinfo for pid %d: %v", r.Pid(), err)
	}
	if len(mounts) <= 0 {
		return nil, fmt.Errorf("no root mount point entry found for pid %d", r.Pid())
	}
	return mounts[0], nil
}

// parentMountInfoFor takes the mountInfo record of a container (child) and
// attempts to locate a mountpoint containing it on the parent.
func parentMountInfoFor(parent IsolationResult, mountInfo *mount.Info) (*mount.Info, error) {
	mounts, err := parent.Mounts(func(m *mount.Info) (bool, bool) {
		return m.Major != mountInfo.Major || m.Minor != mountInfo.Minor ||
			!strings.HasPrefix(mountInfo.Root, m.Root), false
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

// ParentPathForRootMount takes a container (child) and composes a path to
// the root mount point in the context of the parent.
func ParentPathForRootMount(parent IsolationResult, child IsolationResult) (string, error) {
	childRootMountInfo, err := MountInfoRoot(child)
	if err != nil {
		return "", err
	}
	parentMountInfo, err := parentMountInfoFor(parent, childRootMountInfo)
	if err != nil {
		return "", err
	}
	return filepath.Join(parent.MountRoot(), parentMountInfo.Mountpoint, strings.TrimPrefix(childRootMountInfo.Root, parentMountInfo.Root)), nil
}
