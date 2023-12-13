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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package admitters

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/downwardmetrics"
	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/network/link"
	"kubevirt.io/kubevirt/pkg/storage/reservation"
	hwutil "kubevirt.io/kubevirt/pkg/util/hardware"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virtiofs"
)

const requiredFieldFmt = "%s is a required field"

const (
	arrayLenMax = 256
	maxStrLen   = 256

	// cloudInitNetworkMaxLen and CloudInitUserMaxLen are being limited
	// to 2K to allow scaling of config as edits will cause entire object
	// to be distributed to large no of nodes. For larger than 2K, user should
	// use NetworkDataSecretRef and UserDataSecretRef
	cloudInitUserMaxLen    = 2048
	cloudInitNetworkMaxLen = 2048

	// Copied from kubernetes/pkg/apis/core/validation/validation.go
	maxDNSNameservers     = 3
	maxDNSSearchPaths     = 6
	maxDNSSearchListChars = 256
)

var validInterfaceModels = map[string]*struct{}{"e1000": nil, "e1000e": nil, "ne2k_pci": nil, "pcnet": nil, "rtl8139": nil, v1.VirtIO: nil}
var validIOThreadsPolicies = []v1.IOThreadsPolicy{v1.IOThreadsPolicyShared, v1.IOThreadsPolicyAuto}
var validCPUFeaturePolicies = map[string]*struct{}{"": nil, "force": nil, "require": nil, "optional": nil, "disable": nil, "forbid": nil}

var restrictedVmiLabels = map[string]bool{
	v1.CreatedByLabel:               true,
	v1.MigrationJobLabel:            true,
	v1.NodeNameLabel:                true,
	v1.MigrationTargetNodeNameLabel: true,
	v1.NodeSchedulable:              true,
	v1.InstallStrategyLabel:         true,
}

const (
	nameOfTypeNotFoundMessagePattern  = "%s '%s' not found."
	listExceedsLimitMessagePattern    = "%s list exceeds the %d element limit in length"
	valueMustBePositiveMessagePattern = "%s '%s': must be greater than or equal to 0."
)

type VMICreateAdmitter struct {
	ClusterConfig *virtconfig.ClusterConfig
}

func (admitter *VMICreateAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if resp := webhookutils.ValidateSchema(v1.VirtualMachineInstanceGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	accountName := ar.Request.UserInfo.Username
	vmi, _, err := webhookutils.GetVMIFromAdmissionReview(ar)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("spec"), &vmi.Spec, admitter.ClusterConfig)
	// We only want to validate that volumes are mapped to disks or filesystems during VMI admittance, thus this logic is seperated from the above call that is shared with the VM admitter.
	causes = append(causes, validateVirtualMachineInstanceSpecVolumeDisks(k8sfield.NewPath("spec"), &vmi.Spec)...)
	causes = append(causes, ValidateVirtualMachineInstanceMandatoryFields(k8sfield.NewPath("spec"), &vmi.Spec)...)
	causes = append(causes, ValidateVirtualMachineInstanceMetadata(k8sfield.NewPath("metadata"), &vmi.ObjectMeta, admitter.ClusterConfig, accountName)...)
	// In a future, yet undecided, release either libvirt or QEMU are going to check the hyperv dependencies, so we can get rid of this code.
	causes = append(causes, webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(k8sfield.NewPath("spec"), &vmi.Spec)...)
	if webhooks.IsARM64(&vmi.Spec) {
		// Check if there is any unsupported setting if the arch is Arm64
		causes = append(causes, webhooks.ValidateVirtualMachineInstanceArm64Setting(k8sfield.NewPath("spec"), &vmi.Spec)...)
	}
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	reviewResponse := admissionv1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func ValidateVirtualMachineInstanceSpec(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause
	volumeNameMap := make(map[string]*v1.Volume)
	networkNameMap := make(map[string]*v1.Network)

	maxNumberOfDisksExceeded := len(spec.Domain.Devices.Disks) > arrayLenMax
	if maxNumberOfDisksExceeded {
		return appendNewStatusCauseForNumberOfDisksExceeded(field, causes)
	}

	maxNumberOfVolumesExceeded := len(spec.Volumes) > arrayLenMax
	if maxNumberOfVolumesExceeded {
		return appendNewStatusCauseForMaxNumberOfVolumesExceeded(field, causes)
	}

	causes = append(causes, validateHostNameNotConformingToDNSLabelRules(field, spec)...)
	causes = append(causes, validateSubdomainDNSSubdomainRules(field, spec)...)
	causes = append(causes, validateMemoryRequestsNegativeOrNull(field, spec)...)
	causes = append(causes, validateMemoryLimitsNegativeOrNull(field, spec)...)
	causes = append(causes, validateHugepagesMemoryRequests(field, spec)...)
	causes = append(causes, validateGuestMemoryLimit(field, spec)...)
	causes = append(causes, validateEmulatedMachine(field, spec, config)...)
	causes = append(causes, validateFirmwareSerial(field, spec)...)
	causes = append(causes, validateCPURequestNotNegative(field, spec)...)
	causes = append(causes, validateCPULimitNotNegative(field, spec)...)
	causes = append(causes, validateCpuRequestDoesNotExceedLimit(field, spec)...)
	causes = append(causes, validateCpuPinning(field, spec, config)...)
	causes = append(causes, validateNUMA(field, spec, config)...)
	causes = append(causes, validateCPUIsolatorThread(field, spec)...)
	causes = append(causes, validateCPUFeaturePolicies(field, spec)...)
	causes = append(causes, validateCPUHotplug(field, spec)...)
	causes = append(causes, validateStartStrategy(field, spec)...)
	causes = append(causes, validateRealtime(field, spec)...)
	causes = append(causes, validateSpecAffinity(field, spec)...)
	causes = append(causes, validateSpecTopologySpreadConstraints(field, spec)...)
	causes = append(causes, validateArchitecture(field, spec, config)...)

	maxNumberOfInterfacesExceeded := len(spec.Domain.Devices.Interfaces) > arrayLenMax
	if maxNumberOfInterfacesExceeded {
		return appendStatusCauseForMaxNumberOfInterfacesExceeded(field, causes)
	}
	maxNumberOfNetworksExceeded := len(spec.Networks) > arrayLenMax
	if maxNumberOfNetworksExceeded {
		return appendStatusCauseMaxNumberOfNetworksExceeded(field, causes)
	}
	moreThanOnePodInterface := getNumberOfPodInterfaces(spec) > 1
	if moreThanOnePodInterface {
		return appendStatusCauseForMoreThanOnePodInterface(field, causes)
	}

	bootOrderMap, newCauses := validateBootOrder(field, spec, volumeNameMap)
	causes = append(causes, newCauses...)
	podExists, multusDefaultCount, newCauses := validateNetworks(field, spec, networkNameMap)
	causes = append(causes, newCauses...)

	if multusDefaultCount > 1 {
		causes = appendStatusCaseForMoreThanOneMultusDefaultNetwork(field, causes)
	}
	if podExists && multusDefaultCount > 0 {
		causes = appendStatusCauseForPodNetworkDefinedWithMultusDefaultNetworkDefined(field, causes)
	}

	networkInterfaceMap, newCauses, done := validateNetworksMatchInterfaces(field, spec, config, networkNameMap, bootOrderMap)
	causes = append(causes, newCauses...)
	if done {
		return causes
	}

	causes = append(causes, validateNetworksAssignedToInterfaces(field, spec, networkInterfaceMap)...)
	causes = append(causes, validateInterfaceStateValue(field, spec)...)
	causes = append(causes, validateInterfaceBinding(field, spec)...)

	causes = append(causes, validateInputDevices(field, spec)...)
	causes = append(causes, validateIOThreadsPolicy(field, spec)...)
	causes = append(causes, validateProbe(field.Child("readinessProbe"), spec.ReadinessProbe)...)
	causes = append(causes, validateProbe(field.Child("livenessProbe"), spec.LivenessProbe)...)

	if getNumberOfPodInterfaces(spec) < 1 {
		causes = appendStatusCauseForProbeNotAllowedWithNoPodNetworkPresent(field.Child("readinessProbe"), spec.ReadinessProbe, causes)
		causes = appendStatusCauseForProbeNotAllowedWithNoPodNetworkPresent(field.Child("livenessProbe"), spec.LivenessProbe, causes)
	}

	causes = append(causes, validateDomainSpec(field.Child("domain"), &spec.Domain)...)
	causes = append(causes, validateVolumes(field.Child("volumes"), spec.Volumes, config)...)
	causes = append(causes, validateContainerDisks(field, spec)...)

	causes = append(causes, validateAccessCredentials(field.Child("accessCredentials"), spec.AccessCredentials, spec.Volumes)...)

	if spec.DNSPolicy != "" {
		causes = append(causes, validateDNSPolicy(&spec.DNSPolicy, field.Child("dnsPolicy"))...)
	}
	causes = append(causes, validatePodDNSConfig(spec.DNSConfig, &spec.DNSPolicy, field.Child("dnsConfig"))...)
	causes = append(causes, validateLiveMigration(field, spec, config)...)
	causes = append(causes, validateGPUsWithPassthroughEnabled(field, spec, config)...)
	causes = append(causes, validateFilesystemsWithVirtIOFSEnabled(field, spec, config)...)
	causes = append(causes, validateHostDevicesWithPassthroughEnabled(field, spec, config)...)
	causes = append(causes, validateSoundDevices(field, spec)...)
	causes = append(causes, validateLaunchSecurity(field, spec, config)...)
	causes = append(causes, validateVSOCK(field, spec, config)...)
	causes = append(causes, validatePersistentReservation(field, spec, config)...)
	causes = append(causes, validatePersistentState(field, spec, config)...)
	causes = append(causes, validateDownwardMetrics(field, spec, config)...)

	return causes
}

func validateDownwardMetrics(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause

	// Check if serial and feature gate is enabled
	if downwardmetrics.HasDevice(spec) && !config.DownwardMetricsEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "downwardMetrics virtio serial is not allowed: DownwardMetrics feature gate is not enabled",
			Field:   field.Child("domain", "devices", "downwardMetrics").String(),
		})
	}

	return causes
}

func validateVirtualMachineInstanceSpecVolumeDisks(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	diskAndFilesystemNames := make(map[string]struct{})

	for _, disk := range spec.Domain.Devices.Disks {
		diskAndFilesystemNames[disk.Name] = struct{}{}
	}

	for _, fs := range spec.Domain.Devices.Filesystems {
		diskAndFilesystemNames[fs.Name] = struct{}{}
	}

	// Validate that volumes match disks and filesystems correctly
	for idx, volume := range spec.Volumes {
		if volume.MemoryDump != nil {
			continue
		}
		if _, matchingDiskExists := diskAndFilesystemNames[volume.Name]; !matchingDiskExists {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf(nameOfTypeNotFoundMessagePattern, field.Child("domain", "volumes").Index(idx).Child("name").String(), volume.Name),
				Field:   field.Child("domain", "volumes").Index(idx).Child("name").String(),
			})
		}
	}
	return causes
}

