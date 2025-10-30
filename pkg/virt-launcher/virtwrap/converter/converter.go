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
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/config"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/downwardmetrics"
	"kubevirt.io/kubevirt/pkg/emptydisk"
	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/ignition"
	"kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/storage/reservation"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/tpm"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/arch"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/virtio"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
)

const (
	deviceTypeNotCompatibleFmt = "device %s is of type lun. Not compatible with a file based disk"
	defaultIOThread            = uint(1)
	bootMenuTimeoutMS          = uint(10000)
	QEMUSeaBiosDebugPipe       = "/var/run/kubevirt-private/QEMUSeaBiosDebugPipe"
)

type deviceNamer struct {
	existingNameMap map[string]string
	usedDeviceMap   map[string]string
}

type EFIConfiguration struct {
	EFICode      string
	EFIVars      string
	SecureLoader bool
}

type ConverterContext struct {
	Architecture                    arch.Converter
	AllowEmulation                  bool
	Secrets                         map[string]*k8sv1.Secret
	VirtualMachine                  *v1.VirtualMachineInstance
	CPUSet                          []int
	IsBlockPVC                      map[string]bool
	IsBlockDV                       map[string]bool
	ApplyCBT                        map[string]string
	HotplugVolumes                  map[string]v1.VolumeStatus
	PermanentVolumes                map[string]v1.VolumeStatus
	MigratedVolumes                 map[string]string
	DisksInfo                       map[string]*disk.DiskInfo
	SMBios                          *cmdv1.SMBios
	SRIOVDevices                    []api.HostDevice
	GenericHostDevices              []api.HostDevice
	GPUHostDevices                  []api.HostDevice
	EFIConfiguration                *EFIConfiguration
	MemBalloonStatsPeriod           uint
	UseVirtioTransitional           bool
	EphemeraldiskCreator            ephemeraldisk.EphemeralDiskCreatorInterface
	VolumesDiscardIgnore            []string
	Topology                        *cmdv1.Topology
	ExpandDisksEnabled              bool
	UseLaunchSecuritySEV            bool // For AMD SEV/ES/SNP
	UseLaunchSecurityTDX            bool // For Intel TDX
	UseLaunchSecurityPV             bool // For IBM SE(s390-pv)
	FreePageReporting               bool
	BochsForEFIGuests               bool
	SerialConsoleLog                bool
	DomainAttachmentByInterfaceName map[string]string
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

func Convert_v1_Disk_To_api_Disk(c *ConverterContext, diskDevice *v1.Disk, disk *api.Disk, prefixMap map[string]deviceNamer, numQueues *uint, volumeStatusMap map[string]v1.VolumeStatus) error {
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
		disk.ReadOnly = toApiReadOnly(diskDevice.Disk.ReadOnly)
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
		disk.ReadOnly = toApiReadOnly(diskDevice.LUN.ReadOnly)
		if diskDevice.LUN.Reservation {
			setReservation(disk)
		}
	} else if diskDevice.CDRom != nil {
		disk.Device = "cdrom"
		disk.Target.Tray = string(diskDevice.CDRom.Tray)
		disk.Target.Bus = diskDevice.CDRom.Bus
		disk.Target.Device, _ = makeDeviceName(diskDevice.Name, diskDevice.CDRom.Bus, prefixMap)
		if diskDevice.CDRom.ReadOnly != nil {
			disk.ReadOnly = toApiReadOnly(*diskDevice.CDRom.ReadOnly)
		} else {
			disk.ReadOnly = toApiReadOnly(true)
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
			disk.ExpandDisksEnabled = c.ExpandDisksEnabled
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
		Managed: "no",
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

func Convert_v1_BlockSize_To_api_BlockIO(source *v1.Disk, disk *api.Disk) error {
	if source.BlockSize == nil {
		return nil
	}

	if blockSize := source.BlockSize.Custom; blockSize != nil {
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
	if disk.Source.Dev != "" {
		return getOptimalBlockIOForDevice(disk.Source.Dev)
	} else if disk.Source.File != "" {
		return getOptimalBlockIOForFile(disk.Source.File)
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
	var path string
	var err error
	supportDirectIO := true
	mode := v1.DriverCache(disk.Driver.Cache)
	isBlockDev := false

	switch {
	case disk.Source.File != "":
		path = disk.Source.File
	case disk.Source.Dev != "":
		path = disk.Source.Dev
	// handle empty cdrom
	case disk.Device == "cdrom":
		return nil
	default:
		return fmt.Errorf("unable to set a driver cache mode, disk is neither a block device nor a file")
	}

	if mode == "" || mode == v1.CacheNone {
		if isBlockDev {
			supportDirectIO, err = directIOChecker.CheckBlockDevice(path)
		} else {
			supportDirectIO, err = directIOChecker.CheckFile(path)
		}
		if err != nil {
			log.Log.Reason(err).Errorf("Direct IO check failed for %s", path)
		} else if !supportDirectIO {
			log.Log.Infof("%s file system does not support direct I/O", path)
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
		return fmt.Errorf("Unable to use '%s' cache mode, file system where %s is stored does not support direct I/O", mode, path)
	}

	// if user did not set a cache mode and fs supports direct I/O then set cache = 'none'
	// else set cache = 'writethrough
	if mode == "" && supportDirectIO {
		mode = v1.CacheNone
	} else if mode == "" && !supportDirectIO {
		mode = v1.CacheWriteThrough
	}

	disk.Driver.Cache = string(mode)
	log.Log.Infof("Driver cache mode for %s set to %s", path, mode)

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
func SetOptimalIOMode(disk *api.Disk) error {
	var path string

	// If the user explicitly set the io mode do nothing
	if v1.DriverIO(disk.Driver.IO) != "" {
		return nil
	}

	if disk.Source.File != "" {
		path = disk.Source.File
	} else if disk.Source.Dev != "" {
		path = disk.Source.Dev
	} else {
		return nil
	}

	// O_DIRECT is needed for io="native"
	if v1.DriverCache(disk.Driver.Cache) == v1.CacheNone {
		// set native for block device or pre-allocateed image file
		if (disk.Source.Dev != "") || IsPreAllocated(disk.Source.File) {
			disk.Driver.IO = v1.IONative
		}
	}
	// For now we don't explicitly set io=threads even for sparse files as it's
	// not clear it's better for all use-cases
	if v1.DriverIO(disk.Driver.IO) != "" {
		log.Log.Infof("Driver IO mode for %s set to %s", path, disk.Driver.IO)
	}
	return nil
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

func toApiReadOnly(src bool) *api.ReadOnly {
	if src {
		return &api.ReadOnly{}
	}
	return nil
}

// Add_Agent_To_api_Channel creates the channel for guest agent communication
func Add_Agent_To_api_Channel() (channel api.Channel) {
	channel.Type = "unix"
	// let libvirt decide which path to use
	channel.Source = nil
	channel.Target = &api.ChannelTarget{
		Name: "org.qemu.guest_agent.0",
		Type: v1.VirtIO,
	}

	return
}

func Convert_v1_Volume_To_api_Disk(source *v1.Volume, disk *api.Disk, c *ConverterContext, diskIndex int) error {

	if source.ContainerDisk != nil {
		return Convert_v1_ContainerDiskSource_To_api_Disk(source.Name, source.ContainerDisk, disk, c, diskIndex)
	}

	if source.CloudInitNoCloud != nil || source.CloudInitConfigDrive != nil {
		return Convert_v1_CloudInitSource_To_api_Disk(source.VolumeSource, disk, c)
	}

	if source.Sysprep != nil {
		return Convert_v1_SysprepSource_To_api_Disk(source.Name, disk)
	}

	if source.HostDisk != nil {
		return Convert_v1_HostDisk_To_api_Disk(source.Name, source.HostDisk.Path, disk, c)
	}

	if source.PersistentVolumeClaim != nil {
		return Convert_v1_PersistentVolumeClaim_To_api_Disk(source.Name, disk, c)
	}

	if source.DataVolume != nil {
		return Convert_v1_DataVolume_To_api_Disk(source.Name, disk, c)
	}

	if source.Ephemeral != nil {
		return Convert_v1_EphemeralVolumeSource_To_api_Disk(source.Name, disk, c)
	}
	if source.EmptyDisk != nil {
		return Convert_v1_EmptyDiskSource_To_api_Disk(source.Name, source.EmptyDisk, disk)
	}
	if source.ConfigMap != nil {
		return Convert_v1_Config_To_api_Disk(source.Name, disk, config.ConfigMap)
	}
	if source.Secret != nil {
		return Convert_v1_Config_To_api_Disk(source.Name, disk, config.Secret)
	}
	if source.DownwardAPI != nil {
		return Convert_v1_Config_To_api_Disk(source.Name, disk, config.DownwardAPI)
	}
	if source.ServiceAccount != nil {
		return Convert_v1_Config_To_api_Disk(source.Name, disk, config.ServiceAccount)
	}
	if source.DownwardMetrics != nil {
		return Convert_v1_DownwardMetricSource_To_api_Disk(disk, c)
	}

	return fmt.Errorf("disk %s references an unsupported source", disk.Alias.GetName())
}

// Convert_v1_Hotplug_Volume_To_api_Disk convers a hotplug volume to an api disk
func Convert_v1_Hotplug_Volume_To_api_Disk(source *v1.Volume, disk *api.Disk, c *ConverterContext) error {
	// This is here because virt-handler before passing the VMI here replaces all PVCs with host disks in
	// hostdisk.ReplacePVCByHostDisk not quite sure why, but it broken hot plugging PVCs
	if source.HostDisk != nil {
		return Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk(source.Name, disk, c)
	}

	if source.PersistentVolumeClaim != nil {
		return Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk(source.Name, disk, c)
	}

	if source.DataVolume != nil {
		return Convert_v1_Hotplug_DataVolume_To_api_Disk(source.Name, disk, c)
	}
	return fmt.Errorf("hotplug disk %s references an unsupported source", disk.Alias.GetName())
}

// Convert_v1_Missing_Volume_To_api_Disk sets defaults when no volume for disk (cdrom, floppy, etc) is provided
func Convert_v1_Missing_Volume_To_api_Disk(disk *api.Disk) error {
	disk.Type = "block"
	disk.Driver.Type = "raw"
	disk.Driver.Discard = "unmap"
	return nil
}

func Convert_v1_Config_To_api_Disk(volumeName string, disk *api.Disk, configType config.Type) error {
	disk.Type = "file"
	setDiskDriver(disk, "raw", false)
	switch configType {
	case config.ConfigMap:
		disk.Source.File = config.GetConfigMapDiskPath(volumeName)
	case config.Secret:
		disk.Source.File = config.GetSecretDiskPath(volumeName)
	case config.DownwardAPI:
		disk.Source.File = config.GetDownwardAPIDiskPath(volumeName)
	case config.ServiceAccount:
		disk.Source.File = config.GetServiceAccountDiskPath()
	default:
		return fmt.Errorf("Cannot convert config '%s' to disk, unrecognized type", configType)
	}

	return nil
}

func GetFilesystemVolumePath(volumeName string) string {
	return filepath.Join(string(filepath.Separator), "var", "run", "kubevirt-private", "vmi-disks", volumeName, "disk.img")
}

// GetHotplugFilesystemVolumePath returns the path and file name of a hotplug disk image
func GetHotplugFilesystemVolumePath(volumeName string) string {
	return filepath.Join(string(filepath.Separator), "var", "run", "kubevirt", "hotplug-disks", fmt.Sprintf("%s.img", volumeName))
}

func GetBlockDeviceVolumePath(volumeName string) string {
	return filepath.Join(string(filepath.Separator), "dev", volumeName)
}

// GetHotplugBlockDeviceVolumePath returns the path and name of a hotplugged block device
func GetHotplugBlockDeviceVolumePath(volumeName string) string {
	return filepath.Join(string(filepath.Separator), "var", "run", "kubevirt", "hotplug-disks", volumeName)
}

func setDiskDriver(disk *api.Disk, driverType string, discard bool) {
	disk.Driver.Type = driverType
	disk.Driver.ErrorPolicy = v1.DiskErrorPolicyStop
	if discard {
		disk.Driver.Discard = "unmap"
	}
}

func convertVolumeWithCBT(volumeName, cbtPath string, isBlock bool, disk *api.Disk, volumesDiscardIgnore []string) error {
	setDiskDriver(disk, "qcow2", !slices.Contains(volumesDiscardIgnore, volumeName))

	disk.Type = "file"
	disk.Source.File = cbtPath
	disk.Source.DataStore = &api.DataStore{
		Format: &api.DataStoreFormat{
			Type: "raw",
		},
	}

	if isBlock {
		disk.Source.Name = volumeName
		disk.Source.DataStore.Type = "block"
		disk.Source.DataStore.Source = &api.DiskSource{
			Dev: GetBlockDeviceVolumePath(volumeName),
		}
	} else {
		disk.Source.DataStore.Type = "file"
		disk.Source.DataStore.Source = &api.DiskSource{
			File: GetFilesystemVolumePath(volumeName),
		}
	}

	return nil
}

func convertVolumeWithoutCBT(volumeName string, isBlock bool, disk *api.Disk, volumesDiscardIgnore []string) error {
	setDiskDriver(disk, "raw", !slices.Contains(volumesDiscardIgnore, volumeName))

	if isBlock {
		disk.Type = "block"
		disk.Source.Name = volumeName
		disk.Source.Dev = GetBlockDeviceVolumePath(volumeName)
	} else {
		disk.Type = "file"
		disk.Source.File = GetFilesystemVolumePath(volumeName)
	}
	return nil
}

func ConvertVolumeSourceToDisk(volumeName, cbtPath string, isBlock bool, disk *api.Disk, volumesDiscardIgnore []string) error {
	if cbtPath != "" {
		return convertVolumeWithCBT(volumeName, cbtPath, isBlock, disk, volumesDiscardIgnore)
	}
	return convertVolumeWithoutCBT(volumeName, isBlock, disk, volumesDiscardIgnore)
}

func Convert_v1_PersistentVolumeClaim_To_api_Disk(name string, disk *api.Disk, c *ConverterContext) error {
	return ConvertVolumeSourceToDisk(name, c.ApplyCBT[name], c.IsBlockPVC[name], disk, c.VolumesDiscardIgnore)
}

// Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk converts a Hotplugged PVC to an api disk
func Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk(name string, disk *api.Disk, c *ConverterContext) error {
	if c.IsBlockPVC[name] {
		return Convert_v1_Hotplug_BlockVolumeSource_To_api_Disk(name, disk, c.VolumesDiscardIgnore)
	}
	return Convert_v1_Hotplug_FilesystemVolumeSource_To_api_Disk(name, disk, c.VolumesDiscardIgnore)
}

func Convert_v1_DataVolume_To_api_Disk(name string, disk *api.Disk, c *ConverterContext) error {
	return ConvertVolumeSourceToDisk(name, c.ApplyCBT[name], c.IsBlockDV[name], disk, c.VolumesDiscardIgnore)
}

// Convert_v1_Hotplug_DataVolume_To_api_Disk converts a Hotplugged DataVolume to an api disk
func Convert_v1_Hotplug_DataVolume_To_api_Disk(name string, disk *api.Disk, c *ConverterContext) error {
	if c.IsBlockDV[name] {
		return Convert_v1_Hotplug_BlockVolumeSource_To_api_Disk(name, disk, c.VolumesDiscardIgnore)
	}
	return Convert_v1_Hotplug_FilesystemVolumeSource_To_api_Disk(name, disk, c.VolumesDiscardIgnore)
}

// Convert_v1_FilesystemVolumeSource_To_api_Disk takes a FS source and builds the domain Disk representation
func Convert_v1_FilesystemVolumeSource_To_api_Disk(volumeName string, disk *api.Disk, volumesDiscardIgnore []string) error {
	disk.Type = "file"
	setDiskDriver(disk, "raw", false)
	disk.Source.File = GetFilesystemVolumePath(volumeName)
	if !slices.Contains(volumesDiscardIgnore, volumeName) {
		disk.Driver.Discard = "unmap"
	}
	return nil
}

// Convert_v1_Hotplug_FilesystemVolumeSource_To_api_Disk takes a FS source and builds the KVM Disk representation
func Convert_v1_Hotplug_FilesystemVolumeSource_To_api_Disk(volumeName string, disk *api.Disk, volumesDiscardIgnore []string) error {
	disk.Type = "file"
	setDiskDriver(disk, "raw", !slices.Contains(volumesDiscardIgnore, volumeName))
	disk.Source.File = GetHotplugFilesystemVolumePath(volumeName)
	return nil
}

func Convert_v1_BlockVolumeSource_To_api_Disk(volumeName string, disk *api.Disk, volumesDiscardIgnore []string) error {
	disk.Type = "block"
	setDiskDriver(disk, "raw", !slices.Contains(volumesDiscardIgnore, volumeName))
	disk.Source.Name = volumeName
	disk.Source.Dev = GetBlockDeviceVolumePath(volumeName)
	return nil
}

// Convert_v1_Hotplug_BlockVolumeSource_To_api_Disk takes a block device source and builds the domain Disk representation
func Convert_v1_Hotplug_BlockVolumeSource_To_api_Disk(volumeName string, disk *api.Disk, volumesDiscardIgnore []string) error {
	disk.Type = "block"
	setDiskDriver(disk, "raw", !slices.Contains(volumesDiscardIgnore, volumeName))
	disk.Source.Dev = GetHotplugBlockDeviceVolumePath(volumeName)
	return nil
}

func Convert_v1_HostDisk_To_api_Disk(volumeName string, path string, disk *api.Disk, c *ConverterContext) error {
	disk.Type = "file"
	if cbtPath, ok := c.ApplyCBT[volumeName]; ok {
		disk.Driver.Type = "qcow2"
		disk.Source.File = cbtPath
		disk.Source.DataStore = &api.DataStore{
			Type: "file",
			Format: &api.DataStoreFormat{
				Type: "raw",
			},
			Source: &api.DiskSource{
				File: hostdisk.GetMountedHostDiskPath(volumeName, path),
			},
		}
	} else {
		disk.Driver.Type = "raw"
		disk.Source.File = hostdisk.GetMountedHostDiskPath(volumeName, path)
	}
	disk.Driver.ErrorPolicy = v1.DiskErrorPolicyStop
	return nil
}

func Convert_v1_SysprepSource_To_api_Disk(volumeName string, disk *api.Disk) error {
	if disk.Type == "lun" {
		return fmt.Errorf(deviceTypeNotCompatibleFmt, disk.Alias.GetName())
	}

	disk.Source.File = config.GetSysprepDiskPath(volumeName)
	disk.Type = "file"
	disk.Driver.Type = "raw"
	return nil
}

func Convert_v1_CloudInitSource_To_api_Disk(source v1.VolumeSource, disk *api.Disk, c *ConverterContext) error {
	if disk.Type == "lun" {
		return fmt.Errorf(deviceTypeNotCompatibleFmt, disk.Alias.GetName())
	}

	var dataSource cloudinit.DataSourceType
	if source.CloudInitNoCloud != nil {
		dataSource = cloudinit.DataSourceNoCloud
	} else if source.CloudInitConfigDrive != nil {
		dataSource = cloudinit.DataSourceConfigDrive
	} else {
		return fmt.Errorf("Only nocloud and configdrive are valid cloud-init volumes")
	}

	disk.Source.File = cloudinit.GetIsoFilePath(dataSource, c.VirtualMachine.Name, c.VirtualMachine.Namespace)
	disk.Type = "file"
	setDiskDriver(disk, "raw", false)
	return nil
}

func Convert_v1_DownwardMetricSource_To_api_Disk(disk *api.Disk, c *ConverterContext) error {
	disk.Type = "file"
	disk.ReadOnly = toApiReadOnly(true)
	disk.Driver = &api.DiskDriver{
		Type: "raw",
		Name: "qemu",
	}
	// This disk always needs `virtio`. Validation ensures that bus is unset or is already virtio
	disk.Model = virtio.InterpretTransitionalModelType(&c.UseVirtioTransitional, c.Architecture.GetArchitecture())
	disk.Source = api.DiskSource{
		File: config.DownwardMetricDisk,
	}
	return nil
}

func Convert_v1_EmptyDiskSource_To_api_Disk(volumeName string, _ *v1.EmptyDiskSource, disk *api.Disk) error {
	if disk.Type == "lun" {
		return fmt.Errorf(deviceTypeNotCompatibleFmt, disk.Alias.GetName())
	}

	disk.Type = "file"
	disk.Source.File = emptydisk.NewEmptyDiskCreator().FilePathForVolumeName(volumeName)
	setDiskDriver(disk, "qcow2", true)

	return nil
}

func Convert_v1_ContainerDiskSource_To_api_Disk(volumeName string, _ *v1.ContainerDiskSource, disk *api.Disk, c *ConverterContext, diskIndex int) error {
	if disk.Type == "lun" {
		return fmt.Errorf(deviceTypeNotCompatibleFmt, disk.Alias.GetName())
	}
	disk.Type = "file"
	setDiskDriver(disk, "qcow2", true)
	disk.Source.File = c.EphemeraldiskCreator.GetFilePath(volumeName)
	disk.BackingStore = &api.BackingStore{
		Format: &api.BackingStoreFormat{},
		Source: &api.DiskSource{},
	}

	source := containerdisk.GetDiskTargetPathFromLauncherView(diskIndex)
	if info := c.DisksInfo[volumeName]; info != nil {
		disk.BackingStore.Format.Type = info.Format
	} else {
		return fmt.Errorf("no disk info provided for volume %s", volumeName)
	}
	disk.BackingStore.Source.File = source
	disk.BackingStore.Type = "file"

	return nil
}

func Convert_v1_EphemeralVolumeSource_To_api_Disk(volumeName string, disk *api.Disk, c *ConverterContext) error {
	disk.Type = "file"
	setDiskDriver(disk, "qcow2", true)
	disk.Source.File = c.EphemeraldiskCreator.GetFilePath(volumeName)
	disk.BackingStore = &api.BackingStore{
		Format: &api.BackingStoreFormat{},
		Source: &api.DiskSource{},
	}

	backingDisk := &api.Disk{Driver: &api.DiskDriver{}}
	if c.IsBlockPVC[volumeName] {
		if err := Convert_v1_BlockVolumeSource_To_api_Disk(volumeName, backingDisk, c.VolumesDiscardIgnore); err != nil {
			return err
		}
	} else {
		if err := Convert_v1_FilesystemVolumeSource_To_api_Disk(volumeName, backingDisk, c.VolumesDiscardIgnore); err != nil {
			return err
		}
	}
	disk.BackingStore.Format.Type = backingDisk.Driver.Type
	disk.BackingStore.Source = &backingDisk.Source
	disk.BackingStore.Type = backingDisk.Type

	return nil
}

func Convert_v1_Rng_To_api_Rng(_ *v1.Rng, rng *api.Rng, c *ConverterContext) error {

	// default rng model for KVM/QEMU virtualization
	rng.Model = virtio.InterpretTransitionalModelType(&c.UseVirtioTransitional, c.Architecture.GetArchitecture())

	// default backend model, random
	rng.Backend = &api.RngBackend{
		Model: "random",
	}

	// the default source for rng is dev urandom
	rng.Backend.Source = "/dev/urandom"

	if c.UseLaunchSecuritySEV || c.UseLaunchSecurityPV {
		rng.Driver = &api.RngDriver{
			IOMMU: "on",
		}
	}

	return nil
}

func Convert_v1_Usbredir_To_api_Usbredir(vmi *v1.VirtualMachineInstance, domainDevices *api.Devices, _ *ConverterContext) error {
	clientDevices := vmi.Spec.Domain.Devices.ClientPassthrough

	// Default is to have USB Redirection disabled
	if clientDevices == nil {
		return nil
	}

	// Note that at the moment, we don't require any specific input to configure the USB devices
	// so we simply create the maximum allowed dictated by v1.UsbClientPassthroughMaxNumberOf
	redirectDevices := make([]api.RedirectedDevice, v1.UsbClientPassthroughMaxNumberOf)

	for i := 0; i < v1.UsbClientPassthroughMaxNumberOf; i++ {
		path := fmt.Sprintf("/var/run/kubevirt-private/%s/virt-usbredir-%d", vmi.ObjectMeta.UID, i)
		redirectDevices[i] = api.RedirectedDevice{
			Type: "unix",
			Bus:  "usb",
			Source: api.RedirectedDeviceSource{
				Mode: "bind",
				Path: path,
			},
		}
	}
	domainDevices.Redirs = redirectDevices
	return nil
}

func convertPanicDevices(panicDevices []v1.PanicDevice) []api.PanicDevice {
	var domainPanicDevices []api.PanicDevice

	for _, panicDevice := range panicDevices {
		domainPanicDevices = append(domainPanicDevices, api.PanicDevice{Model: panicDevice.Model})
	}

	return domainPanicDevices
}

func Convert_v1_Sound_To_api_Sound(vmi *v1.VirtualMachineInstance, domainDevices *api.Devices, _ *ConverterContext) {
	sound := vmi.Spec.Domain.Devices.Sound

	// Default is to not have any Sound device
	if sound == nil {
		return
	}

	model := "ich9"
	if sound.Model == "ac97" {
		model = "ac97"
	}

	soundCards := make([]api.SoundCard, 1)
	soundCards[0] = api.SoundCard{
		Alias: api.NewUserDefinedAlias(sound.Name),
		Model: model,
	}

	domainDevices.SoundCards = soundCards
}

func Convert_v1_Input_To_api_InputDevice(input *v1.Input, inputDevice *api.Input) error {
	if input.Bus != v1.InputBusVirtio && input.Bus != v1.InputBusUSB && input.Bus != "" {
		return fmt.Errorf("input contains unsupported bus %s", input.Bus)
	}

	if input.Bus != v1.InputBusVirtio && input.Bus != v1.InputBusUSB {
		input.Bus = v1.InputBusUSB
	}

	if input.Type != v1.InputTypeTablet {
		return fmt.Errorf("input contains unsupported type %s", input.Type)
	}

	inputDevice.Bus = input.Bus
	inputDevice.Type = input.Type
	inputDevice.Alias = api.NewUserDefinedAlias(input.Name)

	if input.Bus == v1.InputBusVirtio {
		inputDevice.Model = v1.VirtIO
	}
	return nil
}

func Convert_v1_Clock_To_api_Clock(source *v1.Clock, clock *api.Clock) error {
	if source.UTC != nil {
		clock.Offset = "utc"
		if source.UTC.OffsetSeconds != nil {
			clock.Adjustment = strconv.Itoa(*source.UTC.OffsetSeconds)
		} else {
			clock.Adjustment = "reset"
		}
	} else if source.Timezone != nil {
		clock.Offset = "timezone"
		clock.Timezone = string(*source.Timezone)
	}

	if source.Timer != nil {
		if source.Timer.RTC != nil {
			newTimer := api.Timer{Name: "rtc"}
			newTimer.Track = string(source.Timer.RTC.Track)
			newTimer.TickPolicy = string(source.Timer.RTC.TickPolicy)
			newTimer.Present = boolToYesNo(source.Timer.RTC.Enabled, true)
			clock.Timer = append(clock.Timer, newTimer)
		}
		if source.Timer.PIT != nil {
			newTimer := api.Timer{Name: "pit"}
			newTimer.Present = boolToYesNo(source.Timer.PIT.Enabled, true)
			newTimer.TickPolicy = string(source.Timer.PIT.TickPolicy)
			clock.Timer = append(clock.Timer, newTimer)
		}
		if source.Timer.KVM != nil {
			newTimer := api.Timer{Name: "kvmclock"}
			newTimer.Present = boolToYesNo(source.Timer.KVM.Enabled, true)
			clock.Timer = append(clock.Timer, newTimer)
		}
		if source.Timer.HPET != nil {
			newTimer := api.Timer{Name: "hpet"}
			newTimer.Present = boolToYesNo(source.Timer.HPET.Enabled, true)
			newTimer.TickPolicy = string(source.Timer.HPET.TickPolicy)
			clock.Timer = append(clock.Timer, newTimer)
		}
		if source.Timer.Hyperv != nil {
			newTimer := api.Timer{Name: "hypervclock"}
			newTimer.Present = boolToYesNo(source.Timer.Hyperv.Enabled, true)
			clock.Timer = append(clock.Timer, newTimer)
		}
	}

	return nil
}

func convertFeatureState(source *v1.FeatureState) *api.FeatureState {
	if source != nil {
		return &api.FeatureState{
			State: boolToOnOff(source.Enabled, true),
		}
	}
	return nil
}

func Convert_v1_Features_To_api_Features(source *v1.Features, features *api.Features, c *ConverterContext) error {
	if source.ACPI.Enabled == nil || *source.ACPI.Enabled {
		features.ACPI = &api.FeatureEnabled{}
	}
	if source.SMM != nil {
		if source.SMM.Enabled == nil || *source.SMM.Enabled {
			features.SMM = &api.FeatureEnabled{}
		}
	}
	if source.APIC != nil {
		if source.APIC.Enabled == nil || *source.APIC.Enabled {
			features.APIC = &api.FeatureEnabled{}
		}
	}
	if source.Hyperv != nil {
		features.Hyperv = &api.FeatureHyperv{}
		err := Convert_v1_FeatureHyperv_To_api_FeatureHyperv(source.Hyperv, features.Hyperv)
		if err != nil {
			return nil
		}
	} else if source.HypervPassthrough != nil && *source.HypervPassthrough.Enabled {
		features.Hyperv = &api.FeatureHyperv{
			Mode: api.HypervModePassthrough,
		}
	}
	if source.KVM != nil {
		features.KVM = &api.FeatureKVM{
			Hidden: &api.FeatureState{
				State: boolToOnOff(&source.KVM.Hidden, false),
			},
		}
	}
	if source.Pvspinlock != nil {
		features.PVSpinlock = &api.FeaturePVSpinlock{
			State: boolToOnOff(source.Pvspinlock.Enabled, true),
		}
	}

	if c.UseLaunchSecurityTDX {
		features.PMU = &api.FeatureState{
			State: "off",
		}
	}

	return nil
}

func Convert_v1_FeatureHyperv_To_api_FeatureHyperv(source *v1.FeatureHyperv, hyperv *api.FeatureHyperv) error {
	if source.Spinlocks != nil {
		hyperv.Spinlocks = &api.FeatureSpinlocks{
			State:   boolToOnOff(source.Spinlocks.Enabled, true),
			Retries: source.Spinlocks.Retries,
		}
	}
	if source.VendorID != nil {
		hyperv.VendorID = &api.FeatureVendorID{
			State: boolToOnOff(source.VendorID.Enabled, true),
			Value: source.VendorID.VendorID,
		}
	}

	hyperv.Relaxed = convertFeatureState(source.Relaxed)
	hyperv.Reset = convertFeatureState(source.Reset)
	hyperv.Runtime = convertFeatureState(source.Runtime)
	hyperv.SyNIC = convertFeatureState(source.SyNIC)
	hyperv.SyNICTimer = convertV1ToAPISyNICTimer(source.SyNICTimer)
	hyperv.VAPIC = convertFeatureState(source.VAPIC)
	hyperv.VPIndex = convertFeatureState(source.VPIndex)
	hyperv.Frequencies = convertFeatureState(source.Frequencies)
	hyperv.Reenlightenment = convertFeatureState(source.Reenlightenment)
	hyperv.TLBFlush = convertFeatureState(source.TLBFlush)
	hyperv.IPI = convertFeatureState(source.IPI)
	hyperv.EVMCS = convertFeatureState(source.EVMCS)
	return nil
}

func convertV1ToAPISyNICTimer(syNICTimer *v1.SyNICTimer) *api.SyNICTimer {
	if syNICTimer == nil {
		return nil
	}

	result := &api.SyNICTimer{
		State: boolToOnOff(syNICTimer.Enabled, true),
	}

	if syNICTimer.Direct != nil {
		result.Direct = &api.FeatureState{
			State: boolToOnOff(syNICTimer.Direct.Enabled, true),
		}
	}
	return result
}

func ConvertV1ToAPIBalloning(source *v1.Devices, ballooning *api.MemBalloon, c *ConverterContext) {
	if source != nil && source.AutoattachMemBalloon != nil && !*source.AutoattachMemBalloon {
		ballooning.Model = "none"
		ballooning.Stats = nil
	} else {
		ballooning.Model = virtio.InterpretTransitionalModelType(&c.UseVirtioTransitional, c.Architecture.GetArchitecture())
		if c.MemBalloonStatsPeriod != 0 {
			ballooning.Stats = &api.Stats{Period: c.MemBalloonStatsPeriod}
		}
		if c.UseLaunchSecuritySEV || c.UseLaunchSecurityPV {
			ballooning.Driver = &api.MemBalloonDriver{
				IOMMU: "on",
			}
		}
		ballooning.FreePageReporting = boolToOnOff(&c.FreePageReporting, false)
	}
}

func initializeQEMUCmdAndQEMUArg(domain *api.Domain) {
	if domain.Spec.QEMUCmd == nil {
		domain.Spec.QEMUCmd = &api.Commandline{}
	}

	if domain.Spec.QEMUCmd.QEMUArg == nil {
		domain.Spec.QEMUCmd.QEMUArg = make([]api.Arg, 0)
	}
}

func setupDomainMemory(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if vmi.Spec.Domain.Memory == nil ||
		vmi.Spec.Domain.Memory.MaxGuest == nil ||
		vmi.Spec.Domain.Memory.Guest.Equal(*vmi.Spec.Domain.Memory.MaxGuest) {
		var err error

		domain.Spec.Memory, err = vcpu.QuantityToByte(*vcpu.GetVirtualMemory(vmi))
		if err != nil {
			return err
		}
		return nil
	}

	maxMemory, err := vcpu.QuantityToByte(*vmi.Spec.Domain.Memory.MaxGuest)
	if err != nil {
		return err
	}

	domain.Spec.MaxMemory = &api.MaxMemory{
		Unit:  maxMemory.Unit,
		Value: maxMemory.Value,
	}

	currentMemory, err := vcpu.QuantityToByte(*vmi.Spec.Domain.Memory.Guest)
	if err != nil {
		return err
	}

	domain.Spec.Memory = currentMemory

	return nil
}

func Convert_v1_Firmware_To_related_apis(vmi *v1.VirtualMachineInstance, domain *api.Domain, c *ConverterContext) error {
	firmware := vmi.Spec.Domain.Firmware
	if firmware == nil {
		return nil
	}

	domain.Spec.SysInfo.System = []api.Entry{
		{
			Name:  "uuid",
			Value: string(firmware.UUID),
		},
	}

	if vmi.IsBootloaderEFI() {
		domain.Spec.OS.BootLoader = &api.Loader{
			Path:     c.EFIConfiguration.EFICode,
			ReadOnly: "yes",
			Secure:   boolToYesNo(&c.EFIConfiguration.SecureLoader, false),
		}

		if util.IsSEVSNPVMI(vmi) || util.IsTDXVMI(vmi) {
			// Use stateless firmware for the TDX/SNP VMs
			domain.Spec.OS.BootLoader.Type = "rom"
			domain.Spec.OS.NVRam = nil
		} else {
			domain.Spec.OS.BootLoader.Type = "pflash"
			domain.Spec.OS.NVRam = &api.NVRam{
				Template: c.EFIConfiguration.EFIVars,
				NVRam:    filepath.Join(services.PathForNVram(vmi), vmi.Name+"_VARS.fd"),
			}
		}
	}

	if firmware.Bootloader != nil && firmware.Bootloader.BIOS != nil {
		if firmware.Bootloader.BIOS.UseSerial != nil && *firmware.Bootloader.BIOS.UseSerial {
			domain.Spec.OS.BIOS = &api.BIOS{
				UseSerial: "yes",
			}
		}
	}

	if len(firmware.Serial) > 0 {
		domain.Spec.SysInfo.System = append(domain.Spec.SysInfo.System, api.Entry{
			Name:  "serial",
			Value: firmware.Serial,
		})
	}

	if util.HasKernelBootContainerImage(vmi) {
		kb := firmware.KernelBoot

		log.Log.Object(vmi).Infof("kernel boot defined for VMI. Converting to domain XML")
		if kb.Container.KernelPath != "" {
			kernelPath := containerdisk.GetKernelBootArtifactPathFromLauncherView(kb.Container.KernelPath)
			log.Log.Object(vmi).Infof("setting kernel path for kernel boot: %s", kernelPath)
			domain.Spec.OS.Kernel = kernelPath
		}

		if kb.Container.InitrdPath != "" {
			initrdPath := containerdisk.GetKernelBootArtifactPathFromLauncherView(kb.Container.InitrdPath)
			log.Log.Object(vmi).Infof("setting initrd path for kernel boot: %s", initrdPath)
			domain.Spec.OS.Initrd = initrdPath
		}

	}

	// Define custom command-line arguments even if kernel-boot container is not defined
	if firmware.KernelBoot != nil {
		log.Log.Object(vmi).Infof("setting custom kernel arguments: %s", firmware.KernelBoot.KernelArgs)
		domain.Spec.OS.KernelArgs = firmware.KernelBoot.KernelArgs
	}

	if err := Convert_v1_Firmware_ACPI_To_related_apis(firmware, domain, vmi.Spec.Volumes); err != nil {
		return err
	}

	return nil
}

func Convert_v1_Firmware_ACPI_To_related_apis(firmware *v1.Firmware, domain *api.Domain, volumes []v1.Volume) error {
	if firmware.ACPI == nil {
		return nil
	}

	if firmware.ACPI.SlicNameRef == "" && firmware.ACPI.MsdmNameRef == "" {
		return fmt.Errorf("No ACPI tables were set. Expecting at least one.")
	}

	if domain.Spec.OS.ACPI == nil {
		domain.Spec.OS.ACPI = &api.OSACPI{}
	}

	if val, err := createACPITable("slic", firmware.ACPI.SlicNameRef, volumes); err != nil {
		return err
	} else if val != nil {
		domain.Spec.OS.ACPI.Table = append(domain.Spec.OS.ACPI.Table, *val)
	}

	if val, err := createACPITable("msdm", firmware.ACPI.MsdmNameRef, volumes); err != nil {
		return err
	} else if val != nil {
		domain.Spec.OS.ACPI.Table = append(domain.Spec.OS.ACPI.Table, *val)
	}

	// if field was set but volume was not found, helper function will return error
	return nil
}

func createACPITable(source, volumeName string, volumes []v1.Volume) (*api.ACPITable, error) {
	if volumeName == "" {
		return nil, nil
	}

	for _, volume := range volumes {
		if volume.Name != volumeName {
			continue
		}

		if volume.Secret == nil {
			// Unsupported. This should have been blocked by webhook, so warn user.
			return nil, fmt.Errorf("Firmware's volume type is unsupported for %s", source)
		}

		// Return path to table's binary data
		sourcePath := config.GetSecretSourcePath(volumeName)
		sourcePath = filepath.Join(sourcePath, fmt.Sprintf("%s.bin", source))
		return &api.ACPITable{
			Type: source,
			Path: sourcePath,
		}, nil
	}

	return nil, fmt.Errorf("Firmware's volume for %s was not found", source)
}

func hasIOThreads(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.IOThreadsPolicy != nil {
		return true
	}
	for _, diskDevice := range vmi.Spec.Domain.Devices.Disks {
		if diskDevice.DedicatedIOThread != nil && *diskDevice.DedicatedIOThread {
			return true
		}
	}
	return false
}

func getIOThreadsCountType(vmi *v1.VirtualMachineInstance) (ioThreadCount, autoThreads int) {
	dedicatedThreads := 0

	var threadPoolLimit int
	policy := vmi.Spec.Domain.IOThreadsPolicy
	switch {
	case policy == nil:
		threadPoolLimit = 1
	case *policy == v1.IOThreadsPolicyShared:
		threadPoolLimit = 1
	case *policy == v1.IOThreadsPolicyAuto:
		// When IOThreads policy is set to auto and we've allocated a dedicated
		// pCPU for the emulator thread, we can place IOThread and Emulator thread in the same pCPU
		if vmi.IsCPUDedicated() && vmi.Spec.Domain.CPU.IsolateEmulatorThread {
			threadPoolLimit = 1
		} else {
			numCPUs := 1
			// Requested CPU's is guaranteed to be no greater than the limit
			if cpuRequests, ok := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU]; ok {
				numCPUs = int(cpuRequests.Value())
			} else if cpuLimit, ok := vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU]; ok {
				numCPUs = int(cpuLimit.Value())
			}

			threadPoolLimit = numCPUs * 2
		}
	case *policy == v1.IOThreadsPolicySupplementalPool:
		if vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount != nil {
			ioThreadCount = int(*vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount)
		}
		return
	}

	for _, diskDevice := range vmi.Spec.Domain.Devices.Disks {
		if diskDevice.DedicatedIOThread != nil && *diskDevice.DedicatedIOThread {
			dedicatedThreads += 1
		} else {
			autoThreads += 1
		}
	}

	if (autoThreads + dedicatedThreads) > threadPoolLimit {
		autoThreads = threadPoolLimit - dedicatedThreads
		// We need at least one shared thread
		if autoThreads < 1 {
			autoThreads = 1
		}
	}

	ioThreadCount = autoThreads + dedicatedThreads
	return
}

func setIOThreads(vmi *v1.VirtualMachineInstance, domain *api.Domain, vcpus uint) {
	if !hasIOThreads(vmi) {
		return
	}
	currentAutoThread := defaultIOThread
	ioThreadCount, autoThreads := getIOThreadsCountType(vmi)
	if ioThreadCount != 0 {
		if domain.Spec.IOThreads == nil {
			domain.Spec.IOThreads = &api.IOThreads{}
		}
		domain.Spec.IOThreads.IOThreads = uint(ioThreadCount)
	}
	if vmi.Spec.Domain.IOThreadsPolicy != nil &&
		*vmi.Spec.Domain.IOThreadsPolicy == v1.IOThreadsPolicySupplementalPool {
		iothreads := &api.DiskIOThreads{}
		for id := 1; id <= int(*vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount); id++ {
			iothreads.IOThread = append(iothreads.IOThread, api.DiskIOThread{Id: uint32(id)})
		}
		for i, disk := range domain.Spec.Devices.Disks {
			// Only disks with virtio bus support IOThreads
			if disk.Target.Bus == v1.DiskBusVirtio {
				domain.Spec.Devices.Disks[i].Driver.IOThreads = iothreads
			}
		}
	} else {
		currentDedicatedThread := uint(autoThreads + 1)
		for i, disk := range domain.Spec.Devices.Disks {
			// Only disks with virtio bus support IOThreads
			if disk.Target.Bus == v1.DiskBusVirtio {
				if vmi.Spec.Domain.Devices.Disks[i].DedicatedIOThread != nil && *vmi.Spec.Domain.Devices.Disks[i].DedicatedIOThread {
					domain.Spec.Devices.Disks[i].Driver.IOThread = pointer.P(currentDedicatedThread)
					currentDedicatedThread += 1
				} else {
					domain.Spec.Devices.Disks[i].Driver.IOThread = pointer.P(currentAutoThread)
					// increment the threadId to be used next but wrap around at the thread limit
					// the odd math here is because thread ID's start at 1, not 0
					currentAutoThread = (currentAutoThread % uint(autoThreads)) + 1
				}
			}
		}
	}

	// Virtio-scsi doesn't support IO threads yet, only the SCSI controller supports.
	setIOThreadSCSIController := false
	for i, disk := range domain.Spec.Devices.Disks {
		// Only disks with virtio bus support IOThreads
		if disk.Target.Bus == v1.DiskBusSCSI {
			if vmi.Spec.Domain.Devices.Disks[i].DedicatedIOThread != nil && *vmi.Spec.Domain.Devices.Disks[i].DedicatedIOThread {
				setIOThreadSCSIController = true
				break
			}
		}
	}
	if !setIOThreadSCSIController {
		return
	}
	for i, controller := range domain.Spec.Devices.Controllers {
		if controller.Type == "scsi" {
			if controller.Driver == nil {
				domain.Spec.Devices.Controllers[i].Driver = &api.ControllerDriver{}
			}
			domain.Spec.Devices.Controllers[i].Driver.IOThread = pointer.P(currentAutoThread)
			domain.Spec.Devices.Controllers[i].Driver.Queues = pointer.P(vcpus)
		}
	}
}

func Convert_v1_VirtualMachineInstance_To_api_Domain(vmi *v1.VirtualMachineInstance, domain *api.Domain, c *ConverterContext) (err error) {
	var controllerDriver *api.ControllerDriver

	precond.MustNotBeNil(vmi)
	precond.MustNotBeNil(domain)
	precond.MustNotBeNil(c)

	builder := NewDomainBuilder(metadata.DomainConfigurator{})
	if err := builder.Build(vmi, domain); err != nil {
		return err
	}

	// Set VM CPU cores
	// CPU topology will be created everytime, because user can specify
	// number of cores in vmi.Spec.Domain.Resources.Requests/Limits, not only
	// in vmi.Spec.Domain.CPU
	cpuTopology := vcpu.GetCPUTopology(vmi)
	cpuCount := vcpu.CalculateRequestedVCPUs(cpuTopology)

	domain.Spec.CPU.Topology = cpuTopology
	domain.Spec.VCPU = &api.VCPU{
		Placement: "static",
		CPUs:      cpuCount,
	}
	// set the maximum number of sockets here to allow hot-plug CPUs
	if vmiCPU := vmi.Spec.Domain.CPU; vmiCPU != nil && vmiCPU.MaxSockets != 0 && c.Architecture.SupportCPUHotplug() {
		domainVCPUTopologyForHotplug(vmi, domain)
	}

	kvmPath := "/dev/kvm"
	if _, err := os.Stat(kvmPath); errors.Is(err, os.ErrNotExist) {
		if c.AllowEmulation {
			logger := log.DefaultLogger()
			logger.Infof("Hardware emulation device '%s' not present. Using software emulation.", kvmPath)
			domain.Spec.Type = "qemu"
		} else {
			return fmt.Errorf("hardware emulation device '%s' not present", kvmPath)
		}
	} else if err != nil {
		return err
	}

	newChannel := Add_Agent_To_api_Channel()
	domain.Spec.Devices.Channels = append(domain.Spec.Devices.Channels, newChannel)

	if downwardmetrics.HasDevice(&vmi.Spec) {
		// Handle downwardMetrics
		domain.Spec.Devices.Channels = append(domain.Spec.Devices.Channels, convertDownwardMetricsChannel())
	}

	domain.Spec.SysInfo = &api.SysInfo{}

	err = Convert_v1_Firmware_To_related_apis(vmi, domain, c)
	if err != nil {
		return err
	}

	if c.UseLaunchSecuritySEV || c.UseLaunchSecurityPV {
		controllerDriver = &api.ControllerDriver{
			IOMMU: "on",
		}
	}

	if util.UseLaunchSecurity(vmi) {
		domain.Spec.LaunchSecurity = c.Architecture.LaunchSecurity(vmi)
	}

	if c.SMBios != nil {
		domain.Spec.SysInfo.System = append(domain.Spec.SysInfo.System,
			api.Entry{
				Name:  "manufacturer",
				Value: c.SMBios.Manufacturer,
			},
			api.Entry{
				Name:  "family",
				Value: c.SMBios.Family,
			},
			api.Entry{
				Name:  "product",
				Value: c.SMBios.Product,
			},
			api.Entry{
				Name:  "sku",
				Value: c.SMBios.Sku,
			},
			api.Entry{
				Name:  "version",
				Value: c.SMBios.Version,
			},
		)
	}

	// Take SMBios values from the VirtualMachineOptions
	if c.Architecture.IsSMBiosNeeded() {
		domain.Spec.OS.SMBios = &api.SMBios{
			Mode: "sysinfo",
		}
	}

	if vmi.Spec.Domain.Chassis != nil {
		domain.Spec.SysInfo.Chassis = []api.Entry{
			{
				Name:  "manufacturer",
				Value: vmi.Spec.Domain.Chassis.Manufacturer,
			},
			{
				Name:  "version",
				Value: vmi.Spec.Domain.Chassis.Version,
			},
			{
				Name:  "serial",
				Value: vmi.Spec.Domain.Chassis.Serial,
			},
			{
				Name:  "asset",
				Value: vmi.Spec.Domain.Chassis.Asset,
			},
			{
				Name:  "sku",
				Value: vmi.Spec.Domain.Chassis.Sku,
			},
		}
	}

	if err = setupDomainMemory(vmi, domain); err != nil {
		return err
	}

	var isMemfdRequired = false
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Hugepages != nil {
		domain.Spec.MemoryBacking = &api.MemoryBacking{
			HugePages: &api.HugePages{},
		}
		if val := vmi.Annotations[v1.MemfdMemoryBackend]; val != "false" {
			isMemfdRequired = true
		}
	}
	// virtiofs require shared access
	if util.IsVMIVirtiofsEnabled(vmi) {
		if domain.Spec.MemoryBacking == nil {
			domain.Spec.MemoryBacking = &api.MemoryBacking{}
		}
		domain.Spec.MemoryBacking.Access = &api.MemoryBackingAccess{
			Mode: "shared",
		}
		isMemfdRequired = true
	}

	if isMemfdRequired {
		// Set memfd as memory backend to solve SELinux restrictions
		// See the issue: https://github.com/kubevirt/kubevirt/issues/3781
		domain.Spec.MemoryBacking.Source = &api.MemoryBackingSource{Type: "memfd"}

		// NUMA is required in order to use memfd
		if domain.Spec.CPU.NUMA == nil {
			domain.Spec.CPU.NUMA = &api.NUMA{
				Cells: []api.NUMACell{
					{
						ID:     "0",
						CPUs:   fmt.Sprintf("0-%d", domain.Spec.VCPU.CPUs-1),
						Memory: uint64(vcpu.GetVirtualMemory(vmi).Value() / int64(1024)),
						Unit:   "KiB",
					},
				},
			}
		}
	}

	volumeIndices := map[string]int{}
	volumes := map[string]*v1.Volume{}
	for i, volume := range vmi.Spec.Volumes {
		volumes[volume.Name] = volume.DeepCopy()
		volumeIndices[volume.Name] = i
	}

	var numBlkQueues *uint
	virtioBlkMQRequested := (vmi.Spec.Domain.Devices.BlockMultiQueue != nil) && (*vmi.Spec.Domain.Devices.BlockMultiQueue)
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
			err = Convert_v1_Missing_Volume_To_api_Disk(&newDisk)
		case hpOk:
			err = Convert_v1_Hotplug_Volume_To_api_Disk(volume, &newDisk, c)
		default:
			err = Convert_v1_Volume_To_api_Disk(volume, &newDisk, c, volumeIndices[disk.Name])
		}

		if err != nil {
			return err
		}

		if err := Convert_v1_BlockSize_To_api_BlockIO(&disk, &newDisk); err != nil {
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
	}
	// Handle virtioFS
	domain.Spec.Devices.Filesystems = append(domain.Spec.Devices.Filesystems, convertFileSystems(vmi.Spec.Domain.Devices.Filesystems)...)

	domain.Spec.Devices.PanicDevices = append(domain.Spec.Devices.PanicDevices, convertPanicDevices(vmi.Spec.Domain.Devices.PanicDevices)...)

	Convert_v1_Sound_To_api_Sound(vmi, &domain.Spec.Devices, c)

	if vmi.Spec.Domain.Devices.Watchdog != nil {
		newWatchdog := &api.Watchdog{}
		err := c.Architecture.ConvertWatchdog(vmi.Spec.Domain.Devices.Watchdog, newWatchdog)
		if err != nil {
			return err
		}
		domain.Spec.Devices.Watchdogs = append(domain.Spec.Devices.Watchdogs, *newWatchdog)
	}

	if vmi.Spec.Domain.Devices.Rng != nil {
		newRng := &api.Rng{}
		err := Convert_v1_Rng_To_api_Rng(vmi.Spec.Domain.Devices.Rng, newRng, c)
		if err != nil {
			return err
		}
		domain.Spec.Devices.Rng = newRng
	}

	domain.Spec.Devices.Ballooning = &api.MemBalloon{}
	ConvertV1ToAPIBalloning(&vmi.Spec.Domain.Devices, domain.Spec.Devices.Ballooning, c)

	if vmi.Spec.Domain.Devices.Inputs != nil {
		inputDevices := make([]api.Input, 0)
		for i := range vmi.Spec.Domain.Devices.Inputs {
			inputDevice := api.Input{}
			err := Convert_v1_Input_To_api_InputDevice(&vmi.Spec.Domain.Devices.Inputs[i], &inputDevice)
			if err != nil {
				return err
			}
			inputDevices = append(inputDevices, inputDevice)
		}
		domain.Spec.Devices.Inputs = inputDevices
	}

	err = Convert_v1_Usbredir_To_api_Usbredir(vmi, &domain.Spec.Devices, c)
	if err != nil {
		return err
	}

	// Creating USB controller, disabled by default
	usbController := api.Controller{
		Type:  "usb",
		Index: "0",
		Model: "none",
	}
	if c.Architecture.IsUSBNeeded(vmi) {
		usbController.Model = "qemu-xhci"
	}
	domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, usbController)

	if needsSCSIController(vmi) {
		scsiController := c.Architecture.ScsiController(virtio.InterpretTransitionalModelType(&c.UseVirtioTransitional, c.Architecture.GetArchitecture()), controllerDriver)
		domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, scsiController)
	}

	if c.Architecture.SupportPCIHole64Disabling() && shouldDisablePCIHole64(vmi) {
		domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers,
			api.Controller{
				Type:  "pci",
				Index: "0",
				Model: "pcie-root",
				PCIHole64: &api.PCIHole64{
					Value: 0,
					Unit:  "KiB",
				},
			},
		)
	}

	if vmi.Spec.Domain.Clock != nil {
		clock := vmi.Spec.Domain.Clock
		newClock := &api.Clock{}
		err := Convert_v1_Clock_To_api_Clock(clock, newClock)
		if err != nil {
			return err
		}
		domain.Spec.Clock = newClock
	}

	if vmi.Spec.Domain.Features != nil {
		domain.Spec.Features = &api.Features{}
		err := Convert_v1_Features_To_api_Features(vmi.Spec.Domain.Features, domain.Spec.Features, c)

		if c.Architecture.HasVMPort() {
			domain.Spec.Features.VMPort = &api.FeatureState{State: "off"}
		}

		if err != nil {
			return err
		}
	}

	if machine := vmi.Spec.Domain.Machine; machine != nil {
		domain.Spec.OS.Type.Machine = machine.Type
	}

	if vmi.Spec.Domain.CPU != nil {
		// Set VM CPU model and vendor
		if vmi.Spec.Domain.CPU.Model != "" {
			if vmi.Spec.Domain.CPU.Model == v1.CPUModeHostModel || vmi.Spec.Domain.CPU.Model == v1.CPUModeHostPassthrough {
				domain.Spec.CPU.Mode = vmi.Spec.Domain.CPU.Model
			} else {
				domain.Spec.CPU.Mode = "custom"
				domain.Spec.CPU.Model = vmi.Spec.Domain.CPU.Model
			}
		}

		// Set VM CPU features
		existingFeatures := make(map[string]struct{})
		if vmi.Spec.Domain.CPU.Features != nil {
			for _, feature := range vmi.Spec.Domain.CPU.Features {
				existingFeatures[feature.Name] = struct{}{}
				domain.Spec.CPU.Features = append(domain.Spec.CPU.Features, api.CPUFeature{
					Name:   feature.Name,
					Policy: feature.Policy,
				})
			}
		}

		/*
						Libvirt validation fails when a CPU model is usable
						by QEMU but lacks features listed in
						`/usr/share/libvirt/cpu_map/[CPU Model].xml` on a node
						To avoid the validation error mentioned above we can disable
						deprecated features in the `/usr/share/libvirt/cpu_map/[CPU Model].xml` files.
						Examples of validation error:
			    		https://bugzilla.redhat.com/show_bug.cgi?id=2122283 - resolve by obsolete Opteron_G2
						https://gitlab.com/libvirt/libvirt/-/issues/304 - resolve by disabling mpx which is deprecated
						Issue in Libvirt: https://gitlab.com/libvirt/libvirt/-/issues/608
						once the issue is resolved we can remove mpx disablement
		*/

		_, exists := existingFeatures["mpx"]
		if c.Architecture.RequiresMPXCPUValidation() && !exists && vmi.Spec.Domain.CPU.Model != v1.CPUModeHostModel && vmi.Spec.Domain.CPU.Model != v1.CPUModeHostPassthrough {
			domain.Spec.CPU.Features = append(domain.Spec.CPU.Features, api.CPUFeature{
				Name:   "mpx",
				Policy: "disable",
			})
		}

		// Adjust guest vcpu config. Currently will handle vCPUs to pCPUs pinning
		if vmi.IsCPUDedicated() {
			err = vcpu.AdjustDomainForTopologyAndCPUSet(domain, vmi, c.Topology, c.CPUSet, hasIOThreads(vmi))
			if err != nil {
				return err
			}
		}
	}

	// Make use of the tsc frequency topology hint
	if topology.IsManualTSCFrequencyRequired(vmi) {
		freq := *vmi.Status.TopologyHints.TSCFrequency
		clock := domain.Spec.Clock
		if clock == nil {
			clock = &api.Clock{}
		}
		clock.Timer = append(clock.Timer, api.Timer{Name: "tsc", Frequency: strconv.FormatInt(freq, 10)})
		domain.Spec.Clock = clock
	}

	domain.Spec.Devices.HostDevices = append(domain.Spec.Devices.HostDevices, c.GenericHostDevices...)
	domain.Spec.Devices.HostDevices = append(domain.Spec.Devices.HostDevices, c.GPUHostDevices...)

	if vmi.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU.Model == "" {
		domain.Spec.CPU.Mode = v1.CPUModeHostModel
	}

	if vmi.Spec.Domain.Devices.AutoattachSerialConsole == nil || *vmi.Spec.Domain.Devices.AutoattachSerialConsole {
		// Add mandatory console device
		domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, api.Controller{
			Type:   "virtio-serial",
			Index:  "0",
			Model:  virtio.InterpretTransitionalModelType(&c.UseVirtioTransitional, c.Architecture.GetArchitecture()),
			Driver: controllerDriver,
		})

		var serialPort uint = 0
		var serialType string = "serial"
		domain.Spec.Devices.Consoles = []api.Console{
			{
				Type: "pty",
				Target: &api.ConsoleTarget{
					Type: &serialType,
					Port: &serialPort,
				},
			},
		}

		socketPath := fmt.Sprintf("%s/%s/virt-serial%d", util.VirtPrivateDir, vmi.ObjectMeta.UID, serialPort)
		domain.Spec.Devices.Serials = []api.Serial{
			{
				Type: "unix",
				Target: &api.SerialTarget{
					Port: &serialPort,
				},
				Source: &api.SerialSource{
					Mode: "bind",
					Path: socketPath,
				},
			},
		}

		if c.SerialConsoleLog {
			domain.Spec.Devices.Serials[0].Log = &api.SerialLog{
				File:   fmt.Sprintf("%s-log", socketPath),
				Append: "on",
			}
		}

	}

	if vmi.Spec.Domain.Devices.AutoattachGraphicsDevice == nil || *vmi.Spec.Domain.Devices.AutoattachGraphicsDevice {
		c.Architecture.AddGraphicsDevice(vmi, domain, c.BochsForEFIGuests && vmi.IsBootloaderEFI())
		if vmi.Spec.Domain.Devices.Video != nil {
			video := api.Video{
				Model: api.VideoModel{
					Type:  vmi.Spec.Domain.Devices.Video.Type,
					VRam:  pointer.P(uint(16384)),
					Heads: pointer.P(uint(1)),
				},
			}
			domain.Spec.Devices.Video = []api.Video{video}
		}
		domain.Spec.Devices.Graphics = []api.Graphics{
			{
				Listen: &api.GraphicsListen{
					Type:   "socket",
					Socket: fmt.Sprintf("/var/run/kubevirt-private/%s/virt-vnc", vmi.ObjectMeta.UID),
				},
				Type: "vnc",
			},
		}
	}

	domainInterfaces, err := CreateDomainInterfaces(vmi, c)
	if err != nil {
		return err
	}
	domain.Spec.Devices.Interfaces = append(domain.Spec.Devices.Interfaces, domainInterfaces...)
	domain.Spec.Devices.HostDevices = append(domain.Spec.Devices.HostDevices, c.SRIOVDevices...)

	// Add Ignition Command Line if present
	ignitiondata := vmi.Annotations[v1.IgnitionAnnotation]
	if ignitiondata != "" && strings.Contains(ignitiondata, "ignition") {
		initializeQEMUCmdAndQEMUArg(domain)
		domain.Spec.QEMUCmd.QEMUArg = append(domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: "-fw_cfg"})
		ignitionpath := fmt.Sprintf("%s/%s", ignition.GetDomainBasePath(c.VirtualMachine.Name, c.VirtualMachine.Namespace), ignition.IgnitionFile)
		domain.Spec.QEMUCmd.QEMUArg = append(domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: fmt.Sprintf("name=opt/com.coreos/config,file=%s", ignitionpath)})
	}

	if val := vmi.Annotations[v1.PlacePCIDevicesOnRootComplex]; val == "true" {
		if err := PlacePCIDevicesOnRootComplex(&domain.Spec); err != nil {
			return err
		}
	}

	if c.Architecture.ShouldVerboseLogsBeEnabled() {
		virtLauncherLogVerbosity, err := strconv.Atoi(os.Getenv(services.ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY))
		if err == nil && virtLauncherLogVerbosity > services.EXT_LOG_VERBOSITY_THRESHOLD {
			// isa-debugcon device is only for x86_64
			initializeQEMUCmdAndQEMUArg(domain)

			domain.Spec.QEMUCmd.QEMUArg = append(domain.Spec.QEMUCmd.QEMUArg,
				api.Arg{Value: "-chardev"},
				api.Arg{Value: fmt.Sprintf("file,id=firmwarelog,path=%s", QEMUSeaBiosDebugPipe)},
				api.Arg{Value: "-device"},
				api.Arg{Value: "isa-debugcon,iobase=0x402,chardev=firmwarelog"})
		}
	}

	if tpm.HasDevice(&vmi.Spec) {
		domain.Spec.Devices.TPMs = []api.TPM{
			{
				Model: "tpm-tis",
				Backend: api.TPMBackend{
					Type:    "emulator",
					Version: "2.0",
				},
			},
		}
		if tpm.HasPersistentDevice(&vmi.Spec) {
			domain.Spec.Devices.TPMs[0].Backend.PersistentState = "yes"
			// tpm-crb is not techincally required for persistence, but since there was a desire for both,
			//   we decided to introduce them together. Ultimately, we should use tpm-crb for all cases,
			//   as it is now the generally preferred model
			domain.Spec.Devices.TPMs[0].Model = "tpm-crb"
		}
	}

	// Handle VSOCK CID
	if vmi.Status.VSOCKCID != nil {
		domain.Spec.Devices.VSOCK = &api.VSOCK{
			// Force virtio v1 for vhost-vsock-pci.
			// https://gitlab.com/qemu-project/qemu/-/commit/6209070503989cf4f28549f228989419d4f0b236
			Model: "virtio-non-transitional",
			CID: api.CID{
				Auto:    "no",
				Address: *vmi.Status.VSOCKCID,
			},
		}
	}

	// set bootmenu to give time to access bios
	if vmi.ShouldStartPaused() {
		domain.Spec.OS.BootMenu = &api.BootMenu{
			Enable:  "yes",
			Timeout: pointer.P(bootMenuTimeoutMS),
		}
	}

	setIOThreads(vmi, domain, vcpus)

	return nil
}

