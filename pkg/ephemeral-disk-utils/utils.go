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
 * Copyright The KubeVirt Authors.
 *
 */

package ephemeraldiskutils

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// TODO this should be part of structs, instead of a global
var (
	DefaultOwnershipManager  OwnershipManagerInterface = &OwnershipManager{uid: util.NonRootUID, gid: util.NonRootUID}
	DiskFileOwnershipManager OwnershipManagerInterface = &OwnershipManager{uid: util.QemuUID, gid: util.NonRootUID}
)

// For testing
func MockDefaultOwnershipManager() {
	DefaultOwnershipManager = &nonOpManager{}
	DiskFileOwnershipManager = &nonOpManager{}
}

type nonOpManager struct {
}

func (no *nonOpManager) UnsafeSetFileOwnership(_ string) error {
	return nil
}

func (no *nonOpManager) SetFileOwnership(_ *safepath.Path) error {
	return nil
}

func MockDefaultOwnershipManagerWithFailure() {
	DefaultOwnershipManager = &failureManager{}
	DiskFileOwnershipManager = &failureManager{}
}

type failureManager struct {
}

func (no *failureManager) UnsafeSetFileOwnership(_ string) error {
	panic("unexpected call to UnsafeSetFileOwnership")
}

func (no *failureManager) SetFileOwnership(_ *safepath.Path) error {
	panic("unexpected call to SetFileOwnership")
}

type OwnershipManager struct {
	uid int
	gid int
}

func (om *OwnershipManager) SetFileOwnership(file *safepath.Path) error {
	fd, err := safepath.OpenAtNoFollow(file)
	if err != nil {
		return err
	}
	defer fd.Close()
	return om.UnsafeSetFileOwnership(fd.SafePath())
}

func (om *OwnershipManager) UnsafeSetFileOwnership(file string) error {
	fileInfo, err := os.Stat(file)
	if err != nil {
		return err
	}

	if stat, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
		if om.uid == int(stat.Uid) && om.gid == int(stat.Gid) {
			return nil
		}
	} else {
		return fmt.Errorf("failed to convert stat info")
	}

	return os.Chown(file, om.uid, om.gid)
}

func RemoveFilesIfExist(paths ...string) error {
	var err error
	for _, path := range paths {
		err = os.Remove(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}
func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	exists := false

	if err == nil {
		exists = true
	} else if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	return exists, err
}

type OwnershipManagerInterface interface {
	// Deprecated: UnsafeSetFileOwnership should not be used. Use SetFileOwnership instead.
	UnsafeSetFileOwnership(file string) error
	SetFileOwnership(file *safepath.Path) error
}

func GetEphemeralBackingSourceBlockDevices(domain *api.Domain) map[string]bool {
	isDevEphemeralBackingSource := make(map[string]bool)
	for _, disk := range domain.Spec.Devices.Disks {
		if disk.BackingStore != nil && disk.BackingStore.Source != nil {
			if disk.BackingStore.Type == "block" && disk.BackingStore.Source.Dev != "" && disk.BackingStore.Source.Name != "" {
				isDevEphemeralBackingSource[disk.BackingStore.Source.Name] = true
			}
		}
	}
	return isDevEphemeralBackingSource
}