func validateNetworksMatchInterfaces(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig, networkNameMap map[string]*v1.Network, bootOrderMap map[uint]bool) (networkInterfaceMap map[string]struct{}, causes []metav1.StatusCause, done bool) {

	done = false

	// Make sure interfaces and networks are 1to1 related
	networkInterfaceMap = make(map[string]struct{})

	// Make sure the port name is unique across all the interfaces
	portForwardMap := make(map[string]struct{})

	// Validate that each interface has a matching network
	numOfInterfaces := len(spec.Domain.Devices.Interfaces)
	for idx, iface := range spec.Domain.Devices.Interfaces {

		networkData, networkExists := networkNameMap[iface.Name]

		causes = append(causes, validateInterfaceNetworkBasics(field, networkExists, idx, iface, networkData, config, numOfInterfaces)...)

		causes = append(causes, validateInterfaceNameUnique(field, networkInterfaceMap, iface, idx)...)
		causes = append(causes, validateInterfaceNameFormat(field, iface, idx)...)

		networkInterfaceMap[iface.Name] = struct{}{}

		causes = append(causes, validatePortConfiguration(field, networkExists, networkData, iface, idx, portForwardMap)...)
		causes = append(causes, validateInterfaceModel(field, iface, idx)...)
		causes = append(causes, validateMacAddress(field, iface, idx)...)
		causes = append(causes, validateInterfaceBootOrder(field, iface, idx, bootOrderMap)...)
		causes = append(causes, validateInterfacePciAddress(field, iface, idx)...)

		newCauses, newDone := validateDHCPExtraOptions(field, iface)
		causes = append(causes, newCauses...)
		done = newDone
		if done {
			return nil, causes, done
		}

		causes = append(causes, validateDHCPNTPServersAreValidIPv4Addresses(field, iface, idx)...)
	}
	return networkInterfaceMap, causes, done
}

func validateInterfaceNetworkBasics(field *k8sfield.Path, networkExists bool, idx int, iface v1.Interface, networkData *v1.Network, config *virtconfig.ClusterConfig, numOfInterfaces int) (causes []metav1.StatusCause) {
	if !networkExists {
		causes = appendStatusCauseForNetworkNotFound(field, causes, idx, iface)
	} else if iface.Slirp != nil && networkData.Pod == nil {
		causes = appendStatusCauseForSlirpWithoutPodNetwork(field, causes, idx)
	} else if iface.Slirp != nil && networkData.Pod != nil && !config.IsSlirpInterfaceEnabled() {
		causes = appendStatusCauseForSlirpNotEnabled(field, causes, idx)
	} else if iface.Masquerade != nil && networkData.Pod == nil {
		causes = appendStatusCauseForMasqueradeWithoutPodNetwork(field, causes, idx)
	} else if iface.Masquerade != nil && link.IsReserved(iface.MacAddress) {
		causes = appendStatusCauseForInvalidMasqueradeMacAddress(field, causes, idx)
	} else if iface.InterfaceBindingMethod.Bridge != nil && networkData.NetworkSource.Pod != nil && !config.IsBridgeInterfaceOnPodNetworkEnabled() {
		causes = appendStatusCauseForBridgeNotEnabled(field, causes, idx)
	} else if iface.InterfaceBindingMethod.Macvtap != nil && !config.MacvtapEnabled() {
		causes = appendStatusCauseForMacvtapFeatureGateNotEnabled(field, causes, idx)
	} else if iface.InterfaceBindingMethod.Macvtap != nil && networkData.NetworkSource.Multus == nil {
		causes = appendStatusCauseForMacvtapOnlyAllowedWithMultus(field, causes, idx)
	} else if iface.InterfaceBindingMethod.Passt != nil && !config.PasstEnabled() {
		causes = appendStatusCauseForPasstFeatureGateNotEnabled(field, causes, idx)
	} else if iface.Passt != nil && networkData.Pod == nil {
		causes = appendStatusCauseForPasstWithoutPodNetwork(field, causes, idx)
	} else if iface.Passt != nil && numOfInterfaces > 1 {
		causes = appendStatusCauseForPasstWithMultipleInterfaces(field, causes, idx)
	} else if iface.Binding != nil && !config.NetworkBindingPlugingsEnabled() {
		causes = appendStatusCauseForBindingPluginsFeatureGateNotEnabled(field, causes, idx)
	}
	return causes
}

func validateDHCPExtraOptions(field *k8sfield.Path, iface v1.Interface) (causes []metav1.StatusCause, done bool) {
	done = false
	if iface.DHCPOptions != nil {
		PrivateOptions := iface.DHCPOptions.PrivateOptions
		err := ValidateDuplicateDHCPPrivateOptions(PrivateOptions)
		if err != nil {
			causes = appendStatusCauseForDuplicateDHCPOptionFound(field, causes, err)
			done = true
		}
		for _, DHCPPrivateOption := range PrivateOptions {
			causes = append(causes, validateDHCPPrivateOptionsWithinRange(field, DHCPPrivateOption)...)
		}
	}
	return causes, done
}

func validateDHCPNTPServersAreValidIPv4Addresses(field *k8sfield.Path, iface v1.Interface, idx int) (causes []metav1.StatusCause) {
	if iface.DHCPOptions != nil {
		for index, ip := range iface.DHCPOptions.NTPServers {
			if net.ParseIP(ip).To4() == nil {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "NTP servers must be a list of valid IPv4 addresses.",
					Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("dhcpOptions", "ntpServers").Index(index).String(),
				})
			}
		}
	}
	return causes
}

func validateDHCPPrivateOptionsWithinRange(field *k8sfield.Path, DHCPPrivateOption v1.DHCPPrivateOptions) (causes []metav1.StatusCause) {
	if !(DHCPPrivateOption.Option >= 224 && DHCPPrivateOption.Option <= 254) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "provided DHCPPrivateOptions are out of range, must be in range 224 to 254",
			Field:   field.String(),
		})
	}
	return causes
}

func appendStatusCauseForDuplicateDHCPOptionFound(field *k8sfield.Path, causes []metav1.StatusCause, err error) []metav1.StatusCause {
	causes = append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: fmt.Sprintf("Found Duplicates: %v", err),
		Field:   field.String(),
	})
	return causes
}

func validateInterfacePciAddress(field *k8sfield.Path, iface v1.Interface, idx int) (causes []metav1.StatusCause) {
	if iface.PciAddress != "" {
		_, err := hwutil.ParsePciAddress(iface.PciAddress)
		if err != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("interface %s has malformed PCI address (%s).", field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(), iface.PciAddress),
				Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("pciAddress").String(),
			})
		}
	}
	return causes
}

func validateInterfaceBootOrder(field *k8sfield.Path, iface v1.Interface, idx int, bootOrderMap map[uint]bool) (causes []metav1.StatusCause) {
	if iface.BootOrder != nil {
		order := *iface.BootOrder
		// Verify boot order is greater than 0, if provided
		if order < 1 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s must have a boot order > 0, if supplied", field.Index(idx).String()),
				Field:   field.Index(idx).Child("bootOrder").String(),
			})
		} else {
			// verify that there are no duplicate boot orders
			if bootOrderMap[order] {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("Boot order for %s already set for a different device.", field.Child("domain", "devices", "interfaces").Index(idx).Child("bootOrder").String()),
					Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("bootOrder").String(),
				})
			}
			bootOrderMap[order] = true
		}
	}
	return causes
}

func validateMacAddress(field *k8sfield.Path, iface v1.Interface, idx int) (causes []metav1.StatusCause) {
	if iface.MacAddress != "" {
		mac, err := net.ParseMAC(iface.MacAddress)
		if err != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("interface %s has malformed MAC address (%s).", field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(), iface.MacAddress),
				Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("macAddress").String(),
			})
		}
		if len(mac) > 6 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("interface %s has MAC address (%s) that is too long.", field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(), iface.MacAddress),
				Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("macAddress").String(),
			})
		}
	}
	return causes
}

func validateInterfaceModel(field *k8sfield.Path, iface v1.Interface, idx int) (causes []metav1.StatusCause) {
	if iface.Model != "" {
		if _, exists := validInterfaceModels[iface.Model]; !exists {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: fmt.Sprintf("interface %s uses model %s that is not supported.", field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(), iface.Model),
				Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("model").String(),
			})
		}
	}
	return causes
}

func validatePortConfiguration(field *k8sfield.Path, networkExists bool, networkData *v1.Network, iface v1.Interface, idx int, portForwardMap map[string]struct{}) (causes []metav1.StatusCause) {

	// Check only ports configured on interfaces connected to a pod network
	if networkExists && networkData.Pod != nil && iface.Ports != nil {
		for portIdx, forwardPort := range iface.Ports {
			causes = append(causes, validateForwardPortNonZero(field, forwardPort, idx, portIdx)...)
			causes = append(causes, validateForwardPortInRange(field, forwardPort, idx, portIdx)...)
			causes = append(causes, validateForwardPortProtocol(field, forwardPort, idx, portIdx)...)
			causes = append(causes, validateForwardPortName(field, forwardPort, portForwardMap, idx, portIdx)...)
		}
	}
	return causes
}

func validateForwardPortName(field *k8sfield.Path, forwardPort v1.Port, portForwardMap map[string]struct{}, idx int, portIdx int) (causes []metav1.StatusCause) {
	if forwardPort.Name != "" {
		if _, ok := portForwardMap[forwardPort.Name]; ok {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueDuplicate,
				Message: fmt.Sprintf("Duplicate name of the port: %s", forwardPort.Name),
				Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("ports").Index(portIdx).Child("name").String(),
			})
		}

		if msgs := validation.IsValidPortName(forwardPort.Name); len(msgs) != 0 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Invalid name of the port: %s", forwardPort.Name),
				Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("ports").Index(portIdx).Child("name").String(),
			})
		}

		portForwardMap[forwardPort.Name] = struct{}{}
	}
	return causes
}

func validateForwardPortProtocol(field *k8sfield.Path, forwardPort v1.Port, idx int, portIdx int) (causes []metav1.StatusCause) {
	if forwardPort.Protocol != "" {
		if forwardPort.Protocol != "TCP" && forwardPort.Protocol != "UDP" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "Unknown protocol, only TCP or UDP allowed",
				Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("ports").Index(portIdx).Child("protocol").String(),
			})
		}
	} else {
		forwardPort.Protocol = "TCP"
	}
	return causes
}

func validateForwardPortInRange(field *k8sfield.Path, forwardPort v1.Port, idx int, portIdx int) (causes []metav1.StatusCause) {
	if forwardPort.Port < 0 || forwardPort.Port > 65536 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Port field must be in range 0 < x < 65536.",
			Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("ports").Index(portIdx).String(),
		})
	}
	return causes
}

func validateForwardPortNonZero(field *k8sfield.Path, forwardPort v1.Port, idx int, portIdx int) (causes []metav1.StatusCause) {
	if forwardPort.Port == 0 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "Port field is mandatory.",
			Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("ports").Index(portIdx).String(),
		})
	}
	return causes
}

func appendStatusCauseForMacvtapOnlyAllowedWithMultus(field *k8sfield.Path, causes []metav1.StatusCause, idx int) []metav1.StatusCause {
	causes = append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: "Macvtap interface only implemented with Multus network",
		Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
	})
	return causes
}

func appendStatusCauseForMacvtapFeatureGateNotEnabled(field *k8sfield.Path, causes []metav1.StatusCause, idx int) []metav1.StatusCause {
	causes = append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: "Macvtap feature gate is not enabled",
		Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
	})
	return causes
}

func appendStatusCauseForBridgeNotEnabled(field *k8sfield.Path, causes []metav1.StatusCause, idx int) []metav1.StatusCause {
	causes = append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: "Bridge on pod network configuration is not enabled under kubevirt-config",
		Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
	})
	return causes
}

func appendStatusCauseForMasqueradeWithoutPodNetwork(field *k8sfield.Path, causes []metav1.StatusCause, idx int) []metav1.StatusCause {
	causes = append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: "Masquerade interface only implemented with pod network",
		Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
	})
	return causes
}

func appendStatusCauseForSlirpNotEnabled(field *k8sfield.Path, causes []metav1.StatusCause, idx int) []metav1.StatusCause {
	causes = append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: "Slirp interface is not enabled in kubevirt-config",
		Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
	})
	return causes
}

func appendStatusCauseForSlirpWithoutPodNetwork(field *k8sfield.Path, causes []metav1.StatusCause, idx int) []metav1.StatusCause {
	return append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: "Slirp interface only implemented with pod network",
		Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
	})
}

func appendStatusCauseForNetworkNotFound(field *k8sfield.Path, causes []metav1.StatusCause, idx int, iface v1.Interface) []metav1.StatusCause {
	causes = append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: fmt.Sprintf(nameOfTypeNotFoundMessagePattern, field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(), iface.Name),
		Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
	})
	return causes
}

