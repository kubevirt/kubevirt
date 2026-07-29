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

package converter

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

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
	"kubevirt.io/client-go/precond"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/storage/reservation"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/iothreads"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/kvm"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/mshv"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/network"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/storage"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/virtio"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/disksource"
)

const (
	maxCustomBlockSizeS390x = 4096
)

type deviceNamer struct {
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

func Convert_v1_Disk_To_api_Disk(c *convertertypes.ConverterContext, diskDevice *v1.Disk, disk *api.Disk, prefixMap map[string]deviceNamer, numQueues *uint, volumeStatusMap map[string]v1.VolumeStatus) error {
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
		disk.ReadOnly = storage.ToApiReadOnly(diskDevice.Disk.ReadOnly)
		disk.Serial = diskDevice.Serial
		if diskDevice.Shareable != nil {
			if *diskDevice.Shareable {
				if diskDevice.Cache == "" {
					diskDevice.Cache = v1.CacheNone
				}
				if diskDevice.Cache != v1.CacheNone {
					return fmt.Errorf("a sharable disk requires cache = none got: %v", diskDevice.Cache)
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
		disk.ReadOnly = storage.ToApiReadOnly(diskDevice.LUN.ReadOnly)
		if diskDevice.LUN.Reservation {
			setReservation(disk)
		}
	} else if diskDevice.CDRom != nil {
		disk.Device = "cdrom"
		disk.Target.Tray = string(diskDevice.CDRom.Tray)
		disk.Target.Bus = diskDevice.CDRom.Bus
		disk.Target.Device, _ = makeDeviceName(diskDevice.Name, diskDevice.CDRom.Bus, prefixMap)
		if diskDevice.CDRom.ReadOnly != nil {
			disk.ReadOnly = storage.ToApiReadOnly(*diskDevice.CDRom.ReadOnly)
		} else {
			disk.ReadOnly = storage.ToApiReadOnly(true)
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

func setErrorPolicy(diskDevice *v1.Disk, disk *api.Disk) error {
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

type DirectIOChecker interface {
	CheckBlockDevice(path string) (bool, error)
	CheckFile(path string) (bool, error)
}

type directIOChecker struct{}

func NewDirectIOChecker() DirectIOChecker {
	return &directIOChecker{}
}

func (c *directIOChecker) CheckBlockDevice(path string) (bool, error) {
	return c.check(path, syscall.O_RDONLY)
}

func (c *directIOChecker) CheckFile(path string) (bool, error) {
	flags := syscall.O_RDONLY
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		// try to create the file and perform the check
		flags = flags | syscall.O_CREAT
		defer os.Remove(path)
	}
	return c.check(path, flags)
}

// based on https://gitlab.com/qemu-project/qemu/-/blob/master/util/osdep.c#L344
func (c *directIOChecker) check(path string, flags int) (bool, error) {
	// #nosec No risk for path injection as we only open the file, not read from it. The function leaks only whether the directory to `path` exists.
	f, err := os.OpenFile(path, flags|syscall.O_DIRECT, 0600)
	if err != nil {
		// EINVAL is returned if the filesystem does not support the O_DIRECT flag
		if err, ok := err.(*os.PathError); ok && err.Err == syscall.EINVAL {
			// #nosec No risk for path injection as we only open the file, not read from it. The function leaks only whether the directory to `path` exists.
			f, err := os.OpenFile(path, flags & ^syscall.O_DIRECT, 0600)
			if err == nil {
				defer util.CloseIOAndCheckErr(f, nil)
				return false, nil
			}
		}
		return false, err
	}
	defer util.CloseIOAndCheckErr(f, nil)
	return true, nil
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

func SetDriverCacheMode(disk *api.Disk, directIOChecker DirectIOChecker) error {
	if disk == nil {
		return fmt.Errorf("unable to set a driver cache mode, disk is nil")
	}

	t := disksource.Resolve(*disk)

	if t.BackendPath() == "" {
		if disk.Device == "cdrom" {
			return nil
		}
		return fmt.Errorf("unable to set a driver cache mode, disk has no backend path")
	}

	var err error
	supportDirectIO := true
	mode := v1.DriverCache(disk.Driver.Cache)

	if mode == "" || mode == v1.CacheNone {
		if t.BackendIsBlock() {
			supportDirectIO, err = directIOChecker.CheckBlockDevice(t.BackendPath())
		} else {
			supportDirectIO, err = directIOChecker.CheckFile(t.BackendPath())
		}
		if err != nil {
			log.Log.Reason(err).Errorf("Direct IO check failed for %s", t.BackendPath())
		} else if !supportDirectIO {
			log.Log.Infof("%s file system does not support direct I/O", t.BackendPath())
		}
		// when the disk is backed-up by another file, we need to also check if that
		// file sits on a file system that supports direct I/O
		if backingFile := disk.BackingStore; backingFile != nil {
			backingFilePath := backingFile.Source.File
			backFileDirectIOSupport, err := directIOChecker.CheckFile(backingFilePath)
			if err != nil {
				log.Log.Reason(err).Errorf("Direct IO check failed for %s", backingFilePath)
			} else if !backFileDirectIOSupport {
				log.Log.Infof("%s backing file system does not support direct I/O", backingFilePath)
			}
			supportDirectIO = supportDirectIO && backFileDirectIOSupport
		}
	}

	// if user set a cache mode = 'none' and fs does not support direct I/O then return an error
	if mode == v1.CacheNone && !supportDirectIO {
		return fmt.Errorf("Unable to use '%s' cache mode, file system where %s is stored does not support direct I/O", mode, t.BackendPath())
	}

	// if user did not set a cache mode and fs supports direct I/O then set cache = 'none'
	// else set cache = 'writethrough
	if mode == "" && supportDirectIO {
		mode = v1.CacheNone
	} else if mode == "" && !supportDirectIO {
		mode = v1.CacheWriteThrough
	}

	disk.Driver.Cache = string(mode)
	log.Log.Infof("Driver cache mode for %s set to %s", t.BackendPath(), mode)

	return nil
}

func IsPreAllocated(path string) bool {
	diskInf, err := disk.GetDiskInfo(path)
	if err != nil {
		return false
	}
	// ActualSize can be a little larger then VirtualSize for qcow2
	return diskInf.VirtualSize <= diskInf.ActualSize
}

// Set optimal io mode automatically
func SetOptimalIOMode(disk *api.Disk, isPreAllocated func(path string) bool) {
	if disk == nil {
		return
	}

	ds := disksource.Resolve(*disk)

	// If the user explicitly set the io mode do nothing
	if disk.Driver.IO != "" {
		return
	}

	if ds.BackendPath() == "" {
		return
	}

	// O_DIRECT is needed for io="native"
	if v1.DriverCache(disk.Driver.Cache) == v1.CacheNone {
		// set native for block device or pre-allocateed image file
		if ds.BackendIsBlock() || isPreAllocated(ds.BackendPath()) {
			disk.Driver.IO = v1.IONative
		}
	}
	// For now we don't explicitly set io=threads even for sparse files as it's
	// not clear it's better for all use-cases
	if disk.Driver.IO != "" {
		log.Log.Infof("Driver IO mode for %s set to %s", ds.BackendPath(), disk.Driver.IO)
	}
}

func (n *deviceNamer) getExistingVolumeValue(key string) (string, bool) {
	if _, ok := n.existingNameMap[key]; ok {
		return n.existingNameMap[key], true
	}
	return "", false
}

func (n *deviceNamer) getExistingTargetValue(key string) (string, bool) {
	if _, ok := n.usedDeviceMap[key]; ok {
		return n.usedDeviceMap[key], true
	}
	return "", false
}

func makeDeviceName(diskName string, bus v1.DiskBus, prefixMap map[string]deviceNamer) (string, int) {
	prefix := getPrefixFromBus(bus)
	if _, ok := prefixMap[prefix]; !ok {
		// This should never happen since the prefix map is populated from all disks.
		prefixMap[prefix] = deviceNamer{
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

func assignDiskIOThread(disk *v1.Disk, apiDisk *api.Disk, supplementalIOThreads *api.DiskIOThreads, autoThreads int, currentDedicatedThread, currentAutoThread uint) (uint, uint) {
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

func Convert_v1_VirtualMachineInstance_To_api_Domain(vmi *v1.VirtualMachineInstance, domain *api.Domain, c *convertertypes.ConverterContext) (err error) {

	precond.MustNotBeNil(vmi)
	precond.MustNotBeNil(domain)
	precond.MustNotBeNil(c)

	hasIOThreads := iothreads.HasIOThreads(vmi)
	var ioThreadCount, autoThreads int
	if hasIOThreads {
		ioThreadCount, autoThreads = iothreads.GetIOThreadsCountType(vmi)
	}

	architecture := c.Architecture.GetArchitecture()
	virtioModel := virtio.InterpretTransitionalModelType(
		vmi.Spec.Domain.Devices.UseVirtioTransitional,
		architecture,
	)
	scsiControllerModel := c.Architecture.SCSIControllerModel(virtioModel)

	configurators := []convertertypes.Configurator{
		metadata.DomainConfigurator{},
		network.NewDomainConfigurator(
			network.WithDomainAttachmentByInterfaceName(c.DomainAttachmentByInterfaceName),
			network.WithUseLaunchSecuritySEV(c.UseLaunchSecuritySEV),
			network.WithUseLaunchSecurityPV(c.UseLaunchSecurityPV),
			network.WithROMTuningSupport(c.Architecture.IsROMTuningSupported()),
			network.WithVirtioModel(virtioModel),
		),
		compute.TPMDomainConfigurator{},
		compute.VSOCKDomainConfigurator{ProcPath: c.VSOCKProcPath},
		compute.NewLaunchSecurityDomainConfigurator(architecture),
		compute.ChannelsDomainConfigurator{},
		compute.ClockDomainConfigurator{},
		compute.NewRNGDomainConfigurator(
			compute.RNGWithUseLaunchSecuritySEV(c.UseLaunchSecuritySEV),
			compute.RNGWithUseLaunchSecurityPV(c.UseLaunchSecurityPV),
			compute.RNGWithVirtioModel(virtioModel),
		),
		compute.NewInputDeviceDomainConfigurator(architecture),
		compute.NewBalloonDomainConfigurator(
			compute.BalloonWithUseLaunchSecuritySEV(c.UseLaunchSecuritySEV),
			compute.BalloonWithUseLaunchSecurityPV(c.UseLaunchSecurityPV),
			compute.BalloonWithFreePageReporting(c.FreePageReporting),
			compute.BalloonWithMemBalloonStatsPeriod(c.MemBalloonStatsPeriod),
			compute.BalloonWithVirtioModel(virtioModel),
		),
		compute.NewGraphicsDomainConfigurator(architecture, c.BochsForEFIGuests, c.AllowCrossArchEmulation),
		compute.SoundDomainConfigurator{},
		compute.NewHostDeviceDomainConfigurator(
			c.GenericHostDevices,
			c.GPUHostDevices,
			c.SRIOVDevices,
		),
		compute.NewWatchdogDomainConfigurator(architecture),
		compute.NewConsoleDomainConfigurator(c.SerialConsoleLog),
		compute.PanicDevicesDomainConfigurator{},
		compute.NewHypervisorFeaturesDomainConfigurator(c.Architecture.HasVMPort(), c.UseLaunchSecurityTDX),
		compute.NewSysInfoDomainConfigurator(convertCmdv1SMBIOSToComputeSMBIOS(c.SMBios)),
		compute.NewOSDomainConfigurator(c.Architecture.IsSMBiosNeeded(), convertEFIConfiguration(c.EFIConfiguration)),
		storage.NewVirtiofsConfigurator(),
		compute.UsbRedirectDeviceDomainConfigurator{},
		compute.NewControllersDomainConfigurator(
			compute.ControllersWithUSBNeeded(c.Architecture.IsUSBNeeded(vmi)),
			compute.ControllersWithSCSIModel(scsiControllerModel),
			compute.ControllersWithSCSIIOThreads(uint(autoThreads)),
			compute.ControllersWithUseLaunchSecuritySEV(c.UseLaunchSecuritySEV),
			compute.ControllersWithUseLaunchSecurityPV(c.UseLaunchSecurityPV),
			compute.ControllersWithSupportPCIHole64Disabling(c.Architecture.SupportPCIHole64Disabling()),
			compute.ControllersWithVirtioSerialModel(virtioModel),
		),
		compute.NewQemuCmdDomainConfigurator(c.Architecture.ShouldVerboseLogsBeEnabled()),
		compute.NewCPUDomainConfigurator(
			compute.CPUWithHotplugSupported(c.Architecture.SupportCPUHotplug()),
			compute.CPUWithMPXCPUValidation(c.Architecture.RequiresMPXCPUValidation()),
			compute.CPUWithCrossArchEmulation(c.AllowCrossArchEmulation),
			compute.CPUWithMemfdSupported(c.Architecture.IsMemfdSupported()),
		),
		compute.NewIOThreadsDomainConfigurator(uint(ioThreadCount)),
		compute.MemoryConfigurator{},
		compute.NewMemoryBackingConfigurator(c.Architecture.IsMemfdSupported()),
		compute.RebootPolicyDomainConfigurator{},
		compute.NewIOMMUFDConfigurator(c.IOMMUFDEnabled),
	}

	switch c.HypervisorName {
	case v1.HyperVDirectHypervisorName:
		configurators = append(configurators, mshv.NewMshvDomainConfigurator(c.AllowEmulation, c.HypervisorDeviceAvailable))
	default:
		if c.AllowCrossArchEmulation {
			configurators = append(configurators, kvm.NewKvmDomainConfiguratorWithCrossArch(c.AllowEmulation, c.HypervisorDeviceAvailable, c.AllowCrossArchEmulation, c.HostArchitecture))
		} else {
			configurators = append(configurators, kvm.NewKvmDomainConfigurator(c.AllowEmulation, c.HypervisorDeviceAvailable))
		}
	}

	builder := convertertypes.NewDomainBuilder(configurators...)
	if err := builder.Build(vmi, domain); err != nil {
		return err
	}

	volumeIndices := map[string]int{}
	volumes := map[string]*v1.Volume{}
	for i, volume := range vmi.Spec.Volumes {
		volumes[volume.Name] = volume.DeepCopy()
		volumeIndices[volume.Name] = i
	}

	var numBlkQueues *uint
	virtioBlkMQRequested := (vmi.Spec.Domain.Devices.BlockMultiQueue != nil) && (*vmi.Spec.Domain.Devices.BlockMultiQueue)
	cpuTopology := vcpu.GetCPUTopology(vmi)
	cpuCount := vcpu.CalculateRequestedVCPUs(cpuTopology)
	vcpus := uint(cpuCount)
	if vcpus == 0 {
		vcpus = uint(1)
	}

	if virtioBlkMQRequested {
		numBlkQueues = &vcpus
	}

	volumeStatusMap := make(map[string]v1.VolumeStatus)
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		volumeStatusMap[volumeStatus.Name] = volumeStatus
	}

	prefixMap := newDeviceNamer(vmi.Status.VolumeStatus, vmi.Spec.Domain.Devices.Disks)
	currentAutoThread := uint(1)
	currentDedicatedThread := uint(autoThreads + 1)
	supplementalIOThreads := iothreads.BuildSupplementalPoolIOThreads(vmi)
	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		newDisk := api.Disk{}
		emptyCDRom := false

		err := Convert_v1_Disk_To_api_Disk(c, &disk, &newDisk, prefixMap, numBlkQueues, volumeStatusMap)
		if err != nil {
			return err
		}
		volume := volumes[disk.Name]
		if volume == nil {
			if disk.CDRom == nil {
				return fmt.Errorf("no matching volume with name %s found", disk.Name)
			}
			emptyCDRom = true
		}

		hpStatus, hpOk := c.HotplugVolumes[disk.Name]
		switch {
		case emptyCDRom:
			err = storage.Convert_v1_Missing_Volume_To_api_Disk(&newDisk)
		case hpOk:
			err = storage.Convert_v1_Hotplug_Volume_To_api_Disk(volume, &newDisk, c)
		default:
			err = storage.Convert_v1_Volume_To_api_Disk(volume, &newDisk, c, volumeIndices[disk.Name])
		}

		if err != nil {
			return err
		}

		if err := Convert_v1_BlockSize_To_api_BlockIO(&disk, &newDisk, c.Architecture.GetArchitecture()); err != nil {
			return err
		}

		_, isPermVolume := c.PermanentVolumes[disk.Name]
		// if len(c.PermanentVolumes) == 0, it means the vmi is not ready yet, add all disks
		permReady := isPermVolume || len(c.PermanentVolumes) == 0
		hotplugReady := hpOk && (hpStatus.Phase == v1.HotplugVolumeMounted || hpStatus.Phase == v1.VolumeReady)

		if permReady || hotplugReady || emptyCDRom {
			domain.Spec.Devices.Disks = append(domain.Spec.Devices.Disks, newDisk)
		}
		if err := setErrorPolicy(&disk, &newDisk); err != nil {
			return err
		}
		if hasIOThreads {
			currentDedicatedThread, currentAutoThread = assignDiskIOThread(&disk, &newDisk, supplementalIOThreads, autoThreads, currentDedicatedThread, currentAutoThread)
		}
	}

	if vmi.Spec.Domain.CPU != nil && vmi.IsCPUDedicated() {
		// Adjust guest vcpu config. Currently will handle vCPUs to pCPUs pinning
		if err := vcpu.AdjustDomainForTopologyAndCPUSet(domain, vmi, c.Topology, c.CPUSet); err != nil {
			return err
		}

		if graceIOVirtualizationRequested(c) {
			return configureGraceIOVirtualization(&domain.Spec, c.GraceHostDeviceAliases, c.IOMMUFDEnabled)
		}

		if c.PCINUMAAwareTopologyEnabled {
			if c.Architecture.SupportPCIePlacement() {
				// Strict PCI placement is tied to the VMI API request. At this
				// point the request has already been converted into domain NUMA
				// state, but deriving strictness from the XML shape would make
				// other NUMATune users strict by accident.
				strictPCIPlacement := vmi.Spec.Domain.CPU.NUMA != nil &&
					vmi.Spec.Domain.CPU.NUMA.GuestMappingPassthrough != nil
				var opts []PCIPlacementOption
				if strictPCIPlacement {
					opts = append(opts, WithStrictPCINUMAPlacement())
				}
				if err := PlacePCIDevicesWithNUMAAlignment(&domain.Spec, opts...); err != nil {
					if strictPCIPlacement {
						return fmt.Errorf("failed to process strict PCIe NUMA-aware topology: %w", err)
					}
					log.Log.Reason(err).Warningf("Failed to process PCIe NUMA-aware topology, falling back to default placement")
				}
			} else {
				log.Log.Infof("Skipping PCIe NUMA alignment: architecture %s does not support PCIe placement", c.Architecture.GetArchitecture())
			}
		}
	}

	if val := vmi.Annotations[v1.PlacePCIDevicesOnRootComplex]; val == "true" {
		if c.Architecture.SupportPCIePlacement() {
			if err := PlacePCIDevicesOnRootComplex(&domain.Spec); err != nil {
				return err
			}
		} else {
			log.Log.Infof("Skipping PCIe root complex placement: architecture %s does not support PCIe placement", c.Architecture.GetArchitecture())
		}
	}

	return nil
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

func newDeviceNamer(volumeStatuses []v1.VolumeStatus, disks []v1.Disk) map[string]deviceNamer {
	prefixMap := make(map[string]deviceNamer)
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
			prefixMap[prefix] = deviceNamer{
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

func GetVolumeNameByDisk(disk api.Disk) string {
	return disk.Alias.GetName()
}

// GetVolumeNameByTarget returns the volume name associated to the device target in the domain (e.g vda)
func GetVolumeNameByTarget(domain *api.Domain, target string) string {
	for _, d := range domain.Spec.Devices.Disks {
		if d.Target.Device == target {
			return GetVolumeNameByDisk(d)
		}
	}
	return ""
}

func GracePeriodSeconds(vmi *v1.VirtualMachineInstance) int64 {
	gracePeriodSeconds := v1.DefaultGracePeriodSeconds
	if vmi.Spec.TerminationGracePeriodSeconds != nil {
		gracePeriodSeconds = *vmi.Spec.TerminationGracePeriodSeconds
	}
	return gracePeriodSeconds
}

func convertCmdv1SMBIOSToComputeSMBIOS(input *cmdv1.SMBios) *compute.SMBIOS {
	if input == nil {
		return nil
	}

	return &compute.SMBIOS{
		Manufacturer: input.Manufacturer,
		Product:      input.Product,
		Version:      input.Version,
		SKU:          input.Sku,
		Family:       input.Family,
	}
}

func convertEFIConfiguration(input *convertertypes.EFIConfiguration) *compute.EFIConfiguration {
	if input == nil {
		return nil
	}

	return &compute.EFIConfiguration{
		EFICode:                   input.EFICode,
		EFIVars:                   input.EFIVars,
		SecureLoader:              input.SecureLoader,
		UsesFirmwareAutoSelection: input.UsesFirmwareAutoSelection,
	}
}
