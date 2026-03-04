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
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// detectPCITopologyAndAnnotate checks whether a running VMI needs PCI topology
// version annotation. If the annotation is absent, it inspects the domain to
// determine whether the VM was created under v1 or v2 topology and annotates
// accordingly. This is only needed during the upgrade window.
func (c *VirtualMachineController) detectPCITopologyAndAnnotate(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if domain == nil {
		return nil
	}

	if !vmi.IsRunning() {
		return nil
	}

	if _, exists := vmi.Annotations[v1.PciTopologyVersionAnnotation]; exists {
		return nil
	}

	detectedCount, err := detectPlaceholderCount(vmi, domain, c.clientset)
	if err != nil {
		return fmt.Errorf("failed to detect PCI placeholder count: %v", err)
	}
	v1Expected := calculateHotplugPortCountV1ForDetection(vmi)

	patchOps := patch.New()
	if detectedCount != v1Expected {
		hotpluggedIfaces, err := countHotpluggedInterfaces(vmi, c.clientset)
		if err != nil {
			return fmt.Errorf("failed to count hotplugged interfaces: %v", err)
		}
		bootIfaceCount := len(vmi.Spec.Domain.Devices.Interfaces) - hotpluggedIfaces
		slotTotal := detectedCount + bootIfaceCount
		patchOps.AddOption(
			patch.WithAdd("/metadata/annotations/"+patch.EscapeJSONPointer(v1.PciTopologyVersionAnnotation), v1.PciTopologyVersionV2),
			patch.WithAdd("/metadata/annotations/"+patch.EscapeJSONPointer(v1.PciInterfaceSlotCountAnnotation), strconv.Itoa(slotTotal)),
		)
	} else {
		patchOps.AddOption(
			patch.WithAdd("/metadata/annotations/"+patch.EscapeJSONPointer(v1.PciTopologyVersionAnnotation), v1.PciTopologyVersionV3),
		)
	}

	patchBytes, err := patchOps.GeneratePayload()
	if err != nil {
		return fmt.Errorf("failed to generate PCI topology annotation patch: %v", err)
	}

	_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(
		context.Background(),
		vmi.Name,
		types.JSONPatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to patch VMI with PCI topology annotation: %v", err)
	}

	return nil
}

// detectPlaceholderCount estimates the original placeholder count from the
// running domain. It works by identifying which root port buses are occupied
// by devices, finding the highest such bus, and counting empty root ports
// at or below that bus. Root ports above the highest occupied bus are
// libvirt-added spares and are excluded. Hotplugged devices that consumed
// root ports at runtime are added back since they were originally
// placeholder slots.
func detectPlaceholderCount(vmi *v1.VirtualMachineInstance, domain *api.Domain, client kubernetes.Interface) (int, error) {
	occupiedBuses := collectOccupiedBuses(domain)
	highestOccupied := 0
	for bus := range occupiedBuses {
		if bus > highestOccupied {
			highestOccupied = bus
		}
	}

	emptyBelowHighest := 0
	for _, ctrl := range domain.Spec.Devices.Controllers {
		if ctrl.Type != "pci" || ctrl.Model != "pcie-root-port" {
			continue
		}
		bus, err := strconv.Atoi(ctrl.Index)
		if err != nil {
			continue
		}
		if bus > highestOccupied {
			continue // spare port added by libvirt
		}
		if !occupiedBuses[bus] {
			emptyBelowHighest++
		}
	}

	hotplugged, err := countHotpluggedDevices(vmi, client)
	if err != nil {
		return 0, err
	}

	return emptyBelowHighest + hotplugged, nil
}

// collectOccupiedBuses returns a set of PCI bus numbers that have devices
// on them. Each pcie-root-port with index N provides bus N, so a bus being
// occupied means that root port is in use.
func collectOccupiedBuses(domain *api.Domain) map[int]bool {
	buses := make(map[int]bool)

	markBus := func(addr *api.Address) {
		if bus, ok := parsePCIBus(addr); ok && bus > 0 {
			buses[bus] = true
		}
	}

	for i := range domain.Spec.Devices.Disks {
		markBus(domain.Spec.Devices.Disks[i].Address)
	}
	for i := range domain.Spec.Devices.Interfaces {
		markBus(domain.Spec.Devices.Interfaces[i].Address)
	}
	for i := range domain.Spec.Devices.HostDevices {
		markBus(domain.Spec.Devices.HostDevices[i].Address)
	}
	if domain.Spec.Devices.Rng != nil {
		markBus(domain.Spec.Devices.Rng.Address)
	}
	if domain.Spec.Devices.Ballooning != nil {
		markBus(domain.Spec.Devices.Ballooning.Address)
	}
	for i := range domain.Spec.Devices.Watchdogs {
		markBus(domain.Spec.Devices.Watchdogs[i].Address)
	}
	for i := range domain.Spec.Devices.Inputs {
		markBus(domain.Spec.Devices.Inputs[i].Address)
	}
	for i := range domain.Spec.Devices.Controllers {
		if domain.Spec.Devices.Controllers[i].Type == "pci" {
			continue
		}
		markBus(domain.Spec.Devices.Controllers[i].Address)
	}

	return buses
}

