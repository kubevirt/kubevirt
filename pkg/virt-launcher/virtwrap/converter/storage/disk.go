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
	"slices"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/storage/reservation"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/iothreads"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/virtio"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/disksource"
)

const maxCustomBlockSizeS390x = 4096

type DeviceNamer struct {
	existingNameMap map[string]string
	usedDeviceMap   map[string]string
}

func assignDiskToSCSIController(disk *api.Disk, unit int) {
	// Ensure we assign this disk to the correct scsi controller
	if disk.Address == nil {
		disk.Address = &api.Address{}
	}
	disk.Address.Type = "drive"
	// This should be the index of the virtio-scsi controller, which is hard coded to 0
	disk.Address.Controller = "0"
	disk.Address.Bus = "0"
	disk.Address.Unit = strconv.Itoa(unit)
}

func Convert_v1_Disk_To_api_Disk(c *convertertypes.ConverterContext, diskDevice *v1.Disk, disk *api.Disk, prefixMap map[string]DeviceNamer, numQueues *uint, volumeStatusMap map[string]v1.VolumeStatus) error {
	if diskDevice.Disk != nil {
		var unit int
		disk.Device = "disk"
		disk.Target.Bus = diskDevice.Disk.Bus
		disk.Target.Device, unit = makeDeviceName(diskDevice.Name, diskDevice.Disk.Bus, prefixMap)
		if diskDevice.Disk.Bus == "scsi" {
			assignDiskToSCSIController(disk, unit)
		}
		if diskDevice.Disk.PciAddress != "" {
			if diskDevice.Disk.Bus != v1.DiskBusVirtio {
				return fmt.Errorf("setting a pci address is not allowed for non-virtio bus types, for disk %s", diskDevice.Name)
			}
			addr, err := device.NewPciAddressField(diskDevice.Disk.PciAddress)
			if err != nil {
				return fmt.Errorf("failed to configure disk %s: %v", diskDevice.Name, err)
			}
			disk.Address = addr
		}
		if diskDevice.Disk.Bus == v1.DiskBusVirtio {
			disk.Model = virtio.InterpretTransitionalModelType(&c.UseVirtioTransitional, c.Architecture.GetArchitecture())
		}
		disk.ReadOnly = ToApiReadOnly(diskDevice.Disk.ReadOnly)
		disk.Serial = diskDevice.Serial
		if diskDevice.Shareable != nil {
			if *diskDevice.Shareable {
				if diskDevice.Cache == "" {
					diskDevice.Cache = v1.CacheNone
				}
				if diskDevice.Cache != v1.CacheNone && diskDevice.Cache != v1.CacheDirectSync {
					return fmt.Errorf("a sharable disk requires cache mode none or directsync, got: %v", diskDevice.Cache)
				}
				disk.Shareable = &api.Shareable{}
			}
		}
	} else if diskDevice.LUN != nil {
		var unit int
		disk.Device = "lun"
		disk.Target.Bus = diskDevice.LUN.Bus
		disk.Target.Device, unit = makeDeviceName(diskDevice.Name, diskDevice.LUN.Bus, prefixMap)
		if diskDevice.LUN.Bus == "scsi" {
			assignDiskToSCSIController(disk, unit)
		}
		disk.ReadOnly = ToApiReadOnly(diskDevice.LUN.ReadOnly)
		if diskDevice.LUN.Reservation {
			setReservation(disk)
		}
	} else if diskDevice.CDRom != nil {
		disk.Device = "cdrom"
		disk.Target.Tray = string(diskDevice.CDRom.Tray)
		disk.Target.Bus = diskDevice.CDRom.Bus
		disk.Target.Device, _ = makeDeviceName(diskDevice.Name, diskDevice.CDRom.Bus, prefixMap)
		if diskDevice.CDRom.ReadOnly != nil {
			disk.ReadOnly = ToApiReadOnly(*diskDevice.CDRom.ReadOnly)
		} else {
			disk.ReadOnly = ToApiReadOnly(true)
		}
	}
	disk.Driver = &api.DiskDriver{
		Name:  "qemu",
		Cache: string(diskDevice.Cache),
		IO:    diskDevice.IO,
	}
	if diskDevice.Disk != nil || diskDevice.LUN != nil {
		if !slices.Contains(c.VolumesDiscardIgnore, diskDevice.Name) {
			disk.Driver.Discard = "unmap"
		}
		volumeStatus, ok := volumeStatusMap[diskDevice.Name]
		if ok && volumeStatus.PersistentVolumeClaimInfo != nil {
			disk.FilesystemOverhead = volumeStatus.PersistentVolumeClaimInfo.FilesystemOverhead
			disk.Capacity = storagetypes.GetDiskCapacity(volumeStatus.PersistentVolumeClaimInfo)
		}
	}
	if numQueues != nil && disk.Target.Bus == v1.DiskBusVirtio {
		disk.Driver.Queues = numQueues
	}
	disk.Alias = api.NewUserDefinedAlias(diskDevice.Name)
	if diskDevice.BootOrder != nil {
		disk.BootOrder = &api.BootOrder{Order: *diskDevice.BootOrder}
	}
	if (c.UseLaunchSecuritySEV || c.UseLaunchSecurityPV) && disk.Target.Bus == v1.DiskBusVirtio {
		disk.Driver.IOMMU = "on"
	}

	return nil
}

func setReservation(disk *api.Disk) {
	disk.Source.Reservations = &api.Reservations{
		Managed:   "no",
		Migration: "yes",
		SourceReservations: &api.SourceReservations{
			Type: "unix",
			Path: reservation.GetPrHelperSocketPath(),
			Mode: "client",
		},
	}
}

