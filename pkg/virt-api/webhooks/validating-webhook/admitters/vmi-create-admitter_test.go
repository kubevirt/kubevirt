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

	"kubevirt.io/client-go/api"
	"kubevirt.io/kubevirt/tools/vms-generator/utils"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/rbac"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authentication/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	nodelabellerutil "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
)

var _ = Describe("Validating VMICreate Admitter", func() {
	kv := &v1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt",
			Namespace: "kubevirt",
		},
		Spec: v1.KubeVirtSpec{
			Configuration: v1.KubeVirtConfiguration{
				DeveloperConfiguration: &v1.DeveloperConfiguration{},
			},
		},
		Status: v1.KubeVirtStatus{
			Phase: v1.KubeVirtPhaseDeploying,
		},
	}
	config, _, kvInformer := testutils.NewFakeClusterConfigUsingKV(kv)
	vmiCreateAdmitter := &VMICreateAdmitter{ClusterConfig: config}

	dnsConfigTestOption := "test"
	enableFeatureGate := func(featureGate string) {
		kvConfig := kv.DeepCopy()
		kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{featureGate}
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)
	}
	disableFeatureGates := func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kv)
	}
	enableSlirpInterface := func() {
		kvConfig := kv.DeepCopy()
		kvConfig.Spec.Configuration.NetworkConfiguration = &v1.NetworkConfiguration{
			PermitSlirpInterface: pointer.BoolPtr(true),
		}
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)
	}
	disableBridgeOnPodNetwork := func() {
		kvConfig := kv.DeepCopy()
		kvConfig.Spec.Configuration.NetworkConfiguration = &v1.NetworkConfiguration{
			PermitBridgeInterfaceOnPodNetwork: pointer.BoolPtr(false),
		}

		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)
	}

	AfterEach(func() {
		disableFeatureGates()
	})

	It("should reject invalid VirtualMachineInstance spec on create", func() {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "testdisk",
		})
		vmiBytes, _ := json.Marshal(&vmi)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: vmiBytes,
				},
			},
		}

		resp := vmiCreateAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(len(resp.Result.Details.Causes)).To(Equal(1))
		Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.domain.devices.disks[0].name"))
	})
	It("should reject VMIs without memory after presets were applied", func() {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Resources = v1.ResourceRequirements{}
		vmiBytes, _ := json.Marshal(&vmi)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: vmiBytes,
				},
			},
		}
		resp := vmiCreateAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(len(resp.Result.Details.Causes)).To(Equal(1))
		Expect(resp.Result.Message).To(ContainSubstring("no memory requested"))
	})

	Context("tolerations with eviction policies given", func() {
		var vmi *v1.VirtualMachineInstance
		var policy = v1.EvictionStrategyLiveMigrate
		BeforeEach(func() {
			enableFeatureGate(virtconfig.LiveMigrationGate)
			vmi = api.NewMinimalVMI("testvmi")
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
		It("should reject probes with no probe action configured", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.ReadinessProbe = &v1.Probe{InitialDelaySeconds: 2}
			vmi.Spec.LivenessProbe = &v1.Probe{InitialDelaySeconds: 2}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

			vmiBytes, _ := json.Marshal(&vmi)

			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: vmiBytes,
					},
				},
			}
			resp := vmiCreateAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(Equal(`either spec.readinessProbe.tcpSocket, spec.readinessProbe.exec or spec.readinessProbe.httpGet must be set if a spec.readinessProbe is specified, either spec.livenessProbe.tcpSocket, spec.livenessProbe.exec or spec.livenessProbe.httpGet must be set if a spec.livenessProbe is specified`))
		})
		It("should reject probes with more than one action per probe configured", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.ReadinessProbe = &v1.Probe{
				InitialDelaySeconds: 2,
				Handler: v1.Handler{
					HTTPGet:        &k8sv1.HTTPGetAction{Host: "test", Port: intstr.Parse("80")},
					TCPSocket:      &k8sv1.TCPSocketAction{Host: "lal", Port: intstr.Parse("80")},
					Exec:           &k8sv1.ExecAction{Command: []string{"uname", "-a"}},
					GuestAgentPing: &v1.GuestAgentPing{},
				},
			}
			vmi.Spec.LivenessProbe = &v1.Probe{
				InitialDelaySeconds: 2,
				Handler: v1.Handler{
					HTTPGet:   &k8sv1.HTTPGetAction{Host: "test", Port: intstr.Parse("80")},
					TCPSocket: &k8sv1.TCPSocketAction{Host: "lal", Port: intstr.Parse("80")},
					Exec:      &k8sv1.ExecAction{Command: []string{"uname", "-a"}},
				},
			}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

			vmiBytes, _ := json.Marshal(&vmi)

			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: vmiBytes,
					},
				},
			}
			resp := vmiCreateAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(Equal(`spec.readinessProbe must have exactly one probe type set, spec.livenessProbe must have exactly one probe type set`))
		})
		It("should accept properly configured readiness and liveness probes", func() {
			vmi := api.NewMinimalVMI("testvmi")
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
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

			vmiBytes, _ := json.Marshal(&vmi)

			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: vmiBytes,
					},
				},
			}
			resp := vmiCreateAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeTrue())
		})
		It("should reject properly configured network-based readiness and liveness probes if no Pod Network is present", func() {
			vmi := api.NewMinimalVMI("testvmi")
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

			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: vmiBytes,
					},
				},
			}
			resp := vmiCreateAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(Equal(`spec.readinessProbe.tcpSocket is only allowed if the Pod Network is attached, spec.livenessProbe.httpGet is only allowed if the Pod Network is attached`))
		})
	})

	It("should accept valid vmi spec on create", func() {
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
		vmiBytes, _ := json.Marshal(&vmi)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: vmiBytes,
				},
			},
		}
		resp := vmiCreateAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeTrue())
	})

	It("should allow unknown fields in the status to allow updates", func() {
		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
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

	table.DescribeTable("should reject documents containing unknown or missing fields for", func(data string, validationResult string, gvr metav1.GroupVersionResource, review func(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse) {
		input := map[string]interface{}{}
		json.Unmarshal([]byte(data), &input)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
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

	Context("with VirtualMachineInstance metadata", func() {
		table.DescribeTable(
			"Should allow VMI creation with kubevirt.io/ labels only for kubevirt service accounts",
			func(vmiLabels map[string]string, userAccount string, positive bool) {
				vmi := api.NewMinimalVMI("testvmi")
				vmi.Labels = vmiLabels
				vmiBytes, _ := json.Marshal(&vmi)
				ar := &admissionv1.AdmissionReview{
					Request: &admissionv1.AdmissionRequest{
						Operation: admissionv1.Create,
						UserInfo:  authv1.UserInfo{Username: "system:serviceaccount:kubevirt:" + userAccount},
						Resource:  webhooks.VirtualMachineInstanceGroupVersionResource,
						Object: runtime.RawExtension{
							Raw: vmiBytes,
						},
					},
				}
				resp := vmiCreateAdmitter.Admit(ar)
				if positive {
					Expect(resp.Allowed).To(BeTrue())
				} else {
					Expect(resp.Allowed).To(BeFalse())
					Expect(len(resp.Result.Details.Causes)).To(Equal(1))
					Expect(resp.Result.Details.Causes[0].Message).To(Equal("creation of the following reserved kubevirt.io/ labels on a VMI object is prohibited"))
				}
			},
			table.Entry("Create restricted label by API",
				map[string]string{v1.NodeNameLabel: "someValue"},
				rbac.ApiServiceAccountName,
				true,
			),
			table.Entry("Create restricted label by Handler",
				map[string]string{v1.NodeNameLabel: "someValue"},
				rbac.HandlerServiceAccountName,
				true,
			),
			table.Entry("Create restricted label by Controller",
				map[string]string{v1.NodeNameLabel: "someValue"},
				rbac.ControllerServiceAccountName,
				true,
			),
			table.Entry("Create restricted label by non kubevirt user",
				map[string]string{v1.NodeNameLabel: "someValue"},
				"user-account",
				false,
			),
			table.Entry("Create non restricted kubevirt.io prefixed label by non kubevirt user",
				map[string]string{"kubevirt.io/l": "someValue"},
				"user-account",
				true,
			),
		)
		table.DescribeTable("should reject annotations which require feature gate enabled", func(annotations map[string]string, expectedMsg string) {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.ObjectMeta = metav1.ObjectMeta{
				Annotations: annotations,
			}

			causes := ValidateVirtualMachineInstanceMetadata(k8sfield.NewPath("metadata"), &vmi.ObjectMeta, config, "fake-account")
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(causes[0].Message).To(ContainSubstring(expectedMsg))
		},
			table.Entry("without ExperimentalIgnitionSupport feature gate enabled",
				map[string]string{v1.IgnitionAnnotation: "fake-data"},
				fmt.Sprintf("invalid entry metadata.annotations.%s", v1.IgnitionAnnotation),
			),
			table.Entry("without sidecar feature gate enabled",
				map[string]string{hooks.HookSidecarListAnnotationName: "[{'image': 'fake-image'}]"},
				fmt.Sprintf("invalid entry metadata.annotations.%s", hooks.HookSidecarListAnnotationName),
			),
		)

		table.DescribeTable("should accept annotations which require feature gate enabled", func(annotations map[string]string, featureGate string) {
			enableFeatureGate(featureGate)
			vmi := api.NewMinimalVMI("testvmi")
			vmi.ObjectMeta = metav1.ObjectMeta{
				Annotations: annotations,
			}
			causes := ValidateVirtualMachineInstanceMetadata(k8sfield.NewPath("metadata"), &vmi.ObjectMeta, config, "fake-account")
			Expect(len(causes)).To(Equal(0))
		},
			table.Entry("with ExperimentalIgnitionSupport feature gate enabled",
				map[string]string{v1.IgnitionAnnotation: "fake-data"},
				virtconfig.IgnitionGate,
			),
			table.Entry("with sidecar feature gate enabled",
				map[string]string{hooks.HookSidecarListAnnotationName: "[{'image': 'fake-image'}]"},
				virtconfig.SidecarGate,
			),
		)
	})

	Context("with VirtualMachineInstance spec", func() {
		It("should accept valid machine type", func() {
			vmi := api.NewMinimalVMI("testvmi")
			if webhooks.IsPPC64() {
				vmi.Spec.Domain.Machine = &v1.Machine{Type: "pseries"}
			} else if webhooks.IsARM64() {
				vmi.Spec.Domain.Machine = &v1.Machine{Type: "virt"}
			} else {
				vmi.Spec.Domain.Machine = &v1.Machine{Type: "q35"}
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject invalid machine type", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Machine = &v1.Machine{Type: "test"}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake.domain.machine.type"))
			Expect(causes[0].Message).To(ContainSubstring("fake.domain.machine.type is not supported: test (allowed values:"))
		})

		It("should accept valid hostname", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Hostname = "test"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject invalid hostname", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Hostname = "test+bad"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake.hostname"))
			Expect(causes[0].Message).To(ContainSubstring("does not conform to the kubernetes DNS_LABEL rules : "))
		})
		It("should accept valid subdomain name", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Subdomain = "testsubdomain"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject invalid subdomain name", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Subdomain = "bad+domain"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.subdomain"))
		})
		It("should accept disk and volume lists equal to max element length", func() {
			vmi := api.NewMinimalVMI("testvmi")

			for i := 0; i < arrayLenMax; i++ {
				diskName := fmt.Sprintf("testdisk%d", i)
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: diskName,
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: diskName,
					VolumeSource: v1.VolumeSource{
						ContainerDisk: testutils.NewFakeContainerDiskSource(),
					},
				})
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject disk lists greater than max element length", func() {
			vmi := api.NewMinimalVMI("testvmi")

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
			vmi := api.NewMinimalVMI("testvmi")

			for i := 0; i <= arrayLenMax; i++ {
				volumeName := "testVolume"
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: volumeName,
					VolumeSource: v1.VolumeSource{
						ContainerDisk: testutils.NewFakeContainerDiskSource(),
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
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[0].name"))
		})
		It("should allow supported audio devices", func() {
			supportedDevices := [...]string{"", "ich9", "ac97"}
			vmi := api.NewMinimalVMI("testvmi")

			for _, deviceName := range supportedDevices {
				vmi.Spec.Domain.Devices.Sound = &v1.SoundDevice{
					Name:  "audio-device",
					Model: deviceName,
				}
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(len(causes)).To(Equal(0))
			}
		})
		It("should reject unsupported audio devices", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Sound = &v1.SoundDevice{
				Name:  "audio-device",
				Model: "aNotSupportedDevice",
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.Sound"))
		})
		It("should reject audio devices without name fields", func() {
			vmi := api.NewMinimalVMI("testvmi")

			supportedAudioDevice := "ac97"
			vmi.Spec.Domain.Devices.Sound = &v1.SoundDevice{
				Model: supportedAudioDevice,
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.Sound"))
		})
		It("should reject volume with missing disk / file system", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					CloudInitNoCloud: &v1.CloudInitNoCloudSource{UserData: " "},
				},
			})

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.volumes[0].name"))
		})
		It("should reject multiple disks referencing same volume", func() {
			vmi := api.NewMinimalVMI("testvmi")

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
					ContainerDisk: testutils.NewFakeContainerDiskSource(),
				},
			})
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[1].name"))
		})
		It("should generate multiple causes", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				DiskDevice: v1.DiskDevice{
					Disk:  &v1.DiskTarget{},
					CDRom: &v1.CDRomTarget{Bus: "sata"},
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
				vmi := api.NewMinimalVMI("testvmi")
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
			vm := api.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("-200m"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.cpu"))
		})
		It("should reject negative limits.cpu size value", func() {
			vm := api.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("-3"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.limits.cpu"))
		})
		It("should reject greater requests.cpu than limits.cpu", func() {
			vm := api.NewMinimalVMI("testvm")

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
			vm := api.NewMinimalVMI("testvm")

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
			vm := api.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("-64Mi"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})
		It("should reject small requests.memory size value", func() {
			vm := api.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64m"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})
		It("should reject negative limits.memory size value", func() {
			vm := api.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("-65Mi"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.limits.memory"))
		})
		It("should reject greater requests.memory than limits.memory", func() {
			vm := api.NewMinimalVMI("testvm")

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
			vm := api.NewMinimalVMI("testvm")

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
			vmi := api.NewMinimalVMI("testvmi")

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
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{}}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "1Gi"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})
		It("should allow smaller guest memory than requested memory", func() {
			vmi := api.NewMinimalVMI("testvmi")
			guestMemory := resource.MustParse("1Mi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject bigger guest memory than the memory limit", func() {
			vmi := api.NewMinimalVMI("testvmi")
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
			vmi := api.NewMinimalVMI("testvmi")
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
			vmi := api.NewMinimalVMI("testvmi")
			guestMemory := resource.MustParse("100Mi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})
		It("should reject not divisable by hugepages.size requests.memory", func() {
			vmi := api.NewMinimalVMI("testvmi")

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
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{}}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "2Mi"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject incorrect memory and hugepages size values", func() {
			vmi := api.NewMinimalVMI("testvmi")

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
		It("should allow setting guest memory and hugepages", func() {
			vmi := api.NewMinimalVMI("testvmi")
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
			Expect(len(causes)).To(Equal(0))
		})
		table.DescribeTable("should verify LUN is mapped to PVC volume",
			func(volume *v1.Volume, expectedErrors int) {
				vmi := api.NewMinimalVMI("testvmi")
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
						ContainerDisk: testutils.NewFakeContainerDiskSource(),
					},
				}, 1),
			table.Entry("and accept PVC sources",
				&v1.Volume{
					Name: "testdisk",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{},
					},
				}, 0),
		)
		It("should accept a single interface and network", func() {
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vm.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should accept interface and network lists equal to max element length", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
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
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
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
			vmi := api.NewMinimalVMI("testvmi")
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
		It("should reject disks with the same boot order", func() {
			vmi := api.NewMinimalVMI("testvmi")
			order := uint(1)
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, []v1.Disk{
				{Name: "testvolume1", BootOrder: &order, DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{}}},
				{Name: "testvolume2", BootOrder: &order, DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{}}}}...)

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, []v1.Volume{
				{Name: "testvolume1", VolumeSource: v1.VolumeSource{
					ContainerDisk: testutils.NewFakeContainerDiskSource()}},
				{Name: "testvolume2", VolumeSource: v1.VolumeSource{
					ContainerDisk: testutils.NewFakeContainerDiskSource()}}}...)

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[1].bootOrder"))
			Expect(causes[0].Message).To(Equal("Boot order for " +
				"fake.domain.devices.disks[1].bootOrder already set for a different device."))
		})
		It("should reject interface lists with more than one interface with the same name", func() {
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultBridgeNetworkInterface(),
				*v1.DefaultBridgeNetworkInterface()}
			vm.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			// if this is processed correctly, it should result an error
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[1].name"))
		})
		It("should accept network lists with more than one element", func() {
			vm := api.NewMinimalVMI("testvm")
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
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

			for model := range validInterfaceModels {
				vmi.Spec.Domain.Devices.Interfaces[0].Model = model
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				// if this is processed correctly, it should not result in any error
				Expect(len(causes)).To(Equal(0))
			}
		})

		It("should reject invalid interface model", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces[0].Model = "invalid_model"
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
		})

		It("should reject interfaces with missing network", func() {
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vm.Spec.Networks = []v1.Network{
				{
					Name: "redtest",
					NetworkSource: v1.NetworkSource{
						Pod: &v1.PodNetwork{},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(2))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].name"))
			Expect(causes[1].Field).To(Equal("fake.networks[0].name"))
		})
		It("should reject networks with missing interface", func() {
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{}
			vm.Spec.Networks = []v1.Network{
				{
					Name: "redtest",
					NetworkSource: v1.NetworkSource{
						Pod: &v1.PodNetwork{},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.networks[0].name"))
		})
		It("should reject networks with duplicate names", func() {
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vm.Spec.Networks = []v1.Network{
				{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Pod: &v1.PodNetwork{},
					},
				},
				{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "test"},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.networks[1].name"))
			Expect(causes[0].Message).To(Equal("Network with name \"default\" already exists, every network must have a unique name"))
		})
		It("should reject interface named with unsupported characters", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
				{
					Name: "d.efault",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						Bridge: &v1.InterfaceBridge{},
					},
				},
			}
			vmi.Spec.Networks = []v1.Network{
				{
					Name: "d.efault",
					NetworkSource: v1.NetworkSource{
						Pod: &v1.PodNetwork{},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].name"))
			Expect(causes[0].Message).To(Equal("Network interface name can only contain alphabetical characters, numbers, dashes (-) or underscores (_)"))
		})
		It("should reject unassign multus network", func() {
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
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
			Expect(causes[0].Field).To(Equal("fake.networks[1].name"))
		})
		It("should accept networks with a pod network source and bridge interface", func() {
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vm.Spec.Networks = []v1.Network{
				{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should accept networks with a multus network source and bridge interface", func() {
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vm.Spec.Networks = []v1.Network{
				{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "default"},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(causes).To(BeEmpty())
		})
		It("should reject when multiple types defined for a CNI network", func() {
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vm.Spec.Networks = []v1.Network{
				{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "default1"},
						Pod:    &v1.PodNetwork{},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.networks[0]"))
		})
		It("should allow multiple networks of same CNI type", func() {
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultBridgeNetworkInterface(),
				*v1.DefaultBridgeNetworkInterface(),
				*v1.DefaultBridgeNetworkInterface(),
			}
			vm.Spec.Domain.Devices.Interfaces[0].Name = "multus1"
			vm.Spec.Domain.Devices.Interfaces[1].Name = "multus2"
			// 3rd interfaces uses the default pod network, name is "default"
			vm.Spec.Networks = []v1.Network{
				{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Pod: &v1.PodNetwork{},
					},
				},
				{
					Name: "multus1",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "multus-net1"},
					},
				},
				{
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
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultBridgeNetworkInterface(),
			}
			vm.Spec.Domain.Devices.Interfaces[0].Name = "multus1"
			vm.Spec.Networks = []v1.Network{
				{
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
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultBridgeNetworkInterface(),
				*v1.DefaultBridgeNetworkInterface(),
			}
			vm.Spec.Domain.Devices.Interfaces[0].Name = "multus1"
			vm.Spec.Domain.Devices.Interfaces[1].Name = "multus2"
			vm.Spec.Networks = []v1.Network{
				{
					Name: "multus1",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "multus-net1", Default: true},
					},
				},
				{
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
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultBridgeNetworkInterface(),
				*v1.DefaultBridgeNetworkInterface(),
			}
			vm.Spec.Domain.Devices.Interfaces[1].Name = "multus1"
			vm.Spec.Networks = []v1.Network{
				{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Pod: &v1.PodNetwork{},
					},
				},
				{
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
		It("should reject multus network source without networkName", func() {
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vm.Spec.Networks = []v1.Network{
				{
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
			enableSlirpInterface()
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				}}}
			vm.Spec.Networks = []v1.Network{
				{
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
			enableSlirpInterface()
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				}}}

			vm.Spec.Networks = []v1.Network{
				{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should accept networks with a pod network source and slirp interface with port", func() {
			enableSlirpInterface()
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Port: 80}}}}

			vm.Spec.Networks = []v1.Network{
				{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject networks with a pod network source and slirp interface without specific port", func() {
			enableSlirpInterface()
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Name: "test"}}}}

			vm.Spec.Networks = []v1.Network{
				{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].ports[0]"))
		})
		It("should reject a masquerade interface on a network different than pod", func() {
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Masquerade: &v1.InterfaceMasquerade{},
				},
				Ports: []v1.Port{{Name: "test"}}}}

			vm.Spec.Networks = []v1.Network{
				{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "test"}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].name"))
		})
		It("should reject a masquerade interface with a specified MAC address which is reserved by the BindMechanism", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Masquerade: &v1.InterfaceMasquerade{},
				},
				MacAddress: "02:00:00:00:00:00",
			}}

			vmi.Spec.Networks = []v1.Network{
				{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Message).To(Equal("The requested MAC address is reserved for the in-pod bridge. Please choose another one."))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].macAddress"))
		})
		It("should accept a bridge interface on a pod network when it is permitted", func() {
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vm.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject a bridge interface on a pod network when it is not permitted", func() {
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vm.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			disableBridgeOnPodNetwork()
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].name"))
		})
		It("should reject a bad port name", func() {
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Masquerade: &v1.InterfaceMasquerade{},
				},
				Ports: []v1.Port{{Name: "Test", Port: 80}}}}

			vm.Spec.Networks = []v1.Network{{
				Name:          "default",
				NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
			},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1), "unexpected number of errors")
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].ports[0].name"))
		})
		It("should reject networks with a pod network source and slirp interface with bad protocol type", func() {
			enableSlirpInterface()
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Protocol: "bad", Port: 80}}}}

			vm.Spec.Networks = []v1.Network{
				{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].ports[0].protocol"))
		})
		It("should accept networks with a pod network source and slirp interface with multiple Ports", func() {
			enableSlirpInterface()
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Port: 80}, {Protocol: "UDP", Port: 80}}}}

			vm.Spec.Networks = []v1.Network{
				{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject a macvtap interface on a network different than multus", func() {
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Macvtap: &v1.InterfaceMacvtap{},
				},
			}}

			vm.Spec.Networks = []v1.Network{
				{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			enableFeatureGate(virtconfig.MacvtapGate)
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].name"))
			Expect(causes[0].Message).To(Equal("Macvtap interface only implemented with Multus network"))
		})
		It("should reject a macvtap interface on a multus network when the feature is inactive", func() {
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Macvtap: &v1.InterfaceMacvtap{},
				},
			}}

			vm.Spec.Networks = []v1.Network{
				{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "test"}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].name"))
			Expect(causes[0].Message).To(Equal("Macvtap feature gate is not enabled"))
		})
		It("should accept a macvtap interface on a multus network when the feature is active", func() {
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Macvtap: &v1.InterfaceMacvtap{},
				},
			}}

			vm.Spec.Networks = []v1.Network{
				{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "test"}},
				},
			}

			enableFeatureGate(virtconfig.MacvtapGate)
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(causes).To(HaveLen(0))
		})
		It("should reject port out of range", func() {
			enableSlirpInterface()
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Port: 80000}}}}

			vm.Spec.Networks = []v1.Network{
				{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].ports[0]"))
		})
		It("should reject interface with two ports with the same name", func() {
			enableSlirpInterface()
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Name: "testport", Port: 80}, {Name: "testport", Protocol: "UDP", Port: 80}}}}

			vm.Spec.Networks = []v1.Network{
				{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].ports[1].name"))
		})
		It("should reject two interfaces with same port name", func() {
			enableSlirpInterface()
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Name: "testport", Port: 80}}},
				{
					Name: "default",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						Slirp: &v1.InterfaceSlirp{},
					},
					Ports: []v1.Port{{Name: "testport", Protocol: "UDP", Port: 80}}}}

			vm.Spec.Networks = []v1.Network{
				{
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
			enableSlirpInterface()
			vm := api.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Port: 80}, {Protocol: "UDP", Port: 80}, {Protocol: "TCP", Port: 80}}}}

			vm.Spec.Networks = []v1.Network{
				{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject specs with multiple pod interfaces", func() {
			vm := api.NewMinimalVMI("testvm")
			for i := 1; i < 3; i++ {
				iface := v1.DefaultBridgeNetworkInterface()
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
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			for _, macAddress := range []string{"de:ad:00:00:be:af", "de-ad-00-00-be-af"} {
				vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = macAddress // missing octet
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				// if this is processed correctly, it should not result in any error
				Expect(len(causes)).To(Equal(0))
			}
		})

		It("should reject invalid MAC addresses", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			for _, macAddress := range []string{"de:ad:00:00:be", "de-ad-00-00-be", "de:ad:00:00:be:af:be:af"} {
				vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = macAddress
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(len(causes)).To(Equal(1))
				Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].macAddress"))
			}
		})
		It("should accept valid PCI address", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			for _, pciAddress := range []string{"0000:81:11.1", "0001:02:00.0"} {
				vmi.Spec.Domain.Devices.Interfaces[0].PciAddress = pciAddress
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				// if this is processed correctly, it should not result in any error
				Expect(len(causes)).To(Equal(0))
			}
		})

		It("should reject invalid PCI addresses", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			for _, pciAddress := range []string{"0000:80.10.1", "0000:80:80:1.0", "0000:80:11.15"} {
				vmi.Spec.Domain.Devices.Interfaces[0].PciAddress = pciAddress
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(len(causes)).To(Equal(1))
				Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].pciAddress"))
			}
		})

		It("should accept valid NTP servers", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces[0].DHCPOptions = &v1.DHCPOptions{
				NTPServers: []string{"127.0.0.1", "127.0.0.2"},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should reject non-IPv4 NTP servers", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces[0].DHCPOptions = &v1.DHCPOptions{
				NTPServers: []string{"::1", "hostname"},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(2))
		})

		It("should accept valid DHCPPrivateOptions", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces[0].DHCPOptions = &v1.DHCPOptions{
				PrivateOptions: []v1.DHCPPrivateOptions{{Option: 240, Value: "extra.options.kubevirt.io"}},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should reject invalid DHCPPrivateOptions", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces[0].DHCPOptions = &v1.DHCPOptions{
				PrivateOptions: []v1.DHCPPrivateOptions{{Option: 223, Value: "extra.options.kubevirt.io"}},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
		})

		It("should reject duplicate DHCPPrivateOptions", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces[0].DHCPOptions = &v1.DHCPOptions{
				PrivateOptions: []v1.DHCPPrivateOptions{
					{Option: 240, Value: "extra.options.kubevirt.io"},
					{Option: 240, Value: "sameextra.options.kubevirt.io"}},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
		})

		It("should accept unique DHCPPrivateOptions", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces[0].DHCPOptions = &v1.DHCPOptions{
				PrivateOptions: []v1.DHCPPrivateOptions{
					{Option: 240, Value: "extra.options.kubevirt.io"},
					{Option: 241, Value: "sameextra.options.kubevirt.io"}},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should return error if not unique DHCPPrivateOptions", func() {
			testDHCPPrivateOptions := []v1.DHCPPrivateOptions{
				{Option: 240, Value: "extra.options.kubevirt.io"},
				{Option: 240, Value: "sameextra.options.kubevirt.io"},
			}
			err := ValidateDuplicateDHCPPrivateOptions(testDHCPPrivateOptions)
			Expect(err).To(Equal(fmt.Errorf("You have provided duplicate DHCPPrivateOptions")))
		})

		It("should not return error if unique DHCPPrivateOptions", func() {
			testDHCPPrivateOptions := []v1.DHCPPrivateOptions{
				{Option: 240, Value: "extra.options.kubevirt.io"},
				{Option: 241, Value: "sameextra.options.kubevirt.io"},
			}
			err := ValidateDuplicateDHCPPrivateOptions(testDHCPPrivateOptions)
			Expect(err).To(BeNil())
		})

		It("should reject vmi with a network multiqueue, without virtio nics", func() {
			_true := true
			vmi := api.NewMinimalVMI("testvm")
			nic := *v1.DefaultBridgeNetworkInterface()
			nic.Model = "e1000"
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{nic}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue = &_true
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.networkInterfaceMultiqueue"))
		})

		It("should allow BlockMultiQueue with CPU settings", func() {
			_true := true
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.BlockMultiQueue = &_true
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{}
			vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU] = resource.MustParse("5")

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should ignore CPU settings for explicitly rejected BlockMultiQueue", func() {
			_false := false
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.BlockMultiQueue = &_false

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should allow valid ioThreadsPolicy", func() {
			vmi := api.NewMinimalVMI("testvm")
			var ioThreadPolicy v1.IOThreadsPolicy
			ioThreadPolicy = "auto"
			vmi.Spec.Domain.IOThreadsPolicy = &ioThreadPolicy
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should reject invalid ioThreadsPolicy", func() {
			vmi := api.NewMinimalVMI("testvm")
			var ioThreadPolicy v1.IOThreadsPolicy
			ioThreadPolicy = "bad"
			vmi.Spec.Domain.IOThreadsPolicy = &ioThreadPolicy
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Message).To(Equal(fmt.Sprintf("Invalid IOThreadsPolicy (%s)", ioThreadPolicy)))
		})

		It("should reject GPU devices when feature gate is disabled", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{
					Name:       "gpu1",
					DeviceName: "vendor.com/gpu_name",
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.GPUs"))
		})
		It("should reject virtiofs filesystems when feature gate is disabled", func() {
			vmi := api.NewMinimalVMI("testvm")
			guestMemory := resource.MustParse("64Mi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{
				Hugepages: &v1.Hugepages{},
				Guest:     &guestMemory,
			}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "2Mi"
			vmi.Spec.Domain.Devices.Filesystems = []v1.Filesystem{
				{
					Name:     "sharednfstest",
					Virtiofs: &v1.FilesystemVirtiofs{},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.Filesystems"))
		})
		It("should allow virtiofs filesystems when feature gate is enabled", func() {
			enableFeatureGate(virtconfig.VirtIOFSGate)
			vmi := api.NewMinimalVMI("testvm")
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
			vmi.Spec.Domain.Devices.Filesystems = []v1.Filesystem{
				{
					Name:     "sharednfstest",
					Virtiofs: &v1.FilesystemVirtiofs{},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should accept legacy GPU devices if PermittedHostDevices aren't set", func() {
			kvConfig := kv.DeepCopy()
			kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{virtconfig.GPUGate}
			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)

			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{
					Name:       "gpu1",
					DeviceName: "example.org/deadbeef",
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should accept permitted GPU devices", func() {
			kvConfig := kv.DeepCopy()
			kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{virtconfig.GPUGate}
			kvConfig.Spec.Configuration.PermittedHostDevices = &v1.PermittedHostDevices{
				PciHostDevices: []v1.PciHostDevice{
					{
						PCIVendorSelector: "DEAD:BEEF",
						ResourceName:      "example.org/deadbeef",
					},
				},
			}
			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)

			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{
					Name:       "gpu1",
					DeviceName: "example.org/deadbeef",
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject host devices when feature gate is disabled", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.HostDevices = []v1.HostDevice{
				{
					Name:       "hostdev1",
					DeviceName: "vendor.com/hostdev_name",
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.HostDevices"))
		})
		It("should accept host devices that are not permitted in the hostdev config", func() {
			kvConfig := kv.DeepCopy()
			kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{virtconfig.HostDevicesGate}
			kvConfig.Spec.Configuration.PermittedHostDevices = &v1.PermittedHostDevices{
				PciHostDevices: []v1.PciHostDevice{
					{
						PCIVendorSelector: "DEAD:BEEF",
						ResourceName:      "example.org/deadbeef",
					},
				},
			}
			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.HostDevices = []v1.HostDevice{
				{
					Name:       "hostdev1",
					DeviceName: "example.org/deadbeef1",
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should accept permitted host devices", func() {
			kvConfig := kv.DeepCopy()
			kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{virtconfig.HostDevicesGate}
			kvConfig.Spec.Configuration.PermittedHostDevices = &v1.PermittedHostDevices{
				PciHostDevices: []v1.PciHostDevice{
					{
						PCIVendorSelector: "DEAD:BEEF",
						ResourceName:      "example.org/deadbeef",
					},
				},
			}
			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.HostDevices = []v1.HostDevice{
				{
					Name:       "hostdev1",
					DeviceName: "example.org/deadbeef",
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		table.DescribeTable("Should accept valid DNSPolicy and DNSConfig",
			func(dnsPolicy k8sv1.DNSPolicy, dnsConfig *k8sv1.PodDNSConfig) {
				vmi := api.NewMinimalVMI("testvmi")
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
				vmi := api.NewMinimalVMI("testvmi")
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
			table.Entry("with DNSPolicy None and search domain exceeding max length", k8sv1.DNSNone, &k8sv1.PodDNSConfig{
				Nameservers: []string{"1.2.3.4"},
				Searches:    []string{strings.Repeat("a", maxDNSSearchListChars/2), strings.Repeat("b", maxDNSSearchListChars/2)},
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
		It("should accept valid start strategy", func() {
			vmi := api.NewMinimalVMI("testvmi")
			strategy := v1.StartStrategyPaused
			vmi.Spec.StartStrategy = &strategy

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should allow no start strategy to be set", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.StartStrategy = nil
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject invalid start strategy", func() {
			vmi := api.NewMinimalVMI("testvmi")
			strategy := v1.StartStrategy("invalid")
			vmi.Spec.StartStrategy = &strategy

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake.startStrategy"))
			Expect(causes[0].Message).To(Equal("fake.startStrategy is set with an unrecognized option: invalid"))
		})
		It("should reject spec with paused start strategy and LivenessProbe", func() {
			vmi := api.NewMinimalVMI("testvmi")
			strategy := v1.StartStrategyPaused
			vmi.Spec.StartStrategy = &strategy
			vmi.Spec.LivenessProbe = &v1.Probe{
				InitialDelaySeconds: 2,
				Handler: v1.Handler{
					HTTPGet: &k8sv1.HTTPGetAction{Host: "test", Port: intstr.Parse("80")},
				},
			}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake.startStrategy"))
			Expect(causes[0].Message).To(Equal("either fake.startStrategy or fake.livenessProbe should be provided.Pausing VMI with LivenessProbe is not supported"))
		})
		Context("with kernel boot defined", func() {

			const (
				fakeKernelArgs = "args"
				fakeImage      = "image"
				fakeInitrd     = "initrd"
				fakeKernel     = "kernel"
			)

			table.DescribeTable("", func(kernelArgs, initrdPath, kernelPath, image string, defineContainerNil bool, shouldBeValid bool) {
				vmi := utils.GetVMIKernelBoot()

				kb := vmi.Spec.Domain.Firmware.KernelBoot

				if defineContainerNil {
					kb.Container = nil
				} else {
					kb.KernelArgs = kernelArgs
					kb.Container.KernelPath = kernelPath
					kb.Container.InitrdPath = initrdPath
					kb.Container.Image = image
				}

				kernelBootField := k8sfield.NewPath("spec").Child("domain").Child("firmware").Child("kernelBoot")
				causes := validateKernelBoot(kernelBootField, kb)

				if shouldBeValid {
					Expect(causes).To(BeEmpty())
				} else {
					Expect(causes).ToNot(BeEmpty())
				}
			},
				table.Entry("without kernel args and null container - should approve",
					"", "", "", "", true, true),
				table.Entry("with kernel args and null container - should approve",
					fakeKernelArgs, "", "", "", true, true),
				table.Entry("without kernel args, with container that has image & kernel & initrd defined - should approve",
					"", fakeInitrd, fakeKernel, fakeImage, false, true),
				table.Entry("with kernel args, with container that has image & kernel & initrd defined - should approve",
					fakeKernelArgs, fakeInitrd, fakeKernel, fakeImage, false, true),
				table.Entry("with kernel args, with container that has image & kernel defined - should approve",
					fakeKernelArgs, "", fakeKernel, fakeImage, false, true),
				table.Entry("with kernel args, with container that has image & initrd defined - should approve",
					fakeKernelArgs, fakeInitrd, "", fakeImage, false, true),
				table.Entry("with kernel args, with container that has only image defined - should reject",
					fakeKernelArgs, "", "", fakeImage, false, false),
				table.Entry("with kernel args, with container that has initrd and kernel defined but without image - should reject",
					fakeKernelArgs, fakeInitrd, fakeKernel, "", false, false),
				table.Entry("with kernel args, with container that has nothing defined", "", "", "", "", false, false),
			)
		})
	})

	Context("with cpu pinning", func() {
		var vmi *v1.VirtualMachineInstance
		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.CPU = &v1.CPU{DedicatedCPUPlacement: true}
			enableFeatureGate(virtconfig.NUMAFeatureGate)
		})
		It("should reject NUMA passthrough without DedicatedCPUPlacement without the NUMA feature gate", func() {
			disableFeatureGates()
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{PageSize: "2Mi"}}
			vmi.Spec.Domain.CPU.Cores = 4
			vmi.Spec.Domain.CPU.NUMA = &v1.NUMA{GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{}}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.cpu.numa.guestMappingPassthrough"))
			Expect(causes[0].Message).To(ContainSubstring("NUMA feature gate"))
		})
		It("should reject NUMA passthrough without DedicatedCPUPlacement", func() {
			vmi.Spec.Domain.CPU.NUMA = &v1.NUMA{GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{}}
			vmi.Spec.Domain.CPU.DedicatedCPUPlacement = false
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{PageSize: "2Mi"}}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.cpu.numa.guestMappingPassthrough"))
		})
		table.DescribeTable("should reject NUMA passthrough without hugepages", func(memory *v1.Memory) {
			vmi.Spec.Domain.CPU.NUMA = &v1.NUMA{GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{}}
			vmi.Spec.Domain.CPU.Cores = 4
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("4"),
			}
			vmi.Spec.Domain.CPU.DedicatedCPUPlacement = true
			vmi.Spec.Domain.Memory = memory
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.cpu.numa.guestMappingPassthrough"))
		},
			table.Entry("with no memory element", nil),
			table.Entry("with no hugepages element", &v1.Memory{Hugepages: nil}),
		)
		It("should accept NUMA passthrough with DedicatedCPUPlacement", func() {
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{PageSize: "2Mi"}}
			vmi.Spec.Domain.CPU.Cores = 4
			vmi.Spec.Domain.CPU.NUMA = &v1.NUMA{GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{}}
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("4"),
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})
		It("should reject specs with more than two threads", func() {
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{PageSize: "2Mi"}}
			vmi.Spec.Domain.CPU.Cores = 4
			vmi.Spec.Domain.CPU.Threads = 3
			vmi.Spec.Domain.CPU.NUMA = &v1.NUMA{GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{}}
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("12"),
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("Not more than two threads must be provided at fake.domain.cpu.threads (got 3) when DedicatedCPUPlacement is true"))
		})
		It("should reject specs without cpu reqirements", func() {
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.cpu.dedicatedCpuPlacement"))
		})
		It("should reject specs with IsolateEmulatorThread without DedicatedCPUPlacement set", func() {

			vmi.Spec.Domain.CPU = &v1.CPU{
				DedicatedCPUPlacement: false,
				IsolateEmulatorThread: true,
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.cpu.isolateEmulatorThread"))
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

	Context("with AccessCredentials", func() {
		It("should accept a valid ssh access credential with configdrive propagation", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testdisk",
				VolumeSource: v1.VolumeSource{
					CloudInitConfigDrive: &v1.CloudInitConfigDriveSource{UserData: " "},
				},
			})

			vmi.Spec.AccessCredentials = []v1.AccessCredential{
				{
					SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
						Source: v1.SSHPublicKeyAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: "my-pkey",
							},
						},
						PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
							ConfigDrive: &v1.ConfigDriveSSHPublicKeyAccessCredentialPropagation{},
						},
					},
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should accept a valid ssh access credential with qemu agent propagation", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.AccessCredentials = []v1.AccessCredential{
				{
					SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
						Source: v1.SSHPublicKeyAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: "my-pkey",
							},
						},
						PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
							QemuGuestAgent: &v1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{
								Users: []string{"madeup"},
							},
						},
					},
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should accept a valid user password access credential with qemu agent propagation", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.AccessCredentials = []v1.AccessCredential{
				{
					UserPassword: &v1.UserPasswordAccessCredential{
						Source: v1.UserPasswordAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: "my-pkey",
							},
						},
						PropagationMethod: v1.UserPasswordAccessCredentialPropagationMethod{
							QemuGuestAgent: &v1.QemuGuestAgentUserPasswordAccessCredentialPropagation{},
						},
					},
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should reject a configDrive ssh access credential when no configDrive volume exists", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testdisk",
				VolumeSource: v1.VolumeSource{
					CloudInitNoCloud: &v1.CloudInitNoCloudSource{UserData: " "},
				},
			})

			vmi.Spec.AccessCredentials = []v1.AccessCredential{
				{
					SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
						Source: v1.SSHPublicKeyAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: "my-pkey",
							},
						},
						PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
							ConfigDrive: &v1.ConfigDriveSSHPublicKeyAccessCredentialPropagation{},
						},
					},
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
		})
		It("should reject a ssh access credential without a source", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testdisk",
				VolumeSource: v1.VolumeSource{
					CloudInitConfigDrive: &v1.CloudInitConfigDriveSource{UserData: " "},
				},
			})

			vmi.Spec.AccessCredentials = []v1.AccessCredential{
				{
					SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
						PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
							ConfigDrive: &v1.ConfigDriveSSHPublicKeyAccessCredentialPropagation{},
						},
					},
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
		})

		It("should reject a ssh access credential with qemu agent propagation with no authorized key files listed", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.AccessCredentials = []v1.AccessCredential{
				{
					SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
						Source: v1.SSHPublicKeyAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: "my-pkey",
							},
						},
						PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
							QemuGuestAgent: &v1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{},
						},
					},
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
		})

		It("should reject a userpassword access credential without a source", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.AccessCredentials = []v1.AccessCredential{
				{
					UserPassword: &v1.UserPasswordAccessCredential{
						PropagationMethod: v1.UserPasswordAccessCredentialPropagationMethod{
							QemuGuestAgent: &v1.QemuGuestAgentUserPasswordAccessCredentialPropagation{},
						},
					},
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)

			Expect(len(causes)).To(Equal(1))
		})

		It("should reject a ssh access credential without a propagationMethod", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testdisk",
				VolumeSource: v1.VolumeSource{
					CloudInitConfigDrive: &v1.CloudInitConfigDriveSource{UserData: " "},
				},
			})

			vmi.Spec.AccessCredentials = []v1.AccessCredential{
				{
					SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
						Source: v1.SSHPublicKeyAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: "my-pkey",
							},
						},
					},
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
		})

		It("should reject a userpassword credential without a propagationMethod", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.AccessCredentials = []v1.AccessCredential{
				{
					UserPassword: &v1.UserPasswordAccessCredential{
						Source: v1.UserPasswordAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: "my-pkey",
							},
						},
					},
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
		})
	})

	Context("with CPU features", func() {
		It("should accept valid CPU feature policies", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.CPU = &v1.CPU{
				Features: []v1.CPUFeature{
					{
						Name: "lahf_lm",
					},
				},
			}

			for policy := range validCPUFeaturePolicies {
				vmi.Spec.Domain.CPU.Features[0].Policy = policy
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(len(causes)).To(Equal(0))
			}
		})

		It("should reject invalid CPU feature policy", func() {
			vmi := api.NewMinimalVMI("testvm")
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
				vmi := api.NewMinimalVMI("testvmi")

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
			vmi := api.NewMinimalVMI("testvmi")

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
			vmi := api.NewMinimalVMI("testvmi")

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
			vmi := api.NewMinimalVMI("testvmi")

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
			vmi := api.NewMinimalVMI("testvmi")

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
			vmi := api.NewMinimalVMI("testvmi")

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
			vmi := api.NewMinimalVMI("testvmi")

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
		It("should reject cd-roms using virtio bus", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testcdrom",
				DiskDevice: v1.DiskDevice{
					CDRom: &v1.CDRomTarget{
						Bus: "virtio",
					},
				},
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testcdrom",
				VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{Image: "fake"},
				},
			})

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[0].cdrom.bus"))
			Expect(causes[0].Message).To(Equal("Bus type virtio is invalid for CD-ROM device"))
		})

		It("should accept a boot order greater than '0'", func() {
			vmi := api.NewMinimalVMI("testvmi")
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
			vmi := api.NewMinimalVMI("testvmi")
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
			vmi := api.NewMinimalVMI("testvmi")

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
			vmi := api.NewMinimalVMI("testvmi")

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

		It("should reject disks with unsupported I/O modes", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk1",
				IO:   "native",
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk2",
				IO:   "unsupported",
			})

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[1].io"))
		})

		It("should reject disk with invalid cache mode", func() {
			vmi := api.NewMinimalVMI("testvmi")
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
			vmi := api.NewMinimalVMI("testvmi")
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
			vmi := api.NewMinimalVMI("testvmi")
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
			vmi := api.NewMinimalVMI("testvmi")
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
			vmi := api.NewMinimalVMI("testvmi")
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

		It("Should reject disk with DedicatedIOThread and SATA bus", func() {
			vmi := api.NewMinimalVMI("testvmi")
			_true := true
			_false := false

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks,
				v1.Disk{
					Name:              "disk-with-dedicated-io-thread-and-sata",
					DedicatedIOThread: &_true,
					DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{
						Bus: "sata",
					}},
				},
				v1.Disk{
					Name:              "disk-with-dedicated-io-thread-and-virtio",
					DedicatedIOThread: &_true,
					DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{
						Bus: "virtio",
					}},
				},
				v1.Disk{
					Name:              "disk-without-dedicated-io-thread-and-with-sata",
					DedicatedIOThread: &_false,
					DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{
						Bus: "sata",
					}},
				},
			)

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1)) // Only first disk should fail
			Expect(string(causes[0].Type)).To(Equal("FieldValueNotSupported"))
			Expect(causes[0].Field).To(ContainSubstring("domain.devices.disks"))
			Expect(causes[0].Message).To(Equal(fmt.Sprintf("IOThreads are not supported for disks on a SATA bus")))

		})

		Context("With block size", func() {

			table.DescribeTable("It should accept a disk with a valid block size of", func(logicalSize, physicalSize int) {
				vmi := api.NewMinimalVMI("testvmi")

				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "blockdisk",
					BlockSize: &v1.BlockSize{
						Custom: &v1.CustomBlockSize{
							Logical:  uint(logicalSize),
							Physical: uint(physicalSize),
						},
					},
				})

				causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
				Expect(len(causes)).To(Equal(0))
			},
				table.Entry("a 512n disk", 512, 512),
				table.Entry("a 512e disk", 512, 4096),
				table.Entry("a 4096n (4kn) disk", 4096, 4096),
				table.Entry("a custom 1 MiB disk", 1048576, 1048576),
			)

			table.DescribeTable("It should deny a disk's block size configuration when", func(logicalSize, physicalSize int) {
				vmi := api.NewMinimalVMI("testvmi")

				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "blockdisk",
					BlockSize: &v1.BlockSize{
						Custom: &v1.CustomBlockSize{
							Logical:  uint(logicalSize),
							Physical: uint(physicalSize),
						},
					},
				})

				causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
				Expect(len(causes)).To(Equal(2))
				Expect(causes[0].Field).To(Equal("fake[0].blockSize.custom.logical"))
				Expect(causes[1].Field).To(Equal("fake[0].blockSize.custom.physical"))
			},
				table.Entry("less than 512", 128, 128),
				table.Entry("greater than 2 MiB", 3000000, 3000000),
				table.Entry("not a power of 2", 1234, 1234),
			)

			It("Should deny a disk's block size configuration when logical > physical", func() {
				vmi := api.NewMinimalVMI("testvmi")

				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "blockdisk",
					BlockSize: &v1.BlockSize{
						Custom: &v1.CustomBlockSize{
							Logical:  4096,
							Physical: 512,
						},
					},
				})

				causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
				Expect(len(causes)).To(Equal(1))
				Expect(causes[0].Field).To(Equal("fake[0].blockSize.custom.logical"))
			})

			It("Should accept disks with block size matching enabled", func() {
				vmi := api.NewMinimalVMI("testvmi")

				_true := true
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "blockdisk",
					BlockSize: &v1.BlockSize{
						MatchVolume: &v1.FeatureState{
							Enabled: &_true,
						},
					},
				})

				causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
				Expect(len(causes)).To(Equal(0))
			})

			It("Should reject disk with custom block size and size matching enabled", func() {
				vmi := api.NewMinimalVMI("testvmi")

				_true := true
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "blockdisk",
					BlockSize: &v1.BlockSize{
						Custom: &v1.CustomBlockSize{
							Logical:  1234,
							Physical: 1234,
						},
						MatchVolume: &v1.FeatureState{
							Enabled: &_true,
						},
					},
				})

				causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
				Expect(len(causes)).To(Equal(1))
				Expect(causes[0].Field).To(Equal("fake[0].blockSize"))
			})

			It("Should accept disks with a custom block size and size matching explicitly disabled", func() {
				vmi := api.NewMinimalVMI("testvmi")

				_false := false
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "blockdisk",
					BlockSize: &v1.BlockSize{
						Custom: &v1.CustomBlockSize{
							Logical:  4096,
							Physical: 4096,
						},
						MatchVolume: &v1.FeatureState{
							Enabled: &_false,
						},
					},
				})

				causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
				Expect(len(causes)).To(Equal(0))
			})
		})
	})

	Context("with volume", func() {
		It("should accept a single downwardmetrics volume", func() {
			enableFeatureGate(virtconfig.DownwardMetricsFeatureGate)
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testDownwardMetrics",
				VolumeSource: v1.VolumeSource{
					DownwardMetrics: &v1.DownwardMetricsVolumeSource{},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(BeEmpty())
		})
		It("should reject downwardMetrics volumes if the feature gate is not enabled", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testDownwardMetrics",
				VolumeSource: v1.VolumeSource{
					DownwardMetrics: &v1.DownwardMetricsVolumeSource{},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("downwardMetrics disks are not allowed: DownwardMetrics feature gate is not enabled."))
		})
		It("should reject downwardMetrics volumes if more than one exist", func() {
			enableFeatureGate(virtconfig.DownwardMetricsFeatureGate)
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes,
				v1.Volume{
					Name: "testDownwardMetrics",
					VolumeSource: v1.VolumeSource{
						DownwardMetrics: &v1.DownwardMetricsVolumeSource{},
					},
				},
				v1.Volume{
					Name: "testDownwardMetrics1",
					VolumeSource: v1.VolumeSource{
						DownwardMetrics: &v1.DownwardMetricsVolumeSource{},
					},
				},
			)
			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("fake must have max one downwardMetric volume set"))
		})
		It("should reject hostDisk volumes if the feature gate is not enabled", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testHostDisk",
				VolumeSource: v1.VolumeSource{
					HostDisk: &v1.HostDisk{
						Type: v1.HostDiskExistsOrCreate,
						Path: "/hostdisktest.img",
					},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
		})

		It("should accept hostDisk volumes if the feature gate is enabled", func() {
			enableFeatureGate(virtconfig.HostDiskGate)
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testHostDisk",
				VolumeSource: v1.VolumeSource{
					HostDisk: &v1.HostDisk{
						Type: v1.HostDiskExistsOrCreate,
						Path: "/hostdisktest.img",
					},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(BeEmpty())
		})

		It("should accept sysprep volumes", func() {
			vmi := api.NewMinimalVMI("fake-vmi")
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "sysprep-configmap-volume",
				VolumeSource: v1.VolumeSource{
					Sysprep: &v1.SysprepSource{
						ConfigMap: &k8sv1.LocalObjectReference{
							Name: "test-config",
						},
					},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(BeEmpty())
		})

		It("should reject CloudInitNoCloud volume if either userData or networkData is missing", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testdisk",
				VolumeSource: v1.VolumeSource{
					CloudInitNoCloud: &v1.CloudInitNoCloudSource{},
				},
			})
			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
		})

		It("should accept CloudInitNoCloud volume if it has only a userData source", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testdisk",
				VolumeSource: v1.VolumeSource{
					CloudInitNoCloud: &v1.CloudInitNoCloudSource{UserData: " "},
				},
			})
			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(BeEmpty())
		})

		It("should accept CloudInitNoCloud volume if it has only a networkData source", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testdisk",
				VolumeSource: v1.VolumeSource{
					CloudInitNoCloud: &v1.CloudInitNoCloudSource{NetworkData: " "},
				},
			})
			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(BeEmpty())
		})

		It("should accept CloudInitNoCloud volume if it has both userData and networkData sources", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testdisk",
				VolumeSource: v1.VolumeSource{
					CloudInitNoCloud: &v1.CloudInitNoCloudSource{UserData: " ", NetworkData: " "},
				},
			})
			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(BeEmpty())
		})

	})

	Context("with bootloader", func() {
		It("should accept empty bootloader setting", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Subdomain = "testsubdomain"

			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: nil,
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should accept BIOS", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Subdomain = "testsubdomain"

			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					BIOS: &v1.BIOS{},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should accept EFI with SMM", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Subdomain = "testsubdomain"

			_true := true
			vmi.Spec.Domain.Features = &v1.Features{
				SMM: &v1.FeatureState{
					Enabled: &_true,
				},
			}
			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					EFI: &v1.EFI{},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should not accept EFI without SMM", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Subdomain = "testsubdomain"

			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					EFI: &v1.EFI{},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
		})

		It("should accept EFI without secureBoot and without SMM", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Subdomain = "testsubdomain"

			_false := false
			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					EFI: &v1.EFI{
						SecureBoot: &_false,
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(0))
		})

		It("should not accept BIOS and EFI together", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Subdomain = "testsubdomain"

			_true := true
			vmi.Spec.Domain.Features = &v1.Features{
				SMM: &v1.FeatureState{
					Enabled: &_true,
				},
			}
			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					EFI:  &v1.EFI{},
					BIOS: &v1.BIOS{},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(Equal(1))
		})

		It("should reject disk without a valid DNS-1123 name", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "TESTDISK2",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1))
		})
	})

	Context("with verification for Arm64", func() {
		It("should reject BIOS bootloader", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					BIOS: &v1.BIOS{},
				},
			}

			causes := webhooks.ValidateVirtualMachineInstanceArm64Setting(k8sfield.NewPath("spec"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
		})

		It("should reject UEFI default bootloader", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					EFI: &v1.EFI{},
				},
			}

			causes := webhooks.ValidateVirtualMachineInstanceArm64Setting(k8sfield.NewPath("spec"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
		})

		It("should reject UEFI secure bootloader", func() {
			vmi := api.NewMinimalVMI("testvmi")

			_true := true
			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					EFI: &v1.EFI{
						SecureBoot: &_true,
					},
				},
			}

			causes := webhooks.ValidateVirtualMachineInstanceArm64Setting(k8sfield.NewPath("spec"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
		})

		It("should reject setting cpu model to host-model", func() {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.CPU = &v1.CPU{Model: "host-model"}

			causes := webhooks.ValidateVirtualMachineInstanceArm64Setting(k8sfield.NewPath("spec"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
		})
	})

	Context("with realtime", func() {
		var vmi *v1.VirtualMachineInstance
		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.CPU = &v1.CPU{Realtime: &v1.Realtime{}, Cores: 4}
			enableFeatureGate(virtconfig.NUMAFeatureGate)
		})
		It("should reject the realtime knob without DedicatedCPUPlacement", func() {
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{PageSize: "2Mi"}}
			vmi.Spec.Domain.CPU.NUMA = &v1.NUMA{GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{}}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(len(causes)).To(BeNumerically(">=", 1))
			Expect(causes).To(ContainElement(metav1.StatusCause{Type: metav1.CauseTypeFieldValueRequired, Field: "fake.domain.cpu.dedicatedCpuPlacement", Message: "fake.domain.cpu.dedicatedCpuPlacement must be set to true when fake.domain.cpu.realtime is used"}))
		})
		It("should reject the realtime knob when NUMA Guest Mapping Passthrough is not defined", func() {
			vmi.Spec.Domain.CPU.DedicatedCPUPlacement = true
			vmi.Spec.Domain.CPU.NUMA = &v1.NUMA{}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes).To(ContainElement(metav1.StatusCause{Type: metav1.CauseTypeFieldValueRequired, Field: "fake.domain.cpu.numa.guestMappingPassthrough", Message: "fake.domain.cpu.numa.guestMappingPassthrough must be defined when fake.domain.cpu.realtime is used"}))
		})
		It("should reject the realtime knob when NUMA is nil", func() {
			vmi.Spec.Domain.CPU.DedicatedCPUPlacement = true
			vmi.Spec.Domain.CPU.NUMA = nil
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes).To(ContainElement(metav1.StatusCause{Type: metav1.CauseTypeFieldValueRequired, Field: "fake.domain.cpu.numa.guestMappingPassthrough", Message: "fake.domain.cpu.numa.guestMappingPassthrough must be defined when fake.domain.cpu.realtime is used"}))
		})
	})
})

