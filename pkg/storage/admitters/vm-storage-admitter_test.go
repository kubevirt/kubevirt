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

package admitters

import (
	"context"
	"encoding/json"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

const kubeVirtNamespace = "kubevirt"

var _ = Describe("Validating VM Admitter", func() {
	config, crdInformer, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
	var (
		virtClient *kubecli.MockKubevirtClient
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		k8sClient := k8sfake.NewSimpleClientset()
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		virtClient.EXPECT().AuthorizationV1().Return(k8sClient.AuthorizationV1()).AnyTimes()
	})

	Context("Validate VM DataVolumeTemplate", func() {
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
			causes, err := admitVm(virtClient, admissionv1.Create, config, vm, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(causes).To(BeEmpty())
			causes = ValidateDataVolumeTemplate(k8sfield.NewPath("spec"), &vm.Spec)
			Expect(causes).To(BeEmpty())
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
			causes, err := admitVm(virtClient, admissionv1.Create, config, vm, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(causes[0].Message).To(Equal("Embedded DataVolume namespace another-namespace differs from VM namespace vm-namespace"))
		})
		Context("ValidateDataVolumeTemplate", func() {
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
				causes := ValidateDataVolumeTemplate(k8sfield.NewPath("spec"), &vm.Spec)
				Expect(causes).To(BeEmpty())
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
				causes := ValidateDataVolumeTemplate(k8sfield.NewPath("spec"), &vm.Spec)
				Expect(causes).To(HaveLen(1))
				Expect(causes[0].Message).To(Equal("'name' field must not be empty for DataVolumeTemplate entry spec.dataVolumeTemplate[0].name."))
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
				causes := ValidateDataVolumeTemplate(k8sfield.NewPath("spec"), &vm.Spec)
				Expect(causes).To(HaveLen(1))
				Expect(causes[0].Message).To(Equal("Missing Data volume PVC or Storage"))
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
				causes := ValidateDataVolumeTemplate(k8sfield.NewPath("spec"), &vm.Spec)
				Expect(causes).To(HaveLen(1))
				Expect(causes[0].Message).To(Equal("Duplicate storage definition, both target storage and target pvc defined"))
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
				causes := ValidateDataVolumeTemplate(k8sfield.NewPath("spec"), &vm.Spec)
				Expect(causes).To(HaveLen(1))
				Expect(causes[0].Message).To(Equal("External population is incompatible with Source and SourceRef"))
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
				causes := ValidateDataVolumeTemplate(k8sfield.NewPath("spec"), &vm.Spec)
				Expect(causes).To(HaveLen(1))
				Expect(causes[0].Message).To(Equal("Data volume should have either Source, SourceRef, or be externally populated"))
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
				causes := ValidateDataVolumeTemplate(k8sfield.NewPath("spec"), &vm.Spec)
				Expect(causes).To(HaveLen(1))
				Expect(causes[0].Message).To(Equal("Missing dataVolume valid source"))
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
				causes := ValidateDataVolumeTemplate(k8sfield.NewPath("spec"), &vm.Spec)
				Expect(causes).To(HaveLen(1))
				Expect(causes[0].Message).To(Equal("Multiple dataVolume sources"))
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
				causes := ValidateDataVolumeTemplate(k8sfield.NewPath("spec"), &vm.Spec)
				Expect(causes).To(HaveLen(1))
				Expect(causes[0].Field).To(Equal("spec.dataVolumeTemplate[0]"))
			})
		})
	})

	Context("Validate VM snapshot, restore status", func() {
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
			oldVM := vm.DeepCopy()

			allow := mutateFn(vm)

			causes, err := admitVm(virtClient, admissionv1.Update, config, vm, oldVM)
			Expect(err).ToNot(HaveOccurred())

			if !allow {
				Expect(causes).To(HaveLen(1))
				Expect(causes[0].Field).To(Equal("spec"), causes[0].Message)
			} else {
				Expect(causes).To(BeEmpty())
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
				vm.Spec.RunStrategy = pointer.P(v1.RunStrategyManual)
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
				vm.Spec.RunStrategy = pointer.P(v1.RunStrategyHalted)
			} else {
				vm.Spec.Running = &[]bool{false}[0]
			}
			oldVM := vm.DeepCopy()

			allow := mutateFn(vm)

			causes, err := admitVm(virtClient, admissionv1.Update, config, vm, oldVM)
			Expect(err).ToNot(HaveOccurred())

			if !allow {
				Expect(causes).To(HaveLen(1))
				Expect(causes[0].Field).To(Equal("spec"), causes[0].Message)
			} else {
				Expect(causes).To(BeEmpty())
			}
		},
			Entry("reject update to running true", func(vm *v1.VirtualMachine) bool {
				vm.Spec.Running = &[]bool{true}[0]
				return false
			}, false),
			Entry("reject update of runStrategy", func(vm *v1.VirtualMachine) bool {
				vm.Spec.RunStrategy = pointer.P(v1.RunStrategyManual)
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
	})
})

func admitVm(virtClient *kubecli.MockKubevirtClient, operation admissionv1.Operation, config *virtconfig.ClusterConfig, vm, oldVm *v1.VirtualMachine) ([]metav1.StatusCause, error) {
	vmBytes, _ := json.Marshal(vm)

	ar := &admissionv1.AdmissionRequest{
		Operation: operation,
		Namespace: kubeVirtNamespace,
		Resource:  webhooks.VirtualMachineGroupVersionResource,
		Object: runtime.RawExtension{
			Raw: vmBytes,
		},
	}
	if operation == admissionv1.Update {
		oldVmBytes, _ := json.Marshal(oldVm)
		ar.OldObject = runtime.RawExtension{
			Raw: oldVmBytes,
		}
	}

	return Admit(virtClient, context.Background(), ar, vm, config)
}
