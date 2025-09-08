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

package emptydisk

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/util"
)

const emptyDiskBaseDir = "/var/run/libvirt/empty-disks/"

type emptyDiskCreator struct {
	emptyDiskBaseDir string
	discCreateFunc   func(filePath string, size string) error
}

func (c *emptyDiskCreator) CreateTemporaryDisks(vmi *v1.VirtualMachineInstance) error {
	logger := log.Log.Object(vmi)

	for _, volume := range vmi.Spec.Volumes {
		if volume.EmptyDisk != nil {
			// qemu-img takes the size in bytes or in Kibibytes/Mebibytes/...; lets take bytes
			intSize := volume.EmptyDisk.Capacity.ToDec().ScaledValue(0)
			// round down the size to the nearest 1MiB multiple
			intSize = util.AlignImageSizeTo1MiB(intSize, logger.With("volume", volume.Name))
			if intSize == 0 {
				return fmt.Errorf("the size for volume %s is too low", volume.Name)
			}
			// convert the size to string for qemu-img
			size := strconv.FormatInt(intSize, 10)
			file := filePathForVolumeName(c.emptyDiskBaseDir, volume.Name)
			if err := util.MkdirAllWithNosec(c.emptyDiskBaseDir); err != nil {
				return err
			}
			if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
				if err := c.discCreateFunc(file, size); err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
			if err := ephemeraldiskutils.DefaultOwnershipManager.UnsafeSetFileOwnership(file); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *emptyDiskCreator) FilePathForVolumeName(volumeName string) string {
	return filePathForVolumeName(c.emptyDiskBaseDir, volumeName)
}

func filePathForVolumeName(basedir string, volumeName string) string {
	return path.Join(basedir, volumeName+".qcow2")
}

func createQCOW(file string, size string) error {
	// #nosec No risk for attacker injection. Parameters are predefined strings
	return exec.Command("qemu-img", "create", "-f", "qcow2", file, size).Run()
}

func NewEmptyDiskCreator() *emptyDiskCreator {
	return &emptyDiskCreator{
		emptyDiskBaseDir: emptyDiskBaseDir,
		discCreateFunc:   createQCOW,
	}
}
