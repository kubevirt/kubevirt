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
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"
	fakeclientset "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	"kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/instancetype/v1alpha1"
	"kubevirt.io/client-go/kubecli"

	instancetype "kubevirt.io/kubevirt/pkg/instancetype"
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

	machineTypeFromConfig := "pc-q35-3.0"

	getResponse := func() *admissionv1.AdmissionResponse {
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

	getVMSpecMetaFromResponse := func() (*v1.VirtualMachineSpec, *k8smetav1.ObjectMeta) {
		resp := getResponse()
		Expect(resp.Allowed).To(BeTrue())

		By("Getting the VM spec from the response")
		vmSpec := &v1.VirtualMachineSpec{}
		vmMeta := &k8smetav1.ObjectMeta{}
		patch := []utiltypes.PatchOperation{
			{Value: vmSpec},
			{Value: vmMeta},
		}
		err := json.Unmarshal(resp.Patch, &patch)
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
		vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{}

		mutator = &VMsMutator{}
		mutator.ClusterConfig, _, kvInformer = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
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

		vm.Spec.Template.Spec.Domain.Machine = &v1.Machine{Type: "q35"}

		vmSpec, _ := getVMSpecMetaFromResponse()
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(vm.Spec.Template.Spec.Domain.Machine.Type))
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

	Context("Instancetype and Preferences", func() {
		const resourceUID types.UID = "9160e5de-2540-476a-86d9-af0081aee68a"
		const resourceGeneration int64 = 1

		var (
			fakeInstancetypeClients v1alpha1.InstancetypeV1alpha1Interface
			k8sClient               *k8sfake.Clientset
		)

		BeforeEach(func() {
			k8sClient = k8sfake.NewSimpleClientset()
			virtClient.EXPECT().AppsV1().Return(k8sClient.AppsV1()).AnyTimes()
			fakeInstancetypeClients = fakeclientset.NewSimpleClientset().InstancetypeV1alpha1()
		})

		Context("Instancetype", func() {
			var (
				fakeInstancetypeClient        v1alpha1.VirtualMachineInstancetypeInterface
				fakeClusterInstancetypeClient v1alpha1.VirtualMachineClusterInstancetypeInterface
				instancetypeObj               *instancetypev1alpha1.VirtualMachineInstancetype
				clusterInstancetypeObj        *instancetypev1alpha1.VirtualMachineClusterInstancetype
			)

			BeforeEach(func() {

				fakeInstancetypeClient = fakeInstancetypeClients.VirtualMachineInstancetypes(k8smetav1.NamespaceDefault)
				virtClient.EXPECT().VirtualMachineInstancetype(gomock.Any()).Return(fakeInstancetypeClient).AnyTimes()

				fakeClusterInstancetypeClient = fakeInstancetypeClients.VirtualMachineClusterInstancetypes()
				virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(fakeClusterInstancetypeClient).AnyTimes()

				instancetypeSpec := instancetypev1alpha1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1alpha1.CPUInstancetype{
						Guest: uint32(2),
					},
					Memory: instancetypev1alpha1.MemoryInstancetype{
						Guest: resource.MustParse("128M"),
					},
				}

				instancetypeObj = &instancetypev1alpha1.VirtualMachineInstancetype{
					ObjectMeta: k8smetav1.ObjectMeta{
						Name:       "instancetype",
						Namespace:  k8smetav1.NamespaceDefault,
						UID:        resourceUID,
						Generation: resourceGeneration,
					},
					Spec: instancetypeSpec,
				}
				virtClient.VirtualMachineInstancetype(vm.Namespace).Create(context.Background(), instancetypeObj, k8smetav1.CreateOptions{})

				clusterInstancetypeObj = &instancetypev1alpha1.VirtualMachineClusterInstancetype{
					ObjectMeta: k8smetav1.ObjectMeta{
						Name:       "clusterInstancetype",
						UID:        resourceUID,
						Generation: resourceGeneration,
					},
					Spec: instancetypeSpec,
				}
				virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), clusterInstancetypeObj, k8smetav1.CreateOptions{})

			})

			It("should store VirtualMachineInstancetype", func() {

				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name: instancetypeObj.Name,
					Kind: apiinstancetype.SingularResourceName,
				}

				expectedRevision, err := instancetype.CreateControllerRevision(vm, instancetypeObj)
				Expect(err).ToNot(HaveOccurred())

				vmSpec, _ := getVMSpecMetaFromResponse()

				Expect(vmSpec.Instancetype.RevisionName).To(Equal(expectedRevision.Name))

				revision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), expectedRevision.Name, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				_, ok := revision.Data.Object.(*instancetypev1alpha1.VirtualMachineInstancetype)
				Expect(ok).To(BeTrue(), "Expected VirtualMachineInstancetype in ControllerRevision")

			})

			It("should skip storing VirtualMachineInstancetype if an existing ControllerRevision is present but not referenced by InstancetypeMatcher", func() {
				existingInstancetypeRevision, err := instancetype.CreateControllerRevision(vm, instancetypeObj)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), existingInstancetypeRevision, k8smetav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name: instancetypeObj.Name,
					Kind: apiinstancetype.SingularResourceName,
				}

				vmSpec, _ := getVMSpecMetaFromResponse()

				Expect(vmSpec.Instancetype.RevisionName).To(Equal(existingInstancetypeRevision.Name))

				revision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), existingInstancetypeRevision.Name, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				_, ok := revision.Data.Object.(*instancetypev1alpha1.VirtualMachineInstancetype)
				Expect(ok).To(BeTrue(), "Expected VirtualMachineInstancetype in ControllerRevision")

			})

			It("should reject if an existing ControllerRevision is found with unexpected VirtualMachineInstancetype data", func() {
				unexpectedInstancetype := instancetypeObj.DeepCopy()
				unexpectedInstancetype.Spec.CPU.Guest = 15

				instancetypeRevision, err := instancetype.CreateControllerRevision(vm, unexpectedInstancetype)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeRevision, k8smetav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name: instancetypeObj.Name,
					Kind: apiinstancetype.SingularResourceName,
				}

				resp := getResponse()
				Expect(resp.Allowed).To(BeFalse())

			})

			It("should store VirtualMachineClusterInstancetype", func() {

				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name: clusterInstancetypeObj.Name,
					Kind: apiinstancetype.ClusterSingularResourceName,
				}

				expectedRevision, err := instancetype.CreateControllerRevision(vm, clusterInstancetypeObj)
				Expect(err).ToNot(HaveOccurred())

				vmSpec, _ := getVMSpecMetaFromResponse()

				Expect(vmSpec.Instancetype.RevisionName).To(Equal(expectedRevision.Name))

				revision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), expectedRevision.Name, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				_, ok := revision.Data.Object.(*instancetypev1alpha1.VirtualMachineClusterInstancetype)
				Expect(ok).To(BeTrue(), "Expected VirtualMachineClusterInstancetype in ControllerRevision")

			})

			It("should default kind to ClusterSingularResourceName when not provided", func() {

				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name: clusterInstancetypeObj.Name,
				}

				expectedRevision, err := instancetype.CreateControllerRevision(vm, clusterInstancetypeObj)
				Expect(err).ToNot(HaveOccurred())

				vmSpec, _ := getVMSpecMetaFromResponse()

				Expect(vmSpec.Instancetype.Kind).To(Equal(apiinstancetype.ClusterSingularResourceName))
				Expect(vmSpec.Instancetype.RevisionName).To(Equal(expectedRevision.Name))

				revision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), expectedRevision.Name, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				_, ok := revision.Data.Object.(*instancetypev1alpha1.VirtualMachineClusterInstancetype)
				Expect(ok).To(BeTrue(), "Expected VirtualMachineClusterInstancetype in ControllerRevision")

			})

			It("should reject if an existing ControllerRevision is found with unexpected VirtualMachineClusterInstancetype data", func() {
				unexpectedInstancetype := clusterInstancetypeObj.DeepCopy()
				unexpectedInstancetype.Spec.CPU.Guest = 15

				instancetypeRevision, err := instancetype.CreateControllerRevision(vm, unexpectedInstancetype)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeRevision, k8smetav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name: clusterInstancetypeObj.Name,
					Kind: apiinstancetype.ClusterSingularResourceName,
				}

				resp := getResponse()
				Expect(resp.Allowed).To(BeFalse())

			})

			It("should reject if an invalid InstancetypeMatcher Kind is provided", func() {

				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name: instancetypeObj.Name,
					Kind: "foobar",
				}

				resp := getResponse()
				Expect(resp.Allowed).To(BeFalse())

			})

			It("should reject the request if a VirtualMachineInstancetype cannot be found", func() {

				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name: "foobar",
					Kind: apiinstancetype.SingularResourceName,
				}

				resp := getResponse()
				Expect(resp.Allowed).To(BeFalse())

			})

			It("should reject the request if a VirtualMachineClusterInstancetype cannot be found", func() {

				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name: "foobar",
					Kind: apiinstancetype.ClusterPluralResourceName,
				}

				resp := getResponse()
				Expect(resp.Allowed).To(BeFalse())

			})

		})

		Context("preference", func() {
			var (
				preference                  *instancetypev1alpha1.VirtualMachinePreference
				clusterPreference           *instancetypev1alpha1.VirtualMachineClusterPreference
				fakePreferenceClient        v1alpha1.VirtualMachinePreferenceInterface
				fakeClusterPreferenceClient v1alpha1.VirtualMachineClusterPreferenceInterface
			)

			BeforeEach(func() {

				fakePreferenceClient = fakeInstancetypeClients.VirtualMachinePreferences(k8smetav1.NamespaceDefault)
				virtClient.EXPECT().VirtualMachinePreference(gomock.Any()).Return(fakePreferenceClient).AnyTimes()

				fakeClusterPreferenceClient = fakeInstancetypeClients.VirtualMachineClusterPreferences()
				virtClient.EXPECT().VirtualMachineClusterPreference().Return(fakeClusterPreferenceClient).AnyTimes()

				preferenceSpec := instancetypev1alpha1.VirtualMachinePreferenceSpec{
					Firmware: &instancetypev1alpha1.FirmwarePreferences{
						PreferredUseEfi: pointer.Bool(true),
					},
					Devices: &instancetypev1alpha1.DevicePreferences{
						PreferredDiskBus:        v1.DiskBusVirtio,
						PreferredInterfaceModel: "virtio",
					},
				}
				preference = &instancetypev1alpha1.VirtualMachinePreference{
					ObjectMeta: k8smetav1.ObjectMeta{
						Name:       "preference",
						Namespace:  k8smetav1.NamespaceDefault,
						UID:        resourceUID,
						Generation: resourceGeneration,
					},
					Spec: preferenceSpec,
				}
				virtClient.VirtualMachinePreference(vm.Namespace).Create(context.Background(), preference, k8smetav1.CreateOptions{})

				clusterPreference = &instancetypev1alpha1.VirtualMachineClusterPreference{
					ObjectMeta: k8smetav1.ObjectMeta{
						Name:       "clusterPreference",
						UID:        resourceUID,
						Generation: resourceGeneration,
					},
					Spec: preferenceSpec,
				}
				virtClient.VirtualMachineClusterPreference().Create(context.Background(), clusterPreference, k8smetav1.CreateOptions{})
			})

			It("should store VirtualMachinePreference", func() {

				vm.Spec.Preference = &v1.PreferenceMatcher{
					Name: preference.Name,
					Kind: apiinstancetype.SingularPreferenceResourceName,
				}

				expectedRevision, err := instancetype.CreateControllerRevision(vm, preference)
				Expect(err).ToNot(HaveOccurred())

				vmSpec, _ := getVMSpecMetaFromResponse()

				Expect(vmSpec.Preference.RevisionName).To(Equal(expectedRevision.Name))

				revision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), expectedRevision.Name, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				_, ok := revision.Data.Object.(*instancetypev1alpha1.VirtualMachinePreference)
				Expect(ok).To(BeTrue(), "Expected VirtualMachinePreference in ControllerRevision")

			})

			It("should skip storing VirtualMachinePreference if an existing ControllerRevision is present but not referenced by PreferenceMatcher", func() {
				existingPreferenceRevision, err := instancetype.CreateControllerRevision(vm, preference)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), existingPreferenceRevision, k8smetav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Preference = &v1.PreferenceMatcher{
					Name: preference.Name,
					Kind: apiinstancetype.SingularPreferenceResourceName,
				}

				vmSpec, _ := getVMSpecMetaFromResponse()

				Expect(vmSpec.Preference.RevisionName).To(Equal(existingPreferenceRevision.Name))

				revision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), existingPreferenceRevision.Name, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				_, ok := revision.Data.Object.(*instancetypev1alpha1.VirtualMachinePreference)
				Expect(ok).To(BeTrue(), "Expected VirtualMachinePreference in ControllerRevision")

			})

			It("should reject if an existing ControllerRevision is found with unexpected VirtualMachinePreference data", func() {
				unexpectedPreference := preference.DeepCopy()
				unexpectedPreference.Spec.Devices.PreferredDiskBus = v1.DiskBusSCSI

				instancetypeRevision, err := instancetype.CreateControllerRevision(vm, unexpectedPreference)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeRevision, k8smetav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Preference = &v1.PreferenceMatcher{
					Name: preference.Name,
					Kind: apiinstancetype.SingularPreferenceResourceName,
				}

				resp := getResponse()
				Expect(resp.Allowed).To(BeFalse())

			})

			It("should store VirtualMachineClusterPreference", func() {

				vm.Spec.Preference = &v1.PreferenceMatcher{
					Name: clusterPreference.Name,
					Kind: apiinstancetype.ClusterSingularPreferenceResourceName,
				}

				expectedRevision, err := instancetype.CreateControllerRevision(vm, clusterPreference)
				Expect(err).ToNot(HaveOccurred())

				vmSpec, _ := getVMSpecMetaFromResponse()

				Expect(vmSpec.Preference.RevisionName).To(Equal(expectedRevision.Name))

				revision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), expectedRevision.Name, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				_, ok := revision.Data.Object.(*instancetypev1alpha1.VirtualMachineClusterPreference)
				Expect(ok).To(BeTrue(), "Expected VirtualMachineClusterPreference in ControllerRevision")

			})

			It("should skip storing VirtualMachineClusterPreference if an existing ControllerRevision is present but not referenced by PreferenceMatcher", func() {
				existingPreferenceRevision, err := instancetype.CreateControllerRevision(vm, clusterPreference)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), existingPreferenceRevision, k8smetav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Preference = &v1.PreferenceMatcher{
					Name: clusterPreference.Name,
					Kind: apiinstancetype.ClusterSingularPreferenceResourceName,
				}

				vmSpec, _ := getVMSpecMetaFromResponse()

				Expect(vmSpec.Preference.RevisionName).To(Equal(existingPreferenceRevision.Name))

				revision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), existingPreferenceRevision.Name, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				_, ok := revision.Data.Object.(*instancetypev1alpha1.VirtualMachineClusterPreference)
				Expect(ok).To(BeTrue(), "Expected VirtualMachineClusterPreference in ControllerRevision")

			})

			It("should reject if an existing ControllerRevision is found with unexpected VirtualMachinePreference data", func() {
				unexpectedPreference := clusterPreference.DeepCopy()
				unexpectedPreference.Spec.Devices.PreferredDiskBus = v1.DiskBusSCSI

				instancetypeRevision, err := instancetype.CreateControllerRevision(vm, unexpectedPreference)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeRevision, k8smetav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Preference = &v1.PreferenceMatcher{
					Name: clusterPreference.Name,
					Kind: apiinstancetype.ClusterSingularPreferenceResourceName,
				}

				resp := getResponse()
				Expect(resp.Allowed).To(BeFalse())

			})

			It("should default kind to ClusterSingularPreferenceResourceName when not provided", func() {

				vm.Spec.Preference = &v1.PreferenceMatcher{
					Name: clusterPreference.Name,
				}

				expectedRevision, err := instancetype.CreateControllerRevision(vm, clusterPreference)
				Expect(err).ToNot(HaveOccurred())

				vmSpec, _ := getVMSpecMetaFromResponse()

				Expect(vmSpec.Preference.Kind).To(Equal(apiinstancetype.ClusterSingularPreferenceResourceName))
				Expect(vmSpec.Preference.RevisionName).To(Equal(expectedRevision.Name))

				revision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), expectedRevision.Name, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				_, ok := revision.Data.Object.(*instancetypev1alpha1.VirtualMachineClusterPreference)
				Expect(ok).To(BeTrue(), "Expected VirtualMachineClusterPreference in ControllerRevision")

			})

			It("should reject if an invalid PreferenceMatcher Kind is provided", func() {

				vm.Spec.Preference = &v1.PreferenceMatcher{
					Name: preference.Name,
					Kind: "foobar",
				}

				resp := getResponse()
				Expect(resp.Allowed).To(BeFalse())

			})

			It("should reject the request if a VirtualMachinePreference cannot be found", func() {

				vm.Spec.Preference = &v1.PreferenceMatcher{
					Name: "foobar",
					Kind: apiinstancetype.SingularPreferenceResourceName,
				}

				resp := getResponse()
				Expect(resp.Allowed).To(BeFalse())

			})

			It("should reject the request if a VirtualMachineClusterPreference cannot be found", func() {

				vm.Spec.Preference = &v1.PreferenceMatcher{
					Name: "foobar",
					Kind: apiinstancetype.ClusterSingularPreferenceResourceName,
				}

				resp := getResponse()
				Expect(resp.Allowed).To(BeFalse())

			})
		})
	})
})
