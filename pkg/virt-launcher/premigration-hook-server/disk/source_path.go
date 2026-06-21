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
 */

package disk

import (
	"strings"

	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

// DiskSourcePathHook updates disk source file paths in the domain XML by
// replacing the source domain namespace with the target domain namespace.
// This is the target-side equivalent of the source-side
// updateFilePathsToNewDomain() + convertDisks() functions.
func DiskSourcePathHook(_ *convertertypes.ConverterContext, vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain) error {
	if domain.Devices == nil {
		return nil
	}

	sourceDomainNamespace := getSourceDomainNamespace(vmi)
	targetDomainNamespace := getTargetDomainNamespace(vmi)
	if targetDomainNamespace == "" || sourceDomainNamespace == targetDomainNamespace {
		return nil
	}

	oldSegment := "/" + sourceDomainNamespace + "/"
	newSegment := "/" + targetDomainNamespace + "/"

	for i := range domain.Devices.Disks {
		disk := &domain.Devices.Disks[i]
		if disk.Source == nil {
			continue
		}
		if disk.Source.File != nil && strings.Contains(disk.Source.File.File, oldSegment) {
			oldPath := disk.Source.File.File
			newPath := strings.Replace(oldPath, oldSegment, newSegment, 1)
			log.Log.Object(vmi).V(4).Infof("diskSourcePathHook: updating disk file path from %s to %s", oldPath, newPath)
			disk.Source.File.File = newPath
		}
		if disk.Source.DataStore != nil && disk.Source.DataStore.Source != nil &&
			disk.Source.DataStore.Source.File != nil &&
			strings.Contains(disk.Source.DataStore.Source.File.File, oldSegment) {
			oldPath := disk.Source.DataStore.Source.File.File
			newPath := strings.Replace(oldPath, oldSegment, newSegment, 1)
			log.Log.Object(vmi).V(4).Infof("diskSourcePathHook: updating datastore file path from %s to %s", oldPath, newPath)
			disk.Source.DataStore.Source.File.File = newPath
		}
	}

	return nil
}

func getSourceDomainNamespace(vmi *v1.VirtualMachineInstance) string {
	if vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.SourceState != nil &&
		vmi.Status.MigrationState.SourceState.DomainNamespace != nil {
		return *vmi.Status.MigrationState.SourceState.DomainNamespace
	}
	return vmi.Namespace
}

func getTargetDomainNamespace(vmi *v1.VirtualMachineInstance) string {
	if vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.TargetState != nil &&
		vmi.Status.MigrationState.TargetState.DomainNamespace != nil {
		return *vmi.Status.MigrationState.TargetState.DomainNamespace
	}
	return ""
}
