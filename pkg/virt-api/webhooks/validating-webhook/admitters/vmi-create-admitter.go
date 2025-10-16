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
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"kubevirt.io/kubevirt/pkg/storage/utils"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/downwardmetrics"
	draadmitter "kubevirt.io/kubevirt/pkg/dra/admitter"
	"kubevirt.io/kubevirt/pkg/hooks"
	netadmitter "kubevirt.io/kubevirt/pkg/network/admitter"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/storage/reservation"
	"kubevirt.io/kubevirt/pkg/storage/types"
	hwutil "kubevirt.io/kubevirt/pkg/util/hardware"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

const requiredFieldFmt = "%s is a required field"

const (
	maxStrLen = 256

	// Should be a power of 2
	minCustomBlockSize = 512
	maxCustomBlockSize = 2097152 // 2 MB

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

var validIOThreadsPolicies = []v1.IOThreadsPolicy{v1.IOThreadsPolicyShared, v1.IOThreadsPolicyAuto, v1.IOThreadsPolicySupplementalPool}
var validCPUFeaturePolicies = map[string]*struct{}{"": nil, "force": nil, "require": nil, "optional": nil, "disable": nil, "forbid": nil}
var validPanicDeviceModels = []v1.PanicDeviceModel{v1.Hyperv, v1.Isa, v1.Pvpanic}

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
	valueMustBePositiveMessagePattern = "%s '%s': must be greater than or equal to 0."
)

var isValidExpression = regexp.MustCompile(`^[A-Za-z0-9_.+-]+$`).MatchString

var invalidPanicDeviceModelErrFmt = "invalid PanicDeviceModel(%s)"

// SpecValidator validates the given VMI spec
type SpecValidator func(*k8sfield.Path, *v1.VirtualMachineInstanceSpec, *virtconfig.ClusterConfig) []metav1.StatusCause

type VMICreateAdmitter struct {
	ClusterConfig           *virtconfig.ClusterConfig
	SpecValidators          []SpecValidator
	KubeVirtServiceAccounts map[string]struct{}
}

func (admitter *VMICreateAdmitter) Admit(_ context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if resp := webhookutils.ValidateSchema(v1.VirtualMachineInstanceGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	vmi, _, err := webhookutils.GetVMIFromAdmissionReview(ar)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	var causes []metav1.StatusCause
	clusterCfg := admitter.ClusterConfig.GetConfig()
	if devCfg := clusterCfg.DeveloperConfiguration; devCfg != nil {
		causes = append(causes, featuregate.ValidateFeatureGates(devCfg.FeatureGates, &vmi.Spec)...)
	}

	for _, validateSpec := range admitter.SpecValidators {
		causes = append(causes, validateSpec(k8sfield.NewPath("spec"), &vmi.Spec, admitter.ClusterConfig)...)
	}

	causes = append(causes, ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("spec"), &vmi.Spec, admitter.ClusterConfig)...)
	// We only want to validate that volumes are mapped to disks or filesystems during VMI admittance, thus this logic is seperated from the above call that is shared with the VM admitter.
	causes = append(causes, validateVirtualMachineInstanceSpecVolumeDisks(k8sfield.NewPath("spec"), &vmi.Spec)...)
	causes = append(causes, ValidateVirtualMachineInstanceMandatoryFields(k8sfield.NewPath("spec"), &vmi.Spec)...)

	_, isKubeVirtServiceAccount := admitter.KubeVirtServiceAccounts[ar.Request.UserInfo.Username]
	causes = append(causes, ValidateVirtualMachineInstanceMetadata(k8sfield.NewPath("metadata"), &vmi.ObjectMeta, admitter.ClusterConfig, isKubeVirtServiceAccount)...)
	causes = append(causes, webhooks.ValidateVirtualMachineInstanceHyperv(k8sfield.NewPath("spec").Child("domain").Child("features").Child("hyperv"), &vmi.Spec)...)
	causes = append(causes, ValidateVirtualMachineInstancePerArch(k8sfield.NewPath("spec"), &vmi.Spec)...)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	return &admissionv1.AdmissionResponse{
		Allowed:  true,
		Warnings: warnDeprecatedAPIs(&vmi.Spec, admitter.ClusterConfig),
	}
}

