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
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	osdisk "kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
)

func IsChangedBlockTrackingEnabled(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Status.ChangedBlockTracking != nil &&
		vmi.Status.ChangedBlockTracking.State == v1.ChangedBlockTrackingEnabled
}

func IsChangedBlockTrackingInitializing(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Status.ChangedBlockTracking != nil &&
		vmi.Status.ChangedBlockTracking.State == v1.ChangedBlockTrackingInitializing
}

func ShouldCreateQCOW2Overlay(vmi *v1.VirtualMachineInstance) bool {
	return IsChangedBlockTrackingInitializing(vmi)
}

func ShouldApplyChangedBlockTracking(vmi *v1.VirtualMachineInstance) bool {
	return IsChangedBlockTrackingInitializing(vmi) ||
		IsChangedBlockTrackingEnabled(vmi)
}

var CreateQCOW2Overlay = createQCOW2OverlayFunc

func createQCOW2OverlayFunc(overlayPath, imagePath string, blockDev bool) error {
	if _, err := os.Stat(overlayPath); err == nil {
		log.Log.V(3).Infof("overlay %s already exists", overlayPath)
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		log.Log.Reason(err).Errorf("Error checking QCOW2 overlay %s existence", overlayPath)
		return err
	}

	_, err := os.Create(overlayPath)
	if err != nil {
		log.Log.Reason(err).Errorf("Error creating file QCOW2 overlay %s", overlayPath)
		return err
	}

	defer func(path string) {
		if err != nil {
			log.Log.Errorf("Deleting QCOW2 overlay %s due to failure %s", path, err)
			os.Remove(path)
		}
	}(overlayPath)

	info, err := osdisk.GetDiskInfo(imagePath)
	if err != nil {
		return fmt.Errorf("failed to get image info for image %q", imagePath)
	}
	overlaySize := info.VirtualSize

	qmpCapabilities := `{"execute": "qmp_capabilities"}`
	blockdevCreate := fmt.Sprintf(`{"execute": "blockdev-create", "arguments": {"job-id": "create", "options": {"driver": "qcow2", "file": "file", "data-file": "data-file", "data-file-raw": true, "size": %d}}}`, overlaySize)
	jobDismiss := `{"execute": "job-dismiss", "arguments": {"id": "create"}}`
	quit := `{"execute": "quit"}`
	cmdInput := fmt.Sprintf("%s\n%s\n%s\n%s\n", qmpCapabilities, blockdevCreate, jobDismiss, quit)

	args := append([]string{},
		"--chardev", "stdio,id=stdio", "--monitor", "stdio",
		"--blockdev", fmt.Sprintf("file,node-name=file,filename=%s", overlayPath))

	if blockDev {
		args = append(args, "--blockdev", fmt.Sprintf("host_device,node-name=data-file,filename=%s", imagePath))
	} else {
		args = append(args, "--blockdev", fmt.Sprintf("file,node-name=data-file,filename=%s", imagePath))
	}

	log.Log.V(3).Infof("QCOW2 overlay execute %v", args)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "qemu-storage-daemon", args...)
	cmd.Stdin = bytes.NewBufferString(cmdInput)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create QCOW2 overlay %s: %v, output: %s", overlayPath, err, output)
	}

	log.Log.Infof("QCOW2 overlay %s created successfully", overlayPath)
	return nil
}

func ApplyChangedBlockTracking(vmi *v1.VirtualMachineInstance, c *converter.ConverterContext) error {
	logger := log.Log.Object(vmi)
	applyCBTMap := make(map[string]string)

	// create overlay for every disk supporting changedBlockTracking
	for _, volume := range vmi.Spec.Volumes {
		volumeName := volume.Name
		logger.V(3).Infof("Creating QCOW2 overlay for %+v", volume)
		if !cbt.IsCBTEligibleVolume(&volume) {
			logger.V(3).Infof("SKIP Creating QCOW2 overlay for %s", volume.Name)
			continue
		}

		overlayPath := cbt.GetQCOW2OverlayPath(vmi, volumeName)
		logger.V(3).Infof("QCOW2 overlay path is %s", overlayPath)
		if !ShouldCreateQCOW2Overlay(vmi) {
			applyCBTMap[volumeName] = overlayPath
			continue
		}

		var imagePath string
		blockDev := false
		if c.IsBlockPVC[volumeName] || c.IsBlockDV[volumeName] {
			imagePath = converter.GetBlockDeviceVolumePath(volumeName)
			blockDev = true
		} else {
			imagePath = converter.GetFilesystemVolumePath(volumeName)
		}

		err := CreateQCOW2Overlay(overlayPath, imagePath, blockDev)
		if err != nil {
			return err
		}
		applyCBTMap[volumeName] = overlayPath
	}

	c.ApplyCBT = applyCBTMap
	return nil
}
