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
package compatibility

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"

	corev1 "kubevirt.io/api/core/v1"
	v1 "kubevirt.io/api/instancetype/v1"
	"kubevirt.io/api/instancetype/v1beta1"
	generatedscheme "kubevirt.io/client-go/kubevirt/scheme"
)

// deprecatedTopologyToNew maps deprecated v1beta1 PreferredCPUTopology values to their v1 equivalents.
var deprecatedTopologyToNew = map[v1beta1.PreferredCPUTopology]v1.PreferredCPUTopology{
	v1beta1.DeprecatedPreferCores:   v1.Cores,
	v1beta1.DeprecatedPreferSockets: v1.Sockets,
	v1beta1.DeprecatedPreferThreads: v1.Threads,
	v1beta1.DeprecatedPreferSpread:  v1.Spread,
	v1beta1.DeprecatedPreferAny:     v1.Any,
	// Direct mappings for non-deprecated values
	v1beta1.Cores:   v1.Cores,
	v1beta1.Sockets: v1.Sockets,
	v1beta1.Threads: v1.Threads,
	v1beta1.Spread:  v1.Spread,
	v1beta1.Any:     v1.Any,
}

func GetInstancetypeSpec(revision *appsv1.ControllerRevision) (*v1.VirtualMachineInstancetypeSpec, error) {
	if err := Decode(revision); err != nil {
		return nil, err
	}
	switch obj := revision.Data.Object.(type) {
	case *v1.VirtualMachineInstancetype:
		return &obj.Spec, nil
	case *v1.VirtualMachineClusterInstancetype:
		return &obj.Spec, nil
	default:
		return nil, fmt.Errorf("unexpected type in ControllerRevision: %T", obj)
	}
}

func GetPreferenceSpec(revision *appsv1.ControllerRevision) (*v1.VirtualMachinePreferenceSpec, error) {
	if err := Decode(revision); err != nil {
		return nil, err
	}
	switch obj := revision.Data.Object.(type) {
	case *v1.VirtualMachinePreference:
		return &obj.Spec, nil
	case *v1.VirtualMachineClusterPreference:
		return &obj.Spec, nil
	default:
		return nil, fmt.Errorf("unexpected type in ControllerRevision: %T", obj)
	}
}

func Decode(revision *appsv1.ControllerRevision) error {
	if len(revision.Data.Raw) == 0 {
		return nil
	}
	return decodeControllerRevisionObject(revision)
}

func decodeControllerRevisionObject(revision *appsv1.ControllerRevision) error {
	decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), revision.Data.Raw)
	if err != nil {
		return fmt.Errorf("failed to decode object in ControllerRevision: %w", err)
	}

	convertedObj, err := ConvertToV1(decodedObj)
	if err != nil {
		return fmt.Errorf("failed to convert object in ControllerRevision: %w", err)
	}

	revision.Data.Object = convertedObj
	return nil
}

// ConvertToV1 converts v1beta1 objects to v1. This is exported for use by other packages.
func ConvertToV1(in runtime.Object) (runtime.Object, error) {
	switch obj := in.(type) {
	case *v1beta1.VirtualMachineInstancetype:
		return convertInstancetype(obj), nil
	case *v1beta1.VirtualMachineClusterInstancetype:
		return convertClusterInstancetype(obj), nil
	case *v1beta1.VirtualMachinePreference:
		return convertPreference(obj), nil
	case *v1beta1.VirtualMachineClusterPreference:
		return convertClusterPreference(obj), nil
	case *v1.VirtualMachineInstancetype, *v1.VirtualMachineClusterInstancetype,
		*v1.VirtualMachinePreference, *v1.VirtualMachineClusterPreference:
		// Already v1, return as is
		return in, nil
	default:
		return nil, fmt.Errorf("unexpected type: %T", in)
	}
}

func convertInstancetype(in *v1beta1.VirtualMachineInstancetype) *v1.VirtualMachineInstancetype {
	return &v1.VirtualMachineInstancetype{
		ObjectMeta: in.ObjectMeta,
		Spec:       convertInstancetypeSpec(&in.Spec),
	}
}

func convertClusterInstancetype(in *v1beta1.VirtualMachineClusterInstancetype) *v1.VirtualMachineClusterInstancetype {
	return &v1.VirtualMachineClusterInstancetype{
		ObjectMeta: in.ObjectMeta,
		Spec:       convertInstancetypeSpec(&in.Spec),
	}
}

