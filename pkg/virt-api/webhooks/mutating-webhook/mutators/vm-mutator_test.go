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

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1alpha3 "kubevirt.io/api/instancetype/v1alpha3"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype"

	cdifake "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned/fake"
	fakeclientset "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	instancetypeclientset "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/instancetype/v1alpha3"

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
	var fakeInstancetypeClients instancetypeclientset.InstancetypeV1alpha3Interface
	var fakePreferenceClient instancetypeclientset.VirtualMachinePreferenceInterface
	var fakeClusterPreferenceClient instancetypeclientset.VirtualMachineClusterPreferenceInterface
	var k8sClient *k8sfake.Clientset
	var cdiClient *cdifake.Clientset

	machineTypeFromConfig := "pc-q35-3.0"

	admitVM := func() *admissionv1.AdmissionResponse {
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
		resp := admitVM()
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

		fakeInstancetypeClients = fakeclientset.NewSimpleClientset().InstancetypeV1alpha3()
		fakePreferenceClient = fakeInstancetypeClients.VirtualMachinePreferences(vm.Namespace)
		fakeClusterPreferenceClient = fakeInstancetypeClients.VirtualMachineClusterPreferences()
		virtClient.EXPECT().VirtualMachinePreference(gomock.Any()).Return(fakePreferenceClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineClusterPreference().Return(fakeClusterPreferenceClient).AnyTimes()

		k8sClient = k8sfake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
		cdiClient = cdifake.NewSimpleClientset()
		virtClient.EXPECT().CdiClient().Return(cdiClient).AnyTimes()

		mutator.InstancetypeMethods = instancetype.NewMethods(nil, nil, nil, nil, nil, virtClient)
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
		preference := &instancetypev1alpha3.VirtualMachinePreference{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name: "machineTypePreference",
			},
			TypeMeta: k8smetav1.TypeMeta{
				Kind:       apiinstancetype.SingularPreferenceResourceName,
				APIVersion: instancetypev1alpha3.SchemeGroupVersion.String(),
			},
			Spec: instancetypev1alpha3.VirtualMachinePreferenceSpec{
				Machine: &instancetypev1alpha3.MachinePreferences{
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
		preference := &instancetypev1alpha3.VirtualMachinePreference{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name: "machineTypePreference",
			},
			TypeMeta: k8smetav1.TypeMeta{
				Kind:       apiinstancetype.SingularPreferenceResourceName,
				APIVersion: instancetypev1alpha3.SchemeGroupVersion.String(),
			},
			Spec: instancetypev1alpha3.VirtualMachinePreferenceSpec{
				Machine: &instancetypev1alpha3.MachinePreferences{
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
		preference := &instancetypev1alpha3.VirtualMachineClusterPreference{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name: "machineTypeClusterPreference",
			},
			TypeMeta: k8smetav1.TypeMeta{
				Kind:       apiinstancetype.ClusterSingularPreferenceResourceName,
				APIVersion: instancetypev1alpha3.SchemeGroupVersion.String(),
			},
			Spec: instancetypev1alpha3.VirtualMachinePreferenceSpec{
				Machine: &instancetypev1alpha3.MachinePreferences{
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

	It("should use storage class from VirtualMachinePreference", func() {
		preference := &instancetypev1alpha3.VirtualMachineClusterPreference{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name: "machineTypeClusterPreference",
			},
			TypeMeta: k8smetav1.TypeMeta{
				Kind:       apiinstancetype.ClusterSingularPreferenceResourceName,
				APIVersion: instancetypev1alpha3.SchemeGroupVersion.String(),
			},
			Spec: instancetypev1alpha3.VirtualMachinePreferenceSpec{
				Volumes: &instancetypev1alpha3.VolumePreferences{
					PreferredStorageClassName: "ceph",
				},
			},
		}
		vm.Spec.Preference = &v1.PreferenceMatcher{
			Name: preference.Name,
		}

		_, err := virtClient.VirtualMachineClusterPreference().Create(context.Background(), preference, k8smetav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vmSpec, _ := getVMSpecMetaFromResponse()
		for _, dv := range vmSpec.DataVolumeTemplates {
			Expect(*dv.Spec.PVC.StorageClassName).To(Equal(preference.Spec.Volumes.PreferredStorageClassName))
		}
	})

	It("storage class name already defined in VM, value from VirtualMachinePreference should not be used", func() {
		storageClass := "local"
		storageSpec := v1.DataVolumeTemplateSpec{
			Spec: cdiv1.DataVolumeSpec{
				PVC: &k8sv1.PersistentVolumeClaimSpec{
					StorageClassName: &storageClass,
				},
			},
		}
		vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, storageSpec)

		preference := &instancetypev1alpha3.VirtualMachineClusterPreference{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name: "machineTypeClusterPreference",
			},
			TypeMeta: k8smetav1.TypeMeta{
				Kind:       apiinstancetype.ClusterSingularPreferenceResourceName,
				APIVersion: instancetypev1alpha3.SchemeGroupVersion.String(),
			},
			Spec: instancetypev1alpha3.VirtualMachinePreferenceSpec{
				Volumes: &instancetypev1alpha3.VolumePreferences{
					PreferredStorageClassName: "ceph",
				},
			},
		}

		vm.Spec.Preference = &v1.PreferenceMatcher{
			Name: preference.Name,
		}
		_, err := virtClient.VirtualMachineClusterPreference().Create(context.Background(), preference, k8smetav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vmSpec, _ := getVMSpecMetaFromResponse()
		for _, dv := range vmSpec.DataVolumeTemplates {
			Expect(*dv.Spec.PVC.StorageClassName).To(Equal(storageClass))
		}
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
			vmSpec, _ := getVMSpecMetaFromResponse()
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

			vmSpec, _ := getVMSpecMetaFromResponse()
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

			vmSpec, _ := getVMSpecMetaFromResponse()
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

			vmSpec, _ := getVMSpecMetaFromResponse()
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
			dvWithSourceRef := &v1beta1.DataVolume{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "dvWithSourceRef",
					Namespace: vm.Namespace,
				},
				Spec: v1beta1.DataVolumeSpec{
					SourceRef: &v1beta1.DataVolumeSourceRef{
						Name:      sourceRefName,
						Kind:      sourceRefKind,
						Namespace: &sourceRefNamespace,
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

			vmSpec, _ := getVMSpecMetaFromResponse()
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

			vmSpec, _ := getVMSpecMetaFromResponse()
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

			resp := admitVM()
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(ContainSubstring("unable to find volume %s to infer defaults", inferVolumeName))
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

		DescribeTable("should fail to infer defaults from Volume ", func(volumeSource v1.VolumeSource, messageSubstring string, instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name:         inferVolumeName,
				VolumeSource: volumeSource,
			}}
			resp := admitVM()
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(ContainSubstring(messageSubstring))
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
				}, nil,
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
				},
			),
			Entry("with unknown DataVolume and PersistentVolumeClaim for InstancetypeMatcher",
				v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: unknownDVName,
					},
				}, fmt.Sprintf("persistentvolumeclaims \"%s\" not found", unknownDVName),
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				}, nil,
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
				},
			),
			Entry("with unsupported VolumeSource type for InstancetypeMatcher",
				v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{},
				},
				fmt.Sprintf("unable to infer defaults from volume %s as type is not supported", inferVolumeName),
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				}, nil,
			),
			Entry("with unsupported VolumeSource type for PreferenceMatcher",
				v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{},
				},
				fmt.Sprintf("unable to infer defaults from volume %s as type is not supported", inferVolumeName),
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				},
			),
		)

		DescribeTable("should fail to infer defaults from DataVolume with an unsupported DataVolumeSource", func(instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher) {
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
			resp := admitVM()
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(ContainSubstring("unable to infer defaults from DataVolumeSpec as DataVolumeSource is not supported"))
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

		DescribeTable("should fail to infer defaults from DataVolume with an unknown DataVolumeSourceRef Kind", func(instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher) {
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
			resp := admitVM()
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(ContainSubstring("unable to infer defaults from DataVolumeSourceRef as Kind foo is not supported"))
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

		DescribeTable("should fail to infer defaults from DataSource missing DataVolumeSourcePVC", func(instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher) {
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
			resp := admitVM()
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(ContainSubstring("unable to infer defaults from DataSource that doesn't provide DataVolumeSourcePVC"))
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

		DescribeTable("should fail to infer defaults from PersistentVolumeClaim without default instance type label", func(instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher, requiredLabel string) {
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
			resp := admitVM()
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(ContainSubstring("unable to find required %s label on the volume", requiredLabel))
		},
			Entry("for InstancetypeMatcher",
				&v1.InstancetypeMatcher{
					InferFromVolume: inferVolumeName,
				}, nil, apiinstancetype.DefaultInstancetypeLabel,
			),
			Entry("for PreferenceMatcher",
				nil,
				&v1.PreferenceMatcher{
					InferFromVolume: inferVolumeName,
				}, apiinstancetype.DefaultPreferenceLabel,
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
			vmSpec, _ := getVMSpecMetaFromResponse()
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
			vmSpec, _ := getVMSpecMetaFromResponse()
			Expect(vmSpec.Instancetype).To(Equal(&v1.InstancetypeMatcher{
				Name: defaultInferedNameFromPVC,
				Kind: defaultInferedKindFromPVC,
			}))
			Expect(vmSpec.Preference).To(Equal(&v1.PreferenceMatcher{
				Name: defaultInferedNameFromPVC,
				Kind: defaultInferedKindFromPVC,
			}))
		})
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

		DescribeTable("should fail if", func(instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher, expectedField, expectedMessage string) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			resp := admitVM()
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(ContainSubstring(expectedMessage))
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal(expectedField))
		},
			Entry("InstancetypeMatcher does not provide Name or InferFromVolume", &v1.InstancetypeMatcher{}, nil, k8sfield.NewPath("spec", "instancetype").String(), "Either Name or InferFromVolume should be provided within the InstancetypeMatcher"),
			Entry("InstancetypeMatcher provides Name and InferFromVolume", &v1.InstancetypeMatcher{Name: "foo", InferFromVolume: "bar"}, nil, k8sfield.NewPath("spec", "instancetype", "name").String(), "Name should not be provided when InferFromVolume is used within the InstancetypeMatcher"),
			Entry("InstancetypeMatcher provides Kind and InferFromVolume", &v1.InstancetypeMatcher{Kind: "foo", InferFromVolume: "bar"}, nil, k8sfield.NewPath("spec", "instancetype", "kind").String(), "Kind should not be provided when InferFromVolume is used within the InstancetypeMatcher"),
			Entry("PreferenceMatcher does not provide Name or InferFromVolume", nil, &v1.PreferenceMatcher{}, k8sfield.NewPath("spec", "preference").String(), "Either Name or InferFromVolume should be provided within the PreferenceMatcher"),
			Entry("PreferenceMatcher provides Name and InferFromVolume", nil, &v1.PreferenceMatcher{Name: "foo", InferFromVolume: "bar"}, k8sfield.NewPath("spec", "preference", "name").String(), "Name should not be provided when InferFromVolume is used within the PreferenceMatcher"),
			Entry("PreferenceMatcher provides Kind and InferFromVolume", nil, &v1.PreferenceMatcher{Kind: "foo", InferFromVolume: "bar"}, k8sfield.NewPath("spec", "preference", "kind").String(), "Kind should not be provided when InferFromVolume is used within the PreferenceMatcher"),
		)
	})
})