func warnDeprecatedAPIs(spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []string {
	var warnings []string
	for _, fg := range config.GetConfig().DeveloperConfiguration.FeatureGates {
		deprecatedFeature := featuregate.FeatureGateInfo(fg)
		if deprecatedFeature != nil && deprecatedFeature.State == featuregate.Deprecated && deprecatedFeature.VmiSpecUsed != nil {
			if used := deprecatedFeature.VmiSpecUsed(spec); used {
				warnings = append(warnings, deprecatedFeature.Message)
			}
		}
	}
	return warnings
}

func ValidateVirtualMachineInstancePerArch(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	arch := spec.Architecture

	switch arch {
	case "amd64":
		causes = append(causes, webhooks.ValidateVirtualMachineInstanceAmd64Setting(field, spec)...)
	case "s390x":
		causes = append(causes, webhooks.ValidateVirtualMachineInstanceS390XSetting(field, spec)...)
	case "arm64":
		causes = append(causes, webhooks.ValidateVirtualMachineInstanceArm64Setting(field, spec)...)
	case "ppc64le":
		causes = append(causes, webhooks.ValidateVirtualMachineInstancePPC64LESetting(field, spec)...)
	default:
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("unsupported architecture: %s", arch),
			Field:   field.Child("architecture").String(),
		})
	}

	return causes
}

func ValidateVirtualMachineInstanceSpec(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause

	causes = append(causes, validateHostNameNotConformingToDNSLabelRules(field, spec)...)
	causes = append(causes, validateSubdomainDNSSubdomainRules(field, spec)...)
	causes = append(causes, validateMemoryRequestsNegativeOrNull(field, spec)...)
	causes = append(causes, validateMemoryLimitsNegativeOrNull(field, spec)...)
	causes = append(causes, validateHugepagesMemoryRequests(field, spec)...)
	causes = append(causes, validateGuestMemoryLimit(field, spec, config)...)
	causes = append(causes, validateEmulatedMachine(field, spec, config)...)
	causes = append(causes, validateFirmwareACPI(field.Child("acpi"), spec)...)
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

	netValidator := netadmitter.NewValidator(field, spec, config)
	causes = append(causes, netValidator.Validate()...)

	causes = append(causes, draadmitter.ValidateCreation(field, spec, config)...)

	causes = append(causes, validateBootOrder(field, spec, config)...)

	causes = append(causes, validateInputDevices(field, spec)...)
	causes = append(causes, validateIOThreadsPolicy(field, spec)...)
	causes = append(causes, validateProbe(field.Child("readinessProbe"), spec.ReadinessProbe)...)
	causes = append(causes, validateProbe(field.Child("livenessProbe"), spec.LivenessProbe)...)

	if podNetwork := vmispec.LookupPodNetwork(spec.Networks); podNetwork == nil {
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
	causes = append(causes, validateMDEVRamFB(field, spec)...)
	causes = append(causes, validateHostDevicesWithPassthroughEnabled(field, spec, config)...)
	causes = append(causes, validateSoundDevices(field, spec)...)
	causes = append(causes, validateLaunchSecurity(field, spec, config)...)
	causes = append(causes, validateVSOCK(field, spec, config)...)
	causes = append(causes, validatePersistentReservation(field, spec, config)...)
	causes = append(causes, validateDownwardMetrics(field, spec, config)...)
	causes = append(causes, validateFilesystemsWithVirtIOFSEnabled(field, spec, config)...)
	causes = append(causes, validateVideoConfig(field, spec, config)...)
	causes = append(causes, validatePanicDevices(field, spec, config)...)

	return causes
}

func validateFilesystemsWithVirtIOFSEnabled(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) (causes []metav1.StatusCause) {
	if spec.Domain.Devices.Filesystems == nil {
		return causes
	}

	volumes := types.GetVolumesByName(spec)

	for _, fs := range spec.Domain.Devices.Filesystems {
		volume, ok := volumes[fs.Name]
		if !ok {
			continue
		}

		switch {
		case utils.IsConfigVolume(volume) && (!config.VirtiofsConfigVolumesEnabled() && !config.OldVirtiofsEnabled()):
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "virtiofs is not allowed: virtiofs feature gate is not enabled for config volumes",
				Field:   field.Child("domain", "devices", "filesystems").String(),
			})
		case utils.IsStorageVolume(volume) && (!config.VirtiofsStorageEnabled() && !config.OldVirtiofsEnabled()):
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "virtiofs is not allowed: virtiofs feature gate is not enabled for PVC",
				Field:   field.Child("domain", "devices", "filesystems").String(),
			})
		}
	}

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

func validateInterfaceBootOrder(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, bootOrderMap map[uint]bool) (causes []metav1.StatusCause) {
	for idx, iface := range spec.Domain.Devices.Interfaces {
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

func validateIOThreadsPolicy(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if spec.Domain.IOThreadsPolicy == nil {
		return causes
	}
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

	if *spec.Domain.IOThreadsPolicy == v1.IOThreadsPolicySupplementalPool &&
		(spec.Domain.IOThreads == nil || spec.Domain.IOThreads.SupplementalPoolThreadCount == nil ||
			*spec.Domain.IOThreads.SupplementalPoolThreadCount < 1) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "the number of iothreads needs to be set and positive for the dedicated policy",
			Field:   field.Child("domain", "ioThreads", "count").String(),
		})
	}

	return causes
}