func boolToOnOff(value *bool, defaultOn bool) string {
	return boolToString(value, defaultOn, "on", "off")
}

func boolToYesNo(value *bool, defaultYes bool) string {
	return boolToString(value, defaultYes, "yes", "no")
}

func boolToString(value *bool, defaultPositive bool, positive string, negative string) string {
	toString := func(value bool) string {
		if value {
			return positive
		}
		return negative
	}

	if value == nil {
		return toString(defaultPositive)
	}
	return toString(*value)
}

func needsSCSIController(vmi *v1.VirtualMachineInstance) bool {
	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		if getBusFromDisk(disk) == v1.DiskBusSCSI {
			return true
		}
	}
	return !vmi.Spec.Domain.Devices.DisableHotplug
}

func shouldDisablePCIHole64(vmi *v1.VirtualMachineInstance) bool {
	if val, ok := vmi.Annotations[v1.DisablePCIHole64]; ok {
		return strings.EqualFold(val, "true")
	}
	return false
}

func getBusFromDisk(disk v1.Disk) v1.DiskBus {
	if disk.LUN != nil {
		return disk.LUN.Bus
	}
	if disk.Disk != nil {
		return disk.Disk.Bus
	}
	if disk.CDRom != nil {
		return disk.CDRom.Bus
	}
	return ""
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
		if disk.Disk == nil {
			continue
		}
		prefix := getPrefixFromBus(disk.Disk.Bus)
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

func domainVCPUTopologyForHotplug(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
	cpuTopology := vcpu.GetCPUTopology(vmi)
	cpuCount := vcpu.CalculateRequestedVCPUs(cpuTopology)
	// Always allow to hotplug to minimum of 1 socket
	minEnabledCpuCount := cpuTopology.Cores * cpuTopology.Threads
	// Total vCPU count
	enabledCpuCount := cpuCount
	cpuTopology.Sockets = vmi.Spec.Domain.CPU.MaxSockets
	cpuCount = vcpu.CalculateRequestedVCPUs(cpuTopology)
	VCPUs := &api.VCPUs{}
	for id := uint32(0); id < cpuCount; id++ {
		// Enable all requestd vCPUs
		isEnabled := id < enabledCpuCount
		// There should not be fewer vCPU than cores and threads within a single socket
		isHotpluggable := id >= minEnabledCpuCount
		vcpu := api.VCPUsVCPU{
			ID:           id,
			Enabled:      boolToYesNo(&isEnabled, true),
			Hotpluggable: boolToYesNo(&isHotpluggable, false),
		}
		VCPUs.VCPU = append(VCPUs.VCPU, vcpu)
	}

	domain.Spec.VCPUs = VCPUs
	domain.Spec.CPU.Topology = cpuTopology
	domain.Spec.VCPU = &api.VCPU{
		Placement: "static",
		CPUs:      cpuCount,
	}
}