func appendStatusCauseForInvalidMasqueradeMacAddress(field *k8sfield.Path, causes []metav1.StatusCause, idx int) []metav1.StatusCause {
	causes = append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: "The requested MAC address is reserved for the in-pod bridge. Please choose another one.",
		Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("macAddress").String(),
	})
	return causes
}

func appendStatusCauseForPasstFeatureGateNotEnabled(field *k8sfield.Path, causes []metav1.StatusCause, idx int) []metav1.StatusCause {
	causes = append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: "Passt feature gate is not enabled",
		Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
	})
	return causes
}

func appendStatusCauseForPasstWithoutPodNetwork(field *k8sfield.Path, causes []metav1.StatusCause, idx int) []metav1.StatusCause {
	return append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: "Passt interface only implemented with pod network",
		Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
	})
}

func appendStatusCauseForPasstWithMultipleInterfaces(field *k8sfield.Path, causes []metav1.StatusCause, idx int) []metav1.StatusCause {
	return append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: "Passt interface is only supported as the single interface of the VMI",
		Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
	})
}

func appendStatusCauseForBindingPluginsFeatureGateNotEnabled(field *k8sfield.Path, causes []metav1.StatusCause, idx int) []metav1.StatusCause {
	causes = append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: "Binding plugins feature gate is not enabled",
		Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
	})
	return causes
}

func validateInterfaceNameFormat(field *k8sfield.Path, iface v1.Interface, idx int) (causes []metav1.StatusCause) {
	isValid := regexp.MustCompile(`^[A-Za-z0-9-_]+$`).MatchString
	if !isValid(iface.Name) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Network interface name can only contain alphabetical characters, numbers, dashes (-) or underscores (_)",
			Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
		})
	}
	return causes
}

func validateInterfaceNameUnique(field *k8sfield.Path, networkInterfaceMap map[string]struct{}, iface v1.Interface, idx int) (causes []metav1.StatusCause) {
	if _, networkAlreadyUsed := networkInterfaceMap[iface.Name]; networkAlreadyUsed {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueDuplicate,
			Message: "Only one interface can be connected to one specific network",
			Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
		})
	}
	return causes
}

func validateNetworksAssignedToInterfaces(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, networkInterfaceMap map[string]struct{}) (causes []metav1.StatusCause) {
	networkDuplicates := map[string]struct{}{}
	for i, network := range spec.Networks {
		if _, exists := networkDuplicates[network.Name]; exists {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueDuplicate,
				Message: fmt.Sprintf("Network with name %q already exists, every network must have a unique name", network.Name),
				Field:   field.Child("networks").Index(i).Child("name").String(),
			})
		}
		networkDuplicates[network.Name] = struct{}{}
		if _, exists := networkInterfaceMap[network.Name]; !exists {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: fmt.Sprintf(nameOfTypeNotFoundMessagePattern, field.Child("networks").Index(i).Child("name").String(), network.Name),
				Field:   field.Child("networks").Index(i).Child("name").String(),
			})
		}
	}
	return causes
}

func validateInputDevices(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	for idx, input := range spec.Domain.Devices.Inputs {
		if input.Bus != v1.InputBusVirtio && input.Bus != v1.InputBusUSB && input.Bus != "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "Input device can have only virtio or usb bus.",
				Field:   field.Child("domain", "devices", "inputs").Index(idx).Child("bus").String(),
			})
		}

		if input.Type != v1.InputTypeTablet {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "Input device can have only tablet type.",
				Field:   field.Child("domain", "devices", "inputs").Index(idx).Child("type").String(),
			})
		}
	}
	return causes
}

func validateIOThreadsPolicy(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.IOThreadsPolicy != nil {
		isValidPolicy := func(policy v1.IOThreadsPolicy) bool {
			for _, p := range validIOThreadsPolicies {
				if policy == p {
					return true
				}
			}
			return false
		}
		if !isValidPolicy(*spec.Domain.IOThreadsPolicy) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Invalid IOThreadsPolicy (%s)", *spec.Domain.IOThreadsPolicy),
				Field:   field.Child("domain", "ioThreadsPolicy").String(),
			})
		}
	}
	return causes
}

func validateProbe(field *k8sfield.Path, probe *v1.Probe) (causes []metav1.StatusCause) {
	if probe == nil {
		return causes
	}
	numHandlers := 0

	if probe.HTTPGet != nil {
		numHandlers++
	}
	if probe.TCPSocket != nil {
		numHandlers++
	}
	if probe.Exec != nil {
		numHandlers++
	}
	if probe.GuestAgentPing != nil {
		numHandlers++
	}

	if numHandlers > 1 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s must have exactly one probe type set", field),
			Field:   field.String(),
		})
	}

	if numHandlers < 1 {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("either %s, %s or %s must be set if a %s is specified",
				field.Child("tcpSocket").String(),
				field.Child("exec").String(),
				field.Child("httpGet").String(),
				field,
			),
			Field: field.String(),
		})
	}

	return causes
}

func appendStatusCauseForProbeNotAllowedWithNoPodNetworkPresent(field *k8sfield.Path, probe *v1.Probe, causes []metav1.StatusCause) []metav1.StatusCause {
	if probe == nil {
		return causes
	}

	if probe.HTTPGet != nil {
		causes = append(causes, podNetworkRequiredStatusCause(field.Child("httpGet")))
	}

	if probe.TCPSocket != nil {
		causes = append(causes, podNetworkRequiredStatusCause(field.Child("tcpSocket")))
	}
	return causes
}

func podNetworkRequiredStatusCause(field *k8sfield.Path) metav1.StatusCause {
	return metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: fmt.Sprintf("%s is only allowed if the Pod Network is attached", field.String()),
		Field:   field.String(),
	}
}

func isValidEvictionStrategy(evictionStrategy *v1.EvictionStrategy) bool {
	return evictionStrategy == nil ||
		*evictionStrategy == v1.EvictionStrategyLiveMigrate ||
		*evictionStrategy == v1.EvictionStrategyLiveMigrateIfPossible ||
		*evictionStrategy == v1.EvictionStrategyNone ||
		*evictionStrategy == v1.EvictionStrategyExternal
}

func validateLiveMigration(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) (causes []metav1.StatusCause) {
	evictionStrategy := config.GetConfig().EvictionStrategy

	if spec.EvictionStrategy != nil {
		evictionStrategy = spec.EvictionStrategy
	}
	if !isValidEvictionStrategy(evictionStrategy) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s is set with an unrecognized option: %s", field.Child("evictionStrategy").String(), *spec.EvictionStrategy),
			Field:   field.Child("evictionStrategy").String(),
		})
	}
	return causes
}

func validateGPUsWithPassthroughEnabled(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) (causes []metav1.StatusCause) {
	if spec.Domain.Devices.GPUs != nil && !config.GPUPassthroughEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "GPU feature gate is not enabled in kubevirt-config",
			Field:   field.Child("GPUs").String(),
		})
	}
	return causes
}

// Returns true if any of the filesystems requires a virtiofs container to run as root
func requireAnyVirtiofsPrivilegedContainer(spec *v1.VirtualMachineInstanceSpec) bool {
	if spec.Domain.Devices.Filesystems == nil {
		return false
	}

	volumes := make(map[string]v1.Volume)
	for _, volume := range spec.Volumes {
		volumes[volume.Name] = volume
	}

	// we assume that all filesystems are of type virtiofs
	for _, fs := range spec.Domain.Devices.Filesystems {
		volume, ok := volumes[fs.Name]
		if !ok {
			continue
		}

		if virtiofs.RequiresRootPrivileges(&volume) {
			return true
		}
	}

	return false
}

func validateFilesystemsWithVirtIOFSEnabled(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) (causes []metav1.StatusCause) {
	// The feature gate is mandatory only for a privileged container
	if requireAnyVirtiofsPrivilegedContainer(spec) && !config.VirtiofsEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "virtiofs feature gate is not enabled in kubevirt-config",
			Field:   field.Child("Filesystems").String(),
		})
	}
	return causes
}

func validateHostDevicesWithPassthroughEnabled(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) (causes []metav1.StatusCause) {
	if spec.Domain.Devices.HostDevices != nil && !config.HostDevicesPassthroughEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Host Devices feature gate is not enabled in kubevirt-config",
			Field:   field.Child("HostDevices").String(),
		})
	}
	return causes
}

func validateSoundDevices(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.Devices.Sound != nil {
		model := spec.Domain.Devices.Sound.Model
		if model != "" && model != "ich9" && model != "ac97" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "Sound device type is not supported. Options: 'ich9' or 'ac97'",
				Field:   field.Child("Sound").String(),
			})
		}
		name := spec.Domain.Devices.Sound.Name
		if name == "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "Sound device requires a name field.",
				Field:   field.Child("Sound").String(),
			})
		}
	}
	return causes
}

func validateLaunchSecurity(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) (causes []metav1.StatusCause) {
	launchSecurity := spec.Domain.LaunchSecurity
	if launchSecurity != nil && !config.WorkloadEncryptionSEVEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s feature gate is not enabled in kubevirt-config", virtconfig.WorkloadEncryptionSEV),
			Field:   field.Child("launchSecurity").String(),
		})
	} else if launchSecurity != nil && launchSecurity.SEV != nil {
		firmware := spec.Domain.Firmware
		if firmware == nil || firmware.Bootloader == nil || firmware.Bootloader.EFI == nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "SEV requires OVMF (UEFI)",
				Field:   field.Child("launchSecurity").String(),
			})
		} else if firmware.Bootloader.EFI.SecureBoot == nil || *firmware.Bootloader.EFI.SecureBoot {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "SEV does not work along with SecureBoot",
				Field:   field.Child("launchSecurity").String(),
			})
		}

		startStrategy := spec.StartStrategy
		if launchSecurity.SEV.Attestation != nil && (startStrategy == nil || *startStrategy != v1.StartStrategyPaused) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("SEV attestation requires VMI StartStrategy '%s'", v1.StartStrategyPaused),
				Field:   field.Child("launchSecurity").String(),
			})
		}

		for _, iface := range spec.Domain.Devices.Interfaces {
			if iface.BootOrder != nil {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("SEV does not work with bootable NICs: %s", iface.Name),
					Field:   field.Child("launchSecurity").String(),
				})
			}
		}
	}
	return causes
}

func appendStatusCauseForPodNetworkDefinedWithMultusDefaultNetworkDefined(field *k8sfield.Path, causes []metav1.StatusCause) []metav1.StatusCause {
	return append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: "Pod network cannot be defined when Multus default network is defined",
		Field:   field.Child("networks").String(),
	})
}

func appendStatusCaseForMoreThanOneMultusDefaultNetwork(field *k8sfield.Path, causes []metav1.StatusCause) []metav1.StatusCause {
	return append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: "Multus CNI should only have one default network",
		Field:   field.Child("networks").String(),
	})
}

func validateNetworks(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, networkNameMap map[string]*v1.Network) (podExists bool, multusDefaultCount int, causes []metav1.StatusCause) {

	podExists = false
	multusDefaultCount = 0

	for idx, network := range spec.Networks {

		cniTypesCount := 0
		// network name not needed by default
		networkNameExistsOrNotNeeded := true

		if network.Pod != nil {
			cniTypesCount++
			podExists = true
		}

		if network.NetworkSource.Multus != nil {
			cniTypesCount++
			networkNameExistsOrNotNeeded = network.Multus.NetworkName != ""
			if network.NetworkSource.Multus.Default {
				multusDefaultCount++
			}
		}

		causes = validateNetworkHasOnlyOneType(field, cniTypesCount, causes, idx)

		if !networkNameExistsOrNotNeeded {
			causes = appendStatusCauseForCNIPluginHasNoNetworkName(field, causes, idx)
		}

		networkNameMap[spec.Networks[idx].Name] = &spec.Networks[idx]
	}
	return podExists, multusDefaultCount, causes
}