func convertPreference(in *v1beta1.VirtualMachinePreference) *v1.VirtualMachinePreference {
	return &v1.VirtualMachinePreference{
		ObjectMeta: in.ObjectMeta,
		Spec:       convertPreferenceSpec(&in.Spec),
	}
}

func convertClusterPreference(in *v1beta1.VirtualMachineClusterPreference) *v1.VirtualMachineClusterPreference {
	return &v1.VirtualMachineClusterPreference{
		ObjectMeta: in.ObjectMeta,
		Spec:       convertPreferenceSpec(&in.Spec),
	}
}

func convertInstancetypeSpec(in *v1beta1.VirtualMachineInstancetypeSpec) v1.VirtualMachineInstancetypeSpec {
	return v1.VirtualMachineInstancetypeSpec{
		NodeSelector:    in.NodeSelector,
		SchedulerName:   in.SchedulerName,
		CPU:             convertCPUInstancetype(&in.CPU),
		Memory:          convertMemoryInstancetype(&in.Memory),
		GPUs:            in.GPUs,
		HostDevices:     in.HostDevices,
		IOThreadsPolicy: in.IOThreadsPolicy,
		IOThreads:       in.IOThreads,
		LaunchSecurity:  in.LaunchSecurity,
		Annotations:     in.Annotations,
	}
}

func convertCPUInstancetype(in *v1beta1.CPUInstancetype) v1.CPUInstancetype {
	return v1.CPUInstancetype{
		Guest:                 in.Guest,
		Model:                 in.Model,
		DedicatedCPUPlacement: in.DedicatedCPUPlacement,
		NUMA:                  in.NUMA,
		IsolateEmulatorThread: in.IsolateEmulatorThread,
		Realtime:              in.Realtime,
		MaxSockets:            in.MaxSockets,
	}
}

func convertMemoryInstancetype(in *v1beta1.MemoryInstancetype) v1.MemoryInstancetype {
	return v1.MemoryInstancetype{
		Guest:             in.Guest,
		Hugepages:         in.Hugepages,
		OvercommitPercent: in.OvercommitPercent,
		MaxGuest:          in.MaxGuest,
	}
}

func convertPreferenceSpec(in *v1beta1.VirtualMachinePreferenceSpec) v1.VirtualMachinePreferenceSpec {
	out := v1.VirtualMachinePreferenceSpec{
		PreferredSubdomain:                     in.PreferredSubdomain,
		PreferredTerminationGracePeriodSeconds: in.PreferredTerminationGracePeriodSeconds,
		Annotations:                            in.Annotations,
		PreferSpreadSocketToCoreRatio:          in.PreferSpreadSocketToCoreRatio,
		PreferredArchitecture:                  in.PreferredArchitecture,
	}

	if in.Clock != nil {
		out.Clock = convertClockPreferences(in.Clock)
	}
	if in.CPU != nil {
		out.CPU = convertCPUPreferences(in.CPU)
	}
	if in.Devices != nil {
		out.Devices = convertDevicePreferences(in.Devices)
	}
	if in.Features != nil {
		out.Features = convertFeaturePreferences(in.Features)
	}
	if in.Firmware != nil {
		out.Firmware = convertFirmwarePreferences(in.Firmware)
	}
	if in.Machine != nil {
		out.Machine = convertMachinePreferences(in.Machine)
	}
	if in.Volumes != nil {
		out.Volumes = convertVolumePreferences(in.Volumes)
	}
	if in.Requirements != nil {
		out.Requirements = convertPreferenceRequirements(in.Requirements)
	}

	return out
}

func convertClockPreferences(in *v1beta1.ClockPreferences) *v1.ClockPreferences {
	return &v1.ClockPreferences{
		PreferredClockOffset: in.PreferredClockOffset,
		PreferredTimer:       in.PreferredTimer,
	}
}

func convertCPUPreferences(in *v1beta1.CPUPreferences) *v1.CPUPreferences {
	out := &v1.CPUPreferences{
		PreferredCPUFeatures: in.PreferredCPUFeatures,
	}

	// Convert deprecated topology values to new values
	if in.PreferredCPUTopology != nil {
		if newValue, ok := deprecatedTopologyToNew[*in.PreferredCPUTopology]; ok {
			out.PreferredCPUTopology = &newValue
		}
	}

	if in.SpreadOptions != nil {
		out.SpreadOptions = convertSpreadOptions(in.SpreadOptions)
	}

	return out
}

func convertSpreadOptions(in *v1beta1.SpreadOptions) *v1.SpreadOptions {
	out := &v1.SpreadOptions{
		Ratio: in.Ratio,
	}

	if in.Across != nil {
		across := v1.SpreadAcross(*in.Across)
		out.Across = &across
	}

	return out
}

