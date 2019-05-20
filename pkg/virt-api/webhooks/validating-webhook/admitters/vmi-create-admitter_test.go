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
	"encoding/json"
	"fmt"
	"strconv"
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
	"k8s.io/apimachinery/pkg/util/validation"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Validating VMICreate Admitter", func() {
	config, configMapInformer := testutils.NewFakeClusterConfig(&k8sv1.ConfigMap{})
	vmiCreateAdmitter := &VMICreateAdmitter{ClusterConfig: config}

	dnsConfigTestOption := "test"
	enableFeatureGate := func(featureGate string) {
		testutils.UpdateFakeClusterConfig(configMapInformer, &k8sv1.ConfigMap{
			Data: map[string]string{virtconfig.FeatureGatesKey: featureGate},
		})
	}
	disableFeatureGates := func() {
		testutils.UpdateFakeClusterConfig(configMapInformer, &k8sv1.ConfigMap{})
	}

	AfterEach(func() {
		disableFeatureGates()
	})

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

		resp := vmiCreateAdmitter.Admit(ar)
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
		resp := vmiCreateAdmitter.Admit(ar)
		Expect(resp.Allowed).To(Equal(false))
		Expect(resp.Result.Message).To(ContainSubstring("no memory requested"))
	})

	Context("tolerations with eviction policies given", func() {
		var vmi *v1.VirtualMachineInstance
		var policy = v1.EvictionStrategyLiveMigrate
		BeforeEach(func() {
			enableFeatureGate("LiveMigration")
			vmi = v1.NewMinimalVMI("testvmi")
			vmi.Spec.EvictionStrategy = nil
		})

		table.DescribeTable("it should allow", func(policy v1.EvictionStrategy) {
			vmi.Spec.EvictionStrategy = &policy
			resp := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(resp).To(BeEmpty())
		},
			table.Entry("migration policy to be set", v1.EvictionStrategyLiveMigrate),
		)

		It("should block setting eviction policies if the feature gate is disabled", func() {
			disableFeatureGates()
			vmi.Spec.EvictionStrategy = &policy
			resp := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(resp[0].Message).To(ContainSubstring("LiveMigration feature gate is not enabled"))
		})

		It("should allow no eviction policy to be set", func() {
			vmi.Spec.EvictionStrategy = nil
			resp := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(resp).To(BeEmpty())
		})

		It("should  not allow unknown eviction policies", func() {
			policy := v1.EvictionStrategy("fantasy")
			vmi.Spec.EvictionStrategy = &policy
			resp := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(resp).To(HaveLen(1))
			Expect(resp[0].Message).To(Equal("fake.evictionStrategy is set with an unrecognized option: fantasy"))
		})
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
			resp := vmiCreateAdmitter.Admit(ar)
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
			resp := vmiCreateAdmitter.Admit(ar)
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
			resp := vmiCreateAdmitter.Admit(ar)
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
			resp := vmiCreateAdmitter.Admit(ar)
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
		resp := vmiCreateAdmitter.Admit(ar)
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
		resp := vmiCreateAdmitter.Admit(ar)
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
			vmiCreateAdmitter.Admit,
		),
	)

	Context("with VirtualMachineInstance spec", func() {
		It("should accept valid machine type", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Machine.Type = "q35"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject invalid machine type", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Machine.Type = "test"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake.domain.machine.type"))
			Expect(causes[0].Message).To(ContainSubstring("fake.domain.machine.type is not supported: test (allowed values:"))
		})

		It("should accept valid hostname", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Hostname = "test"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject invalid hostname", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Hostname = "test+bad"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake.hostname"))
			Expect(causes[0].Message).To(ContainSubstring("does not conform to the kubernetes DNS_LABEL rules : "))
		})
		It("should accept valid subdomain name", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Subdomain = "testsubdomain"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject invalid subdomain name", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Subdomain = "bad+domain"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[1].name"))
		})
		It("should generate multiple causes", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				DiskDevice: v1.DiskDevice{
					Disk:  &v1.DiskTarget{},
					CDRom: &v1.CDRomTarget{},
				},
			})

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			// missing volume and multiple targets set. should result in 2 causes
			Expect(len(causes)).To(Equal(2))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[0].name"))
			Expect(causes[1].Field).To(Equal("fake.domain.devices.disks[0]"))
		})

		table.DescribeTable("should verify input device",
			func(input v1.Input, expectedErrors int, expectedErrorTypes []string, expectMessage string) {
				vmi := v1.NewMinimalVMI("testvmi")
				vmi.Spec.Domain.Devices.Inputs = append(vmi.Spec.Domain.Devices.Inputs, input)
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(len(causes)).To(Equal(expectedErrors), fmt.Sprintf("Expect %d errors", expectedErrors))
				for i, errorType := range expectedErrorTypes {
					Expect(causes[i].Field).To(Equal(errorType), expectMessage)
				}
			},
			table.Entry("and accept input with virtio bus",
				v1.Input{
					Type: "tablet",
					Name: "tablet0",
					Bus:  "virtio",
				}, 0, []string{}, "Expect no errors"),
			table.Entry("and accept input with usb bus",
				v1.Input{
					Type: "tablet",
					Name: "tablet0",
					Bus:  "usb",
				}, 0, []string{}, "Expect no errors"),
			table.Entry("and accept input without bus",
				v1.Input{
					Type: "tablet",
					Name: "tablet0",
				}, 0, []string{}, "Expect no errors"),
			table.Entry("and reject input with ps2 bus",
				v1.Input{
					Type: "tablet",
					Name: "tablet0",
					Bus:  "ps2",
				}, 1, []string{"fake.domain.devices.inputs[0].bus"}, "Expect bus error"),
			table.Entry("and reject input with keyboard type and virtio bus",
				v1.Input{
					Type: "keyboard",
					Name: "tablet0",
					Bus:  "virtio",
				}, 1, []string{"fake.domain.devices.inputs[0].type"}, "Expect type error"),
			table.Entry("and reject input with keyboard type and usb bus",
				v1.Input{
					Type: "keyboard",
					Name: "tablet0",
					Bus:  "usb",
				}, 1, []string{"fake.domain.devices.inputs[0].type"}, "Expect type error"),
			table.Entry("and reject input with wrong type and wrong bus",
				v1.Input{
					Type: "keyboard",
					Name: "tablet0",
					Bus:  "ps2",
				}, 2, []string{"fake.domain.devices.inputs[0].bus", "fake.domain.devices.inputs[0].type"}, "Expect type error"),
		)

		It("should reject negative requests.cpu value", func() {
			vm := v1.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("-200m"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.cpu"))
		})
		It("should reject negative limits.cpu size value", func() {
			vm := v1.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("-3"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.limits.cpu"))
		})
		It("should reject greater requests.cpu than limits.cpu", func() {
			vm := v1.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("2500m"),
			}
			vm.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("500m"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.cpu"))
		})
		It("should accept correct cpu size values", func() {
			vm := v1.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("1500m"),
			}
			vm.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("2"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should reject negative requests.memory size value", func() {
			vm := v1.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("-64Mi"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})
		It("should reject small requests.memory size value", func() {
			vm := v1.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64m"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})
		It("should reject negative limits.memory size value", func() {
			vm := v1.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("-65Mi"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject incorrect hugepages size format", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{}}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "2ab"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})
		It("should allow setting guest memory when no limit is set", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			guestMemory := resource.MustParse("100Mi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})
		It("should reject not divisable by hugepages.size requests.memory", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("65Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{}}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "2Gi"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject incorrect memory and hugepages size values", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{}}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "10Mi"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
			Expect(causes[0].Message).To(Equal("fake.domain.resources.requests.memory '64Mi' " +
				"is not a multiple of the page size fake.domain.hugepages.size '10Mi'"))
		})
		It("should reject setting guest memory and hugepages", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			guestMemory := resource.MustParse("64Mi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}
			vmi.Spec.Domain.Memory = &v1.Memory{
				Hugepages: &v1.Hugepages{},
				Guest:     &guestMemory,
			}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "2Mi"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
			Expect(causes[0].Message).To(ContainSubstring("'fake.domain.memory.guest' and " +
				"'fake.domain.memory.hugepages.size' must not be set at the same time"))
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

				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should accept interface and network lists equal to max element length", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			for i := 1; i < arrayLenMax; i++ {
				networkName := fmt.Sprintf("default%d", i)

				vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces,
					v1.Interface{Name: networkName,
						InterfaceBindingMethod: v1.InterfaceBindingMethod{
							Bridge: &v1.InterfaceBridge{}}})

				vmi.Spec.Networks = append(vmi.Spec.Networks,
					v1.Network{Name: networkName, NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: networkName}}})
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject interface lists greater than max element length", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			for i := 0; i < arrayLenMax; i++ {
				networkName := fmt.Sprintf("default%d", i)
				vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces,
					v1.Interface{Name: networkName,
						InterfaceBindingMethod: v1.InterfaceBindingMethod{
							Bridge: &v1.InterfaceBridge{}}})
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Message).To(Equal(fmt.Sprintf("fake.domain.devices.interfaces "+
				"list exceeds the %d element limit in length", arrayLenMax)))
		})
		It("should reject network lists greater than max element length", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			for i := 0; i < arrayLenMax; i++ {
				networkName := fmt.Sprintf("default%d", i)
				vmi.Spec.Networks = append(vmi.Spec.Networks,
					v1.Network{Name: networkName, NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: networkName}}})
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Message).To(Equal(fmt.Sprintf("fake.networks "+
				"list exceeds the %d element limit in length", arrayLenMax)))
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			// if this is processed correctly, it should result in a single error
			// If multiple causes occurred, then the spec was processed too far.
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.volumes"))
		})
		It("should reject disks with the same boot order", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			order := uint(1)
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, []v1.Disk{
				{Name: "testvolume1", BootOrder: &order, DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{}}},
				{Name: "testvolume2", BootOrder: &order, DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{}}}}...)

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, []v1.Volume{
				{Name: "testvolume1", VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{}}},
				{Name: "testvolume2", VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{}}}}...)

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[1].bootOrder"))
			Expect(causes[0].Message).To(Equal("Boot order for " +
				"fake.domain.devices.disks[1].bootOrder already set for a different device."))
		})
		It("should reject interface lists with more than one interface with the same name", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultNetworkInterface(),
				*v1.DefaultNetworkInterface()}
			vm.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				// if this is processed correctly, it should not result in any error
				Expect(len(causes)).To(Equal(0))
			}
		})

		It("should reject invalid interface model", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces[0].Model = "invalid_model"
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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
						Multus: &v1.MultusNetwork{NetworkName: "test-conf"},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should accept networks with a multus network source and bridge interface", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "default"},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(causes).To(BeEmpty())
		})
		It("should accept networks with a genie network source and bridge interface", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Genie: &v1.GenieNetwork{NetworkName: "default"},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(causes).To(BeEmpty())
		})
		It("should reject when multiple types defined for a CNI network", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "default1"},
						Genie:  &v1.GenieNetwork{NetworkName: "default2"},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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
						Multus: &v1.MultusNetwork{NetworkName: "multus-net1"},
					},
				},
				v1.Network{
					Name: "multus2",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "multus-net2"},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(causes).To(BeEmpty())
		})
		It("should allow single multus network with a multus default", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultNetworkInterface(),
			}
			vm.Spec.Domain.Devices.Interfaces[0].Name = "multus1"
			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name: "multus1",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "multus-net1", Default: true},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(causes).To(BeEmpty())
		})
		It("should reject multiple multus networks with a multus default", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultNetworkInterface(),
				*v1.DefaultNetworkInterface(),
			}
			vm.Spec.Domain.Devices.Interfaces[0].Name = "multus1"
			vm.Spec.Domain.Devices.Interfaces[1].Name = "multus2"
			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name: "multus1",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "multus-net1", Default: true},
					},
				},
				v1.Network{
					Name: "multus2",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "multus-net2", Default: true},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake.networks"))
			Expect(causes[0].Message).To(Equal("Multus CNI should only have one default network"))
		})
		It("should reject pod network with a multus default", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultNetworkInterface(),
				*v1.DefaultNetworkInterface(),
			}
			vm.Spec.Domain.Devices.Interfaces[1].Name = "multus1"
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
						Multus: &v1.MultusNetwork{NetworkName: "multus-net1", Default: true},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake.networks"))
			Expect(causes[0].Message).To(Equal("Pod network cannot be defined when Multus default network is defined"))
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
						Multus: &v1.MultusNetwork{NetworkName: "default1"},
					},
				},
				v1.Network{
					Name: "genie",
					NetworkSource: v1.NetworkSource{
						Genie: &v1.GenieNetwork{NetworkName: "default2"},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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
						Genie: &v1.GenieNetwork{NetworkName: "genie-net"},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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
						Multus: &v1.MultusNetwork{},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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
						Multus: &v1.MultusNetwork{NetworkName: "default"},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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
					NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "test"}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
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

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.interfaces"))
		})

		It("should accept valid MAC address", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			for _, macAddress := range []string{"de:ad:00:00:be:af", "de-ad-00-00-be-af"} {
				vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = macAddress // missing octet
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should reject non-IPv4 NTP servers", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces[0].DHCPOptions = &v1.DHCPOptions{
				NTPServers: []string{"::1", "hostname"},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(2))
		})

		It("should accept valid DHCPPrivateOptions", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces[0].DHCPOptions = &v1.DHCPOptions{
				PrivateOptions: []v1.DHCPPrivateOptions{v1.DHCPPrivateOptions{Option: 240, Value: "extra.options.kubevirt.io"}},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should reject invalid DHCPPrivateOptions", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces[0].DHCPOptions = &v1.DHCPOptions{
				PrivateOptions: []v1.DHCPPrivateOptions{v1.DHCPPrivateOptions{Option: 223, Value: "extra.options.kubevirt.io"}},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
		})

		It("should reject duplicate DHCPPrivateOptions", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces[0].DHCPOptions = &v1.DHCPOptions{
				PrivateOptions: []v1.DHCPPrivateOptions{
					v1.DHCPPrivateOptions{Option: 240, Value: "extra.options.kubevirt.io"},
					v1.DHCPPrivateOptions{Option: 240, Value: "sameextra.options.kubevirt.io"}},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
		})

		It("should accept unique DHCPPrivateOptions", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces[0].DHCPOptions = &v1.DHCPOptions{
				PrivateOptions: []v1.DHCPPrivateOptions{
					v1.DHCPPrivateOptions{Option: 240, Value: "extra.options.kubevirt.io"},
					v1.DHCPPrivateOptions{Option: 241, Value: "sameextra.options.kubevirt.io"}},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should return error if not unique DHCPPrivateOptions", func() {
			testDHCPPrivateOptions := []v1.DHCPPrivateOptions{
				v1.DHCPPrivateOptions{Option: 240, Value: "extra.options.kubevirt.io"},
				v1.DHCPPrivateOptions{Option: 240, Value: "sameextra.options.kubevirt.io"},
			}
			err := ValidateDuplicateDHCPPrivateOptions(testDHCPPrivateOptions)
			Expect(err).To(Equal(fmt.Errorf("You have provided duplicate DHCPPrivateOptions")))
		})

		It("should not return error if unique DHCPPrivateOptions", func() {
			testDHCPPrivateOptions := []v1.DHCPPrivateOptions{
				v1.DHCPPrivateOptions{Option: 240, Value: "extra.options.kubevirt.io"},
				v1.DHCPPrivateOptions{Option: 241, Value: "sameextra.options.kubevirt.io"},
			}
			err := ValidateDuplicateDHCPPrivateOptions(testDHCPPrivateOptions)
			Expect(err).To(BeNil())
		})

		It("should reject vmi with a network multiqueue, without virtio nics", func() {
			_true := true
			vmi := v1.NewMinimalVMI("testvm")
			nic := *v1.DefaultNetworkInterface()
			nic.Model = "e1000"
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{nic}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue = &_true
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(2))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.networkInterfaceMultiqueue"))
		})

		It("should reject nic multi queue without CPU settings", func() {
			_true := true
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue = &_true

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.networkInterfaceMultiqueue"))
		})

		It("should reject BlockMultiQueue without CPU settings", func() {
			_true := true
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.BlockMultiQueue = &_true

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.blockMultiQueue"))
		})

		It("should allow BlockMultiQueue with CPU settings", func() {
			_true := true
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.BlockMultiQueue = &_true
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{}
			vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU] = resource.MustParse("5")

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should ignore CPU settings for explicitly rejected BlockMultiQueue", func() {
			_false := false
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.BlockMultiQueue = &_false

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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
						Multus: &v1.MultusNetwork{NetworkName: "sriov"},
					},
				},
			)

			disableFeatureGates()

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[1].name"))
		})

		It("should allow valid ioThreadsPolicy", func() {
			vmi := v1.NewMinimalVMI("testvm")
			var ioThreadPolicy v1.IOThreadsPolicy
			ioThreadPolicy = "auto"
			vmi.Spec.Domain.IOThreadsPolicy = &ioThreadPolicy
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should reject invalid ioThreadsPolicy", func() {
			vmi := v1.NewMinimalVMI("testvm")
			var ioThreadPolicy v1.IOThreadsPolicy
			ioThreadPolicy = "bad"
			vmi.Spec.Domain.IOThreadsPolicy = &ioThreadPolicy
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Message).To(Equal(fmt.Sprintf("Invalid IOThreadsPolicy (%s)", ioThreadPolicy)))
		})

		table.DescribeTable("Should accept valid DNSPolicy and DNSConfig",
			func(dnsPolicy k8sv1.DNSPolicy, dnsConfig *k8sv1.PodDNSConfig) {
				vmi := v1.NewMinimalVMI("testvmi")
				vmi.Spec.DNSPolicy = dnsPolicy
				vmi.Spec.DNSConfig = dnsConfig
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(len(causes)).To(Equal(0))
			},
			table.Entry("with DNSPolicy ClusterFirstWithHostNet", k8sv1.DNSClusterFirstWithHostNet, &k8sv1.PodDNSConfig{}),
			table.Entry("with DNSPolicy ClusterFirst", k8sv1.DNSClusterFirst, &k8sv1.PodDNSConfig{}),
			table.Entry("with DNSPolicy Default", k8sv1.DNSDefault, &k8sv1.PodDNSConfig{}),
			table.Entry("with DNSPolicy None and one nameserver", k8sv1.DNSNone, &k8sv1.PodDNSConfig{Nameservers: []string{"1.2.3.4"}}),
			table.Entry("with DNSPolicy None max nameservers and max search domains", k8sv1.DNSNone, &k8sv1.PodDNSConfig{
				Nameservers: []string{"1.2.3.4", "5.6.7.8", "9.8.0.1"},
				Searches:    []string{"1", "2", "3", "4", "5", "6"},
			}),
			table.Entry("with DNSPolicy None max nameservers and max length search domain", k8sv1.DNSNone, &k8sv1.PodDNSConfig{
				Nameservers: []string{"1.2.3.4", "5.6.7.8", "9.8.0.1"},
				Searches:    []string{strings.Repeat("a", maxDNSSearchListChars/2), strings.Repeat("b", (maxDNSSearchListChars/2)-1)},
			}),
			table.Entry("with empty DNSPolicy", nil, nil),
		)

		table.DescribeTable("Should reject invalid DNSPolicy and DNSConfig",
			func(dnsPolicy k8sv1.DNSPolicy, dnsConfig *k8sv1.PodDNSConfig, causeCount int, causeMessage []string) {
				vmi := v1.NewMinimalVMI("testvmi")
				vmi.Spec.DNSPolicy = dnsPolicy
				vmi.Spec.DNSConfig = dnsConfig
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(len(causes)).To(Equal(causeCount))
				for i := 0; i < causeCount; i++ {
					Expect(causes[i].Message).To(Equal(causeMessage[i]))
				}
			},
			table.Entry("with invalid DNSPolicy FakePolicy", k8sv1.DNSPolicy("FakePolicy"), &k8sv1.PodDNSConfig{}, 1,
				[]string{"DNSPolicy: FakePolicy is not supported, valid values: [ClusterFirstWithHostNet ClusterFirst Default None ]"}),
			table.Entry("with DNSPolicy None and no nameserver", k8sv1.DNSNone, &k8sv1.PodDNSConfig{}, 1,
				[]string{"must provide at least one DNS nameserver when `dnsPolicy` is None"}),
			table.Entry("with DNSPolicy None and too many nameservers", k8sv1.DNSNone, &k8sv1.PodDNSConfig{
				Nameservers: []string{"1.2.3.4", "5.6.7.8", "9.8.0.1", "2.3.4.5"},
			}, 1, []string{"must not have more than 3 nameservers: [1.2.3.4 5.6.7.8 9.8.0.1 2.3.4.5]"}),
			table.Entry("with DNSPolicy None and a non ip nameserver", k8sv1.DNSNone, &k8sv1.PodDNSConfig{
				Nameservers: []string{"1.2.3.c"},
			}, 1, []string{"must be valid IP address: 1.2.3.c"}),
			table.Entry("with DNSPolicy None and too many search domains", k8sv1.DNSNone, &k8sv1.PodDNSConfig{
				Nameservers: []string{"1.2.3.4"},
				Searches:    []string{"1", "2", "3", "4", "5", "6", "7"},
			}, 1, []string{"must not have more than 6 search paths"}),
			table.Entry("with DNSPolicy None and seach domain exceeding max length", k8sv1.DNSNone, &k8sv1.PodDNSConfig{
				Nameservers: []string{"1.2.3.4"},
				Searches:    []string{strings.Repeat("a", maxDNSSearchListChars/2), strings.Repeat("b", (maxDNSSearchListChars / 2))},
			}, 1, []string{fmt.Sprintf("must not have more than %v characters (including spaces) in the search list", maxDNSSearchListChars)}),
			table.Entry("with DNSPolicy None and bad IsDNS1123Subdomain", k8sv1.DNSNone, &k8sv1.PodDNSConfig{
				Nameservers: []string{"1.2.3.4"},
				Searches:    []string{strings.Repeat("a", validation.DNS1123SubdomainMaxLength+1)},
			}, 1, []string{fmt.Sprintf("must be no more than %v characters", validation.DNS1123SubdomainMaxLength)}),
			table.Entry("with DNSPolicy None and bad options", k8sv1.DNSNone, &k8sv1.PodDNSConfig{
				Nameservers: []string{"1.2.3.4"},
				Options: []k8sv1.PodDNSConfigOption{
					{Value: &dnsConfigTestOption},
				},
			}, 1, []string{fmt.Sprintf("Option.Name must not be empty for value: %s", dnsConfigTestOption)}),
			table.Entry("with DNSPolicy None and nil DNSConfig", k8sv1.DNSNone, interface{}(nil), 1,
				[]string{fmt.Sprintf("must provide `dnsConfig` when `dnsPolicy` is %s", k8sv1.DNSNone)}),
		)
	})
	Context("with cpu pinning", func() {
		var vmi *v1.VirtualMachineInstance
		BeforeEach(func() {
			vmi = v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.CPU = &v1.CPU{DedicatedCPUPlacement: true}
		})
		It("should reject specs without cpu reqirements", func() {
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.cpu.dedicatedCpuPlacement"))
		})
		It("should reject specs without inconsistent cpu reqirements", func() {
			vmi.Spec.Domain.CPU.Cores = 4
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("2"),
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.cpu.dedicatedCpuPlacement"))
		})
		It("should reject specs with non-integer cpu limits values", func() {
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("800m"),
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.limits.cpu"))
		})
		It("should reject specs with non-integer cpu requests values", func() {
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceCPU:    resource.MustParse("800m"),
				k8sv1.ResourceMemory: resource.MustParse("8Mi"),
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
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
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})
	})

	Context("with CPU features", func() {
		It("should accept valid CPU feature policies", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.CPU = &v1.CPU{
				Features: []v1.CPUFeature{
					{
						Name: "lahf_lm",
					},
				},
			}

			for _, policy := range validCPUFeaturePolicies {
				vmi.Spec.Domain.CPU.Features[0].Policy = policy
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(len(causes)).To(Equal(0))
			}
		})

		It("should reject invalid CPU feature policy", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.CPU = &v1.CPU{
				Features: []v1.CPUFeature{
					{
						Name:   "lahf_lm",
						Policy: "invalid_policy",
					},
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
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
			table.Entry("with CDRom target",
				v1.Disk{Name: "testdisk", DiskDevice: v1.DiskDevice{CDRom: &v1.CDRomTarget{}}},
			),
		)

		It("should reject floppy disks", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "floppydisk",
				DiskDevice: v1.DiskDevice{
					Floppy: &v1.FloppyTarget{},
				},
			})
			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].name"))
		})

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
					Disk:  &v1.DiskTarget{},
					CDRom: &v1.CDRomTarget{},
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
				Name: "testdisk1",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: "virtio",
					},
				},
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk2",
				DiskDevice: v1.DiskDevice{
					LUN: &v1.LunTarget{
						Bus: "sata",
					},
				},
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk3",
				DiskDevice: v1.DiskDevice{
					CDRom: &v1.CDRomTarget{
						Bus: "scsi",
					},
				},
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk4",
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
				Name: "testdisk1",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: "ide",
					},
				},
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk2",
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

		It("should reject disk with invalid cache mode", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk", Cache: "unspported", DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{}}})

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake[0].cache"))
			Expect(causes[0].Message).To(Equal("fake[0].cache has invalid value unspported"))
		})

		It("should reject disk count > arrayLenMax", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			for i := 0; i <= arrayLenMax; i++ {
				name := strconv.Itoa(i)
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "testdisk" + name, DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{}}})
			}

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake"))
			Expect(causes[0].Message).To(Equal(fmt.Sprintf("fake list exceeds the %d "+
				"element limit in length", arrayLenMax)))
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

	Context("with bootloader", func() {
		It("should accept empty bootloader setting", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Subdomain = "testsubdomain"

			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: nil,
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should accept BIOS", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Subdomain = "testsubdomain"

			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					BIOS: &v1.BIOS{},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should accept EFI", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Subdomain = "testsubdomain"

			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					EFI: &v1.EFI{},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should not accept BIOS and EFI together", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Subdomain = "testsubdomain"

			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					EFI:  &v1.EFI{},
					BIOS: &v1.BIOS{},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
		})
	})
})