func appendStatusCauseForCNIPluginHasNoNetworkName(field *k8sfield.Path, incomingCauses []metav1.StatusCause, idx int) (causes []metav1.StatusCause) {
	causes = append(incomingCauses, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueRequired,
		Message: "CNI delegating plugin must have a networkName",
		Field:   field.Child("networks").Index(idx).String(),
	})
	return causes
}

func validateNetworkHasOnlyOneType(field *k8sfield.Path, cniTypesCount int, causes []metav1.StatusCause, idx int) []metav1.StatusCause {
	if cniTypesCount == 0 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "should have a network type",
			Field:   field.Child("networks").Index(idx).String(),
		})
	} else if cniTypesCount > 1 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "should have only one network type",
			Field:   field.Child("networks").Index(idx).String(),
		})
	}
	return causes
}

func validateBootOrder(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, volumeNameMap map[string]*v1.Volume) (bootOrderMap map[uint]bool, causes []metav1.StatusCause) {
	// used to validate uniqueness of boot orders among disks and interfaces
	bootOrderMap = make(map[uint]bool)

	for i, volume := range spec.Volumes {
		volumeNameMap[volume.Name] = &spec.Volumes[i]
	}

	// Validate disks match volumes correctly
	for idx, disk := range spec.Domain.Devices.Disks {
		var matchingVolume *v1.Volume

		matchingVolume, volumeExists := volumeNameMap[disk.Name]

		if !volumeExists {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf(nameOfTypeNotFoundMessagePattern, field.Child("domain", "devices", "disks").Index(idx).Child("Name").String(), disk.Name),
				Field:   field.Child("domain", "devices", "disks").Index(idx).Child("name").String(),
			})
		}

		// Verify Lun disks are only mapped to network/block devices.
		if disk.LUN != nil && volumeExists && matchingVolume.PersistentVolumeClaim == nil && matchingVolume.DataVolume == nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s can only be mapped to a DataVolume or PersistentVolumeClaim volume.", field.Child("domain", "devices", "disks").Index(idx).Child("lun").String()),
				Field:   field.Child("domain", "devices", "disks").Index(idx).Child("lun").String(),
			})
		}

		// Verify that DownwardMetrics is mapped to disk
		if volumeExists && matchingVolume.DownwardMetrics != nil {
			if disk.Disk == nil {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueRequired,
					Message: fmt.Sprintf("DownwardMetrics volume must be mapped to a disk, but disk is not set on %v.", field.Child("domain", "devices", "disks").Index(idx).Child("disk").String()),
					Field:   field.Child("domain", "devices", "disks").Index(idx).Child("disk").String(),
				})
			} else if disk.Disk != nil && disk.Disk.Bus != v1.DiskBusVirtio && disk.Disk.Bus != "" {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("DownwardMetrics volume must be mapped to virtio bus, but %v is set to %v", field.Child("domain", "devices", "disks").Index(idx).Child("disk").Child("bus").String(), disk.Disk.Bus),
					Field:   field.Child("domain", "devices", "disks").Index(idx).Child("disk").Child("bus").String(),
				})
			}
		}

		// verify that there are no duplicate boot orders
		if disk.BootOrder != nil {
			order := *disk.BootOrder
			if bootOrderMap[order] {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("Boot order for %s already set for a different device.", field.Child("domain", "devices", "disks").Index(idx).Child("bootOrder").String()),
					Field:   field.Child("domain", "devices", "disks").Index(idx).Child("bootOrder").String(),
				})
			}
			bootOrderMap[order] = true
		}
	}

	return bootOrderMap, causes
}

func appendStatusCauseForMoreThanOnePodInterface(field *k8sfield.Path, causes []metav1.StatusCause) []metav1.StatusCause {
	return append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueDuplicate,
		Message: fmt.Sprintf("more than one interface is connected to a pod network in %s", field.Child("interfaces").String()),
		Field:   field.Child("interfaces").String(),
	})
}

func appendStatusCauseMaxNumberOfNetworksExceeded(field *k8sfield.Path, causes []metav1.StatusCause) []metav1.StatusCause {
	return append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: fmt.Sprintf(listExceedsLimitMessagePattern, field.Child("networks").String(), arrayLenMax),
		Field:   field.Child("networks").String(),
	})
}

func appendStatusCauseForMaxNumberOfInterfacesExceeded(field *k8sfield.Path, causes []metav1.StatusCause) []metav1.StatusCause {
	return append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: fmt.Sprintf(listExceedsLimitMessagePattern, field.Child("domain", "devices", "interfaces").String(), arrayLenMax),
		Field:   field.Child("domain", "devices", "interfaces").String(),
	})
}

func validateCPUFeaturePolicies(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.CPU != nil && spec.Domain.CPU.Features != nil {
		for idx, feature := range spec.Domain.CPU.Features {
			if _, exists := validCPUFeaturePolicies[feature.Policy]; !exists {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueNotSupported,
					Message: fmt.Sprintf("CPU feature %s uses policy %s that is not supported.", feature.Name, feature.Policy),
					Field:   field.Child("domain", "cpu", "features").Index(idx).Child("policy").String(),
				})
			}
		}
	}
	return causes
}

func validateCPUIsolatorThread(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.CPU != nil && spec.Domain.CPU.IsolateEmulatorThread && !spec.Domain.CPU.DedicatedCPUPlacement {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "IsolateEmulatorThread should be only set in combination with DedicatedCPUPlacement",
			Field:   field.Child("domain", "cpu", "isolateEmulatorThread").String(),
		})
	}
	return causes
}

func validateCpuPinning(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) (causes []metav1.StatusCause) {
	if spec.Domain.CPU != nil && spec.Domain.CPU.DedicatedCPUPlacement {
		causes = append(causes, validateMemoryLimitAndRequestProvided(field, spec)...)
		causes = append(causes, validateCPURequestIsInteger(field, spec)...)
		causes = append(causes, validateCPULimitIsInteger(field, spec)...)
		causes = append(causes, validateMemoryRequestsAndLimits(field, spec)...)
		causes = append(causes, validateRequestLimitOrCoresProvidedOnDedicatedCPUPlacement(field, spec)...)
		causes = append(causes, validateRequestEqualsLimitOnDedicatedCPUPlacement(field, spec)...)
		causes = append(causes, validateRequestOrLimitWithCoresProvidedOnDedicatedCPUPlacement(field, spec)...)
		causes = append(causes, validateThreadCountOnArchitecture(field, spec, config)...)
		causes = append(causes, validateThreadCountOnDedicatedCPUPlacement(field, spec)...)
	}
	return causes
}

func validateNUMA(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) (causes []metav1.StatusCause) {
	if spec.Domain.CPU != nil && spec.Domain.CPU.NUMA != nil && spec.Domain.CPU.NUMA.GuestMappingPassthrough != nil {
		if !config.NUMAEnabled() {
			causes = append(causes, metav1.StatusCause{
				Type: metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("NUMA feature gate is not enabled in kubevirt-config, invalid entry %s",
					field.Child("domain", "cpu", "numa", "guestMappingPassthrough").String()),
				Field: field.Child("domain", "cpu", "numa", "guestMappingPassthrough").String(),
			})
		}
		if !spec.Domain.CPU.DedicatedCPUPlacement {
			causes = append(causes, metav1.StatusCause{
				Type: metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s must be set to true when NUMA topology strategy is set in %s",
					field.Child("domain", "cpu", "dedicatedCpuPlacement").String(),
					field.Child("domain", "cpu", "numa", "guestMappingPassthrough").String(),
				),
				Field: field.Child("domain", "cpu", "numa", "guestMappingPassthrough").String(),
			})
		}
		if spec.Domain.Memory == nil || spec.Domain.Memory.Hugepages == nil {
			causes = append(causes, metav1.StatusCause{
				Type: metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s must be requested when NUMA topology strategy is set in %s",
					field.Child("domain", "memory", "hugepages").String(),
					field.Child("domain", "cpu", "numa", "guestMappingPassthrough").String(),
				),
				Field: field.Child("domain", "cpu", "numa", "guestMappingPassthrough").String(),
			})
		}
	}
	return causes
}

func validateThreadCountOnArchitecture(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) (causes []metav1.StatusCause) {
	arch := spec.Architecture
	if arch == "" {
		arch = config.GetDefaultArchitecture()
	}

	// Verify CPU thread count requested is 1 for ARM64 VMI architecture.
	if spec.Domain.CPU != nil && spec.Domain.CPU.Threads > 1 && virtconfig.IsARM64(arch) {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("threads must not be greater than 1 at %v (got %v) when %v is arm64",
				field.Child("domain", "cpu", "threads").String(),
				spec.Domain.CPU.Threads,
				field.Child("architecture").String(),
			),
			Field: field.Child("architecture").String(),
		})
	}
	return causes
}

func validateThreadCountOnDedicatedCPUPlacement(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.CPU != nil && spec.Domain.CPU.Threads > 2 {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Not more than two threads must be provided at %v (got %v) when DedicatedCPUPlacement is true",
				field.Child("domain", "cpu", "threads").String(),
				spec.Domain.CPU.Threads,
			),
			Field: field.Child("domain", "cpu", "dedicatedCpuPlacement").String(),
		})
	}
	return causes
}

func validateRequestOrLimitWithCoresProvidedOnDedicatedCPUPlacement(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if (spec.Domain.Resources.Requests.Cpu().Value() > 0 || spec.Domain.Resources.Limits.Cpu().Value() > 0) && hwutil.GetNumberOfVCPUs(spec.Domain.CPU) > 0 &&
		spec.Domain.Resources.Requests.Cpu().Value() != hwutil.GetNumberOfVCPUs(spec.Domain.CPU) && spec.Domain.Resources.Limits.Cpu().Value() != hwutil.GetNumberOfVCPUs(spec.Domain.CPU) {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s or %s must not be provided at the same time with %s when DedicatedCPUPlacement is true ",
				field.Child("domain", "resources", "requests", "cpu").String(),
				field.Child("domain", "resources", "limits", "cpu").String(),
				field.Child("domain", "cpu", "cores").String(),
			),
			Field: field.Child("domain", "cpu", "dedicatedCpuPlacement").String(),
		})
	}
	return causes
}

func validateRequestEqualsLimitOnDedicatedCPUPlacement(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.Resources.Requests.Cpu().Value() > 0 && spec.Domain.Resources.Limits.Cpu().Value() > 0 && spec.Domain.Resources.Requests.Cpu().Value() != spec.Domain.Resources.Limits.Cpu().Value() {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s or %s must be equal when DedicatedCPUPlacement is true ",
				field.Child("domain", "resources", "requests", "cpu").String(),
				field.Child("domain", "resources", "limits", "cpu").String(),
			),
			Field: field.Child("domain", "cpu", "dedicatedCpuPlacement").String(),
		})
	}
	return causes
}

func validateRequestLimitOrCoresProvidedOnDedicatedCPUPlacement(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.Resources.Requests.Cpu().Value() == 0 && spec.Domain.Resources.Limits.Cpu().Value() == 0 && hwutil.GetNumberOfVCPUs(spec.Domain.CPU) == 0 {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("either %s or %s or %s must be provided when DedicatedCPUPlacement is true ",
				field.Child("domain", "resources", "requests", "cpu").String(),
				field.Child("domain", "resources", "limits", "cpu").String(),
				field.Child("domain", "cpu", "cores").String(),
			),
			Field: field.Child("domain", "cpu", "dedicatedCpuPlacement").String(),
		})
	}
	return causes
}

func validateStartStrategy(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.StartStrategy != nil {
		if *spec.StartStrategy != v1.StartStrategyPaused {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s is set with an unrecognized option: %s", field.Child("startStrategy").String(), *spec.StartStrategy),
				Field:   field.Child("startStrategy").String(),
			})
		} else if spec.LivenessProbe != nil {
			causes = append(causes, metav1.StatusCause{
				Type: metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("either %s or %s should be provided.Pausing VMI with LivenessProbe is not supported",
					field.Child("startStrategy").String(),
					field.Child("livenessProbe").String(),
				),
				Field: field.Child("startStrategy").String(),
			})
		}
	}
	return causes
}