// parsePCIBus extracts the bus number from a PCI address.
func parsePCIBus(addr *api.Address) (int, bool) {
	if addr == nil || addr.Type != "pci" {
		return 0, false
	}
	bus, err := strconv.ParseInt(strings.TrimPrefix(addr.Bus, "0x"), 16, 32)
	if err != nil {
		return 0, false
	}
	return int(bus), true
}

// countHotpluggedDevices counts devices that were hotplugged after initial boot
// and consumed pcie-root-port slots. This includes hotplugged virtio volumes
// and hotplugged network interfaces.
func countHotpluggedDevices(vmi *v1.VirtualMachineInstance, client kubernetes.Interface) (int, error) {
	disks := countHotpluggedDisks(vmi)

	ifaces, err := countHotpluggedInterfaces(vmi, client)
	if err != nil {
		return 0, err
	}

	return disks + ifaces, nil
}

// countHotpluggedDisks counts virtio disk volumes that were hotplugged after
// initial boot and consumed pcie-root-port slots.
func countHotpluggedDisks(vmi *v1.VirtualMachineInstance) int {
	count := 0
	for _, vs := range vmi.Status.VolumeStatus {
		if vs.HotplugVolume == nil {
			continue
		}
		for _, disk := range vmi.Spec.Domain.Devices.Disks {
			if disk.Name == vs.Name && disk.DiskDevice.Disk != nil && disk.DiskDevice.Disk.Bus == v1.DiskBusVirtio {
				count++
				break
			}
		}
	}
	return count
}

// countHotpluggedInterfaces counts network interfaces that were hotplugged
// after initial boot by comparing current VMI interfaces against the boot-time
// VM spec stored in the ControllerRevision.
func countHotpluggedInterfaces(vmi *v1.VirtualMachineInstance, client kubernetes.Interface) (int, error) {
	bootInterfaces, err := getBootTimeInterfaces(vmi, client)
	if err != nil {
		return 0, fmt.Errorf("failed to get boot-time interfaces from ControllerRevision: %v", err)
	}
	if bootInterfaces == nil {
		return 0, nil
	}

	bootIfaceNames := make(map[string]bool, len(bootInterfaces))
	for _, iface := range bootInterfaces {
		bootIfaceNames[iface.Name] = true
	}

	count := 0
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if !bootIfaceNames[iface.Name] {
			count++
		}
	}
	return count, nil
}

// vmRevisionData mirrors the structure used by virt-controller to store
// the VM spec in a ControllerRevision. Defined locally to avoid importing
// from virt-controller.
type vmRevisionData struct {
	Spec v1.VirtualMachineSpec `json:"spec"`
}

// getBootTimeInterfaces retrieves the network interfaces from the VM spec
// snapshot stored in the ControllerRevision at boot time.
func getBootTimeInterfaces(vmi *v1.VirtualMachineInstance, client kubernetes.Interface) ([]v1.Interface, error) {
	revisionName := vmi.Status.VirtualMachineRevisionName
	if revisionName == "" {
		return nil, nil
	}

	// This API call is acceptable because detection runs at most once per VMI
	// during the upgrade window, not on every reconcile.
	revision, err := client.AppsV1().ControllerRevisions(vmi.Namespace).Get(
		context.Background(), revisionName, metav1.GetOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get ControllerRevision %s: %v", revisionName, err)
	}

	var revisionData vmRevisionData
	if err := json.Unmarshal(revision.Data.Raw, &revisionData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ControllerRevision data: %v", err)
	}

	return revisionData.Spec.Template.Spec.Domain.Devices.Interfaces, nil
}

// calculateHotplugPortCountV1ForDetection computes the v1 placeholder count
// from the VMI spec. This is the same formula as in
// pkg/virt-launcher/virtwrap/manager.go but duplicated here to avoid a
// circular dependency between virt-handler and virt-launcher packages.
// This formula is frozen (v1 is legacy) and must stay in sync with its counterpart.
func calculateHotplugPortCountV1ForDetection(vmi *v1.VirtualMachineInstance) int {
	if vmi.Annotations[v1.PlacePCIDevicesOnRootComplex] == "true" {
		return 0
	}
	interfaces := vmi.Spec.Domain.Devices.Interfaces
	if len(interfaces) == 0 {
		return 0
	}
	return max(0, 4-len(interfaces))
}
