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
		return fmt.Errorf("Resolve diskdir : %v", err)
	}

	diskPath, err := safepath.JoinNoFollow(diskDir, img.Name)
	fileNotExists := errors.Is(err, unix.ENOENT)
	if err != nil && !fileNotExists {
		return fmt.Errorf("Resolve diskPath :%v", err)
	}

	if fileNotExists {
		if err := c.handleRequestedSizeAndCreateSparseRaw(vmi, diskDir, img); err != nil {
			return err
		}

		diskPath, err = safepath.JoinNoFollow(diskDir, img.Name)
		if err != nil {
			return fmt.Errorf("last %v", err)
		}
		if err := ephemeraldiskutils.DefaultOwnershipManager.SetFileOwnership(diskPath); err != nil {
			log.Log.Reason(err).Errorf("Couldn't set Ownership on %s: %v", diskPath, err)
			return err
		}
	}
	return nil
}

func (c *Creator) handleRequestedSizeAndCreateSparseRaw(vmi *v1.VirtualMachineInstance, diskDir *safepath.Path, img Image) error {
	size, err := c.dirBytesAvailableFunc(diskDir, c.minimumPVCReserveBytes)
	availableSize := int64(size)
	if err != nil {
		return err
	}
	requestedSize := img.Size
	if requestedSize > availableSize {
		requestedSize, err = c.shrinkRequestedSize(vmi, requestedSize, availableSize, img)
		if err != nil {
			return err
		}
	}
	err = createSparseRaw(diskDir, img.Name, requestedSize)
	if err != nil {
		log.Log.Reason(err).Errorf("Couldn't create a sparse raw file %s in %s, error: %v", img.Name, diskDir, err)
		return err
	}
	return nil
}

func (c *Creator) shrinkRequestedSize(vmi *v1.VirtualMachineInstance, requestedSize int64, availableSize int64, img Image) (int64, error) {
	// Some storage provisioners provide less space than requested, due to filesystem overhead etc.
	// We tolerate some difference in requested and available capacity up to some degree.
	// This can be configured with pvcTolerateLessSpaceUpToPercent in the KubeVirt CR's developerConfiguration.
	toleratedSize := requestedSize * (100 - int64(c.lessPVCSpaceToleration)) / 100
	if toleratedSize > availableSize {
		return 0, fmt.Errorf("unable to create %s, not enough space, demanded size %d B is bigger than available space %d B, also after taking %v %% toleration into account",
			filepath.Join(img.Dir, img.Name), uint64(requestedSize), availableSize, c.lessPVCSpaceToleration)
	}

	msg := fmt.Sprintf("PV size too small: expected %v B, found %v B. Using it anyway, it is within %v %% toleration", requestedSize, availableSize, c.lessPVCSpaceToleration)
	log.Log.Info(msg)
	c.recorder.Event(vmi, EventTypeToleratedSmallPV, EventReasonToleratedSmallPV, msg)
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
	return stat.Bavail*uint64(stat.Bsize) - reserve, nil
}

func createSparseRaw(diskdir *safepath.Path, diskName string, size int64) (err error) {
	offset := size - 1
	if filepath.Base(diskName) != diskName {
		return fmt.Errorf("Disk name needs to be base")
	}

	err = safepath.TouchAtNoFollow(diskdir, filepath.Base(diskName), 0666)
	if err != nil {
		return fmt.Errorf("Failed touch %s,%s : %v", diskdir, diskName, err)
	}

	diskPath, err := safepath.JoinNoFollow(diskdir, diskName)
	if err != nil {
		return fmt.Errorf("Failed append  %s,%s : %v", diskdir, diskName, err)
	}

	sFile, err := safepath.OpenAtNoFollow(diskPath)
	if err != nil {
		return fmt.Errorf("Failed NewFile %s,%s : %v", diskdir, diskName, err)
	}
	defer util.CloseIOAndCheckErr(sFile, &err)

	f, err := os.OpenFile(sFile.SafePath(), os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer util.CloseIOAndCheckErr(f, &err)

	_, err = f.WriteAt([]byte{0}, offset)
	if err != nil {
		return err
	}
	return nil
}
