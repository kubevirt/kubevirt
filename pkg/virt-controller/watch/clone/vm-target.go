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

	"kubevirt.io/client-go/log"

	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"
	k6tv1 "kubevirt.io/api/core/v1"
)

func generatePatches(source *k6tv1.VirtualMachine, cloneSpec *clonev1alpha1.VirtualMachineCloneSpec) (patches []string) {

	macAddressPatches := generateMacAddressPatches(source.Spec.Template.Spec.Domain.Devices.Interfaces, cloneSpec.NewMacAddresses)
	patches = append(patches, macAddressPatches...)

	smBiosPatches := generateSmbiosSerialPatches(source.Spec.Template.Spec.Domain.Firmware, cloneSpec.NewSMBiosSerial)
	patches = append(patches, smBiosPatches...)

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