func validateMemoryRequestsAndLimits(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.Resources.Requests.Memory().Value() > 0 && spec.Domain.Resources.Limits.Memory().Value() > 0 && spec.Domain.Resources.Requests.Memory().Value() != spec.Domain.Resources.Limits.Memory().Value() {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s must be equal to %s",
				field.Child("domain", "resources", "requests", "memory").String(),
				field.Child("domain", "resources", "limits", "memory").String(),
			),
			Field: field.Child("domain", "resources", "requests", "memory").String(),
		})
	}
	return causes
}

func validateCPULimitIsInteger(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.Resources.Limits.Cpu().Value() > 0 && spec.Domain.Resources.Limits.Cpu().Value()*1000 != spec.Domain.Resources.Limits.Cpu().MilliValue() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "provided resources CPU limits must be an interger",
			Field:   field.Child("domain", "resources", "limits", "cpu").String(),
		})
	}
	return causes
}

func validateCPURequestIsInteger(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.Resources.Requests.Cpu().Value() > 0 && spec.Domain.Resources.Requests.Cpu().Value()*1000 != spec.Domain.Resources.Requests.Cpu().MilliValue() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "provided resources CPU requests must be an interger",
			Field:   field.Child("domain", "resources", "requests", "cpu").String(),
		})
	}
	return causes
}

func validateMemoryLimitAndRequestProvided(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.Resources.Limits.Memory().Value() == 0 && spec.Domain.Resources.Requests.Memory().Value() == 0 {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s or %s should be provided",
				field.Child("domain", "resources", "requests", "memory").String(),
				field.Child("domain", "resources", "limits", "memory").String(),
			),
			Field: field.Child("domain", "resources", "limits", "memory").String(),
		})
	}
	return causes
}

func validateCpuRequestDoesNotExceedLimit(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.Resources.Limits.Cpu().MilliValue() > 0 &&
		spec.Domain.Resources.Requests.Cpu().MilliValue() > spec.Domain.Resources.Limits.Cpu().MilliValue() {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s '%s' is greater than %s '%s'", field.Child("domain", "resources", "requests", "cpu").String(),
				spec.Domain.Resources.Requests.Cpu(),
				field.Child("domain", "resources", "limits", "cpu").String(),
				spec.Domain.Resources.Limits.Cpu()),
			Field: field.Child("domain", "resources", "requests", "cpu").String(),
		})
	}
	return causes
}

func validateCPULimitNotNegative(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.Resources.Limits.Cpu().MilliValue() < 0 {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf(valueMustBePositiveMessagePattern, field.Child("domain", "resources", "limits", "cpu").String(),
				spec.Domain.Resources.Limits.Cpu()),
			Field: field.Child("domain", "resources", "limits", "cpu").String(),
		})
	}
	return causes
}

func validateCPURequestNotNegative(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.Resources.Requests.Cpu().MilliValue() < 0 {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf(valueMustBePositiveMessagePattern, field.Child("domain", "resources", "requests", "cpu").String(),
				spec.Domain.Resources.Requests.Cpu()),
			Field: field.Child("domain", "resources", "requests", "cpu").String(),
		})
	}
	return causes
}

func validateFirmwareSerial(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.Firmware != nil && len(spec.Domain.Firmware.Serial) > 0 {
		// Verify serial number is within valid length, if provided
		if len(spec.Domain.Firmware.Serial) > maxStrLen {
			causes = append(causes, metav1.StatusCause{
				Type: metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s must be less than or equal to %d in length, if specified",
					field.Child("domain", "firmware", "serial").String(),
					maxStrLen,
				),
				Field: field.Child("domain", "firmware", "serial").String(),
			})
		}
		// Verify serial number is made up of valid characters for libvirt, if provided
		isValid := regexp.MustCompile(`^[A-Za-z0-9_.+-]+$`).MatchString
		if !isValid(spec.Domain.Firmware.Serial) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s must be made up of the following characters [A-Za-z0-9_.+-], if specified", field.Child("domain", "firmware", "serial").String()),
				Field:   field.Child("domain", "firmware", "serial").String(),
			})
		}
	}
	return causes
}

func validateEmulatedMachine(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) (causes []metav1.StatusCause) {
	if machine := spec.Domain.Machine; machine != nil && len(machine.Type) > 0 {
		supportedMachines := config.GetEmulatedMachines(spec.Architecture)
		var match = false
		for _, val := range supportedMachines {
			if regexp.MustCompile(val).MatchString(machine.Type) {
				match = true
				break
			}
		}
		if !match {
			causes = append(causes, metav1.StatusCause{
				Type: metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s is not supported: %s (allowed values: %v)",
					field.Child("domain", "machine", "type").String(),
					machine.Type,
					supportedMachines,
				),
				Field: field.Child("domain", "machine", "type").String(),
			})
		}
	}
	return causes
}

func validateGuestMemoryLimit(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.Memory != nil && spec.Domain.Memory.Guest != nil {
		limits := spec.Domain.Resources.Limits.Memory().Value()
		guest := spec.Domain.Memory.Guest.Value()
		if limits < guest && limits != 0 {
			causes = append(causes, metav1.StatusCause{
				Type: metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s '%s' must be equal to or less than the memory limit %s '%s'",
					field.Child("domain", "memory", "guest").String(),
					spec.Domain.Memory.Guest,
					field.Child("domain", "resources", "limits", "memory").String(),
					spec.Domain.Resources.Limits.Memory(),
				),
				Field: field.Child("domain", "memory", "guest").String(),
			})
		}
	}
	return causes
}

func validateHugepagesMemoryRequests(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.Memory != nil && spec.Domain.Memory.Hugepages != nil {
		hugepagesSize, err := resource.ParseQuantity(spec.Domain.Memory.Hugepages.PageSize)
		if err != nil {
			causes = append(causes, metav1.StatusCause{
				Type: metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s '%s': %s",
					field.Child("domain", "hugepages", "size").String(),
					spec.Domain.Memory.Hugepages.PageSize,
					resource.ErrFormatWrong,
				),
				Field: field.Child("domain", "hugepages", "size").String(),
			})
		} else {
			vmMemory := spec.Domain.Resources.Requests.Memory().Value()
			if vmMemory < hugepagesSize.Value() {
				causes = append(causes, metav1.StatusCause{
					Type: metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s '%s' must be equal to or larger than page size %s '%s'",
						field.Child("domain", "resources", "requests", "memory").String(),
						spec.Domain.Resources.Requests.Memory(),
						field.Child("domain", "hugepages", "size").String(),
						spec.Domain.Memory.Hugepages.PageSize,
					),
					Field: field.Child("domain", "resources", "requests", "memory").String(),
				})
			} else if vmMemory%hugepagesSize.Value() != 0 {
				causes = append(causes, metav1.StatusCause{
					Type: metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s '%s' is not a multiple of the page size %s '%s'",
						field.Child("domain", "resources", "requests", "memory").String(),
						spec.Domain.Resources.Requests.Memory(),
						field.Child("domain", "hugepages", "size").String(),
						spec.Domain.Memory.Hugepages.PageSize,
					),
					Field: field.Child("domain", "resources", "requests", "memory").String(),
				})
			}
		}
	}
	return causes
}

func validateMemoryLimitsNegativeOrNull(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.Resources.Limits.Memory().Value() < 0 {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf(valueMustBePositiveMessagePattern, field.Child("domain", "resources", "limits", "memory").String(),
				spec.Domain.Resources.Limits.Memory()),
			Field: field.Child("domain", "resources", "limits", "memory").String(),
		})
	}

	if spec.Domain.Resources.Limits.Memory().Value() > 0 &&
		spec.Domain.Resources.Requests.Memory().Value() > spec.Domain.Resources.Limits.Memory().Value() {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s '%s' is greater than %s '%s'", field.Child("domain", "resources", "requests", "memory").String(),
				spec.Domain.Resources.Requests.Memory(),
				field.Child("domain", "resources", "limits", "memory").String(),
				spec.Domain.Resources.Limits.Memory()),
			Field: field.Child("domain", "resources", "requests", "memory").String(),
		})
	}
	return causes
}

func validateMemoryRequestsNegativeOrNull(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.Resources.Requests.Memory().Value() < 0 {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf(valueMustBePositiveMessagePattern, field.Child("domain", "resources", "requests", "memory").String(),
				spec.Domain.Resources.Requests.Memory()),
			Field: field.Child("domain", "resources", "requests", "memory").String(),
		})
	} else if spec.Domain.Resources.Requests.Memory().Value() > 0 && spec.Domain.Resources.Requests.Memory().Cmp(resource.MustParse("1M")) < 0 {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s '%s': must be greater than or equal to 1M.", field.Child("domain", "resources", "requests", "memory").String(),
				spec.Domain.Resources.Requests.Memory()),
			Field: field.Child("domain", "resources", "requests", "memory").String(),
		})
	}
	return causes
}

func validateSubdomainDNSSubdomainRules(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Subdomain != "" {
		errors := validation.IsDNS1123Subdomain(spec.Subdomain)
		if len(errors) != 0 {
			causes = append(causes, metav1.StatusCause{
				Type: metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s does not conform to the kubernetes DNS_SUBDOMAIN rules : %s",
					field.Child("subdomain").String(), strings.Join(errors, ", ")),
				Field: field.Child("subdomain").String(),
			})
		}
	}
	return causes
}

func validateHostNameNotConformingToDNSLabelRules(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Hostname != "" {
		errors := validation.IsDNS1123Label(spec.Hostname)
		if len(errors) != 0 {
			causes = appendNewStatusCauseForHostNameNotConformingToDNSLabelRules(field, causes, errors)
		}
	}
	return causes
}

func validateRealtime(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.CPU != nil && spec.Domain.CPU.Realtime != nil {
		causes = append(causes, validateCPURealtime(field, spec)...)
		causes = append(causes, validateMemoryRealtime(field, spec)...)
	}
	return causes
}

func validateArchitecture(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) (causes []metav1.StatusCause) {
	if spec.Architecture != "" && spec.Architecture != runtime.GOARCH && !config.MultiArchitectureEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("multi-architecture feature gate is not enabled in kubevirt-config, invalid entry %s",
				field.Child("architecture").String()),
			Field: field.Child("architecture").String(),
		})

	}
	return causes
}

func validateContainerDisks(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	for idx, volume := range spec.Volumes {
		if volume.ContainerDisk == nil || volume.ContainerDisk.Path == "" {
			continue
		}
		causes = append(causes, validatePath(field.Child("volumes").Index(idx).Child("containerDisk"), volume.ContainerDisk.Path)...)
	}
	return causes
}

func validatePath(field *k8sfield.Path, path string) (causes []metav1.StatusCause) {
	if path == "/" {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s must not point to root",
				field.String(),
			),
			Field: field.String(),
		})
	} else {
		cleanedPath := filepath.Join("/", path)
		providedPath := strings.TrimSuffix(path, "/") // Join trims suffix slashes

		if cleanedPath != providedPath {
			causes = append(causes, metav1.StatusCause{
				Type: metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s must be an absolute path to a file without relative components",
					field.String(),
				),
				Field: field.String(),
			})
		}
	}
	return causes

}

func validateCPURealtime(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if !spec.Domain.CPU.DedicatedCPUPlacement {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("%s must be set to true when %s is used",
				field.Child("domain", "cpu", "dedicatedCpuPlacement").String(),
				field.Child("domain", "cpu", "realtime").String(),
			),
			Field: field.Child("domain", "cpu", "dedicatedCpuPlacement").String(),
		})
	}
	return causes
}