var _ = Describe("Function getNumberOfPodInterfaces()", func() {
	config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})

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
			Name: "testnet",
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
				ContainerDisk: testutils.NewFakeContainerDiskSource(),
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
				ContainerDisk: testutils.NewFakeContainerDiskSource(),
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

	It("Should validate VMIs without HyperV configuration", func() {
		vmi := api.NewMinimalVMI("testvmi")
		Expect(vmi.Spec.Domain.Features).To(BeNil())
		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(len(causes)).To(Equal(0))
	})

	It("Should validate VMIs with empty HyperV configuration", func() {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{},
		}
		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(len(causes)).To(Equal(0))
	})

	It("Should validate VMIs with hyperv configuration without deps", func() {
		_true := true
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				Relaxed: &v1.FeatureState{
					Enabled: &_true,
				},
				Runtime: &v1.FeatureState{
					Enabled: &_true,
				},
				Reset: &v1.FeatureState{
					Enabled: &_true,
				},
			},
		}
		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(len(causes)).To(Equal(0))
	})

	It("Should validate VMIs with hyperv EVMCS configuration without deps and detect multiple issues", func() {
		_true := true
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				EVMCS: &v1.FeatureState{
					Enabled: &_true,
				},
			},
		}
		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(len(causes)).To(Equal(2), "should return error")
		Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid), "type should equal")
		Expect(causes[0].Field).To(Equal("spec.domain.features.hyperv.evmcs"), "field should equal")
		Expect(causes[1].Type).To(Equal(metav1.CauseTypeFieldValueRequired), "type should equal")
		Expect(causes[1].Field).To(Equal("spec.domain.cpu.features"), "field should equal")
	})
	It("Should validate VMIs with hyperv EVMCS configuration without deps", func() {
		_true := true
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.CPU = &v1.CPU{
			Features: []v1.CPUFeature{
				{
					Name:   nodelabellerutil.VmxFeature,
					Policy: nodelabellerutil.RequirePolicy,
				},
			},
		}
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				EVMCS: &v1.FeatureState{
					Enabled: &_true,
				},
			},
		}
		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(len(causes)).To(Equal(1), "should return error")
		Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid), "type should equal")
		Expect(causes[0].Field).To(Equal("spec.domain.features.hyperv.evmcs"), "field should equal")
	})

	It("Should validate VMIs with hyperv EVMCS configuration with hyperv deps, but without vmx cpu feature", func() {
		_true := true
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				EVMCS: &v1.FeatureState{
					Enabled: &_true,
				},
				VAPIC: &v1.FeatureState{
					Enabled: &_true,
				},
			},
		}
		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(len(causes)).To(Equal(1), "should return error")
		Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueRequired), "type should equal")
		Expect(causes[0].Field).To(Equal("spec.domain.cpu.features"), "field should equal")
	})

	It("Should validate VMIs with hyperv EVMCS configuration with vmx forbid", func() {
		_true := true
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.CPU = &v1.CPU{
			Features: []v1.CPUFeature{
				{
					Name:   nodelabellerutil.VmxFeature,
					Policy: "forbid",
				},
			},
		}
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				EVMCS: &v1.FeatureState{
					Enabled: &_true,
				},
				VAPIC: &v1.FeatureState{
					Enabled: &_true,
				},
			},
		}
		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(len(causes)).To(Equal(1), "should return error")
		Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid), "type should equal")
		Expect(causes[0].Field).To(Equal("spec.domain.cpu.features[0].policy"), "field should equal")
	})

	It("Should validate VMIs with hyperv EVMCS configuration with wrong vmx policy", func() {
		_true := true
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.CPU = &v1.CPU{
			Features: []v1.CPUFeature{
				{
					Name:   nodelabellerutil.VmxFeature,
					Policy: nodelabellerutil.RequirePolicy,
				},
			},
		}
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				EVMCS: &v1.FeatureState{
					Enabled: &_true,
				},
				VAPIC: &v1.FeatureState{
					Enabled: &_true,
				},
			},
		}
		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(len(causes)).To(Equal(0), "should not return error")
	})

	It("Should not validate VMIs with broken hyperv deps", func() {
		_true := true
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				Relaxed: &v1.FeatureState{
					Enabled: &_true,
				},
				SyNIC: &v1.FeatureState{
					Enabled: &_true,
				},
				SyNICTimer: &v1.SyNICTimer{
					Enabled: &_true,
				},
			},
		}
		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(len(causes)).To(BeNumerically(">=", 1))
	})

	It("Should validate VMIs with correct hyperv deps", func() {
		_true := true
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				Relaxed: &v1.FeatureState{
					Enabled: &_true,
				},
				VPIndex: &v1.FeatureState{
					Enabled: &_true,
				},
				SyNIC: &v1.FeatureState{
					Enabled: &_true,
				},
				SyNICTimer: &v1.SyNICTimer{
					Enabled: &_true,
				},
			},
		}

		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(len(causes)).To(Equal(0))
	})
})
