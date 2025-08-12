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

package mutators

import (
	"context"
	"encoding/json"
	"net/http"
	rt "runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	cdifake "kubevirt.io/client-go/containerizeddataimporter/fake"
	fakeclientset "kubevirt.io/client-go/kubevirt/fake"
	instancetypeclientset "kubevirt.io/client-go/kubevirt/typed/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	instancetypeVMWebhooks "kubevirt.io/kubevirt/pkg/instancetype/webhooks/vm"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("VirtualMachine Mutator", func() {
	var vm *v1.VirtualMachine
	var kvStore cache.Store
	var mutator *VMsMutator
	var ctrl *gomock.Controller
	var virtClient *kubecli.MockKubevirtClient
	var fakeInstancetypeClients instancetypeclientset.InstancetypeV1beta1Interface
	var fakePreferenceClient instancetypeclientset.VirtualMachinePreferenceInterface
	var fakeClusterPreferenceClient instancetypeclientset.VirtualMachineClusterPreferenceInterface
	var k8sClient *k8sfake.Clientset
	var cdiClient *cdifake.Clientset

	machineTypeFromConfig := "pc-q35-3.0"
	ignoreInferFromVolumeFailure := v1.IgnoreInferFromVolumeFailure
	rejectInferFromVolumeFailure := v1.RejectInferFromVolumeFailure

	admitVM := func(op admissionv1.Operation) *admissionv1.AdmissionResponse {
		vmBytes, err := json.Marshal(vm)
		Expect(err).ToNot(HaveOccurred())
		By("Creating the test admissions review from the VM")
		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Operation: op,
				Resource:  k8smetav1.GroupVersionResource{Group: v1.VirtualMachineGroupVersionKind.Group, Version: v1.VirtualMachineGroupVersionKind.Version, Resource: "virtualmachines"},
				Object: runtime.RawExtension{
					Raw: vmBytes,
				},
			},
		}
		By("Mutating the VM")
		return mutator.Mutate(ar)
	}

	admitVMWithArch := func(arch string, op admissionv1.Operation) *admissionv1.AdmissionResponse {
		vm.Spec.Template.Spec.Architecture = arch
		return admitVM(op)
	}

	getVMSpecMetaFromResponse := func(resp *admissionv1.AdmissionResponse) (*v1.VirtualMachineSpec, *k8smetav1.ObjectMeta) {
		By("Getting the VM spec from the response")
		vmSpec := &v1.VirtualMachineSpec{}
		vmMeta := &k8smetav1.ObjectMeta{}
		patchOps := []patch.PatchOperation{
			{Value: vmSpec},
			{Value: vmMeta},
		}
		err := json.Unmarshal(resp.Patch, &patchOps)
		Expect(err).ToNot(HaveOccurred())
		Expect(patchOps).NotTo(BeEmpty())

		return vmSpec, vmMeta
	}

	getVMSpecMetaFromResponseCreate := func() (*v1.VirtualMachineSpec, *k8smetav1.ObjectMeta) {
		resp := admitVM(admissionv1.Create)
		Expect(resp.Allowed).To(BeTrue())
		return getVMSpecMetaFromResponse(resp)
	}

	getVMSpecMetaFromResponseCreateWithArch := func(arch string) (*v1.VirtualMachineSpec, *k8smetav1.ObjectMeta) {
		resp := admitVMWithArch(arch, admissionv1.Create)
		Expect(resp.Allowed).To(BeTrue())
		return getVMSpecMetaFromResponse(resp)
	}

	getResponseFromVMUpdate := func(oldVM *v1.VirtualMachine, newVM *v1.VirtualMachine) *admissionv1.AdmissionResponse {
		oldVMBytes, err := json.Marshal(oldVM)
		Expect(err).ToNot(HaveOccurred())
		newVMBytes, err := json.Marshal(newVM)
		Expect(err).ToNot(HaveOccurred())
		By("Creating the test admissions review from the VM")
		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Operation: admissionv1.Update,
				Resource:  k8smetav1.GroupVersionResource{Group: v1.VirtualMachineGroupVersionKind.Group, Version: v1.VirtualMachineGroupVersionKind.Version, Resource: "virtualmachines"},
				Object: runtime.RawExtension{
					Raw: newVMBytes,
				},
				OldObject: runtime.RawExtension{
					Raw: oldVMBytes,
				},
			},
		}
		By("Mutating the VM")
		return mutator.Mutate(ar)
	}

	BeforeEach(func() {
		vm = &v1.VirtualMachine{
			ObjectMeta: k8smetav1.ObjectMeta{
				Labels: map[string]string{"test": "test"},
			},
		}
		vm.Namespace = k8sv1.NamespaceDefault
		vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{}

		mutator = &VMsMutator{}
		mutator.ClusterConfig, _, kvStore = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})

		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		fakeInstancetypeClients = fakeclientset.NewSimpleClientset().InstancetypeV1beta1()
		fakePreferenceClient = fakeInstancetypeClients.VirtualMachinePreferences(vm.Namespace)
		fakeClusterPreferenceClient = fakeInstancetypeClients.VirtualMachineClusterPreferences()
		virtClient.EXPECT().VirtualMachinePreference(gomock.Any()).Return(fakePreferenceClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineClusterPreference().Return(fakeClusterPreferenceClient).AnyTimes()

		k8sClient = k8sfake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
		cdiClient = cdifake.NewSimpleClientset()
		virtClient.EXPECT().CdiClient().Return(cdiClient).AnyTimes()

		mutator.instancetypeMutator = instancetypeVMWebhooks.NewMutator(virtClient)
	})

	It("should allow VM being deleted without applying mutations", func() {
		now := k8smetav1.Now()
		vm.ObjectMeta.DeletionTimestamp = &now
		resp := admitVM(admissionv1.Delete)
		Expect(resp.Allowed).To(BeTrue())
		Expect(resp.Patch).To(BeEmpty())
	})

	DescribeTable("should apply defaults on VM create", func(arch string, result string) {
		vmSpec, _ := getVMSpecMetaFromResponseCreateWithArch(arch)
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(result))
		Expect(vmSpec.Template.Spec.Domain.Firmware.UUID).ToNot(BeNil())
	},
		Entry("ppc64le", "ppc64le", "pseries"),
		Entry("arm64", "arm64", "virt"),
		Entry("s390x", "s390x", "s390-ccw-virtio"),
		Entry("amd64", "amd64", "q35"),
	)

	DescribeTable("should apply configurable defaults on VM create", func(arch string, amd64MachineType string, arm64MachineType string, ppc64leMachineType string, s390xMachineType string, result string) {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					ArchitectureConfiguration: &v1.ArchConfiguration{
						Amd64:   &v1.ArchSpecificConfiguration{MachineType: amd64MachineType},
						Arm64:   &v1.ArchSpecificConfiguration{MachineType: arm64MachineType},
						Ppc64le: &v1.ArchSpecificConfiguration{MachineType: ppc64leMachineType},
						S390x:   &v1.ArchSpecificConfiguration{MachineType: s390xMachineType},
					},
				},
			},
		})

		vmSpec, _ := getVMSpecMetaFromResponseCreateWithArch(arch)
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(result))

	},
		Entry("when override is for amd64 architecture", "amd64", machineTypeFromConfig, "", "", "", machineTypeFromConfig),
		Entry("when override is for arm64 architecture", "arm64", "", machineTypeFromConfig, "", "", machineTypeFromConfig),
		Entry("when override is for ppc64le architecture", "ppc64le", "", "", machineTypeFromConfig, "", machineTypeFromConfig),
		Entry("when override is for s390x architecture", "s390x", "", "", "", machineTypeFromConfig, machineTypeFromConfig),
	)

	It("should not override default architecture with defaults on VM create", func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
			Status: v1.KubeVirtStatus{
				DefaultArchitecture: "arm64",
			},
		})

		vmSpec, _ := getVMSpecMetaFromResponseCreateWithArch("amd64")
		Expect(vmSpec.Template.Spec.Architecture).To(Equal("amd64"))
	})

	DescribeTable("should not override specified properties with defaults on VM create", func(arch string) {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					MachineType: machineTypeFromConfig,
				},
			},
		})

		vm.Spec.Template.Spec.Domain.Machine = &v1.Machine{Type: "pc-q35-2.0"}

		vmSpec, _ := getVMSpecMetaFromResponseCreateWithArch(arch)
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(vm.Spec.Template.Spec.Domain.Machine.Type))
	},
		Entry("amd64", "amd64"),
		Entry("s390x", "s390x"),
		Entry("arm64", "arm64"),
	)

	DescribeTable("should not override user specified MachineType with PreferredMachineType or cluster config on VM create", func(arch string) {
		vm.Spec.Template.Spec.Domain.Machine = &v1.Machine{Type: "pc-q35-2.0"}
		preference := &instancetypev1beta1.VirtualMachinePreference{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name: "machineTypePreference",
			},
			TypeMeta: k8smetav1.TypeMeta{
				Kind:       apiinstancetype.SingularPreferenceResourceName,
				APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
			},
			Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
				Machine: &instancetypev1beta1.MachinePreferences{
					PreferredMachineType: "pc-q35-4.0",
				},
			},
		}
		_, err := virtClient.VirtualMachinePreference(vm.Namespace).Create(context.Background(), preference, k8smetav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vm.Spec.Preference = &v1.PreferenceMatcher{
			Name: preference.Name,
			Kind: apiinstancetype.SingularPreferenceResourceName,
		}

		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					MachineType: machineTypeFromConfig,
				},
			},
		})

		vmSpec, _ := getVMSpecMetaFromResponseCreateWithArch(arch)
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(vm.Spec.Template.Spec.Domain.Machine.Type))
	},
		Entry("amd64", "amd64"),
		Entry("s390x", "s390x"),
		Entry("arm64", "arm64"),
	)

	DescribeTable("should use PreferredMachineType over cluster config on VM create", func(arch string) {
		preference := &instancetypev1beta1.VirtualMachinePreference{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name: "machineTypePreference",
			},
			TypeMeta: k8smetav1.TypeMeta{
				Kind:       apiinstancetype.SingularPreferenceResourceName,
				APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
			},
			Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
				Machine: &instancetypev1beta1.MachinePreferences{
					PreferredMachineType: "pc-q35-4.0",
				},
			},
		}
		_, err := virtClient.VirtualMachinePreference(vm.Namespace).Create(context.Background(), preference, k8smetav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vm.Spec.Preference = &v1.PreferenceMatcher{
			Name: preference.Name,
			Kind: apiinstancetype.SingularPreferenceResourceName,
		}

		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					MachineType: machineTypeFromConfig,
				},
			},
		})

		vmSpec, _ := getVMSpecMetaFromResponseCreateWithArch(arch)
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(preference.Spec.Machine.PreferredMachineType))
	},
		Entry("amd64", "amd64"),
		Entry("s390x", "s390x"),
		Entry("arm64", "arm64"),
	)

	DescribeTable("should ignore error looking up preference and apply cluster config on VM create", func(arch string) {
		vm.Spec.Preference = &v1.PreferenceMatcher{
			Name: "foobar",
			Kind: apiinstancetype.SingularPreferenceResourceName,
		}

		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					ArchitectureConfiguration: &v1.ArchConfiguration{
						Amd64:   &v1.ArchSpecificConfiguration{MachineType: machineTypeFromConfig},
						Arm64:   &v1.ArchSpecificConfiguration{MachineType: machineTypeFromConfig},
						Ppc64le: &v1.ArchSpecificConfiguration{MachineType: machineTypeFromConfig},
						S390x:   &v1.ArchSpecificConfiguration{MachineType: machineTypeFromConfig},
					},
				},
			},
		})

		vmSpec, _ := getVMSpecMetaFromResponseCreateWithArch(arch)
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(machineTypeFromConfig))
	},
		Entry("amd64", "amd64"),
		Entry("arm64", "arm64"),
		Entry("s390x", "s390x"),
	)

	It("should default instancetype kind to ClusterSingularResourceName when not provided", func() {
		vm.Spec.Instancetype = &v1.InstancetypeMatcher{
			Name: "foobar",
		}
		vmSpec, _ := getVMSpecMetaFromResponseCreate()
		Expect(vmSpec.Instancetype.Kind).To(Equal(apiinstancetype.ClusterSingularResourceName))
	})

	It("should default preference kind to ClusterSingularPreferenceResourceName when not provided", func() {
		vm.Spec.Preference = &v1.PreferenceMatcher{
			Name: "foobar",
		}
		vmSpec, _ := getVMSpecMetaFromResponseCreate()
		Expect(vmSpec.Preference.Kind).To(Equal(apiinstancetype.ClusterSingularPreferenceResourceName))
	})

	DescribeTable("should use PreferredMachineType from ClusterSingularPreferenceResourceName when no preference kind is provided", func(arch string) {
		preference := &instancetypev1beta1.VirtualMachineClusterPreference{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name: "machineTypeClusterPreference",
			},
			TypeMeta: k8smetav1.TypeMeta{
				Kind:       apiinstancetype.ClusterSingularPreferenceResourceName,
				APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
			},
			Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
				Machine: &instancetypev1beta1.MachinePreferences{
					PreferredMachineType: "pc-q35-5.0",
				},
			},
		}
		_, err := virtClient.VirtualMachineClusterPreference().Create(context.Background(), preference, k8smetav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vm.Spec.Preference = &v1.PreferenceMatcher{
			Name: preference.Name,
		}

		vmSpec, _ := getVMSpecMetaFromResponseCreateWithArch(arch)
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(preference.Spec.Machine.PreferredMachineType))
	},
		Entry("amd64", "amd64"),
		Entry("s390x", "s390x"),
		Entry("arm64", "arm64"),
	)

	DescribeTable("should admit valid values to InferFromVolumePolicy", func(instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher) {
		vm.Spec.Instancetype = instancetypeMatcher
		vm.Spec.Preference = preferenceMatcher
		resp := admitVM(admissionv1.Create)
		Expect(resp.Allowed).To(BeTrue())
	},
		Entry("InstancetypeMatcher with IgnoreInferFromVolumeFailure", &v1.InstancetypeMatcher{Name: "bar", InferFromVolumeFailurePolicy: &ignoreInferFromVolumeFailure}, nil),
		Entry("InstancetypeMatcher with RejectInferFromVolumeFailure", &v1.InstancetypeMatcher{Name: "bar", InferFromVolumeFailurePolicy: &rejectInferFromVolumeFailure}, nil),
		Entry("PreferenceMatcher with IgnoreInferFromVolumeFailure", nil, &v1.PreferenceMatcher{Name: "bar", InferFromVolumeFailurePolicy: &ignoreInferFromVolumeFailure}),
		Entry("PreferenceMatcher with RejectInferFromVolumeFailure", nil, &v1.PreferenceMatcher{Name: "bar", InferFromVolumeFailurePolicy: &rejectInferFromVolumeFailure}),
	)

	Context("setPreferenceStorageClassName", func() {

		var preference *instancetypev1beta1.VirtualMachineClusterPreference

		BeforeEach(func() {
			preference = &instancetypev1beta1.VirtualMachineClusterPreference{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name: "machineTypeClusterPreference",
				},
				TypeMeta: k8smetav1.TypeMeta{
					Kind:       apiinstancetype.ClusterSingularPreferenceResourceName,
					APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
				},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					Volumes: &instancetypev1beta1.VolumePreferences{
						PreferredStorageClassName: "ceph",
					},
				},
			}
			_, err := virtClient.VirtualMachineClusterPreference().Create(context.Background(), preference, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: preference.Name,
			}

			vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{{
				Spec: cdiv1.DataVolumeSpec{},
			}}
		})

		assertPVCStorageClassName := func(dataVolumeTemapltes []v1.DataVolumeTemplateSpec, expectedStorageClassName string) {
			Expect(dataVolumeTemapltes).To(HaveLen(1))
			Expect(*dataVolumeTemapltes[0].Spec.PVC.StorageClassName).To(Equal(expectedStorageClassName))
		}

		assertStorageStorageClassName := func(dataVolumeTemapltes []v1.DataVolumeTemplateSpec, expectedStorageClassName string) {
			Expect(dataVolumeTemapltes).To(HaveLen(1))
			Expect(*dataVolumeTemapltes[0].Spec.Storage.StorageClassName).To(Equal(expectedStorageClassName))
		}

		It("[test_id:10328]should apply PreferredStorageClassName to PVC", func() {
			vm.Spec.DataVolumeTemplates[0].Spec.PVC = &k8sv1.PersistentVolumeClaimSpec{}
			vmSpec, _ := getVMSpecMetaFromResponseCreate()
			assertPVCStorageClassName(vmSpec.DataVolumeTemplates, preference.Spec.Volumes.PreferredStorageClassName)
		})

		It("[test_id:10329]should apply PreferredStorageClassName to Storage", func() {
			vm.Spec.DataVolumeTemplates[0].Spec.Storage = &cdiv1.StorageSpec{}
			vmSpec, _ := getVMSpecMetaFromResponseCreate()
			assertStorageStorageClassName(vmSpec.DataVolumeTemplates, preference.Spec.Volumes.PreferredStorageClassName)
		})

		It("should not fail if DataVolumeSpec PersistentVolumeClaimSpec is nil - bug #9868", func() {
			vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{{
				Spec: cdiv1.DataVolumeSpec{},
			}}
			resp := admitVM(admissionv1.Create)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("should not overwrite storageclass already defined in PVC of DataVolumeTemplate", func() {
			storageClass := "local"
			vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{{
				Spec: cdiv1.DataVolumeSpec{
					PVC: &k8sv1.PersistentVolumeClaimSpec{
						StorageClassName: &storageClass,
					},
				},
			}}
			vmSpec, _ := getVMSpecMetaFromResponseCreate()
			assertPVCStorageClassName(vmSpec.DataVolumeTemplates, storageClass)
		})

		It("should not overwrite storageclass already defined in Storage of DataVolumeTemplate", func() {
			storageClass := "local"
			vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{{
				Spec: cdiv1.DataVolumeSpec{
					Storage: &cdiv1.StorageSpec{
						StorageClassName: &storageClass,
					},
				},
			}}
			vmSpec, _ := getVMSpecMetaFromResponseCreate()
			assertStorageStorageClassName(vmSpec.DataVolumeTemplates, storageClass)
		})
	})

	Context("on update", func() {
		var oldVM, newVM *v1.VirtualMachine
		BeforeEach(func() {
			virtClient.EXPECT().AppsV1().Return(k8sClient.AppsV1()).AnyTimes()
			oldVM = &v1.VirtualMachine{
				Spec: v1.VirtualMachineSpec{
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						Spec: v1.VirtualMachineInstanceSpec{
							Domain: v1.DomainSpec{
								Devices: v1.Devices{},
							},
						},
					},
				},
			}
			newVM = &v1.VirtualMachine{
				Spec: v1.VirtualMachineSpec{
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						Spec: v1.VirtualMachineInstanceSpec{
							Domain: v1.DomainSpec{
								Devices: v1.Devices{},
							},
						},
					},
				},
			}
		})
		Context("of InstancetypeMatcher", func() {
			const (
				instancetypeName   = "instancetype"
				instancetypeCRName = "instancetypeCR"
			)
			It("should accept request without changes", func() {
				oldVM.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name:         instancetypeName,
					RevisionName: instancetypeCRName,
					Kind:         apiinstancetype.ClusterSingularResourceName,
				}
				newVM.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name:         instancetypeName,
					RevisionName: instancetypeCRName,
					Kind:         apiinstancetype.ClusterSingularResourceName,
				}
				resp := getResponseFromVMUpdate(oldVM, newVM)
				Expect(resp.Allowed).To(BeTrue())
			})
			It("should accept request when updating name and clearing RevisionName", func() {
				oldVM.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name:         instancetypeName,
					RevisionName: instancetypeCRName,
					Kind:         apiinstancetype.ClusterSingularResourceName,
				}
				newVM.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name: "foo",
					Kind: apiinstancetype.ClusterSingularResourceName,
				}
				resp := getResponseFromVMUpdate(oldVM, newVM)
				Expect(resp.Allowed).To(BeTrue())
			})
			It("should accept request when updating name and RevisionName", func() {
				oldVM.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name:         instancetypeName,
					RevisionName: instancetypeCRName,
					Kind:         apiinstancetype.ClusterSingularResourceName,
				}
				newVM.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name:         "foo",
					RevisionName: "bar",
					Kind:         apiinstancetype.ClusterSingularResourceName,
				}
				resp := getResponseFromVMUpdate(oldVM, newVM)
				Expect(resp.Allowed).To(BeTrue())
			})
			It("should accept request changing RevisionName", func() {
				oldVM.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name:         instancetypeName,
					RevisionName: instancetypeCRName,
					Kind:         apiinstancetype.ClusterSingularResourceName,
				}
				newVM.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name:         instancetypeName,
					RevisionName: "foo",
					Kind:         apiinstancetype.ClusterSingularResourceName,
				}
				resp := getResponseFromVMUpdate(oldVM, newVM)
				Expect(resp.Allowed).To(BeTrue())
			})
			It("should accept request when introducing matcher", func() {
				oldVM.Spec.Instancetype = nil
				newVM.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name:         instancetypeName,
					RevisionName: "foo",
					Kind:         apiinstancetype.ClusterSingularResourceName,
				}
				resp := getResponseFromVMUpdate(oldVM, newVM)
				Expect(resp.Allowed).To(BeTrue())
			})
			It("should accept request when removing matcher", func() {
				oldVM.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name:         instancetypeName,
					RevisionName: instancetypeCRName,
					Kind:         apiinstancetype.ClusterSingularResourceName,
				}
				newVM.Spec.Instancetype = nil
				resp := getResponseFromVMUpdate(oldVM, newVM)
				Expect(resp.Allowed).To(BeTrue())
			})
			It("should reject request changing name without without clearing RevisionName", func() {
				oldVM.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name:         instancetypeName,
					RevisionName: instancetypeCRName,
					Kind:         apiinstancetype.ClusterSingularResourceName,
				}
				newVM.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name:         "foo",
					RevisionName: instancetypeCRName,
					Kind:         apiinstancetype.ClusterSingularResourceName,
				}
				resp := getResponseFromVMUpdate(oldVM, newVM)
				Expect(resp.Allowed).To(BeFalse())
			})
		})
		Context("of PreferenceMatcher", func() {
			const (
				preferenceName   = "preference"
				preferenceCRName = "preferenceCR"
			)
			It("should accept request without changes", func() {
				oldVM.Spec.Preference = &v1.PreferenceMatcher{
					Name:         preferenceName,
					RevisionName: preferenceCRName,
					Kind:         apiinstancetype.ClusterSingularPreferenceResourceName,
				}
				newVM.Spec.Preference = &v1.PreferenceMatcher{
					Name:         preferenceName,
					RevisionName: preferenceCRName,
					Kind:         apiinstancetype.ClusterSingularPreferenceResourceName,
				}
				resp := getResponseFromVMUpdate(oldVM, newVM)
				Expect(resp.Allowed).To(BeTrue())
			})
			It("should accept request when updating name and RevisionName", func() {
				oldVM.Spec.Preference = &v1.PreferenceMatcher{
					Name:         preferenceName,
					RevisionName: preferenceCRName,
					Kind:         apiinstancetype.ClusterSingularPreferenceResourceName,
				}
				newVM.Spec.Preference = &v1.PreferenceMatcher{
					Name:         "foo",
					RevisionName: "bar",
					Kind:         apiinstancetype.ClusterSingularPreferenceResourceName,
				}
				resp := getResponseFromVMUpdate(oldVM, newVM)
				Expect(resp.Allowed).To(BeTrue())
			})
			It("should accept request when updating name and clearing RevisionName", func() {
				oldVM.Spec.Preference = &v1.PreferenceMatcher{
					Name:         preferenceName,
					RevisionName: preferenceCRName,
					Kind:         apiinstancetype.ClusterSingularPreferenceResourceName,
				}
				newVM.Spec.Preference = &v1.PreferenceMatcher{
					Name: "foo",
					Kind: apiinstancetype.ClusterSingularPreferenceResourceName,
				}
				resp := getResponseFromVMUpdate(oldVM, newVM)
				Expect(resp.Allowed).To(BeTrue())
			})
			It("should accept request changing RevisionName", func() {
				oldVM.Spec.Preference = &v1.PreferenceMatcher{
					Name:         preferenceName,
					RevisionName: preferenceCRName,
					Kind:         apiinstancetype.ClusterSingularPreferenceResourceName,
				}
				newVM.Spec.Preference = &v1.PreferenceMatcher{
					Name:         preferenceName,
					RevisionName: "foo",
					Kind:         apiinstancetype.ClusterSingularPreferenceResourceName,
				}
				resp := getResponseFromVMUpdate(oldVM, newVM)
				Expect(resp.Allowed).To(BeTrue())
			})
			It("should accept request when introducing matcher", func() {
				oldVM.Spec.Preference = nil
				newVM.Spec.Preference = &v1.PreferenceMatcher{
					Name:         preferenceName,
					RevisionName: "foo",
					Kind:         apiinstancetype.ClusterSingularPreferenceResourceName,
				}
				resp := getResponseFromVMUpdate(oldVM, newVM)
				Expect(resp.Allowed).To(BeTrue())
			})
			It("should accept request when removing matcher", func() {
				oldVM.Spec.Preference = &v1.PreferenceMatcher{
					Name:         preferenceName,
					RevisionName: preferenceCRName,
					Kind:         apiinstancetype.ClusterSingularPreferenceResourceName,
				}
				newVM.Spec.Preference = nil
				resp := getResponseFromVMUpdate(oldVM, newVM)
				Expect(resp.Allowed).To(BeTrue())
			})
			It("should reject request changing name without clearing RevisionName", func() {
				oldVM.Spec.Preference = &v1.PreferenceMatcher{
					Name:         preferenceName,
					RevisionName: preferenceCRName,
					Kind:         apiinstancetype.ClusterSingularPreferenceResourceName,
				}
				newVM.Spec.Preference = &v1.PreferenceMatcher{
					Name:         "foo",
					RevisionName: preferenceCRName,
					Kind:         apiinstancetype.ClusterSingularPreferenceResourceName,
				}
				resp := getResponseFromVMUpdate(oldVM, newVM)
				Expect(resp.Allowed).To(BeFalse())
			})
		})

		It("should NOT assign new UUID when VM template spec lacks one on update", func() {
			oldVM.Spec.Template.Spec.Domain.Firmware = nil
			newVM.Spec.Template.Spec.Domain.Firmware = nil

			resp := getResponseFromVMUpdate(oldVM, newVM)
			Expect(resp.Allowed).To(BeTrue())

			vmSpec := &v1.VirtualMachineSpec{}
			vmMeta := &k8smetav1.ObjectMeta{}
			patchOps := []patch.PatchOperation{
				{Value: vmSpec},
				{Value: vmMeta},
			}
			err := json.Unmarshal(resp.Patch, &patchOps)
			Expect(err).ToNot(HaveOccurred())
			Expect(patchOps).NotTo(BeEmpty())
			Expect(vmSpec.Template.Spec.Domain.Firmware).To(BeNil())
		})

		It("should preserve existing UUID when VM template spec has one", func() {
			testUUID := types.UID("existing-test-uid")
			newUUID := types.UID("new-test-uid")
			oldVM.Spec.Template.Spec.Domain.Firmware = &v1.Firmware{UUID: testUUID}
			newVM.Spec.Template.Spec.Domain.Firmware = &v1.Firmware{UUID: newUUID}

			resp := getResponseFromVMUpdate(oldVM, newVM)
			Expect(resp.Allowed).To(BeTrue())

			vmSpec := &v1.VirtualMachineSpec{}
			vmMeta := &k8smetav1.ObjectMeta{}
			patchOps := []patch.PatchOperation{
				{Value: vmSpec},
				{Value: vmMeta},
			}
			err := json.Unmarshal(resp.Patch, &patchOps)
			Expect(err).ToNot(HaveOccurred())
			Expect(vmSpec.Template.Spec.Domain.Firmware).ToNot(BeNil())
			Expect(vmSpec.Template.Spec.Domain.Firmware.UUID).To(Equal(newUUID))
		})
	})

	It("should default architecture to compiled architecture when not provided", func() {
		// provide empty string for architecture so that default will apply
		vmSpec, _ := getVMSpecMetaFromResponseCreate()
		Expect(vmSpec.Template.Spec.Architecture).To(Equal(rt.GOARCH))
	})

	It("should allow resource.Quantity fields to accept integer and float values", func() {
		// Raw JSON representation of the VM
		rawVMJSON := []byte(`{
        "apiVersion": "kubevirt.io/v1",
        "kind": "VirtualMachine",
        "metadata": {
            "name": "test-vm",
            "namespace": "default"
        },
        "spec": {
            "template": {
                "spec": {
                    "domain": {
                        "resources": {
                            "requests": {
                                "memory": 22436758,
                                "cpu": "2"
                            },
                            "limits": {
                                "memory": 22436758,
                                "cpu": 2.5
                            }
                        },
                        "memory": {
                            "guest": 2048234
                        },
                        "devices": {
                            "disks": [
                                {
                                    "disk": {
                                        "bus": "virtio"
                                    },
                                    "name": "containerdisk"
                                },
                                {
                                    "disk": {
                                        "bus": "virtio"
                                    },
                                    "name": "cloudinitdisk"
                                }
                            ]
                        }
                    }
                }
            }
        }
    }`)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: k8smetav1.GroupVersionResource{
					Group:    v1.VirtualMachineGroupVersionKind.Group,
					Version:  v1.VirtualMachineGroupVersionKind.Version,
					Resource: "virtualmachines",
				},
				Object: runtime.RawExtension{
					Raw: rawVMJSON,
				},
			},
		}

		resp := mutator.Mutate(ar)
		Expect(resp.Allowed).To(BeTrue())
	})

	Context("failure tests", func() {
		invalidInferFromVolumeFailurePolicy := v1.InferFromVolumeFailurePolicy("not-valid")

		It("should fail if passed resource is not VirtualMachine", func() {
			vmBytes, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())

			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Resource: k8smetav1.GroupVersionResource{Group: "nonexisting.kubevirt.io", Version: "v1", Resource: "nonexistent"},
					Object: runtime.RawExtension{
						Raw: vmBytes,
					},
				},
			}

			resp := mutator.Mutate(ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Code).To(Equal(int32(http.StatusBadRequest)))
			Expect(resp.Result.Message).To(ContainSubstring("expect resource to be"))
		})

		It("should fail if passed json is not VirtualMachine type", func() {
			notVm := struct {
				TestField string `json:"testField"`
			}{
				TestField: "test-string",
			}

			jsonBytes, err := json.Marshal(notVm)
			Expect(err).ToNot(HaveOccurred())

			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Resource: k8smetav1.GroupVersionResource{Group: v1.VirtualMachineGroupVersionKind.Group, Version: v1.VirtualMachineGroupVersionKind.Version, Resource: "virtualmachines"},
					Object: runtime.RawExtension{
						Raw: jsonBytes,
					},
				},
			}

			resp := mutator.Mutate(ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Code).To(Equal(int32(http.StatusUnprocessableEntity)))
		})

		DescribeTable("should fail if", func(instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher, expectedField, expectedMessage string) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			resp := admitVM(admissionv1.Create)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(ContainSubstring(expectedMessage))
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal(expectedField))
		},
			Entry("InstancetypeMatcher does not provide Name or InferFromVolume", &v1.InstancetypeMatcher{}, nil, k8sfield.NewPath("spec", "instancetype").String(), "Either Name or InferFromVolume should be provided within the InstancetypeMatcher"),
			Entry("InstancetypeMatcher provides Name and InferFromVolume", &v1.InstancetypeMatcher{Name: "foo", InferFromVolume: "bar"}, nil, k8sfield.NewPath("spec", "instancetype", "name").String(), "Name should not be provided when InferFromVolume is used within the InstancetypeMatcher"),
			Entry("InstancetypeMatcher provides Kind and InferFromVolume", &v1.InstancetypeMatcher{Kind: "foo", InferFromVolume: "bar"}, nil, k8sfield.NewPath("spec", "instancetype", "kind").String(), "Kind should not be provided when InferFromVolume is used within the InstancetypeMatcher"),
			Entry("InstancetypeMatcher provides invalid value to InferFromVolumeFailurePolicy", &v1.InstancetypeMatcher{InferFromVolume: "bar", InferFromVolumeFailurePolicy: &invalidInferFromVolumeFailurePolicy}, nil, k8sfield.NewPath("spec", "instancetype", "inferFromVolumeFailurePolicy").String(), "Invalid value 'not-valid' for InferFromVolumeFailurePolicy"),
			Entry("PreferenceMatcher does not provide Name or InferFromVolume", nil, &v1.PreferenceMatcher{}, k8sfield.NewPath("spec", "preference").String(), "Either Name or InferFromVolume should be provided within the PreferenceMatcher"),
			Entry("PreferenceMatcher provides Name and InferFromVolume", nil, &v1.PreferenceMatcher{Name: "foo", InferFromVolume: "bar"}, k8sfield.NewPath("spec", "preference", "name").String(), "Name should not be provided when InferFromVolume is used within the PreferenceMatcher"),
			Entry("PreferenceMatcher provides Kind and InferFromVolume", nil, &v1.PreferenceMatcher{Kind: "foo", InferFromVolume: "bar"}, k8sfield.NewPath("spec", "preference", "kind").String(), "Kind should not be provided when InferFromVolume is used within the PreferenceMatcher"),
			Entry("PreferenceMatcher provides invalid value to InferFromVolumeFailurePolicy", nil, &v1.PreferenceMatcher{InferFromVolume: "bar", InferFromVolumeFailurePolicy: &invalidInferFromVolumeFailurePolicy}, k8sfield.NewPath("spec", "preference", "inferFromVolumeFailurePolicy").String(), "Invalid value 'not-valid' for InferFromVolumeFailurePolicy"),
		)
	})
})