func validateProbe(field *k8sfield.Path, probe *v1.Probe) []metav1.StatusCause {
	var causes []metav1.StatusCause
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

func validateLiveMigration(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause
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

func countConfiguredMDEVRamFBs(spec *v1.VirtualMachineInstanceSpec) int {
	count := 0
	for _, device := range spec.Domain.Devices.GPUs {
		if device.VirtualGPUOptions != nil &&
			device.VirtualGPUOptions.Display != nil &&
			(device.VirtualGPUOptions.Display.Enabled == nil || *device.VirtualGPUOptions.Display.Enabled) &&
			(device.VirtualGPUOptions.Display.RamFB == nil || (device.VirtualGPUOptions.Display.RamFB.Enabled != nil && *device.VirtualGPUOptions.Display.RamFB.Enabled)) {
			count++
		}
	}
	return count
}

func validateMDEVRamFB(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if countConfiguredMDEVRamFBs(spec) > 1 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "configuring multiple displays with ramfb is not valid ",
			Field:   field.Child("GPUs").String(),
		})

	}
	return causes
}

func validateHostDevicesWithPassthroughEnabled(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if spec.Domain.Devices.HostDevices != nil && !config.HostDevicesPassthroughEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Host Devices feature gate is not enabled in kubevirt-config",
			Field:   field.Child("HostDevices").String(),
		})
	}
	return causes
}

func validateSoundDevices(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if spec.Domain.Devices.Sound == nil {
		return causes
	}
	model := spec.Domain.Devices.Sound.Model
	if model != "" && model != "ich9" && model != "ac97" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Sound device type is not supported. Options: 'ich9' or 'ac97'",
			Field:   field.Child("Sound").String(),
		})
	}
	if spec.Domain.Devices.Sound.Name == "" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Sound device requires a name field.",
			Field:   field.Child("Sound").String(),
		})
	}

	return causes
}

func validateLaunchSecurity(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause
	launchSecurity := spec.Domain.LaunchSecurity
	if launchSecurity == nil {
		return causes
	}
	if !config.SecureExecutionEnabled() && webhooks.IsS390X(spec) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s feature gate is not enabled in kubevirt-config", featuregate.SecureExecution),
			Field:   field.Child("launchSecurity").String(),
		})
	}
	if !config.WorkloadEncryptionSEVEnabled() && launchSecurity.SEV != nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s feature gate is not enabled in kubevirt-config", featuregate.WorkloadEncryptionSEV),
			Field:   field.Child("launchSecurity").String(),
		})
	} else if launchSecurity.SEV != nil {
		firmware := spec.Domain.Firmware
		if !efiBootEnabled(firmware) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "SEV requires OVMF (UEFI)",
				Field:   field.Child("launchSecurity").String(),
			})
		} else if secureBootEnabled(firmware) {
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

func validateBootOrder(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause
	// used to validate uniqueness of boot orders among disks and interfaces
	bootOrderMap := make(map[uint]bool)
	volumeNameMap := make(map[string]*v1.Volume)

	for i, volume := range spec.Volumes {
		volumeNameMap[volume.Name] = &spec.Volumes[i]
	}

	// Validate disks match volumes correctly
	for idx, disk := range spec.Domain.Devices.Disks {
		var matchingVolume *v1.Volume

		matchingVolume, volumeExists := volumeNameMap[disk.Name]

		if !volumeExists {
			if disk.CDRom == nil {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf(nameOfTypeNotFoundMessagePattern, field.Child("domain", "devices", "disks").Index(idx).Child("Name").String(), disk.Name),
					Field:   field.Child("domain", "devices", "disks").Index(idx).Child("name").String(),
				})
			} else if !config.DeclarativeHotplugVolumesEnabled() {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s feature gate not enabled, cannot define an empty CD-ROM disk", featuregate.DeclarativeHotplugVolumesGate),
					Field:   field.Child("domain", "devices", "disks").Index(idx).Child("name").String(),
				})
			}
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

	causes = append(causes, validateInterfaceBootOrder(field, spec, bootOrderMap)...)

	return causes
}

func validateCPUFeaturePolicies(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
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

func validateCPUIsolatorThread(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if spec.Domain.CPU != nil && spec.Domain.CPU.IsolateEmulatorThread && !spec.Domain.CPU.DedicatedCPUPlacement {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "IsolateEmulatorThread should be only set in combination with DedicatedCPUPlacement",
			Field:   field.Child("domain", "cpu", "isolateEmulatorThread").String(),
		})
	}
	return causes
}

