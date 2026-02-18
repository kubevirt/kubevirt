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
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	osdisk "kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
)

func IsChangedBlockTrackingEnabled(vmi *v1.VirtualMachineInstance) bool {
	return cbt.CompareCBTState(vmi.Status.ChangedBlockTracking, v1.ChangedBlockTrackingEnabled)
}

func shouldCreateQCOW2Overlay(vmi *v1.VirtualMachineInstance, isHotplug bool, hotplugPhase v1.VolumePhase) bool {
	if cbt.CBTStateInitializing(vmi.Status.ChangedBlockTracking) {
		return true
	}

	if !IsChangedBlockTrackingEnabled(vmi) {
		return false
	}

	// For hotplug volumes with CBT enabled, only create overlay when mounted but not yet in domain.
	// Once VolumeReady, the volume is in the domain and the overlay was already created.
	return isHotplug && hotplugPhase == v1.HotplugVolumeMounted
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

	args := append([]string{},
		"--chardev", "stdio,id=stdio", "--monitor", "stdio",
		"--blockdev", fmt.Sprintf("file,node-name=file,filename=%s", overlayPath))

	if blockDev {
		args = append(args, "--blockdev", fmt.Sprintf("host_device,node-name=data-file,filename=%s", imagePath))
	} else {
		args = append(args, "--blockdev", fmt.Sprintf("file,node-name=data-file,filename=%s", imagePath))
	}

	log.Log.V(3).Infof("QCOW2 overlay execute %v", args)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "qemu-storage-daemon", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe for qemu-storage-daemon: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe for qemu-storage-daemon: %w", err)
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err = cmd.Start(); err != nil {
		return fmt.Errorf("failed to start qemu-storage-daemon: %w", err)
	}

	// Read QMP output in background; signal when blockdev-create job concludes.
	var outputBuf bytes.Buffer
	concluded := make(chan struct{})
	scanDone := make(chan struct{})
	go func() {
		defer close(scanDone)
		scanner := bufio.NewScanner(stdout)
		closed := false
		for scanner.Scan() {
			line := scanner.Text()
			outputBuf.WriteString(line + "\n")
			if !closed && strings.Contains(line, `"status": "concluded"`) {
				close(concluded)
				closed = true
			}
		}
	}()

	fmt.Fprintf(stdin, "%s\n%s\n", qmpCapabilities, blockdevCreate)

	var selectErr error
	select {
	case <-concluded:
		fmt.Fprintf(stdin, "%s\n%s\n", jobDismiss, quit)
	case <-scanDone:
		selectErr = fmt.Errorf("qemu-storage-daemon exited without job concluding for overlay %s", overlayPath)
	case <-ctx.Done():
		selectErr = fmt.Errorf("timed out waiting for qemu-storage-daemon to create overlay %s", overlayPath)
	}

	stdin.Close()
	waitErr := cmd.Wait()
	if selectErr != nil || waitErr != nil {
		return fmt.Errorf("failed to create QCOW2 overlay %s: %w, output: %s%s",
			overlayPath, errors.Join(selectErr, waitErr), outputBuf.String(), stderrBuf.String())
	}

	log.Log.Infof("QCOW2 overlay %s created successfully", overlayPath)
	return nil
}

func DeleteQCOW2Overlay(vmi *v1.VirtualMachineInstance, volumeName string) error {
	if !cbt.HasCBTStateEnabled(vmi.Status.ChangedBlockTracking) {
		return nil
	}

	overlayPath := cbt.GetQCOW2OverlayPath(vmi, volumeName)
	if err := os.Remove(overlayPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to delete QCOW2 overlay %s for volume %s: %w", overlayPath, volumeName, err)
	}
	log.Log.Infof("QCOW2 overlay %s deleted for unplugged volume %s", overlayPath, volumeName)
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

		hotplugStatus, isHotplug := c.HotplugVolumes[volumeName]

		if !shouldCreateQCOW2Overlay(vmi, isHotplug, hotplugStatus.Phase) {
			applyCBTMap[volumeName] = overlayPath
			continue
		}

		isBlock := c.IsBlockPVC[volumeName] || c.IsBlockDV[volumeName]
		imagePath := converter.GetVolumeImagePath(volumeName, isBlock, isHotplug)

		err := CreateQCOW2Overlay(overlayPath, imagePath, isBlock)
		if err != nil {
			return err
		}
		applyCBTMap[volumeName] = overlayPath
	}

	c.ApplyCBT = applyCBTMap
	return nil
}
