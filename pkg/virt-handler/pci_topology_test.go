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
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func pciAddr(bus string) *api.Address {
	return &api.Address{Type: "pci", Bus: bus}
}

func rootPort(index int) api.Controller {
	return api.Controller{Type: "pci", Model: "pcie-root-port", Index: fmt.Sprintf("%d", index)}
}

func fakeRevision(name, namespace string, interfaces []v1.Interface) *appsv1.ControllerRevision {
	data := vmRevisionData{
		Spec: v1.VirtualMachineSpec{
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Interfaces: interfaces,
						},
					},
				},
			},
		},
	}
	raw, _ := json.Marshal(data)
	return &appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: runtime.RawExtension{Raw: raw},
	}
}

func fakeClientWithRevision(revision *appsv1.ControllerRevision) *k8sfake.Clientset {
	if revision == nil {
		return k8sfake.NewSimpleClientset()
	}
	return k8sfake.NewSimpleClientset(revision)
}

var _ = Describe("PCI Topology Detection", func() {

	Describe("parsePCIBus", func() {
		It("should return false for nil address", func() {
			_, ok := parsePCIBus(nil)
			Expect(ok).To(BeFalse())
		})

		It("should return false for non-PCI address", func() {
			_, ok := parsePCIBus(&api.Address{Type: "drive", Bus: "0x01"})
			Expect(ok).To(BeFalse())
		})

		It("should parse hex bus numbers", func() {
			bus, ok := parsePCIBus(pciAddr("0x00"))
			Expect(ok).To(BeTrue())
			Expect(bus).To(Equal(0))

			bus, ok = parsePCIBus(pciAddr("0x0e"))
			Expect(ok).To(BeTrue())
			Expect(bus).To(Equal(14))
		})
	})

	Describe("collectOccupiedBuses", func() {
		It("should collect buses from all device types", func() {
			domain := &api.Domain{}
			domain.Spec.Devices.Disks = []api.Disk{
				{Address: pciAddr("0x07")},
				{Address: pciAddr("0x00")}, // bus 0, not collected (not > 0)
				{Address: nil},             // no address
			}
			domain.Spec.Devices.Interfaces = []api.Interface{
				{Address: pciAddr("0x01")},
			}
			domain.Spec.Devices.Controllers = []api.Controller{
				{Type: "pci", Model: "pcie-root-port", Address: pciAddr("0x00")}, // PCI controller, skipped
				{Type: "scsi", Address: pciAddr("0x05")},
				{Type: "virtio-serial", Address: pciAddr("0x06")},
			}
			domain.Spec.Devices.Ballooning = &api.MemBalloon{Address: pciAddr("0x08")}

			buses := collectOccupiedBuses(domain)
			Expect(buses).To(Equal(map[int]bool{
				1: true, // interface
				5: true, // scsi controller
				6: true, // virtio-serial controller
				7: true, // disk
				8: true, // memballoon
			}))
		})
	})

	Describe("countHotpluggedDevices", func() {
		It("should count only hotplugged virtio volumes", func() {
			vmi := &v1.VirtualMachineInstance{}
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{Name: "vol1", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: v1.DiskBusVirtio}}},
				{Name: "vol2", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: v1.DiskBusSCSI}}},
				{Name: "vol3", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: v1.DiskBusVirtio}}},
			}
			vmi.Status.VolumeStatus = []v1.VolumeStatus{
				{Name: "vol1", HotplugVolume: &v1.HotplugVolumeStatus{}},
				{Name: "vol2", HotplugVolume: &v1.HotplugVolumeStatus{}}, // SCSI, not counted
				{Name: "vol3"}, // not hotplugged
			}
			client := fakeClientWithRevision(nil)
			Expect(countHotpluggedDevices(vmi, client)).To(Equal(1))
		})

		It("should count hotplugged interfaces using ControllerRevision", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
			}
			vmi.Status.VirtualMachineRevisionName = "test-revision"
			// Boot-time had only "default" interface
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
				{Name: "default"},
				{Name: "hotplugged1"},
				{Name: "hotplugged2"},
			}
			revision := fakeRevision("test-revision", "default", []v1.Interface{
				{Name: "default"},
			})
			client := fakeClientWithRevision(revision)
			Expect(countHotpluggedDevices(vmi, client)).To(Equal(2))
		})

		It("should return 0 hotplugged interfaces when no revision exists", func() {
			vmi := &v1.VirtualMachineInstance{}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
				{Name: "default"},
				{Name: "secondary"},
			}
			client := fakeClientWithRevision(nil)
			Expect(countHotpluggedDevices(vmi, client)).To(Equal(0))
		})

		It("should return error when ControllerRevision lookup fails", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
			}
			vmi.Status.VirtualMachineRevisionName = "nonexistent-revision"
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
				{Name: "default"},
				{Name: "hotplugged1"},
			}
			client := fakeClientWithRevision(nil)
			_, err := countHotpluggedDevices(vmi, client)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("detectPlaceholderCount", func() {
		It("should exclude trailing spare root ports added by libvirt", func() {
			// Simulates a v1 domain: 9 root ports, devices on buses 1,5,6,7,8
			// Buses 2,3,4 are empty (former placeholders), bus 9 is empty (spare)
			vmi := &v1.VirtualMachineInstance{}
			domain := &api.Domain{}
			for i := 1; i <= 9; i++ {
				domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, rootPort(i))
			}
			domain.Spec.Devices.Interfaces = []api.Interface{
				{Address: pciAddr("0x01")},
			}
			domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers,
				api.Controller{Type: "scsi", Address: pciAddr("0x05")},
				api.Controller{Type: "virtio-serial", Address: pciAddr("0x06")},
			)
			domain.Spec.Devices.Disks = []api.Disk{
				{Address: pciAddr("0x07")},
			}
			domain.Spec.Devices.Ballooning = &api.MemBalloon{Address: pciAddr("0x08")}
			// Bus 9 is empty (spare) — should not be counted
			Expect(detectPlaceholderCount(vmi, domain, fakeClientWithRevision(nil))).To(Equal(3))
		})

		It("should add back hotplugged devices", func() {
			vmi := &v1.VirtualMachineInstance{}
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{Name: "hp1", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: v1.DiskBusVirtio}}},
			}
			vmi.Status.VolumeStatus = []v1.VolumeStatus{
				{Name: "hp1", HotplugVolume: &v1.HotplugVolumeStatus{}},
			}

			domain := &api.Domain{}
			// 10 root ports, devices on buses 1-5, bus 10 is spare
			for i := 1; i <= 10; i++ {
				domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, rootPort(i))
			}
			for i := 1; i <= 5; i++ {
				domain.Spec.Devices.Disks = append(domain.Spec.Devices.Disks, api.Disk{
					Address: pciAddr(fmt.Sprintf("0x%02x", i)),
				})
			}
			// Buses 6-9 empty below highest (5), bus 10 is spare
			// empty below highest = 0, hotplugged = 1, total = 0 + 1 = 1
			// Wait: highest occupied is 5, all root ports 6-10 are above → all spares
			// So empty below highest = 0, + hotplugged = 1
			Expect(detectPlaceholderCount(vmi, domain, fakeClientWithRevision(nil))).To(Equal(1))
		})

		It("should handle v2 topology with interleaved empty ports", func() {
			// Simulates v2 domain: 16 root ports, devices on buses 1,12,13,14,15
			// Buses 2-11 are empty (former placeholders), bus 16 is spare
			vmi := &v1.VirtualMachineInstance{}
			domain := &api.Domain{}
			for i := 1; i <= 16; i++ {
				domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, rootPort(i))
			}
			domain.Spec.Devices.Interfaces = []api.Interface{
				{Address: pciAddr("0x01")},
			}
			domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers,
				api.Controller{Type: "scsi", Address: pciAddr("0x0c")},
				api.Controller{Type: "virtio-serial", Address: pciAddr("0x0d")},
			)
			domain.Spec.Devices.Disks = []api.Disk{
				{Address: pciAddr("0x0e")},
			}
			domain.Spec.Devices.Ballooning = &api.MemBalloon{Address: pciAddr("0x0f")}
			// Highest occupied = 15 (0x0f). Empty ports at or below 15: buses 2-11 = 10
			// Bus 16 is spare (above highest)
			Expect(detectPlaceholderCount(vmi, domain, fakeClientWithRevision(nil))).To(Equal(10))
		})
	})

	Describe("calculateHotplugPortCountV1ForDetection", func() {
		It("should return 0 when PlacePCIDevicesOnRootComplex is true", func() {
			vmi := &v1.VirtualMachineInstance{}
			vmi.Annotations = map[string]string{v1.PlacePCIDevicesOnRootComplex: "true"}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default"}}
			Expect(calculateHotplugPortCountV1ForDetection(vmi)).To(Equal(0))
		})

		It("should return 0 when there are no interfaces", func() {
			vmi := &v1.VirtualMachineInstance{}
			Expect(calculateHotplugPortCountV1ForDetection(vmi)).To(Equal(0))
		})

		DescribeTable("should return correct count based on interface count",
			func(ifaceCount, expected int) {
				vmi := &v1.VirtualMachineInstance{}
				for i := 0; i < ifaceCount; i++ {
					vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces,
						v1.Interface{Name: fmt.Sprintf("iface%d", i)})
				}
				Expect(calculateHotplugPortCountV1ForDetection(vmi)).To(Equal(expected))
			},
			Entry("1 interface", 1, 3),
			Entry("2 interfaces", 2, 2),
			Entry("3 interfaces", 3, 1),
			Entry("4 interfaces", 4, 0),
			Entry("5 interfaces", 5, 0),
		)
	})

	Describe("detectPCITopologyAndAnnotate", func() {
		var ctrl *gomock.Controller
		var virtClient *kubecli.MockKubevirtClient
		var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
		var c *VirtualMachineController

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
			virtClient.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface).AnyTimes()
			c = &VirtualMachineController{}
			c.clientset = virtClient
		})

		It("should skip when domain is nil", func() {
			vmi := &v1.VirtualMachineInstance{}
			vmi.Status.Phase = v1.Running
			Expect(c.detectPCITopologyAndAnnotate(vmi, nil)).To(Succeed())
		})

		It("should skip when VMI is not running", func() {
			vmi := &v1.VirtualMachineInstance{}
			vmi.Status.Phase = v1.Scheduled
			domain := &api.Domain{}
			Expect(c.detectPCITopologyAndAnnotate(vmi, domain)).To(Succeed())
		})

		It("should skip when annotation already exists", func() {
			vmi := &v1.VirtualMachineInstance{}
			vmi.Status.Phase = v1.Running
			vmi.Annotations = map[string]string{v1.PciTopologyVersionAnnotation: v1.PciTopologyVersionV3}
			domain := &api.Domain{}
			Expect(c.detectPCITopologyAndAnnotate(vmi, domain)).To(Succeed())
		})

		It("should annotate as v3 when detected count matches v1 expected", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi",
					Namespace: "default",
					Annotations: map[string]string{
						"existing": "annotation",
					},
				},
			}
			vmi.Status.Phase = v1.Running
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default"}}

			// v1 expects 3 placeholders for 1 interface
			// 8 root ports + 1 spare = 9 total
			// Devices on buses 1,5,6,7,8. Empty at or below 8: buses 2,3,4 = 3. Matches v1.
			domain := &api.Domain{}
			for i := 1; i <= 9; i++ {
				domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, rootPort(i))
			}
			domain.Spec.Devices.Interfaces = []api.Interface{
				{Address: pciAddr("0x01")},
			}
			domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers,
				api.Controller{Type: "scsi", Address: pciAddr("0x05")},
				api.Controller{Type: "virtio-serial", Address: pciAddr("0x06")},
			)
			domain.Spec.Devices.Disks = []api.Disk{
				{Address: pciAddr("0x07")},
			}
			domain.Spec.Devices.Ballooning = &api.MemBalloon{Address: pciAddr("0x08")}

			vmiInterface.EXPECT().Patch(
				gomock.Any(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any(),
			).DoAndReturn(func(_ interface{}, _ string, _ types.PatchType, patchData []byte, _ metav1.PatchOptions, _ ...string) (*v1.VirtualMachineInstance, error) {
				var ops []patch.PatchOperation
				Expect(json.Unmarshal(patchData, &ops)).To(Succeed())
				Expect(ops).To(HaveLen(1))
				Expect(ops[0].Op).To(Equal("add"))
				Expect(ops[0].Value).To(Equal(v1.PciTopologyVersionV3))
				return vmi, nil
			})

			Expect(c.detectPCITopologyAndAnnotate(vmi, domain)).To(Succeed())
		})

		It("should annotate as v2 with frozen slot total when detected count differs from v1", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi",
					Namespace: "default",
					Annotations: map[string]string{
						"existing": "annotation",
					},
				},
			}
			vmi.Status.Phase = v1.Running
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default"}}

			// v1 expects 3 placeholders for 1 interface
			// 16 root ports + 1 spare = 17 total
			// Devices on buses 1,12,13,14,15. Empty at or below 15: buses 2-11 = 10. 10 != 3 → v2.
			// Slot total = 10 placeholders + 1 boot interface = 11.
			domain := &api.Domain{}
			for i := 1; i <= 17; i++ {
				domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, rootPort(i))
			}
			domain.Spec.Devices.Interfaces = []api.Interface{
				{Address: pciAddr("0x01")},
			}
			domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers,
				api.Controller{Type: "scsi", Address: pciAddr("0x0c")},
				api.Controller{Type: "virtio-serial", Address: pciAddr("0x0d")},
			)
			domain.Spec.Devices.Disks = []api.Disk{
				{Address: pciAddr("0x0e")},
			}
			domain.Spec.Devices.Ballooning = &api.MemBalloon{Address: pciAddr("0x0f")}

			vmiInterface.EXPECT().Patch(
				gomock.Any(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any(),
			).DoAndReturn(func(_ interface{}, _ string, _ types.PatchType, patchData []byte, _ metav1.PatchOptions, _ ...string) (*v1.VirtualMachineInstance, error) {
				var ops []patch.PatchOperation
				Expect(json.Unmarshal(patchData, &ops)).To(Succeed())
				Expect(ops).To(HaveLen(2))
				Expect(ops[0].Op).To(Equal("add"))
				Expect(ops[0].Value).To(Equal(v1.PciTopologyVersionV2))
				Expect(ops[1].Op).To(Equal("add"))
				Expect(ops[1].Value).To(Equal("11"))
				return vmi, nil
			})

			Expect(c.detectPCITopologyAndAnnotate(vmi, domain)).To(Succeed())
		})
	})
})