func validateCpuPinning(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause
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

func validateNUMA(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause
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

func validateThreadCountOnArchitecture(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause
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

func validateThreadCountOnDedicatedCPUPlacement(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
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

func validateRequestOrLimitWithCoresProvidedOnDedicatedCPUPlacement(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
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

func validateRequestEqualsLimitOnDedicatedCPUPlacement(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
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

func validateRequestLimitOrCoresProvidedOnDedicatedCPUPlacement(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
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

func validateStartStrategy(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if spec.StartStrategy == nil {
		return causes
	}
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

	return causes
}

func validateMemoryRequestsAndLimits(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
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

func validateCPULimitIsInteger(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if spec.Domain.Resources.Limits.Cpu().Value() > 0 && spec.Domain.Resources.Limits.Cpu().Value()*1000 != spec.Domain.Resources.Limits.Cpu().MilliValue() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "provided resources CPU limits must be an interger",
			Field:   field.Child("domain", "resources", "limits", "cpu").String(),
		})
	}
	return causes
}

func validateCPURequestIsInteger(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if spec.Domain.Resources.Requests.Cpu().Value() > 0 && spec.Domain.Resources.Requests.Cpu().Value()*1000 != spec.Domain.Resources.Requests.Cpu().MilliValue() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "provided resources CPU requests must be an interger",
			Field:   field.Child("domain", "resources", "requests", "cpu").String(),
		})
	}
	return causes
}

func validateMemoryLimitAndRequestProvided(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if spec.Domain.Resources.Limits.Memory().Value() == 0 && spec.Domain.Resources.Requests.Memory().Value() == 0 &&
		spec.Domain.Memory.Hugepages == nil && spec.Domain.Memory.Guest.Value() == 0 {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s, %s, %s or %s should be provided",
				field.Child("domain", "resources", "requests", "memory").String(),
				field.Child("domain", "resources", "limits", "memory").String(),
				field.Child("domain", "memory", "hugepages").String(),
				field.Child("domain", "memory", "guest").String(),
			),
			Field: field.Child("domain", "resources", "limits", "memory").String(),
		})
	}
	return causes
}

func validateCpuRequestDoesNotExceedLimit(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
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

func validateCPULimitNotNegative(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
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

func validateCPURequestNotNegative(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
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

func validateEmulatedMachine(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if machine := spec.Domain.Machine; machine != nil && len(machine.Type) > 0 {
		supportedMachines := config.GetEmulatedMachines(spec.Architecture)
		var match = false
		for _, val := range supportedMachines {
			// The pattern are hardcoded, so this should not throw an error
			if ok, _ := filepath.Match(val, machine.Type); ok {
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

func validateGuestMemoryLimit(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if config.IsVMRolloutStrategyLiveUpdate() {
		return causes
	}
	if spec.Domain.Memory == nil || spec.Domain.Memory.Guest == nil {
		return causes
	}
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
	return causes
}

func validateHugepagesMemoryRequests(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if spec.Domain.Memory == nil || spec.Domain.Memory.Hugepages == nil {
		return causes
	}
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
		return causes
	}
	vmMemory := spec.Domain.Resources.Requests.Memory().Value()
	if vmMemory == 0 && spec.Domain.Memory != nil {
		vmMemory = spec.Domain.Memory.Guest.Value()
	}
	if vmMemory != 0 && vmMemory < hugepagesSize.Value() {
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

	return causes
}

func validateMemoryLimitsNegativeOrNull(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
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

func validateMemoryRequestsNegativeOrNull(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
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

func validateSubdomainDNSSubdomainRules(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if spec.Subdomain == "" {
		return causes
	}
	if errors := validation.IsDNS1123Subdomain(spec.Subdomain); len(errors) != 0 {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s does not conform to the kubernetes DNS_SUBDOMAIN rules : %s",
				field.Child("subdomain").String(), strings.Join(errors, ", ")),
			Field: field.Child("subdomain").String(),
		})
	}

	return causes
}

func validateHostNameNotConformingToDNSLabelRules(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if spec.Hostname == "" {
		return causes
	}
	if errors := validation.IsDNS1123Label(spec.Hostname); len(errors) != 0 {
		causes = appendNewStatusCauseForHostNameNotConformingToDNSLabelRules(field, causes, errors)
	}

	return causes
}

func validateRealtime(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if spec.Domain.CPU != nil && spec.Domain.CPU.Realtime != nil {
		causes = append(causes, validateCPURealtime(field, spec)...)
		causes = append(causes, validateMemoryRealtime(field, spec)...)
	}
	return causes
}

func validateArchitecture(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if spec.Architecture != "" && spec.Architecture != runtime.GOARCH && !config.MultiArchitectureEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("%s feature gate is not enabled in kubevirt-config, invalid entry %s", featuregate.MultiArchitecture,
				field.Child("architecture").String()),
			Field: field.Child("architecture").String(),
		})

	}
	return causes
}

func validateContainerDisks(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	for idx, volume := range spec.Volumes {
		if volume.ContainerDisk == nil || volume.ContainerDisk.Path == "" {
			continue
		}
		causes = append(causes, validatePath(field.Child("volumes").Index(idx).Child("containerDisk"), volume.ContainerDisk.Path)...)
	}
	return causes
}

func validatePath(field *k8sfield.Path, path string) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if path == "/" {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s must not point to root",
				field.String(),
			),
			Field: field.String(),
		})
		return causes
	}
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

	return causes

}

