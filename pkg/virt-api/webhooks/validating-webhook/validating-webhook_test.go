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

package validating_webhook

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"k8s.io/api/admission/v1beta1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/datavolumecontroller/v1alpha1"
	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

var _ = Describe("Validating Webhook", func() {
	var vmiInformer cache.SharedIndexInformer

	BeforeSuite(func() {
		vmiInformer, _ = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
		webhooks.SetInformers(&webhooks.Informers{
			VMIInformer: vmiInformer,
		})
	})

	Context("with VirtualMachineInstance admission review", func() {
		It("should reject invalid VirtualMachineInstance spec on create", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})
			vmiBytes, _ := json.Marshal(&vmi)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: vmiBytes,
					},
				},
			}

			resp := admitVMICreate(ar)
			Expect(resp.Allowed).To(Equal(false))
			Expect(len(resp.Result.Details.Causes)).To(Equal(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.domain.devices.disks[0].name"))
		})
		It("should reject VMIs without memory after presets were applied", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})
			vmi.Spec.Domain.Resources = v1.ResourceRequirements{}
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{},
				},
			})
			vmiBytes, _ := json.Marshal(&vmi)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: vmiBytes,
					},
				},
			}
			resp := admitVMICreate(ar)
			Expect(resp.Allowed).To(Equal(false))
			Expect(resp.Result.Message).To(ContainSubstring("no memory requested"))
		})

		Context("with probes given", func() {
			It("should reject probes with not probe action configured", func() {
				vmi := v1.NewMinimalVMI("testvmi")
				m := resource.MustParse("64M")
				vmi.Spec.Domain.Memory = &v1.Memory{Guest: &m}
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{}
				vmi.Spec.ReadinessProbe = &v1.Probe{InitialDelaySeconds: 2}
				vmi.Spec.LivenessProbe = &v1.Probe{InitialDelaySeconds: 2}
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
				vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

				vmiBytes, _ := json.Marshal(&vmi)

				ar := &v1beta1.AdmissionReview{
					Request: &v1beta1.AdmissionRequest{
						Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
						Object: runtime.RawExtension{
							Raw: vmiBytes,
						},
					},
				}
				resp := admitVMICreate(ar)
				Expect(resp.Allowed).To(Equal(false))
				Expect(resp.Result.Message).To(Equal(`either spec.readinessProbe.tcpSocket or spec.readinessProbe.httpGet must be set if a spec.readinessProbe is specified, either spec.livenessProbe.tcpSocket or spec.livenessProbe.httpGet must be set if a spec.livenessProbe is specified`))
			})
			It("should reject probes with more than one action per probe configured", func() {
				vmi := v1.NewMinimalVMI("testvmi")
				m := resource.MustParse("64M")
				vmi.Spec.Domain.Memory = &v1.Memory{Guest: &m}
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{}
				vmi.Spec.ReadinessProbe = &v1.Probe{
					InitialDelaySeconds: 2,
					Handler: v1.Handler{
						HTTPGet:   &k8sv1.HTTPGetAction{Host: "test", Port: intstr.Parse("80")},
						TCPSocket: &k8sv1.TCPSocketAction{Host: "lal", Port: intstr.Parse("80")},
					},
				}
				vmi.Spec.LivenessProbe = &v1.Probe{
					InitialDelaySeconds: 2,
					Handler: v1.Handler{
						HTTPGet:   &k8sv1.HTTPGetAction{Host: "test", Port: intstr.Parse("80")},
						TCPSocket: &k8sv1.TCPSocketAction{Host: "lal", Port: intstr.Parse("80")},
					},
				}
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
				vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

				vmiBytes, _ := json.Marshal(&vmi)

				ar := &v1beta1.AdmissionReview{
					Request: &v1beta1.AdmissionRequest{
						Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
						Object: runtime.RawExtension{
							Raw: vmiBytes,
						},
					},
				}
				resp := admitVMICreate(ar)
				Expect(resp.Allowed).To(Equal(false))
				Expect(resp.Result.Message).To(Equal(`spec.readinessProbe must have exactly one probe type set, spec.livenessProbe must have exactly one probe type set`))
			})
			It("should accept properly configured readiness and liveness probes", func() {
				vmi := v1.NewMinimalVMI("testvmi")
				m := resource.MustParse("64M")
				vmi.Spec.Domain.Memory = &v1.Memory{Guest: &m}
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{}
				vmi.Spec.ReadinessProbe = &v1.Probe{
					InitialDelaySeconds: 2,
					Handler: v1.Handler{
						TCPSocket: &k8sv1.TCPSocketAction{Host: "lal", Port: intstr.Parse("80")},
					},
				}
				vmi.Spec.LivenessProbe = &v1.Probe{
					InitialDelaySeconds: 2,
					Handler: v1.Handler{
						HTTPGet: &k8sv1.HTTPGetAction{Host: "test", Port: intstr.Parse("80")},
					},
				}
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
				vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

				vmiBytes, _ := json.Marshal(&vmi)

				ar := &v1beta1.AdmissionReview{
					Request: &v1beta1.AdmissionRequest{
						Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
						Object: runtime.RawExtension{
							Raw: vmiBytes,
						},
					},
				}
				resp := admitVMICreate(ar)
				Expect(resp.Allowed).To(Equal(true))
			})
			It("should reject properly configured readiness and liveness probes if no Pod Network is present", func() {
				vmi := v1.NewMinimalVMI("testvmi")
				m := resource.MustParse("64M")
				vmi.Spec.Domain.Memory = &v1.Memory{Guest: &m}
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{}
				vmi.Spec.ReadinessProbe = &v1.Probe{
					InitialDelaySeconds: 2,
					Handler: v1.Handler{
						TCPSocket: &k8sv1.TCPSocketAction{Host: "lal", Port: intstr.Parse("80")},
					},
				}
				vmi.Spec.LivenessProbe = &v1.Probe{
					InitialDelaySeconds: 2,
					Handler: v1.Handler{
						HTTPGet: &k8sv1.HTTPGetAction{Host: "test", Port: intstr.Parse("80")},
					},
				}

				vmiBytes, _ := json.Marshal(&vmi)

				ar := &v1beta1.AdmissionReview{
					Request: &v1beta1.AdmissionRequest{
						Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
						Object: runtime.RawExtension{
							Raw: vmiBytes,
						},
					},
				}
				resp := admitVMICreate(ar)
				Expect(resp.Allowed).To(Equal(false))
				Expect(resp.Result.Message).To(Equal(`spec.livenessProbe is only allowed if the Pod Network is attached, spec.readinessProbe is only allowed if the Pod Network is attached`))
			})
		})

		It("should accept valid vmi spec on create", func() {
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
			vmiBytes, _ := json.Marshal(&vmi)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: vmiBytes,
					},
				},
			}
			resp := admitVMICreate(ar)
			Expect(resp.Allowed).To(Equal(true))
		})

		It("should allow unknown fields in the status to allow updates", func() {
			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: []byte(`{"very": "unknown", "spec": { "extremely": "unknown" }, "status": {"unknown": "allowed"}}`),
					},
				},
			}
			resp := admitVMICreate(ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(Equal(`.very in body is a forbidden property, spec.extremely in body is a forbidden property, spec.domain in body is required`))
		})

		table.DescribeTable("should reject documents containing unknown or missing fields for", func(data string, validationResult string, gvr metav1.GroupVersionResource, review func(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse) {
			input := map[string]interface{}{}
			json.Unmarshal([]byte(data), &input)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: gvr,
					Object: runtime.RawExtension{
						Raw: []byte(data),
					},
				},
			}
			resp := review(ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(Equal(validationResult))
		},
			table.Entry("VirtualMachineInstance creation",
				`{"very": "unknown", "spec": { "extremely": "unknown" }}`,
				`.very in body is a forbidden property, spec.extremely in body is a forbidden property, spec.domain in body is required`,
				webhooks.VirtualMachineInstanceGroupVersionResource,
				admitVMICreate,
			),
			table.Entry("VirtualMachineInstance update",
				`{"very": "unknown", "spec": { "extremely": "unknown" }}`,
				`.very in body is a forbidden property, spec.extremely in body is a forbidden property, spec.domain in body is required`,
				webhooks.VirtualMachineInstanceGroupVersionResource,
				admitVMIUpdate,
			),
			table.Entry("VirtualMachineInstancePreset creation and update",
				`{"very": "unknown", "spec": { "extremely": "unknown" }}`,
				`.very in body is a forbidden property, spec.extremely in body is a forbidden property, spec.selector in body is required`,
				webhooks.VirtualMachineInstancePresetGroupVersionResource,
				admitVMIPreset,
			),
			table.Entry("Migration creation ",
				`{"very": "unknown", "spec": { "extremely": "unknown" }}`,
				`.very in body is a forbidden property, spec.extremely in body is a forbidden property`,
				webhooks.MigrationGroupVersionResource,
				admitMigrationCreate,
			),
			table.Entry("Migration update",
				`{"very": "unknown", "spec": { "extremely": "unknown" }}`,
				`.very in body is a forbidden property, spec.extremely in body is a forbidden property`,
				webhooks.MigrationGroupVersionResource,
				admitMigrationCreate,
			),
			table.Entry("VirtualMachine creation and update",
				`{"very": "unknown", "spec": { "extremely": "unknown" }}`,
				`.very in body is a forbidden property, spec.extremely in body is a forbidden property, spec.running in body is required, spec.template in body is required`,
				webhooks.VirtualMachineGroupVersionResource,
				admitVMs,
			),
			table.Entry("VirtualMachineInstanceReplicaSet creation and update",
				`{"very": "unknown", "spec": { "extremely": "unknown" }}`,
				`.very in body is a forbidden property, spec.extremely in body is a forbidden property, spec.selector in body is required, spec.template in body is required`,
				webhooks.VirtualMachineInstanceReplicaSetGroupVersionResource,
				admitVMIRS,
			),
		)
		It("should reject valid VirtualMachineInstance spec on update", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			updateVmi := vmi.DeepCopy()
			updateVmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})
			updateVmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testdisk",
				VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{},
				},
			})
			newVMIBytes, _ := json.Marshal(&updateVmi)
			oldVMIBytes, _ := json.Marshal(&vmi)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: newVMIBytes,
					},
					OldObject: runtime.RawExtension{
						Raw: oldVMIBytes,
					},
					Operation: v1beta1.Update,
				},
			}

			resp := admitVMIUpdate(ar)
			Expect(resp.Allowed).To(Equal(false))
			Expect(len(resp.Result.Details.Causes)).To(Equal(1))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("update of VMI object is restricted"))
		})
	})
	Context("with VMIRS admission review", func() {
		table.DescribeTable("reject invalid VirtualMachineInstance spec", func(vmirs *v1.VirtualMachineInstanceReplicaSet, causes []string) {
			vmirsBytes, _ := json.Marshal(&vmirs)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.VirtualMachineInstanceReplicaSetGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: vmirsBytes,
					},
				},
			}

			resp := admitVMIRS(ar)
			Expect(resp.Allowed).To(Equal(false))
			Expect(resp.Result.Details.Causes).To(HaveLen(len(causes)))
			for i, cause := range causes {
				Expect(resp.Result.Details.Causes[i].Field).To(Equal(cause))
			}
		},
			table.Entry("with missing volume and missing labels", &v1.VirtualMachineInstanceReplicaSet{
				Spec: v1.VirtualMachineInstanceReplicaSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"match": "this"},
					},
					Template: newVirtualMachineBuilder().WithDisk(v1.Disk{
						Name: "testdisk",
					}).BuildTemplate(),
				},
			}, []string{
				"spec.template.spec.domain.devices.disks[0].name",
				"spec.selector",
			}),
			table.Entry("with mismatching label selectors", &v1.VirtualMachineInstanceReplicaSet{
				Spec: v1.VirtualMachineInstanceReplicaSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"match": "not"},
					},
					Template: newVirtualMachineBuilder().WithLabel("match", "this").BuildTemplate(),
				},
			}, []string{
				"spec.selector",
			}),
		)
		It("should accept valid vmi spec", func() {
			vmirs := &v1.VirtualMachineInstanceReplicaSet{
				Spec: v1.VirtualMachineInstanceReplicaSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"match": "me"},
					},
					Template: newVirtualMachineBuilder().
						WithDisk(v1.Disk{
							Name: "testdisk",
						}).
						WithVolume(v1.Volume{
							Name: "testdisk",
							VolumeSource: v1.VolumeSource{
								ContainerDisk: &v1.ContainerDiskSource{},
							},
						}).
						WithLabel("match", "me").
						BuildTemplate(),
				},
			}
			vmirsBytes, _ := json.Marshal(&vmirs)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.VirtualMachineInstanceReplicaSetGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: vmirsBytes,
					},
				},
			}

			resp := admitVMIRS(ar)
			Expect(resp.Allowed).To(Equal(true))
		})
	})

	Context("with VM admission review", func() {
		It("reject invalid VirtualMachineInstance spec", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})
			vm := &v1.VirtualMachine{
				Spec: v1.VirtualMachineSpec{
					Running: false,
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

			resp := admitVMs(ar)
			Expect(resp.Allowed).To(Equal(false))
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
					Running: false,
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

			resp := admitVMs(ar)
			Expect(resp.Allowed).To(Equal(true))
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
					Running: false,
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						Spec: vmi.Spec,
					},
				},
			}

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
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

			os.Setenv("FEATURE_GATES", "DataVolumes")
			resp := admitVMs(ar)
			Expect(resp.Allowed).To(Equal(true))
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
					Running: false,
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						Spec: vmi.Spec,
					},
				},
			}

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
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

			os.Setenv("FEATURE_GATES", "DataVolumes")
			resp := admitVMs(ar)
			Expect(resp.Allowed).To(Equal(false))
			Expect(len(resp.Result.Details.Causes)).To(Equal(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.dataVolumeTemplate[0]"))
		})
	})
	Context("with VMIPreset admission review", func() {
		It("reject invalid VirtualMachineInstance spec", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmiPDomain := &v1.DomainSpec{}
			vmiDomainByte, _ := json.Marshal(vmi.Spec.Domain)
			Expect(json.Unmarshal(vmiDomainByte, &vmiPDomain)).To(BeNil())

			vmiPDomain.Devices.Disks = append(vmiPDomain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				DiskDevice: v1.DiskDevice{
					Disk:   &v1.DiskTarget{},
					Floppy: &v1.FloppyTarget{},
				},
			})
			vmiPreset := &v1.VirtualMachineInstancePreset{
				Spec: v1.VirtualMachineInstancePresetSpec{
					Domain: vmiPDomain,
				},
			}
			vmiPresetBytes, _ := json.Marshal(vmiPreset)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.VirtualMachineInstancePresetGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: vmiPresetBytes,
					},
				},
			}

			resp := admitVMIPreset(ar)
			Expect(resp.Allowed).To(Equal(false))
			Expect(len(resp.Result.Details.Causes)).To(Equal(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.domain.devices.disks[0]"))
		})
		It("should accept valid vmi spec", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})

			vmiPreset := &v1.VirtualMachineInstancePreset{
				Spec: v1.VirtualMachineInstancePresetSpec{
					Domain: &v1.DomainSpec{},
				},
			}
			vmiPresetBytes, _ := json.Marshal(&vmiPreset)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.VirtualMachineInstancePresetGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: vmiPresetBytes,
					},
				},
			}

			resp := admitVMIPreset(ar)
			Expect(resp.Allowed).To(Equal(true))
		})
	})

	Context("with VirtualMachineInstanceMigration admission review", func() {
		It("should reject invalid Migration spec on create", func() {
			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "",
				},
			}
			migrationBytes, _ := json.Marshal(&migration)

			os.Setenv("FEATURE_GATES", "LiveMigration")

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.MigrationGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: migrationBytes,
					},
				},
			}

			resp := admitMigrationCreate(ar)
			Expect(resp.Allowed).To(Equal(false))
			Expect(len(resp.Result.Details.Causes)).To(Equal(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.vmiName"))
		})

		It("should accept valid Migration spec on create", func() {
			vmi := v1.NewMinimalVMI("testvmimigrate1")

			informers := webhooks.GetInformers()
			informers.VMIInformer.GetIndexer().Add(vmi)

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: vmi.Namespace,
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "testvmimigrate1",
				},
			}
			migrationBytes, _ := json.Marshal(&migration)

			os.Setenv("FEATURE_GATES", "LiveMigration")

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.MigrationGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: migrationBytes,
					},
				},
			}

			resp := admitMigrationCreate(ar)
			Expect(resp.Allowed).To(Equal(true))
		})

		It("should reject valid Migration spec on create when feature gate isn't enabled", func() {
			vmi := v1.NewMinimalVMI("testvmimigrate1")

			informers := webhooks.GetInformers()
			informers.VMIInformer.GetIndexer().Add(vmi)

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: vmi.Namespace,
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "testvmimigrate1",
				},
			}
			migrationBytes, _ := json.Marshal(&migration)

			os.Setenv("FEATURE_GATES", "")

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.MigrationGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: migrationBytes,
					},
				},
			}

			resp := admitMigrationCreate(ar)
			Expect(resp.Allowed).To(Equal(false))
		})

		It("should reject Migration spec on create when another VMI migration is in-flight", func() {
			vmi := v1.NewMinimalVMI("testmigratevmi2")
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "123",
				Completed:    false,
				Failed:       false,
			}

			informers := webhooks.GetInformers()
			informers.VMIInformer.GetIndexer().Add(vmi)

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "testmigratevmi2",
				},
			}
			migrationBytes, _ := json.Marshal(&migration)

			os.Setenv("FEATURE_GATES", "LiveMigration")

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.MigrationGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: migrationBytes,
					},
				},
			}

			resp := admitMigrationCreate(ar)
			Expect(resp.Allowed).To(Equal(false))
		})

		It("should accept Migration spec on create when previous VMI migration completed", func() {
			vmi := v1.NewMinimalVMI("testmigratevmi4")
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "123",
				Completed:    true,
				Failed:       false,
			}

			informers := webhooks.GetInformers()
			informers.VMIInformer.GetIndexer().Add(vmi)

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: vmi.Namespace,
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "testmigratevmi4",
				},
			}
			migrationBytes, _ := json.Marshal(&migration)

			os.Setenv("FEATURE_GATES", "LiveMigration")

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.MigrationGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: migrationBytes,
					},
				},
			}

			resp := admitMigrationCreate(ar)
			Expect(resp.Allowed).To(Equal(true))
		})

		It("should reject Migration spec on create when VMI is finalized", func() {
			vmi := v1.NewMinimalVMI("testmigratevmi3")
			vmi.Status.Phase = v1.Succeeded

			informers := webhooks.GetInformers()
			informers.VMIInformer.GetIndexer().Add(vmi)

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "testmigratevmi3",
				},
			}
			migrationBytes, _ := json.Marshal(&migration)

			os.Setenv("FEATURE_GATES", "LiveMigration")

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.MigrationGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: migrationBytes,
					},
				},
			}

			resp := admitMigrationCreate(ar)
			Expect(resp.Allowed).To(Equal(false))
		})

		It("should reject Migration spec for non-migratable VMIs", func() {
			vmi := v1.NewMinimalVMI("testmigratevmi3")
			vmi.Status.Phase = v1.Succeeded
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:    v1.VirtualMachineInstanceIsMigratable,
					Status:  k8sv1.ConditionFalse,
					Reason:  v1.VirtualMachineInstanceReasonDisksNotMigratable,
					Message: "cannot migrate VMI with mixes shared and non-shared volumes",
				},
			}

			informers := webhooks.GetInformers()
			informers.VMIInformer.GetIndexer().Add(vmi)

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "testmigratevmi3",
				},
			}
			migrationBytes, _ := json.Marshal(&migration)

			os.Setenv("FEATURE_GATES", "LiveMigration")

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.MigrationGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: migrationBytes,
					},
				},
			}

			resp := admitMigrationCreate(ar)
			Expect(resp.Allowed).To(Equal(false))
		})

		It("should reject Migration on update if spec changes", func() {
			vmi := v1.NewMinimalVMI("testmigratevmiupdate")

			informers := webhooks.GetInformers()
			informers.VMIInformer.GetIndexer().Add(vmi)

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "somemigrationthatchanged",
					Namespace: "default",
					UID:       "abc",
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "testmigratevmiupdate",
				},
			}
			oldMigrationBytes, _ := json.Marshal(&migration)

			newMigration := migration.DeepCopy()
			newMigration.Spec.VMIName = "somethingelse"
			newMigrationBytes, _ := json.Marshal(&newMigration)

			os.Setenv("FEATURE_GATES", "LiveMigration")

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.MigrationGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: newMigrationBytes,
					},
					OldObject: runtime.RawExtension{
						Raw: oldMigrationBytes,
					},
					Operation: v1beta1.Update,
				},
			}

			resp := admitMigrationUpdate(ar)
			Expect(resp.Allowed).To(Equal(false))
		})

		It("should accept Migration on update if spec doesn't change", func() {
			vmi := v1.NewMinimalVMI("testmigratevmiupdate-nochange")

			informers := webhooks.GetInformers()
			informers.VMIInformer.GetIndexer().Add(vmi)

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "somemigration",
					Namespace: "default",
					UID:       "1234",
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "testmigratevmiupdate-nochange",
				},
			}

			migrationBytes, _ := json.Marshal(&migration)

			os.Setenv("FEATURE_GATES", "LiveMigration")

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.MigrationGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: migrationBytes,
					},
					OldObject: runtime.RawExtension{
						Raw: migrationBytes,
					},
					Operation: v1beta1.Update,
				},
			}

			resp := admitMigrationUpdate(ar)
			Expect(resp.Allowed).To(Equal(true))
		})
	})

	Context("with VirtualMachineInstance spec", func() {
		It("should accept valid subdomain name", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Subdomain = "testsubdomain"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject invalid subdomain name", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Subdomain = "bad+domain"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.subdomain"))
		})
		It("should accept disk and volume lists equal to max element length", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			for i := 0; i < arrayLenMax; i++ {
				diskName := fmt.Sprintf("testDisk%d", i)
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: diskName,
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: diskName,
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{},
					},
				})
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject disk lists greater than max element length", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			for i := 0; i <= arrayLenMax; i++ {
				diskName := "testDisk"
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: diskName,
				})
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			// if this is processed correctly, it should result in a single error
			// If multiple causes occurred, then the spec was processed too far.
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks"))
		})
		It("should reject volume lists greater than max element length", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			for i := 0; i <= arrayLenMax; i++ {
				volumeName := "testVolume"
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: volumeName,
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{},
					},
				})
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			// if this is processed correctly, it should result in a single error
			// If multiple causes occurred, then the spec was processed too far.
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.volumes"))
		})

		It("should reject disk with missing volume", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[0].name"))
		})
		It("should reject multiple disks referencing same volume", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			// verify two disks referencing the same volume are rejected
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testdisk",
				VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{},
				},
			})
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[1].name"))
		})
		It("should generate multiple causes", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				DiskDevice: v1.DiskDevice{
					Disk:   &v1.DiskTarget{},
					Floppy: &v1.FloppyTarget{},
				},
			})

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			// missing volume and multiple targets set. should result in 2 causes
			Expect(len(causes)).To(Equal(2))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[0].name"))
			Expect(causes[1].Field).To(Equal("fake.domain.devices.disks[0]"))
		})
		It("should reject negative requests.memory size value", func() {
			vm := v1.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("-64Mi"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})
		It("should reject negative limits.memory size value", func() {
			vm := v1.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("-65Mi"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.limits.memory"))
		})
		It("should reject greater requests.memory than limits.memory", func() {
			vm := v1.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("128Mi"),
			}
			vm.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})
		It("should accept correct memory size values", func() {
			vm := v1.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vm.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("65Mi"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject incorrect hugepages size format", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{}}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "2ab"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.hugepages.size"))
		})
		It("should reject greater hugepages.size than requests.memory", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{}}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "1Gi"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})
		It("should reject smaller guest memory than requested memory", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			guestMemory := resource.MustParse("1Mi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.memory.guest"))
		})
		It("should reject bigger guest memory than the memory limit", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			guestMemory := resource.MustParse("128Mi")

			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.memory.guest"))
		})
		It("should allow guest memory which is between requests and limits", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			guestMemory := resource.MustParse("100Mi")

			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("128Mi"),
			}
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(causes).To(BeEmpty())
		})
		It("should allow setting guest memory when no limit is set", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			guestMemory := resource.MustParse("100Mi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(causes).To(BeEmpty())
		})
		It("should reject not divisable by hugepages.size requests.memory", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("65Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{}}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "2Gi"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})
		It("should accept correct memory and hugepages size values", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{}}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "2Mi"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		table.DescribeTable("should verify LUN is mapped to PVC volume",
			func(volume *v1.Volume, expectedErrors int) {
				vmi := v1.NewMinimalVMI("testvmi")
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "testdisk",
					DiskDevice: v1.DiskDevice{
						LUN: &v1.LunTarget{},
					},
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, *volume)

				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
				Expect(len(causes)).To(Equal(expectedErrors))
			},
			table.Entry("and reject non PVC sources",
				&v1.Volume{
					Name: "testdisk",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{},
					},
				}, 1),
			table.Entry("and accept PVC sources",
				&v1.Volume{
					Name: "testdisk",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{},
					},
				}, 0),
		)
		It("should accept a single interface and network", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vm.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject interface lists with more than one interface with the same name", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultNetworkInterface(),
				*v1.DefaultNetworkInterface()}
			vm.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			// if this is processed correctly, it should result an error
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[1].name"))
		})
		It("should accept network lists with more than one element", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
				{Name: "default2", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
			vm.Spec.Networks = []v1.Network{{Name: "default", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
				{Name: "default2", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			// if this is processed correctly, it should result an error only about duplicate pod network configuration
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Message).To(Equal("more than one interface is connected to a pod network in fake.interfaces"))
		})

		It("should accept valid interface models", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

			for _, model := range validInterfaceModels {
				vmi.Spec.Domain.Devices.Interfaces[0].Model = model
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
				// if this is processed correctly, it should not result in any error
				Expect(len(causes)).To(Equal(0))
			}
		})

		It("should reject invalid interface model", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces[0].Model = "invalid_model"
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
		})

		It("should reject interfaces with missing network", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name: "redtest",
					NetworkSource: v1.NetworkSource{
						Pod: &v1.PodNetwork{},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].name"))
		})
		It("should reject unassign multus network", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vm.Spec.Networks = []v1.Network{
				{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Pod: &v1.PodNetwork{},
					},
				},
				{
					Name: "redtest",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.CniNetwork{NetworkName: "test-conf"},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.networks"))
		})
		It("should accept networks with a pod network source and bridge interface", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		It("should accept networks with a multus network source and bridge interface", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.CniNetwork{NetworkName: "default"},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(causes).To(BeEmpty())
		})
		It("should accept networks with a genie network source and bridge interface", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Genie: &v1.CniNetwork{NetworkName: "default"},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(causes).To(BeEmpty())
		})
		It("should reject when multiple types defined for a CNI network", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.CniNetwork{NetworkName: "default1"},
						Genie:  &v1.CniNetwork{NetworkName: "default2"},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.networks[0]"))
		})
		It("should allow multiple networks of same CNI type", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultNetworkInterface(),
				*v1.DefaultNetworkInterface(),
				*v1.DefaultNetworkInterface(),
			}
			vm.Spec.Domain.Devices.Interfaces[0].Name = "multus1"
			vm.Spec.Domain.Devices.Interfaces[1].Name = "multus2"
			// 3rd interfaces uses the default pod network, name is "default"
			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Pod: &v1.PodNetwork{},
					},
				},
				v1.Network{
					Name: "multus1",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.CniNetwork{NetworkName: "multus-net1"},
					},
				},
				v1.Network{
					Name: "multus2",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.CniNetwork{NetworkName: "multus-net2"},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(causes).To(BeEmpty())
		})
		It("should reject when CNI networks of different types are defined", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultNetworkInterface(),
				*v1.DefaultNetworkInterface(),
			}
			vm.Spec.Domain.Devices.Interfaces[0].Name = "multus"
			vm.Spec.Domain.Devices.Interfaces[1].Name = "genie"
			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name: "multus",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.CniNetwork{NetworkName: "default1"},
					},
				},
				v1.Network{
					Name: "genie",
					NetworkSource: v1.NetworkSource{
						Genie: &v1.CniNetwork{NetworkName: "default2"},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.networks[1]"))
		})
		It("should reject pod and Genie CNI networks combination", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultNetworkInterface(),
				*v1.DefaultNetworkInterface(),
			}
			// 1st network is the default pod network, name is "default"
			vm.Spec.Domain.Devices.Interfaces[1].Name = "genie"
			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Pod: &v1.PodNetwork{},
					},
				},
				v1.Network{
					Name: "genie",
					NetworkSource: v1.NetworkSource{
						Genie: &v1.CniNetwork{NetworkName: "genie-net"},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.networks[1]"))
		})
		It("should reject multus network source without networkName", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.CniNetwork{},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.networks[0]"))
		})
		It("should reject networks with a multus network source and slirp interface", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				}}}
			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.CniNetwork{NetworkName: "default"},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
		})
		It("should accept networks with a pod network source and slirp interface", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				}}}

			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		It("should accept networks with a pod network source and slirp interface with port", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Port: 80}}}}

			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject networks with a pod network source and slirp interface without specific port", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Name: "test"}}}}

			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].ports[0]"))
		})
		It("should reject a masquerade interface on a network different than pod", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Masquerade: &v1.InterfaceMasquerade{},
				},
				Ports: []v1.Port{{Name: "test"}}}}

			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Multus: &v1.CniNetwork{NetworkName: "test"}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].name"))
		})
		It("should reject networks with a pod network source and slirp interface with bad protocol type", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Protocol: "bad", Port: 80}}}}

			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].ports[0].protocol"))
		})
		It("should accept networks with a pod network source and slirp interface with multiple Ports", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Port: 80}, {Protocol: "UDP", Port: 80}}}}

			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject port out of range", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Port: 80000}}}}

			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].ports[0]"))
		})
		It("should reject interface with two ports with the same name", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Name: "testPort", Port: 80}, {Name: "testPort", Protocol: "UDP", Port: 80}}}}

			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].ports[1].name"))
		})
		It("should reject two interfaces with same port name", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Name: "testPort", Port: 80}}},
				v1.Interface{
					Name: "default",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						Slirp: &v1.InterfaceSlirp{},
					},
					Ports: []v1.Port{{Name: "testPort", Protocol: "UDP", Port: 80}}}}

			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(2))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[1].name"))
			Expect(causes[1].Field).To(Equal("fake.domain.devices.interfaces[1].ports[0].name"))
		})
		It("should allow interface with two same ports and protocol", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Port: 80}, {Protocol: "UDP", Port: 80}, {Protocol: "TCP", Port: 80}}}}

			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject specs with multiple pod interfaces", func() {
			vm := v1.NewMinimalVMI("testvm")
			for i := 1; i < 3; i++ {
				iface := v1.DefaultNetworkInterface()
				net := v1.DefaultPodNetwork()

				// make sure whatever the error we receive is not related to duplicate names
				name := fmt.Sprintf("podnet%d", i)
				iface.Name = name
				net.Name = name

				vm.Spec.Domain.Devices.Interfaces = append(vm.Spec.Domain.Devices.Interfaces, *iface)
				vm.Spec.Networks = append(vm.Spec.Networks, *net)
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.interfaces"))
		})

		It("should accept valid MAC address", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			for _, macAddress := range []string{"de:ad:00:00:be:af", "de-ad-00-00-be-af"} {
				vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = macAddress // missing octet
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
				// if this is processed correctly, it should not result in any error
				Expect(len(causes)).To(Equal(0))
			}
		})

		It("should reject invalid MAC addresses", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			for _, macAddress := range []string{"de:ad:00:00:be", "de-ad-00-00-be", "de:ad:00:00:be:af:be:af"} {
				vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = macAddress
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
				Expect(len(causes)).To(Equal(1))
				Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].macAddress"))
			}
		})
		It("should accept valid PCI address", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			for _, pciAddress := range []string{"0000:81:11.1", "0001:02:00.0"} {
				vmi.Spec.Domain.Devices.Interfaces[0].PciAddress = pciAddress
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
				// if this is processed correctly, it should not result in any error
				Expect(len(causes)).To(Equal(0))
			}
		})

		It("should reject invalid PCI addresses", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			for _, pciAddress := range []string{"0000:80.10.1", "0000:80:80:1.0", "0000:80:11.15"} {
				vmi.Spec.Domain.Devices.Interfaces[0].PciAddress = pciAddress
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
				Expect(len(causes)).To(Equal(1))
				Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].pciAddress"))
			}
		})

		It("should accept valid NTP servers", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces[0].DHCPOptions = &v1.DHCPOptions{
				NTPServers: []string{"127.0.0.1", "127.0.0.2"},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(0))
		})

		It("should reject non-IPv4 NTP servers", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces[0].DHCPOptions = &v1.DHCPOptions{
				NTPServers: []string{"::1", "hostname"},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(2))
		})

		It("should reject vmi with a network multiqueue, without virtio nics", func() {
			_true := true
			vmi := v1.NewMinimalVMI("testvm")
			nic := *v1.DefaultNetworkInterface()
			nic.Model = "e1000"
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{nic}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue = &_true
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(2))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.networkInterfaceMultiqueue"))
		})

		It("should reject nic multi queue without CPU settings", func() {
			_true := true
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue = &_true

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.networkInterfaceMultiqueue"))
		})

		It("should reject BlockMultiQueue without CPU settings", func() {
			_true := true
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.BlockMultiQueue = &_true

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.blockMultiQueue"))
		})

		It("should allow BlockMultiQueue with CPU settings", func() {
			_true := true
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.BlockMultiQueue = &_true
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{}
			vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU] = resource.MustParse("5")

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(0))
		})

		It("should ignore CPU settings for explicitly rejected BlockMultiQueue", func() {
			_false := false
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.BlockMultiQueue = &_false

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(0))
		})

		It("should reject SRIOV interface when feature gate is disabled", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

			vmi.Spec.Domain.Devices.Interfaces = append(
				vmi.Spec.Domain.Devices.Interfaces,
				v1.Interface{
					Name: "sriov",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						SRIOV: &v1.InterfaceSRIOV{},
					},
				},
			)
			vmi.Spec.Networks = append(
				vmi.Spec.Networks,
				v1.Network{
					Name: "sriov",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.CniNetwork{NetworkName: "sriov"},
					},
				},
			)

			os.Setenv("FEATURE_GATES", "")

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[1].name"))
		})
	})
	Context("with cpu pinning", func() {
		var vmi *v1.VirtualMachineInstance
		BeforeEach(func() {
			vmi = v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.CPU = &v1.CPU{DedicatedCPUPlacement: true}
		})
		It("should reject specs without cpu reqirements", func() {
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.cpu.dedicatedCpuPlacement"))
		})
		It("should reject specs without inconsistent cpu reqirements", func() {
			vmi.Spec.Domain.CPU.Cores = 4
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("2"),
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.cpu.dedicatedCpuPlacement"))
		})
		It("should reject specs with non-integer cpu limits values", func() {
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("800m"),
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.limits.cpu"))
		})
		It("should reject specs with non-integer cpu requests values", func() {
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceCPU:    resource.MustParse("800m"),
				k8sv1.ResourceMemory: resource.MustParse("8Mi"),
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.cpu"))
		})
		It("should not allow cpu overcommit", func() {
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceCPU:    resource.MustParse("4"),
				k8sv1.ResourceMemory: resource.MustParse("8Mi"),
			}
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceCPU:    resource.MustParse("2"),
				k8sv1.ResourceMemory: resource.MustParse("8Mi"),
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.cpu.dedicatedCpuPlacement"))
		})
		It("should reject specs without a memory specification", func() {
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("4"),
			}
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("4"),
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.limits.memory"))
		})
		It("should reject specs with inconsistent memory specification", func() {
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceCPU:    resource.MustParse("1"),
				k8sv1.ResourceMemory: resource.MustParse("8Mi"),
			}
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceCPU:    resource.MustParse("1"),
				k8sv1.ResourceMemory: resource.MustParse("4Mi"),
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})
	})
	Context("with Volume", func() {
		table.DescribeTable("should accept valid volumes",
			func(volumeSource v1.VolumeSource) {
				vmi := v1.NewMinimalVMI("testvmi")
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name:         "testvolume",
					VolumeSource: volumeSource,
				})

				os.Setenv("FEATURE_GATES", "DataVolumes")
				causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
				Expect(len(causes)).To(Equal(0))
			},
			table.Entry("with pvc volume source", v1.VolumeSource{PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{}}),
			table.Entry("with cloud-init volume source", v1.VolumeSource{CloudInitNoCloud: &v1.CloudInitNoCloudSource{UserData: "fake"}}),
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

			os.Setenv("FEATURE_GATES", "")
			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0]"))
		})
		It("should reject volume with no volume source set", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
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

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
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

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[1].name"))
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

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
			Expect(len(causes)).To(Equal(expectedErrors))
			for _, cause := range causes {
				Expect(cause.Field).To(ContainSubstring("fake[0].cloudInitNoCloud"))
			}
		},
			table.Entry("should accept userdata under max limit", 10, 0, false),
			table.Entry("should accept userdata equal max limit", cloudInitMaxLen, 0, false),
			table.Entry("should reject userdata greater than max limit", cloudInitMaxLen+1, 1, false),
			table.Entry("should accept userdata base64 under max limit", 10, 0, true),
			table.Entry("should accept userdata base64 equal max limit", cloudInitMaxLen, 0, true),
			table.Entry("should reject userdata base64 greater than max limit", cloudInitMaxLen+1, 1, true),
		)

		It("should reject cloud-init with invalid base64 data", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					CloudInitNoCloud: &v1.CloudInitNoCloudSource{
						UserDataBase64: "#######garbage******",
					},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].cloudInitNoCloud.userDataBase64"))
		})

		It("should reject hostDisk without required parameters", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					HostDisk: &v1.HostDisk{},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
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

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
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

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
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

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
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

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
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

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
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

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
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

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake"))
		})
	})

	Context("with Disk", func() {
		table.DescribeTable("should accept valid disks",
			func(disk v1.Disk) {
				vmi := v1.NewMinimalVMI("testvmi")

				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, disk)

				causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
				Expect(len(causes)).To(Equal(0))

			},
			table.Entry("with Disk target",
				v1.Disk{Name: "testdisk", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{}}},
			),
			table.Entry("with LUN target",
				v1.Disk{Name: "testdisk", DiskDevice: v1.DiskDevice{LUN: &v1.LunTarget{}}},
			),
			table.Entry("with Floppy target",
				v1.Disk{Name: "testdisk", DiskDevice: v1.DiskDevice{Floppy: &v1.FloppyTarget{}}},
			),
			table.Entry("with CDRom target",
				v1.Disk{Name: "testdisk", DiskDevice: v1.DiskDevice{CDRom: &v1.CDRomTarget{}}},
			),
		)
		It("should allow disk without a target", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				// disk without a target defaults to DiskTarget
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{Image: "fake"},
				},
			})

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject disks with duplicate names ", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})
			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[1].name"))
		})

		It("should reject disks with PCI address on a non-virtio bus ", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						PciAddress: "0000:04:10.0",
						Bus:        "scsi"},
				},
			})
			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks.disk[0].pciAddress"))
		})

		It("should reject disks malformed PCI addresses ", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						PciAddress: "0000:81:100.a",
						Bus:        "virtio"},
				},
			})
			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks.disk[0].pciAddress"))
		})

		It("should reject disk with multiple targets ", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				DiskDevice: v1.DiskDevice{
					Disk:   &v1.DiskTarget{},
					Floppy: &v1.FloppyTarget{},
				},
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{Image: "fake"},
				},
			})

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0]"))
		})

		It("should accept a boot order greater than '0'", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			order := uint(1)

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:      "testdisk",
				BootOrder: &order,
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(0))
		})

		It("should reject a disk with a boot order of '0'", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			order := uint(0)

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:      "testdisk",
				BootOrder: &order,
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].bootOrder"))
		})

		It("should accept disks with supported or unspecified buses", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk1",
				VolumeName: "testvolume1",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: "virtio",
					},
				},
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk2",
				VolumeName: "testvolume2",
				DiskDevice: v1.DiskDevice{
					LUN: &v1.LunTarget{
						Bus: "sata",
					},
				},
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk3",
				VolumeName: "testvolume3",
				DiskDevice: v1.DiskDevice{
					CDRom: &v1.CDRomTarget{
						Bus: "scsi",
					},
				},
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk4",
				VolumeName: "testvolume4",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(0))
		})

		It("should reject disks with unsupported buses", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk1",
				VolumeName: "testvolume1",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: "ide",
					},
				},
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk2",
				VolumeName: "testvolume2",
				DiskDevice: v1.DiskDevice{
					LUN: &v1.LunTarget{
						Bus: "unsupported",
					},
				},
			})

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(2))
			Expect(causes[0].Field).To(Equal("fake[0].disk.bus"))
			Expect(causes[1].Field).To(Equal("fake[1].lun.bus"))
		})

		It("should reject invalid SN characters", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			order := uint(1)
			sn := "$$$$"

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:      "testdisk2",
				BootOrder: &order,
				Serial:    sn,
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].serial"))
		})

		It("should reject SN > maxStrLen characters", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			order := uint(1)
			sn := strings.Repeat("1", maxStrLen+1)

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:      "testdisk2",
				BootOrder: &order,
				Serial:    sn,
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].serial"))
		})

		It("should accept valid SN", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			order := uint(1)

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:      "testdisk2",
				BootOrder: &order,
				Serial:    "SN-1_a",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(0))
		})

	})
})

