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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package clone

import (
	"fmt"
	"regexp"
	"strings"

	"kubevirt.io/client-go/log"

	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"
	k6tv1 "kubevirt.io/api/core/v1"
)

func generatePatches(source *k6tv1.VirtualMachine, cloneSpec *clonev1alpha1.VirtualMachineCloneSpec) (patches []string) {

	macAddressPatches := generateMacAddressPatches(source.Spec.Template.Spec.Domain.Devices.Interfaces, cloneSpec.NewMacAddresses)
	patches = append(patches, macAddressPatches...)

	smBiosPatches := generateSmbiosSerialPatches(source.Spec.Template.Spec.Domain.Firmware, cloneSpec.NewSMBiosSerial)
	patches = append(patches, smBiosPatches...)

	labelsPatches := generateLabelPatches(source.Labels, cloneSpec.LabelFilters)
	patches = append(patches, labelsPatches...)

	annotationPatches := generateAnnotationPatches(source.Annotations, cloneSpec.AnnotationFilters)
	patches = append(patches, annotationPatches...)

	templateLabelsPatches := generateTemplateLabelPatches(source.Spec.Template.ObjectMeta.Labels, cloneSpec.Template.LabelFilters)
	patches = append(patches, templateLabelsPatches...)

	templateAnnotationPatches := generateTemplateAnnotationPatches(source.Spec.Template.ObjectMeta.Annotations, cloneSpec.Template.AnnotationFilters)
	patches = append(patches, templateAnnotationPatches...)

	firmwareUUIDPatches := generateFirmwareUUIDPatches(source.Spec.Template.Spec.Domain.Firmware)
	patches = append(patches, firmwareUUIDPatches...)

	log.Log.V(defaultVerbosityLevel).Object(source).Infof("patches generated for vm %s clone: %v", source.Name, patches)
	return patches
}

func generateMacAddressPatches(interfaces []k6tv1.Interface, newMacAddresses map[string]string) (patches []string) {
	const macAddressPatchPattern = `{"op": "replace", "path": "/spec/template/spec/domain/devices/interfaces/%d/macAddress", "value": "%s"}`

	for idx, iface := range interfaces {
		// If a new mac address is not specified for the current interface an empty mac address would be assigned.
		// This is OK for clusters that have Kube Mac Pool enabled. For clusters that don't have KMP it is the users'
		// responsibility to assign new mac address to every network interface.
		newMac := newMacAddresses[iface.Name]
		patches = append(patches, fmt.Sprintf(macAddressPatchPattern, idx, newMac))
	}

	return patches
}

func generateSmbiosSerialPatches(firmware *k6tv1.Firmware, newSMBiosSerial *string) (patches []string) {
	const smbiosSerialPatchPattern = `{"op": "replace", "path": "/spec/template/spec/domain/firmware/serial", "value": "%s"}`

	if firmware == nil {
		return
	}

	newSerial := ""
	if newSMBiosSerial != nil {
		newSerial = *newSMBiosSerial
	}

	patch := fmt.Sprintf(smbiosSerialPatchPattern, newSerial)
	return []string{patch}
}

func generateLabelPatches(labels map[string]string, filters []string) (patches []string) {
	const basePath = "/metadata/labels"
	return generateStrStrMapPatches(labels, filters, basePath)
}

func generateTemplateLabelPatches(labels map[string]string, filters []string) (patches []string) {
	const basePath = "/spec/template/metadata/labels"
	return generateStrStrMapPatches(labels, filters, basePath)
}

func generateAnnotationPatches(annotations map[string]string, filters []string) (patches []string) {
	const basePath = "/metadata/annotations"
	return generateStrStrMapPatches(annotations, filters, basePath)
}

func generateTemplateAnnotationPatches(annotations map[string]string, filters []string) (patches []string) {
	const basePath = "/spec/template/metadata/annotations"
	return generateStrStrMapPatches(annotations, filters, basePath)
}

func generateStrStrMapPatches(m map[string]string, filters []string, baseJsonPath string) (patches []string) {
	appendRemovalPatch := func(key string) {
		const patchPattern = `{"op": "remove", "path": "%s/%s"}`

		key = addKeyEscapeCharacters(key)
		patches = append(patches, fmt.Sprintf(patchPattern, baseJsonPath, key))
	}

	if filters == nil {
		return nil
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
	for key, _ := range m {
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
	for originalKey, _ := range m {
		if _, isIncluded := includedKeys[originalKey]; !isIncluded {
			appendRemovalPatch(originalKey)
		}
	}

	return patches
}

// Replaces "/" and "~" chars with their escape characters. For more info: http://jsonpatch.com.
func addKeyEscapeCharacters(key string) string {
	const (
		tilda           = "~"
		slash           = "/"
		tildaEscapeChar = "~0"
		slashEscapeChar = "~1"
	)

	// Important to replace tilda first since slash's escape character also contains a tilda char
	key = strings.ReplaceAll(key, tilda, tildaEscapeChar)
	key = strings.ReplaceAll(key, slash, slashEscapeChar)

	return key
}

func generateFirmwareUUIDPatches(firmware *k6tv1.Firmware) (patches []string) {
	const firmwareUUIDPatch = `{"op": "replace", "path": "/spec/template/spec/domain/firmware/uuid", "value": ""}`

	if firmware == nil {
		return nil
	}

	return []string{firmwareUUIDPatch}
}