func validateCPURealtime(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
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

func validateMemoryRealtime(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
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

// ValidateVirtualMachineInstanceMandatoryFields should be invoked after all defaults and presets are applied.
// It is only meant to be used for VMI reviews, not if they are templates on other objects
func ValidateVirtualMachineInstanceMandatoryFields(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	requests := spec.Domain.Resources.Requests.Memory().Value()
	if requests != 0 {
		return causes
	}

	if spec.Domain.Memory == nil || spec.Domain.Memory != nil &&
		spec.Domain.Memory.Guest == nil && spec.Domain.Memory.Hugepages == nil {
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

func ValidateVirtualMachineInstanceMetadata(field *k8sfield.Path, metadata *metav1.ObjectMeta, config *virtconfig.ClusterConfig, isKubeVirtServiceAccount bool) []metav1.StatusCause {
	var causes []metav1.StatusCause
	annotations := metadata.Annotations
	labels := metadata.Labels
	// Validate kubevirt.io labels presence. Restricted labels allowed
	// to be created only by known service accounts
	if !isKubeVirtServiceAccount {
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

	return causes
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

	if dnsConfig == nil {
		return causes
	}
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

func validateFirmwareACPI(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if spec.Domain.Firmware == nil || spec.Domain.Firmware.ACPI == nil {
		return causes
	}

	acpi := spec.Domain.Firmware.ACPI
	if acpi.SlicNameRef == "" && acpi.MsdmNameRef == "" {
		return append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("ACPI was set but no SLIC nor MSDM volume reference was set"),
			Field:   field.String(),
		})
	}

	causes = append(causes, validateACPIRef(field, acpi.SlicNameRef, spec.Volumes, "slicNameRef")...)
	causes = append(causes, validateACPIRef(field, acpi.MsdmNameRef, spec.Volumes, "msdmNameRef")...)
	return causes
}

func validateACPIRef(field *k8sfield.Path, nameRef string, volumes []v1.Volume, fieldName string) []metav1.StatusCause {
	if nameRef == "" {
		return nil
	}

	for _, volume := range volumes {
		if nameRef != volume.Name {
			continue
		}

		if volume.Secret != nil {
			return nil
		}

		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s refers to Volume of unsupported type.", field.String()),
			Field:   field.Child(fieldName).String(),
		}}
	}

	return []metav1.StatusCause{{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: fmt.Sprintf("%s does not have a matching Volume.", field.String()),
		Field:   field.Child(fieldName).String(),
	}}
}

func validateFirmware(field *k8sfield.Path, firmware *v1.Firmware) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if firmware != nil {
		causes = append(causes, validateBootloader(field.Child("bootloader"), firmware.Bootloader)...)
		causes = append(causes, validateKernelBoot(field.Child("kernelBoot"), firmware.KernelBoot)...)
	}

	return causes
}

func efiBootEnabled(firmware *v1.Firmware) bool {
	return firmware != nil && firmware.Bootloader != nil && firmware.Bootloader.EFI != nil
}

func secureBootEnabled(firmware *v1.Firmware) bool {
	return efiBootEnabled(firmware) &&
		(firmware.Bootloader.EFI.SecureBoot == nil || *firmware.Bootloader.EFI.SecureBoot)
}

func smmFeatureEnabled(features *v1.Features) bool {
	return features != nil && features.SMM != nil && (features.SMM.Enabled == nil || *features.SMM.Enabled)
}

func validateDomainSpec(field *k8sfield.Path, spec *v1.DomainSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	causes = append(causes, validateDevices(field.Child("devices"), &spec.Devices)...)
	causes = append(causes, validateFirmware(field.Child("firmware"), spec.Firmware)...)

	if secureBootEnabled(spec.Firmware) && !smmFeatureEnabled(spec.Features) {
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

func validateDiskName(field *k8sfield.Path, idx int, disks []v1.Disk) []metav1.StatusCause {
	var causes []metav1.StatusCause
	for otherIdx, disk := range disks {
		if otherIdx < idx && disk.Name == disks[idx].Name {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s and %s must not have the same Name.", field.Index(idx).String(), field.Index(otherIdx).String()),
				Field:   field.Index(idx).Child("name").String(),
			})
		}
	}
	return causes
}

func validateDeviceTarget(field *k8sfield.Path, idx int, disk v1.Disk) []metav1.StatusCause {
	var causes []metav1.StatusCause
	deviceTargetSetCount := 0
	if disk.Disk != nil {
		deviceTargetSetCount++
	}
	if disk.LUN != nil {
		deviceTargetSetCount++
	}
	if disk.CDRom != nil {
		deviceTargetSetCount++
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
	return causes
}

func validatePciAddress(field *k8sfield.Path, idx int, disk v1.Disk) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if disk.Disk == nil || disk.Disk.PciAddress == "" {
		return causes
	}

	if disk.Disk.Bus != v1.DiskBusVirtio {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("disk %s - setting a PCI address is only possible with bus type virtio.", field.Child("domain", "devices", "disks", "disk").Index(idx).Child("name").String()),
			Field:   field.Child("domain", "devices", "disks", "disk").Index(idx).Child("pciAddress").String(),
		})
	}

	if _, err := hwutil.ParsePciAddress(disk.Disk.PciAddress); err != nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("disk %s has malformed PCI address (%s).", field.Child("domain", "devices", "disks", "disk").Index(idx).Child("name").String(), disk.Disk.PciAddress),
			Field:   field.Child("domain", "devices", "disks", "disk").Index(idx).Child("pciAddress").String(),
		})
	}
	return causes
}

