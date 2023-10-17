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
 * Copyright 2019 Red Hat, Inc.
 */

package mutators

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	rt "runtime"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype"

	cdifake "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned/fake"
	fakeclientset "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	instancetypeclientset "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

var _ = Describe("VirtualMachine Mutator", func() {
	var vm *v1.VirtualMachine
	var kvInformer cache.SharedIndexInformer
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

	admitVM := func(arch string) *admissionv1.AdmissionResponse {
		vm.Spec.Template.Spec.Architecture = arch
		vmBytes, err := json.Marshal(vm)
		Expect(err).ToNot(HaveOccurred())
		By("Creating the test admissions review from the VM")
		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: k8smetav1.GroupVersionResource{Group: v1.VirtualMachineGroupVersionKind.Group, Version: v1.VirtualMachineGroupVersionKind.Version, Resource: "virtualmachines"},
				Object: runtime.RawExtension{
					Raw: vmBytes,
				},
			},
		}
		By("Mutating the VM")
		return mutator.Mutate(ar)
	}

	getVMSpecMetaFromResponse := func(arch string) (*v1.VirtualMachineSpec, *k8smetav1.ObjectMeta) {
		resp := admitVM(arch)
		Expect(resp.Allowed).To(BeTrue())

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
		mutator.ClusterConfig, _, kvInformer = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})

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

		mutator.InstancetypeMethods = &instancetype.InstancetypeMethods{Clientset: virtClient}
	})

	It("should allow VM being deleted without applying mutations", func() {
		now := k8smetav1.Now()
		vm.ObjectMeta.DeletionTimestamp = &now
		resp := admitVM(rt.GOARCH)
		Expect(resp.Allowed).To(BeTrue())
		Expect(resp.Patch).To(BeEmpty())
	})

	It("should apply defaults on VM create", func() {
		vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
		if webhooks.IsPPC64(&vmSpec.Template.Spec) {
			Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal("pseries"))
		} else if webhooks.IsARM64(&vmSpec.Template.Spec) {
			Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal("virt"))
		} else {
			Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal("q35"))
		}
	})

	DescribeTable("should apply configurable defaults on VM create", func(arch string, amd64MachineType string, arm64MachineType string, ppcle64MachineType string, result string) {
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					ArchitectureConfiguration: &v1.ArchConfiguration{
						Amd64:   &v1.ArchSpecificConfiguration{MachineType: amd64MachineType},
						Arm64:   &v1.ArchSpecificConfiguration{MachineType: arm64MachineType},
						Ppc64le: &v1.ArchSpecificConfiguration{MachineType: ppcle64MachineType},
					},
				},
			},
		})

		vmSpec, _ := getVMSpecMetaFromResponse(arch)
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(machineTypeFromConfig))

	},
		Entry("when override is for amd64 architecture", "amd64", machineTypeFromConfig, "", "", machineTypeFromConfig),
		Entry("when override is for arm64 architecture", "arm64", "", machineTypeFromConfig, "", machineTypeFromConfig),
		Entry("when override is for ppc64le architecture", "ppc64le", "", "", machineTypeFromConfig, machineTypeFromConfig),
	)

	It("should not override default architecture with defaults on VM create", func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
			Status: v1.KubeVirtStatus{
				DefaultArchitecture: "arm64",
			},
		})

		vmSpec, _ := getVMSpecMetaFromResponse("amd64")
		Expect(vmSpec.Template.Spec.Architecture).To(Equal("amd64"))

	})

	It("should not override specified properties with defaults on VM create", func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					MachineType: machineTypeFromConfig,
				},
			},
		})

		vm.Spec.Template.Spec.Domain.Machine = &v1.Machine{Type: "pc-q35-2.0"}

		vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(vm.Spec.Template.Spec.Domain.Machine.Type))
	})

	It("should not override user specified MachineType with PreferredMachineType or cluster config on VM create", func() {
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

		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					MachineType: machineTypeFromConfig,
				},
			},
		})

		vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(vm.Spec.Template.Spec.Domain.Machine.Type))
	})

	It("should use PreferredMachineType over cluster config on VM create", func() {
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

		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					MachineType: machineTypeFromConfig,
				},
			},
		})

		vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(preference.Spec.Machine.PreferredMachineType))
	})

	It("should ignore error looking up preference and apply cluster config on VM create", func() {
		vm.Spec.Preference = &v1.PreferenceMatcher{
			Name: "foobar",
			Kind: apiinstancetype.SingularPreferenceResourceName,
		}

		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					ArchitectureConfiguration: &v1.ArchConfiguration{
						Amd64:   &v1.ArchSpecificConfiguration{MachineType: machineTypeFromConfig},
						Arm64:   &v1.ArchSpecificConfiguration{MachineType: machineTypeFromConfig},
						Ppc64le: &v1.ArchSpecificConfiguration{MachineType: machineTypeFromConfig},
					},
				},
			},
		})

		vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(machineTypeFromConfig))
	})

	It("should default instancetype kind to ClusterSingularResourceName when not provided", func() {
		vm.Spec.Instancetype = &v1.InstancetypeMatcher{
			Name: "foobar",
		}
		vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
		Expect(vmSpec.Instancetype.Kind).To(Equal(apiinstancetype.ClusterSingularResourceName))
	})

	It("should default preference kind to ClusterSingularPreferenceResourceName when not provided", func() {
		vm.Spec.Preference = &v1.PreferenceMatcher{
			Name: "foobar",
		}
		vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
		Expect(vmSpec.Preference.Kind).To(Equal(apiinstancetype.ClusterSingularPreferenceResourceName))
	})

	It("should use PreferredMachineType from ClusterSingularPreferenceResourceName when no preference kind is provided", func() {
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

		vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(preference.Spec.Machine.PreferredMachineType))
	})

	DescribeTable("should admit valid values to InferFromVolumePolicy", func(instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher) {
		vm.Spec.Instancetype = instancetypeMatcher
		vm.Spec.Preference = preferenceMatcher
		resp := admitVM(rt.GOARCH)
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
				Spec: v1beta1.DataVolumeSpec{},
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

		It("should apply PreferredStorageClassName to PVC", func() {
			vm.Spec.DataVolumeTemplates[0].Spec.PVC = &k8sv1.PersistentVolumeClaimSpec{}
			vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
			assertPVCStorageClassName(vmSpec.DataVolumeTemplates, preference.Spec.Volumes.PreferredStorageClassName)
		})

		It("should apply PreferredStorageClassName to Storage", func() {
			vm.Spec.DataVolumeTemplates[0].Spec.Storage = &v1beta1.StorageSpec{}
			vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
			assertStorageStorageClassName(vmSpec.DataVolumeTemplates, preference.Spec.Volumes.PreferredStorageClassName)
		})

		It("should not fail if DataVolumeSpec PersistentVolumeClaimSpec is nil - bug #9868", func() {
			vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{{
				Spec: v1beta1.DataVolumeSpec{},
			}}
			resp := admitVM(rt.GOARCH)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("should not overwrite storageclass already defined in PVC of DataVolumeTemplate", func() {
			storageClass := "local"
			vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{{
				Spec: v1beta1.DataVolumeSpec{
					PVC: &k8sv1.PersistentVolumeClaimSpec{
						StorageClassName: &storageClass,
					},
				},
			}}
			vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
			assertPVCStorageClassName(vmSpec.DataVolumeTemplates, storageClass)
		})

		It("should not overwrite storageclass already defined in Storage of DataVolumeTemplate", func() {
			storageClass := "local"
			vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{{
				Spec: v1beta1.DataVolumeSpec{
					Storage: &v1beta1.StorageSpec{
						StorageClassName: &storageClass,
					},
				},
			}}
			vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
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
	})

	Context("with InferFromVolume enabled", func() {

		var (
			pvc               *k8sv1.PersistentVolumeClaim
			dvWithSourcePVC   *v1beta1.DataVolume
			dvWithAnnotations *v1beta1.DataVolume
			dsWithSourcePVC   *v1beta1.DataSource
			dsWithAnnotations *v1beta1.DataSource
		)

		const (
			inferVolumeName           = "inferVolumeName"
			defaultInferedNameFromPVC = "defaultInferedNameFromPVC"
			defaultInferedKindFromPVC = "defaultInferedKindFromPVC"
			defaultInferedNameFromDV  = "defaultInferedNameFromDV"
			defaultInferedKindFromDV  = "defaultInferedKindFromDV"
			defaultInferedNameFromDS  = "defaultInferedNameFromDS"
			defaultInferedKindFromDS  = "defaultInferedKindFromDS"
			pvcName                   = "pvcName"
			dvWithSourcePVCName       = "dvWithSourcePVCName"
			dsWithSourcePVCName       = "dsWithSourcePVCName"
			dsWithAnnotationsName     = "dsWithAnnotationsName"
			unknownPVCName            = "unknownPVCName"
			unknownDVName             = "unknownDVName"
		)

		BeforeEach(func() {
			pvc = &k8sv1.PersistentVolumeClaim{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      pvcName,
					Namespace: vm.Namespace,
					Labels: map[string]string{
						apiinstancetype.DefaultInstancetypeLabel:     defaultInferedNameFromPVC,
						apiinstancetype.DefaultInstancetypeKindLabel: defaultInferedKindFromPVC,
						apiinstancetype.DefaultPreferenceLabel:       defaultInferedNameFromPVC,
						apiinstancetype.DefaultPreferenceKindLabel:   defaultInferedKindFromPVC,
					},
				},
			}
			pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Create(context.Background(), pvc, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			dvWithSourcePVC = &v1beta1.DataVolume{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      dvWithSourcePVCName,
					Namespace: vm.Namespace,
				},
				Spec: v1beta1.DataVolumeSpec{
					Source: &v1beta1.DataVolumeSource{
						PVC: &v1beta1.DataVolumeSourcePVC{
							Name:      pvc.Name,
							Namespace: pvc.Namespace,
						},
					},
				},
			}
			dvWithSourcePVC, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(context.Background(), dvWithSourcePVC, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			dsWithSourcePVC = &v1beta1.DataSource{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      dsWithSourcePVCName,
					Namespace: vm.Namespace,
				},
				Spec: v1beta1.DataSourceSpec{
					Source: v1beta1.DataSourceSource{
						PVC: &v1beta1.DataVolumeSourcePVC{
							Name:      pvc.Name,
							Namespace: pvc.Namespace,
						},
					},
				},
			}
			dsWithSourcePVC, err = virtClient.CdiClient().CdiV1beta1().DataSources(vm.Namespace).Create(context.Background(), dsWithSourcePVC, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			dsWithAnnotations = &v1beta1.DataSource{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      dsWithAnnotationsName,
					Namespace: vm.Namespace,
					Labels: map[string]string{
						apiinstancetype.DefaultInstancetypeLabel:     defaultInferedNameFromDS,
						apiinstancetype.DefaultInstancetypeKindLabel: defaultInferedKindFromDS,
						apiinstancetype.DefaultPreferenceLabel:       defaultInferedNameFromDS,
						apiinstancetype.DefaultPreferenceKindLabel:   defaultInferedKindFromDS,
					},
				},
				Spec: v1beta1.DataSourceSpec{
					Source: v1beta1.DataSourceSource{
						PVC: &v1beta1.DataVolumeSourcePVC{
							Name:      pvc.Name,
							Namespace: pvc.Namespace,
						},
					},
				},
			}
			_, err = virtClient.CdiClient().CdiV1beta1().DataSources(vm.Namespace).Create(context.Background(), dsWithAnnotations, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		DescribeTable("should infer defaults from VolumeSource and PersistentVolumeClaim", func(instancetypeMatcher, expectedInstancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher, expectedPreferenceMatcher *v1.PreferenceMatcher) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvc.Name,
						},
					},
				},
			}}
			vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
			Expect(vmSpec.Instancetype).To(Equal(expectedInstancetypeMatcher))
			Expect(vmSpec.Preference).To(Equal(expectedPreferenceMatcher))
		},
			Entry("for InstancetypeMatcher",
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.InstancetypeMatcher{
					Name: defaultInferedNameFromPVC,
					Kind: defaultInferedKindFromPVC,
				}, nil, nil,
			),
			Entry("for PreferenceMatcher",
				nil, nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.PreferenceMatcher{
					Name: defaultInferedNameFromPVC,
					Kind: defaultInferedKindFromPVC,
				},
			),
		)

		DescribeTable("should infer defaults from DataVolumeSource and PersistentVolumeClaim", func(instancetypeMatcher, expectedInstancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher, expectedPreferenceMatcher *v1.PreferenceMatcher) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: dvWithSourcePVCName,
					},
				},
			}}

			vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
			Expect(vmSpec.Instancetype).To(Equal(expectedInstancetypeMatcher))
			Expect(vmSpec.Preference).To(Equal(expectedPreferenceMatcher))
		},
			Entry("for InstancetypeMatcher",
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.InstancetypeMatcher{
					Name: defaultInferedNameFromPVC,
					Kind: defaultInferedKindFromPVC,
				}, nil, nil,
			),
			Entry("for PreferenceMatcher",
				nil, nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.PreferenceMatcher{
					Name: defaultInferedNameFromPVC,
					Kind: defaultInferedKindFromPVC,
				},
			),
		)

		DescribeTable("should infer defaults from DataVolumeTemplate, DataVolumeSourcePVC and PersistentVolumeClaim", func(instancetypeMatcher, expectedInstancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher, expectedPreferenceMatcher *v1.PreferenceMatcher) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dataVolume",
					},
				},
			}}
			vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name: "dataVolume",
				},
				Spec: v1beta1.DataVolumeSpec{
					Source: &v1beta1.DataVolumeSource{
						PVC: &v1beta1.DataVolumeSourcePVC{
							Name:      pvc.Name,
							Namespace: pvc.Namespace,
						},
					},
				},
			}}

			vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
			Expect(vmSpec.Instancetype).To(Equal(expectedInstancetypeMatcher))
			Expect(vmSpec.Preference).To(Equal(expectedPreferenceMatcher))
		},
			Entry("for InstancetypeMatcher",
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.InstancetypeMatcher{
					Name: defaultInferedNameFromPVC,
					Kind: defaultInferedKindFromPVC,
				}, nil, nil,
			),
			Entry("for PreferenceMatcher",
				nil, nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.PreferenceMatcher{
					Name: defaultInferedNameFromPVC,
					Kind: defaultInferedKindFromPVC,
				},
			),
		)
		DescribeTable("should infer defaults from DataVolume with labels", func(instancetypeMatcher, expectedInstancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher, expectedPreferenceMatcher *v1.PreferenceMatcher) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			dvWithAnnotations = &v1beta1.DataVolume{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "dvWithAnnotations",
					Namespace: vm.Namespace,
					Labels: map[string]string{
						apiinstancetype.DefaultInstancetypeLabel:     defaultInferedNameFromDV,
						apiinstancetype.DefaultInstancetypeKindLabel: defaultInferedKindFromDV,
						apiinstancetype.DefaultPreferenceLabel:       defaultInferedNameFromDV,
						apiinstancetype.DefaultPreferenceKindLabel:   defaultInferedKindFromDV,
					},
				},
				Spec: v1beta1.DataVolumeSpec{
					Source: &v1beta1.DataVolumeSource{
						PVC: &v1beta1.DataVolumeSourcePVC{
							Name:      pvc.Name,
							Namespace: pvc.Namespace,
						},
					},
				},
			}
			_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(context.Background(), dvWithAnnotations, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: dvWithAnnotations.Name,
					},
				},
			})

			vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
			Expect(vmSpec.Instancetype).To(Equal(expectedInstancetypeMatcher))
			Expect(vmSpec.Preference).To(Equal(expectedPreferenceMatcher))
		},
			Entry("for InstancetypeMatcher",
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.InstancetypeMatcher{
					Name: defaultInferedNameFromDV,
					Kind: defaultInferedKindFromDV,
				}, nil, nil,
			),
			Entry("for PreferenceMatcher",
				nil, nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.PreferenceMatcher{
					Name: defaultInferedNameFromDV,
					Kind: defaultInferedKindFromDV,
				},
			),
		)

		DescribeTable("should infer defaults from DataVolume, DataVolumeSourceRef", func(sourceRefName, sourceRefKind, sourceRefNamespace string, instancetypeMatcher, expectedInstancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher, expectedPreferenceMatcher *v1.PreferenceMatcher) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			var sourceRefNamespacePointer *string
			if sourceRefNamespace != "" {
				sourceRefNamespacePointer = &sourceRefNamespace
			}
			dvWithSourceRef := &v1beta1.DataVolume{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "dvWithSourceRef",
					Namespace: vm.Namespace,
				},
				Spec: v1beta1.DataVolumeSpec{
					SourceRef: &v1beta1.DataVolumeSourceRef{
						Name:      sourceRefName,
						Kind:      sourceRefKind,
						Namespace: sourceRefNamespacePointer,
					},
				},
			}
			_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(context.Background(), dvWithSourceRef, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: dvWithSourceRef.Name,
					},
				},
			}}

			vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
			Expect(vmSpec.Instancetype).To(Equal(expectedInstancetypeMatcher))
			Expect(vmSpec.Preference).To(Equal(expectedPreferenceMatcher))
		},
			Entry(",DataSource and PersistentVolumeClaim for InstancetypeMatcher",
				dsWithSourcePVCName, "DataSource", k8sv1.NamespaceDefault,
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.InstancetypeMatcher{
					Name: defaultInferedNameFromPVC,
					Kind: defaultInferedKindFromPVC,
				}, nil, nil,
			),
			Entry(",DataSource and PersistentVolumeClaim for PreferenceMatcher",
				dsWithSourcePVCName, "DataSource", k8sv1.NamespaceDefault,
				nil, nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.PreferenceMatcher{
					Name: defaultInferedNameFromPVC,
					Kind: defaultInferedKindFromPVC,
				},
			),
			Entry("and DataSource with annotations for InstancetypeMatcher",
				dsWithAnnotationsName, "DataSource", k8sv1.NamespaceDefault,
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.InstancetypeMatcher{
					Name: defaultInferedNameFromDS,
					Kind: defaultInferedKindFromDS,
				}, nil, nil,
			),
			Entry("and DataSource with annotations for PreferenceMatcher",
				dsWithAnnotationsName, "DataSource", k8sv1.NamespaceDefault,
				nil, nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.PreferenceMatcher{
					Name: defaultInferedNameFromDS,
					Kind: defaultInferedKindFromDS,
				},
			),
			Entry(",DataSource without namespace and PersistentVolumeClaim for InstancetypeMatcher",
				dsWithSourcePVCName, "DataSource", "",
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.InstancetypeMatcher{
					Name: defaultInferedNameFromPVC,
					Kind: defaultInferedKindFromPVC,
				}, nil, nil,
			),
			Entry(",DataSource without namespace and PersistentVolumeClaim for PreferenceMatcher",
				dsWithSourcePVCName, "DataSource", "",
				nil, nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.PreferenceMatcher{
					Name: defaultInferedNameFromPVC,
					Kind: defaultInferedKindFromPVC,
				},
			),
			Entry("and DataSource without namespace with annotations for InstancetypeMatcher",
				dsWithAnnotationsName, "DataSource", "",
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.InstancetypeMatcher{
					Name: defaultInferedNameFromDS,
					Kind: defaultInferedKindFromDS,
				}, nil, nil,
			),
			Entry("and DataSource without namespace with annotations for PreferenceMatcher",
				dsWithAnnotationsName, "DataSource", "",
				nil, nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.PreferenceMatcher{
					Name: defaultInferedNameFromDS,
					Kind: defaultInferedKindFromDS,
				},
			),
		)

		DescribeTable("should infer defaults from DataVolumeTemplate, DataVolumeSourceRef, DataSource and PersistentVolumeClaim", func(sourceRefName, sourceRefNamespace string, instancetypeMatcher, expectedInstancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher, expectedPreferenceMatcher *v1.PreferenceMatcher) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name: "dataVolume",
				},
				Spec: v1beta1.DataVolumeSpec{
					SourceRef: &v1beta1.DataVolumeSourceRef{
						Name:      sourceRefName,
						Kind:      "DataSource",
						Namespace: &sourceRefNamespace,
					},
				},
			}}
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dataVolume",
					},
				},
			}}

			vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
			Expect(vmSpec.Instancetype).To(Equal(expectedInstancetypeMatcher))
			Expect(vmSpec.Preference).To(Equal(expectedPreferenceMatcher))
		},
			Entry("for InstancetypeMatcher",
				dsWithSourcePVCName, k8sv1.NamespaceDefault,
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.InstancetypeMatcher{
					Name: defaultInferedNameFromPVC,
					Kind: defaultInferedKindFromPVC,
				}, nil, nil,
			),
			Entry("for PreferenceMatcher",
				dsWithSourcePVCName, k8sv1.NamespaceDefault,
				nil, nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.PreferenceMatcher{
					Name: defaultInferedNameFromPVC,
					Kind: defaultInferedKindFromPVC,
				},
			),
			Entry("and DataSource with annotations for InstancetypeMatcher",
				dsWithAnnotationsName, k8sv1.NamespaceDefault,
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.InstancetypeMatcher{
					Name: defaultInferedNameFromDS,
					Kind: defaultInferedKindFromDS,
				}, nil, nil,
			),
			Entry("and DataSource with annotations for PreferenceMatcher",
				dsWithAnnotationsName, k8sv1.NamespaceDefault,
				nil, nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				},
				&v1.PreferenceMatcher{
					Name: defaultInferedNameFromDS,
					Kind: defaultInferedKindFromDS,
				},
			),
		)

		DescribeTable("should fail to infer defaults from unknown Volume ", func(instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher

			// Remove all volumes to cause the failure
			vm.Spec.Template.Spec.Volumes = []v1.Volume{}

			resp := admitVM(rt.GOARCH)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(ContainSubstring("unable to find volume %s to infer defaults", inferVolumeName))
		},
			Entry("for InstancetypeMatcher",
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				}, nil,
			),
			Entry("for InstancetypeMatcher with IgnoreInferFromVolumeFailure",
				&v1.InstancetypeMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &ignoreInferFromVolumeFailure,
				}, nil,
			),
			Entry("for InstancetypeMatcher with RejectInferFromVolumeFailure",
				&v1.InstancetypeMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &rejectInferFromVolumeFailure,
				}, nil,
			),
			Entry("for PreferenceMatcher",
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				},
			),
			Entry("for PreferenceMatcher with IgnoreInferFromVolumeFailure",
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &ignoreInferFromVolumeFailure,
				},
			),
			Entry("for PreferenceMatcher with RejectInferFromVolumeFailure",
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &rejectInferFromVolumeFailure,
				},
			),
		)

		DescribeTable("should fail to infer defaults from Volume ", func(volumeSource v1.VolumeSource, messageSubstring string, instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher, allowed bool) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name:         inferVolumeName,
				VolumeSource: volumeSource,
			}}
			resp := admitVM(rt.GOARCH)

			Expect(resp.Allowed).To(Equal(allowed))
			if allowed {
				// Expect matchers to be cleared on failure during inference
				vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
				Expect(vmSpec.Instancetype).To(BeNil())
				Expect(vmSpec.Preference).To(BeNil())
			} else {
				Expect(resp.Result.Message).To(ContainSubstring(messageSubstring))
			}
		},
			Entry("with unknown PersistentVolumeClaim for InstancetypeMatcher",
				v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: unknownPVCName,
						},
					},
				},
				fmt.Sprintf("persistentvolumeclaims \"%s\" not found", unknownPVCName),
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				}, nil, false,
			),
			Entry("with unknown PersistentVolumeClaim for InstancetypeMatcher with IgnoreInferFromVolumeFailure",
				v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: unknownPVCName,
						},
					},
				},
				fmt.Sprintf("persistentvolumeclaims \"%s\" not found", unknownPVCName),
				&v1.InstancetypeMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &ignoreInferFromVolumeFailure,
				}, nil, false,
			),
			Entry("with unknown PersistentVolumeClaim for InstancetypeMatcher with RejectInferFromVolumeFailure",
				v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: unknownPVCName,
						},
					},
				},
				fmt.Sprintf("persistentvolumeclaims \"%s\" not found", unknownPVCName),
				&v1.InstancetypeMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &rejectInferFromVolumeFailure,
				}, nil, false,
			),
			Entry("with unknown PersistentVolumeClaim for PreferenceMatcher",
				v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: unknownPVCName,
						},
					},
				},
				fmt.Sprintf("persistentvolumeclaims \"%s\" not found", unknownPVCName),
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				}, false,
			),
			Entry("with unknown PersistentVolumeClaim for PreferenceMatcher with IgnoreInferFromVolumeFailure",
				v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: unknownPVCName,
						},
					},
				},
				fmt.Sprintf("persistentvolumeclaims \"%s\" not found", unknownPVCName),
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &ignoreInferFromVolumeFailure,
				}, false,
			),
			Entry("with unknown PersistentVolumeClaim for PreferenceMatcher with RejectInferFromVolumeFailure",
				v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: unknownPVCName,
						},
					},
				},
				fmt.Sprintf("persistentvolumeclaims \"%s\" not found", unknownPVCName),
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &rejectInferFromVolumeFailure,
				}, false,
			),
			Entry("with unknown DataVolume and PersistentVolumeClaim for InstancetypeMatcher",
				v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: unknownDVName,
					},
				}, fmt.Sprintf("persistentvolumeclaims \"%s\" not found", unknownDVName),
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				}, nil, false,
			),
			Entry("with unknown DataVolume and PersistentVolumeClaim for InstancetypeMatcher with IgnoreInferFromVolumeFailure",
				v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: unknownDVName,
					},
				}, fmt.Sprintf("persistentvolumeclaims \"%s\" not found", unknownDVName),
				&v1.InstancetypeMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &ignoreInferFromVolumeFailure,
				}, nil, false,
			),
			Entry("with unknown DataVolume and PersistentVolumeClaim for InstancetypeMatcher with RejectInferFromVolumeFailure",
				v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: unknownDVName,
					},
				}, fmt.Sprintf("persistentvolumeclaims \"%s\" not found", unknownDVName),
				&v1.InstancetypeMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &rejectInferFromVolumeFailure,
				}, nil, false,
			),
			Entry("with unknown DataVolume and PersistentVolumeClaim for PreferenceMatcher",
				v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: unknownDVName,
					},
				}, fmt.Sprintf("persistentvolumeclaims \"%s\" not found", unknownDVName),
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				}, false,
			),
			Entry("with unknown DataVolume and PersistentVolumeClaim for PreferenceMatcher with IgnoreInferFromVolumeFailure",
				v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: unknownDVName,
					},
				}, fmt.Sprintf("persistentvolumeclaims \"%s\" not found", unknownDVName),
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &ignoreInferFromVolumeFailure,
				}, false,
			),
			Entry("with unknown DataVolume and PersistentVolumeClaim for PreferenceMatcher with RejectInferFromVolumeFailure",
				v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: unknownDVName,
					},
				}, fmt.Sprintf("persistentvolumeclaims \"%s\" not found", unknownDVName),
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &rejectInferFromVolumeFailure,
				}, false,
			),
			Entry("with unsupported VolumeSource type for InstancetypeMatcher",
				v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{},
				},
				fmt.Sprintf("unable to infer defaults from volume %s as type is not supported", inferVolumeName),
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				}, nil, false,
			),
			Entry("but still admit with unsupported VolumeSource type for InstancetypeMatcher with IgnoreInferFromVolumeFailure",
				v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{},
				}, "",
				&v1.InstancetypeMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &ignoreInferFromVolumeFailure,
				}, nil, true,
			),
			Entry("with unsupported VolumeSource type for InstancetypeMatcher with RejectInferFromVolumeFailure",
				v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{},
				},
				fmt.Sprintf("unable to infer defaults from volume %s as type is not supported", inferVolumeName),
				&v1.InstancetypeMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &rejectInferFromVolumeFailure,
				}, nil, false,
			),
			Entry("with unsupported VolumeSource type for PreferenceMatcher",
				v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{},
				},
				fmt.Sprintf("unable to infer defaults from volume %s as type is not supported", inferVolumeName),
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				}, false,
			),
			Entry("but still admit with unsupported VolumeSource type for PreferenceMatcher with IgnoreInferFromVolumeFailure",
				v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{},
				}, "", nil,
				&v1.PreferenceMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &ignoreInferFromVolumeFailure,
				}, true,
			),
			Entry("with unsupported VolumeSource type for PreferenceMatcher with RejectInferFromVolumeFailure",
				v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{},
				},
				fmt.Sprintf("unable to infer defaults from volume %s as type is not supported", inferVolumeName),
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &rejectInferFromVolumeFailure,
				}, false,
			),
		)

		DescribeTable("should fail to infer defaults from DataVolume with an unsupported DataVolumeSource", func(instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher, allowed bool) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			dvWithUnsupportedSource := &v1beta1.DataVolume{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "dvWithSourceRef",
					Namespace: vm.Namespace,
				},
				Spec: v1beta1.DataVolumeSpec{
					Source: &v1beta1.DataVolumeSource{
						VDDK: &v1beta1.DataVolumeSourceVDDK{},
					},
				},
			}
			_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(context.Background(), dvWithUnsupportedSource, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: dvWithUnsupportedSource.Name,
					},
				},
			}}
			resp := admitVM(rt.GOARCH)
			Expect(resp.Allowed).To(Equal(allowed))
			if allowed {
				// Expect matchers to be cleared on failure during inference
				vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
				Expect(vmSpec.Instancetype).To(BeNil())
				Expect(vmSpec.Preference).To(BeNil())
			} else {
				Expect(resp.Result.Message).To(ContainSubstring("unable to infer defaults from DataVolumeSpec as DataVolumeSource is not supported"))
			}
		},
			Entry("for InstancetypeMatcher",
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				}, nil, false,
			),
			Entry("but still admit for InstancetypeMatcher with IgnoreInferFromVolumeFailure",
				&v1.InstancetypeMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &ignoreInferFromVolumeFailure,
				}, nil, true,
			),
			Entry("for InstancetypeMatcher with RejectInferFromVolumeFailure",
				&v1.InstancetypeMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &rejectInferFromVolumeFailure,
				}, nil, false,
			),
			Entry("for PreferenceMatcher",
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				}, false,
			),
			Entry("but still admit for PreferenceMatcher with IgnoreInferFromVolumeFailure",
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &ignoreInferFromVolumeFailure,
				}, true,
			),
			Entry("for PreferenceMatcher with RejectInferFromVolumeFailure",
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &rejectInferFromVolumeFailure,
				}, false,
			),
		)

		DescribeTable("should fail to infer defaults from DataVolume with an unknown DataVolumeSourceRef Kind", func(instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher, allowed bool) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			dvWithUnknownSourceRefKind := &v1beta1.DataVolume{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "dvWithSourceRef",
					Namespace: vm.Namespace,
				},
				Spec: v1beta1.DataVolumeSpec{
					SourceRef: &v1beta1.DataVolumeSourceRef{
						Kind: "foo",
					},
				},
			}
			_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(context.Background(), dvWithUnknownSourceRefKind, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: dvWithUnknownSourceRefKind.Name,
					},
				},
			}}
			resp := admitVM(rt.GOARCH)
			Expect(resp.Allowed).To(Equal(allowed))
			if allowed {
				// Expect matchers to be cleared on failure during inference
				vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
				Expect(vmSpec.Instancetype).To(BeNil())
				Expect(vmSpec.Preference).To(BeNil())
			} else {
				Expect(resp.Result.Message).To(ContainSubstring("unable to infer defaults from DataVolumeSourceRef as Kind foo is not supported"))
			}
		},
			Entry("for InstancetypeMatcher",
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				}, nil, false,
			),
			Entry("but still admit for InstancetypeMatcher with IgnoreInferFromVolumeFailure",
				&v1.InstancetypeMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &ignoreInferFromVolumeFailure,
				}, nil, true,
			),
			Entry("for InstancetypeMatcher with RejectInferFromVolumeFailure",
				&v1.InstancetypeMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &rejectInferFromVolumeFailure,
				}, nil, false,
			),
			Entry("for PreferenceMatcher",
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				}, false,
			),
			Entry("but still admit for PreferenceMatcher with IgnoreInferFromVolumeFailure",
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &ignoreInferFromVolumeFailure,
				}, true,
			),
			Entry("for PreferenceMatcher with RejectInferFromVolumeFailure",
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &rejectInferFromVolumeFailure,
				}, false,
			),
		)

		DescribeTable("should fail to infer defaults from DataSource missing DataVolumeSourcePVC", func(instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher, allowed bool) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			dsWithoutSourcePVC := &v1beta1.DataSource{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "dsWithoutSourcePVC",
					Namespace: vm.Namespace,
				},
				Spec: v1beta1.DataSourceSpec{
					Source: v1beta1.DataSourceSource{},
				},
			}
			_, err := virtClient.CdiClient().CdiV1beta1().DataSources(vm.Namespace).Create(context.Background(), dsWithoutSourcePVC, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name: "dataVolume",
				},
				Spec: v1beta1.DataVolumeSpec{
					SourceRef: &v1beta1.DataVolumeSourceRef{
						Kind:      "DataSource",
						Name:      dsWithoutSourcePVC.Name,
						Namespace: &dsWithoutSourcePVC.Namespace,
					},
				},
			}}
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dataVolume",
					},
				},
			}}
			resp := admitVM(rt.GOARCH)
			Expect(resp.Allowed).To(Equal(allowed))
			if allowed {
				// Expect matchers to be cleared on failure during inference
				vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
				Expect(vmSpec.Instancetype).To(BeNil())
				Expect(vmSpec.Preference).To(BeNil())
			} else {
				Expect(resp.Result.Message).To(ContainSubstring("unable to infer defaults from DataSource that doesn't provide DataVolumeSourcePVC"))
			}
		},
			Entry("for InstancetypeMatcher",
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				}, nil, false,
			),
			Entry("but still admit for InstancetypeMatcher with IgnoreInferFromVolumeFailure",
				&v1.InstancetypeMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &ignoreInferFromVolumeFailure,
				}, nil, true,
			),
			Entry("for InstancetypeMatcher with RejectInferFromVolumeFailure",
				&v1.InstancetypeMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &rejectInferFromVolumeFailure,
				}, nil, false,
			),
			Entry("for PreferenceMatcher",
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				}, false,
			),
			Entry("but still admit for PreferenceMatcher with IgnoreInferFromVolumeFailure",
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &ignoreInferFromVolumeFailure,
				}, true,
			),
			Entry("for PreferenceMatcher with RejectInferFromVolumeFailure",
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &rejectInferFromVolumeFailure,
				}, false,
			),
		)

		DescribeTable("should fail to infer defaults from PersistentVolumeClaim without default instance type label", func(instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher, requiredLabel string, allowed bool) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			pvcWithoutLabels := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "pvcWithoutLabels",
					Namespace: vm.Namespace,
				},
			}
			_, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Create(context.Background(), pvcWithoutLabels, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcWithoutLabels.Name,
						},
					},
				},
			}}
			resp := admitVM(rt.GOARCH)
			Expect(resp.Allowed).To(Equal(allowed))
			if allowed {
				// Expect matchers to be cleared on failure during inference
				vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
				Expect(vmSpec.Instancetype).To(BeNil())
				Expect(vmSpec.Preference).To(BeNil())
			} else {
				Expect(resp.Result.Message).To(ContainSubstring("unable to find required %s label on the volume", requiredLabel))
			}
		},
			Entry("for InstancetypeMatcher",
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				}, nil, apiinstancetype.DefaultInstancetypeLabel, false,
			),
			Entry("but still admit for InstancetypeMatcher with IgnoreInferFromVolumeFailure",
				&v1.InstancetypeMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &ignoreInferFromVolumeFailure,
				}, nil, apiinstancetype.DefaultInstancetypeLabel, true,
			),
			Entry("for InstancetypeMatcher with RejectInferFromVolumeFailure",
				&v1.InstancetypeMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &rejectInferFromVolumeFailure,
				}, nil, apiinstancetype.DefaultInstancetypeLabel, false,
			),
			Entry("for PreferenceMatcher",
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				}, apiinstancetype.DefaultPreferenceLabel, false,
			),
			Entry("but still admit for PreferenceMatcher with with IgnoreInferFromVolumeFailure",
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &ignoreInferFromVolumeFailure,
				}, apiinstancetype.DefaultPreferenceLabel, true,
			),
			Entry("for PreferenceMatcher with RejectInferFromVolumeFailure",
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume:              inferVolumeName,
					InferFromVolumeFailurePolicy: &rejectInferFromVolumeFailure,
				}, apiinstancetype.DefaultPreferenceLabel, false,
			),
		)

		DescribeTable("should use cluster kind when default kind is not provided while inferring defaults", func(instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			pvcWithoutKindAnnotations := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "pvcWithoutKindAnnotations",
					Namespace: vm.Namespace,
					Labels: map[string]string{
						apiinstancetype.DefaultInstancetypeLabel: defaultInferedNameFromPVC,
						apiinstancetype.DefaultPreferenceLabel:   defaultInferedNameFromPVC,
					},
				},
			}
			_, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Create(context.Background(), pvcWithoutKindAnnotations, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcWithoutKindAnnotations.Name,
						},
					},
				},
			}}
			vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
			if instancetypeMatcher != nil {
				Expect(vmSpec.Instancetype.Kind).To(Equal(apiinstancetype.ClusterSingularResourceName))
			}
			if preferenceMatcher != nil {
				Expect(vmSpec.Preference.Kind).To(Equal(apiinstancetype.ClusterSingularPreferenceResourceName))
			}
		},
			Entry("for InstancetypeMatcher",
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				}, nil,
			),
			Entry("for PreferenceMatcher",
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				},
			),
		)

		It("should infer defaults from garbage collected DataVolume using PVC with the same name", func() {
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				InferFromVolume: inferVolumeName,
			}
			vm.Spec.Preference = &v1.PreferenceMatcher{
				InferFromVolume: inferVolumeName,
			}
			// No DataVolume with the name of pvcName exists but a PVC does
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: pvcName,
					},
				},
			}}
			vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)
			Expect(vmSpec.Instancetype).To(Equal(&v1.InstancetypeMatcher{
				Name: defaultInferedNameFromPVC,
				Kind: defaultInferedKindFromPVC,
			}))
			Expect(vmSpec.Preference).To(Equal(&v1.PreferenceMatcher{
				Name: defaultInferedNameFromPVC,
				Kind: defaultInferedKindFromPVC,
			}))
		})

		DescribeTable("When inference was successful", func(failurePolicy v1.InferFromVolumeFailurePolicy, expectMemoryCleared bool) {
			By("Setting guest memory")
			guestMemory := resource.MustParse("512Mi")
			vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{
				Guest: &guestMemory,
			}

			By("Creating a VM using a PVC as boot and inference Volume")
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: &failurePolicy,
			}
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvc.Name,
						},
					},
				},
			}}
			vmSpec, _ := getVMSpecMetaFromResponse(rt.GOARCH)

			expectedInstancetypeMatcher := &v1.InstancetypeMatcher{
				Name: defaultInferedNameFromPVC,
				Kind: defaultInferedKindFromPVC,
			}
			Expect(vmSpec.Instancetype).To(Equal(expectedInstancetypeMatcher))

			if expectMemoryCleared {
				Expect(vmSpec.Template.Spec.Domain.Memory).To(BeNil())
			} else {
				Expect(vmSpec.Template.Spec.Domain.Memory).ToNot(BeNil())
				Expect(vmSpec.Template.Spec.Domain.Memory.Guest).ToNot(BeNil())
				Expect(*vmSpec.Template.Spec.Domain.Memory.Guest).To(Equal(guestMemory))
			}
		},
			Entry("it should clear guest memory when ignoring inference failures", v1.IgnoreInferFromVolumeFailure, true),
			Entry("it should not clear guest memory when rejecting inference failures", v1.RejectInferFromVolumeFailure, false),
		)
	})

	It("should default architecture to compiled architecture when not provided", func() {
		// provide empty string for architecture so that default will apply
		vmSpec, _ := getVMSpecMetaFromResponse("")
		Expect(vmSpec.Template.Spec.Architecture).To(Equal(rt.GOARCH))
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
			resp := admitVM(rt.GOARCH)
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