func validateMemoryRealtime(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.CPU.NUMA == nil || spec.Domain.CPU.NUMA.GuestMappingPassthrough == nil {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("%s must be defined when %s is used",
				field.Child("domain", "cpu", "numa", "guestMappingPassthrough").String(),
				field.Child("domain", "cpu", "realtime").String(),
			),
			Field: field.Child("domain", "cpu", "numa", "guestMappingPassthrough").String(),
		})
	}
	return causes
}

func appendNewStatusCauseForHostNameNotConformingToDNSLabelRules(field *k8sfield.Path, causes []metav1.StatusCause, errors []string) []metav1.StatusCause {
	return append(causes, metav1.StatusCause{
		Type: metav1.CauseTypeFieldValueInvalid,
		Message: fmt.Sprintf("%s does not conform to the kubernetes DNS_LABEL rules : %s",
			field.Child("hostname").String(), strings.Join(errors, ", ")),
		Field: field.Child("hostname").String(),
	})
}

func appendNewStatusCauseForMaxNumberOfVolumesExceeded(field *k8sfield.Path, causes []metav1.StatusCause) []metav1.StatusCause {
	return append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: fmt.Sprintf(listExceedsLimitMessagePattern, field.Child("volumes").String(), arrayLenMax),
		Field:   field.Child("volumes").String(),
	})
}

func appendNewStatusCauseForNumberOfDisksExceeded(field *k8sfield.Path, causes []metav1.StatusCause) []metav1.StatusCause {
	return append(causes, metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: fmt.Sprintf(listExceedsLimitMessagePattern, field.Child("domain", "devices", "disks").String(), arrayLenMax),
		Field:   field.Child("domain", "devices", "disks").String(),
	})
}

// ValidateVirtualMachineInstanceMandatoryFields should be invoked after all defaults and presets are applied.
// It is only meant to be used for VMI reviews, not if they are templates on other objects
func ValidateVirtualMachineInstanceMandatoryFields(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	requests := spec.Domain.Resources.Requests.Memory().Value()

	if requests == 0 &&
		(spec.Domain.Memory == nil || spec.Domain.Memory != nil &&
			spec.Domain.Memory.Guest == nil && spec.Domain.Memory.Hugepages == nil) {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("no memory requested, at least one of '%s', '%s' or '%s' must be set",
				field.Child("domain", "memory", "guest").String(),
				field.Child("domain", "memory", "hugepages", "size").String(),
				field.Child("domain", "resources", "requests", "memory").String()),
		})
	}
	return causes
}

func ValidateVirtualMachineInstanceMetadata(field *k8sfield.Path, metadata *metav1.ObjectMeta, config *virtconfig.ClusterConfig, accountName string) []metav1.StatusCause {

	var causes []metav1.StatusCause
	annotations := metadata.Annotations
	labels := metadata.Labels
	// Validate kubevirt.io labels presence. Restricted labels allowed
	// to be created only by known service accounts
	if !webhooks.IsKubeVirtServiceAccount(accountName) {
		if len(filterKubevirtLabels(labels)) > 0 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: "creation of the following reserved kubevirt.io/ labels on a VMI object is prohibited",
				Field:   field.Child("labels").String(),
			})
		}
	}

	// Validate ignition feature gate if set when the corresponding annotation is found
	if annotations[v1.IgnitionAnnotation] != "" && !config.IgnitionEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("ExperimentalIgnitionSupport feature gate is not enabled in kubevirt-config, invalid entry %s",
				field.Child("annotations").Child(v1.IgnitionAnnotation).String()),
			Field: field.Child("annotations").String(),
		})
	}

	// Validate sidecar feature gate if set when the corresponding annotation is found
	if annotations[hooks.HookSidecarListAnnotationName] != "" && !config.SidecarEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("sidecar feature gate is not enabled in kubevirt-config, invalid entry %s",
				field.Child("annotations", hooks.HookSidecarListAnnotationName).String()),
			Field: field.Child("annotations").String(),
		})
	}

	if threadCountStr, exists := metadata.Annotations[cmdclient.MultiThreadedQemuMigrationAnnotation]; exists {
		threadCount, err := strconv.Atoi(threadCountStr)
		invalidEntry := field.Child("annotations", cmdclient.MultiThreadedQemuMigrationAnnotation).String()

		if err != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("cannot parse %s to int: %s, invalid entry %s", threadCountStr, err.Error(), invalidEntry),
				Field:   field.Child("annotations").String(),
			})
		} else if threadCount <= 1 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("thread count (%s) must be larger than 1. invalid entry %s", threadCountStr, invalidEntry),
				Field:   field.Child("annotations").String(),
			})
		}
	}

	return causes
}

func ValidateDuplicateDHCPPrivateOptions(PrivateOptions []v1.DHCPPrivateOptions) error {
	isUnique := map[int]bool{}
	for _, DHCPPrivateOption := range PrivateOptions {
		if isUnique[DHCPPrivateOption.Option] {
			return errors.New("you have provided duplicate DHCPPrivateOptions")
		}
		isUnique[DHCPPrivateOption.Option] = true
	}
	return nil
}

// Copied from kubernetes/pkg/apis/core/validation/validation.go
func validatePodDNSConfig(dnsConfig *k8sv1.PodDNSConfig, dnsPolicy *k8sv1.DNSPolicy, field *k8sfield.Path) []metav1.StatusCause {
	var causes []metav1.StatusCause

	// Validate DNSNone case. Must provide at least one DNS name server.
	if dnsPolicy != nil && *dnsPolicy == k8sv1.DNSNone {
		if dnsConfig == nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: fmt.Sprintf("must provide `dnsConfig` when `dnsPolicy` is %s", k8sv1.DNSNone),
				Field:   field.String(),
			})
			return causes
		}
		if len(dnsConfig.Nameservers) == 0 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: fmt.Sprintf("must provide at least one DNS nameserver when `dnsPolicy` is %s", k8sv1.DNSNone),
				Field:   "nameservers",
			})
			return causes
		}
	}

	if dnsConfig != nil {
		// Validate nameservers.
		if len(dnsConfig.Nameservers) > maxDNSNameservers {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("must not have more than %v nameservers: %s", maxDNSNameservers, dnsConfig.Nameservers),
				Field:   "nameservers",
			})
		}
		for _, ns := range dnsConfig.Nameservers {
			if ip := net.ParseIP(ns); ip == nil {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("must be valid IP address: %s", ns),
					Field:   "nameservers",
				})
			}
		}
		// Validate searches.
		if len(dnsConfig.Searches) > maxDNSSearchPaths {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("must not have more than %v search paths", maxDNSSearchPaths),
				Field:   "searchDomains",
			})
		}
		// Include the space between search paths.
		if len(strings.Join(dnsConfig.Searches, " ")) > maxDNSSearchListChars {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("must not have more than %v characters (including spaces) in the search list", maxDNSSearchListChars),
				Field:   "searchDomains",
			})
		}
		for _, search := range dnsConfig.Searches {
			for _, msg := range validation.IsDNS1123Subdomain(search) {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%v", msg),
					Field:   "searchDomains",
				})
			}
		}
		// Validate options.
		for _, option := range dnsConfig.Options {
			if len(option.Name) == 0 {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("Option.Name must not be empty"),
					Field:   "options",
				})
			}
		}
	}
	return causes
}

// Copied from kubernetes/pkg/apis/core/validation/validation.go
func validateDNSPolicy(dnsPolicy *k8sv1.DNSPolicy, field *k8sfield.Path) []metav1.StatusCause {
	var causes []metav1.StatusCause
	switch *dnsPolicy {
	case k8sv1.DNSClusterFirstWithHostNet, k8sv1.DNSClusterFirst, k8sv1.DNSDefault, k8sv1.DNSNone, "":
	default:
		validValues := []string{string(k8sv1.DNSClusterFirstWithHostNet), string(k8sv1.DNSClusterFirst), string(k8sv1.DNSDefault), string(k8sv1.DNSNone), ""}
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: fmt.Sprintf("DNSPolicy: %s is not supported, valid values: %s", *dnsPolicy, validValues),
			Field:   field.String(),
		})
	}
	return causes
}

func validateBootloader(field *k8sfield.Path, bootloader *v1.Bootloader) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if bootloader != nil && bootloader.EFI != nil && bootloader.BIOS != nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s has both EFI and BIOS configured, but they are mutually exclusive.", field.String()),
			Field:   field.String(),
		})
	}

	return causes
}

func validateFirmware(field *k8sfield.Path, firmware *v1.Firmware) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if firmware != nil {
		causes = append(causes, validateBootloader(field.Child("bootloader"), firmware.Bootloader)...)
		causes = append(causes, validateKernelBoot(field.Child("kernelBoot"), firmware.KernelBoot)...)
	}

	return causes
}

func validateDomainSpec(field *k8sfield.Path, spec *v1.DomainSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	causes = append(causes, validateDevices(field.Child("devices"), &spec.Devices)...)
	causes = append(causes, validateFirmware(field.Child("firmware"), spec.Firmware)...)

	if spec.Firmware != nil && spec.Firmware.Bootloader != nil && spec.Firmware.Bootloader.EFI != nil &&
		(spec.Firmware.Bootloader.EFI.SecureBoot == nil || *spec.Firmware.Bootloader.EFI.SecureBoot) &&
		(spec.Features == nil || spec.Features.SMM == nil || (spec.Features.SMM.Enabled != nil && !*spec.Features.SMM.Enabled)) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s has EFI SecureBoot enabled. SecureBoot requires SMM, which is currently disabled.", field.String()),
			Field:   field.String(),
		})
	}

	return causes
}

func validateAccessCredentials(field *k8sfield.Path, accessCredentials []v1.AccessCredential, volumes []v1.Volume) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if len(accessCredentials) > arrayLenMax {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf(listExceedsLimitMessagePattern, field.String(), arrayLenMax),
			Field:   field.String(),
		})
		// We won't process anything over the limit
		return causes
	}

	hasNoCloudVolume := false
	for _, volume := range volumes {
		if volume.CloudInitNoCloud != nil {
			hasNoCloudVolume = true
			break
		}
	}
	hasConfigDriveVolume := false
	for _, volume := range volumes {
		if volume.CloudInitConfigDrive != nil {
			hasConfigDriveVolume = true
			break
		}
	}

	for idx, accessCred := range accessCredentials {

		count := 0
		// one access cred type must be selected
		if accessCred.SSHPublicKey != nil {
			count++

			sourceCount := 0
			methodCount := 0
			if accessCred.SSHPublicKey.Source.Secret != nil {
				sourceCount++
			}

			if accessCred.SSHPublicKey.PropagationMethod.NoCloud != nil {
				methodCount++
				if !hasNoCloudVolume {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("%s requires a noCloud volume to exist when the noCloud propagationMethod is in use.", field.Index(idx).String()),
						Field:   field.Index(idx).Child("sshPublicKey", "propagationMethod").String(),
					})

				}
			}

			if accessCred.SSHPublicKey.PropagationMethod.ConfigDrive != nil {
				methodCount++
				if !hasConfigDriveVolume {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("%s requires a configDrive volume to exist when the configDrive propagationMethod is in use.", field.Index(idx).String()),
						Field:   field.Index(idx).Child("sshPublicKey", "propagationMethod").String(),
					})

				}
			}

			if accessCred.SSHPublicKey.PropagationMethod.QemuGuestAgent != nil {

				if len(accessCred.SSHPublicKey.PropagationMethod.QemuGuestAgent.Users) == 0 {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("%s requires at least one user to be present in the users list", field.Index(idx).String()),
						Field:   field.Index(idx).Child("sshPublicKey", "propagationMethod", "qemuGuestAgent", "users").String(),
					})
				}

				methodCount++
			}

			if sourceCount != 1 {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s must have exactly one source set", field.Index(idx).String()),
					Field:   field.Index(idx).Child("sshPublicKey", "source").String(),
				})
			}
			if methodCount != 1 {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s must have exactly one propagationMethod set", field.Index(idx).String()),
					Field:   field.Index(idx).Child("sshPublicKey", "propagationMethod").String(),
				})
			}
		}

		if accessCred.UserPassword != nil {
			count++

			sourceCount := 0
			methodCount := 0
			if accessCred.UserPassword.Source.Secret != nil {
				sourceCount++
			}

			if accessCred.UserPassword.PropagationMethod.QemuGuestAgent != nil {
				methodCount++
			}

			if sourceCount != 1 {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s must have exactly one source set", field.Index(idx).String()),
					Field:   field.Index(idx).Child("userPassword", "source").String(),
				})
			}
			if methodCount != 1 {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s must have exactly one propagationMethod set", field.Index(idx).String()),
					Field:   field.Index(idx).Child("userPassword", "propagationMethod").String(),
				})
			}
		}

		if count != 1 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s must have exactly one access credential type set", field.Index(idx).String()),
				Field:   field.Index(idx).String(),
			})
		}

	}

	return causes
}