var _ = Describe("Function getNumberOfPodInterfaces()", func() {

	It("should work for empty network list", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		Expect(getNumberOfPodInterfaces(spec)).To(Equal(0))
	})

	It("should work for non-empty network list without pod network", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		net := v1.Network{}
		spec.Networks = []v1.Network{net}
		Expect(getNumberOfPodInterfaces(spec)).To(Equal(0))
	})

	It("should work for pod network with missing pod interface", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		net := v1.Network{
			NetworkSource: v1.NetworkSource{
				Pod: &v1.PodNetwork{},
			},
		}
		spec.Networks = []v1.Network{net}
		Expect(getNumberOfPodInterfaces(spec)).To(Equal(0))
	})

	It("should work for valid pod network / interface combination", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		net := v1.Network{
			NetworkSource: v1.NetworkSource{
				Pod: &v1.PodNetwork{},
			},
			Name: "testnet",
		}
		iface := v1.Interface{Name: net.Name}
		spec.Networks = []v1.Network{net}
		spec.Domain.Devices.Interfaces = []v1.Interface{iface}
		Expect(getNumberOfPodInterfaces(spec)).To(Equal(1))
	})

	It("should work for multiple pod network / interface combinations", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		net1 := v1.Network{
			NetworkSource: v1.NetworkSource{
				Pod: &v1.PodNetwork{},
			},
			Name: "testnet1",
		}
		iface1 := v1.Interface{Name: net1.Name}
		net2 := v1.Network{
			NetworkSource: v1.NetworkSource{
				Pod: &v1.PodNetwork{},
			},
			Name: "testnet2",
		}
		iface2 := v1.Interface{Name: net2.Name}
		spec.Networks = []v1.Network{net1, net2}
		spec.Domain.Devices.Interfaces = []v1.Interface{iface1, iface2}
		Expect(getNumberOfPodInterfaces(spec)).To(Equal(2))
	})
	It("when network source is not configured", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		net1 := v1.Network{
			NetworkSource: v1.NetworkSource{},
			Name:          "testnet1",
		}
		iface1 := v1.Interface{Name: net1.Name}
		spec.Networks = []v1.Network{net1}
		spec.Domain.Devices.Interfaces = []v1.Interface{iface1}
		causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), spec)
		Expect(causes).To(HaveLen(1))
	})
	It("should reject when more than one network source is configured", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		net1 := v1.Network{
			NetworkSource: v1.NetworkSource{
				Pod:    &v1.PodNetwork{},
				Multus: &v1.CniNetwork{NetworkName: "testnet1"},
			},
		}
		iface1 := v1.Interface{Name: net1.Name}
		spec.Networks = []v1.Network{net1}
		spec.Domain.Devices.Interfaces = []v1.Interface{iface1}
		causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), spec)
		Expect(causes).To(HaveLen(1))
	})
	It("should work when boot order is given to interfaces", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		net := v1.Network{
			NetworkSource: v1.NetworkSource{
				Pod: &v1.PodNetwork{},
			},
			Name: "testnet",
		}
		order := uint(1)
		iface := v1.Interface{Name: net.Name, BootOrder: &order}
		spec.Networks = []v1.Network{net}
		spec.Domain.Devices.Interfaces = []v1.Interface{iface}
		causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), spec)
		Expect(causes).To(HaveLen(0))
	})
	It("should fail when invalid boot order is given to interface", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		net := v1.Network{
			NetworkSource: v1.NetworkSource{
				Pod: &v1.PodNetwork{},
			},
			Name: "testnet",
		}
		order := uint(0)
		iface := v1.Interface{Name: net.Name, BootOrder: &order}
		spec.Networks = []v1.Network{net}
		spec.Domain.Devices.Interfaces = []v1.Interface{iface}
		causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), spec)
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Field).To(Equal("fake[0].bootOrder"))
	})
	It("should work when different boot orders are given to devices", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		net := v1.Network{
			NetworkSource: v1.NetworkSource{
				Pod: &v1.PodNetwork{},
			},
			Name: "testnet",
		}
		order1 := uint(7)
		iface := v1.Interface{Name: net.Name, BootOrder: &order1}
		spec.Networks = []v1.Network{net}
		spec.Domain.Devices.Interfaces = []v1.Interface{iface}
		order2 := uint(77)
		disk := v1.Disk{
			Name:      "testdisk",
			BootOrder: &order2,
			Serial:    "SN-1_a",
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{},
			},
		}
		spec.Domain.Devices.Disks = []v1.Disk{disk}
		volume := v1.Volume{
			Name: "testdisk",
			VolumeSource: v1.VolumeSource{
				ContainerDisk: &v1.ContainerDiskSource{},
			},
		}

		spec.Volumes = []v1.Volume{volume}
		spec.Domain.Devices.Disks = []v1.Disk{disk}
		causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), spec)
		Expect(causes).To(HaveLen(0))
	})
	It("should fail when same boot order is given to more than one device", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		net := v1.Network{
			NetworkSource: v1.NetworkSource{
				Pod: &v1.PodNetwork{},
			},
			Name: "testnet",
		}
		order := uint(7)
		iface := v1.Interface{Name: net.Name, BootOrder: &order}
		spec.Networks = []v1.Network{net}
		spec.Domain.Devices.Interfaces = []v1.Interface{iface}
		disk := v1.Disk{
			Name:      "testdisk",
			BootOrder: &order,
			Serial:    "SN-1_a",
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{},
			},
		}
		spec.Domain.Devices.Disks = []v1.Disk{disk}
		volume := v1.Volume{
			Name: "testdisk",
			VolumeSource: v1.VolumeSource{
				ContainerDisk: &v1.ContainerDiskSource{},
			},
		}
		spec.Volumes = []v1.Volume{volume}

		causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), spec)
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Field).To(ContainSubstring("bootOrder"))
	})
})

