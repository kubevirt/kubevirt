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

package clone

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"kubevirt.io/client-go/log"

	clone "kubevirt.io/api/clone/v1beta1"
	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

func generatePatches(source *k6tv1.VirtualMachine, cloneSpec *clone.VirtualMachineCloneSpec) ([]string, error) {
	patchSet := patch.New()
	addMacAddressPatches(patchSet, source.Spec.Template.Spec.Domain.Devices.Interfaces, cloneSpec.NewMacAddresses)
	addSmbiosSerialPatches(patchSet, source.Spec.Template.Spec.Domain.Firmware, cloneSpec.NewSMBiosSerial)
	addRemovePatchesFromFilter(patchSet, source.Labels, cloneSpec.LabelFilters, "/metadata/labels")
	addAnnotationPatches(patchSet, source.Annotations, cloneSpec.AnnotationFilters)
	addRemovePatchesFromFilter(patchSet, source.Spec.Template.ObjectMeta.Labels, cloneSpec.Template.LabelFilters, "/spec/template/metadata/labels")
	addRemovePatchesFromFilter(patchSet, source.Spec.Template.ObjectMeta.Annotations, cloneSpec.Template.AnnotationFilters, "/spec/template/metadata/annotations")
	addFirmwareUUIDPatches(patchSet, source.Spec.Template.Spec.Domain.Firmware)

	patches, err := generateStringPatchOperations(patchSet)
	if err != nil {
		return nil, err
	}

	patches = append(patches, cloneSpec.Patches...)

	log.Log.V(defaultVerbosityLevel).Object(source).Infof("patches generated for vm %s clone: %v", source.Name, patches)
	return patches, nil
}

func generateStringPatchOperations(set *patch.PatchSet) ([]string, error) {
	var patches []string
	for _, patchOp := range set.GetPatches() {
		payloadBytes, err := json.Marshal(patchOp)
		if err != nil {
			return nil, err
		}
		patches = append(patches, string(payloadBytes))
	}

	return patches, nil
}

func addMacAddressPatches(patchSet *patch.PatchSet, interfaces []k6tv1.Interface, newMacAddresses map[string]string) {
	for idx, iface := range interfaces {
		// If a new mac address is not specified for the current interface an empty mac address would be assigned.
		// This is OK for clusters that have Kube Mac Pool enabled. For clusters that don't have KMP it is the users'
		// responsibility to assign new mac address to every network interface.
		newMac := newMacAddresses[iface.Name]
		patchSet.AddOption(patch.WithReplace(fmt.Sprintf("/spec/template/spec/domain/devices/interfaces/%d/macAddress", idx), newMac))
	}
}

func addSmbiosSerialPatches(patchSet *patch.PatchSet, firmware *k6tv1.Firmware, newSMBiosSerial *string) {
	if firmware == nil {
		return
	}

	newSerial := ""
	if newSMBiosSerial != nil {
		newSerial = *newSMBiosSerial
	}

	patchSet.AddOption(patch.WithReplace("/spec/template/spec/domain/firmware/serial", newSerial))
}

func addAnnotationPatches(patchSet *patch.PatchSet, annotations map[string]string, filters []string) {
	// Some keys are needed for restore functionality.
	// Deleting the item from the annotation list prevents
	// from remove patch being generated
	delete(annotations, "restore.kubevirt.io/lastRestoreUID")
	addRemovePatchesFromFilter(patchSet, annotations, filters, "/metadata/annotations")
}

func addRemovePatchesFromFilter(patchSet *patch.PatchSet, m map[string]string, filters []string, baseJSONPath string) {
	if filters == nil {
		return
	}

	var regularFilters, negationFilters []string
	for _, filter := range filters {
		// wildcard alone is not a legal wildcard
		if filter == "*" {
			regularFilters = append(regularFilters, ".*")
			continue
		}

		if strings.HasPrefix(filter, "!") {
			negationFilters = append(negationFilters, filter[1:])
		} else {
			regularFilters = append(regularFilters, filter)
		}
	}

	matchRegex := func(regex, s string) (matched bool) {
		var err error

		matched, err = regexp.MatchString(regex, s)
		if err != nil {
			log.Log.Errorf("matching regex %s to string %s failed: %v", regex, s, err)
		}
		return matched
	}

	includedKeys := map[string]struct{}{}
	// Negation filters have precedence, therefore regular filters would be applied first
	for key := range m {
		for _, filter := range regularFilters {
			if matchRegex(filter, key) {
				includedKeys[key] = struct{}{}
			}
		}

		for _, negationFilter := range negationFilters {
			if matchRegex(negationFilter, key) {
				delete(includedKeys, key)
			}
		}
	}

	// Appending removal patches
	for originalKey := range m {
		if _, isIncluded := includedKeys[originalKey]; !isIncluded {
			patchSet.AddOption(patch.WithRemove(fmt.Sprintf("%s/%s", baseJSONPath, patch.EscapeJSONPointer(originalKey))))
		}
	}
}

func addFirmwareUUIDPatches(patchSet *patch.PatchSet, firmware *k6tv1.Firmware) {
	if firmware == nil {
		return
	}

	patchSet.AddOption(patch.WithReplace("/spec/template/spec/domain/firmware/uuid", ""))
}
