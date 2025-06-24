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
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	rt "runtime"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"
	fakeclientset "kubevirt.io/client-go/kubevirt/fake"

	v1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	instancetypeWebhooks "kubevirt.io/kubevirt/pkg/instancetype/webhooks/vm"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/liveupdate/memory"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/framework/checks"
)

var _ = Describe("Validating VM Admitter", func() {
	config, crdInformer, kvStore := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
	var (
		vmsAdmitter        *VMsAdmitter
		dataSourceInformer cache.SharedIndexInformer
		namespaceInformer  cache.SharedIndexInformer
		mockVMIClient      *kubecli.MockVirtualMachineInstanceInterface
		virtClient         *kubecli.MockKubevirtClient
		k8sClient          *k8sfake.Clientset
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

		ctrl := gomock.NewController(GinkgoT())
		mockVMIClient = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		k8sClient = k8sfake.NewSimpleClientset()
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		const kubeVirtNamespace = "kubevirt"
		vmsAdmitter = &VMsAdmitter{
			VirtClient:              virtClient,
			DataSourceInformer:      dataSourceInformer,
			NamespaceInformer:       namespaceInformer,
			ClusterConfig:           config,
			InstancetypeAdmitter:    instancetypeWebhooks.NewMockAdmitter(),
			KubeVirtServiceAccounts: webhooks.KubeVirtServiceAccounts(kubeVirtNamespace),
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
		Entry("with valid request to add volume (virtio)", []v1.VirtualMachineVolumeRequest{
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "testdisk2",
					Disk: &v1.Disk{
						Name: "testdisk2",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: "virtio",
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
		Entry("with invalid request to add volume to a LUN disk (virtio bus)", []v1.VirtualMachineVolumeRequest{
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "testlun2",
					Disk: &v1.Disk{
						Name: "testlun2",
						DiskDevice: v1.DiskDevice{
							LUN: &v1.LunTarget{
								Bus: "virtio",
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
			false),
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
		Entry("with valid request to add volume with dedicated IOThreads snd virtio bus", []v1.VirtualMachineVolumeRequest{
			{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "testdisk2",
					Disk: &v1.Disk{
						Name: "testdisk",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: "virtio",
							},
						},
						DedicatedIOThread: pointer.P(true),
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

	Context("with Volume", func() {

		BeforeEach(func() {
			enableFeatureGate(featuregate.HostDiskGate)
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

	Context("Instancetype", func() {
		BeforeEach(func() {
			virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(
				fakeclientset.NewSimpleClientset().InstancetypeV1beta1().VirtualMachineClusterInstancetypes()).AnyTimes()

			vmsAdmitter.InstancetypeAdmitter = instancetypeWebhooks.NewAdmitter(virtClient)
		})
		It("should not apply instancetype to the VMISpec of the original VM", func() {
			const clusterInstancetypeName = "clusterInstancetype"
			clusterInstancetype := &instancetypev1beta1.VirtualMachineClusterInstancetype{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterInstancetypeName,
				},
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{
						Guest: uint32(2),
					},
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("128Mi"),
					},
				},
			}
			_, err := virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), clusterInstancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// The VM should be admitted successfully
			vm := libvmi.NewVirtualMachine(
				libvmi.New(libvmi.WithNamespace(metav1.NamespaceDefault)),
				libvmi.WithClusterInstancetype(clusterInstancetypeName),
			)
			response := admitVm(vmsAdmitter, vm)
			Expect(response.Allowed).To(BeTrue())

			// Ensure CPU has remained nil within the now admitted VMISpec
			Expect(vm.Spec.Template.Spec.Domain.CPU).To(BeNil())
		})
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
					enableFeatureGate(featuregate.WorkloadEncryptionSEV)
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
					enableFeatureGate(featuregate.MultiArchitecture)
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

	})

	It("should raise a warning when Deprecated API is used", func() {
		const testsFGName = "test-deprecated"
		featuregate.RegisterFeatureGate(featuregate.FeatureGate{
			Name:        testsFGName,
			State:       featuregate.Deprecated,
			VmiSpecUsed: func(_ *v1.VirtualMachineInstanceSpec) bool { return true },
		})
		DeferCleanup(featuregate.UnregisterFeatureGate, testsFGName)
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

	It("should reject request when Discontinued feature is used", func() {
		const fgName = "test-discontinued"
		const fgMessage = "FG is discontinued"
		featuregate.RegisterFeatureGate(featuregate.FeatureGate{
			Name:        fgName,
			State:       featuregate.Discontinued,
			VmiSpecUsed: func(_ *v1.VirtualMachineInstanceSpec) bool { return true },
			Message:     fgMessage,
		})
		DeferCleanup(featuregate.UnregisterFeatureGate, fgName)
		enableFeatureGate(fgName)

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
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result).ToNot(BeNil())
		Expect(resp.Result.Message).To(Equal(fgMessage))
		Expect(resp.Result.Details.Causes).To(HaveLen(1))
		Expect(resp.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueNotSupported))
		Expect(resp.Result.Details.Causes[0].Message).To(Equal(fgMessage))
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
			Operation: admissionv1.Create,
		},
	}

	return admitter.Admit(context.Background(), ar)
}
