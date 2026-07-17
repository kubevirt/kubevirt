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
// replacing the source domain namespace/name with the target domain namespace/name.
// This is the target-side equivalent of the source-side
// updateFilePathsToNewDomain() + convertDisks() functions.
func DiskSourcePathHook(_ *convertertypes.ConverterContext, vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain) error {
	if domain.Devices == nil {
		return nil
	}

	sourceDomainNamespace := getSourceDomainNamespace(vmi)
	targetDomainNamespace := getTargetDomainNamespace(vmi)
	sourceDomainName := getSourceDomainName(vmi)
	targetDomainName := getTargetDomainName(vmi)

	nsUnchanged := targetDomainNamespace == "" || sourceDomainNamespace == "" || sourceDomainNamespace == targetDomainNamespace
	nameUnchanged := targetDomainName == "" || sourceDomainName == "" || sourceDomainName == targetDomainName
	if nsUnchanged && nameUnchanged {
		return nil
	}

	for i := range domain.Devices.Disks {
		disk := &domain.Devices.Disks[i]
		if disk.Source == nil {
			continue
		}
		if disk.Source.File != nil {
			oldPath := disk.Source.File.File
			newPath := rewriteDomainPath(oldPath, sourceDomainNamespace, targetDomainNamespace, sourceDomainName, targetDomainName)
			if newPath != oldPath {
				log.Log.Object(vmi).V(4).Infof("diskSourcePathHook: updating disk file path from %s to %s", oldPath, newPath)
				disk.Source.File.File = newPath
			}
		}
		if disk.Source.DataStore != nil && disk.Source.DataStore.Source != nil &&
			disk.Source.DataStore.Source.File != nil {
			oldPath := disk.Source.DataStore.Source.File.File
			newPath := rewriteDomainPath(oldPath, sourceDomainNamespace, targetDomainNamespace, sourceDomainName, targetDomainName)
			if newPath != oldPath {
				log.Log.Object(vmi).V(4).Infof("diskSourcePathHook: updating datastore file path from %s to %s", oldPath, newPath)
				disk.Source.DataStore.Source.File.File = newPath
			}
		}
	}

	return nil
}

// rewriteDomainPath replaces /{sourceNS}/ and /{sourceName}/ path segments with the
// target equivalents. Namespace is rewritten before name so paths like
// .../cloud-init-data/{ns}/{name}/... are updated correctly for decentralized migration.
// A segment at the end of the path (no trailing slash) is also rewritten.
func rewriteDomainPath(path, sourceNS, targetNS, sourceName, targetName string) string {
	newPath := replacePathSegment(path, sourceNS, targetNS)
	return replacePathSegment(newPath, sourceName, targetName)
}

// replacePathSegment rewrites the first path segment matching oldSeg to newSeg.
// Matches either "/{oldSeg}/" mid-path or "/{oldSeg}" at the end of the path.
func replacePathSegment(path, oldSeg, newSeg string) string {
	if oldSeg == "" || newSeg == "" || oldSeg == newSeg {
		return path
	}
	mid := "/" + oldSeg + "/"
	if strings.Contains(path, mid) {
		return strings.Replace(path, mid, "/"+newSeg+"/", 1)
	}
	suffix := "/" + oldSeg
	if strings.HasSuffix(path, suffix) {
		return strings.TrimSuffix(path, oldSeg) + newSeg
	}
	return path
}

func getSourceDomainNamespace(vmi *v1.VirtualMachineInstance) string {
	if vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.SourceState != nil &&
		vmi.Status.MigrationState.SourceState.DomainNamespace != nil &&
		*vmi.Status.MigrationState.SourceState.DomainNamespace != "" {
		return *vmi.Status.MigrationState.SourceState.DomainNamespace
	}
	// Do not fall back to vmi.Namespace: on the target this hook runs against the
	// target VMI, whose Namespace is the destination, not the source embedded in DestXML.
	return ""
}

func getTargetDomainNamespace(vmi *v1.VirtualMachineInstance) string {
	if vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.TargetState != nil &&
		vmi.Status.MigrationState.TargetState.DomainNamespace != nil &&
		*vmi.Status.MigrationState.TargetState.DomainNamespace != "" {
		return *vmi.Status.MigrationState.TargetState.DomainNamespace
	}
	return ""
}

func getSourceDomainName(vmi *v1.VirtualMachineInstance) string {
	if vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.SourceState != nil &&
		vmi.Status.MigrationState.SourceState.DomainName != nil &&
		*vmi.Status.MigrationState.SourceState.DomainName != "" {
		return *vmi.Status.MigrationState.SourceState.DomainName
	}
	// Do not fall back to vmi.Name: on the target this hook runs against the target VMI,
	// whose Name is the destination name, not the source name embedded in DestXML paths.
	return ""
}

func getTargetDomainName(vmi *v1.VirtualMachineInstance) string {
	if vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.TargetState != nil &&
		vmi.Status.MigrationState.TargetState.DomainName != nil &&
		*vmi.Status.MigrationState.TargetState.DomainName != "" {
		return *vmi.Status.MigrationState.TargetState.DomainName
	}
	return vmi.Name
}
