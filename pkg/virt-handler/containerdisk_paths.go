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

package virthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// isLegacyContainerDiskNaming returns true if this domain was started
// before v2 (volume-name-based) container disk paths were introduced.
func isLegacyContainerDiskNaming(domain *api.Domain) bool {
	return domain.Spec.Metadata.KubeVirt.ContainerDiskNaming != "v2"
}

// buildContainerDiskPathMap inspects the live domain XML and returns
// a map of volumeName → actual file path on disk.
// Only meaningful for legacy (pre-v2) domains.
func buildContainerDiskPathMap(vmi *v1.VirtualMachineInstance, domain *api.Domain) map[string]string {
	pathMap := make(map[string]string)

	// Build alias → volumeName lookup. KubeVirt sets disk alias to "ua-{volumeName}".
	aliasToVolume := make(map[string]string)
	for _, vol := range vmi.Spec.Volumes {
		if vol.ContainerDisk != nil {
			aliasToVolume[api.UserAliasPrefix+vol.Name] = vol.Name
		}
	}

	for _, disk := range domain.Spec.Devices.Disks {
		if disk.Alias == nil {
			continue
		}
		volumeName, ok := aliasToVolume[disk.Alias.GetName()]
		if !ok {
			continue
		}
		if disk.Source.File == "" {
			continue
		}
		filePath := disk.Source.File
		// Only record index-based filenames (e.g. disk_2.img), not already-v2 names
		base := filepath.Base(filePath)
		if strings.HasPrefix(base, "disk_") && strings.HasSuffix(base, ".img") {
			pathMap[volumeName] = filePath
		}
	}
	return pathMap
}

// syncContainerDiskPathAnnotation writes the legacy container disk path
// annotation on the VMI so the migration target can set up bind mounts.
// Called from the main reconcile loop when the VMI is running.
func (c *VirtualMachineController) syncContainerDiskPathAnnotation(
	vmi *v1.VirtualMachineInstance,
	domain *api.Domain,
) error {
	if domain == nil {
		return nil
	}

	// Nothing to do for new-style VMs
	if !isLegacyContainerDiskNaming(domain) {
		return nil
	}

	// Idempotent: skip if already annotated
	if _, exists := vmi.Annotations[v1.ContainerDiskPathsAnnotation]; exists {
		return nil
	}

	// Skip VMIs with no container disks
	hasContainerDisk := false
	for _, vol := range vmi.Spec.Volumes {
		if vol.ContainerDisk != nil {
			hasContainerDisk = true
			break
		}
	}
	if !hasContainerDisk {
		return nil
	}

	pathMap := buildContainerDiskPathMap(vmi, domain)
	if len(pathMap) == 0 {
		return nil
	}

	encoded, err := json.Marshal(pathMap)
	if err != nil {
		return fmt.Errorf("failed to marshal containerdisk path map: %v", err)
	}

	log.Log.Object(vmi).Infof("Annotating VMI with legacy container disk paths: %s", string(encoded))

	if vmi.Annotations == nil {
		vmi.Annotations = map[string]string{}
	}
	vmi.Annotations[v1.ContainerDiskPathsAnnotation] = string(encoded)

	_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Update(
		context.Background(), vmi, metav1.UpdateOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to update VMI with containerdisk paths annotation: %v", err)
	}
	return nil
}
