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
	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/disksource"
)

type blockResizeArgs struct {
	size  uint64
	flags libvirt.DomainBlockResizeFlags
}

// getBlockResizeArgs returns the arguments to pass to dom.BlockResize.
// For direct block devices, size is 0 with DOMAIN_BLOCK_RESIZE_CAPACITY
// so libvirt queries the real device size (handles LUKS headers correctly).
// For file-backed and overlay disks, size is the computed guest-visible size.
func getBlockResizeArgs(disk api.Disk, ds disksource.ResolvedDiskSource) (blockResizeArgs, bool) {
	if ds.BackendIsBlock() && !ds.HasOverlay() {
		return blockResizeArgs{
			size:  0,
			flags: libvirt.DOMAIN_BLOCK_RESIZE_BYTES | libvirt.DOMAIN_BLOCK_RESIZE_CAPACITY,
		}, true
	}
	guestSize, ok := possibleGuestSize(disk, ds)
	if !ok || guestSize == 0 {
		// only good reason to pass 0 as size is block device and no overlay
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
		trackSize := int64(rszArgs.size)
		if trackSize == 0 {
			// For block devices libvirt infers the size, so we track PVC capacity
			// as a change trigger. To report the actual guest-visible size back
			// (e.g. for VolumeStatus.CurrentGuestSize), we would need to query
			// the device size via ioctl/blockdev --getsize64 after resize.
			trackSize = *disk.Capacity
		}
		size, seen := l.guestDiskSizes[name]
		if !seen {
			logger.V(1).Infof("tracking disk %s with initial size %d", name, trackSize)
			l.guestDiskSizes[name] = trackSize
			continue
		}
		if size == trackSize {
			continue
		}
		logger.V(3).Infof("disk %s size changed from %d to %d", name, size, trackSize)
		logger.V(1).Infof("resizing disk %s with flags %d with size %d", name, rszArgs.flags, rszArgs.size)
		if err := dom.BlockResize(ds.SourcePath(), rszArgs.size, rszArgs.flags); err != nil {
			logger.Reason(err).Errorf("libvirt failed to expand disk image %v", disk)
			continue
		}
		l.guestDiskSizes[name] = trackSize
	}
}
