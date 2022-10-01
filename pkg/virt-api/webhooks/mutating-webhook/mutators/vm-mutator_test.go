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
	"net/http"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	"kubevirt.io/client-go/kubecli"

	fakeclientset "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	instancetypeclientset "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/instancetype/v1alpha2"

	"kubevirt.io/kubevirt/pkg/instancetype"
	"kubevirt.io/kubevirt/pkg/testutils"
	utiltypes "kubevirt.io/kubevirt/pkg/util/types"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

var _ = Describe("VirtualMachine Mutator", func() {
	var vm *v1.VirtualMachine
	var kvInformer cache.SharedIndexInformer
	var mutator *VMsMutator
	var ctrl *gomock.Controller
	var virtClient *kubecli.MockKubevirtClient
	var fakeInstancetypeClients instancetypeclientset.InstancetypeV1alpha2Interface
	var fakePreferenceClient instancetypeclientset.VirtualMachinePreferenceInterface
	var fakeClusterPreferenceClient instancetypeclientset.VirtualMachineClusterPreferenceInterface

	machineTypeFromConfig := "pc-q35-3.0"

	getVMSpecMetaFromResponse := func() (*v1.VirtualMachineSpec, *k8smetav1.ObjectMeta) {
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
		resp := mutator.Mutate(ar)
		Expect(resp.Allowed).To(BeTrue())

		By("Getting the VM spec from the response")
		vmSpec := &v1.VirtualMachineSpec{}
		vmMeta := &k8smetav1.ObjectMeta{}
		patch := []utiltypes.PatchOperation{
			{Value: vmSpec},
			{Value: vmMeta},
		}
		err = json.Unmarshal(resp.Patch, &patch)
		Expect(err).ToNot(HaveOccurred())
		Expect(patch).NotTo(BeEmpty())

		return vmSpec, vmMeta
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

		fakeInstancetypeClients = fakeclientset.NewSimpleClientset().InstancetypeV1alpha2()
		fakePreferenceClient = fakeInstancetypeClients.VirtualMachinePreferences(vm.Namespace)
		fakeClusterPreferenceClient = fakeInstancetypeClients.VirtualMachineClusterPreferences()
		virtClient.EXPECT().VirtualMachinePreference(gomock.Any()).Return(fakePreferenceClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineClusterPreference().Return(fakeClusterPreferenceClient).AnyTimes()

		mutator.InstancetypeMethods = instancetype.NewMethods(virtClient)
	})

	It("should apply defaults on VM create", func() {
		vmSpec, _ := getVMSpecMetaFromResponse()
		if webhooks.IsPPC64() {
			Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal("pseries"))
		} else if webhooks.IsARM64() {
			Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal("virt"))
		} else {
			Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal("q35"))
		}
	})

	It("should apply configurable defaults on VM create", func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					MachineType: machineTypeFromConfig,
				},
			},
		})

		vmSpec, _ := getVMSpecMetaFromResponse()
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(machineTypeFromConfig))
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

		vmSpec, _ := getVMSpecMetaFromResponse()
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(vm.Spec.Template.Spec.Domain.Machine.Type))
	})

	It("should not override user specified MachineType with PreferredMachineType or cluster config on VM create", func() {
		vm.Spec.Template.Spec.Domain.Machine = &v1.Machine{Type: "pc-q35-2.0"}
		preference := &instancetypev1alpha2.VirtualMachinePreference{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name: "machineTypePreference",
			},
			TypeMeta: k8smetav1.TypeMeta{
				Kind:       apiinstancetype.SingularPreferenceResourceName,
				APIVersion: instancetypev1alpha2.SchemeGroupVersion.String(),
			},
			Spec: instancetypev1alpha2.VirtualMachinePreferenceSpec{
				Machine: &instancetypev1alpha2.MachinePreferences{
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

		vmSpec, _ := getVMSpecMetaFromResponse()
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(vm.Spec.Template.Spec.Domain.Machine.Type))
	})

	It("should use PreferredMachineType over cluster config on VM create", func() {
		preference := &instancetypev1alpha2.VirtualMachinePreference{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name: "machineTypePreference",
			},
			TypeMeta: k8smetav1.TypeMeta{
				Kind:       apiinstancetype.SingularPreferenceResourceName,
				APIVersion: instancetypev1alpha2.SchemeGroupVersion.String(),
			},
			Spec: instancetypev1alpha2.VirtualMachinePreferenceSpec{
				Machine: &instancetypev1alpha2.MachinePreferences{
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

		vmSpec, _ := getVMSpecMetaFromResponse()
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
					MachineType: machineTypeFromConfig,
				},
			},
		})

		vmSpec, _ := getVMSpecMetaFromResponse()
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(machineTypeFromConfig))
	})

	It("should default instancetype kind to ClusterSingularResourceName when not provided", func() {
		vm.Spec.Instancetype = &v1.InstancetypeMatcher{
			Name: "foobar",
		}
		vmSpec, _ := getVMSpecMetaFromResponse()
		Expect(vmSpec.Instancetype.Kind).To(Equal(apiinstancetype.ClusterSingularResourceName))
	})

	It("should default preference kind to ClusterSingularPreferenceResourceName when not provided", func() {
		vm.Spec.Preference = &v1.PreferenceMatcher{
			Name: "foobar",
		}
		vmSpec, _ := getVMSpecMetaFromResponse()
		Expect(vmSpec.Preference.Kind).To(Equal(apiinstancetype.ClusterSingularPreferenceResourceName))
	})

	It("should use PreferredMachineType from ClusterSingularPreferenceResourceName when no preference kind is provided", func() {
		preference := &instancetypev1alpha2.VirtualMachineClusterPreference{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name: "machineTypeClusterPreference",
			},
			TypeMeta: k8smetav1.TypeMeta{
				Kind:       apiinstancetype.ClusterSingularPreferenceResourceName,
				APIVersion: instancetypev1alpha2.SchemeGroupVersion.String(),
			},
			Spec: instancetypev1alpha2.VirtualMachinePreferenceSpec{
				Machine: &instancetypev1alpha2.MachinePreferences{
					PreferredMachineType: "pc-q35-5.0",
				},
			},
		}
		_, err := virtClient.VirtualMachineClusterPreference().Create(context.Background(), preference, k8smetav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vm.Spec.Preference = &v1.PreferenceMatcher{
			Name: preference.Name,
		}

		vmSpec, _ := getVMSpecMetaFromResponse()
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(preference.Spec.Machine.PreferredMachineType))
	})

	Context("failure tests", func() {
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
	})
})