func validateBootOrderValue(field *k8sfield.Path, idx int, disk v1.Disk) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if disk.BootOrder != nil && *disk.BootOrder < 1 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s must have a boot order > 0, if supplied", field.Index(idx).String()),
			Field:   field.Index(idx).Child("bootOrder").String(),
		})
	}
	return causes
}

func getDiskBus(disk v1.Disk) v1.DiskBus {
	switch {
	case disk.Disk != nil:
		return disk.Disk.Bus
	case disk.LUN != nil:
		return disk.LUN.Bus
	case disk.CDRom != nil:
		return disk.CDRom.Bus
	default:
		return ""
	}
}

func getDiskType(disk v1.Disk) string {
	switch {
	case disk.Disk != nil:
		return "disk"
	case disk.LUN != nil:
		return "lun"
	case disk.CDRom != nil:
		return "cdrom"
	default:
		return ""
	}
}

func validateBusSupport(field *k8sfield.Path, idx int, disk v1.Disk) []metav1.StatusCause {
	var causes []metav1.StatusCause
	bus := getDiskBus(disk)
	diskType := getDiskType(disk)
	if bus == "" {
		return causes
	}
	switch bus {
	case "ide":
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "IDE bus is not supported",
			Field:   field.Index(idx).Child(diskType, "bus").String(),
		})
	case v1.DiskBusVirtio:
		// special case. virtio is incompatible with CD-ROM for q35 machine types
		if diskType == "cdrom" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Bus type %s is invalid for CD-ROM device", bus),
				Field:   field.Index(idx).Child("cdrom", "bus").String(),
			})
		}
	case v1.DiskBusSATA:
		// sata disks (in contrast to sata cdroms) don't support readOnly
		if disk.Disk != nil && disk.Disk.ReadOnly {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s hard-disks do not support read-only.", bus),
				Field:   field.Index(idx).Child("disk", "bus").String(),
			})
		}
	case v1.DiskBusSCSI, v1.DiskBusUSB:
		break
	default:
		supportedBuses := []v1.DiskBus{v1.DiskBusVirtio, v1.DiskBusSCSI, v1.DiskBusSATA, v1.DiskBusUSB}
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s is set with an unrecognized bus %s, must be one of: %v", field.Index(idx).String(), bus, supportedBuses),
			Field:   field.Index(idx).Child(diskType, "bus").String(),
		})
	}
	// Reject defining DedicatedIOThread to a disk without VirtIO bus since this configuration
	// is not supported in libvirt.
	if disk.DedicatedIOThread != nil && *disk.DedicatedIOThread && bus != v1.DiskBusVirtio {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: fmt.Sprintf("IOThreads are not supported for disks on a %s bus", bus),
			Field:   field.Child("domain", "devices", "disks").Index(idx).String(),
		})
	}
	return causes
}

