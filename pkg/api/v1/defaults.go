package v1

import (
	"strings"

	"github.com/pborman/uuid"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
)

var _true = t(true)
var _false = t(false)

// TODO: make it const?
// no special meaning, randomly generated on my box.
// TODO: do we want to use another constants? see examples in RFC4122
var _uuid_ns = uuid.Parse("6a1a24a1-4061-4607-8bf4-a3963d0c5895")

func SetDefaults_HPETTimer(obj *HPETTimer) {
	if obj.Enabled == nil {
		obj.Enabled = _true
	}
}

func SetDefaults_PITTimer(obj *PITTimer) {
	if obj.Enabled == nil {
		obj.Enabled = _true
	}
}

func SetDefaults_KVMTimer(obj *KVMTimer) {
	if obj.Enabled == nil {
		obj.Enabled = _true
	}
}

func SetDefaults_HypervTimer(obj *HypervTimer) {
	if obj.Enabled == nil {
		obj.Enabled = _true
	}
}

func SetDefaults_RTCTimer(obj *RTCTimer) {
	if obj.Enabled == nil {
		obj.Enabled = _true
	}
}

func SetDefaults_FeatureState(obj *FeatureState) {
	if obj.Enabled == nil {
		obj.Enabled = _true
	}
}

func SetDefaults_FeatureVendorID(obj *FeatureVendorID) {
	if obj.Enabled == nil {
		obj.Enabled = _true
	}
}

func SetDefaults_DiskDevice(obj *DiskDevice) {
	if obj.Disk == nil &&
		obj.CDRom == nil &&
		obj.Floppy == nil &&
		obj.LUN == nil {
		obj.Disk = &DiskTarget{}
	}
}

func SetDefaults_Watchdog(obj *Watchdog) {
	if obj.I6300ESB == nil {
		obj.I6300ESB = &I6300ESBWatchdog{}
	}
}

func SetDefaults_CDRomTarget(obj *CDRomTarget) {
	if obj.ReadOnly == nil {
		obj.ReadOnly = _true
	}
	if obj.Tray == "" {
		obj.Tray = TrayStateClosed
	}
}

func SetDefaults_FloppyTarget(obj *FloppyTarget) {
	if obj.Tray == "" {
		obj.Tray = TrayStateClosed
	}
}

func SetDefaults_FeatureSpinlocks(obj *FeatureSpinlocks) {
	if obj.Enabled == nil {
		obj.Enabled = _true
	}
	if *obj.Enabled == *_true && obj.Retries == nil {
		obj.Retries = ui32(4096)
	}
}

func SetDefaults_I6300ESBWatchdog(obj *I6300ESBWatchdog) {
	if obj.Action == "" {
		obj.Action = WatchdogActionReset
	}
}

func SetDefaults_Firmware(obj *Firmware) {
	if obj.UUID == "" {
		obj.UUID = types.UID(uuid.NewRandom().String())
	}
}

func SetDefaults_VirtualMachine(obj *VirtualMachine) {
	// FIXME we need proper validation and configurable defaulting instead of this
	if _, exists := obj.Spec.Domain.Resources.Requests[v1.ResourceMemory]; !exists {
		obj.Spec.Domain.Resources.Requests = v1.ResourceList{
			v1.ResourceMemory: resource.MustParse("8192Ki"),
		}
	}
	if obj.Spec.Domain.Firmware == nil {
		obj.Spec.Domain.Firmware = &Firmware{}
	}
	if obj.Spec.Domain.Firmware.UUID == "" {
		obj.Spec.Domain.Firmware.UUID = types.UID(uuid.NewSHA1(_uuid_ns, []byte(obj.Name)).String())
	}

	if obj.Spec.Domain.Features == nil {
		obj.Spec.Domain.Features = &Features{}
	}
	if obj.Spec.Domain.Machine.Type == "" {
		obj.Spec.Domain.Machine.Type = "q35"
	}
	setDefaults_DiskFromMachineType(obj)
}

func setDefaults_DiskFromMachineType(obj *VirtualMachine) {
	bus := diskBusFromMachine(obj.Spec.Domain.Machine.Type)

	for i := range obj.Spec.Domain.Devices.Disks {
		disk := &obj.Spec.Domain.Devices.Disks[i].DiskDevice

		SetDefaults_DiskDevice(disk)

		if disk.Disk != nil && disk.Disk.Bus == "" {
			disk.Disk.Bus = bus
		}
		if disk.CDRom != nil && disk.CDRom.Bus == "" {
			disk.CDRom.Bus = bus
		}
		if disk.LUN != nil && disk.LUN.Bus == "" {
			disk.LUN.Bus = bus
		}
	}
}

func diskBusFromMachine(machine string) string {
	// catches: "q35", "pc-q35-*"
	// see /path/to/qemu-kvm -machine help
	if strings.HasPrefix(machine, "pc-q35") || strings.HasPrefix(machine, "q35") {
		return "sata"
	}
	// safe fallback for x86_64, but very slow
	return "ide"
}

func t(v bool) *bool {
	return &v
}

func ui32(v uint32) *uint32 {
	return &v
}
