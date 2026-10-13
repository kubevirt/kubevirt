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

package diskimg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/util"
)

const (
	EventReasonToleratedSmallPV = "ToleratedSmallPV"
	EventTypeToleratedSmallPV   = k8sv1.EventTypeNormal
)

type Image struct {
	Dir  string
	Name string
	Size int64
}

type Creator struct {
	dirBytesAvailableFunc  func(dir *safepath.Path, reserve uint64) (uint64, error)
	recorder               record.EventRecorder
	lessPVCSpaceToleration int
	minimumPVCReserveBytes uint64
	mountRoot              *safepath.Path
}

func NewCreator(recorder record.EventRecorder, lessPVCSpaceToleration int, minimumPVCReserveBytes uint64, mountRoot *safepath.Path) Creator {
	return Creator{
		dirBytesAvailableFunc:  dirBytesAvailable,
		recorder:               recorder,
		lessPVCSpaceToleration: lessPVCSpaceToleration,
		minimumPVCReserveBytes: minimumPVCReserveBytes,
		mountRoot:              mountRoot,
	}
}

func (c *Creator) Create(vmi *v1.VirtualMachineInstance, images []Image) error {
	for _, img := range images {
		if err := c.createIfMissing(vmi, img); err != nil {
			return err
		}
	}
	return nil
}

func (c *Creator) createIfMissing(vmi *v1.VirtualMachineInstance, img Image) error {
	diskDir, err := c.mountRoot.AppendAndResolveWithRelativeRoot(img.Dir)
	if err != nil {
		return fmt.Errorf("failed to resolve disk image directory %s: %w", img.Dir, err)
	}

	_, err = safepath.JoinNoFollow(diskDir, img.Name)
	if err == nil {
		return nil
	}
	if !errors.Is(err, unix.ENOENT) {
		return fmt.Errorf("failed to resolve disk image %s in %s: %w", img.Name, diskDir, err)
	}

	size, err := c.imageSize(diskDir, img)
	if err != nil {
		return err
	}
	diskPath, err := createSparseRaw(diskDir, img.Name, size)
	if err != nil {
		return fmt.Errorf("failed to create disk image %s in %s: %w", img.Name, diskDir, err)
	}
	if err := ephemeraldiskutils.DefaultOwnershipManager.SetFileOwnership(diskPath); err != nil {
		return fmt.Errorf("failed to set ownership of disk image %s in %s: %w", img.Name, diskDir, err)
	}
	if size < img.Size {
		msg := fmt.Sprintf("PV size too small: expected %v B, found %v B. Using it anyway, it is within %v %% toleration", img.Size, size, c.lessPVCSpaceToleration)
		log.Log.Info(msg)
		c.recorder.Event(vmi, EventTypeToleratedSmallPV, EventReasonToleratedSmallPV, msg)
	}
	return nil
}

func (c *Creator) imageSize(diskDir *safepath.Path, img Image) (int64, error) {
	available, err := c.dirBytesAvailableFunc(diskDir, c.minimumPVCReserveBytes)
	if err != nil {
		return 0, fmt.Errorf("failed to determine available space in %s: %w", diskDir, err)
	}
	if img.Size <= int64(available) {
		return img.Size, nil
	}
	return c.shrinkRequestedSize(int64(available), img)
}

func (c *Creator) shrinkRequestedSize(availableSize int64, img Image) (int64, error) {
	// Some storage provisioners provide less space than requested, due to filesystem overhead etc.
	// We tolerate some difference in requested and available capacity up to some degree.
	// This can be configured with pvcTolerateLessSpaceUpToPercent in the KubeVirt CR's developerConfiguration.
	toleratedSize := img.Size * (100 - int64(c.lessPVCSpaceToleration)) / 100
	if toleratedSize > availableSize {
		return 0, fmt.Errorf("unable to create %s, not enough space, demanded size %d B is bigger than available space %d B, also after taking %v %% toleration into account",
			filepath.Join(img.Dir, img.Name), uint64(img.Size), availableSize, c.lessPVCSpaceToleration)
	}
	return availableSize, nil
}

func dirBytesAvailable(dir *safepath.Path, reserve uint64) (uint64, error) {
	var stat syscall.Statfs_t
	err := dir.ExecuteNoFollow(func(safePath string) error {
		return syscall.Statfs(safePath, &stat)
	})
	if err != nil {
		return 0, err
	}
	free := stat.Bavail * uint64(stat.Bsize)
	if free <= reserve {
		return 0, nil
	}
	return free - reserve, nil
}

func createSparseRaw(diskDir *safepath.Path, diskName string, size int64) (*safepath.Path, error) {
	if err := safepath.TouchAtNoFollow(diskDir, diskName, 0666); err != nil {
		return nil, err
	}
	diskPath, err := safepath.JoinNoFollow(diskDir, diskName)
	if err != nil {
		return nil, err
	}
	err = diskPath.ExecuteNoFollow(func(safePath string) (err error) {
		f, err := os.OpenFile(safePath, os.O_WRONLY, 0)
		if err != nil {
			return err
		}
		defer util.CloseIOAndCheckErr(f, &err)
		_, err = f.WriteAt([]byte{0}, size-1)
		return err
	})
	if err != nil {
		return nil, err
	}
	return diskPath, nil
}
