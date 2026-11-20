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

package admitters

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	hwutil "kubevirt.io/kubevirt/pkg/util/hardware"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate/compute"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate/legacy"
)

const graceVirtMachineType = "virt"

type gracePCIDeviceRequest struct {
	path         *k8sfield.Path
	resourceName string
}

func validateGraceIOVirtualization(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	graceRequests, ambiguousNVIDIARequests := gracePCIDeviceRequests(field, spec, config.GetPermittedHostDevices())
	if len(graceRequests) == 0 && len(ambiguousNVIDIARequests) == 0 {
		return nil
	}

	var causes []metav1.StatusCause
	if config.GraceIOVirtualizationEnabled() {
		for _, request := range ambiguousNVIDIARequests {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("GraceIOVirtualization requires an exact pciVendorSelector for NVIDIA PCI host device resource %q", request.resourceName),
				Field:   request.path.String(),
			})
		}
	}

	if len(graceRequests) == 0 {
		return causes
	}

	if !config.GraceIOVirtualizationEnabled() {
		for _, request := range graceRequests {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s feature gate must be enabled for NVIDIA Grace GPU passthrough resource %q", compute.GraceIOVirtualization, request.resourceName),
				Field:   request.path.String(),
			})
		}
		return causes
	}

	if effectiveGraceArchitecture(spec, config) != "arm64" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "GraceIOVirtualization requires arm64 architecture",
			Field:   field.Child("architecture").String(),
		})
	}

	if !strings.HasPrefix(effectiveGraceMachineType(spec, config), graceVirtMachineType) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("GraceIOVirtualization requires %q machine type", graceVirtMachineType),
			Field:   field.Child("domain", "machine", "type").String(),
		})
	}

	if spec.Domain.CPU == nil || !spec.Domain.CPU.DedicatedCPUPlacement {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "GraceIOVirtualization requires dedicated CPU placement",
			Field:   field.Child("domain", "cpu", "dedicatedCpuPlacement").String(),
		})
	}

	if !config.PCINUMAAwareTopologyEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s feature gate must be enabled when GraceIOVirtualization is used",
				legacy.PCINUMAAwareTopologyEnabled),
			Field: field.Child("domain", "devices").String(),
		})
	}

	if !config.IOMMUFDEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s feature gate must be enabled when GraceIOVirtualization is used",
				compute.IOMMUFDGate),
			Field: field.Child("domain", "devices").String(),
		})
	}

	return causes
}

func validateGraceIOVirtualizationAnnotations(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, annotations map[string]string, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	graceRequests, _ := gracePCIDeviceRequests(k8sfield.NewPath("spec"), spec, config.GetPermittedHostDevices())
	if len(graceRequests) == 0 {
		return nil
	}

	var causes []metav1.StatusCause
	if annotations[v1.PlacePCIDevicesOnRootComplex] == "true" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "GraceIOVirtualization is not compatible with placing PCI devices on the root complex",
			Field:   field.Child("annotations").Key(v1.PlacePCIDevicesOnRootComplex).String(),
		})
	}
	if annotations[v1.DisablePCIHole64] == "true" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "GraceIOVirtualization requires the 64-bit PCI hole",
			Field:   field.Child("annotations").Key(v1.DisablePCIHole64).String(),
		})
	}
	return causes
}

func gracePCIDeviceRequests(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, permittedHostDevices *v1.PermittedHostDevices) ([]gracePCIDeviceRequest, []gracePCIDeviceRequest) {
	resourceSelectors := permittedPCIResourceSelectors(permittedHostDevices)
	if len(resourceSelectors) == 0 {
		return nil, nil
	}

	var graceRequests []gracePCIDeviceRequest
	var ambiguousNVIDIARequests []gracePCIDeviceRequest
	for _, request := range requestedPCIDeviceResources(field, spec) {
		selector, exists := resourceSelectors[request.resourceName]
		if !exists {
			continue
		}

		if hwutil.IsNVIDIAGracePCIVendorSelector(selector) {
			graceRequests = append(graceRequests, request)
			continue
		}
		if hwutil.IsAmbiguousNVIDIAPCIVendorSelector(selector) {
			ambiguousNVIDIARequests = append(ambiguousNVIDIARequests, request)
		}
	}

	return graceRequests, ambiguousNVIDIARequests
}

func permittedPCIResourceSelectors(permittedHostDevices *v1.PermittedHostDevices) map[string]string {
	if permittedHostDevices == nil || len(permittedHostDevices.PciHostDevices) == 0 {
		return nil
	}

	selectors := map[string]string{}
	for _, pciHostDevice := range permittedHostDevices.PciHostDevices {
		if pciHostDevice.ResourceName == "" || pciHostDevice.PCIVendorSelector == "" {
			continue
		}
		selectors[pciHostDevice.ResourceName] = pciHostDevice.PCIVendorSelector
	}
	return selectors
}

func requestedPCIDeviceResources(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []gracePCIDeviceRequest {
	var requests []gracePCIDeviceRequest
	for index, gpu := range spec.Domain.Devices.GPUs {
		if gpu.DeviceName == "" {
			continue
		}
		requests = append(requests, gracePCIDeviceRequest{
			path:         field.Child("domain", "devices", "gpus").Index(index).Child("deviceName"),
			resourceName: gpu.DeviceName,
		})
	}
	for index, hostDevice := range spec.Domain.Devices.HostDevices {
		if hostDevice.DeviceName == "" {
			continue
		}
		requests = append(requests, gracePCIDeviceRequest{
			path:         field.Child("domain", "devices", "hostDevices").Index(index).Child("deviceName"),
			resourceName: hostDevice.DeviceName,
		})
	}
	return requests
}

func effectiveGraceArchitecture(spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) string {
	if spec.Architecture != "" {
		return spec.Architecture
	}
	return config.GetDefaultArchitecture()
}

func effectiveGraceMachineType(spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) string {
	if spec.Domain.Machine != nil && spec.Domain.Machine.Type != "" {
		return spec.Domain.Machine.Type
	}
	return config.GetMachineType(effectiveGraceArchitecture(spec, config))
}
