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
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/disksource"
)

const (
	maxCustomBlockSizeS390x = 4096
)

func Convert_v1_BlockSize_To_api_BlockIO(source *v1.Disk, disk *api.Disk, arch string) error {
	if source.BlockSize == nil {
		return nil
	}

	if blockSize := source.BlockSize.Custom; blockSize != nil {
		if arch == "s390x" &&
			(blockSize.Logical > maxCustomBlockSizeS390x || blockSize.Physical > maxCustomBlockSizeS390x) {
			return fmt.Errorf(
				"custom block size (logical=%d, physical=%d) exceeds the maximum supported size of %d for architecture %s",
				blockSize.Logical, blockSize.Physical, maxCustomBlockSizeS390x, arch)
		}
		disk.BlockIO = &api.BlockIO{
			LogicalBlockSize:  blockSize.Logical,
			PhysicalBlockSize: blockSize.Physical,
		}
		// TODO: as of the time of writing this, KubeVirt uses libvirt < v11.6.0
		// which means that a discard_granularity value of 0 is omitted.
		// remove this comment once upgraded.
		if blockSize.DiscardGranularity != nil {
			disk.BlockIO.DiscardGranularity = pointer.P(*blockSize.DiscardGranularity)
		}
	} else if matchFeature := source.BlockSize.MatchVolume; matchFeature != nil && (matchFeature.Enabled == nil || *matchFeature.Enabled) {
		blockIO, err := getOptimalBlockIO(disk)
		if err != nil {
			return fmt.Errorf("failed to configure disk with block size detection enabled: %v", err)
		}
		disk.BlockIO = blockIO
	}
	return nil
}

func getOptimalBlockIO(disk *api.Disk) (*api.BlockIO, error) {
	if disk == nil {
		return nil, fmt.Errorf("disk is nil")
	}

	ds := disksource.Resolve(*disk)
	if ds.BackendIsBlock() {
		return getOptimalBlockIOForDevice(ds.BackendPath())
	} else if ds.BackendPath() != "" {
		return getOptimalBlockIOForFile(ds.BackendPath())
	}
	return nil, fmt.Errorf("disk is neither a block device nor a file")
}

func getOptimalBlockIOForDevice(path string) (*api.BlockIO, error) {
	safePath, err := safepath.JoinAndResolveWithRelativeRoot("/", path)
	if err != nil {
		return nil, err
	}
	fd, err := safepath.OpenAtNoFollow(safePath)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s. Reason: %w", safePath, err)
	}
	defer util.CloseIOAndCheckErr(fd, nil)

	f, err := os.OpenFile(fd.SafePath(), os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer util.CloseIOAndCheckErr(f, &err)

	logicalSize, err := unix.IoctlGetUint32(int(f.Fd()), unix.BLKSSZGET)
	if err != nil {
		return nil, fmt.Errorf("unable to get logical block size from device %s: %w", path, err)
	}
	physicalSize, err := unix.IoctlGetUint32(int(f.Fd()), unix.BLKPBSZGET)
	if err != nil {
		return nil, fmt.Errorf("unable to get physical block size from device %s: %w", path, err)
	}

	log.Log.Infof("Detected logical size of %d and physical size of %d for device %s", logicalSize, physicalSize, path)

	if logicalSize == 0 && physicalSize == 0 {
		return nil, fmt.Errorf("block sizes returned by device %v are 0", path)
	}

	discardGranularity, err := getDiscardGranularity(safePath)
	if err != nil {
		return nil, err
	}

	log.Log.Infof("Detected discard granularity of %d for device %v", discardGranularity, path)

	blockIO := &api.BlockIO{
		LogicalBlockSize:   uint(logicalSize),
		PhysicalBlockSize:  uint(physicalSize),
		DiscardGranularity: pointer.P(uint(discardGranularity)),
	}
	if logicalSize == 0 || physicalSize == 0 {
		if logicalSize > physicalSize {
			log.Log.Infof("Invalid physical size %d. Matching it to the logical size %d", physicalSize, logicalSize)
			blockIO.PhysicalBlockSize = uint(logicalSize)
		} else {
			log.Log.Infof("Invalid logical size %d. Matching it to the physical size %d", logicalSize, physicalSize)
			blockIO.LogicalBlockSize = uint(physicalSize)
		}
	}
	if *blockIO.DiscardGranularity%blockIO.LogicalBlockSize != 0 {
		log.Log.Infof("Invalid discard granularity %d. Matching it to physical size %d", *blockIO.DiscardGranularity, blockIO.PhysicalBlockSize)
		blockIO.DiscardGranularity = pointer.P(uint(physicalSize))
	}
	return blockIO, nil
}

func getDiscardGranularity(safePath *safepath.Path) (uint64, error) {
	fileInfo, err := safepath.StatAtNoFollow(safePath)
	if err != nil {
		return 0, fmt.Errorf("could not stat file %s. Reason: %w", safePath.String(), err)
	}
	stat := fileInfo.Sys().(*syscall.Stat_t)
	rdev := uint64(stat.Rdev) //nolint:unconvert // Rdev is uint32 on e.g. MIPS.
	major := unix.Major(rdev)
	minor := unix.Minor(rdev)

	raw, err := os.ReadFile(fmt.Sprintf("/sys/dev/block/%d:%d/queue/discard_granularity", major, minor))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// On the off chance that we can't stat the discard granularity, set it to disabled.
			return 0, nil
		}
		return 0, fmt.Errorf("cannot read discard granularity for device %s: %w", safePath.String(), err)
	}
	discardGranularity, err := strconv.ParseUint(strings.TrimSpace(string(raw)), 10, 0)
	if err != nil {
		return 0, err
	}

	return discardGranularity, err
}

// getOptimalBlockIOForFile determines the optimal sizes based on the filesystem settings
// the VM's disk image is residing on. A filesystem does not differentiate between sizes.
// The physical size will always match the logical size. The rest is up to the filesystem.
func getOptimalBlockIOForFile(path string) (*api.BlockIO, error) {
	var statfs unix.Statfs_t
	if err := unix.Statfs(path, &statfs); err != nil {
		return nil, fmt.Errorf("failed to stat file %v: %v", path, err)
	}
	blockSize := uint(statfs.Bsize)
	return &api.BlockIO{
		LogicalBlockSize:   blockSize,
		PhysicalBlockSize:  blockSize,
		DiscardGranularity: &blockSize,
	}, nil
}