func validateSerialNumValue(field *k8sfield.Path, idx int, disk v1.Disk) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if disk.Serial != "" && !isValidExpression(disk.Serial) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s must be made up of the following characters [A-Za-z0-9_.+-], if specified", field.Index(idx).String()),
			Field:   field.Index(idx).Child("serial").String(),
		})
	}
	return causes
}

func validateSerialNumLength(field *k8sfield.Path, idx int, disk v1.Disk) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if disk.Serial != "" && len([]rune(disk.Serial)) > maxStrLen {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s must be less than or equal to %d in length, if specified", field.Index(idx).String(), maxStrLen),
			Field:   field.Index(idx).Child("serial").String(),
		})
	}
	return causes
}

func validateCacheMode(field *k8sfield.Path, idx int, disk v1.Disk) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if disk.Cache != "" && disk.Cache != v1.CacheNone && disk.Cache != v1.CacheWriteThrough && disk.Cache != v1.CacheWriteBack {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s has invalid value %s", field.Index(idx).Child("cache").String(), disk.Cache),
			Field:   field.Index(idx).Child("cache").String(),
		})
	}
	return causes
}

func validateIOMode(field *k8sfield.Path, idx int, disk v1.Disk) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if disk.IO != "" && disk.IO != v1.IONative && disk.IO != v1.IOThreads {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: fmt.Sprintf("Disk IO mode for %s is not supported. Supported modes are: native, threads.", field),
			Field:   field.Child("domain", "devices", "disks").Index(idx).Child("io").String(),
		})
	}
	return causes
}

func validateErrorPolicy(field *k8sfield.Path, idx int, disk v1.Disk) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if disk.ErrorPolicy != nil && *disk.ErrorPolicy != v1.DiskErrorPolicyStop && *disk.ErrorPolicy != v1.DiskErrorPolicyIgnore && *disk.ErrorPolicy != v1.DiskErrorPolicyReport && *disk.ErrorPolicy != v1.DiskErrorPolicyEnospace {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s has invalid value \"%s\"", field.Index(idx).Child("errorPolicy").String(), *disk.ErrorPolicy),
			Field:   field.Index(idx).Child("errorPolicy").String(),
		})
	}
	return causes
}

func validateDiskNameAsContainerName(field *k8sfield.Path, idx int, disk v1.Disk) []metav1.StatusCause {
	var causes []metav1.StatusCause
	for _, err := range validation.IsDNS1123Label(disk.Name) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: err,
			Field:   field.Child("domain", "devices", "disks").Index(idx).Child("name").String(),
		})
	}
	return causes
}

func validateCustomBlockSize(field *k8sfield.Path, idx int, blockType string, size uint) []metav1.StatusCause {
	var causes []metav1.StatusCause
	switch {
	case size < minCustomBlockSize:
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Provided size of %d is less than the supported minimum size of %d", size, minCustomBlockSize),
			Field:   field.Index(idx).Child("blockSize").Child("custom").Child(blockType).String(),
		})
	case size > maxCustomBlockSize:
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Provided size of %d is greater than the supported maximum size of %d", size, maxCustomBlockSize),
			Field:   field.Index(idx).Child("blockSize").Child("custom").Child(blockType).String(),
		})
	case size&(size-1) != 0:
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Provided size of %d is not a power of 2", size),
			Field:   field.Index(idx).Child("blockSize").Child("custom").Child(blockType).String(),
		})
	}
	return causes
}

func validateBlockSize(field *k8sfield.Path, idx int, disk v1.Disk) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if disk.BlockSize == nil || disk.BlockSize.Custom == nil {
		return causes
	}
	if disk.BlockSize.MatchVolume != nil && (disk.BlockSize.MatchVolume.Enabled == nil || *disk.BlockSize.MatchVolume.Enabled) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Block size matching can't be enabled together with a custom value",
			Field:   field.Index(idx).Child("blockSize").String(),
		})
		return causes
	}
	customSize := disk.BlockSize.Custom
	if customSize.Logical > customSize.Physical {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Logical size %d must be the same or less than the physical size of %d", customSize.Logical, customSize.Physical),
			Field:   field.Index(idx).Child("blockSize").Child("custom").Child("logical").String(),
		})
	} else {
		causes = append(causes, validateCustomBlockSize(field, idx, "logical", customSize.Logical)...)
		causes = append(causes, validateCustomBlockSize(field, idx, "physical", customSize.Physical)...)
	}
	return causes
}

