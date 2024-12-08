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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package admitters

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	rt "runtime"
	"strings"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/client-go/api"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	v1 "kubevirt.io/api/core/v1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype"
	"kubevirt.io/kubevirt/pkg/liveupdate/memory"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/deprecation"
	"kubevirt.io/kubevirt/tests/framework/checks"
)

var _ = Describe("Validating VM Admitter", func() {
	config, crdInformer, kvStore := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
	var (
		vmsAdmitter         *VMsAdmitter
		dataSourceInformer  cache.SharedIndexInformer
		namespaceInformer   cache.SharedIndexInformer
		instancetypeMethods *testutils.MockInstancetypeMethods
		mockVMIClient       *kubecli.MockVirtualMachineInstanceInterface
		virtClient          *kubecli.MockKubevirtClient
		k8sClient           *k8sfake.Clientset
	)

	enableFeatureGate := func(featureGates ...string) {
		kv := testutils.GetFakeKubeVirtClusterConfig(kvStore)
		if kv.Spec.Configuration.DeveloperConfiguration == nil {
			kv.Spec.Configuration.DeveloperConfiguration = &v1.DeveloperConfiguration{}
		}
		if kv.Spec.Configuration.DeveloperConfiguration.FeatureGates == nil {
			kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = featureGates
		} else {
			kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = append(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates, featureGates...)
		}
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
	}
	disableFeatureGates := func() {
		kv := testutils.GetFakeKubeVirtClusterConfig(kvStore)
		if kv.Spec.Configuration.DeveloperConfiguration != nil {
			kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = make([]string, 0)
		}
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
	}
	enableLiveUpdate := func() {
		kv := testutils.GetFakeKubeVirtClusterConfig(kvStore)
		kv.Spec.Configuration.VMRolloutStrategy = pointer.P(v1.VMRolloutStrategyLiveUpdate)
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
	}

	runStrategyManual := v1.RunStrategyManual
	runStrategyHalted := v1.RunStrategyHalted

	BeforeEach(func() {
		dataSourceInformer, _ = testutils.NewFakeInformerFor(&cdiv1.DataSource{})
		namespaceInformer, _ = testutils.NewFakeInformerFor(&k8sv1.Namespace{})
		ns1 := &k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns1",
			},
		}
		ns2 := &k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns2",
			},
		}
		ns3 := &k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns3",
			},
		}
		Expect(namespaceInformer.GetStore().Add(ns1)).To(Succeed())
		Expect(namespaceInformer.GetStore().Add(ns2)).To(Succeed())
		Expect(namespaceInformer.GetStore().Add(ns3)).To(Succeed())
		instancetypeMethods = testutils.NewMockInstancetypeMethods()

		ctrl := gomock.NewController(GinkgoT())
		mockVMIClient = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		k8sClient = k8sfake.NewSimpleClientset()
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmsAdmitter = &VMsAdmitter{
			VirtClient:          virtClient,
			DataSourceInformer:  dataSourceInformer,
			NamespaceInformer:   namespaceInformer,
			ClusterConfig:       config,
			InstancetypeMethods: instancetypeMethods,
		}
		virtClient.EXPECT().AuthorizationV1().Return(k8sClient.AuthorizationV1()).AnyTimes()
	})

	Context("with an invalid VM", func() {
		It("should reject the request with unrecognized field", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vm := &v1.VirtualMachine{
				Spec: v1.VirtualMachineSpec{
					Running: pointer.P(false),
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						Spec: vmi.Spec,
					},
				},
			}
			jsonBytes, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())

			// change the name of a required field (like domain) so validation will fail
			jsonString := strings.Replace(string(jsonBytes), "domain", "not-a-domain", -1)

			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Resource: webhooks.VirtualMachineGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: []byte(jsonString),
					},
				},
			}

			resp := vmsAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(2))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("spec.template.spec.not-a-domain in body is a forbidden property"))
			Expect(resp.Result.Details.Causes[1].Message).To(Equal("spec.template.spec.domain in body is required"))
			Expect(resp.Result.Message).To(Equal("spec.template.spec.not-a-domain in body is a forbidden property, spec.template.spec.domain in body is required"))
		})

		It("reject syntax valid VM, but with invalid spec", func() {
			vmi := api.NewMinimalVMI("testvmi")
			// Add a disk that doesn't map to a volume.
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})
			vm := &v1.VirtualMachine{
				Spec: v1.VirtualMachineSpec{
					Running: pointer.P(false),
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						Spec: vmi.Spec,
					},
				},
			}

			resp := admitVm(vmsAdmitter, vm)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.template.spec.domain.devices.disks[0].name"))
		})
	})

	It("should allow VM that is being deleted", func() {
		vmi := api.NewMinimalVMI("testvmi")
		now := metav1.Now()
		vm := &v1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				DeletionTimestamp: &now,
			},
			Spec: v1.VirtualMachineSpec{
				Running: pointer.P(false),
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: vmi.Spec,
				},
			},
		}
		resp := admitVm(vmsAdmitter, vm)
		Expect(resp.Allowed).To(BeTrue())
	})

	It("should allow VM with missing volume disk or filesystem", func() {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "testvol",
			VolumeSource: v1.VolumeSource{
				ContainerDisk: testutils.NewFakeContainerDiskSource(),
			},
		})
		vm := &v1.VirtualMachine{
			Spec: v1.VirtualMachineSpec{
				Running: pointer.P(false),
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: vmi.Spec,
				},
			},
		}
		resp := admitVm(vmsAdmitter, vm)
		Expect(resp.Allowed).To(BeTrue())
	})

	It("should accept valid vmi spec", func() {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "testdisk",
		})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "testdisk",
			VolumeSource: v1.VolumeSource{
				ContainerDisk: testutils.NewFakeContainerDiskSource(),
			},
		})

		vm := &v1.VirtualMachine{
			Spec: v1.VirtualMachineSpec{
				Running: pointer.P(false),
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: vmi.Spec,
				},
			},
		}

		resp := admitVm(vmsAdmitter, vm)
		Expect(resp.Allowed).To(BeTrue())
	})

	It("should accept VM requesting hugepages but missing spec.template.spec.domain.resources.requests.memory - bug #9102", func() {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Resources = v1.ResourceRequirements{}
		guestMemory := resource.MustParse("1Gi")
		vmi.Spec.Domain.Memory = &v1.Memory{
			Guest: &guestMemory,
			Hugepages: &v1.Hugepages{
				PageSize: "2Mi",
			},
		}
		vm := &v1.VirtualMachine{
			Spec: v1.VirtualMachineSpec{
				Running: pointer.P(false),
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: vmi.Spec,
				},
			},
		}

		resp := admitVm(vmsAdmitter, vm)
		Expect(resp.Allowed).To(BeTrue())
	})

	It("should accept VM requesting hugepages but missing spec.template.spec.domain.memory.guest", func() {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Memory = &v1.Memory{
			Hugepages: &v1.Hugepages{
				PageSize: "2Mi",
			},
		}
		vmi.Spec.Domain.Resources = v1.ResourceRequirements{
			Requests: k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("1Gi"),
			},
		}

		vm := &v1.VirtualMachine{
			Spec: v1.VirtualMachineSpec{
				Running: pointer.P(false),
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: vmi.Spec,
				},
			},
		}

		resp := admitVm(vmsAdmitter, vm)
		Expect(resp.Allowed).To(BeTrue())
	})

	DescribeTable("should reject VolumeRequests on a migrating vm", func(requests []v1.VirtualMachineVolumeRequest) {
		now := metav1.Now()
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Status = v1.VirtualMachineInstanceStatus{
			MigrationState: &v1.VirtualMachineInstanceMigrationState{
				StartTimestamp: &now,
			},
		}
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "testdisk",
		})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "testdisk",
			VolumeSource: v1.VolumeSource{
				ContainerDisk: testutils.NewFakeContainerDiskSource(),
			},
		})

		vm := &v1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vmi.Name,
				Namespace: vmi.Namespace,
			},
			Spec: v1.VirtualMachineSpec{
				Running: pointer.P(false),
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: *vmi.Spec.DeepCopy(),
				},
			},
			Status: v1.VirtualMachineStatus{
				VolumeRequests: requests,
				Ready:          true,
			},
		}
		vmBytes, _ := json.Marshal(&vm)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: webhooks.VirtualMachineGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: vmBytes,
				},
			},
		}

		virtClient.EXPECT().VirtualMachineInstance(gomock.Any()).Return(mockVMIClient)
		mockVMIClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(vmi, nil)
		resp := vmsAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeFalse())
	},
		Entry("with valid request to add volume", []v1.VirtualMachineVolumeRequest{
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "testdisk2",
					Disk: &v1.Disk{
						Name: "testdisk2",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: "scsi",
							},
						},
					},
					VolumeSource: &v1.HotplugVolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "madeup",
						}},
					},
				},
			},
		}),
		Entry("with valid request to remove volume", []v1.VirtualMachineVolumeRequest{
			{
				RemoveVolumeOptions: &v1.RemoveVolumeOptions{
					Name: "testdisk",
				},
			},
		}),
	)

	DescribeTable("should validate VolumeRequest on running vm", func(requests []v1.VirtualMachineVolumeRequest, isValid bool) {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "testdisk",
		})
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "testpvcdisk",
		})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "testdisk",
			VolumeSource: v1.VolumeSource{
				ContainerDisk: testutils.NewFakeContainerDiskSource(),
			},
		})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "testpvcdisk",
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: "testpvcdiskclaim",
				}},
			},
		})

		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "a-pvcdisk",
		})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "a-pvcdisk",
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: "a-pvcdiskclaim",
				}},
			},
		})

		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "t-pvcdisk",
		})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "t-pvcdisk",
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: "t-pvcdiskclaim",
				}},
			},
		})

		vm := &v1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vmi.Name,
				Namespace: vmi.Namespace,
			},
			Spec: v1.VirtualMachineSpec{
				Running: pointer.P(false),
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: *vmi.Spec.DeepCopy(),
				},
			},
			Status: v1.VirtualMachineStatus{
				VolumeRequests: requests,
				Ready:          true,
			},
		}

		// add some additional volumes to the running VMI so we can simulate
		// more advanced validation scenarios where VM and VMI specs drift.
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "testpvcdisk-extra",
		})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "testpvcdisk-extra",
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: "testpvcdiskclaim-extra",
				},
				},
			},
		})

		virtClient.EXPECT().VirtualMachineInstance(gomock.Any()).Return(mockVMIClient)
		mockVMIClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(vmi, nil)
		resp := admitVm(vmsAdmitter, vm)
		Expect(resp.Allowed).To(Equal(isValid))
	},
		Entry("with valid request to add volume", []v1.VirtualMachineVolumeRequest{
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "testdisk2",
					Disk: &v1.Disk{
						Name: "testdisk2",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: "scsi",
							},
						},
					},
					VolumeSource: &v1.HotplugVolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "madeup",
						},
						},
					},
				},
			},
		},
			true),
		Entry("with valid request to add volume to a LUN disk", []v1.VirtualMachineVolumeRequest{
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "testlun2",
					Disk: &v1.Disk{
						Name: "testlun2",
						DiskDevice: v1.DiskDevice{
							LUN: &v1.LunTarget{
								Bus: "scsi",
							},
						},
					},
					VolumeSource: &v1.HotplugVolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "madeupLUN",
						},
						},
					},
				},
			},
		},
			true),
		Entry("with invalid request to add volume with invalid disk/bus combination", []v1.VirtualMachineVolumeRequest{
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "testLUN-usb",
					Disk: &v1.Disk{
						Name: "testLUN-usb",
						DiskDevice: v1.DiskDevice{
							LUN: &v1.LunTarget{
								Bus: "usb",
							},
						},
					},
					VolumeSource: &v1.HotplugVolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "invalidCombination",
						}},
					},
				},
			},
		},
			false),
		Entry("with invalid request to add volume with invalid disk type", []v1.VirtualMachineVolumeRequest{
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "testCDRom",
					Disk: &v1.Disk{
						Name: "testCDRom",
						DiskDevice: v1.DiskDevice{
							CDRom: &v1.CDRomTarget{
								Bus: "scsi",
							},
						},
					},
					VolumeSource: &v1.HotplugVolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "cdRomtest",
						}},
					},
				},
			},
		},
			false),
		Entry("with invalid request to add volume with dedicated IOThreads", []v1.VirtualMachineVolumeRequest{
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "testDisk",
					Disk: &v1.Disk{
						Name: "testDisk",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: "scsi",
							},
						},
						DedicatedIOThread: pointer.P(true),
					},
					VolumeSource: &v1.HotplugVolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "diskTest",
						}},
					},
				},
			},
		},
			false),
		Entry("with invalid request to add volume that conflicts with running vmi", []v1.VirtualMachineVolumeRequest{
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "testpvcdisk-extra",
					Disk: &v1.Disk{
						Name: "testpvcdisk-extra",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: "scsi",
							},
						},
					},
					VolumeSource: &v1.HotplugVolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "NOT-IDENTICAL-TO-WHAT-IS-IN-VMI",
						}},
					},
				},
			},
		},
			false),
		Entry("with valid request to add volume that is identical to one in vmi", []v1.VirtualMachineVolumeRequest{
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "a-pvcdisk",
					Disk: &v1.Disk{
						Name: "a-pvcdisk",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: "scsi",
							},
						},
					},
					VolumeSource: &v1.HotplugVolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "a-pvcdiskclaim",
						}},
					},
				},
			},
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "testpvcdisk-extra1",
					Disk: &v1.Disk{
						Name: "testpvcdisk-extra1",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: "scsi",
							},
						},
					},
					VolumeSource: &v1.HotplugVolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testpvcdiskclaim-extra1",
						}},
					},
				},
			},
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "testpvcdisk-extra",
					Disk: &v1.Disk{
						Name: "testpvcdisk-extra",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: "scsi",
							},
						},
					},
					VolumeSource: &v1.HotplugVolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testpvcdiskclaim-extra",
						}},
					},
				},
			},
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "testpvcdisk-extra2",
					Disk: &v1.Disk{
						Name: "testpvcdisk-extra2",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: "scsi",
							},
						},
					},
					VolumeSource: &v1.HotplugVolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testpvcdiskclaim-extra2",
						}},
					},
				},
			},
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "t-pvcdisk",
					Disk: &v1.Disk{
						Name: "t-pvcdisk",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: "scsi",
							},
						},
					},
					VolumeSource: &v1.HotplugVolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "t-pvcdiskclaim",
						}},
					},
				},
			},
		},
			true),
	)

	DescribeTable("should validate VolumeRequest on offline vm", func(requests []v1.VirtualMachineVolumeRequest, isValid bool) {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "testdisk",
		})
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "testpvcdisk",
		})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "testdisk",
			VolumeSource: v1.VolumeSource{
				ContainerDisk: testutils.NewFakeContainerDiskSource(),
			},
		})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "testpvcdisk",
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: "testpvcdiskclaim",
				}},
			},
		})

		vm := &v1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vmi.Name,
				Namespace: vmi.Namespace,
			},
			Spec: v1.VirtualMachineSpec{
				Running: pointer.P(false),
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: vmi.Spec,
				},
			},
			Status: v1.VirtualMachineStatus{
				VolumeRequests: requests,
			},
		}

		resp := admitVm(vmsAdmitter, vm)
		Expect(resp.Allowed).To(Equal(isValid))
	},
		Entry("with valid request to add volume", []v1.VirtualMachineVolumeRequest{
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "testdisk2",
					Disk: &v1.Disk{
						Name: "testdisk2",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: "scsi",
							},
						},
					},
					VolumeSource: &v1.HotplugVolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "madeup",
						}},
					},
				},
			},
		},
			true),

		Entry("with invalid request to add the same volume twice", []v1.VirtualMachineVolumeRequest{
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "testdisk2",
					Disk: &v1.Disk{
						Name: "testdisk2",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: "scsi",
							},
						},
					},
					VolumeSource: &v1.HotplugVolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "madeup",
						}},
					},
				},
			},
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "testdisk2",
					Disk: &v1.Disk{
						Name: "testdisk2",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: "scsi",
							},
						},
					},
					VolumeSource: &v1.HotplugVolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "madeup",
						}},
					},
				},
			},
		},
			false),
		Entry("with invalid request to add volume that already exists", []v1.VirtualMachineVolumeRequest{
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "testdisk",
					Disk: &v1.Disk{
						Name: "testdisk",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: "scsi",
							},
						},
					},
					VolumeSource: &v1.HotplugVolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "madeup",
						}},
					},
				},
			},
		},
			false),

		Entry("with valid request to add the exact same volume that already exists.", []v1.VirtualMachineVolumeRequest{
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "testdisk",
					Disk: &v1.Disk{
						Name: "testpvcdisk",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: "scsi",
							},
						},
					},
					VolumeSource: &v1.HotplugVolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testpvcdiskclaim",
						}},
					},
				},
			},
		},
			false),
		Entry("with valid request to remove volume", []v1.VirtualMachineVolumeRequest{
			{
				RemoveVolumeOptions: &v1.RemoveVolumeOptions{
					Name: "testdisk",
				},
			},
		},
			true),
		Entry("with invalid request to remove same volume twice", []v1.VirtualMachineVolumeRequest{
			{
				RemoveVolumeOptions: &v1.RemoveVolumeOptions{
					Name: "testdisk",
				},
			},
			{
				RemoveVolumeOptions: &v1.RemoveVolumeOptions{
					Name: "testdisk",
				},
			},
		},
			false),
		Entry("with invalid request with no options", []v1.VirtualMachineVolumeRequest{
			{},
		},
			false),
		Entry("with invalid request with multiple options", []v1.VirtualMachineVolumeRequest{
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "testdisk2",
					Disk: &v1.Disk{
						Name: "testdisk2",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: "scsi",
							},
						},
					},
					VolumeSource: &v1.HotplugVolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "madeup",
						}},
					},
				},
				RemoveVolumeOptions: &v1.RemoveVolumeOptions{
					Name: "testdisk",
				},
			},
		},
			false),
	)

	Context("validateDataVolumeTemplate", func() {
		var vm *v1.VirtualMachine
		apiGroup := "kubevirt.io"

		BeforeEach(func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testdisk",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})

			vm = &v1.VirtualMachine{
				Spec: v1.VirtualMachineSpec{
					Running: pointer.P(false),
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						Spec: vmi.Spec,
					},
				},
			}
		})

		It("should accept valid DataVolumeTemplate", func() {
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
				Spec: cdiv1.DataVolumeSpec{
					PVC: &k8sv1.PersistentVolumeClaimSpec{},
					Source: &cdiv1.DataVolumeSource{
						Blank: &cdiv1.DataVolumeBlankImage{},
					},
				},
			})

			testutils.AddDataVolumeAPI(crdInformer)
			resp := admitVm(vmsAdmitter, vm)
			Expect(resp.Result).To(BeNil())
			Expect(resp.Allowed).To(BeTrue())
		})
		It("should accept DataVolumeTemplate with deleted sourceRef if vm is going to be deleted", func() {
			now := metav1.Now()
			vm.DeletionTimestamp = &now

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
				Spec: cdiv1.DataVolumeSpec{
					PVC: &k8sv1.PersistentVolumeClaimSpec{},
					SourceRef: &cdiv1.DataVolumeSourceRef{
						Kind: "DataSource",
						Name: "fakeName",
					},
				},
			})

			testutils.AddDataVolumeAPI(crdInformer)
			resp := admitVm(vmsAdmitter, vm)
			Expect(resp.Allowed).To(BeTrue())
		})
		It("should reject invalid DataVolumeTemplate with no dataVolume name", func() {
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: cdiv1.DataVolumeSpec{
					PVC: &k8sv1.PersistentVolumeClaimSpec{},
					Source: &cdiv1.DataVolumeSource{
						Blank: &cdiv1.DataVolumeBlankImage{},
					},
				},
			})

			testutils.AddDataVolumeAPI(crdInformer)
			resp := admitVm(vmsAdmitter, vm)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("'name' field must not be empty for DataVolumeTemplate entry spec.dataVolumeTemplate[0].name."))
		})
		It("should reject invalid DataVolumeTemplate with no PVC nor Storage", func() {
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
				Spec: cdiv1.DataVolumeSpec{
					Source: &cdiv1.DataVolumeSource{
						Blank: &cdiv1.DataVolumeBlankImage{},
					},
				},
			})

			testutils.AddDataVolumeAPI(crdInformer)
			resp := admitVm(vmsAdmitter, vm)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("Missing Data volume PVC or Storage"))
		})
		It("should reject invalid DataVolumeTemplate with both PVC and Storage", func() {
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
				Spec: cdiv1.DataVolumeSpec{
					PVC:     &k8sv1.PersistentVolumeClaimSpec{},
					Storage: &cdiv1.StorageSpec{},
					Source: &cdiv1.DataVolumeSource{
						Blank: &cdiv1.DataVolumeBlankImage{},
					},
				},
			})

			testutils.AddDataVolumeAPI(crdInformer)
			resp := admitVm(vmsAdmitter, vm)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("Duplicate storage definition, both target storage and target pvc defined"))
		})
		It("should reject invalid DataVolumeTemplate with both datasource and Source", func() {
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
				Spec: cdiv1.DataVolumeSpec{
					PVC: &k8sv1.PersistentVolumeClaimSpec{
						DataSource: &k8sv1.TypedLocalObjectReference{
							APIGroup: &apiGroup,
						},
					},
					Source: &cdiv1.DataVolumeSource{
						Blank: &cdiv1.DataVolumeBlankImage{},
					},
				},
			})

			testutils.AddDataVolumeAPI(crdInformer)
			resp := admitVm(vmsAdmitter, vm)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("External population is incompatible with Source and SourceRef"))
		})
		It("should reject invalid DataVolumeTemplate with no datasource, source or sourceref", func() {
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
				Spec: cdiv1.DataVolumeSpec{
					PVC: &k8sv1.PersistentVolumeClaimSpec{},
				},
			})

			testutils.AddDataVolumeAPI(crdInformer)
			resp := admitVm(vmsAdmitter, vm)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("Data volume should have either Source, SourceRef, or be externally populated"))
		})
		It("should reject invalid DataVolumeTemplate with no valid Source", func() {
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
				Spec: cdiv1.DataVolumeSpec{
					PVC:    &k8sv1.PersistentVolumeClaimSpec{},
					Source: &cdiv1.DataVolumeSource{},
				},
			})

			testutils.AddDataVolumeAPI(crdInformer)
			resp := admitVm(vmsAdmitter, vm)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("Missing dataVolume valid source"))
		})
		It("should reject invalid DataVolumeTemplate with multiple Sources", func() {
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
				Spec: cdiv1.DataVolumeSpec{
					PVC: &k8sv1.PersistentVolumeClaimSpec{},
					Source: &cdiv1.DataVolumeSource{
						Blank: &cdiv1.DataVolumeBlankImage{},
						HTTP:  &cdiv1.DataVolumeSourceHTTP{},
					},
				},
			})

			testutils.AddDataVolumeAPI(crdInformer)
			resp := admitVm(vmsAdmitter, vm)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("Multiple dataVolume sources"))
		})
		It("should reject invalid DataVolumeTemplate with no Volume reference in VMI template", func() {
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: "testdisk",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "WRONG-DATAVOLUME",
					},
				},
			}}

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
				Spec: cdiv1.DataVolumeSpec{
					PVC: &k8sv1.PersistentVolumeClaimSpec{},
					Source: &cdiv1.DataVolumeSource{
						Blank: &cdiv1.DataVolumeBlankImage{},
					},
				},
			})

			testutils.AddDataVolumeAPI(crdInformer)
			resp := admitVm(vmsAdmitter, vm)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.dataVolumeTemplate[0]"))
		})
	})

	It("should reject VM with DataVolumeTemplate in another namespace", func() {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "testdisk",
		})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "testdisk",
			VolumeSource: v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name: "dv1",
				},
			},
		})

		vm := &v1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "vm-namespace",
			},
			Spec: v1.VirtualMachineSpec{
				Running: pointer.P(false),
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: vmi.Spec,
				},
			},
		}

		vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dv1",
				Namespace: "another-namespace",
			},
			Spec: cdiv1.DataVolumeSpec{
				PVC: &k8sv1.PersistentVolumeClaimSpec{},
				Source: &cdiv1.DataVolumeSource{
					Blank: &cdiv1.DataVolumeBlankImage{},
				},
			},
		})

		testutils.AddDataVolumeAPI(crdInformer)
		resp := admitVm(vmsAdmitter, vm)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Details.Causes[0].Message).To(Equal("Embedded DataVolume namespace another-namespace differs from VM namespace vm-namespace"))
	})

	Context("with Volume", func() {

		BeforeEach(func() {
			enableFeatureGate(virtconfig.HostDiskGate)
		})

		AfterEach(func() {
			disableFeatureGates()
		})

		DescribeTable("should accept valid volumes",
			func(volumeSource v1.VolumeSource) {
				vmi := api.NewMinimalVMI("testvmi")
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name:         "testvolume",
					VolumeSource: volumeSource,
				})

				testutils.AddDataVolumeAPI(crdInformer)
				causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
				Expect(causes).To(BeEmpty())
			},
			Entry("with pvc volume source", v1.VolumeSource{PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{}}),
			Entry("with cloud-init volume source", v1.VolumeSource{CloudInitNoCloud: &v1.CloudInitNoCloudSource{UserData: "fake", NetworkData: "fake"}}),
			Entry("with containerDisk volume source", v1.VolumeSource{ContainerDisk: testutils.NewFakeContainerDiskSource()}),
			Entry("with ephemeral volume source", v1.VolumeSource{Ephemeral: &v1.EphemeralVolumeSource{}}),
			Entry("with emptyDisk volume source", v1.VolumeSource{EmptyDisk: &v1.EmptyDiskSource{}}),
			Entry("with dataVolume volume source", v1.VolumeSource{DataVolume: &v1.DataVolumeSource{Name: "fake"}}),
			Entry("with hostDisk volume source", v1.VolumeSource{HostDisk: &v1.HostDisk{Path: "fake", Type: v1.HostDiskExistsOrCreate}}),
			Entry("with configMap volume source", v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{LocalObjectReference: k8sv1.LocalObjectReference{Name: "fake"}}}),
			Entry("with secret volume source", v1.VolumeSource{Secret: &v1.SecretVolumeSource{SecretName: "fake"}}),
			Entry("with serviceAccount volume source", v1.VolumeSource{ServiceAccount: &v1.ServiceAccountVolumeSource{ServiceAccountName: "fake"}}),
		)
		It("should allow create a vm using a DataVolume when cdi doesnt exist", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name:         "testvolume",
				VolumeSource: v1.VolumeSource{DataVolume: &v1.DataVolumeSource{Name: "fake"}},
			})

			testutils.RemoveDataVolumeAPI(crdInformer)
			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(BeEmpty())
		})
		It("should reject DataVolume when DataVolume name is not set", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name:         "testvolume",
				VolumeSource: v1.VolumeSource{DataVolume: &v1.DataVolumeSource{Name: ""}},
			})

			testutils.AddDataVolumeAPI(crdInformer)
			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueRequired"))
			Expect(causes[0].Field).To(Equal("fake[0].name"))
			Expect(causes[0].Message).To(Equal("DataVolume 'name' must be set"))
		})
		It("should reject volume with no volume source set", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake[0]"))
		})
		It("should reject volume with multiple volume sources set", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					ContainerDisk:         testutils.NewFakeContainerDiskSource(),
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake[0]"))
		})
		It("should reject volumes with duplicate names", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					ContainerDisk: testutils.NewFakeContainerDiskSource(),
				},
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					ContainerDisk: testutils.NewFakeContainerDiskSource(),
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake[1].name"))
		})

		DescribeTable("should verify cloud-init userdata length", func(userDataLen int, expectedErrors int, base64Encode bool) {
			vmi := api.NewMinimalVMI("testvmi")

			// generate fake userdata
			userdata := ""
			for i := 0; i < userDataLen; i++ {
				userdata = fmt.Sprintf("%sa", userdata)
			}

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{VolumeSource: v1.VolumeSource{CloudInitNoCloud: &v1.CloudInitNoCloudSource{}}})

			if base64Encode {
				vmi.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.UserDataBase64 = base64.StdEncoding.EncodeToString([]byte(userdata))
			} else {
				vmi.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.UserData = userdata
			}

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(expectedErrors))
			for _, cause := range causes {
				Expect(cause.Field).To(ContainSubstring("fake[0].cloudInitNoCloud"))
			}
		},
			Entry("should accept userdata under max limit", 10, 0, false),
			Entry("should accept userdata equal max limit", cloudInitUserMaxLen, 0, false),
			Entry("should reject userdata greater than max limit", cloudInitUserMaxLen+1, 1, false),
			Entry("should accept userdata base64 under max limit", 10, 0, true),
			Entry("should accept userdata base64 equal max limit", cloudInitUserMaxLen, 0, true),
			Entry("should reject userdata base64 greater than max limit", cloudInitUserMaxLen+1, 1, true),
		)

		DescribeTable("should verify cloud-init networkdata length", func(networkDataLen int, expectedErrors int, base64Encode bool) {
			vmi := api.NewMinimalVMI("testvmi")

			// generate fake networkdata
			networkdata := ""
			for i := 0; i < networkDataLen; i++ {
				networkdata = fmt.Sprintf("%sa", networkdata)
			}

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{VolumeSource: v1.VolumeSource{CloudInitNoCloud: &v1.CloudInitNoCloudSource{}}})
			vmi.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.UserData = "#config"

			if base64Encode {
				vmi.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.NetworkDataBase64 = base64.StdEncoding.EncodeToString([]byte(networkdata))
			} else {
				vmi.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.NetworkData = networkdata
			}

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(expectedErrors))
			for _, cause := range causes {
				Expect(cause.Field).To(ContainSubstring("fake[0].cloudInitNoCloud"))
			}
		},
			Entry("should accept networkdata under max limit", 10, 0, false),
			Entry("should accept networkdata equal max limit", cloudInitNetworkMaxLen, 0, false),
			Entry("should reject networkdata greater than max limit", cloudInitNetworkMaxLen+1, 1, false),
			Entry("should accept networkdata base64 under max limit", 10, 0, true),
			Entry("should accept networkdata base64 equal max limit", cloudInitNetworkMaxLen, 0, true),
			Entry("should reject networkdata base64 greater than max limit", cloudInitNetworkMaxLen+1, 1, true),
		)

		It("should reject cloud-init with invalid base64 userdata", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					CloudInitNoCloud: &v1.CloudInitNoCloudSource{
						UserDataBase64: "#######garbage******",
					},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake[0].cloudInitNoCloud.userDataBase64"))
		})

		It("should reject cloud-init with invalid base64 networkdata", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					CloudInitNoCloud: &v1.CloudInitNoCloudSource{
						UserData:          "fake",
						NetworkDataBase64: "#######garbage******",
					},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake[0].cloudInitNoCloud.networkDataBase64"))
		})

		It("should reject cloud-init with multiple userdata sources", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					CloudInitNoCloud: &v1.CloudInitNoCloudSource{
						UserData: "fake",
						UserDataSecretRef: &k8sv1.LocalObjectReference{
							Name: "fake",
						},
					},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake[0].cloudInitNoCloud"))
		})

		It("should reject cloud-init with multiple networkdata sources", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					CloudInitNoCloud: &v1.CloudInitNoCloudSource{
						UserData:    "fake",
						NetworkData: "fake",
						NetworkDataSecretRef: &k8sv1.LocalObjectReference{
							Name: "fake",
						},
					},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake[0].cloudInitNoCloud"))
		})

		It("should reject hostDisk without required parameters", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					HostDisk: &v1.HostDisk{},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(2))
			Expect(causes[0].Field).To(Equal("fake[0].hostDisk.path"))
			Expect(causes[1].Field).To(Equal("fake[0].hostDisk.type"))
		})

		It("should reject hostDisk without given 'path'", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					HostDisk: &v1.HostDisk{
						Type: v1.HostDiskExistsOrCreate,
					},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake[0].hostDisk.path"))
		})

		It("should reject hostDisk with invalid type", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					HostDisk: &v1.HostDisk{
						Path: "fakePath",
						Type: "fakeType",
					},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake[0].hostDisk.type"))
		})

		It("should reject hostDisk when the capacity is specified with a `DiskExists` type", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					HostDisk: &v1.HostDisk{
						Path:     "fakePath",
						Type:     v1.HostDiskExists,
						Capacity: resource.MustParse("1Gi"),
					},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake[0].hostDisk.capacity"))
		})

		It("should reject a configMap without the configMapName field", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake[0].configMap.name"))
		})

		It("should reject a secret without the secretName field", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake[0].secret.secretName"))
		})

		It("should reject a serviceAccount without the serviceAccountName field", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					ServiceAccount: &v1.ServiceAccountVolumeSource{},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake[0].serviceAccount.serviceAccountName"))
		})

		It("should reject multiple serviceAccounts", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "sa1",
				VolumeSource: v1.VolumeSource{
					ServiceAccount: &v1.ServiceAccountVolumeSource{ServiceAccountName: "test1"},
				},
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "sa2",
				VolumeSource: v1.VolumeSource{
					ServiceAccount: &v1.ServiceAccountVolumeSource{ServiceAccountName: "test2"},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake"))
		})
	})

	DescribeTable("when snapshot is in progress, should", func(mutateFn func(*v1.VirtualMachine) bool) {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Devices.Disks = []v1.Disk{
			{
				Name: "orginalvolume",
			},
		}
		vmi.Spec.Volumes = []v1.Volume{
			{
				Name:         "orginalvolume",
				VolumeSource: v1.VolumeSource{EmptyDisk: &v1.EmptyDiskSource{}},
			},
		}
		vm := &v1.VirtualMachine{
			Spec: v1.VirtualMachineSpec{
				Running: &[]bool{false}[0],
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: vmi.Spec,
				},
			},
			Status: v1.VirtualMachineStatus{
				SnapshotInProgress: &[]string{"testsnapshot"}[0],
			},
		}
		oldObjectBytes, _ := json.Marshal(vm)

		allow := mutateFn(vm)
		objectBytes, _ := json.Marshal(vm)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Operation: admissionv1.Update,
				Resource:  webhooks.VirtualMachineGroupVersionResource,
				OldObject: runtime.RawExtension{
					Raw: oldObjectBytes,
				},
				Object: runtime.RawExtension{
					Raw: objectBytes,
				},
			},
		}

		resp := vmsAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(Equal(allow))

		if !allow {
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec"), resp.Result.Details.Causes[0].Message)
		}
	},
		Entry("reject update to disks", func(vm *v1.VirtualMachine) bool {
			vm.Spec.Template.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "testvolume",
				},
			}
			vm.Spec.Template.Spec.Volumes = []v1.Volume{
				{
					Name:         "testvolume",
					VolumeSource: v1.VolumeSource{EmptyDisk: &v1.EmptyDiskSource{}},
				},
			}
			return false
		}),
		Entry("reject adding volumes", func(vm *v1.VirtualMachine) bool {
			vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testvolume",
			})
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name:         "testvolume",
				VolumeSource: v1.VolumeSource{EmptyDisk: &v1.EmptyDiskSource{}},
			})
			return false
		}),
		Entry("reject update to volumees", func(vm *v1.VirtualMachine) bool {
			vm.Spec.Template.Spec.Volumes[0].VolumeSource = v1.VolumeSource{DataVolume: &v1.DataVolumeSource{Name: "fake"}}
			return false
		}),
		Entry("accept update to spec, that is not volumes or running state", func(vm *v1.VirtualMachine) bool {
			vm.Spec.Template.Spec.Affinity = &k8sv1.Affinity{}
			return true
		}),
		Entry("reject update to running state", func(vm *v1.VirtualMachine) bool {
			vm.Spec.Running = &[]bool{true}[0]
			return false
		}),
		Entry("accept update to running state, if value doesn't change", func(vm *v1.VirtualMachine) bool {
			vm.Spec.Running = &[]bool{false}[0]
			return true
		}),
		Entry("reject update to running state, when switch state type", func(vm *v1.VirtualMachine) bool {
			vm.Spec.Running = nil
			vm.Spec.RunStrategy = &runStrategyManual
			return false
		}),
		Entry("accept update to metadata", func(vm *v1.VirtualMachine) bool {
			vm.Annotations = map[string]string{"foo": "bar"}
			return true
		}),
		Entry("accept update to status", func(vm *v1.VirtualMachine) bool {
			vm.Status.Ready = true
			return true
		}),
	)

	DescribeTable("when restore is in progress, should", func(mutateFn func(*v1.VirtualMachine) bool, updateRunStrategy bool) {
		vmi := api.NewMinimalVMI("testvmi")
		vm := &v1.VirtualMachine{
			Spec: v1.VirtualMachineSpec{
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: vmi.Spec,
				},
			},
			Status: v1.VirtualMachineStatus{
				RestoreInProgress: &[]string{"testrestore"}[0],
			},
		}
		if updateRunStrategy {
			vm.Spec.RunStrategy = &runStrategyHalted
		} else {
			vm.Spec.Running = &[]bool{false}[0]
		}
		oldObjectBytes, _ := json.Marshal(vm)

		allow := mutateFn(vm)
		objectBytes, _ := json.Marshal(vm)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Operation: admissionv1.Update,
				Resource:  webhooks.VirtualMachineGroupVersionResource,
				OldObject: runtime.RawExtension{
					Raw: oldObjectBytes,
				},
				Object: runtime.RawExtension{
					Raw: objectBytes,
				},
			},
		}

		resp := vmsAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(Equal(allow))

		if !allow {
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec"))
		}
	},
		Entry("reject update to running true", func(vm *v1.VirtualMachine) bool {
			vm.Spec.Running = &[]bool{true}[0]
			return false
		}, false),
		Entry("reject update of runStrategy", func(vm *v1.VirtualMachine) bool {
			vm.Spec.RunStrategy = &runStrategyManual
			return false
		}, true),
		Entry("accept update to spec except running true", func(vm *v1.VirtualMachine) bool {
			vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{}
			return true
		}, false),
		Entry("accept update to metadata", func(vm *v1.VirtualMachine) bool {
			vm.Annotations = map[string]string{"foo": "bar"}
			return true
		}, false),
		Entry("accept update to status", func(vm *v1.VirtualMachine) bool {
			vm.Status.Ready = true
			return true
		}, false),
	)

	Context("Instancetype", func() {
		var (
			vm *v1.VirtualMachine
		)

		BeforeEach(func() {
			vmi := api.NewMinimalVMI("testvmi")
			vm = &v1.VirtualMachine{
				Spec: v1.VirtualMachineSpec{
					Instancetype: &v1.InstancetypeMatcher{
						Name: "test",
						Kind: instancetypeapi.SingularResourceName,
					},
					Preference: &v1.PreferenceMatcher{
						Name: "test",
						Kind: instancetypeapi.SingularPreferenceResourceName,
					},
					Running: pointer.P(false),
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						Spec: vmi.Spec,
					},
				},
			}
		})

		It("should reject if instancetype is not found", func() {
			instancetypeMethods.FindInstancetypeSpecFunc = func(_ *v1.VirtualMachine) (*instancetypev1beta1.VirtualMachineInstancetypeSpec, error) {
				return nil, fmt.Errorf("instancetype not found")
			}

			response := admitVm(vmsAdmitter, vm)
			Expect(response.Allowed).To(BeFalse())
			Expect(response.Result.Details.Causes).To(HaveLen(1))
			Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueNotFound))
			Expect(response.Result.Details.Causes[0].Field).To(Equal("spec.instancetype"))
		})

		It("should reject if preference is not found", func() {
			instancetypeMethods.FindPreferenceSpecFunc = func(_ *v1.VirtualMachine) (*instancetypev1beta1.VirtualMachinePreferenceSpec, error) {
				return nil, fmt.Errorf("preference not found")
			}

			response := admitVm(vmsAdmitter, vm)
			Expect(response.Allowed).To(BeFalse())
			Expect(response.Result.Details.Causes).To(HaveLen(1))
			Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueNotFound))
			Expect(response.Result.Details.Causes[0].Field).To(Equal("spec.preference"))
		})

		It("should reject if instancetype fails to apply to VMI", func() {
			var (
				basePath = k8sfield.NewPath("spec", "template", "spec")
				path1    = basePath.Child("example", "path")
				path2    = basePath.Child("domain", "example", "path")
			)
			instancetypeMethods.FindInstancetypeSpecFunc = func(_ *v1.VirtualMachine) (*instancetypev1beta1.VirtualMachineInstancetypeSpec, error) {
				return &instancetypev1beta1.VirtualMachineInstancetypeSpec{}, nil
			}
			instancetypeMethods.FindInstancetypeSpecFunc = func(_ *v1.VirtualMachine) (*instancetypev1beta1.VirtualMachineInstancetypeSpec, error) {
				return &instancetypev1beta1.VirtualMachineInstancetypeSpec{}, nil
			}
			instancetypeMethods.FindPreferenceSpecFunc = func(_ *v1.VirtualMachine) (*instancetypev1beta1.VirtualMachinePreferenceSpec, error) {
				return &instancetypev1beta1.VirtualMachinePreferenceSpec{}, nil
			}
			instancetypeMethods.ApplyToVmiFunc = func(_ *k8sfield.Path, _ *instancetypev1beta1.VirtualMachineInstancetypeSpec, _ *instancetypev1beta1.VirtualMachinePreferenceSpec, _ *v1.VirtualMachineInstanceSpec, vmiMetadata *metav1.ObjectMeta) instancetype.Conflicts {
				return instancetype.Conflicts{path1, path2}
			}

			response := admitVm(vmsAdmitter, vm)
			Expect(response.Allowed).To(BeFalse())
			Expect(response.Result.Details.Causes).To(HaveLen(2))
			Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(response.Result.Details.Causes[0].Field).To(Equal(path1.String()))
			Expect(response.Result.Details.Causes[1].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(response.Result.Details.Causes[1].Field).To(Equal(path2.String()))
		})

		It("should apply instancetype to VMI before validating VMI", func() {
			// Test that VMI without instancetype application is valid
			response := admitVm(vmsAdmitter, vm)
			Expect(response.Allowed).To(BeTrue())

			// Instancetype application sets invalid memory value
			instancetypeMethods.FindInstancetypeSpecFunc = func(_ *v1.VirtualMachine) (*instancetypev1beta1.VirtualMachineInstancetypeSpec, error) {
				return &instancetypev1beta1.VirtualMachineInstancetypeSpec{}, nil
			}
			instancetypeMethods.ApplyToVmiFunc = func(_ *k8sfield.Path, _ *instancetypev1beta1.VirtualMachineInstancetypeSpec, _ *instancetypev1beta1.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec, vmiMetadata *metav1.ObjectMeta) instancetype.Conflicts {
				vmiSpec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("-1Mi")
				return nil
			}

			// Test that VMI fails
			response = admitVm(vmsAdmitter, vm)
			Expect(response.Allowed).To(BeFalse())
			Expect(response.Result.Details.Causes).To(HaveLen(1))
			Expect(response.Result.Details.Causes[0].Field).
				To(Equal("spec.template.spec.domain.resources.requests.memory"))
		})

		It("should not apply instancetype to the VMISpec of the original VM", func() {

			instancetypeMethods.FindInstancetypeSpecFunc = func(_ *v1.VirtualMachine) (*instancetypev1beta1.VirtualMachineInstancetypeSpec, error) {
				return &instancetypev1beta1.VirtualMachineInstancetypeSpec{}, nil
			}

			// Mock out ApplyToVmiFunc so that it applies some changes to the CPU of the provided VMISpec
			instancetypeMethods.ApplyToVmiFunc = func(_ *k8sfield.Path, _ *instancetypev1beta1.VirtualMachineInstancetypeSpec, _ *instancetypev1beta1.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec, vmiMetadata *metav1.ObjectMeta) instancetype.Conflicts {
				vmiSpec.Domain.CPU = &v1.CPU{Cores: 1, Threads: 1, Sockets: 1}
				return nil
			}

			// Nil out CPU within the DomainSpec of the VMISpec being admitted to assert this remains untouched
			vm.Spec.Template.Spec.Domain.CPU = nil

			// The VM should be admitted successfully
			response := admitVm(vmsAdmitter, vm)
			Expect(response.Allowed).To(BeTrue())

			// Ensure CPU has remained nil within the now admitted VMISpec
			Expect(vm.Spec.Template.Spec.Domain.CPU).To(BeNil())

		})

		It("should reject if preference requirements are not met", func() {
			instancetypeMethods.FindPreferenceSpecFunc = func(_ *v1.VirtualMachine) (*instancetypev1beta1.VirtualMachinePreferenceSpec, error) {
				return &instancetypev1beta1.VirtualMachinePreferenceSpec{}, nil
			}
			instancetypeMethods.CheckPreferenceRequirementsFunc = func(_ *instancetypev1beta1.VirtualMachineInstancetypeSpec, _ *instancetypev1beta1.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec) (instancetype.Conflicts, error) {
				return instancetype.Conflicts{k8sfield.NewPath("spec", "instancetype")}, fmt.Errorf("requirements not met")
			}
			response := admitVm(vmsAdmitter, vm)
			Expect(response.Allowed).To(BeFalse())
			Expect(response.Result.Details.Causes).To(HaveLen(1))
			Expect(response.Result.Details.Causes[0].Field).To(Equal("spec.instancetype"))
			Expect(response.Result.Details.Causes[0].Message).To(ContainSubstring("failure checking preference requirements"))
		})

		DescribeTable("should reject if PreferSpread requested with", func(vCPUs uint32, preferenceSpec instancetypev1beta1.VirtualMachinePreferenceSpec, expectedMessage string) {
			instancetypeMethods.FindPreferenceSpecFunc = func(_ *v1.VirtualMachine) (*instancetypev1beta1.VirtualMachinePreferenceSpec, error) {
				return &preferenceSpec, nil
			}

			instancetypeMethods.FindInstancetypeSpecFunc = func(_ *v1.VirtualMachine) (*instancetypev1beta1.VirtualMachineInstancetypeSpec, error) {
				return &instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{
						Guest: vCPUs,
					},
				}, nil
			}

			response := admitVm(vmsAdmitter, vm)
			Expect(response.Allowed).To(BeFalse())
			Expect(response.Result.Details.Causes).To(HaveLen(1))
			Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(response.Result.Details.Causes[0].Message).To(Equal(expectedMessage))
			Expect(response.Result.Details.Causes[0].Field).To(Equal(instancetypeCPUGuestPath))
		},
			Entry("3 vCPUs, default of SpreadAcrossSocketsCores and default SocketCoreRatio of 2 with spread",
				uint32(3),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Spread),
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 3, 2),
			),
			Entry("3 vCPUs, default of SpreadAcrossSocketsCores and default SocketCoreRatio of 2 with preferSpread",
				uint32(3),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.DeprecatedPreferSpread),
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 3, 2),
			),
			Entry("2 vCPUs, default of SpreadAcrossSocketsCores and SocketCoreRatio via PreferSpreadSocketToCoreRatio of 3 with spread",
				uint32(2),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					PreferSpreadSocketToCoreRatio: uint32(3),
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Spread),
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 2, 3),
			),
			Entry("2 vCPUs, default of SpreadAcrossSocketsCores and SocketCoreRatio via PreferSpreadSocketToCoreRatio of 3 with preferSpread",
				uint32(2),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					PreferSpreadSocketToCoreRatio: uint32(3),
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.DeprecatedPreferSpread),
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 2, 3),
			),
			Entry("2 vCPUs, default of SpreadAcrossSocketsCores and SocketCoreRatio via SpreadOptions of 3 with spread",
				uint32(2),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Spread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(3)),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 2, 3),
			),
			Entry("2 vCPUs, default of SpreadAcrossSocketsCores and SocketCoreRatio via SpreadOptions of 3 with preferSpread",
				uint32(2),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.DeprecatedPreferSpread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(3)),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 2, 3),
			),
			Entry("4 vCPUs, default of SpreadAcrossSocketsCores and SocketCoreRatio via PreferSpreadSocketToCoreRatio of 3 with spread",
				uint32(4),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					PreferSpreadSocketToCoreRatio: uint32(3),
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Spread),
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 4, 3),
			),
			Entry("4 vCPUs, default of SpreadAcrossSocketsCores and SocketCoreRatio via PreferSpreadSocketToCoreRatio of 3 with preferSpread",
				uint32(4),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					PreferSpreadSocketToCoreRatio: uint32(3),
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.DeprecatedPreferSpread),
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 4, 3),
			),
			Entry("4 vCPUs, default of SpreadAcrossSocketsCores and SocketCoreRatio via SpreadOptions of 3 with spread",
				uint32(4),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Spread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(3)),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 4, 3),
			),
			Entry("4 vCPUs, default of SpreadAcrossSocketsCores and SocketCoreRatio via SpreadOptions of 3 with preferSpread",
				uint32(4),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.DeprecatedPreferSpread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(3)),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 4, 3),
			),
			Entry("3 vCPUs and SpreadAcrossCoresThreads with spread",
				uint32(3),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Spread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcrossCoresThreads),
						},
					},
				},
				fmt.Sprintf(spreadAcrossCoresThreadsErrFmt, 3, 2),
			),
			Entry("3 vCPUs and SpreadAcrossCoresThreads with preferSpread",
				uint32(3),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.DeprecatedPreferSpread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcrossCoresThreads),
						},
					},
				},
				fmt.Sprintf(spreadAcrossCoresThreadsErrFmt, 3, 2),
			),
			Entry("5 vCPUs and SpreadAcrossCoresThreads with spread",
				uint32(5),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Spread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcrossCoresThreads),
						},
					},
				},
				fmt.Sprintf(spreadAcrossCoresThreadsErrFmt, 5, 2),
			),
			Entry("5 vCPUs and SpreadAcrossCoresThreads with preferSpread",
				uint32(5),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.DeprecatedPreferSpread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcrossCoresThreads),
						},
					},
				},
				fmt.Sprintf(spreadAcrossCoresThreadsErrFmt, 5, 2),
			),
			Entry("5 vCPUs, SpreadAcrossSocketsCoresThreads and default SocketCoreRatio of 2 with spread",
				uint32(5),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Spread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcrossSocketsCoresThreads),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresThreadsErrFmt, 5, 2, 2),
			),
			Entry("5 vCPUs, SpreadAcrossSocketsCoresThreads and default SocketCoreRatio of 2 with preferSpread",
				uint32(5),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.DeprecatedPreferSpread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcrossSocketsCoresThreads),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresThreadsErrFmt, 5, 2, 2),
			),
			Entry("6 vCPUs, SpreadAcrossSocketsCoresThreads and SocketCoreRatio via PreferSpreadSocketToCoreRatio of 4 with spread",
				uint32(6),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					PreferSpreadSocketToCoreRatio: uint32(4),
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Spread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcrossSocketsCoresThreads),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresThreadsErrFmt, 6, 2, 4),
			),
			Entry("6 vCPUs, SpreadAcrossSocketsCoresThreads and SocketCoreRatio via PreferSpreadSocketToCoreRatio of 4 with preferSpread",
				uint32(6),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					PreferSpreadSocketToCoreRatio: uint32(4),
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.DeprecatedPreferSpread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcrossSocketsCoresThreads),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresThreadsErrFmt, 6, 2, 4),
			),
			Entry("6 vCPUs, SpreadAcrossSocketsCoresThreads and SocketCoreRatio via SpreadOptions of 4 with spread",
				uint32(6),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Spread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcrossSocketsCoresThreads),
							Ratio:  pointer.P(uint32(4)),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresThreadsErrFmt, 6, 2, 4),
			),
			Entry("6 vCPUs, SpreadAcrossSocketsCoresThreads and SocketCoreRatio via SpreadOptions of 4 with preferSpread",
				uint32(6),
				instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.DeprecatedPreferSpread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcrossSocketsCoresThreads),
							Ratio:  pointer.P(uint32(4)),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresThreadsErrFmt, 6, 2, 4),
			),
		)

		DescribeTable("should admit VM with preference using preferSpread and without instancetype", func(preferredCPUTopology instancetypev1beta1.PreferredCPUTopology) {
			vm.Spec.Instancetype = nil
			instancetypeMethods.FindPreferenceSpecFunc = func(_ *v1.VirtualMachine) (*instancetypev1beta1.VirtualMachinePreferenceSpec, error) {
				return &instancetypev1beta1.VirtualMachinePreferenceSpec{CPU: &instancetypev1beta1.CPUPreferences{PreferredCPUTopology: &preferredCPUTopology}}, nil
			}
			response := admitVm(vmsAdmitter, vm)
			Expect(response.Allowed).To(BeTrue())
		},
			Entry("with spread", instancetypev1beta1.Spread),
			Entry("with preferSpread", instancetypev1beta1.DeprecatedPreferSpread),
		)
	})

	Context("Live update", func() {
		var vm *v1.VirtualMachine

		BeforeEach(func() {
			vmi := api.NewMinimalVMI("testvmi")
			enableLiveUpdate()
			vm = &v1.VirtualMachine{
				Spec: v1.VirtualMachineSpec{
					Running: pointer.P(false),
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						Spec: vmi.Spec,
					},
				},
			}
		})

		AfterEach(func() {
			disableFeatureGates()
		})

		Context("CPU", func() {
			const maximumSockets uint32 = 24

			BeforeEach(func() {
				vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
					MaxSockets: maximumSockets,
				}
			})

			It("should reject VM creation when number of sockets exceeds the maximum configured", func() {
				vm.Spec.Template.Spec.Domain.CPU.Sockets = maximumSockets + 1
				response := admitVm(vmsAdmitter, vm)
				Expect(response.Allowed).To(BeFalse())
				Expect(response.Result.Details.Causes[0].Field).To(Equal("spec.template.spec.domain.cpu.sockets"))
				Expect(response.Result.Details.Causes[0].Message).To(ContainSubstring("Number of sockets in CPU topology is greater than the maximum sockets allowed"))
			})

			When("Hot CPU change is in progress", func() {
				BeforeEach(func() {
					vm.Status.Ready = true
				})
			})
		})

		Context("Memory", func() {
			var maxGuest resource.Quantity

			BeforeEach(func() {
				checks.SkipIfS390X(rt.GOARCH, "Memory hotplug is not supported for s390x")
				guest := resource.MustParse("1Gi")
				maxGuest = resource.MustParse("4Gi")

				vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{
					Guest:    &guest,
					MaxGuest: &maxGuest,
				}
				vm.Spec.Template.Spec.Architecture = rt.GOARCH
				vm.Spec.Template.Spec.Domain.Resources.Limits = nil
				vm.Spec.Template.Spec.Domain.Resources.Requests = nil
				vm.Status.Ready = true
			})

			DescribeTable("should reject VM creation if", func(vmSetup func(*v1.VirtualMachine), cause metav1.StatusCause) {
				vmSetup(vm)

				response := admitVm(vmsAdmitter, vm)
				Expect(response.Allowed).To(BeFalse())
				Expect(response.Result.Details.Causes).To(ContainElement(cause))
			},
				Entry("realtime is configured", func(vm *v1.VirtualMachine) {
					vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
						DedicatedCPUPlacement: true,
						Realtime:              &v1.Realtime{},
						NUMA: &v1.NUMA{
							GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{},
						},
					}
					vm.Spec.Template.Spec.Domain.Memory.Hugepages = &v1.Hugepages{
						PageSize: "2Mi",
					}
				}, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Field:   "spec.template.spec.domain.memory.guest",
					Message: "Memory hotplug is not compatible with realtime VMs",
				}),
				Entry("launchSecurity is configured", func(vm *v1.VirtualMachine) {
					enableFeatureGate(virtconfig.WorkloadEncryptionSEV)
					vm.Spec.Template.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{}
				}, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Field:   "spec.template.spec.domain.memory.guest",
					Message: "Memory hotplug is not compatible with encrypted VMs",
				}),
				Entry("guest mapping passthrough is configured", func(vm *v1.VirtualMachine) {
					vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
						DedicatedCPUPlacement: true,
						NUMA: &v1.NUMA{
							GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{},
						},
					}
					vm.Spec.Template.Spec.Domain.Memory.Hugepages = &v1.Hugepages{
						PageSize: "2Mi",
					}
				}, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Field:   "spec.template.spec.domain.memory.guest",
					Message: "Memory hotplug is not compatible with guest mapping passthrough",
				}),
				Entry("guest memory is not set", func(vm *v1.VirtualMachine) {
					vm.Spec.Template.Spec.Domain.Memory.Guest = nil
				}, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Field:   "spec.template.spec.domain.memory.guest",
					Message: "Guest memory must be configured when memory hotplug is enabled",
				}),
				Entry("guest memory is greater than maxGuest", func(vm *v1.VirtualMachine) {
					moreThanMax := maxGuest.DeepCopy()
					moreThanMax.Add(resource.MustParse("16Mi"))

					vm.Spec.Template.Spec.Domain.Memory.Guest = &moreThanMax
				}, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Field:   "spec.template.spec.domain.memory.guest",
					Message: "Guest memory is greater than the configured maxGuest memory",
				}),
				Entry("maxGuest is not properly aligned", func(vm *v1.VirtualMachine) {
					unAlignedMemory := resource.MustParse("2049Mi")
					vm.Spec.Template.Spec.Domain.Memory.MaxGuest = &unAlignedMemory
				}, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Field:   "spec.template.spec.domain.memory.guest",
					Message: fmt.Sprintf("MaxGuest must be %s aligned", resource.NewQuantity(memory.HotplugBlockAlignmentBytes, resource.BinarySI)),
				}),
				Entry("guest memory is not properly aligned", func(vm *v1.VirtualMachine) {
					unAlignedMemory := resource.MustParse("1025Mi")
					vm.Spec.Template.Spec.Domain.Memory.Guest = &unAlignedMemory
				}, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Field:   "spec.template.spec.domain.memory.guest",
					Message: fmt.Sprintf("Guest memory must be %s aligned", resource.NewQuantity(memory.HotplugBlockAlignmentBytes, resource.BinarySI)),
				}),
				Entry("guest memory with hugepages is not properly aligned", func(vm *v1.VirtualMachine) {
					vm.Spec.Template.Spec.Domain.Memory.Guest = pointer.P(resource.MustParse("2G"))
					vm.Spec.Template.Spec.Domain.Memory.MaxGuest = pointer.P(resource.MustParse("16Gi"))
					vm.Spec.Template.Spec.Domain.Memory.Hugepages = &v1.Hugepages{PageSize: "1Gi"}
				}, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Field:   "spec.template.spec.domain.memory.guest",
					Message: fmt.Sprintf("Guest memory must be %s aligned", resource.NewQuantity(memory.Hotplug1GHugePagesBlockAlignmentBytes, resource.BinarySI)),
				}),
				Entry("architecture is not amd64 or arm64", func(vm *v1.VirtualMachine) {
					enableFeatureGate(virtconfig.MultiArchitecture)
					vm.Spec.Template.Spec.Architecture = "risc-v"
				}, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Field:   "spec.template.spec.domain.memory.guest",
					Message: "Memory hotplug is only available for x86_64 and arm64 VMs",
				}),
				Entry("guest memory is less than 1Gi", func(vm *v1.VirtualMachine) {
					vm.Spec.Template.Spec.Domain.Memory.Guest = pointer.P(resource.MustParse("512Mi"))
				}, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Field:   "spec.template.spec.domain.memory.guest",
					Message: "Memory hotplug is only available for VMs with at least 1Gi of guest memory",
				}),
			)
		})

		Context("Update volume strategy", func() {
			It("should accept the VM with the feature gate enabled", func() {
				enableFeatureGate(virtconfig.VolumesUpdateStrategy)
				vm.Spec.UpdateVolumesStrategy = pointer.P(v1.UpdateVolumesStrategyReplacement)
				resp := admitVm(vmsAdmitter, vm)
				Expect(resp.Allowed).To(BeTrue())
				Expect(resp.Result).To(BeNil())
			})
			It("should reject the VM creation if the feature gate isn't enabled", func() {
				vm.Spec.UpdateVolumesStrategy = pointer.P(v1.UpdateVolumesStrategyReplacement)
				resp := admitVm(vmsAdmitter, vm)
				Expect(resp.Allowed).To(BeFalse())
				Expect(resp.Result.Details.Causes).To(ContainElement(metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Field:   "updateVolumesStrategy",
					Message: fmt.Sprintf("%s feature gate is not enabled in kubevirt-config", virtconfig.VolumesUpdateStrategy),
				}))
			})
			It("should accept the VM with the feature gate enabled for volume migration", func() {
				enableFeatureGate(virtconfig.VolumesUpdateStrategy, virtconfig.VolumeMigration)
				vm.Spec.UpdateVolumesStrategy = pointer.P(v1.UpdateVolumesStrategyMigration)
				resp := admitVm(vmsAdmitter, vm)
				Expect(resp.Allowed).To(BeTrue())
				Expect(resp.Result).To(BeNil())
			})
			It("should reject the VM creation if the volume migration feature gate isn't enabled", func() {
				enableFeatureGate(virtconfig.VolumesUpdateStrategy)
				vm.Spec.UpdateVolumesStrategy = pointer.P(v1.UpdateVolumesStrategyMigration)
				resp := admitVm(vmsAdmitter, vm)
				Expect(resp.Allowed).To(BeFalse())
				Expect(resp.Result.Details.Causes).To(ContainElement(metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Field:   "updateVolumesStrategy",
					Message: fmt.Sprintf("%s feature gate is not enabled in kubevirt-config", virtconfig.VolumeMigration),
				}))
			})
		})
	})

	It("should raise a warning when Deprecated API is used", func() {
		const testsFGName = "test-deprecated"
		deprecation.RegisterFeatureGate(deprecation.FeatureGate{
			Name:        testsFGName,
			State:       deprecation.Deprecated,
			VmiSpecUsed: func(_ *v1.VirtualMachineInstanceSpec) bool { return true },
		})
		DeferCleanup(deprecation.UnregisterFeatureGate, testsFGName)
		enableFeatureGate(testsFGName)

		vmi := api.NewMinimalVMI("testvmi")
		vm := &v1.VirtualMachine{
			Spec: v1.VirtualMachineSpec{
				Running: pointer.P(false),
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: vmi.Spec,
				},
			},
		}

		resp := admitVm(vmsAdmitter, vm)
		Expect(resp.Allowed).To(BeTrue())
		Expect(resp.Result).To(BeNil())
		Expect(resp.Warnings).To(HaveLen(2))
		Expect(resp.Warnings).To(ConsistOf(
			HavePrefix("feature gate test-deprecated is deprecated"),
			HavePrefix("spec.running is deprecated, please use spec.runStrategy instead.")))
	})
})

func admitVm(admitter *VMsAdmitter, vm *v1.VirtualMachine) *admissionv1.AdmissionResponse {
	vmBytes, _ := json.Marshal(vm)

	ar := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Resource: webhooks.VirtualMachineGroupVersionResource,
			Object: runtime.RawExtension{
				Raw: vmBytes,
			},
		},
	}

	return admitter.Admit(context.Background(), ar)
}
