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
	"path/filepath"
	"regexp"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	hwutil "kubevirt.io/kubevirt/pkg/util/hardware"
)

const (
	maxStrLen = 256

	// Should be a power of 2
	minCustomBlockSize = 512
	maxCustomBlockSize = 2097152 // 2 MB
)

var isValidExpression = regexp.MustCompile(`^[A-Za-z0-9_.+-]+$`).MatchString

func ValidateDisks(field *k8sfield.Path, disks []v1.Disk) []metav1.StatusCause {
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

func ValidateContainerDisks(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	for idx, volume := range spec.Volumes {
		if volume.ContainerDisk == nil || volume.ContainerDisk.Path == "" {
			continue
		}
		causes = append(causes, ValidatePath(field.Child("volumes").Index(idx).Child("containerDisk"), volume.ContainerDisk.Path)...)
	}
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
	} else if getDiskBus(disk) == v1.DiskBusSATA && customSize.Logical != minCustomBlockSize {
		// For IDE and SATA disks in QEMU, the emulated controllers only support a logical size of 512 bytes.
		// https://gitlab.com/qemu-project/qemu/-/blob/f0007b7f03e2d7fc33e71c3a582f2364c51a226b/hw/ide/ide-dev.c#L105
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Logical size %d must be %d for SATA devices", customSize.Logical, minCustomBlockSize),
			Field:   field.Index(idx).Child("blockSize").Child("custom").Child("logical").String(),
		})
	} else if customSize.DiscardGranularity != nil && customSize.Logical != 0 && *customSize.DiscardGranularity%customSize.Logical != 0 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Discard granularity %d must be multiples of logical size %d", *customSize.DiscardGranularity, customSize.Logical),
			Field:   field.Index(idx).Child("blockSize").Child("custom").Child("discardGranularity").String(),
		})
	} else {
		causes = append(causes, validateCustomBlockSize(field, idx, "logical", customSize.Logical)...)
		causes = append(causes, validateCustomBlockSize(field, idx, "physical", customSize.Physical)...)
	}
	return causes
}

func ValidatePath(field *k8sfield.Path, path string) []metav1.StatusCause {
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
