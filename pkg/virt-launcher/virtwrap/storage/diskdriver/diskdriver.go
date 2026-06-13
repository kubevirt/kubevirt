/*
 * This file is part of the KubeVirt project
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
 * Copyright The KubeVirt Authors.
 *
 */

package diskdriver

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/disksource"
)

const filePermissions = 0o600

type ioChecker interface {
	CheckBlockDevice(path string) (bool, error)
	CheckFile(path string) (bool, error)
}

type Configurator struct {
	ioChecker ioChecker
}

func New() *Configurator {
	return &Configurator{&directIOChecker{}}
}

func NewMock() *Configurator {
	return &Configurator{&mockIOChecker{}}
}

func (c *Configurator) SetDriverCacheMode(d *api.Disk) error {
	if d == nil {
		return fmt.Errorf("unable to set a driver cache mode, disk is nil")
	}

	t := disksource.Resolve(*d)

	if t.BackendPath() == "" {
		if d.Device == "cdrom" {
			return nil
		}
		return fmt.Errorf("unable to set a driver cache mode, disk has no backend path")
	}

	mode := v1.DriverCache(d.Driver.Cache)

	switch mode {
	case v1.CacheNone:
		if !c.supportDirectIO(t, d.BackingStore) {
			return fmt.Errorf("unable to use '%s' cache mode, file system where %s is stored does not support direct I/O", mode, t.BackendPath())
		}
	case "":
		if c.supportDirectIO(t, d.BackingStore) {
			mode = v1.CacheNone
		} else {
			mode = v1.CacheWriteThrough
		}
	case v1.CacheWriteThrough, v1.CacheWriteBack:
	}

	d.Driver.Cache = string(mode)
	log.Log.Infof("Driver cache mode for %s set to %s", t.BackendPath(), mode)

	return nil
}

func (c *Configurator) supportDirectIO(t disksource.ResolvedDiskSource, backingStore *api.BackingStore) bool {
	supported := c.checkPathDirectIO(t.BackendPath(), t.BackendIsBlock())

	// when the disk is backed-up by another file, we need to also check if that
	// file sits on a file system that supports direct I/O
	if backingStore != nil && backingStore.Source != nil {
		supported = supported && c.checkPathDirectIO(backingStore.Source.File, false)
	}

	return supported
}

func (c *Configurator) checkPathDirectIO(path string, isBlock bool) bool {
	var supported bool
	var err error
	if isBlock {
		supported, err = c.ioChecker.CheckBlockDevice(path)
	} else {
		supported, err = c.ioChecker.CheckFile(path)
	}
	if err != nil {
		log.Log.Reason(err).Errorf("Direct IO check failed for %s", path)
	} else if !supported {
		log.Log.Infof("%s file system does not support direct I/O", path)
	}
	return supported
}

func SetOptimalIOMode(d *api.Disk) {
	if d == nil {
		return
	}

	ds := disksource.Resolve(*d)

	if d.Driver.IO != "" {
		return
	}

	if ds.BackendPath() == "" {
		return
	}

	if v1.DriverCache(d.Driver.Cache) == v1.CacheNone {
		if ds.BackendIsBlock() || IsPreAllocated(ds.BackendPath()) {
			d.Driver.IO = v1.IONative
		}
	}
	if d.Driver.IO != "" {
		log.Log.Infof("Driver IO mode for %s set to %s", ds.BackendPath(), d.Driver.IO)
	}
}

var IsPreAllocated = func(path string) bool {
	diskInf, err := disk.GetDiskInfo(path)
	if err != nil {
		return false
	}
	return diskInf.VirtualSize <= diskInf.ActualSize
}

// directIOChecker probes whether a path's filesystem supports O_DIRECT.
// Based on https://gitlab.com/qemu-project/qemu/-/blob/master/util/osdep.c#L344
type directIOChecker struct{}

func (c *directIOChecker) CheckBlockDevice(path string) (bool, error) {
	return c.check(path, syscall.O_RDONLY)
}

func (c *directIOChecker) CheckFile(path string) (bool, error) {
	flags := syscall.O_RDONLY
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		flags |= syscall.O_CREAT
		defer os.Remove(path)
	}
	return c.check(path, flags)
}

func (c *directIOChecker) check(path string, flags int) (bool, error) {
	// #nosec No risk for path injection as we only open the file, not read from it.
	// The function leaks only whether the directory to `path` exists.
	f, err := os.OpenFile(path, flags|syscall.O_DIRECT, filePermissions)
	if err == nil {
		defer util.CloseIOAndCheckErr(f, nil)
		return true, nil
	}

	var pathErr *os.PathError
	if !errors.As(err, &pathErr) || pathErr.Err != syscall.EINVAL {
		return false, err
	}

	f, err = os.OpenFile(path, flags&^syscall.O_DIRECT, filePermissions)
	if err != nil {
		return false, err
	}
	defer util.CloseIOAndCheckErr(f, nil)
	return false, nil
}

type mockIOChecker struct{}

func (m *mockIOChecker) CheckBlockDevice(_ string) (bool, error) { return true, nil }
func (m *mockIOChecker) CheckFile(_ string) (bool, error)        { return true, nil }
