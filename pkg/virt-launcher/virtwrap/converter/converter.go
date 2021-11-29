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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
*/

package converter

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"

	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"

	"kubevirt.io/kubevirt/pkg/virt-controller/services"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/config"

	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/emptydisk"
	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/ignition"
	"kubevirt.io/kubevirt/pkg/util"
)

type HostDeviceType string

// The location of uefi boot loader on ARM64 is different from that on x86
const (
	defaultIOThread                = uint(1)
	HostDevicePCI   HostDeviceType = "pci"
	HostDeviceMDEV  HostDeviceType = "mdev"
	resolvConf                     = "/etc/resolv.conf"
)

const (
	multiQueueMaxQueues  = uint32(256)
	QEMUSeaBiosDebugPipe = "/var/run/kubevirt-private/QEMUSeaBiosDebugPipe"
)

type deviceNamer struct {
	existingNameMap map[string]string
	usedDeviceMap   map[string]string
}

type HostDevicesList struct {
	Type     HostDeviceType
	AddrList []string
}

type EFIConfiguration struct {
	EFICode      string
	EFIVars      string
	SecureLoader bool
}

type ConverterContext struct {
	Architecture          string
	AllowEmulation        bool
	Secrets               map[string]*k8sv1.Secret
	VirtualMachine        *v1.VirtualMachineInstance
	CPUSet                []int
	IsBlockPVC            map[string]bool
	IsBlockDV             map[string]bool
	HotplugVolumes        map[string]v1.VolumeStatus
	PermanentVolumes      map[string]v1.VolumeStatus
	DisksInfo             map[string]*cmdv1.DiskInfo
	SMBios                *cmdv1.SMBios
	SRIOVDevices          []api.HostDevice
	LegacyHostDevices     []api.HostDevice
	GenericHostDevices    []api.HostDevice
	GPUHostDevices        []api.HostDevice
	EFIConfiguration      *EFIConfiguration
	MemBalloonStatsPeriod uint
	UseVirtioTransitional bool
	EphemeraldiskCreator  ephemeraldisk.EphemeralDiskCreatorInterface
	VolumesDiscardIgnore  []string
	Topology              *cmdv1.Topology
	CpuScheduler          *api.VCPUScheduler
	ExpandDisksEnabled    bool
}

func contains(volumes []string, name string) bool {
	for _, v := range volumes {
		if name == v {
			return true
		}
	}
	return false
}

func isAMD64(arch string) bool {
	if arch == "amd64" {
		return true
	}
	return false
}

func isPPC64(arch string) bool {
	if arch == "ppc64le" {
		return true
	}
	return false
}

func isARM64(arch string) bool {
	if arch == "arm64" {
		return true
	}
	return false
}