type virtualMachineBuilder struct {
	disks   []v1.Disk
	volumes []v1.Volume
	labels  map[string]string
}

func (b *virtualMachineBuilder) WithDisk(disk v1.Disk) *virtualMachineBuilder {
	b.disks = append(b.disks, disk)
	return b
}

func (b *virtualMachineBuilder) WithLabel(key string, value string) *virtualMachineBuilder {
	b.labels[key] = value
	return b
}

func (b *virtualMachineBuilder) WithVolume(volume v1.Volume) *virtualMachineBuilder {
	b.volumes = append(b.volumes, volume)
	return b
}

func (b *virtualMachineBuilder) Build() *v1.VirtualMachineInstance {

	vmi := v1.NewMinimalVMI("testvmi")
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, b.disks...)
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, b.volumes...)
	vmi.Labels = b.labels

	return vmi
}

func (b *virtualMachineBuilder) BuildTemplate() *v1.VirtualMachineInstanceTemplateSpec {
	vmi := b.Build()

	return &v1.VirtualMachineInstanceTemplateSpec{
		ObjectMeta: vmi.ObjectMeta,
		Spec:       vmi.Spec,
	}

}

func newVirtualMachineBuilder() *virtualMachineBuilder {
	return &virtualMachineBuilder{
		labels: map[string]string{},
	}
}
