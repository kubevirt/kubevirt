package v1

import (
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/types"
)

var _true = t(true)
var _false = t(false)

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

func SetDefaults_SyNICTimer(obj *SyNICTimer) {
	if obj.Enabled == nil {
		obj.Enabled = _true
	}

	if obj.Direct != nil && obj.Direct.Enabled == nil {
		obj.Direct.Enabled = _true
	}
}

func SetDefaults_FeatureAPIC(obj *FeatureAPIC) {
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
		obj.UUID = types.UID(uuid.NewString())
	}
}

func SetDefaults_VirtualMachineInstance(obj *VirtualMachineInstance) {
	if obj.Spec.Domain.Firmware == nil {
		obj.Spec.Domain.Firmware = &Firmware{}
	}

	if obj.Spec.Domain.Features == nil {
		obj.Spec.Domain.Features = &Features{}
	}

	setDefaults_Disk(obj)
	setDefaults_Input(obj)
	SetDefaults_Probe(obj.Spec.ReadinessProbe)
	SetDefaults_Probe(obj.Spec.LivenessProbe)
}

func setDefaults_Disk(obj *VirtualMachineInstance) {
	for i := range obj.Spec.Domain.Devices.Disks {
		disk := &obj.Spec.Domain.Devices.Disks[i].DiskDevice
		SetDefaults_DiskDevice(disk)
	}
}

func setDefaults_Input(obj *VirtualMachineInstance) {
	for i := range obj.Spec.Domain.Devices.Inputs {
		input := &obj.Spec.Domain.Devices.Inputs[i]

		if input.Bus == "" {
			input.Bus = InputBusUSB
		}

		if input.Type == "" {
			input.Type = InputTypeTablet
		}
	}
}

func SetDefaults_Probe(probe *Probe) {
	if probe == nil {
		return
	}

	if probe.TimeoutSeconds < 1 {
		probe.TimeoutSeconds = 1
	}

	if probe.PeriodSeconds < 1 {
		probe.PeriodSeconds = 10
	}

	if probe.SuccessThreshold < 1 {
		probe.SuccessThreshold = 1
	}

	if probe.FailureThreshold < 1 {
		probe.FailureThreshold = 3
	}
}

func DefaultBridgeNetworkInterface() *Interface {
	iface := &Interface{
		Name: "default",
		InterfaceBindingMethod: InterfaceBindingMethod{
			Bridge: &InterfaceBridge{},
		},
	}
	return iface
}

func DefaultMasqueradeNetworkInterface() *Interface {
	iface := &Interface{
		Name: "default",
		InterfaceBindingMethod: InterfaceBindingMethod{
			Masquerade: &InterfaceMasquerade{},
		},
	}
	return iface
}

func DefaultPodNetwork() *Network {
	defaultNet := &Network{
		Name: "default",
		NetworkSource: NetworkSource{
			Pod: &PodNetwork{},
		},
	}
	return defaultNet
}

func t(v bool) *bool {
	return &v
}

func ui32(v uint32) *uint32 {
	return &v
}