func SetErrorPolicy(diskDevice *v1.Disk, disk *api.Disk) error {
	if diskDevice.ErrorPolicy == nil {
		disk.Driver.ErrorPolicy = v1.DiskErrorPolicyStop
		return nil
	}
	switch *diskDevice.ErrorPolicy {
	case v1.DiskErrorPolicyStop, v1.DiskErrorPolicyIgnore, v1.DiskErrorPolicyReport, v1.DiskErrorPolicyEnospace:
		disk.Driver.ErrorPolicy = *diskDevice.ErrorPolicy
	default:
		return fmt.Errorf("error policy %s not recognized", *diskDevice.ErrorPolicy)
	}
	return nil
}

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

func (n *DeviceNamer) getExistingVolumeValue(key string) (string, bool) {
	if _, ok := n.existingNameMap[key]; ok {
		return n.existingNameMap[key], true
	}
	return "", false
}

func (n *DeviceNamer) getExistingTargetValue(key string) (string, bool) {
	if _, ok := n.usedDeviceMap[key]; ok {
		return n.usedDeviceMap[key], true
	}
	return "", false
}

func makeDeviceName(diskName string, bus v1.DiskBus, prefixMap map[string]DeviceNamer) (string, int) {
	prefix := getPrefixFromBus(bus)
	if _, ok := prefixMap[prefix]; !ok {
		// This should never happen since the prefix map is populated from all disks.
		prefixMap[prefix] = DeviceNamer{
			existingNameMap: make(map[string]string),
			usedDeviceMap:   make(map[string]string),
		}
	}
	deviceNamer := prefixMap[prefix]
	if name, ok := deviceNamer.getExistingVolumeValue(diskName); ok {
		for i := 0; i < 26*26*26; i++ {
			calculatedName := FormatDeviceName(prefix, i)
			if calculatedName == name {
				return name, i
			}
		}
		log.Log.Error("Unable to determine index of device")
		return name, 0
	}
	// Name not found yet, generate next new one.
	for i := 0; i < 26*26*26; i++ {
		name := FormatDeviceName(prefix, i)
		if _, ok := deviceNamer.getExistingTargetValue(name); !ok {
			deviceNamer.existingNameMap[diskName] = name
			deviceNamer.usedDeviceMap[name] = diskName
			return name, i
		}
	}
	return "", 0
}

// port of http://elixir.free-electrons.com/linux/v4.15/source/drivers/scsi/sd.c#L3211
func FormatDeviceName(prefix string, index int) string {
	base := int('z' - 'a' + 1)
	name := ""

	for index >= 0 {
		name = string(rune('a'+(index%base))) + name
		index = (index / base) - 1
	}
	return prefix + name
}

func getPrefixFromBus(bus v1.DiskBus) string {
	switch bus {
	case v1.DiskBusVirtio:
		return "vd"
	case v1.DiskBusSATA, v1.DiskBusSCSI, v1.DiskBusUSB:
		return "sd"
	default:
		log.Log.Errorf("Unrecognized bus '%s'", bus)
		return ""
	}
}

func NewDeviceNamer(volumeStatuses []v1.VolumeStatus, disks []v1.Disk) map[string]DeviceNamer {
	prefixMap := make(map[string]DeviceNamer)
	volumeTargetMap := make(map[string]string)
	for _, volumeStatus := range volumeStatuses {
		if volumeStatus.Target != "" {
			volumeTargetMap[volumeStatus.Name] = volumeStatus.Target
		}
	}

	for _, disk := range disks {
		var prefix string
		switch {
		case disk.Disk != nil:
			prefix = getPrefixFromBus(disk.Disk.Bus)
		case disk.LUN != nil:
			prefix = getPrefixFromBus(disk.LUN.Bus)
		case disk.CDRom != nil:
			prefix = getPrefixFromBus(disk.CDRom.Bus)
		default:
			continue
		}

		if _, ok := prefixMap[prefix]; !ok {
			prefixMap[prefix] = DeviceNamer{
				existingNameMap: make(map[string]string),
				usedDeviceMap:   make(map[string]string),
			}
		}
		namer := prefixMap[prefix]
		if _, ok := volumeTargetMap[disk.Name]; ok {
			namer.existingNameMap[disk.Name] = volumeTargetMap[disk.Name]
			namer.usedDeviceMap[volumeTargetMap[disk.Name]] = disk.Name
		}
	}
	return prefixMap
}

func AssignDiskIOThread(disk *v1.Disk, apiDisk *api.Disk, supplementalIOThreads *api.DiskIOThreads, autoThreads int, currentDedicatedThread, currentAutoThread uint) (uint, uint) {
	if apiDisk.Target.Bus == v1.DiskBusVirtio {
		if supplementalIOThreads != nil {
			apiDisk.Driver.IOThreads = supplementalIOThreads
		} else {
			if iothreads.HasDedicatedIOThread(*disk) {
				apiDisk.Driver.IOThread = pointer.P(currentDedicatedThread)
				currentDedicatedThread += 1
			} else {
				apiDisk.Driver.IOThread = pointer.P(currentAutoThread)
				// increment the threadId to be used next but wrap around at the thread limit
				// the odd math here is because thread ID's start at 1, not 0
				currentAutoThread = (currentAutoThread % uint(autoThreads)) + 1
			}
		}
	}
	return currentDedicatedThread, currentAutoThread
}
