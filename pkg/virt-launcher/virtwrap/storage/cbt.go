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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	osdisk "kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
)

func DiskHasDataStore(disk *api.Disk) bool {
	return disk != nil && disk.Source.DataStore != nil
}

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
		err = fmt.Errorf("failed to get image info for image %q: %w", imagePath, err)
		return err
	}
	overlaySize := info.VirtualSize

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

	stdin, err := cmd.StdinPipe()
	if err != nil {
		err = fmt.Errorf("failed to create stdin pipe for qemu-storage-daemon: %w", err)
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		err = fmt.Errorf("failed to create stdout pipe for qemu-storage-daemon: %w", err)
		return err
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err = cmd.Start(); err != nil {
		err = fmt.Errorf("failed to start qemu-storage-daemon: %w", err)
		return err
	}

	output, sessionErr := runOverlayQMPSession(ctx, stdin, stdout, overlaySize, overlayPath)
	waitErr := cmd.Wait()
	if sessionErr != nil || waitErr != nil {
		err = fmt.Errorf("failed to create QCOW2 overlay %s: %w, output: %s%s",
			overlayPath, errors.Join(sessionErr, waitErr), output, stderrBuf.String())
		return err
	}

	log.Log.Infof("QCOW2 overlay %s created successfully", overlayPath)
	return nil
}

func runOverlayQMPSession(ctx context.Context, stdin io.WriteCloser, stdout io.Reader,
	overlaySize int64, overlayPath string) (string, error) {

	qmpCapabilities := `{"execute": "qmp_capabilities"}`
	blockdevCreate := fmt.Sprintf(`{"execute": "blockdev-create", "arguments": {"job-id": "create", "options": {"driver": "qcow2", "file": "file", "data-file": "data-file", "data-file-raw": true, "size": %d}}}`, overlaySize)
	queryJobs := `{"execute": "query-jobs"}`
	jobDismiss := `{"execute": "job-dismiss", "arguments": {"id": "create"}}`
	quit := `{"execute": "quit"}`

	var outputBuf bytes.Buffer
	var jobErr, scanErr error
	concludedChan := make(chan struct{})
	scanDone := make(chan struct{})
	go func() {
		defer close(scanDone)
		scanner := bufio.NewScanner(stdout)
		concluded := false
		for scanner.Scan() {
			line := scanner.Text()
			outputBuf.WriteString(line + "\n")
			resp := parseQMPResponse(line)
			if !concluded && resp.isJobConcluded() {
				close(concludedChan)
				concluded = true
			}
			if concluded && jobErr == nil {
				jobErr = resp.jobError()
			}
		}
		scanErr = scanner.Err()
	}()

	fmt.Fprintf(stdin, "%s\n%s\n", qmpCapabilities, blockdevCreate)

	var err error
	select {
	case <-concludedChan:
		fmt.Fprintf(stdin, "%s\n%s\n%s\n", queryJobs, jobDismiss, quit)
	case <-scanDone:
		err = fmt.Errorf("qemu-storage-daemon exited without job concluding for overlay %s", overlayPath)
	case <-ctx.Done():
		err = fmt.Errorf("timed out waiting for qemu-storage-daemon to create overlay %s", overlayPath)
	}

	stdin.Close()
	<-scanDone

	if scanErr != nil {
		err = errors.Join(err, fmt.Errorf("error reading qemu-storage-daemon output for overlay %s: %w", overlayPath, scanErr))
	}
	if jobErr != nil {
		err = errors.Join(err, fmt.Errorf("blockdev-create job failed for overlay %s: %w", overlayPath, jobErr))
	}

	return outputBuf.String(), err
}

type jobInfo struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error"`
}

type qmpResponse struct {
	Event  string    `json:"event"`
	Data   jobInfo   `json:"data"`
	Return []jobInfo `json:"return"`
}

func parseQMPResponse(line string) qmpResponse {
	var resp qmpResponse
	json.Unmarshal([]byte(line), &resp)
	return resp
}

func (r qmpResponse) isJobConcluded() bool {
	return r.Event == "JOB_STATUS_CHANGE" && r.Data.Status == "concluded"
}

func (r qmpResponse) jobError() error {
	for _, job := range r.Return {
		if job.ID == "create" && job.Error != "" {
			return errors.New(job.Error)
		}
	}
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

func isBackendStorageRWO(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Status.MigrationState == nil {
		return false
	}
	sourcePVC := vmi.Status.MigrationState.SourcePersistentStatePVCName
	targetPVC := vmi.Status.MigrationState.TargetPersistentStatePVCName
	return sourcePVC != "" && targetPVC != "" && sourcePVC != targetPVC
}

// ApplyChangedBlockTrackingForMigration creates qcow2 overlays on the migration target.
// For RWX backend storage, the existing overlay is used.
// For RWO backend storage, new overlays are created on the target backend PVC.
func ApplyChangedBlockTrackingForMigration(vmi *v1.VirtualMachineInstance, c *converter.ConverterContext) error {
	logger := log.Log.Object(vmi)
	applyCBTMap := make(map[string]string)

	for _, volume := range vmi.Spec.Volumes {
		if !cbt.IsCBTEligibleVolume(&volume) {
			continue
		}

		volumeName := volume.Name
		overlayPath := cbt.GetQCOW2OverlayPath(vmi, volumeName)

		if isBackendStorageRWO(vmi) {
			_, isHotplug := c.HotplugVolumes[volumeName]
			isBlock := c.IsBlockPVC[volumeName] || c.IsBlockDV[volumeName]
			imagePath := converter.GetVolumeImagePath(volumeName, isBlock, isHotplug)

			logger.V(3).Infof("Creating CBT overlay for migration: %s -> %s (block=%v, hotplug=%v)", overlayPath, imagePath, isBlock, isHotplug)
			if err := CreateQCOW2Overlay(overlayPath, imagePath, isBlock); err != nil {
				return fmt.Errorf("failed to create CBT overlay for volume %s: %v", volumeName, err)
			}
		} else {
			logger.V(3).Infof("Using existing CBT overlay for migration (RWX backend): %s", overlayPath)
		}

		applyCBTMap[volumeName] = overlayPath
	}

	c.ApplyCBT = applyCBTMap
	return nil
}