func validateVolumes(field *k8sfield.Path, volumes []v1.Volume, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause
	nameMap := make(map[string]int)

	// check that we have max 1 instance of below disks
	serviceAccountVolumeCount := 0
	downwardMetricVolumeCount := 0
	memoryDumpVolumeCount := 0

	for idx, volume := range volumes {
		// verify name is unique
		otherIdx, ok := nameMap[volume.Name]
		if !ok {
			nameMap[volume.Name] = idx
		} else {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s and %s must not have the same Name.", field.Index(idx).String(), field.Index(otherIdx).String()),
				Field:   field.Index(idx).Child("name").String(),
			})
		}

		// Verify exactly one source is set
		volumeSourceSetCount := 0
		if volume.PersistentVolumeClaim != nil {
			volumeSourceSetCount++
		}
		if volume.Sysprep != nil {
			volumeSourceSetCount++
		}
		if volume.CloudInitNoCloud != nil {
			volumeSourceSetCount++
		}
		if volume.CloudInitConfigDrive != nil {
			volumeSourceSetCount++
		}
		if volume.ContainerDisk != nil {
			volumeSourceSetCount++
		}
		if volume.Ephemeral != nil {
			volumeSourceSetCount++
		}
		if volume.EmptyDisk != nil {
			volumeSourceSetCount++
		}
		if volume.HostDisk != nil {
			volumeSourceSetCount++
		}
		if volume.DataVolume != nil {
			if !config.HasDataVolumeAPI() {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "DataVolume api is not present in cluster. CDI must be installed for DataVolume support.",
					Field:   field.Index(idx).String(),
				})
			}

			if volume.DataVolume.Name == "" {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueRequired,
					Message: "DataVolume 'name' must be set",
					Field:   field.Index(idx).Child("name").String(),
				})
			}
			volumeSourceSetCount++
		}
		if volume.ConfigMap != nil {
			volumeSourceSetCount++
		}
		if volume.Secret != nil {
			volumeSourceSetCount++
		}
		if volume.DownwardAPI != nil {
			volumeSourceSetCount++
		}
		if volume.ServiceAccount != nil {
			volumeSourceSetCount++
			serviceAccountVolumeCount++
		}
		if volume.DownwardMetrics != nil {
			downwardMetricVolumeCount++
			volumeSourceSetCount++
		}
		if volume.MemoryDump != nil {
			memoryDumpVolumeCount++
			volumeSourceSetCount++
		}

		if volumeSourceSetCount != 1 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s must have exactly one source type set", field.Index(idx).String()),
				Field:   field.Index(idx).String(),
			})
		}

		// Verify cloud init data is within size limits
		if volume.CloudInitNoCloud != nil || volume.CloudInitConfigDrive != nil {
			var userDataSecretRef, networkDataSecretRef *k8sv1.LocalObjectReference
			var dataSourceType, userData, userDataBase64, networkData, networkDataBase64 string
			if volume.CloudInitNoCloud != nil {
				dataSourceType = "cloudInitNoCloud"
				userDataSecretRef = volume.CloudInitNoCloud.UserDataSecretRef
				userDataBase64 = volume.CloudInitNoCloud.UserDataBase64
				userData = volume.CloudInitNoCloud.UserData
				networkDataSecretRef = volume.CloudInitNoCloud.NetworkDataSecretRef
				networkDataBase64 = volume.CloudInitNoCloud.NetworkDataBase64
				networkData = volume.CloudInitNoCloud.NetworkData
			} else if volume.CloudInitConfigDrive != nil {
				dataSourceType = "cloudInitConfigDrive"
				userDataSecretRef = volume.CloudInitConfigDrive.UserDataSecretRef
				userDataBase64 = volume.CloudInitConfigDrive.UserDataBase64
				userData = volume.CloudInitConfigDrive.UserData
				networkDataSecretRef = volume.CloudInitConfigDrive.NetworkDataSecretRef
				networkDataBase64 = volume.CloudInitConfigDrive.NetworkDataBase64
				networkData = volume.CloudInitConfigDrive.NetworkData
			}

			userDataLen := 0
			userDataSourceCount := 0
			networkDataLen := 0
			networkDataSourceCount := 0

			if userDataSecretRef != nil && userDataSecretRef.Name != "" {
				userDataSourceCount++
			}
			if userDataBase64 != "" {
				userDataSourceCount++
				userData, err := base64.StdEncoding.DecodeString(userDataBase64)
				if err != nil {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("%s.%s.userDataBase64 is not a valid base64 value.", field.Index(idx).Child(dataSourceType, "userDataBase64").String(), dataSourceType),
						Field:   field.Index(idx).Child(dataSourceType, "userDataBase64").String(),
					})
				}
				userDataLen = len(userData)
			}
			if userData != "" {
				userDataSourceCount++
				userDataLen = len(userData)
			}

			if userDataSourceCount > 1 {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s must have only one userdatasource set.", field.Index(idx).Child(dataSourceType).String()),
					Field:   field.Index(idx).Child(dataSourceType).String(),
				})
			}

			if userDataLen > cloudInitUserMaxLen {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s userdata exceeds %d byte limit. Should use UserDataSecretRef for larger data.", field.Index(idx).Child(dataSourceType).String(), cloudInitUserMaxLen),
					Field:   field.Index(idx).Child(dataSourceType).String(),
				})
			}

			if networkDataSecretRef != nil && networkDataSecretRef.Name != "" {
				networkDataSourceCount++
			}
			if networkDataBase64 != "" {
				networkDataSourceCount++
				networkData, err := base64.StdEncoding.DecodeString(networkDataBase64)
				if err != nil {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("%s.%s.networkDataBase64 is not a valid base64 value.", field.Index(idx).Child(dataSourceType, "networkDataBase64").String(), dataSourceType),
						Field:   field.Index(idx).Child(dataSourceType, "networkDataBase64").String(),
					})
				}
				networkDataLen = len(networkData)
			}
			if networkData != "" {
				networkDataSourceCount++
				networkDataLen = len(networkData)
			}

			if networkDataSourceCount > 1 {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s must have only one networkdata source set.", field.Index(idx).Child(dataSourceType).String()),
					Field:   field.Index(idx).Child(dataSourceType).String(),
				})
			}

			if networkDataLen > cloudInitNetworkMaxLen {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s networkdata exceeds %d byte limit. Should use NetworkDataSecretRef for larger data.", field.Index(idx).Child(dataSourceType).String(), cloudInitNetworkMaxLen),
					Field:   field.Index(idx).Child(dataSourceType).String(),
				})
			}

			if userDataSourceCount == 0 && networkDataSourceCount == 0 {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s must have at least one userdatasource or one networkdatasource set.", field.Index(idx).Child(dataSourceType).String()),
					Field:   field.Index(idx).Child(dataSourceType).String(),
				})
			}
		}

		if volume.DownwardMetrics != nil && !config.DownwardMetricsEnabled() {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "downwardMetrics disks are not allowed: DownwardMetrics feature gate is not enabled.",
				Field:   field.Index(idx).String(),
			})
		}

		// validate HostDisk data
		if hostDisk := volume.HostDisk; hostDisk != nil {
			if !config.HostDiskEnabled() {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "HostDisk feature gate is not enabled",
					Field:   field.Index(idx).String(),
				})
			}
			if hostDisk.Path == "" {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueNotFound,
					Message: fmt.Sprintf("%s is required for hostDisk volume", field.Index(idx).Child("hostDisk", "path").String()),
					Field:   field.Index(idx).Child("hostDisk", "path").String(),
				})
			}

			if hostDisk.Type != v1.HostDiskExists && hostDisk.Type != v1.HostDiskExistsOrCreate {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s has invalid value '%s', allowed are '%s' or '%s'", field.Index(idx).Child("hostDisk", "type").String(), hostDisk.Type, v1.HostDiskExists, v1.HostDiskExistsOrCreate),
					Field:   field.Index(idx).Child("hostDisk", "type").String(),
				})
			}

			// if disk.img already exists and user knows that by specifying type 'Disk' it is pointless to set capacity
			if hostDisk.Type == v1.HostDiskExists && !hostDisk.Capacity.IsZero() {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s is allowed to pass only with %s equal to '%s'", field.Index(idx).Child("hostDisk", "capacity").String(), field.Index(idx).Child("hostDisk", "type").String(), v1.HostDiskExistsOrCreate),
					Field:   field.Index(idx).Child("hostDisk", "capacity").String(),
				})
			}
		}

		if volume.ConfigMap != nil {
			if volume.ConfigMap.LocalObjectReference.Name == "" {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf(requiredFieldFmt, field.Index(idx).Child("configMap", "name").String()),
					Field:   field.Index(idx).Child("configMap", "name").String(),
				})
			}
		}

		if volume.Secret != nil {
			if volume.Secret.SecretName == "" {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf(requiredFieldFmt, field.Index(idx).Child("secret", "secretName").String()),
					Field:   field.Index(idx).Child("secret", "secretName").String(),
				})
			}
		}

		if volume.ServiceAccount != nil {
			if volume.ServiceAccount.ServiceAccountName == "" {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf(requiredFieldFmt, field.Index(idx).Child("serviceAccount", "serviceAccountName").String()),
					Field:   field.Index(idx).Child("serviceAccount", "serviceAccountName").String(),
				})
			}
		}
	}

	if serviceAccountVolumeCount > 1 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s must have max one serviceAccount volume set", field.String()),
			Field:   field.String(),
		})
	}

	if downwardMetricVolumeCount > 1 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s must have max one downwardMetric volume set", field.String()),
			Field:   field.String(),
		})
	}
	if memoryDumpVolumeCount > 1 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s must have max one memory dump volume set", field.String()),
			Field:   field.String(),
		})
	}

	return causes
}

func validateDevices(field *k8sfield.Path, devices *v1.Devices) []metav1.StatusCause {
	var causes []metav1.StatusCause
	causes = append(causes, validateDisks(field.Child("disks"), devices.Disks)...)
	return causes
}

func getNumberOfPodInterfaces(spec *v1.VirtualMachineInstanceSpec) int {
	nPodInterfaces := 0
	for _, net := range spec.Networks {
		if net.Pod != nil {
			for _, iface := range spec.Domain.Devices.Interfaces {
				if iface.Name == net.Name {
					nPodInterfaces++
					break // we maintain 1-to-1 relationship between networks and interfaces
				}
			}
		}
	}
	return nPodInterfaces
}