func validateDisks(field *k8sfield.Path, disks []v1.Disk) []metav1.StatusCause {
	var causes []metav1.StatusCause
	for idx, disk := range disks {
		causes = append(causes, validateDiskName(field, idx, disks)...)
		causes = append(causes, validateDeviceTarget(field, idx, disk)...)
		causes = append(causes, validatePciAddress(field, idx, disk)...)
		causes = append(causes, validateBootOrderValue(field, idx, disk)...)
		causes = append(causes, validateBusSupport(field, idx, disk)...)
		causes = append(causes, validateSerialNumValue(field, idx, disk)...)
		causes = append(causes, validateSerialNumLength(field, idx, disk)...)
		causes = append(causes, validateCacheMode(field, idx, disk)...)
		causes = append(causes, validateIOMode(field, idx, disk)...)
		causes = append(causes, validateErrorPolicy(field, idx, disk)...)
		// Verify disk and volume name can be a valid container name since disk
		// name can become a container name which will fail to schedule if invalid
		causes = append(causes, validateDiskNameAsContainerName(field, idx, disk)...)
		causes = append(causes, validateBlockSize(field, idx, disk)...)
	}
	return causes
}

// Rejects kernel boot defined with initrd/kernel path but without an image
func validateKernelBoot(field *k8sfield.Path, kernelBoot *v1.KernelBoot) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if kernelBoot == nil {
		return causes
	}

	if kernelBoot.Container == nil {
		if kernelBoot.KernelArgs != "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "kernel arguments cannot be provided without an external kernel",
				Field:   field.Child("kernelArgs").String(),
			})
		}
		return causes
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

	return causes
}

// validateSpecAffinity is function that validate spec.affinity
// instead of bring in the whole kubernetes lib we simply copy it from kubernetes/pkg/apis/core/validation/validation.go
func validateSpecAffinity(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if spec.Affinity == nil {
		return causes
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

	return causes
}

// validateSpecTopologySpreadConstraints is function that validate spec.validateSpecTopologySpreadConstraints
// instead of bring in the whole kubernetes lib we simply copy it from kubernetes/pkg/apis/core/validation/validation.go
func validateSpecTopologySpreadConstraints(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if spec.TopologySpreadConstraints == nil {
		return causes
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

	return causes
}

func validateVSOCK(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if spec.Domain.Devices.AutoattachVSOCK == nil || !*spec.Domain.Devices.AutoattachVSOCK {
		return causes
	}

	if !config.VSOCKEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s feature gate is not enabled in kubevirt-config", featuregate.VSOCKGate),
			Field:   field.Child("domain", "devices", "autoattachVSOCK").String(),
		})
	}

	return causes
}

func validatePersistentReservation(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if !reservation.HasVMISpecPersistentReservation(spec) {
		return causes
	}

	if !config.PersistentReservationEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s feature gate is not enabled in kubevirt-config", featuregate.PersistentReservation),
			Field:   field.Child("domain", "devices", "disks", "luns", "reservation").String(),
		})
	}

	return causes
}

func validateCPUHotplug(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
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

func validateVideoConfig(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if spec.Domain.Devices.Video == nil {
		return causes
	}

	if !config.VideoConfigEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Video configuration is specified but the %s feature gate is not enabled", featuregate.VideoConfig),
			Field:   field.Child("video").String(),
		})
		return causes
	}

	if spec.Domain.Devices.AutoattachGraphicsDevice != nil && !*spec.Domain.Devices.AutoattachGraphicsDevice {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Video configuration is not allowed when autoattachGraphicsDevice is set to false",
			Field:   field.Child("video").String(),
		})
	}

	return causes
}

func validatePanicDeviceModel(field *k8sfield.Path, model *v1.PanicDeviceModel) *metav1.StatusCause {
	if model == nil {
		return nil
	}
	if !slices.Contains(validPanicDeviceModels, *model) {
		return &metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf(invalidPanicDeviceModelErrFmt, *model),
			Field:   field.String(),
		}
	}
	return nil
}

func validatePanicDevices(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if len(spec.Domain.Devices.PanicDevices) == 0 {
		return causes
	}
	if spec.Domain.Devices.PanicDevices != nil && !config.PanicDevicesEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Panic Devices feature gate is not enabled in kubevirt-config",
			Field:   field.Child("domain", "devices", "panicDevices").String(),
		})
		return causes
	}

	arch := spec.Architecture
	if arch == "" {
		arch = config.GetDefaultArchitecture()
	}

	if arch == "s390x" || arch == "ppc64le" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("custom panic devices are not supported on %s architecture", arch),
			Field:   field.Child("domain", "devices", "panicDevices").String(),
		})
	}

	for idx, panicDevice := range spec.Domain.Devices.PanicDevices {
		if cause := validatePanicDeviceModel(field.Child("domain", "devices", "panicDevices").Index(idx).Child("model"), panicDevice.Model); cause != nil {
			causes = append(causes, *cause)
		}
	}

	return causes
}
