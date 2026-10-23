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

package virtwrap

import (
	"errors"
	"fmt"
	"os"

	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/disksource"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
)

type blockResizeArgs struct {
	size  uint64
	flags libvirt.DomainBlockResizeFlags
}

// getBlockResizeArgs returns the arguments for dom.BlockResize.
// Block backends use CAPACITY to let libvirt determine the usable size,
// including LUKS overhead. File-backed disks use an explicit guest size.
func getBlockResizeArgs(disk api.Disk, ds disksource.ResolvedDiskSource) (blockResizeArgs, bool) {
	if ds.BackendIsBlock() {
		return blockResizeArgs{
			size:  0,
			flags: libvirt.DOMAIN_BLOCK_RESIZE_BYTES | libvirt.DOMAIN_BLOCK_RESIZE_CAPACITY,
		}, true
	}
	if _, err := os.Stat(ds.BackendPath()); errors.Is(err, os.ErrNotExist) {
		return blockResizeArgs{}, false
	}
	guestSize, ok := possibleGuestSize(disk, ds)
	if !ok || guestSize == 0 {
		// file-backed disks need an explicit, non-zero guest size; size 0 is
		// only valid together with CAPACITY, used above for block backends
		// https://libvirt.org/html/libvirt-libvirt-domain.html#virDomainBlockResize
		return blockResizeArgs{}, false
	}
	return blockResizeArgs{
		size:  uint64(guestSize),
		flags: libvirt.DOMAIN_BLOCK_RESIZE_BYTES,
	}, true
}

func (l *LibvirtDomainManager) expandDisksOnline(dom cli.VirDomain, domain *api.Domain, vmi *v1.VirtualMachineInstance) {
	logger := log.Log.Object(vmi)
	for _, disk := range domain.Spec.Devices.Disks {
		name := disk.Alias.GetName()
		if !isPVCBacked(name, vmi) || disk.Capacity == nil {
			continue
		}
		ds := disksource.Resolve(disk)
		rszArgs, ok := getBlockResizeArgs(disk, ds)
		if !ok {
			logger.V(3).Infof("skipping resize of disk %s: unable to determine guest size", name)
			continue
		}
		trackSize := *disk.Capacity
		size, seen := l.guestDiskSizes[name]
		if !seen {
			logger.V(1).Infof("tracking disk %s with initial size %d", name, trackSize)
			l.guestDiskSizes[name] = trackSize
			continue
		}
		if size >= trackSize {
			continue
		}
		logger.V(3).Infof("disk %s size changed from %d to %d", name, size, trackSize)
		logger.V(1).Infof("resizing disk %s with flags %d with size %d", name, rszArgs.flags, rszArgs.size)
		if err := l.resizeWithFallback(dom, ds, rszArgs, logger); err != nil {
			logger.Reason(err).Errorf("libvirt failed to expand disk image %v", disk)
			continue
		}
		l.guestDiskSizes[name] = trackSize
	}
}

// resizeWithFallback retries unsupported overlay-on-block resizes with an
// explicit size for compatibility with older libvirt builds.
//
// TODO(#18505): remove the fallback once all supported libvirt builds (>=12.3.0)
// support CAPACITY for qcow2 images with a raw block data-file.
func (l *LibvirtDomainManager) resizeWithFallback(dom cli.VirDomain, ds disksource.ResolvedDiskSource, rszArgs blockResizeArgs, logger *log.FilteredLogger) error {
	err := dom.BlockResize(ds.SourcePath(), rszArgs.size, rszArgs.flags)
	if err == nil {
		return nil
	}
	if !ds.HasOverlay() || !ds.BackendIsBlock() || !domainerrors.IsOperationUnsupported(err) {
		return err
	}

	diskInfo, infoErr := l.diskInfoGetter(ds.BackendPath())
	if infoErr != nil {
		return fmt.Errorf("failed to get size of block data-file %s: %w", ds.BackendPath(), infoErr)
	}
	if diskInfo.VirtualSize <= 0 {
		return fmt.Errorf("invalid virtual size %d for block data-file %s", diskInfo.VirtualSize, ds.BackendPath())
	}
	logger.V(1).Reason(err).Infof("CAPACITY resize unsupported for %s, retrying with explicit size %d", ds.SourcePath(), diskInfo.VirtualSize)
	return dom.BlockResize(ds.SourcePath(), uint64(diskInfo.VirtualSize), libvirt.DOMAIN_BLOCK_RESIZE_BYTES)
}