func Convert_v1_Disk_To_api_Disk(c *ConverterContext, diskDevice *v1.Disk, disk *api.Disk, prefixMap map[string]deviceNamer, numQueues *uint, volumeStatusMap map[string]v1.VolumeStatus) error {
	if diskDevice.Disk != nil {
		var unit int
		disk.Device = "disk"
		disk.Target.Bus = diskDevice.Disk.Bus
		if diskDevice.Disk.Bus == "scsi" {
			// Ensure we assign this disk to the correct scsi controller
			if disk.Address == nil {
				disk.Address = &api.Address{}
			}
			disk.Address.Type = "drive"
			// This should be the index of the virtio-scsi controller, which is hard coded to 0
			disk.Address.Controller = "0"
			disk.Address.Bus = "0"
		}
		disk.Target.Device, unit = makeDeviceName(diskDevice.Name, diskDevice.Disk.Bus, prefixMap)
		if diskDevice.Disk.Bus == "scsi" {
			disk.Address.Unit = strconv.Itoa(unit)
		}
		if diskDevice.Disk.PciAddress != "" {
			if diskDevice.Disk.Bus != "virtio" {
				return fmt.Errorf("setting a pci address is not allowed for non-virtio bus types, for disk %s", diskDevice.Name)
			}
			addr, err := device.NewPciAddressField(diskDevice.Disk.PciAddress)
			if err != nil {
				return fmt.Errorf("failed to configure disk %s: %v", diskDevice.Name, err)
			}
			disk.Address = addr
		}
		if diskDevice.Disk.Bus == "virtio" {
			disk.Model = translateModel(c, "virtio")
		}
		disk.ReadOnly = toApiReadOnly(diskDevice.Disk.ReadOnly)
		disk.Serial = diskDevice.Serial
	} else if diskDevice.LUN != nil {
		disk.Device = "lun"
		disk.Target.Bus = diskDevice.LUN.Bus
		disk.Target.Device, _ = makeDeviceName(diskDevice.Name, diskDevice.LUN.Bus, prefixMap)
		disk.ReadOnly = toApiReadOnly(diskDevice.LUN.ReadOnly)
	} else if diskDevice.Floppy != nil {
		disk.Device = "floppy"
		disk.Target.Bus = "fdc"
		disk.Target.Tray = string(diskDevice.Floppy.Tray)
		disk.Target.Device, _ = makeDeviceName(diskDevice.Name, disk.Target.Bus, prefixMap)
		disk.ReadOnly = toApiReadOnly(diskDevice.Floppy.ReadOnly)
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
		Name:        "qemu",
		Cache:       string(diskDevice.Cache),
		IO:          diskDevice.IO,
		ErrorPolicy: "stop",
	}
	if diskDevice.Disk != nil || diskDevice.LUN != nil {
		if !contains(c.VolumesDiscardIgnore, diskDevice.Name) {
			disk.Driver.Discard = "unmap"
		}
		volumeStatus, ok := volumeStatusMap[diskDevice.Name]
		if ok && volumeStatus.PersistentVolumeClaimInfo != nil {
			disk.FilesystemOverhead = volumeStatus.PersistentVolumeClaimInfo.FilesystemOverhead
			capacity, ok := volumeStatus.PersistentVolumeClaimInfo.Capacity[k8sv1.ResourceStorage]
			if ok {
				disk.Capacity = &capacity
			}
		}
		disk.ExpandDisksEnabled = c.ExpandDisksEnabled
	}
	if numQueues != nil && disk.Target.Bus == "virtio" {
		disk.Driver.Queues = numQueues
	}
	disk.Alias = api.NewUserDefinedAlias(diskDevice.Name)
	if diskDevice.BootOrder != nil {
		disk.BootOrder = &api.BootOrder{Order: *diskDevice.BootOrder}
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
	if _, err := os.Stat(path); os.IsNotExist(err) {
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

// getOptimalBlockIOForDevice determines the optimal sizes based on the physical device properties.
func getOptimalBlockIOForDevice(path string) (*api.BlockIO, error) {
	f, err := os.OpenFile(path, syscall.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("unable to open device %v: %v", path, err)
	}
	defer util.CloseIOAndCheckErr(f, nil)

	logicalSize, err := unix.IoctlGetInt(int(f.Fd()), unix.BLKSSZGET)
	if err != nil {
		return nil, fmt.Errorf("unable to get logical block size from device %v: %v", path, err)
	}
	physicalSize, err := unix.IoctlGetInt(int(f.Fd()), unix.BLKBSZGET)
	if err != nil {
		return nil, fmt.Errorf("unable to get physical block size from device %v: %v", path, err)
	}

	log.Log.Infof("Detected logical size of %d and physical size of %d for device %v", logicalSize, physicalSize, path)

	if logicalSize == 0 && physicalSize == 0 {
		return nil, fmt.Errorf("block sizes returned by device %v are 0", path)
	}
	blockIO := &api.BlockIO{
		LogicalBlockSize:  uint(logicalSize),
		PhysicalBlockSize: uint(physicalSize),
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
	return blockIO, nil
}

// getOptimalBlockIOForFile determines the optimal sizes based on the filesystem settings
// the VM's disk image is residing on. A filesystem does not differentiate between sizes.
// The physical size will always match the logical size. The rest is up to the filesystem.
func getOptimalBlockIOForFile(path string) (*api.BlockIO, error) {
	var statfs syscall.Statfs_t
	err := syscall.Statfs(path, &statfs)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %v: %v", path, err)
	}
	return &api.BlockIO{
		LogicalBlockSize:  uint(statfs.Bsize),
		PhysicalBlockSize: uint(statfs.Bsize),
	}, nil
}

func SetDriverCacheMode(disk *api.Disk, directIOChecker DirectIOChecker) error {
	var path string
	var err error
	supportDirectIO := true
	mode := v1.DriverCache(disk.Driver.Cache)
	isBlockDev := false

	if disk.Source.File != "" {
		path = disk.Source.File
	} else if disk.Source.Dev != "" {
		path = disk.Source.Dev
		isBlockDev = true
	} else {
		return fmt.Errorf("Unable to set a driver cache mode, disk is neither a block device nor a file")
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
	diskInf, err := GetImageInfo(path)
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

func makeDeviceName(diskName, bus string, prefixMap map[string]deviceNamer) (string, int) {
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
		Type: "virtio",
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
		return Convert_v1_HostDisk_To_api_Disk(source.Name, source.HostDisk.Path, disk)
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

func Convert_v1_Config_To_api_Disk(volumeName string, disk *api.Disk, configType config.Type) error {
	disk.Type = "file"
	disk.Driver.Type = "raw"
	disk.Driver.ErrorPolicy = "stop"
	switch configType {
	case config.ConfigMap:
		disk.Source.File = config.GetConfigMapDiskPath(volumeName)
		break
	case config.Secret:
		disk.Source.File = config.GetSecretDiskPath(volumeName)
		break
	case config.DownwardAPI:
		disk.Source.File = config.GetDownwardAPIDiskPath(volumeName)
		break
	case config.ServiceAccount:
		disk.Source.File = config.GetServiceAccountDiskPath()
		break
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

func Convert_v1_PersistentVolumeClaim_To_api_Disk(name string, disk *api.Disk, c *ConverterContext) error {
	if c.IsBlockPVC[name] {
		return Convert_v1_BlockVolumeSource_To_api_Disk(name, disk, c.VolumesDiscardIgnore)
	}
	return Convert_v1_FilesystemVolumeSource_To_api_Disk(name, disk, c.VolumesDiscardIgnore)
}

// Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk converts a Hotplugged PVC to an api disk
func Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk(name string, disk *api.Disk, c *ConverterContext) error {
	if c.IsBlockPVC[name] {
		return Convert_v1_Hotplug_BlockVolumeSource_To_api_Disk(name, disk, c.VolumesDiscardIgnore)
	}
	return Convert_v1_Hotplug_FilesystemVolumeSource_To_api_Disk(name, disk, c.VolumesDiscardIgnore)
}

func Convert_v1_DataVolume_To_api_Disk(name string, disk *api.Disk, c *ConverterContext) error {
	if c.IsBlockDV[name] {
		return Convert_v1_BlockVolumeSource_To_api_Disk(name, disk, c.VolumesDiscardIgnore)
	}
	return Convert_v1_FilesystemVolumeSource_To_api_Disk(name, disk, c.VolumesDiscardIgnore)
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
	disk.Driver.Type = "raw"
	disk.Driver.ErrorPolicy = "stop"
	disk.Source.File = GetFilesystemVolumePath(volumeName)
	if !contains(volumesDiscardIgnore, volumeName) {
		disk.Driver.Discard = "unmap"
	}
	return nil
}

// Convert_v1_Hotplug_FilesystemVolumeSource_To_api_Disk takes a FS source and builds the KVM Disk representation
func Convert_v1_Hotplug_FilesystemVolumeSource_To_api_Disk(volumeName string, disk *api.Disk, volumesDiscardIgnore []string) error {
	disk.Type = "file"
	disk.Driver.Type = "raw"
	disk.Driver.ErrorPolicy = "stop"
	if !contains(volumesDiscardIgnore, volumeName) {
		disk.Driver.Discard = "unmap"
	}
	disk.Source.File = GetHotplugFilesystemVolumePath(volumeName)
	return nil
}

func Convert_v1_BlockVolumeSource_To_api_Disk(volumeName string, disk *api.Disk, volumesDiscardIgnore []string) error {
	disk.Type = "block"
	disk.Driver.Type = "raw"
	disk.Driver.ErrorPolicy = "stop"
	if !contains(volumesDiscardIgnore, volumeName) {
		disk.Driver.Discard = "unmap"
	}
	disk.Source.Name = volumeName
	disk.Source.Dev = GetBlockDeviceVolumePath(volumeName)
	return nil
}

// Convert_v1_Hotplug_BlockVolumeSource_To_api_Disk takes a block device source and builds the domain Disk representation
func Convert_v1_Hotplug_BlockVolumeSource_To_api_Disk(volumeName string, disk *api.Disk, volumesDiscardIgnore []string) error {
	disk.Type = "block"
	disk.Driver.Type = "raw"
	disk.Driver.ErrorPolicy = "stop"
	if !contains(volumesDiscardIgnore, volumeName) {
		disk.Driver.Discard = "unmap"
	}
	disk.Source.Dev = GetHotplugBlockDeviceVolumePath(volumeName)
	return nil
}

func Convert_v1_HostDisk_To_api_Disk(volumeName string, path string, disk *api.Disk) error {
	disk.Type = "file"
	disk.Driver.Type = "raw"
	disk.Driver.ErrorPolicy = "stop"
	disk.Source.File = hostdisk.GetMountedHostDiskPath(volumeName, path)
	return nil
}

func Convert_v1_SysprepSource_To_api_Disk(volumeName string, disk *api.Disk) error {
	if disk.Type == "lun" {
		return fmt.Errorf("device %s is of type lun. Not compatible with a file based disk", disk.Alias.GetName())
	}

	disk.Source.File = config.GetSysprepDiskPath(volumeName)
	disk.Type = "file"
	disk.Driver.Type = "raw"
	return nil
}

func Convert_v1_CloudInitSource_To_api_Disk(source v1.VolumeSource, disk *api.Disk, c *ConverterContext) error {
	if disk.Type == "lun" {
		return fmt.Errorf("device %s is of type lun. Not compatible with a file based disk", disk.Alias.GetName())
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
	disk.Driver.Type = "raw"
	disk.Driver.ErrorPolicy = "stop"
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
	disk.Model = translateModel(c, "virtio")
	disk.Source = api.DiskSource{
		File: config.DownwardMetricDisk,
	}
	return nil
}

func Convert_v1_EmptyDiskSource_To_api_Disk(volumeName string, _ *v1.EmptyDiskSource, disk *api.Disk) error {
	if disk.Type == "lun" {
		return fmt.Errorf("device %s is of type lun. Not compatible with a file based disk", disk.Alias.GetName())
	}

	disk.Type = "file"
	disk.Driver.Type = "qcow2"
	disk.Driver.Discard = "unmap"
	disk.Source.File = emptydisk.NewEmptyDiskCreator().FilePathForVolumeName(volumeName)
	disk.Driver.ErrorPolicy = "stop"

	return nil
}

func Convert_v1_ContainerDiskSource_To_api_Disk(volumeName string, _ *v1.ContainerDiskSource, disk *api.Disk, c *ConverterContext, diskIndex int) error {
	if disk.Type == "lun" {
		return fmt.Errorf("device %s is of type lun. Not compatible with a file based disk", disk.Alias.GetName())
	}
	disk.Type = "file"
	disk.Driver.Type = "qcow2"
	disk.Driver.ErrorPolicy = "stop"
	disk.Driver.Discard = "unmap"
	disk.Source.File = c.EphemeraldiskCreator.GetFilePath(volumeName)
	disk.BackingStore = &api.BackingStore{
		Format: &api.BackingStoreFormat{},
		Source: &api.DiskSource{},
	}

	source := containerdisk.GetDiskTargetPathFromLauncherView(diskIndex)
	if info, _ := c.DisksInfo[volumeName]; info != nil {
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
	disk.Driver.Type = "qcow2"
	disk.Driver.ErrorPolicy = "stop"
	disk.Driver.Discard = "unmap"
	disk.Source.File = c.EphemeraldiskCreator.GetFilePath(volumeName)
	disk.BackingStore = &api.BackingStore{
		Format: &api.BackingStoreFormat{},
		Source: &api.DiskSource{},
	}
	if !contains(c.VolumesDiscardIgnore, volumeName) {
		disk.Driver.Discard = "unmap"
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

func Convert_v1_Watchdog_To_api_Watchdog(source *v1.Watchdog, watchdog *api.Watchdog, _ *ConverterContext) error {
	watchdog.Alias = api.NewUserDefinedAlias(source.Name)
	if source.I6300ESB != nil {
		watchdog.Model = "i6300esb"
		watchdog.Action = string(source.I6300ESB.Action)
		return nil
	}
	return fmt.Errorf("watchdog %s can't be mapped, no watchdog type specified", source.Name)
}

func Convert_v1_Rng_To_api_Rng(_ *v1.Rng, rng *api.Rng, c *ConverterContext) error {

	// default rng model for KVM/QEMU virtualization
	rng.Model = translateModel(c, "virtio")

	// default backend model, random
	rng.Backend = &api.RngBackend{
		Model: "random",
	}

	// the default source for rng is dev urandom
	rng.Backend.Source = "/dev/urandom"

	return nil
}

func Convert_v1_Usbredir_To_api_Usbredir(vmi *v1.VirtualMachineInstance, domainDevices *api.Devices, _ *ConverterContext) (bool, error) {
	clientDevices := vmi.Spec.Domain.Devices.ClientPassthrough

	// Default is to have USB Redirection disabled
	if clientDevices == nil {
		return false, nil
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
	return true, nil
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
	return
}

func Convert_v1_Input_To_api_InputDevice(input *v1.Input, inputDevice *api.Input) error {
	if input.Bus != "virtio" && input.Bus != "usb" && input.Bus != "" {
		return fmt.Errorf("input contains unsupported bus %s", input.Bus)
	}

	if input.Bus != "virtio" && input.Bus != "usb" {
		input.Bus = "usb"
	}

	if input.Type != "tablet" {
		return fmt.Errorf("input contains unsupported type %s", input.Type)
	}

	inputDevice.Bus = input.Bus
	inputDevice.Type = input.Type
	inputDevice.Alias = api.NewUserDefinedAlias(input.Name)

	if input.Bus == "virtio" {
		inputDevice.Model = "virtio"
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
	if source != nil && source.AutoattachMemBalloon != nil && *source.AutoattachMemBalloon == false {
		ballooning.Model = "none"
		ballooning.Stats = nil
	} else {
		ballooning.Model = translateModel(c, "virtio")
		if c.MemBalloonStatsPeriod != 0 {
			ballooning.Stats = &api.Stats{Period: c.MemBalloonStatsPeriod}
		}

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

func Convert_v1_VirtualMachineInstance_To_api_Domain(vmi *v1.VirtualMachineInstance, domain *api.Domain, c *ConverterContext) (err error) {
	precond.MustNotBeNil(vmi)
	precond.MustNotBeNil(domain)
	precond.MustNotBeNil(c)

	domain.Spec.Name = api.VMINamespaceKeyFunc(vmi)
	domain.ObjectMeta.Name = vmi.ObjectMeta.Name
	domain.ObjectMeta.Namespace = vmi.ObjectMeta.Namespace

	// Set VM CPU cores
	// CPU topology will be created everytime, because user can specify
	// number of cores in vmi.Spec.Domain.Resources.Requests/Limits, not only
	// in vmi.Spec.Domain.CPU
	cpuTopology := getCPUTopology(vmi)
	cpuCount := vcpu.CalculateRequestedVCPUs(cpuTopology)
	domain.Spec.CPU.Topology = cpuTopology
	domain.Spec.VCPU = &api.VCPU{
		Placement: "static",
		CPUs:      cpuCount,
	}

	kvmPath := "/dev/kvm"
	if softwareEmulation, err := util.UseSoftwareEmulationForDevice(kvmPath, c.AllowEmulation); err != nil {
		return err
	} else if softwareEmulation {
		logger := log.DefaultLogger()
		logger.Infof("Hardware emulation device '%s' not present. Using software emulation.", kvmPath)
		domain.Spec.Type = "qemu"
	} else if _, err := os.Stat(kvmPath); os.IsNotExist(err) {
		return fmt.Errorf("hardware emulation device '%s' not present", kvmPath)
	} else if err != nil {
		return err
	}

	virtioNetProhibited := false
	vhostNetPath := "/dev/vhost-net"
	if softwareEmulation, err := util.UseSoftwareEmulationForDevice(vhostNetPath, c.AllowEmulation); err != nil {
		return err
	} else if softwareEmulation {
		logger := log.DefaultLogger()
		logger.Infof("In-kernel virtio-net device emulation '%s' not present. Falling back to QEMU userland emulation.", vhostNetPath)
	} else if _, err := os.Stat(vhostNetPath); os.IsNotExist(err) {
		virtioNetProhibited = true
	} else if err != nil {
		return err
	}

	// Spec metadata

	newChannel := Add_Agent_To_api_Channel()
	domain.Spec.Devices.Channels = append(domain.Spec.Devices.Channels, newChannel)

	domain.Spec.Metadata.KubeVirt.UID = vmi.UID
	gracePeriodSeconds := v1.DefaultGracePeriodSeconds
	if vmi.Spec.TerminationGracePeriodSeconds != nil {
		gracePeriodSeconds = *vmi.Spec.TerminationGracePeriodSeconds
	}
	domain.Spec.Metadata.KubeVirt.GracePeriod = &api.GracePeriodMetadata{
		DeletionGracePeriodSeconds: gracePeriodSeconds,
	}

	domain.Spec.SysInfo = &api.SysInfo{}
	if vmi.Spec.Domain.Firmware != nil {
		domain.Spec.SysInfo.System = []api.Entry{
			{
				Name:  "uuid",
				Value: string(vmi.Spec.Domain.Firmware.UUID),
			},
		}

		if vmi.Spec.Domain.Firmware.Bootloader != nil && vmi.Spec.Domain.Firmware.Bootloader.EFI != nil {
			domain.Spec.OS.BootLoader = &api.Loader{
				Path:     c.EFIConfiguration.EFICode,
				ReadOnly: "yes",
				Secure:   boolToYesNo(&c.EFIConfiguration.SecureLoader, false),
				Type:     "pflash",
			}

			domain.Spec.OS.NVRam = &api.NVRam{
				NVRam:    filepath.Join("/tmp", domain.Spec.Name),
				Template: c.EFIConfiguration.EFIVars,
			}
		}

		if vmi.Spec.Domain.Firmware.Bootloader != nil && vmi.Spec.Domain.Firmware.Bootloader.BIOS != nil {
			if vmi.Spec.Domain.Firmware.Bootloader.BIOS.UseSerial != nil && *vmi.Spec.Domain.Firmware.Bootloader.BIOS.UseSerial {
				domain.Spec.OS.BIOS = &api.BIOS{
					UseSerial: "yes",
				}
			}
		}

		if len(vmi.Spec.Domain.Firmware.Serial) > 0 {
			domain.Spec.SysInfo.System = append(domain.Spec.SysInfo.System, api.Entry{Name: "serial", Value: string(vmi.Spec.Domain.Firmware.Serial)})
		}

		if util.HasKernelBootContainerImage(vmi) {
			kb := vmi.Spec.Domain.Firmware.KernelBoot

			log.Log.Object(vmi).Infof("kernel boot defined for VMI. Converting to domain XML")
			if kb.Container.KernelPath != "" {
				kernelPath := containerdisk.GetKernelBootArtifactPathFromLauncherView(kb.Container.KernelPath)
				log.Log.Object(vmi).Infof("setting kernel path for kernel boot: " + kernelPath)
				domain.Spec.OS.Kernel = kernelPath
			}

			if kb.Container.InitrdPath != "" {
				initrdPath := containerdisk.GetKernelBootArtifactPathFromLauncherView(kb.Container.InitrdPath)
				log.Log.Object(vmi).Infof("setting initrd path for kernel boot: " + initrdPath)
				domain.Spec.OS.Initrd = initrdPath
			}

		}

		// Define custom command-line arguments even if kernel-boot container is not defined
		if f := vmi.Spec.Domain.Firmware; f != nil && f.KernelBoot != nil {
			log.Log.Object(vmi).Infof("setting custom kernel arguments: " + f.KernelBoot.KernelArgs)
			domain.Spec.OS.KernelArgs = f.KernelBoot.KernelArgs
		}

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
	// SMBios option does not work in Power, attempting to set it will result in the following error message:
	// "Option not supported for this target" issued by qemu-system-ppc64, so don't set it in case GOARCH is ppc64le
	// ARM64 use UEFI boot by default, set SMBios is unnecessory.
	if isAMD64(c.Architecture) {
		domain.Spec.OS.SMBios = &api.SMBios{
			Mode: "sysinfo",
		}
	}

	if vmi.Spec.Domain.Chassis != nil {
		domain.Spec.SysInfo.Chassis = []api.Entry{
			{
				Name:  "manufacturer",
				Value: string(vmi.Spec.Domain.Chassis.Manufacturer),
			},
			{
				Name:  "version",
				Value: string(vmi.Spec.Domain.Chassis.Version),
			},
			{
				Name:  "serial",
				Value: string(vmi.Spec.Domain.Chassis.Serial),
			},
			{
				Name:  "asset",
				Value: string(vmi.Spec.Domain.Chassis.Asset),
			},
			{
				Name:  "sku",
				Value: string(vmi.Spec.Domain.Chassis.Sku),
			},
		}
	}

	if domain.Spec.Memory, err = QuantityToByte(*getVirtualMemory(vmi)); err != nil {
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
		domain.Spec.CPU.NUMA = &api.NUMA{
			Cells: []api.NUMACell{
				{
					ID:     "0",
					CPUs:   fmt.Sprintf("0-%d", domain.Spec.VCPU.CPUs-1),
					Memory: uint64(getVirtualMemory(vmi).Value() / int64(1024)),
					Unit:   "KiB",
				},
			},
		}
	}

	volumeIndices := map[string]int{}
	volumes := map[string]*v1.Volume{}
	for i, volume := range vmi.Spec.Volumes {
		volumes[volume.Name] = volume.DeepCopy()
		volumeIndices[volume.Name] = i
	}

	dedicatedThreads := 0
	autoThreads := 0
	useIOThreads := false
	threadPoolLimit := 1

	if vmi.Spec.Domain.IOThreadsPolicy != nil {
		useIOThreads = true

		if (*vmi.Spec.Domain.IOThreadsPolicy) == v1.IOThreadsPolicyAuto {
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
		}
	}
	for _, diskDevice := range vmi.Spec.Domain.Devices.Disks {
		dedicatedThread := false
		if diskDevice.DedicatedIOThread != nil {
			dedicatedThread = *diskDevice.DedicatedIOThread
		}
		if dedicatedThread {
			useIOThreads = true
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

	ioThreadCount := (autoThreads + dedicatedThreads)
	if ioThreadCount != 0 {
		if domain.Spec.IOThreads == nil {
			domain.Spec.IOThreads = &api.IOThreads{}
		}
		domain.Spec.IOThreads.IOThreads = uint(ioThreadCount)
	}

	currentAutoThread := defaultIOThread
	currentDedicatedThread := uint(autoThreads + 1)

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

		err := Convert_v1_Disk_To_api_Disk(c, &disk, &newDisk, prefixMap, numBlkQueues, volumeStatusMap)
		if err != nil {
			return err
		}
		volume := volumes[disk.Name]
		if volume == nil {
			return fmt.Errorf("No matching volume with name %s found", disk.Name)
		}

		if _, ok := c.HotplugVolumes[disk.Name]; !ok {
			err = Convert_v1_Volume_To_api_Disk(volume, &newDisk, c, volumeIndices[disk.Name])
		} else {
			err = Convert_v1_Hotplug_Volume_To_api_Disk(volume, &newDisk, c)
		}
		if err != nil {
			return err
		}

		if err := Convert_v1_BlockSize_To_api_BlockIO(&disk, &newDisk); err != nil {
			return err
		}

		if useIOThreads {
			if _, ok := c.HotplugVolumes[disk.Name]; !ok {
				ioThreadId := defaultIOThread
				dedicatedThread := false
				if disk.DedicatedIOThread != nil {
					dedicatedThread = *disk.DedicatedIOThread
				}

				if dedicatedThread {
					ioThreadId = currentDedicatedThread
					currentDedicatedThread += 1
				} else {
					ioThreadId = currentAutoThread
					// increment the threadId to be used next but wrap around at the thread limit
					// the odd math here is because thread ID's start at 1, not 0
					currentAutoThread = (currentAutoThread % uint(autoThreads)) + 1
				}
				newDisk.Driver.IOThread = &ioThreadId
			} else {
				newDisk.Driver.IO = v1.IOThreads
			}
		}

		hpStatus, hpOk := c.HotplugVolumes[disk.Name]
		// if len(c.PermanentVolumes) == 0, it means the vmi is not ready yet, add all disks
		if _, ok := c.PermanentVolumes[disk.Name]; ok || len(c.PermanentVolumes) == 0 || (hpOk && (hpStatus.Phase == v1.HotplugVolumeMounted || hpStatus.Phase == v1.VolumeReady)) {
			domain.Spec.Devices.Disks = append(domain.Spec.Devices.Disks, newDisk)
		}
	}
	// Handle virtioFS
	for _, fs := range vmi.Spec.Domain.Devices.Filesystems {
		if fs.Virtiofs != nil {
			newFS := api.FilesystemDevice{}

			newFS.Type = "mount"
			newFS.AccessMode = "passthrough"
			newFS.Driver = &api.FilesystemDriver{
				Type:  "virtiofs",
				Queue: "1024",
			}
			newFS.Binary = &api.FilesystemBinary{
				Path:  "/usr/libexec/virtiofsd",
				Xattr: "on",
				Cache: &api.FilesystemBinaryCache{
					Mode: "none",
				},
				Lock: &api.FilesystemBinaryLock{
					Posix: "on",
					Flock: "on",
				},
			}
			newFS.Target = &api.FilesystemTarget{
				Dir: fs.Name,
			}

			volume := volumes[fs.Name]
			if volume == nil {
				return fmt.Errorf("No matching volume with name %s found", fs.Name)
			}
			volDir, _ := filepath.Split(GetFilesystemVolumePath(volume.Name))
			newFS.Source = &api.FilesystemSource{}
			newFS.Source.Dir = volDir
			domain.Spec.Devices.Filesystems = append(domain.Spec.Devices.Filesystems, newFS)
		}
	}

	Convert_v1_Sound_To_api_Sound(vmi, &domain.Spec.Devices, c)

	if vmi.Spec.Domain.Devices.Watchdog != nil {
		newWatchdog := &api.Watchdog{}
		err := Convert_v1_Watchdog_To_api_Watchdog(vmi.Spec.Domain.Devices.Watchdog, newWatchdog, c)
		if err != nil {
			return err
		}
		domain.Spec.Devices.Watchdog = newWatchdog
	}

	if vmi.Spec.Domain.Devices.Rng != nil {
		newRng := &api.Rng{}
		err := Convert_v1_Rng_To_api_Rng(vmi.Spec.Domain.Devices.Rng, newRng, c)
		if err != nil {
			return err
		}
		domain.Spec.Devices.Rng = newRng
	}

	isUSBDevicePresent := false
	if vmi.Spec.Domain.Devices.Inputs != nil {
		inputDevices := make([]api.Input, 0)
		for i := range vmi.Spec.Domain.Devices.Inputs {
			inputDevice := api.Input{}
			err := Convert_v1_Input_To_api_InputDevice(&vmi.Spec.Domain.Devices.Inputs[i], &inputDevice)
			if err != nil {
				return err
			}
			inputDevices = append(inputDevices, inputDevice)
			if inputDevice.Bus == "usb" {
				isUSBDevicePresent = true
			}
		}
		domain.Spec.Devices.Inputs = inputDevices
	}

	isUSBRedirEnabled, err := Convert_v1_Usbredir_To_api_Usbredir(vmi, &domain.Spec.Devices, c)
	if err != nil {
		return err
	}

	domain.Spec.Devices.Ballooning = &api.MemBalloon{}
	ConvertV1ToAPIBalloning(&vmi.Spec.Domain.Devices, domain.Spec.Devices.Ballooning, c)

	//usb controller is turned on, only when user specify input device with usb bus,
	//otherwise it is turned off
	//In ppc64le usb devices like mouse / keyboard are set by default,
	//so we can't disable the controller otherwise we run into the following error:
	//"unsupported configuration: USB is disabled for this domain, but USB devices are present in the domain XML"
	if !isUSBDevicePresent && !isUSBRedirEnabled && isAMD64(c.Architecture) {
		// disable usb controller
		domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, api.Controller{
			Type:  "usb",
			Index: "0",
			Model: "none",
		})
	} else {
		domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, api.Controller{
			Type:  "usb",
			Index: "0",
			Model: "qemu-xhci",
		})
	}

	if needsSCSIControler(vmi) {
		scsiController := api.Controller{
			Type:  "scsi",
			Index: "0",
			Model: translateModel(c, "virtio"),
		}
		if useIOThreads {
			scsiController.Driver = &api.ControllerDriver{
				IOThread: &currentAutoThread,
				Queues:   &vcpus,
			}
		}
		domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, scsiController)
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
		if vmi.Spec.Domain.CPU.Features != nil {
			for _, feature := range vmi.Spec.Domain.CPU.Features {
				domain.Spec.CPU.Features = append(domain.Spec.CPU.Features, api.CPUFeature{
					Name:   feature.Name,
					Policy: feature.Policy,
				})
			}
		}

		// Make use of the tsc frequency topology hint
		if topology.VMIHasInvTSCFeature(vmi) && vmi.Status.TopologyHints != nil && vmi.Status.TopologyHints.TSCFrequency != nil {
			freq := *vmi.Status.TopologyHints.TSCFrequency
			clock := domain.Spec.Clock
			if clock == nil {
				clock = &api.Clock{}
			}
			clock.Timer = append(clock.Timer, api.Timer{Name: "tsc", Frequency: strconv.FormatInt(freq, 10)})
			domain.Spec.Clock = clock
		}

		// Adjust guest vcpu config. Currently will handle vCPUs to pCPUs pinning
		if vmi.IsCPUDedicated() {
			var cpuPool vcpu.VCPUPool
			if isNumaPassthrough(vmi) {
				cpuPool = vcpu.NewStrictCPUPool(domain.Spec.CPU.Topology, c.Topology, c.CPUSet)
			} else {
				cpuPool = vcpu.NewRelaxedCPUPool(domain.Spec.CPU.Topology, c.Topology, c.CPUSet)
			}
			cpuTune, err := cpuPool.FitCores()
			if err != nil {
				log.Log.Reason(err).Error("failed to format domain cputune.")
				return err
			}
			domain.Spec.CPUTune = cpuTune

			// always add the hint-dedicated feature when dedicatedCPUs are requested.
			if domain.Spec.Features.KVM == nil {
				domain.Spec.Features.KVM = &api.FeatureKVM{}
			}
			domain.Spec.Features.KVM.HintDedicated = &api.FeatureState{
				State: "on",
			}

			var emulatorThread uint32
			if vmi.Spec.Domain.CPU.IsolateEmulatorThread {
				emulatorThread, err = cpuPool.FitThread()
				if err != nil {
					e := fmt.Errorf("no CPU allocated for the emulation thread: %v", err)
					log.Log.Reason(e).Error("failed to format emulation thread pin")
					return e
				}
				appendDomainEmulatorThreadPin(domain, emulatorThread)
			}
			if useIOThreads {
				if err := formatDomainIOThreadPin(vmi, domain, emulatorThread, c); err != nil {
					log.Log.Reason(err).Error("failed to format domain iothread pinning.")
					return err
				}
			}
			if vmi.IsRealtimeEnabled() {
				// RT settings
				// To be configured by manifest
				// - CPU Model: Host Passthrough
				// - VCPU (placement type and number)
				// - VCPU Pin (DedicatedCPUPlacement)
				// - USB controller should be disabled if no input type usb is found
				// - Memballoning can be disabled when setting 'autoattachMemBalloon' to false
				formatVCPUScheduler(domain, vmi)
				domain.Spec.Features.PMU = &api.FeatureState{State: "off"}
			}

			if isNumaPassthrough(vmi) {
				if err := numaMapping(vmi, &domain.Spec, c.Topology); err != nil {
					log.Log.Reason(err).Error("failed to calculate passed through NUMA topology.")
					return err
				}
			}
		}
	}

	domain.Spec.Devices.HostDevices = append(domain.Spec.Devices.HostDevices, c.GenericHostDevices...)
	domain.Spec.Devices.HostDevices = append(domain.Spec.Devices.HostDevices, c.GPUHostDevices...)

	// This is needed to support a legacy approach to device assignment
	// Append HostDevices to DomXML if GPU is requested
	if util.IsGPUVMI(vmi) {
		domain.Spec.Devices.HostDevices = append(domain.Spec.Devices.HostDevices, c.LegacyHostDevices...)
	}

	if vmi.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU.Model == "" {
		domain.Spec.CPU.Mode = v1.CPUModeHostModel
	}

	if vmi.Spec.Domain.Devices.AutoattachSerialConsole == nil || *vmi.Spec.Domain.Devices.AutoattachSerialConsole == true {
		// Add mandatory console device
		domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, api.Controller{
			Type:  "virtio-serial",
			Index: "0",
			Model: translateModel(c, "virtio"),
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

		domain.Spec.Devices.Serials = []api.Serial{
			{
				Type: "unix",
				Target: &api.SerialTarget{
					Port: &serialPort,
				},
				Source: &api.SerialSource{
					Mode: "bind",
					Path: fmt.Sprintf("/var/run/kubevirt-private/%s/virt-serial%d", vmi.ObjectMeta.UID, serialPort),
				},
			},
		}
	}

	if vmi.Spec.Domain.Devices.AutoattachGraphicsDevice == nil || *vmi.Spec.Domain.Devices.AutoattachGraphicsDevice == true {
		var heads uint = 1
		var vram uint = 16384
		// For arm64, qemu-kvm only support virtio-gpu display device, so set it as default video device.
		// tablet and keyboard devices are necessary for control the VM via vnc connection
		if isARM64(c.Architecture) {
			domain.Spec.Devices.Video = []api.Video{
				{
					Model: api.VideoModel{
						Type:  "virtio",
						Heads: &heads,
					},
				},
			}

			if !hasTabletDevice(vmi) {
				domain.Spec.Devices.Inputs = append(domain.Spec.Devices.Inputs,
					api.Input{
						Bus:  "usb",
						Type: "tablet",
					},
				)
			}

			domain.Spec.Devices.Inputs = append(domain.Spec.Devices.Inputs,
				api.Input{
					Bus:  "usb",
					Type: "keyboard",
				},
			)
		} else {
			domain.Spec.Devices.Video = []api.Video{
				{
					Model: api.VideoModel{
						Type:  "vga",
						Heads: &heads,
						VRam:  &vram,
					},
				},
			}
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

	domainInterfaces, err := createDomainInterfaces(vmi, domain, c, virtioNetProhibited)
	if err != nil {
		return err
	}
	domain.Spec.Devices.Interfaces = append(domain.Spec.Devices.Interfaces, domainInterfaces...)
	domain.Spec.Devices.HostDevices = append(domain.Spec.Devices.HostDevices, c.SRIOVDevices...)

	// Add Ignition Command Line if present
	ignitiondata, _ := vmi.Annotations[v1.IgnitionAnnotation]
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

	if virtLauncherLogVerbosity, err := strconv.Atoi(os.Getenv(services.ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY)); err == nil && (virtLauncherLogVerbosity > services.EXT_LOG_VERBOSITY_THRESHOLD) {

		initializeQEMUCmdAndQEMUArg(domain)

		domain.Spec.QEMUCmd.QEMUArg = append(domain.Spec.QEMUCmd.QEMUArg,
			api.Arg{Value: "-chardev"},
			api.Arg{Value: fmt.Sprintf("file,id=firmwarelog,path=%s", QEMUSeaBiosDebugPipe)},
			api.Arg{Value: "-device"},
			api.Arg{Value: "isa-debugcon,iobase=0x402,chardev=firmwarelog"})
	}

	return nil
}

func getVirtualMemory(vmi *v1.VirtualMachineInstance) *resource.Quantity {
	// In case that guest memory is explicitly set, return it
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Guest != nil {
		return vmi.Spec.Domain.Memory.Guest
	}

	// Otherwise, take memory from the memory-limit, if set
	if v, ok := vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceMemory]; ok {
		return &v
	}

	// Otherwise, take memory from the requested memory
	v, _ := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]
	return &v
}

func getCPUTopology(vmi *v1.VirtualMachineInstance) *api.CPUTopology {
	cores := uint32(1)
	threads := uint32(1)
	sockets := uint32(1)
	vmiCPU := vmi.Spec.Domain.CPU
	if vmiCPU != nil {
		if vmiCPU.Cores != 0 {
			cores = vmiCPU.Cores
		}

		if vmiCPU.Threads != 0 {
			threads = vmiCPU.Threads
		}

		if vmiCPU.Sockets != 0 {
			sockets = vmiCPU.Sockets
		}
	}
	// A default guest CPU topology is being set in API mutator webhook, if nothing provided by a user.
	// However this setting is still required to handle situations when the webhook fails to set a default topology.
	if vmiCPU == nil || (vmiCPU.Cores == 0 && vmiCPU.Sockets == 0 && vmiCPU.Threads == 0) {
		//if cores, sockets, threads are not set, take value from domain resources request or limits and
		//set value into sockets, which have best performance (https://bugzilla.redhat.com/show_bug.cgi?id=1653453)
		resources := vmi.Spec.Domain.Resources
		if cpuLimit, ok := resources.Limits[k8sv1.ResourceCPU]; ok {
			sockets = uint32(cpuLimit.Value())
		} else if cpuRequests, ok := resources.Requests[k8sv1.ResourceCPU]; ok {
			sockets = uint32(cpuRequests.Value())
		}
	}

	return &api.CPUTopology{
		Sockets: sockets,
		Cores:   cores,
		Threads: threads,
	}
}

func appendDomainEmulatorThreadPin(domain *api.Domain, allocatedCpu uint32) {
	emulatorThread := api.CPUEmulatorPin{
		CPUSet: strconv.Itoa(int(allocatedCpu)),
	}
	domain.Spec.CPUTune.EmulatorPin = &emulatorThread
}

func appendDomainIOThreadPin(domain *api.Domain, thread uint32, cpuset string) {
	iothreadPin := api.CPUTuneIOThreadPin{}
	iothreadPin.IOThread = thread
	iothreadPin.CPUSet = cpuset
	domain.Spec.CPUTune.IOThreadPin = append(domain.Spec.CPUTune.IOThreadPin, iothreadPin)
}

func formatDomainIOThreadPin(vmi *v1.VirtualMachineInstance, domain *api.Domain, emulatorThread uint32, c *ConverterContext) error {
	iothreads := int(domain.Spec.IOThreads.IOThreads)
	vcpus := int(vcpu.CalculateRequestedVCPUs(domain.Spec.CPU.Topology))

	if vmi.IsCPUDedicated() && vmi.Spec.Domain.CPU.IsolateEmulatorThread {
		// pin the IOThread on the same pCPU as the emulator thread
		cpuset := strconv.Itoa(int(emulatorThread))
		appendDomainIOThreadPin(domain, uint32(1), cpuset)
	} else if iothreads >= vcpus {
		// pin an IOThread on a CPU
		for thread := 1; thread <= iothreads; thread++ {
			cpuset := fmt.Sprintf("%d", c.CPUSet[thread%vcpus])
			appendDomainIOThreadPin(domain, uint32(thread), cpuset)
		}
	} else {
		// the following will pin IOThreads to a set of cpus of a balanced size
		// for example, for 3 threads and 8 cpus the output will look like:
		// thread cpus
		//   1    0,1,2
		//   2    3,4,5
		//   3    6,7
		series := vcpus % iothreads
		curr := 0
		for thread := 1; thread <= iothreads; thread++ {
			remainder := vcpus/iothreads - 1
			if thread <= series {
				remainder += 1
			}
			end := curr + remainder
			slice := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(c.CPUSet[curr:end+1])), ","), "[]")
			appendDomainIOThreadPin(domain, uint32(thread), slice)
			curr = end + 1
		}
	}
	return nil
}

func QuantityToByte(quantity resource.Quantity) (api.Memory, error) {
	memorySize, int := quantity.AsInt64()
	if !int {
		memorySize = quantity.Value() - 1
	}

	if memorySize < 0 {
		return api.Memory{Unit: "b"}, fmt.Errorf("Memory size '%s' must be greater than or equal to 0", quantity.String())
	}
	return api.Memory{
		Value: uint64(memorySize),
		Unit:  "b",
	}, nil
}

func QuantityToMebiByte(quantity resource.Quantity) (uint64, error) {
	bytes, err := QuantityToByte(quantity)
	if err != nil {
		return 0, err
	}
	if bytes.Value == 0 {
		return 0, nil
	} else if bytes.Value < 1048576 {
		return 1, nil
	}
	return uint64(float64(bytes.Value)/1048576 + 0.5), nil
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

func GetImageInfo(imagePath string) (*containerdisk.DiskInfo, error) {

	// #nosec No risk for attacket injection. Only get information about an image
	out, err := exec.Command(
		"/usr/bin/qemu-img", "info", imagePath, "--output", "json",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to invoke qemu-img: %v", err)
	}
	info := &containerdisk.DiskInfo{}
	err = json.Unmarshal(out, info)
	if err != nil {
		return nil, fmt.Errorf("failed to parse disk info: %v", err)
	}
	return info, err
}

func needsSCSIControler(vmi *v1.VirtualMachineInstance) bool {
	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		if disk.LUN != nil && disk.LUN.Bus == "scsi" {
			return true
		}
		if disk.Disk != nil && disk.Disk.Bus == "scsi" {
			return true
		}
		if disk.CDRom != nil && disk.CDRom.Bus == "scsi" {
			return true
		}
	}
	return !vmi.Spec.Domain.Devices.DisableHotplug
}

func getPrefixFromBus(bus string) string {
	switch bus {
	case "virtio":
		return "vd"
	case "sata", "scsi":
		return "sd"
	case "fdc":
		return "fd"
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

func translateModel(ctx *ConverterContext, bus string) string {
	switch bus {
	case "virtio":
		if ctx.UseVirtioTransitional {
			return "virtio-transitional"
		} else {
			return "virtio-non-transitional"
		}
	default:
		return bus
	}
}

// GetVolumeNameByTarget returns the volume name associated to the device target in the domain (e.g vda)
func GetVolumeNameByTarget(domain *api.Domain, target string) string {
	for _, d := range domain.Spec.Devices.Disks {
		if d.Target.Device == target {
			return d.Alias.GetName()
		}
	}
	return ""
}

func isNumaPassthrough(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.CPU.NUMA != nil && vmi.Spec.Domain.CPU.NUMA.GuestMappingPassthrough != nil
}

func hasTabletDevice(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.Devices.Inputs != nil {
		for _, device := range vmi.Spec.Domain.Devices.Inputs {
			if device.Type == "tablet" {
				return true
			}
		}
	}
	return false
}