var _ = Describe("Function getNumberOfPodInterfaces()", func() {
	config, _ := testutils.NewFakeClusterConfig(&k8sv1.ConfigMap{})

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
		causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), spec, config)
		Expect(causes).To(HaveLen(1))
	})
	It("should reject when more than one network source is configured", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		net1 := v1.Network{
			NetworkSource: v1.NetworkSource{
				Pod:    &v1.PodNetwork{},
				Multus: &v1.MultusNetwork{NetworkName: "testnet1"},
			},
		}
		iface1 := v1.Interface{Name: net1.Name}
		spec.Networks = []v1.Network{net1}
		spec.Domain.Devices.Interfaces = []v1.Interface{iface1}
		causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), spec, config)
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
		causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), spec, config)
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
		causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), spec, config)
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
		causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), spec, config)
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

		causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), spec, config)
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Field).To(ContainSubstring("bootOrder"))
	})
	It("should reject a serial number whose length is greater than 256", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		sn := strings.Repeat("1", maxStrLen+1)

		spec.Domain.Firmware = &v1.Firmware{Serial: sn}

		causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), spec, config)
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Field).To(ContainSubstring("serial"))
	})
	It("should reject a serial number with invalid characters", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		sn := "$$$$"

		spec.Domain.Firmware = &v1.Firmware{Serial: sn}

		causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), spec, config)
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Field).To(ContainSubstring("serial"))
	})
	It("should accept a valid serial number", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		sn := "6a1a24a1-4061-4607-8bf4-a3963d0c5895"

		spec.Domain.Firmware = &v1.Firmware{Serial: sn}

		causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), spec, config)
		Expect(len(causes)).To(Equal(0))
	})
})
