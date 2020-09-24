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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	"kubevirt.io/kubevirt/pkg/virt-operator/creation/rbac"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"k8s.io/api/admission/v1beta1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	authv1 "k8s.io/api/authentication/v1"

	v1 "kubevirt.io/client-go/api/v1"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Validating VM Admitter", func() {
	config, configMapInformer, crdInformer, _ := testutils.NewFakeClusterConfig(&k8sv1.ConfigMap{})
	var vmsAdmitter *VMsAdmitter

	enableFeatureGate := func(featureGate string) {
		testutils.UpdateFakeClusterConfig(configMapInformer, &k8sv1.ConfigMap{
			Data: map[string]string{virtconfig.FeatureGatesKey: featureGate},
		})
	}
	disableFeatureGates := func() {
		testutils.UpdateFakeClusterConfig(configMapInformer, &k8sv1.ConfigMap{})
	}

	notRunning := false

	BeforeEach(func() {
		vmsAdmitter = &VMsAdmitter{
			ClusterConfig: config,
			cloneAuthFunc: func(pvcNamespace, pvcName, saNamespace, saName string) (bool, string, error) {
				return true, "", nil
			},
		}
	})

	It("reject invalid VirtualMachineInstance spec", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "testdisk",
		})
		vm := &v1.VirtualMachine{
			Spec: v1.VirtualMachineSpec{
				Running: &notRunning,
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: vmi.Spec,
				},
			},
		}
		vmBytes, _ := json.Marshal(&vm)

		ar := &v1beta1.AdmissionReview{
			Request: &v1beta1.AdmissionRequest{
				Resource: webhooks.VirtualMachineGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: vmBytes,
				},
			},
		}

		resp := vmsAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(len(resp.Result.Details.Causes)).To(Equal(1))
		Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.template.spec.domain.devices.disks[0].name"))
	})

	It("should accept valid vmi spec", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "testdisk",
		})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "testdisk",
			VolumeSource: v1.VolumeSource{
				ContainerDisk: &v1.ContainerDiskSource{},
			},
		})

		vm := &v1.VirtualMachine{
			Spec: v1.VirtualMachineSpec{
				Running: &notRunning,
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: vmi.Spec,
				},
			},
		}
		vmBytes, _ := json.Marshal(&vm)

		ar := &v1beta1.AdmissionReview{
			Request: &v1beta1.AdmissionRequest{
				Resource: webhooks.VirtualMachineGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: vmBytes,
				},
			},
		}

		resp := vmsAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeTrue())
	})

	It("should accept valid DataVolumeTemplate", func() {
		vmi := v1.NewMinimalVMI("testvmi")
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
			Spec: v1.VirtualMachineSpec{
				Running: &notRunning,
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: vmi.Spec,
				},
			},
		}

		vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dv1",
			},
			Spec: cdiv1.DataVolumeSpec{
				PVC: &k8sv1.PersistentVolumeClaimSpec{},
			},
		})

		vmBytes, _ := json.Marshal(&vm)

		ar := &v1beta1.AdmissionReview{
			Request: &v1beta1.AdmissionRequest{
				Resource: webhooks.VirtualMachineGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: vmBytes,
				},
			},
		}

		testutils.AddDataVolumeAPI(crdInformer)
		resp := vmsAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeTrue())
	})

	It("should reject invalid DataVolumeTemplate with no Volume reference in VMI template", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "testdisk",
		})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "testdisk",
			VolumeSource: v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name: "WRONG-DATAVOLUME",
				},
			},
		})

		vm := &v1.VirtualMachine{
			Spec: v1.VirtualMachineSpec{
				Running: &notRunning,
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: vmi.Spec,
				},
			},
		}

		vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dv1",
			},
			// this is needed as we have 'better' openapi spec now
			Spec: cdiv1.DataVolumeSpec{
				PVC: &k8sv1.PersistentVolumeClaimSpec{},
			},
		})

		vmBytes, _ := json.Marshal(&vm)

		ar := &v1beta1.AdmissionReview{
			Request: &v1beta1.AdmissionRequest{
				Resource: webhooks.VirtualMachineGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: vmBytes,
				},
			},
		}

		testutils.AddDataVolumeAPI(crdInformer)
		resp := vmsAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(len(resp.Result.Details.Causes)).To(Equal(1))
		Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.dataVolumeTemplate[0]"))
	})

	Context("VM rename", func() {
		var (
			vm         *v1.VirtualMachine
			ar         *v1beta1.AdmissionReview
			running    bool
			notRunning bool
		)

		BeforeEach(func() {
			running = true
			notRunning = false
			vmName := "testvm"
			vmi := v1.NewMinimalVMI(vmName)
			vm = &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vmName,
					Namespace: metav1.NamespaceDefault,
				},
				Spec: v1.VirtualMachineSpec{
					RunStrategy: nil,
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						Spec: vmi.Spec,
					},
				},
			}
		})

		Context("vm creation", func() {
			BeforeEach(func() {
				ar = &v1beta1.AdmissionReview{
					Request: &v1beta1.AdmissionRequest{
						Operation: v1beta1.Create,
						Resource:  webhooks.VirtualMachineGroupVersionResource,
					},
				}
			})

			It("should reject a VM with rename request", func() {
				vm.Spec.Running = &notRunning
				vm.Status = v1.VirtualMachineStatus{
					StateChangeRequests: []v1.VirtualMachineStateChangeRequest{
						{
							Action: v1.RenameRequest,
							Data: map[string]string{
								"newName": "new-name",
							},
						},
					},
				}

				rawObject, err := json.Marshal(vm)
				Expect(err).ToNot(HaveOccurred())

				ar.Request.Object.Raw = rawObject

				resp := vmsAdmitter.Admit(ar)
				Expect(resp.Allowed).To(BeFalse())
				Expect(len(resp.Result.Details.Causes)).To(Equal(1))
				Expect(resp.Result.Details.Causes[0].Field).
					To(Equal("Status.stateChangeRequests"))
			})

			It("should accept a VM with no rename requests", func() {
				vm.Spec.Running = &notRunning
				rawObject, err := json.Marshal(vm)
				Expect(err).ToNot(HaveOccurred())

				ar.Request.Object.Raw = rawObject

				resp := vmsAdmitter.Admit(ar)
				Expect(resp.Allowed).To(BeTrue())
			})
		})

		Context("vm update/patch", func() {
			BeforeEach(func() {
				ar = &v1beta1.AdmissionReview{
					Request: &v1beta1.AdmissionRequest{
						Operation: v1beta1.Update,
						Resource:  webhooks.VirtualMachineGroupVersionResource,
					},
				}
			})

			It("should accept a VM with rename request", func() {
				rawOldObject, err := json.Marshal(vm)
				Expect(err).ToNot(HaveOccurred())

				ar.Request.OldObject.Raw = rawOldObject

				vm.Spec.Running = &notRunning
				vm.Status = v1.VirtualMachineStatus{
					StateChangeRequests: []v1.VirtualMachineStateChangeRequest{
						{
							Action: v1.RenameRequest,
							Data: map[string]string{
								"newName": "new-name",
							},
						},
					},
				}

				rawObject, err := json.Marshal(vm)
				Expect(err).ToNot(HaveOccurred())

				ar.Request.Object.Raw = rawObject

				resp := vmsAdmitter.Admit(ar)
				Expect(resp.Allowed).To(BeTrue())
			})

			It("should reject a VM with invalid rename request", func() {
				rawOldObject, err := json.Marshal(vm)
				Expect(err).ToNot(HaveOccurred())

				ar.Request.OldObject.Raw = rawOldObject

				vm.Spec.Running = &notRunning
				vm.Status = v1.VirtualMachineStatus{
					StateChangeRequests: []v1.VirtualMachineStateChangeRequest{
						{
							Action: v1.RenameRequest,
							Data: map[string]string{
								"newName": "invalid name <>?:;",
							},
						},
					},
				}

				rawObject, err := json.Marshal(vm)
				Expect(err).ToNot(HaveOccurred())

				ar.Request.Object.Raw = rawObject

				resp := vmsAdmitter.Admit(ar)
				Expect(resp.Allowed).To(BeFalse())
				Expect(len(resp.Result.Details.Causes)).To(Equal(1))

				cause := resp.Result.Details.Causes[0]
				Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
				Expect(cause.Field).To(Equal("status.stateChangeRequests"))
			})

			It("should reject a rename request when the VM is running", func() {
				vm.Spec.Running = &running

				rawOldObject, err := json.Marshal(vm)
				Expect(err).ToNot(HaveOccurred())

				ar.Request.OldObject.Raw = rawOldObject

				vm.Status = v1.VirtualMachineStatus{
					Created: true,
					StateChangeRequests: []v1.VirtualMachineStateChangeRequest{
						{
							Action: v1.RenameRequest,
							Data: map[string]string{
								"newName": "new-name",
							},
						},
					},
				}

				rawObject, err := json.Marshal(vm)
				Expect(err).ToNot(HaveOccurred())

				ar.Request.Object.Raw = rawObject

				resp := vmsAdmitter.Admit(ar)
				Expect(resp.Allowed).To(BeFalse())
				Expect(len(resp.Result.Details.Causes)).To(Equal(1))
				Expect(resp.Result.Details.Causes[0].Field).
					To(Equal("spec.running"))
			})

			It("should accept a VM with no rename requests", func() {
				rawOldObject, err := json.Marshal(vm)
				Expect(err).ToNot(HaveOccurred())
				ar.Request.OldObject.Raw = rawOldObject

				vm.Spec.Running = &notRunning
				rawObject, err := json.Marshal(vm)
				Expect(err).ToNot(HaveOccurred())

				ar.Request.Object.Raw = rawObject

				resp := vmsAdmitter.Admit(ar)
				Expect(resp.Allowed).To(BeTrue())
			})

			It("should accept a VM metadata update during rename process from KV service accounts", func() {
				ar.Request.UserInfo = authv1.UserInfo{Username: "system:serviceaccount:kubevirt:" + rbac.ControllerServiceAccountName}
				annotations := make(map[string]string)
				vm.Spec.Running = &notRunning
				vm.Status = v1.VirtualMachineStatus{
					StateChangeRequests: []v1.VirtualMachineStateChangeRequest{
						{
							Action: v1.RenameRequest,
							Data: map[string]string{
								"newName": "new-name",
							},
						},
					},
				}
				rawOldObject, err := json.Marshal(vm)
				Expect(err).ToNot(HaveOccurred())
				ar.Request.OldObject.Raw = rawOldObject

				annotations["testKey"] = "testValue"
				vm.ObjectMeta.Annotations = annotations

				rawObject, err := json.Marshal(vm)
				Expect(err).ToNot(HaveOccurred())
				ar.Request.Object.Raw = rawObject

				resp := vmsAdmitter.Admit(ar)
				Expect(resp.Allowed).To(BeTrue())
			})

			It("should accept a VM status update during rename process from KV service accounts", func() {
				ar.Request.UserInfo = authv1.UserInfo{Username: "system:serviceaccount:kubevirt:" + rbac.ControllerServiceAccountName}
				vm.Spec.Running = &notRunning
				vm.Status = v1.VirtualMachineStatus{
					StateChangeRequests: []v1.VirtualMachineStateChangeRequest{
						{
							Action: v1.RenameRequest,
							Data: map[string]string{
								"newName": "new-name",
							},
						},
					},
				}
				rawOldObject, err := json.Marshal(vm)
				Expect(err).ToNot(HaveOccurred())
				ar.Request.OldObject.Raw = rawOldObject

				vm.Status.Ready = true

				rawObject, err := json.Marshal(vm)
				Expect(err).ToNot(HaveOccurred())
				ar.Request.Object.Raw = rawObject

				resp := vmsAdmitter.Admit(ar)
				Expect(resp.Allowed).To(BeTrue())
			})

			It("should reject a VM spec update during rename process from KV service accounts", func() {
				ar.Request.UserInfo = authv1.UserInfo{Username: "system:serviceaccount:kubevirt:" + rbac.ControllerServiceAccountName}
				vm.Spec.Running = &notRunning
				nodeSelection := make(map[string]string)
				vm.Status = v1.VirtualMachineStatus{
					StateChangeRequests: []v1.VirtualMachineStateChangeRequest{
						{
							Action: v1.RenameRequest,
							Data: map[string]string{
								"newName": "new-name",
							},
						},
					},
				}
				rawOldObject, err := json.Marshal(vm)
				Expect(err).ToNot(HaveOccurred())
				ar.Request.OldObject.Raw = rawOldObject

				nodeSelection["testKey"] = "testValue"
				vm.Spec.Template.Spec.NodeSelector = nodeSelection

				rawObject, err := json.Marshal(vm)
				Expect(err).ToNot(HaveOccurred())
				ar.Request.Object.Raw = rawObject

				resp := vmsAdmitter.Admit(ar)
				Expect(resp.Allowed).To(BeFalse())
				Expect(len(resp.Result.Details.Causes)).To(Equal(1))
				Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec"))
				Expect(resp.Result.Details.Causes[0].Message).To(Equal("Cannot update VM spec until rename process completes"))
			})

			It("should reject a VM modification during rename process from an arbitrary user", func() {
				ar.Request.UserInfo = authv1.UserInfo{Username: "testuser"}
				vm.Spec.Running = &notRunning
				vm.Status = v1.VirtualMachineStatus{
					StateChangeRequests: []v1.VirtualMachineStateChangeRequest{
						{
							Action: v1.RenameRequest,
							Data: map[string]string{
								"newName": "new-name",
							},
						},
					},
				}
				rawOldObject, err := json.Marshal(vm)
				Expect(err).ToNot(HaveOccurred())
				ar.Request.OldObject.Raw = rawOldObject

				vm.Status.Ready = true

				rawObject, err := json.Marshal(vm)
				Expect(err).ToNot(HaveOccurred())
				ar.Request.Object.Raw = rawObject

				resp := vmsAdmitter.Admit(ar)
				Expect(resp.Allowed).To(BeFalse())
				Expect(len(resp.Result.Details.Causes)).To(Equal(1))
				Expect(resp.Result.Details.Causes[0].Message).To(Equal("Modifying a VM during a rename process is restricted to Kubevirt core components"))

			})
		})
	})

	Context("with Volume", func() {

		BeforeEach(func() {
			enableFeatureGate(virtconfig.HostDiskGate)
		})

		AfterEach(func() {
			disableFeatureGates()
		})

		table.DescribeTable("should accept valid volumes",
			func(volumeSource v1.VolumeSource) {
				vmi := v1.NewMinimalVMI("testvmi")
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name:         "testvolume",
					VolumeSource: volumeSource,
				})

				testutils.AddDataVolumeAPI(crdInformer)
				causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
				Expect(len(causes)).To(Equal(0))
			},
			table.Entry("with pvc volume source", v1.VolumeSource{PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{}}),
			table.Entry("with cloud-init volume source", v1.VolumeSource{CloudInitNoCloud: &v1.CloudInitNoCloudSource{UserData: "fake", NetworkData: "fake"}}),
			table.Entry("with containerDisk volume source", v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{}}),
			table.Entry("with ephemeral volume source", v1.VolumeSource{Ephemeral: &v1.EphemeralVolumeSource{}}),
			table.Entry("with emptyDisk volume source", v1.VolumeSource{EmptyDisk: &v1.EmptyDiskSource{}}),
			table.Entry("with dataVolume volume source", v1.VolumeSource{DataVolume: &v1.DataVolumeSource{Name: "fake"}}),
			table.Entry("with hostDisk volume source", v1.VolumeSource{HostDisk: &v1.HostDisk{Path: "fake", Type: v1.HostDiskExistsOrCreate}}),
			table.Entry("with configMap volume source", v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{LocalObjectReference: k8sv1.LocalObjectReference{Name: "fake"}}}),
			table.Entry("with secret volume source", v1.VolumeSource{Secret: &v1.SecretVolumeSource{SecretName: "fake"}}),
			table.Entry("with serviceAccount volume source", v1.VolumeSource{ServiceAccount: &v1.ServiceAccountVolumeSource{ServiceAccountName: "fake"}}),
		)
		It("should reject DataVolume when feature gate is disabled", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name:         "testvolume",
				VolumeSource: v1.VolumeSource{DataVolume: &v1.DataVolumeSource{Name: "fake"}},
			})

			testutils.RemoveDataVolumeAPI(crdInformer)
			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0]"))
		})
		It("should reject DataVolume when DataVolume name is not set", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name:         "testvolume",
				VolumeSource: v1.VolumeSource{DataVolume: &v1.DataVolumeSource{Name: ""}},
			})

			testutils.AddDataVolumeAPI(crdInformer)
			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(len(causes)).To(Equal(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueRequired"))
			Expect(causes[0].Field).To(Equal("fake[0].name"))
			Expect(causes[0].Message).To(Equal("DataVolume 'name' must be set"))
		})
		It("should reject volume with no volume source set", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0]"))
		})
		It("should reject volume with multiple volume sources set", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					ContainerDisk:         &v1.ContainerDiskSource{},
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0]"))
		})
		It("should reject volumes with duplicate names", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{},
				},
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[1].name"))
		})
		It("should reject volume count > arrayLenMax", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			for i := 0; i <= arrayLenMax; i++ {
				name := strconv.Itoa(i)

				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: "testvolume" + name,
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{},
					},
				})
			}

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(len(causes)).To(Equal(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake"))
			Expect(causes[0].Message).To(Equal(fmt.Sprintf("fake list exceeds the %d element limit in length", arrayLenMax)))
		})

		table.DescribeTable("should verify cloud-init userdata length", func(userDataLen int, expectedErrors int, base64Encode bool) {
			vmi := v1.NewMinimalVMI("testvmi")

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
			Expect(len(causes)).To(Equal(expectedErrors))
			for _, cause := range causes {
				Expect(cause.Field).To(ContainSubstring("fake[0].cloudInitNoCloud"))
			}
		},
			table.Entry("should accept userdata under max limit", 10, 0, false),
			table.Entry("should accept userdata equal max limit", cloudInitUserMaxLen, 0, false),
			table.Entry("should reject userdata greater than max limit", cloudInitUserMaxLen+1, 1, false),
			table.Entry("should accept userdata base64 under max limit", 10, 0, true),
			table.Entry("should accept userdata base64 equal max limit", cloudInitUserMaxLen, 0, true),
			table.Entry("should reject userdata base64 greater than max limit", cloudInitUserMaxLen+1, 1, true),
		)

		table.DescribeTable("should verify cloud-init networkdata length", func(networkDataLen int, expectedErrors int, base64Encode bool) {
			vmi := v1.NewMinimalVMI("testvmi")

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
			Expect(len(causes)).To(Equal(expectedErrors))
			for _, cause := range causes {
				Expect(cause.Field).To(ContainSubstring("fake[0].cloudInitNoCloud"))
			}
		},
			table.Entry("should accept networkdata under max limit", 10, 0, false),
			table.Entry("should accept networkdata equal max limit", cloudInitNetworkMaxLen, 0, false),
			table.Entry("should reject networkdata greater than max limit", cloudInitNetworkMaxLen+1, 1, false),
			table.Entry("should accept networkdata base64 under max limit", 10, 0, true),
			table.Entry("should accept networkdata base64 equal max limit", cloudInitNetworkMaxLen, 0, true),
			table.Entry("should reject networkdata base64 greater than max limit", cloudInitNetworkMaxLen+1, 1, true),
		)

		It("should reject cloud-init with invalid base64 userdata", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					CloudInitNoCloud: &v1.CloudInitNoCloudSource{
						UserDataBase64: "#######garbage******",
					},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].cloudInitNoCloud.userDataBase64"))
		})

		It("should reject cloud-init with invalid base64 networkdata", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					CloudInitNoCloud: &v1.CloudInitNoCloudSource{
						UserData:          "fake",
						NetworkDataBase64: "#######garbage******",
					},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].cloudInitNoCloud.networkDataBase64"))
		})

		It("should reject cloud-init with multiple userdata sources", func() {
			vmi := v1.NewMinimalVMI("testvmi")

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
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].cloudInitNoCloud"))
		})

		It("should reject cloud-init with multiple networkdata sources", func() {
			vmi := v1.NewMinimalVMI("testvmi")

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
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].cloudInitNoCloud"))
		})

		It("should reject hostDisk without required parameters", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					HostDisk: &v1.HostDisk{},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(len(causes)).To(Equal(2))
			Expect(causes[0].Field).To(Equal("fake[0].hostDisk.path"))
			Expect(causes[1].Field).To(Equal("fake[0].hostDisk.type"))
		})

		It("should reject hostDisk without given 'path'", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					HostDisk: &v1.HostDisk{
						Type: v1.HostDiskExistsOrCreate,
					},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].hostDisk.path"))
		})

		It("should reject hostDisk with invalid type", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					HostDisk: &v1.HostDisk{
						Path: "fakePath",
						Type: "fakeType",
					},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].hostDisk.type"))
		})

		It("should reject hostDisk when the capacity is specified with a `DiskExists` type", func() {
			vmi := v1.NewMinimalVMI("testvmi")
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
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].hostDisk.capacity"))
		})

		It("should reject a configMap without the configMapName field", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].configMap.name"))
		})

		It("should reject a secret without the secretName field", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].secret.secretName"))
		})

		It("should reject a serviceAccount without the serviceAccountName field", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					ServiceAccount: &v1.ServiceAccountVolumeSource{},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].serviceAccount.serviceAccountName"))
		})

		It("should reject multiple serviceAccounts", func() {
			vmi := v1.NewMinimalVMI("testvmi")

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
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake"))
		})

		table.DescribeTable("should successfully authorize clone", func(arNamespace, vmNamespace, sourceNamespace,
			serviceAccount, expectedSourceNamespace, expectedTargetNamespace, expectedServiceAccount string) {

			vm := &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: vmNamespace,
				},
				Spec: v1.VirtualMachineSpec{
					Template: &v1.VirtualMachineInstanceTemplateSpec{},
					DataVolumeTemplates: []v1.DataVolumeTemplateSpec{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "whatever",
							},
							Spec: cdiv1.DataVolumeSpec{
								Source: cdiv1.DataVolumeSource{
									PVC: &cdiv1.DataVolumeSourcePVC{
										Name:      "whocares",
										Namespace: sourceNamespace,
									},
								},
							},
						},
					},
				},
			}

			if serviceAccount != "" {
				vm.Spec.Template.Spec.Volumes = []v1.Volume{
					{
						VolumeSource: v1.VolumeSource{
							ServiceAccount: &v1.ServiceAccountVolumeSource{
								ServiceAccountName: serviceAccount,
							},
						},
					},
				}
			}

			ar := &v1beta1.AdmissionRequest{
				Namespace: arNamespace,
			}

			vmsAdmitter.cloneAuthFunc = makeCloneAdmitFunc(expectedSourceNamespace, "whocares",
				expectedTargetNamespace, expectedServiceAccount)
			causes, err := vmsAdmitter.authorizeVirtualMachineSpec(ar, vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(causes).To(BeEmpty())
		},
			table.Entry("when source namespace suppied", "ns1", "", "ns3", "", "ns3", "ns1", "default"),
			table.Entry("when vm namespace suppied and source not", "ns1", "ns2", "", "", "ns2", "ns2", "default"),
			table.Entry("when source namespace suppied", "ns1", "", "ns3", "", "ns3", "ns1", "default"),
			table.Entry("when ar namespace suppied and vm/source not", "ns1", "", "", "", "ns1", "ns1", "default"),
			table.Entry("when everything suppied", "ns1", "ns2", "ns3", "", "ns3", "ns2", "default"),
			table.Entry("when everything suppied", "ns1", "ns2", "ns3", "sa", "ns3", "ns2", "sa"),
		)

		table.DescribeTable("should deny clone", func(sourceNamespace, sourceName, failMessage string, failErr error, expectedMessage string) {
			vm := &v1.VirtualMachine{
				Spec: v1.VirtualMachineSpec{
					Template: &v1.VirtualMachineInstanceTemplateSpec{},
					DataVolumeTemplates: []v1.DataVolumeTemplateSpec{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "whatever",
							},
							Spec: cdiv1.DataVolumeSpec{
								Source: cdiv1.DataVolumeSource{
									PVC: &cdiv1.DataVolumeSourcePVC{
										Name:      sourceName,
										Namespace: sourceNamespace,
									},
								},
							},
						},
					},
				},
			}

			ar := &v1beta1.AdmissionRequest{}

			vmsAdmitter.cloneAuthFunc = makeCloneAdmitFailFunc(failMessage, failErr)
			causes, err := vmsAdmitter.authorizeVirtualMachineSpec(ar, vm)
			if failErr != nil {
				Expect(err).To(Equal(failErr))
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(causes).To(HaveLen(1))
				Expect(causes[0].Message).To(Equal(expectedMessage))
			}
		},
			table.Entry("when source name not supplied", "", "sourceName", "Clone source /sourceName invalid", nil, "Clone source /sourceName invalid"),
			table.Entry("when source namespace not supplied", "sourceNamespace", "", "Clone source sourceNamespace/ invalid", nil, "Clone source sourceNamespace/ invalid"),
			table.Entry("when user not authorized", "sourceNamespace", "sourceName", "no permission", nil, "Authorization failed, message is: no permission"),
			table.Entry("error occurs", "sourceNamespace", "sourceName", "", fmt.Errorf("bad error"), ""),
		)
	})

	table.DescribeTable("when snapshot is in progress, should", func(mutateFn func(*v1.VirtualMachine) bool) {
		vmi := v1.NewMinimalVMI("testvmi")
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

		ar := &v1beta1.AdmissionReview{
			Request: &v1beta1.AdmissionRequest{
				Operation: v1beta1.Update,
				Resource:  webhooks.VirtualMachineGroupVersionResource,
				OldObject: runtime.RawExtension{
					Raw: oldObjectBytes,
				},
				Object: runtime.RawExtension{
					Raw: objectBytes,
				},
			},
		}

		resp := vmsAdmitter.Admit(ar)
		Expect(resp.Allowed).To(Equal(allow))

		if !allow {
			Expect(len(resp.Result.Details.Causes)).To(Equal(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec"))
		}
	},
		table.Entry("reject update to spec", func(vm *v1.VirtualMachine) bool {
			vm.Spec.Running = &[]bool{true}[0]
			return false
		}),
		table.Entry("accept update to metadata", func(vm *v1.VirtualMachine) bool {
			vm.Annotations = map[string]string{"foo": "bar"}
			return true
		}),
		table.Entry("accept update to status", func(vm *v1.VirtualMachine) bool {
			vm.Status.Ready = true
			return true
		}),
	)
})

func makeCloneAdmitFunc(expectedSourceNamespace, expectedPVCName, expectedTargetNamespace, expectedServiceAccount string) CloneAuthFunc {
	return func(pvcNamespace, pvcName, saNamespace, saName string) (bool, string, error) {
		Expect(pvcNamespace).Should(Equal(expectedSourceNamespace))
		Expect(pvcName).Should(Equal(expectedPVCName))
		Expect(saNamespace).Should(Equal(expectedTargetNamespace))
		Expect(saName).Should(Equal(expectedServiceAccount))
		return true, "", nil
	}
}

func makeCloneAdmitFailFunc(message string, err error) CloneAuthFunc {
	return func(pvcNamespace, pvcName, saNamespace, saName string) (bool, string, error) {
		return false, message, err
	}
}