func convertDevicePreferences(in *v1beta1.DevicePreferences) *v1.DevicePreferences {
	return &v1.DevicePreferences{
		PreferredAutoattachGraphicsDevice:   in.PreferredAutoattachGraphicsDevice,
		PreferredAutoattachMemBalloon:       in.PreferredAutoattachMemBalloon,
		PreferredAutoattachPodInterface:     in.PreferredAutoattachPodInterface,
		PreferredAutoattachSerialConsole:    in.PreferredAutoattachSerialConsole,
		PreferredAutoattachInputDevice:      in.PreferredAutoattachInputDevice,
		PreferredDisableHotplug:             in.PreferredDisableHotplug,
		PreferredVirtualGPUOptions:          in.PreferredVirtualGPUOptions,
		PreferredSoundModel:                 in.PreferredSoundModel,
		PreferredUseVirtioTransitional:      in.PreferredUseVirtioTransitional,
		PreferredInputBus:                   in.PreferredInputBus,
		PreferredInputType:                  in.PreferredInputType,
		PreferredDiskBus:                    in.PreferredDiskBus,
		PreferredLunBus:                     in.PreferredLunBus,
		PreferredCdromBus:                   in.PreferredCdromBus,
		PreferredDiskDedicatedIoThread:      in.PreferredDiskDedicatedIoThread,
		PreferredDiskCache:                  in.PreferredDiskCache,
		PreferredDiskIO:                     in.PreferredDiskIO,
		PreferredDiskBlockSize:              in.PreferredDiskBlockSize,
		PreferredInterfaceModel:             in.PreferredInterfaceModel,
		PreferredRng:                        in.PreferredRng,
		PreferredBlockMultiQueue:            in.PreferredBlockMultiQueue,
		PreferredNetworkInterfaceMultiQueue: in.PreferredNetworkInterfaceMultiQueue,
		PreferredTPM:                        in.PreferredTPM,
		PreferredInterfaceMasquerade:        in.PreferredInterfaceMasquerade,
		PreferredPanicDeviceModel:           in.PreferredPanicDeviceModel,
	}
}

func convertFeaturePreferences(in *v1beta1.FeaturePreferences) *v1.FeaturePreferences {
	return &v1.FeaturePreferences{
		PreferredAcpi:       in.PreferredAcpi,
		PreferredApic:       in.PreferredApic,
		PreferredHyperv:     in.PreferredHyperv,
		PreferredKvm:        in.PreferredKvm,
		PreferredPvspinlock: in.PreferredPvspinlock,
		PreferredSmm:        in.PreferredSmm,
	}
}

func convertFirmwarePreferences(in *v1beta1.FirmwarePreferences) *v1.FirmwarePreferences {
	out := &v1.FirmwarePreferences{
		PreferredUseBios:       in.PreferredUseBios,
		PreferredUseBiosSerial: in.PreferredUseBiosSerial,
		PreferredEfi:           in.PreferredEfi,
	}

	// Convert deprecated fields to PreferredEfi if PreferredEfi is not already set
	if out.PreferredEfi == nil {
		if in.DeprecatedPreferredUseEfi != nil && *in.DeprecatedPreferredUseEfi {
			out.PreferredEfi = &corev1.EFI{}
			if in.DeprecatedPreferredUseSecureBoot != nil {
				out.PreferredEfi.SecureBoot = in.DeprecatedPreferredUseSecureBoot
			}
		}
	}

	return out
}

func convertMachinePreferences(in *v1beta1.MachinePreferences) *v1.MachinePreferences {
	return &v1.MachinePreferences{
		PreferredMachineType: in.PreferredMachineType,
	}
}

func convertVolumePreferences(in *v1beta1.VolumePreferences) *v1.VolumePreferences {
	return &v1.VolumePreferences{
		PreferredStorageClassName: in.PreferredStorageClassName,
	}
}

func convertPreferenceRequirements(in *v1beta1.PreferenceRequirements) *v1.PreferenceRequirements {
	out := &v1.PreferenceRequirements{
		Architecture: in.Architecture,
	}

	if in.CPU != nil {
		out.CPU = &v1.CPUPreferenceRequirement{
			Guest: in.CPU.Guest,
		}
	}

	if in.Memory != nil {
		out.Memory = &v1.MemoryPreferenceRequirement{
			Guest: in.Memory.Guest,
		}
	}

	return out
}
