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
	"os"
	"path/filepath"
	"strings"

	"github.com/containernetworking/plugins/pkg/ns"

	"kubevirt.io/kubevirt/pkg/util"
)

type MountInfo struct {
	DeviceContainingFile string
	Root                 string
	MountPoint           string
}

// The unit test suite overwrites this function
var mountInfoFunc = func(pid int) string {
	return fmt.Sprintf("/proc/%d/mountinfo", pid)
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
	MountRoot() string
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

// IsMounted checks if a path in the mount namespace of a
// given process isolation result is a mount point. Works with symlinks.
func (r *realIsolationResult) IsMounted(mountPoint string) (isMounted bool, err error) {
	mountPoint, err = filepath.EvalSymlinks(mountPoint)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("could not resolve mount point path: %v", err)
	}
	if err = forEachRecord(r.mountInfo(), func(record []string) bool {
		isMounted = record[4] == mountPoint
		return isMounted
	}); err != nil {
		return false, err
	}
	return
}

// IsBlockDevice check if the path given is a block device or not.
func (r *realIsolationResult) IsBlockDevice(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
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
func (r *realIsolationResult) FullPath(mountInfo *MountInfo) (path string, err error) {
	// Handle btrfs subvolumes: mountInfo.Root seems to already provide the needed path
	if strings.HasPrefix(mountInfo.Root, "/@") {
		path = filepath.Join(r.MountRoot(), strings.TrimPrefix(mountInfo.Root, "/@"))
		return
	}

	parentMountInfo, err := r.ParentMountInfoFor(mountInfo)
	if err != nil {
		return
	}
	path = filepath.Join(r.MountRoot(), parentMountInfo.Root, parentMountInfo.MountPoint, mountInfo.Root)
	return
}

func (r *realIsolationResult) NetNamespace() string {
	return fmt.Sprintf("/proc/%d/ns/net", r.pid)
}

func (r *realIsolationResult) MountRoot() string {
	return fmt.Sprintf("/proc/%d/root", r.pid)
}

func (r *realIsolationResult) Pid() int {
	return r.pid
}

func (r *realIsolationResult) Controller() []string {
	return r.controller
}

func NodeIsolationResult() *realIsolationResult {
	return &realIsolationResult{
		pid: 1,
	}
}
