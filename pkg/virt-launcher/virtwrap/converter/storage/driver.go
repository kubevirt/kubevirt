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

package storage

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/disksource"
)

//nolint:gocyclo
func SetDriverCacheMode(d *api.Disk, directIOChecker DirectIOChecker) error {
	if d == nil {
		return fmt.Errorf("unable to set a driver cache mode, disk is nil")
	}

	t := disksource.Resolve(*d)

	if t.BackendPath() == "" {
		if d.Device == deviceCdrom {
			return nil
		}
		return fmt.Errorf("unable to set a driver cache mode, disk has no backend path")
	}

	var err error
	supportDirectIO := true
	mode := v1.DriverCache(d.Driver.Cache)

	if mode == "" || mode == v1.CacheNone {
		if t.BackendIsBlock() {
			supportDirectIO, err = directIOChecker.CheckBlockDevice(t.BackendPath())
		} else {
			supportDirectIO, err = directIOChecker.CheckFile(t.BackendPath())
		}
		if err != nil {
			log.Log.Reason(err).Errorf("Direct IO check failed for %s", t.BackendPath())
		} else if !supportDirectIO {
			log.Log.Infof("%s file system does not support direct I/O", t.BackendPath())
		}
		// when the disk is backed-up by another file, we need to also check if that
		// file sits on a file system that supports direct I/O
		if backingFile := d.BackingStore; backingFile != nil {
			backingFilePath := backingFile.Source.File
			backFileDirectIOSupport, err := directIOChecker.CheckFile(backingFilePath)
			if err != nil {
				log.Log.Reason(err).Errorf("Direct IO check failed for %s", backingFilePath)
			} else if !backFileDirectIOSupport {
				log.Log.Infof("%s backing file system does not support direct I/O", backingFilePath)
			}
			supportDirectIO = supportDirectIO && backFileDirectIOSupport
		}
	}

	// if user set a cache mode = 'none' and fs does not support direct I/O then return an error
	if mode == v1.CacheNone && !supportDirectIO {
		return fmt.Errorf("unable to use '%s' cache mode, file system where %s is stored does not support direct I/O", mode, t.BackendPath())
	}

	// if user did not set a cache mode and fs supports direct I/O then set cache = 'none'
	// else set cache = 'writethrough
	if mode == "" && supportDirectIO {
		mode = v1.CacheNone
	} else if mode == "" && !supportDirectIO {
		mode = v1.CacheWriteThrough
	}

	d.Driver.Cache = string(mode)
	log.Log.Infof("Driver cache mode for %s set to %s", t.BackendPath(), mode)

	return nil
}

func IsPreAllocated(path string) bool {
	diskInf, err := disk.GetDiskInfo(path)
	if err != nil {
		return false
	}
	// ActualSize can be a little larger then VirtualSize for qcow2
	return diskInf.VirtualSize <= diskInf.ActualSize
}

// Set optimal io mode automatically
func SetOptimalIOMode(d *api.Disk, isPreAllocated func(path string) bool) {
	if d == nil {
		return
	}

	ds := disksource.Resolve(*d)

	// If the user explicitly set the io mode do nothing
	if d.Driver.IO != "" {
		return
	}

	if ds.BackendPath() == "" {
		return
	}

	// O_DIRECT is needed for io="native"
	if v1.DriverCache(d.Driver.Cache) == v1.CacheNone {
		// set native for block device or pre-allocateed image file
		if ds.BackendIsBlock() || isPreAllocated(ds.BackendPath()) {
			d.Driver.IO = v1.IONative
		}
	}
	// For now we don't explicitly set io=threads even for sparse files as it's
	// not clear it's better for all use-cases
	if d.Driver.IO != "" {
		log.Log.Infof("Driver IO mode for %s set to %s", ds.BackendPath(), d.Driver.IO)
	}
}