func validateDisks(field *k8sfield.Path, disks []v1.Disk) []metav1.StatusCause {
	var causes []metav1.StatusCause
	nameMap := make(map[string]int)

	for idx, disk := range disks {
		// verify name is unique
		otherIdx, ok := nameMap[disk.Name]
		if !ok {
			nameMap[disk.Name] = idx
		} else {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s and %s must not have the same Name.", field.Index(idx).String(), field.Index(otherIdx).String()),
				Field:   field.Index(idx).Child("name").String(),
			})
		}

		// Verify only a single device type is set.
		deviceTargetSetCount := 0
		var diskType string
		var bus v1.DiskBus
		if disk.Disk != nil {
			deviceTargetSetCount++
			diskType = "disk"
			bus = disk.Disk.Bus
		}
		if disk.LUN != nil {
			deviceTargetSetCount++
			diskType = "lun"
			bus = disk.LUN.Bus
		}
		if disk.CDRom != nil {
			deviceTargetSetCount++
			diskType = "cdrom"
			bus = disk.CDRom.Bus
		}

		// NOTE: not setting a device target is okay. We default to Disk.
		// However, only a single device target is allowed to be set at a time.
		if deviceTargetSetCount > 1 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s can only have a single target type defined", field.Index(idx).String()),
				Field:   field.Index(idx).String(),
			})
		}

		// Verify pci address
		if disk.Disk != nil && disk.Disk.PciAddress != "" {
			if disk.Disk.Bus != v1.DiskBusVirtio {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("disk %s - setting a PCI address is only possible with bus type virtio.", field.Child("domain", "devices", "disks", "disk").Index(idx).Child("name").String()),
					Field:   field.Child("domain", "devices", "disks", "disk").Index(idx).Child("pciAddress").String(),
				})
			}

			_, err := hwutil.ParsePciAddress(disk.Disk.PciAddress)
			if err != nil {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("disk %s has malformed PCI address (%s).", field.Child("domain", "devices", "disks", "disk").Index(idx).Child("name").String(), disk.Disk.PciAddress),
					Field:   field.Child("domain", "devices", "disks", "disk").Index(idx).Child("pciAddress").String(),
				})
			}
		}

		// Verify boot order is greater than 0, if provided
		if disk.BootOrder != nil && *disk.BootOrder < 1 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s must have a boot order > 0, if supplied", field.Index(idx).String()),
				Field:   field.Index(idx).Child("bootOrder").String(),
			})
		}

		// Verify bus is supported, if provided
		if len(bus) > 0 {
			if bus == "ide" {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "IDE bus is not supported",
					Field:   field.Index(idx).Child(diskType, "bus").String(),
				})
			} else {
				buses := []v1.DiskBus{v1.DiskBusVirtio, v1.DiskBusSCSI, v1.DiskBusSATA, v1.DiskBusUSB}
				validBus := false
				for _, b := range buses {
					if b == bus {
						validBus = true
					}
				}
				if !validBus {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("%s is set with an unrecognized bus %s, must be one of: %v", field.Index(idx).String(), bus, buses),
						Field:   field.Index(idx).Child(diskType, "bus").String(),
					})
				}

				// special case. virtio is incompatible with CD-ROM for q35 machine types
				if diskType == "cdrom" && bus == v1.DiskBusVirtio {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("Bus type %s is invalid for CD-ROM device", bus),
						Field:   field.Index(idx).Child("cdrom", "bus").String(),
					})

				}
				// sata disks (in contrast to sata cdroms) don't support readOnly
				if disk.Disk != nil && bus == v1.DiskBusSATA && disk.Disk.ReadOnly {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("%s hard-disks do not support read-only.", bus),
						Field:   field.Index(idx).Child("disk", "bus").String(),
					})
				}
			}

			// Reject defining DedicatedIOThread to a disk with SATA bus since this configuration
			// is not supported in libvirt.
			isIOThreadsWithSataBus := disk.DedicatedIOThread != nil && *disk.DedicatedIOThread &&
				(disk.DiskDevice.Disk != nil) && (disk.DiskDevice.Disk.Bus == v1.DiskBusSATA)
			if isIOThreadsWithSataBus {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueNotSupported,
					Message: "IOThreads are not supported for disks on a SATA bus",
					Field:   field.Child("domain", "devices", "disks").Index(idx).String(),
				})
			}
		}

		// Verify serial number is made up of valid characters for libvirt, if provided
		isValid := regexp.MustCompile(`^[A-Za-z0-9_.+-]+$`).MatchString
		if disk.Serial != "" && !isValid(disk.Serial) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s must be made up of the following characters [A-Za-z0-9_.+-], if specified", field.Index(idx).String()),
				Field:   field.Index(idx).Child("serial").String(),
			})
		}

		// Verify serial number is within valid length, if provided
		if disk.Serial != "" && len([]rune(disk.Serial)) > maxStrLen {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s must be less than or equal to %d in length, if specified", field.Index(idx).String(), maxStrLen),
				Field:   field.Index(idx).Child("serial").String(),
			})
		}

		// Verify if cache mode is valid
		if disk.Cache != "" && disk.Cache != v1.CacheNone && disk.Cache != v1.CacheWriteThrough && disk.Cache != v1.CacheWriteBack {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s has invalid value %s", field.Index(idx).Child("cache").String(), disk.Cache),
				Field:   field.Index(idx).Child("cache").String(),
			})
		}

		if disk.IO != "" && disk.IO != v1.IONative && disk.IO != v1.IOThreads {
			field := field.Child("domain", "devices", "disks").Index(idx).Child("io").String()
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: fmt.Sprintf("Disk IO mode for %s is not supported. Supported modes are: native, threads.", field),
				Field:   field,
			})
		}

		// Verify if error_policy is valid
		if disk.ErrorPolicy != nil && *disk.ErrorPolicy != v1.DiskErrorPolicyStop && *disk.ErrorPolicy != v1.DiskErrorPolicyIgnore && *disk.ErrorPolicy != v1.DiskErrorPolicyReport && *disk.ErrorPolicy != v1.DiskErrorPolicyEnospace {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s has invalid value \"%s\"", field.Index(idx).Child("errorPolicy").String(), *disk.ErrorPolicy),
				Field:   field.Index(idx).Child("errorPolicy").String(),
			})
		}

		// Verify disk and volume name can be a valid container name since disk
		// name can become a container name which will fail to schedule if invalid
		errs := validation.IsDNS1123Label(disk.Name)

		for _, err := range errs {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: err,
				Field:   field.Child("domain", "devices", "disks").Index(idx).Child("name").String(),
			})
		}

		if disk.BlockSize != nil {
			hasCustomBlockSize := disk.BlockSize.Custom != nil
			hasVolumeMatchingEnabled := disk.BlockSize.MatchVolume != nil && (disk.BlockSize.MatchVolume.Enabled == nil || *disk.BlockSize.MatchVolume.Enabled)
			if hasCustomBlockSize && hasVolumeMatchingEnabled {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "Block size matching can't be enabled together with a custom value",
					Field:   field.Index(idx).Child("blockSize").String(),
				})
			} else if hasCustomBlockSize {
				customSize := disk.BlockSize.Custom
				if customSize.Logical > customSize.Physical {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("Logical size %d must be the same or less than the physical size of %d", customSize.Logical, customSize.Physical),
						Field:   field.Index(idx).Child("blockSize").Child("custom").Child("logical").String(),
					})
				} else {
					checkSize := func(size uint) (bool, string) {
						if size < 512 {
							return false, fmt.Sprintf("Provided size of %d is less than the supported minimum size of 512", size)
						} else if size > 2097152 {
							return false, fmt.Sprintf("Provided size of %d is greater than the supported maximum size of 2 MiB", size)
						} else if size&(size-1) != 0 {
							return false, fmt.Sprintf("Provided size of %d is not a power of 2", size)
						}
						return true, ""
					}
					if sizeOk, reason := checkSize(customSize.Logical); !sizeOk {
						causes = append(causes, metav1.StatusCause{
							Type:    metav1.CauseTypeFieldValueInvalid,
							Message: reason,
							Field:   field.Index(idx).Child("blockSize").Child("custom").Child("logical").String(),
						})
					}
					if sizeOk, reason := checkSize(customSize.Physical); !sizeOk {
						causes = append(causes, metav1.StatusCause{
							Type:    metav1.CauseTypeFieldValueInvalid,
							Message: reason,
							Field:   field.Index(idx).Child("blockSize").Child("custom").Child("physical").String(),
						})
					}
				}
			}
		}
	}

	return causes
}

// Rejects kernel boot defined with initrd/kernel path but without an image
func validateKernelBoot(field *k8sfield.Path, kernelBoot *v1.KernelBoot) (causes []metav1.StatusCause) {
	if kernelBoot == nil {
		return
	}

	if kernelBoot.Container == nil {
		if kernelBoot.KernelArgs != "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "kernel arguments cannot be provided without an external kernel",
				Field:   field.Child("kernelArgs").String(),
			})
		}
		return
	}

	container := kernelBoot.Container
	containerField := field.Child("container")

	if container.Image == "" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("%s must be defined with an image", containerField),
			Field:   containerField.Child("image").String(),
		})
	}

	if container.InitrdPath == "" && container.KernelPath == "" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("%s must be defined with at least one of the following: kernelPath, initrdPath", containerField),
			Field:   containerField.String(),
		})
	}

	if container.KernelPath != "" {
		causes = append(causes, validatePath(containerField.Child("kernelPath"), container.KernelPath)...)
	}
	if container.InitrdPath != "" {
		causes = append(causes, validatePath(containerField.Child("initrdPath"), container.InitrdPath)...)
	}

	return
}

// validateSpecAffinity is function that validate spec.affinity
// instead of bring in the whole kubernetes lib we simply copy it from kubernetes/pkg/apis/core/validation/validation.go
func validateSpecAffinity(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Affinity == nil {
		return
	}

	errorList := validateAffinity(spec.Affinity, field)

	//convert errorList to []metav1.StatusCause
	for _, validationErr := range errorList {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: validationErr.Error(),
			Field:   validationErr.Field,
		})
	}

	return
}

// validateSpecTopologySpreadConstraints is function that validate spec.validateSpecTopologySpreadConstraints
// instead of bring in the whole kubernetes lib we simply copy it from kubernetes/pkg/apis/core/validation/validation.go
func validateSpecTopologySpreadConstraints(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.TopologySpreadConstraints == nil {
		return
	}

	errorList := validateTopologySpreadConstraints(spec.TopologySpreadConstraints, field.Child("topologySpreadConstraints"))

	//convert errorList to []metav1.StatusCause
	for _, validationErr := range errorList {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: validationErr.Error(),
			Field:   validationErr.Field,
		})
	}

	return
}

func validateVSOCK(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) (causes []metav1.StatusCause) {
	if spec.Domain.Devices.AutoattachVSOCK == nil || !*spec.Domain.Devices.AutoattachVSOCK {
		return
	}

	if !config.VSOCKEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s feature gate is not enabled in kubevirt-config", virtconfig.VSOCKGate),
			Field:   field.Child("domain", "devices", "autoattachVSOCK").String(),
		})
	}

	return
}

func validatePersistentReservation(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) (causes []metav1.StatusCause) {
	if !reservation.HasVMISpecPersistentReservation(spec) {
		return
	}

	if !config.PersistentReservationEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s feature gate is not enabled in kubevirt-config", virtconfig.PersistentReservation),
			Field:   field.Child("domain", "devices", "disks", "luns", "reservation").String(),
		})
	}

	return
}

func validatePersistentState(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) (causes []metav1.StatusCause) {
	if !backendstorage.IsBackendStorageNeededForVMI(spec) {
		return
	}

	if !config.VMPersistentStateEnabled() {
		if backendstorage.HasPersistentTPMDevice(spec) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s feature gate is not enabled in kubevirt-config", virtconfig.VMPersistentState),
				Field:   field.Child("domain", "devices", "tpm", "persistent").String(),
			})
		}
		if backendstorage.HasPersistentEFI(spec) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s feature gate is not enabled in kubevirt-config", virtconfig.VMPersistentState),
				Field:   field.Child("domain", "firmware", "bootloader", "efi", "persistent").String(),
			})
		}
	}

	return
}

func validateCPUHotplug(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (causes []metav1.StatusCause) {
	if spec.Domain.CPU != nil && spec.Domain.CPU.MaxSockets != 0 {
		if spec.Domain.CPU.Sockets > spec.Domain.CPU.MaxSockets {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Number of sockets in CPU topology is greater than the maximum sockets allowed"),
				Field:   field.Child("domain", "cpu", "sockets").String(),
			})
		}
	}
	return causes
}
