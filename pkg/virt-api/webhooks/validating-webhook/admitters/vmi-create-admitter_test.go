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
 *
 */

package admitters

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authentication/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	nodelabellerutil "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
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
			Phase:               v1.KubeVirtPhaseDeploying,
			DefaultArchitecture: "amd64",
		},
	}
	config, _, kvStore := testutils.NewFakeClusterConfigUsingKV(kv)
	const kubeVirtNamespace = "kubevirt"
	kubeVirtServiceAccounts := webhooks.KubeVirtServiceAccounts(kubeVirtNamespace)
	vmiCreateAdmitter := &VMICreateAdmitter{ClusterConfig: config, KubeVirtServiceAccounts: kubeVirtServiceAccounts}

	dnsConfigTestOption := "test"
	enableFeatureGates := func(featureGates ...string) {
		kvConfig := kv.DeepCopy()
		kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = featureGates
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvConfig)
	}
	disableFeatureGates := func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
	}

	updateDefaultArchitecture := func(defaultArchitecture string) {
		kvConfig := kv.DeepCopy()
		kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{featuregate.MultiArchitecture}
		kvConfig.Status.DefaultArchitecture = defaultArchitecture

		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvConfig)
	}

	AfterEach(func() {
		disableFeatureGates()
	})

	It("when spec validator pass, should allow", func() {
		ar, err := newAdmissionReviewForVMICreation(newBaseVmi())
		Expect(err).ToNot(HaveOccurred())

		admitter := &VMICreateAdmitter{
			ClusterConfig:           config,
			KubeVirtServiceAccounts: kubeVirtServiceAccounts,
			SpecValidators:          []SpecValidator{newValidateStub()},
		}
		resp := admitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeTrue())
	})

	It("when spec validator fail, should reject", func() {
		expectedStatusCauses := []metav1.StatusCause{{Type: "test", Message: "test", Field: "test"}}
		admitter := &VMICreateAdmitter{
			ClusterConfig:           config,
			KubeVirtServiceAccounts: kubeVirtServiceAccounts,
			SpecValidators:          []SpecValidator{newValidateStub(expectedStatusCauses...)},
		}
		ar, err := newAdmissionReviewForVMICreation(newBaseVmi())
		Expect(err).ToNot(HaveOccurred())

		resp := admitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Details.Causes).To(Equal(expectedStatusCauses))
	})

	It("should reject invalid VirtualMachineInstance spec on create", func() {
		vmi := newBaseVmi()
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "testdisk",
		})

		ar, err := newAdmissionReviewForVMICreation(vmi)
		Expect(err).ToNot(HaveOccurred())

		resp := vmiCreateAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Details.Causes).To(HaveLen(1))
		Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.domain.devices.disks[0].name"))
	})

	It("should reject VMIs without memory after presets were applied", func() {
		vmi := newBaseVmi()
		vmi.Spec.Domain.Resources = v1.ResourceRequirements{}

		ar, err := newAdmissionReviewForVMICreation(vmi)
		Expect(err).ToNot(HaveOccurred())

		resp := vmiCreateAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Details.Causes).To(HaveLen(1))
		Expect(resp.Result.Message).To(ContainSubstring("no memory requested"))
	})

	It("should allow Clock without Timer", func() {
		const offsetSeconds = 5
		vmi := newBaseVmi(withDomainClock(
			&v1.Clock{
				ClockOffset: v1.ClockOffset{
					UTC: &v1.ClockOffsetUTC{
						OffsetSeconds: pointer.P(offsetSeconds),
					},
				},
			},
		))

		ar, err := newAdmissionReviewForVMICreation(vmi)
		Expect(err).ToNot(HaveOccurred())

		resp := vmiCreateAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeTrue())
	})

	DescribeTable("container disk path validation should fail", func(containerDiskPath, expectedCause string) {
		vmi := newBaseVmi(libvmi.WithContainerDisk("testdisk", "testimage"))
		vmi.Spec.Volumes[0].ContainerDisk.Path = containerDiskPath

		ar, err := newAdmissionReviewForVMICreation(vmi)
		Expect(err).ToNot(HaveOccurred())

		resp := vmiCreateAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Details.Causes).To(HaveLen(1))
		Expect(resp.Result.Message).To(Equal(expectedCause))
	},
		Entry("when path is not absolute", "a/b/c", "spec.volumes[0].containerDisk must be an absolute path to a file without relative components"),
		Entry("when path contains relative components", "/a/b/c/../d", "spec.volumes[0].containerDisk must be an absolute path to a file without relative components"),
		Entry("when path is root", "/", "spec.volumes[0].containerDisk must not point to root"),
	)

	DescribeTable("container disk path validation should succeed", func(containerDiskPath string) {
		vmi := newBaseVmi(libvmi.WithContainerDisk("testdisk", "testimage"))
		vmi.Spec.Volumes[0].ContainerDisk.Path = containerDiskPath

		ar, err := newAdmissionReviewForVMICreation(vmi)
		Expect(err).ToNot(HaveOccurred())

		resp := vmiCreateAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeTrue())
	},
		Entry("when path is absolute", "/a/b/c"),
		Entry("when path is absolute and has trailing slash", "/a/b/c/"),
	)

	Context("with eviction strategies", func() {
		DescribeTable("it should allow", func(vmi *v1.VirtualMachineInstance) {
			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)

			Expect(resp.Allowed).To(BeTrue())
			Expect(resp.Result).To(BeNil())
		},
			Entry("eviction strategy to be set to LiveMigrate",
				newBaseVmi(libvmi.WithEvictionStrategy(v1.EvictionStrategyLiveMigrate)),
			),
			Entry("eviction strategy to be set None",
				newBaseVmi(libvmi.WithEvictionStrategy(v1.EvictionStrategyNone)),
			),
			Entry("eviction strategy to be set External",
				newBaseVmi(libvmi.WithEvictionStrategy(v1.EvictionStrategyExternal)),
			),
			Entry("eviction strategy to be set to LiveMigrateIfPossible",
				newBaseVmi(libvmi.WithEvictionStrategy(v1.EvictionStrategyLiveMigrateIfPossible)),
			),
			Entry("eviction strategy to be set to nil (unspecified)",
				newBaseVmi(),
			),
		)

		It("should not allow unknown eviction strategy", func() {
			vmi := newBaseVmi(libvmi.WithEvictionStrategy(v1.EvictionStrategy("fantasy")))

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)

			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(Equal("spec.evictionStrategy is set with an unrecognized option: fantasy"))
		})
	})

	Context("with probes given", func() {
		It("should reject probes with no probe action configured", func() {
			vmi := newBaseVmi(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				withReadinessProbe(&v1.Probe{InitialDelaySeconds: 2}),
				withLivenessProbe(&v1.Probe{InitialDelaySeconds: 2}),
			)

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(Equal(`either spec.readinessProbe.tcpSocket, spec.readinessProbe.exec or spec.readinessProbe.httpGet must be set if a spec.readinessProbe is specified, either spec.livenessProbe.tcpSocket, spec.livenessProbe.exec or spec.livenessProbe.httpGet must be set if a spec.livenessProbe is specified`))
		})
		It("should reject probes with more than one action per probe configured", func() {
			vmi := newBaseVmi(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				withReadinessProbe(&v1.Probe{
					InitialDelaySeconds: 2,
					Handler: v1.Handler{
						HTTPGet:        &k8sv1.HTTPGetAction{Host: "test", Port: intstr.Parse("80")},
						TCPSocket:      &k8sv1.TCPSocketAction{Host: "lal", Port: intstr.Parse("80")},
						Exec:           &k8sv1.ExecAction{Command: []string{"uname", "-a"}},
						GuestAgentPing: &v1.GuestAgentPing{},
					},
				}),
				withLivenessProbe(&v1.Probe{
					InitialDelaySeconds: 2,
					Handler: v1.Handler{
						HTTPGet:   &k8sv1.HTTPGetAction{Host: "test", Port: intstr.Parse("80")},
						TCPSocket: &k8sv1.TCPSocketAction{Host: "lal", Port: intstr.Parse("80")},
						Exec:      &k8sv1.ExecAction{Command: []string{"uname", "-a"}},
					},
				}),
			)

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(Equal(`spec.readinessProbe must have exactly one probe type set, spec.livenessProbe must have exactly one probe type set`))
		})
		It("should accept properly configured readiness and liveness probes", func() {
			vmi := newBaseVmi(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				withReadinessProbe(&v1.Probe{
					InitialDelaySeconds: 2,
					Handler: v1.Handler{
						TCPSocket: &k8sv1.TCPSocketAction{Host: "lal", Port: intstr.Parse("80")},
					},
				}),
				withLivenessProbe(&v1.Probe{
					InitialDelaySeconds: 2,
					Handler: v1.Handler{
						HTTPGet: &k8sv1.HTTPGetAction{Host: "test", Port: intstr.Parse("80")},
					},
				}),
			)

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})
		It("should reject properly configured network-based readiness and liveness probes if no Pod Network is present", func() {
			vmi := newBaseVmi(
				libvmi.WithAutoAttachPodInterface(false),
				withReadinessProbe(&v1.Probe{
					InitialDelaySeconds: 2,
					Handler: v1.Handler{
						TCPSocket: &k8sv1.TCPSocketAction{Host: "lal", Port: intstr.Parse("80")},
					},
				}),
				withLivenessProbe(&v1.Probe{
					InitialDelaySeconds: 2,
					Handler: v1.Handler{
						HTTPGet: &k8sv1.HTTPGetAction{Host: "test", Port: intstr.Parse("80")},
					},
				}),
			)

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(Equal(`spec.readinessProbe.tcpSocket is only allowed if the Pod Network is attached, spec.livenessProbe.httpGet is only allowed if the Pod Network is attached`))
		})
	})

	It("should accept valid vmi spec on create", func() {
		vmi := newBaseVmi(libvmi.WithContainerDisk("testdisk", "testimage"))

		ar, err := newAdmissionReviewForVMICreation(vmi)
		Expect(err).ToNot(HaveOccurred())

		resp := vmiCreateAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeTrue())
	})

	It("should allow unknown fields in the status to allow updates", func() {
		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
				Object: k8sruntime.RawExtension{
					Raw: []byte(`{"very": "unknown", "spec": { "extremely": "unknown" }, "status": {"unknown": "allowed"}}`),
				},
			},
		}
		resp := vmiCreateAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Message).To(Equal(`.very in body is a forbidden property, spec.extremely in body is a forbidden property, spec.domain in body is required`))
	})

	DescribeTable("should reject documents containing unknown or missing fields for", func(data string, validationResult string, gvr metav1.GroupVersionResource, review func(ctx context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse) {
		input := map[string]interface{}{}
		json.Unmarshal([]byte(data), &input)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: gvr,
				Object: k8sruntime.RawExtension{
					Raw: []byte(data),
				},
			},
		}
		resp := review(context.Background(), ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Message).To(Equal(validationResult))
	},
		Entry("VirtualMachineInstance creation",
			`{"very": "unknown", "spec": { "extremely": "unknown" }}`,
			`.very in body is a forbidden property, spec.extremely in body is a forbidden property, spec.domain in body is required`,
			webhooks.VirtualMachineInstanceGroupVersionResource,
			vmiCreateAdmitter.Admit,
		),
	)

	Context("with VirtualMachineInstance metadata", func() {
		DescribeTable(
			"Should allow VMI creation with kubevirt.io/ labels only for kubevirt service accounts",
			func(labels map[string]string, userAccount string) {
				vmi := newBaseVmi()
				vmi.Labels = labels

				ar, err := newAdmissionReviewForVMICreation(vmi)
				Expect(err).ToNot(HaveOccurred())
				ar.Request.UserInfo = authv1.UserInfo{Username: "system:serviceaccount:kubevirt:" + userAccount}

				resp := vmiCreateAdmitter.Admit(context.Background(), ar)
				Expect(resp.Allowed).To(BeTrue())
				Expect(resp.Result).To(BeNil())
			},
			Entry("Create restricted label by API",
				map[string]string{v1.NodeNameLabel: "someValue"},
				components.ApiServiceAccountName,
			),
			Entry("Create restricted label by Handler",
				map[string]string{v1.NodeNameLabel: "someValue"},
				components.HandlerServiceAccountName,
			),
			Entry("Create restricted label by Controller",
				map[string]string{v1.NodeNameLabel: "someValue"},
				components.ControllerServiceAccountName,
			),
			Entry("Create non restricted kubevirt.io prefixed label by non kubevirt user",
				map[string]string{"kubevirt.io/l": "someValue"},
				"user-account",
			),
		)

		It("should reject restricted label by non kubevirt user", func() {
			vmi := newBaseVmi(libvmi.WithLabel(v1.NodeNameLabel, "someValue"))

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())
			ar.Request.UserInfo = authv1.UserInfo{Username: "system:serviceaccount:fake:" + "user-account"}

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("creation of the following reserved kubevirt.io/ labels on a VMI object is prohibited"))
		})

		DescribeTable("should reject annotations which require feature gate enabled", func(annotations map[string]string, expectedMsg string) {
			vmi := newBaseVmi()
			vmi.Annotations = annotations

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())
			ar.Request.UserInfo = authv1.UserInfo{Username: "fake-account"}

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(resp.Result.Details.Causes[0].Message).To(ContainSubstring(expectedMsg))
		},
			Entry("without ExperimentalIgnitionSupport feature gate enabled",
				map[string]string{v1.IgnitionAnnotation: "fake-data"},
				fmt.Sprintf("invalid entry metadata.annotations.%s", v1.IgnitionAnnotation),
			),
			Entry("without sidecar feature gate enabled",
				map[string]string{hooks.HookSidecarListAnnotationName: "[{'image': 'fake-image'}]"},
				fmt.Sprintf("invalid entry metadata.annotations.%s", hooks.HookSidecarListAnnotationName),
			),
		)

		DescribeTable("should accept annotations which require feature gate enabled", func(annotations map[string]string, featureGate string) {
			enableFeatureGates(featureGate)
			vmi := newBaseVmi()
			vmi.Annotations = annotations

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())
			ar.Request.UserInfo = authv1.UserInfo{Username: "fake-account"}

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
			Expect(resp.Result).To(BeNil())
		},
			Entry("with ExperimentalIgnitionSupport feature gate enabled",
				map[string]string{v1.IgnitionAnnotation: "fake-data"},
				featuregate.IgnitionGate,
			),
			Entry("with sidecar feature gate enabled",
				map[string]string{hooks.HookSidecarListAnnotationName: "[{'image': 'fake-image'}]"},
				featuregate.SidecarGate,
			),
		)
	})

	Context("with VirtualMachineInstance spec", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
		})
		DescribeTable("should accept valid machine type", func(arch string, machineType string) {
			enableFeatureGates(featuregate.MultiArchitecture)

			vmi.Spec.Architecture = arch
			vmi.Spec.Domain.Machine = &v1.Machine{Type: machineType}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		},
			Entry("when architecture is amd64", "amd64", "q35"),
			Entry("when architecture is arm64", "arm64", "virt"),
			Entry("when architecture is ppc64le", "ppc64le", "pseries"),
			Entry("when architecture is s390x", "s390x", "s390-ccw-virtio"),
		)

		DescribeTable("should reject invalid machine type", func(arch string, machineType string) {
			enableFeatureGates(featuregate.MultiArchitecture)

			vmi.Spec.Architecture = arch
			vmi.Spec.Domain.Machine = &v1.Machine{Type: machineType}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake.domain.machine.type"))
			Expect(causes[0].Message).To(ContainSubstring(fmt.Sprintf("fake.domain.machine.type is not supported: %s (allowed values:", machineType)))
		},
			Entry("Simple wrong value", "amd64", "test"),
			Entry("Wrong prefix amd64 q35", "amd64", "test-q35"),
			Entry("Wrong prefix amd64 pc-q35", "amd64", "test-pc-q35"),
			Entry("Wrong prefix arm64", "arm64", "test-virt"),
			Entry("Wrong prefix ppc64le", "ppc64le", "test-pseries"),
			Entry("Wrong prefix s390x", "s390x", "test-s390-ccw-virtio"),
		)

		It("should accept valid hostname", func() {
			vmi.Spec.Hostname = "test"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should reject invalid hostname", func() {
			vmi.Spec.Hostname = "test+bad"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake.hostname"))
			Expect(causes[0].Message).To(ContainSubstring("does not conform to the kubernetes DNS_LABEL rules : "))
		})

		It("should accept valid subdomain name", func() {
			vmi.Spec.Subdomain = "testsubdomain"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should reject invalid subdomain name", func() {
			vmi.Spec.Subdomain = "bad+domain"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.subdomain"))
		})

		It("should reject disk with missing volume", func() {
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[0].name"))
		})

		It("should allow cd-rom disk with missing volume and featuregate", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				DiskDevice: v1.DiskDevice{
					CDRom: &v1.CDRomTarget{
						Bus: v1.DiskBusSATA,
					},
				},
				Name: "testdisk",
			})

			enableFeatureGates(featuregate.DeclarativeHotplugVolumesGate)
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should reject cd-rom disk with missing volume and featuregate", func() {
			vmi := api.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				DiskDevice: v1.DiskDevice{
					CDRom: &v1.CDRomTarget{
						Bus: v1.DiskBusSATA,
					},
				},
				Name: "testdisk",
			})

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(Equal(fmt.Sprintf("%s feature gate not enabled, cannot define an empty CD-ROM disk", featuregate.DeclarativeHotplugVolumesGate)))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[0].name"))
		})
		It("should allow supported audio devices", func() {
			supportedDevices := [...]string{"", "ich9", "ac97"}

			for _, deviceName := range supportedDevices {
				vmi.Spec.Domain.Devices.Sound = &v1.SoundDevice{
					Name:  "audio-device",
					Model: deviceName,
				}
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(causes).To(BeEmpty())
			}
		})

		It("should reject unsupported audio devices", func() {
			vmi.Spec.Domain.Devices.Sound = &v1.SoundDevice{
				Name:  "audio-device",
				Model: "aNotSupportedDevice",
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.Sound"))
			Expect(causes[0].Message).To(ContainSubstring("Sound device type is not supported"))
		})

		It("should reject audio devices without name fields", func() {
			supportedAudioDevice := "ac97"
			vmi.Spec.Domain.Devices.Sound = &v1.SoundDevice{
				Model: supportedAudioDevice,
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.Sound"))
		})

		It("should reject volume with missing disk / file system", func() {
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					CloudInitNoCloud: &v1.CloudInitNoCloudSource{UserData: " "},
				},
			})

			causes := validateVirtualMachineInstanceSpecVolumeDisks(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.volumes[0].name"))
		})

		It("should reject multiple disks referencing same volume", func() {
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
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[1].name"))
		})

		It("should generate multiple causes", func() {
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
					LUN: &v1.LunTarget{
						Bus: v1.DiskBusSCSI,
					},
				},
			})

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			// missing volume and multiple targets set. should result in 2 causes
			Expect(causes).To(HaveLen(2))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[0].name"))
			Expect(causes[1].Field).To(Equal("fake.domain.devices.disks[0]"))
		})

		DescribeTable("should verify input device",
			func(input v1.Input, expectedErrors int, expectedErrorTypes []string, expectMessage string) {
				vmi.Spec.Domain.Devices.Inputs = append(vmi.Spec.Domain.Devices.Inputs, input)
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(causes).To(HaveLen(expectedErrors), fmt.Sprintf("Expect %d errors", expectedErrors))
				for i, errorType := range expectedErrorTypes {
					Expect(causes[i].Field).To(Equal(errorType), expectMessage)
				}
			},
			Entry("and accept input with virtio bus",
				v1.Input{
					Type: v1.InputTypeTablet,
					Name: "tablet0",
					Bus:  v1.InputBusVirtio,
				}, 0, []string{}, "Expect no errors"),
			Entry("and accept input with usb bus",
				v1.Input{
					Type: v1.InputTypeTablet,
					Name: "tablet0",
					Bus:  v1.InputBusUSB,
				}, 0, []string{}, "Expect no errors"),
			Entry("and accept input without bus",
				v1.Input{
					Type: v1.InputTypeTablet,
					Name: "tablet0",
				}, 0, []string{}, "Expect no errors"),
			Entry("and reject input with ps2 bus",
				v1.Input{
					Type: v1.InputTypeTablet,
					Name: "tablet0",
					Bus:  v1.InputBus("ps2"),
				}, 1, []string{"fake.domain.devices.inputs[0].bus"}, "Expect bus error"),
			Entry("and reject input with keyboard type and virtio bus",
				v1.Input{
					Type: v1.InputTypeKeyboard,
					Name: "tablet0",
					Bus:  v1.InputBusVirtio,
				}, 1, []string{"fake.domain.devices.inputs[0].type"}, "Expect type error"),
			Entry("and reject input with keyboard type and usb bus",
				v1.Input{
					Type: v1.InputTypeKeyboard,
					Name: "tablet0",
					Bus:  v1.InputBusUSB,
				}, 1, []string{"fake.domain.devices.inputs[0].type"}, "Expect type error"),
			Entry("and reject input with wrong type and wrong bus",
				v1.Input{
					Type: v1.InputTypeKeyboard,
					Name: "tablet0",
					Bus:  v1.InputBus("ps2"),
				}, 2, []string{"fake.domain.devices.inputs[0].bus", "fake.domain.devices.inputs[0].type"}, "Expect type error"),
		)

		It("should reject negative requests.cpu value", func() {
			vm := api.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("-200m"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.cpu"))
		})

		It("should reject negative limits.cpu size value", func() {
			vm := api.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("-3"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(causes).To(HaveLen(1))
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
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.cpu"))
		})

		It("should accept correct cpu size values even if vmRolloutStrategy is set to Stage", func() {
			kvConfig := kv.DeepCopy()
			kvConfig.Spec.Configuration.VMRolloutStrategy = pointer.P(v1.VMRolloutStrategyStage)
			testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvConfig)

			vm := api.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("1500m"),
			}
			vm.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("2"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should reject negative requests.memory size value", func() {
			vm := api.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("-64Mi"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})

		It("should reject small requests.memory size value", func() {
			vm := api.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64m"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})

		It("should reject negative limits.memory size value", func() {
			vm := api.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("-65Mi"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(causes).To(HaveLen(1))
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
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})

		It("should accept correct memory size values even if vmRolloutStrategy is set to Stage", func() {
			kvConfig := kv.DeepCopy()
			kvConfig.Spec.Configuration.VMRolloutStrategy = pointer.P(v1.VMRolloutStrategyStage)
			testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvConfig)

			vm := api.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vm.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("65Mi"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should reject incorrect hugepages size format", func() {
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{}}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "2ab"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.hugepages.size"))
		})

		It("should reject greater hugepages.size than requests.memory", func() {
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{}}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "1Gi"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})

		It("should allow smaller guest memory than requested memory even if vmRolloutStrategy is set to Stage", func() {
			kvConfig := kv.DeepCopy()
			kvConfig.Spec.Configuration.VMRolloutStrategy = pointer.P(v1.VMRolloutStrategyStage)
			testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvConfig)

			guestMemory := resource.MustParse("1Mi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should reject bigger guest memory than the memory limit if vmRolloutStrategy is set to Stage", func() {
			kvConfig := kv.DeepCopy()
			kvConfig.Spec.Configuration.VMRolloutStrategy = pointer.P(v1.VMRolloutStrategyStage)
			testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvConfig)

			guestMemory := resource.MustParse("128Mi")

			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.memory.guest"))
		})

		It("should allow bigger guest memory than the memory limit", func() {
			guestMemory := resource.MustParse("128Mi")

			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should allow guest memory which is between requests and limits", func() {
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
			guestMemory := resource.MustParse("100Mi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should reject not divisable by hugepages.size requests.memory", func() {
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("65Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{}}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "2Gi"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})

		It("should accept correct memory and hugepages size values", func() {
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{}}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "2Mi"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should reject incorrect memory and hugepages size values", func() {
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{}}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "10Mi"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
			Expect(causes[0].Message).To(Equal("fake.domain.resources.requests.memory '64Mi' " +
				"is not a multiple of the page size fake.domain.hugepages.size '10Mi'"))
		})

		It("should allow setting guest memory and hugepages", func() {
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
			Expect(causes).To(BeEmpty())
		})

		DescribeTable("should verify LUN is mapped to PVC volume",
			func(volume *v1.Volume, expectedErrors int) {
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "testdisk",
					DiskDevice: v1.DiskDevice{
						LUN: &v1.LunTarget{},
					},
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, *volume)

				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(causes).To(HaveLen(expectedErrors))
			},
			Entry("and reject non PVC sources",
				&v1.Volume{
					Name: "testdisk",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: testutils.NewFakeContainerDiskSource(),
					},
				}, 1),
			Entry("and accept PVC sources",
				&v1.Volume{
					Name: "testdisk",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{},
					},
				}, 0),
			Entry("and accept DataVolume sources",
				&v1.Volume{
					Name: "testdisk",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "testDV",
						},
					},
				}, 0),
		)

		It("should reject disks with the same boot order", func() {
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
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[1].bootOrder"))
			Expect(causes[0].Message).To(Equal("Boot order for " +
				"fake.domain.devices.disks[1].bootOrder already set for a different device."))
		})

		It("should raise a warning when Deprecated API is used", func() {
			const testsFGName = "test-deprecated"
			vmi.Spec.Architecture = runtime.GOARCH
			featuregate.RegisterFeatureGate(featuregate.FeatureGate{
				Name:        testsFGName,
				State:       featuregate.Deprecated,
				VmiSpecUsed: func(_ *v1.VirtualMachineInstanceSpec) bool { return true },
			})
			DeferCleanup(featuregate.UnregisterFeatureGate, testsFGName)
			enableFeatureGates(testsFGName)

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).NotTo(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
			Expect(resp.Result).To(BeNil())
			Expect(resp.Warnings).To(HaveLen(1))
		})

		It("should allow BlockMultiQueue with CPU settings", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.BlockMultiQueue = pointer.P(true)
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{}
			vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU] = resource.MustParse("5")

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should ignore CPU settings for explicitly rejected BlockMultiQueue", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.BlockMultiQueue = pointer.P(false)

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should allow valid ioThreadsPolicy", func() {
			vmi := api.NewMinimalVMI("testvm")
			var ioThreadPolicy v1.IOThreadsPolicy
			ioThreadPolicy = "auto"
			vmi.Spec.Domain.IOThreadsPolicy = &ioThreadPolicy
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should reject invalid ioThreadsPolicy", func() {
			vmi := api.NewMinimalVMI("testvm")
			var ioThreadPolicy v1.IOThreadsPolicy
			ioThreadPolicy = "bad"
			vmi.Spec.Domain.IOThreadsPolicy = &ioThreadPolicy
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(Equal(fmt.Sprintf("Invalid IOThreadsPolicy (%s)", ioThreadPolicy)))
		})

		It("should reject invalid ioThreadsPolicy to supplementalPool and invalid number of IOthreads", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.IOThreadsPolicy = pointer.P(v1.IOThreadsPolicySupplementalPool)
			vmi.Spec.Domain.IOThreads = &v1.DiskIOThreads{
				SupplementalPoolThreadCount: pointer.P(uint32(0)),
			}
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores:                 2,
				DedicatedCPUPlacement: true,
				IsolateEmulatorThread: true,
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("spec"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("spec.domain.ioThreads.count"))
			Expect(causes[0].Message).To(Equal("the number of iothreads needs to be set and positive for the dedicated policy"))
		})

		It("should reject invalid ioThreadsPolicy to supplementalPool and unsetted number of IOthreads", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.IOThreadsPolicy = pointer.P(v1.IOThreadsPolicySupplementalPool)
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("spec"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("spec.domain.ioThreads.count"))
			Expect(causes[0].Message).To(Equal("the number of iothreads needs to be set and positive for the dedicated policy"))
		})

		It("should reject multiple configurations of vGPU displays with ramfb", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{
					Name:       "gpu1",
					DeviceName: "vendor.com/gpu_name",
					VirtualGPUOptions: &v1.VGPUOptions{
						Display: &v1.VGPUDisplayOptions{
							Enabled: pointer.P(true),
						},
					},
				},
				{
					Name:       "gpu2",
					DeviceName: "vendor.com/gpu_name1",
					VirtualGPUOptions: &v1.VGPUOptions{
						Display: &v1.VGPUDisplayOptions{
							Enabled: pointer.P(true),
						},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.GPUs"))
		})

		It("should accept legacy GPU devices if PermittedHostDevices aren't set", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{
					Name:       "gpu1",
					DeviceName: "example.org/deadbeef",
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should accept permitted GPU devices", func() {
			kvConfig := kv.DeepCopy()
			kvConfig.Spec.Configuration.PermittedHostDevices = &v1.PermittedHostDevices{
				PciHostDevices: []v1.PciHostDevice{
					{
						PCIVendorSelector: "DEAD:BEEF",
						ResourceName:      "example.org/deadbeef",
					},
				},
			}
			testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvConfig)

			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{
					Name:       "gpu1",
					DeviceName: "example.org/deadbeef",
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		DescribeTable("virtiofs filesystems using", func(featureGate string, shouldAllow bool, vmiOption libvmi.Option) {
			if featureGate != "" {
				enableFeatureGates(featureGate)
			}

			vmi := libvmi.New(vmiOption)
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)

			if shouldAllow {
				Expect(causes).To(BeEmpty())
			} else {
				Expect(causes).To(HaveLen(1))
				Expect(causes[0].Field).To(Equal("fake.domain.devices.filesystems"))
			}

		},
			Entry("PVC should be rejected when feature gate is disabled", "", false, libvmi.WithFilesystemPVC("sharedtestdisk")),
			Entry("PVC should be accepted when feature gate is enabled", featuregate.VirtIOFSStorageVolumeGate, true, libvmi.WithFilesystemPVC("sharedtestdisk")),

			Entry("DV should be rejected when feature gate is disabled", "", false, libvmi.WithFilesystemDV("sharedtestdisk")),
			Entry("DV should be accepted when feature gate is enabled", featuregate.VirtIOFSStorageVolumeGate, true, libvmi.WithFilesystemDV("sharedtestdisk")),
			Entry("configmap should be rejected when the feature gate is disabled", "", false, libvmi.WithConfigMapFs("sharedconfigmap", "sharedconfigmap")),
			Entry("configmap should be accepted when the feature gate is enabled", featuregate.VirtIOFSConfigVolumesGate, true, libvmi.WithConfigMapFs("sharedconfigmap", "sharedconfigmap")),
			Entry("PVC should be accepted when the deprecated feature gate is enabled", featuregate.VirtIOFSGate, true, libvmi.WithFilesystemPVC("sharedtestdisk")),
			Entry("DV should be accepted when the deprecated feature gate is enabled", featuregate.VirtIOFSGate, true, libvmi.WithFilesystemDV("sharedtestdisk")),
			Entry("config map should be accepted when the deprecated feature gate is enabled", featuregate.VirtIOFSGate, true, libvmi.WithConfigMapFs("sharedconfigmap", "sharedconfigmap")),
		)

		It("should reject host devices when feature gate is disabled", func() {
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.HostDevices = []v1.HostDevice{
				{
					Name:       "hostdev1",
					DeviceName: "vendor.com/hostdev_name",
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.HostDevices"))
		})

		It("should accept host devices that are not permitted in the hostdev config", func() {
			kvConfig := kv.DeepCopy()
			kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{featuregate.HostDevicesGate}
			kvConfig.Spec.Configuration.PermittedHostDevices = &v1.PermittedHostDevices{
				PciHostDevices: []v1.PciHostDevice{
					{
						PCIVendorSelector: "DEAD:BEEF",
						ResourceName:      "example.org/deadbeef",
					},
				},
			}
			testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvConfig)
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.HostDevices = []v1.HostDevice{
				{
					Name:       "hostdev1",
					DeviceName: "example.org/deadbeef1",
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should accept permitted host devices", func() {
			kvConfig := kv.DeepCopy()
			kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{featuregate.HostDevicesGate}
			kvConfig.Spec.Configuration.PermittedHostDevices = &v1.PermittedHostDevices{
				PciHostDevices: []v1.PciHostDevice{
					{
						PCIVendorSelector: "DEAD:BEEF",
						ResourceName:      "example.org/deadbeef",
					},
				},
			}
			testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvConfig)
			vmi := api.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.HostDevices = []v1.HostDevice{
				{
					Name:       "hostdev1",
					DeviceName: "example.org/deadbeef",
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		DescribeTable("Should accept valid DNSPolicy and DNSConfig",
			func(dnsPolicy k8sv1.DNSPolicy, dnsConfig *k8sv1.PodDNSConfig) {

				vmi.Spec.DNSPolicy = dnsPolicy
				vmi.Spec.DNSConfig = dnsConfig
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(causes).To(BeEmpty())
			},
			Entry("with DNSPolicy ClusterFirstWithHostNet", k8sv1.DNSClusterFirstWithHostNet, &k8sv1.PodDNSConfig{}),
			Entry("with DNSPolicy ClusterFirst", k8sv1.DNSClusterFirst, &k8sv1.PodDNSConfig{}),
			Entry("with DNSPolicy Default", k8sv1.DNSDefault, &k8sv1.PodDNSConfig{}),
			Entry("with DNSPolicy None and one nameserver", k8sv1.DNSNone, &k8sv1.PodDNSConfig{Nameservers: []string{"1.2.3.4"}}),
			Entry("with DNSPolicy None max nameservers and max search domains", k8sv1.DNSNone, &k8sv1.PodDNSConfig{
				Nameservers: []string{"1.2.3.4", "5.6.7.8", "9.8.0.1"},
				Searches:    []string{"1", "2", "3", "4", "5", "6"},
			}),
			Entry("with DNSPolicy None max nameservers and max length search domain", k8sv1.DNSNone, &k8sv1.PodDNSConfig{
				Nameservers: []string{"1.2.3.4", "5.6.7.8", "9.8.0.1"},
				Searches:    []string{strings.Repeat("a", maxDNSSearchListChars/2), strings.Repeat("b", (maxDNSSearchListChars/2)-1)},
			}),
			Entry("with empty DNSPolicy", nil, nil),
		)

		DescribeTable("Should reject invalid DNSPolicy and DNSConfig",
			func(dnsPolicy k8sv1.DNSPolicy, dnsConfig *k8sv1.PodDNSConfig, causeCount int, causeMessage []string) {

				vmi.Spec.DNSPolicy = dnsPolicy
				vmi.Spec.DNSConfig = dnsConfig
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(causes).To(HaveLen(causeCount))
				for i := 0; i < causeCount; i++ {
					Expect(causes[i].Message).To(Equal(causeMessage[i]))
				}
			},
			Entry("with invalid DNSPolicy FakePolicy", k8sv1.DNSPolicy("FakePolicy"), &k8sv1.PodDNSConfig{}, 1,
				[]string{"DNSPolicy: FakePolicy is not supported, valid values: [ClusterFirstWithHostNet ClusterFirst Default None ]"}),
			Entry("with DNSPolicy None and no nameserver", k8sv1.DNSNone, &k8sv1.PodDNSConfig{}, 1,
				[]string{"must provide at least one DNS nameserver when `dnsPolicy` is None"}),
			Entry("with DNSPolicy None and too many nameservers", k8sv1.DNSNone, &k8sv1.PodDNSConfig{
				Nameservers: []string{"1.2.3.4", "5.6.7.8", "9.8.0.1", "2.3.4.5"},
			}, 1, []string{"must not have more than 3 nameservers: [1.2.3.4 5.6.7.8 9.8.0.1 2.3.4.5]"}),
			Entry("with DNSPolicy None and a non ip nameserver", k8sv1.DNSNone, &k8sv1.PodDNSConfig{
				Nameservers: []string{"1.2.3.c"},
			}, 1, []string{"must be valid IP address: 1.2.3.c"}),
			Entry("with DNSPolicy None and too many search domains", k8sv1.DNSNone, &k8sv1.PodDNSConfig{
				Nameservers: []string{"1.2.3.4"},
				Searches:    []string{"1", "2", "3", "4", "5", "6", "7"},
			}, 1, []string{"must not have more than 6 search paths"}),
			Entry("with DNSPolicy None and search domain exceeding max length", k8sv1.DNSNone, &k8sv1.PodDNSConfig{
				Nameservers: []string{"1.2.3.4"},
				Searches:    []string{strings.Repeat("a", maxDNSSearchListChars/2), strings.Repeat("b", maxDNSSearchListChars/2)},
			}, 1, []string{fmt.Sprintf("must not have more than %v characters (including spaces) in the search list", maxDNSSearchListChars)}),
			Entry("with DNSPolicy None and bad IsDNS1123Subdomain", k8sv1.DNSNone, &k8sv1.PodDNSConfig{
				Nameservers: []string{"1.2.3.4"},
				Searches:    []string{strings.Repeat("a", validation.DNS1123SubdomainMaxLength+1)},
			}, 1, []string{fmt.Sprintf("must be no more than %v characters", validation.DNS1123SubdomainMaxLength)}),
			Entry("with DNSPolicy None and bad options", k8sv1.DNSNone, &k8sv1.PodDNSConfig{
				Nameservers: []string{"1.2.3.4"},
				Options: []k8sv1.PodDNSConfigOption{
					{Value: &dnsConfigTestOption},
				},
			}, 1, []string{"Option.Name must not be empty"}),
			Entry("with DNSPolicy None and nil DNSConfig", k8sv1.DNSNone, interface{}(nil), 1,
				[]string{fmt.Sprintf("must provide `dnsConfig` when `dnsPolicy` is %s", k8sv1.DNSNone)}),
		)

		It("should accept valid start strategy", func() {
			strategy := v1.StartStrategyPaused
			vmi.Spec.StartStrategy = &strategy

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should allow no start strategy to be set", func() {
			vmi.Spec.StartStrategy = nil
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should reject invalid start strategy", func() {
			strategy := v1.StartStrategy("invalid")
			vmi.Spec.StartStrategy = &strategy

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake.startStrategy"))
			Expect(causes[0].Message).To(Equal("fake.startStrategy is set with an unrecognized option: invalid"))
		})

		It("should reject spec with paused start strategy and LivenessProbe", func() {
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
			Expect(causes).To(HaveLen(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake.startStrategy"))
			Expect(causes[0].Message).To(Equal("either fake.startStrategy or fake.livenessProbe should be provided.Pausing VMI with LivenessProbe is not supported"))
		})

		Context("with panic devices defined", func() {
			It("should fail when PanicDevices featuregate is disabled", func() {
				vmi := api.NewMinimalVMI("testvm")
				vmi.Spec.Domain.Devices.PanicDevices = []v1.PanicDevice{{Model: pointer.P(v1.Hyperv)}}
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(causes).To(HaveLen(1))
				Expect(causes[0].Field).To(Equal("fake.domain.devices.panicDevices"))
				Expect(causes[0].Message).To(Equal("Panic Devices feature gate is not enabled in kubevirt-config"))
			})

			It("should allow valid panic device model", func() {
				enableFeatureGates(featuregate.PanicDevicesGate)
				vmi := api.NewMinimalVMI("testvm")
				vmi.Spec.Domain.Devices.PanicDevices = []v1.PanicDevice{{Model: pointer.P(v1.Hyperv)}}
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(causes).To(BeEmpty())
			})

			It("should reject invalid panic device model", func() {
				enableFeatureGates(featuregate.PanicDevicesGate)
				vmi := api.NewMinimalVMI("testvm")
				panicDeviceModel := v1.PanicDeviceModel("bad")
				vmi.Spec.Domain.Devices.PanicDevices = []v1.PanicDevice{{Model: &panicDeviceModel}}
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(causes).To(HaveLen(1))
				Expect(causes[0].Field).To(Equal("fake.domain.devices.panicDevices[0].model"))
				Expect(causes[0].Message).To(Equal(fmt.Sprintf(invalidPanicDeviceModelErrFmt, panicDeviceModel)))
			})

			It("should reject panic devices on s390x architecture", func() {
				enableFeatureGates(featuregate.MultiArchitecture, featuregate.PanicDevicesGate)
				vmi := api.NewMinimalVMI("testvm")
				vmi.Spec.Architecture = "s390x"
				vmi.Spec.Domain.Devices.PanicDevices = []v1.PanicDevice{{Model: pointer.P(v1.Isa)}}
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(causes).To(HaveLen(1))
				Expect(causes[0].Field).To(Equal("fake.domain.devices.panicDevices"))
				Expect(causes[0].Message).To(Equal("custom panic devices are not supported on s390x architecture"))
			})
		})

		Context("with kernel boot defined", func() {

			createKernelBoot := func(kernelArgs, initrdPath, kernelPath, image string) *v1.KernelBoot {
				var kbContainer *v1.KernelBootContainer
				if image != "" || kernelPath != "" || initrdPath != "" {
					kbContainer = &v1.KernelBootContainer{
						Image:      image,
						KernelPath: kernelPath,
						InitrdPath: initrdPath,
					}
				}

				return &v1.KernelBoot{
					KernelArgs: kernelArgs,
					Container:  kbContainer,
				}
			}

			const (
				validKernelArgs   = "args"
				withoutKernelArgs = ""

				validImage   = "image"
				withoutImage = ""

				invalidInitrd = "initrd"
				validInitrd   = "/initrd"
				withoutInitrd = ""

				invalidKernel = "kernel"
				validKernel   = "/kernel"
				withoutKernel = ""
			)

			DescribeTable("", func(kernelBoot *v1.KernelBoot, shouldBeValid bool) {
				kernelBootField := k8sfield.NewPath("spec").Child("domain").Child("firmware").Child("kernelBoot")
				causes := validateKernelBoot(kernelBootField, kernelBoot)

				if shouldBeValid {
					Expect(causes).To(BeEmpty())
				} else {
					Expect(causes).ToNot(BeEmpty())
				}
			},
				Entry("without kernel args and null container - should approve",
					createKernelBoot(withoutKernelArgs, withoutInitrd, withoutKernel, withoutImage), true),
				Entry("with kernel args and null container - should reject",
					createKernelBoot(validKernelArgs, withoutInitrd, withoutKernel, withoutImage), false),
				Entry("without kernel args, with container that has image & kernel & initrd defined - should approve",
					createKernelBoot(withoutKernelArgs, validInitrd, validKernel, validImage), true),
				Entry("with kernel args, with container that has image & kernel & initrd defined - should approve",
					createKernelBoot(validKernelArgs, validInitrd, validKernel, validImage), true),
				Entry("with kernel args, with container that has image & kernel defined - should approve",
					createKernelBoot(validKernelArgs, withoutInitrd, validKernel, validImage), true),
				Entry("with kernel args, with container that has image & initrd defined - should approve",
					createKernelBoot(validKernelArgs, validInitrd, withoutKernel, validImage), true),
				Entry("with kernel args, with container that has only image defined - should reject",
					createKernelBoot(validKernelArgs, withoutInitrd, withoutKernel, validImage), false),
				Entry("with invalid kernel path - should reject",
					createKernelBoot(validKernelArgs, validInitrd, invalidKernel, validImage), false),
				Entry("with invalid initrd path - should reject",
					createKernelBoot(validKernelArgs, invalidInitrd, validKernel, validImage), false),
				Entry("with kernel args, with container that has initrd and kernel defined but without image - should reject",
					createKernelBoot(validKernelArgs, validInitrd, validKernel, withoutImage), false),
			)
		})

		It("should detect invalid containerDisk paths", func() {
			spec := &v1.VirtualMachineInstanceSpec{}
			disk := v1.Disk{
				Name:   "testdisk",
				Serial: "SN-1_a",
			}
			spec.Domain.Devices.Disks = []v1.Disk{disk}
			volume := v1.Volume{
				Name: "testdisk",
				VolumeSource: v1.VolumeSource{
					ContainerDisk: testutils.NewFakeContainerDiskSource(),
				},
			}
			volume.ContainerDisk.Path = "invalid"

			spec.Volumes = []v1.Volume{volume}
			spec.Domain.Devices.Disks = []v1.Disk{disk}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), spec, config)
			Expect(causes).To(HaveLen(1))
		})
	})

	Context("with cpu pinning", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = newBaseVmi(libvmi.WithDedicatedCPUPlacement())
		})

		It("should reject NUMA passthrough without DedicatedCPUPlacement", func() {
			vmi.Spec.Domain.CPU.NUMA = &v1.NUMA{GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{}}
			vmi.Spec.Domain.CPU.DedicatedCPUPlacement = false
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{PageSize: "2Mi"}}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.cpu.numa.guestMappingPassthrough"))
		})

		DescribeTable("should reject NUMA passthrough without hugepages", func(memory *v1.Memory) {
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
			Entry("with no memory element", nil),
			Entry("with no hugepages element", &v1.Memory{Hugepages: nil}),
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

		It("should reject vmi with threads > 1 for arm64 arch", func() {
			enableFeatureGates(featuregate.MultiArchitecture)
			vmi.Spec.Domain.CPU.Threads = 2
			vmi.Spec.Architecture = "arm64"
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(ContainElement(
				metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Field:   "fake.architecture",
					Message: "threads must not be greater than 1 at fake.domain.cpu.threads (got 2) when fake.architecture is arm64",
				},
			))
		})

		It("should accept vmi with threads == 1 for arm64 arch", func() {
			enableFeatureGates(featuregate.MultiArchitecture)
			vmi.Spec.Domain.CPU.Threads = 1
			vmi.Spec.Architecture = "arm64"
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should accept vmi with threads > 1 for amd64 arch", func() {
			enableFeatureGates(featuregate.MultiArchitecture)
			vmi.Spec.Domain.CPU.Threads = 2
			vmi.Spec.Architecture = "amd64"
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should reject vmi with threads > 1 if arch is not specified and default arch is arm64", func() {
			updateDefaultArchitecture("arm64")
			Expect(config.GetDefaultArchitecture()).To(Equal("arm64"))
			vmi.Spec.Architecture = ""
			vmi.Spec.Domain.CPU.Threads = 2
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.architecture"))
			Expect(causes[0].Message).To(Equal("threads must not be greater than 1 at fake.domain.cpu.threads (got 2) when fake.architecture is arm64"))
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
			Expect(causes).To(ContainElement(
				metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Field:   "fake.domain.cpu.dedicatedCpuPlacement",
					Message: "Not more than two threads must be provided at fake.domain.cpu.threads (got 3) when DedicatedCPUPlacement is true",
				},
			))
		})

		It("should reject specs without cpu reqirements", func() {
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.cpu.dedicatedCpuPlacement"))
		})

		It("should reject specs with IsolateEmulatorThread without DedicatedCPUPlacement set", func() {
			vmi.Spec.Domain.CPU = &v1.CPU{
				DedicatedCPUPlacement: false,
				IsolateEmulatorThread: true,
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.cpu.isolateEmulatorThread"))
		})

		It("should reject specs without inconsistent cpu reqirements", func() {
			vmi.Spec.Domain.CPU.Cores = 4
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("2"),
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.cpu.dedicatedCpuPlacement"))
		})

		It("should reject specs with non-integer cpu limits values", func() {
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("800m"),
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.limits.cpu"))
		})

		It("should reject specs with non-integer cpu requests values", func() {
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceCPU:    resource.MustParse("800m"),
				k8sv1.ResourceMemory: resource.MustParse("8Mi"),
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
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
			Expect(causes).To(HaveLen(1))
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
			Expect(causes).To(HaveLen(1))
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
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})
	})

	Context("with AccessCredentials", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
		})

		It("should accept a valid ssh access credential with configdrive propagation", func() {
			vmi := newBaseVmi(libvmi.WithCloudInitConfigDrive(libvmici.WithConfigDriveUserData(" ")))

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
			Expect(causes).To(BeEmpty())
		})

		It("should accept a valid ssh access credential with qemu agent propagation", func() {
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
			Expect(causes).To(BeEmpty())
		})

		It("should accept a valid user password access credential with qemu agent propagation", func() {
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
			Expect(causes).To(BeEmpty())
		})

		It("should reject a noCloud ssh access credential when no noCloud volume exists", func() {
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
							NoCloud: &v1.NoCloudSSHPublicKeyAccessCredentialPropagation{},
						},
					},
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("requires a noCloud volume to exist"))
		})

		It("should reject a configDrive ssh access credential when no configDrive volume exists", func() {
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
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("requires a configDrive volume to exist"))
		})

		It("should reject a ssh access credential without a source", func() {
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
			Expect(causes).To(HaveLen(1))
		})

		It("should reject a ssh access credential with qemu agent propagation with no authorized key files listed", func() {
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
			Expect(causes).To(HaveLen(1))
		})

		It("should reject a userpassword access credential without a source", func() {
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

			Expect(causes).To(HaveLen(1))
		})

		It("should reject a ssh access credential without a propagationMethod", func() {
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
			Expect(causes).To(HaveLen(1))
		})

		It("should reject a userpassword credential without a propagationMethod", func() {
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
			Expect(causes).To(HaveLen(1))
		})
	})

	Context("with CPU features", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
		})

		It("should accept valid CPU feature policies", func() {
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
				Expect(causes).To(BeEmpty())
			}
		})

		It("should reject invalid CPU feature policy", func() {
			vmi.Spec.Domain.CPU = &v1.CPU{
				Features: []v1.CPUFeature{
					{
						Name:   "lahf_lm",
						Policy: "invalid_policy",
					},
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
		})
	})

	Context("with downwardmetrics virtio serial", func() {
		var vmi *v1.VirtualMachineInstance
		validate := func() []metav1.StatusCause {
			return validateDownwardMetrics(k8sfield.NewPath("fake"), &vmi.Spec, config)
		}

		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.DownwardMetrics = &v1.DownwardMetrics{}
		})

		It("should accept a single virtio serial", func() {
			enableFeatureGates(featuregate.DownwardMetricsFeatureGate)
			causes := validate()
			Expect(causes).To(BeEmpty())
		})

		It("should reject if feature gate is not enabled", func() {
			causes := validate()
			Expect(causes).To(HaveLen(1))
			Expect(causes).To(ContainElement(metav1.StatusCause{Type: metav1.CauseTypeFieldValueInvalid,
				Field:   "fake.domain.devices.downwardMetrics",
				Message: "downwardMetrics virtio serial is not allowed: DownwardMetrics feature gate is not enabled"}))
		})
	})

	Context("with volume", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
		})

		It("should accept a single downwardmetrics volume", func() {
			enableFeatureGates(featuregate.DownwardMetricsFeatureGate)

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
			enableFeatureGates(featuregate.DownwardMetricsFeatureGate)

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
			enableFeatureGates(featuregate.HostDiskGate)

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

		It("should accept a single memoryDump volume without a matching disk", func() {
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testMemoryDump",
				VolumeSource: v1.VolumeSource{
					MemoryDump: testutils.NewFakeMemoryDumpSource("testMemoryDump"),
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(BeEmpty())
		})

		It("should reject memoryDump volumes if more than one exist", func() {
			vmi.Spec.Volumes = append(vmi.Spec.Volumes,
				v1.Volume{
					Name: "testMemoryDump",
					VolumeSource: v1.VolumeSource{
						MemoryDump: testutils.NewFakeMemoryDumpSource("testMemoryDump"),
					},
				},
				v1.Volume{
					Name: "testMemoryDump2",
					VolumeSource: v1.VolumeSource{
						MemoryDump: testutils.NewFakeMemoryDumpSource("testMemoryDump2"),
					},
				},
			)
			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("fake must have max one memory dump volume set"))
		})

	})

	Context("with bootloader", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
		})

		It("should accept empty bootloader setting", func() {
			vmi.Spec.Subdomain = "testsubdomain"

			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: nil,
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should accept BIOS", func() {
			vmi.Spec.Subdomain = "testsubdomain"

			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					BIOS: &v1.BIOS{},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should accept EFI with SMM", func() {
			vmi.Spec.Subdomain = "testsubdomain"

			vmi.Spec.Domain.Features = &v1.Features{
				SMM: &v1.FeatureState{
					Enabled: pointer.P(true),
				},
			}
			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					EFI: &v1.EFI{},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should not accept EFI without SMM", func() {
			vmi.Spec.Subdomain = "testsubdomain"

			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					EFI: &v1.EFI{},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
		})

		It("should accept EFI without secureBoot and without SMM", func() {
			vmi.Spec.Subdomain = "testsubdomain"

			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					EFI: &v1.EFI{
						SecureBoot: pointer.P(false),
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should not accept BIOS and EFI together", func() {
			vmi.Spec.Subdomain = "testsubdomain"

			vmi.Spec.Domain.Features = &v1.Features{
				SMM: &v1.FeatureState{
					Enabled: pointer.P(true),
				},
			}
			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					EFI:  &v1.EFI{},
					BIOS: &v1.BIOS{},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
		})

	})

	Context("with verification for Arm64", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
		})

		It("should reject BIOS bootloader", func() {
			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					BIOS: &v1.BIOS{},
				},
			}

			causes := webhooks.ValidateVirtualMachineInstanceArm64Setting(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.firmware.bootloader.bios"))
			Expect(causes[0].Message).To(Equal("Arm64 does not support bios boot, please change to uefi boot"))
		})

		// When setting UEFI default bootloader, UEFI secure bootloader would be applied which is not supported on Arm64
		It("should reject UEFI default bootloader", func() {
			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					EFI: &v1.EFI{},
				},
			}

			causes := webhooks.ValidateVirtualMachineInstanceArm64Setting(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.firmware.bootloader.efi.secureboot"))
			Expect(causes[0].Message).To(Equal("UEFI secure boot is currently not supported on aarch64 Arch"))
		})

		It("should reject UEFI secure bootloader", func() {
			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					EFI: &v1.EFI{
						SecureBoot: pointer.P(true),
					},
				},
			}

			causes := webhooks.ValidateVirtualMachineInstanceArm64Setting(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.firmware.bootloader.efi.secureboot"))
			Expect(causes[0].Message).To(Equal("UEFI secure boot is currently not supported on aarch64 Arch"))
		})

		DescribeTable("should validate ACPI", func(acpi *v1.ACPI, volumes []v1.Volume, expectedLen int, expectedMessage string) {
			vmi.Spec.Domain.Firmware = &v1.Firmware{ACPI: acpi}
			vmi.Spec.Volumes = volumes
			causes := validateFirmwareACPI(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(causes).To(HaveLen(expectedLen))
			if expectedLen != 0 {
				Expect(causes[0].Message).To(ContainSubstring(expectedMessage))
			}
		},
			Entry("Not set is ok", nil, []v1.Volume{}, 0, ""),
			Entry("ACPI SLIC with Volume match is ok",
				&v1.ACPI{SlicNameRef: "slic"},
				[]v1.Volume{
					{
						Name: "slic",
						VolumeSource: v1.VolumeSource{
							Secret: &v1.SecretVolumeSource{SecretName: "secret-slic"},
						},
					},
				}, 0, ""),
			Entry("ACPI MSDM with Volume match is ok",
				&v1.ACPI{SlicNameRef: "msdm"},
				[]v1.Volume{
					{
						Name: "msdm",
						VolumeSource: v1.VolumeSource{
							Secret: &v1.SecretVolumeSource{SecretName: "secret-msdm"},
						},
					},
				}, 0, ""),
			Entry("ACPI SLIC without Volume match should fail",
				&v1.ACPI{SlicNameRef: "slic"},
				[]v1.Volume{}, 1, "does not have a matching Volume"),
			Entry("ACPI MSDM without Volume match should fail",
				&v1.ACPI{MsdmNameRef: "msdm"},
				[]v1.Volume{}, 1, "does not have a matching Volume"),
			Entry("ACPI SLIC with wrong Volume type should fail",
				&v1.ACPI{SlicNameRef: "slic"},
				[]v1.Volume{
					{
						Name: "slic",
						VolumeSource: v1.VolumeSource{
							ConfigMap: &v1.ConfigMapVolumeSource{
								LocalObjectReference: k8sv1.LocalObjectReference{Name: "configmap-slic"},
							},
						},
					},
				}, 1, "Volume of unsupported type"),
			Entry("ACPI MSDM with wrong Volume type should fail",
				&v1.ACPI{MsdmNameRef: "msdm"},
				[]v1.Volume{
					{
						Name: "msdm",
						VolumeSource: v1.VolumeSource{
							ConfigMap: &v1.ConfigMapVolumeSource{
								LocalObjectReference: k8sv1.LocalObjectReference{Name: "configmap-msdm"},
							},
						},
					},
				}, 1, "Volume of unsupported type"),
		)

		DescribeTable("validating cpu model with", func(model string, expectedLen int) {
			vmi.Spec.Domain.CPU = &v1.CPU{Model: model}

			causes := webhooks.ValidateVirtualMachineInstanceArm64Setting(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(causes).To(HaveLen(expectedLen))
			if expectedLen != 0 {
				Expect(causes[0].Field).To(Equal("fake.domain.cpu.model"))
				Expect(causes[0].Message).To(Equal(fmt.Sprintf("currently, %v is the only model supported on Arm64", v1.CPUModeHostPassthrough)))
			}
		},
			Entry("host-model should get rejected with arm64", "host-model", 1),
			Entry("named model should get rejected with arm64", "Cooperlake", 1),
			Entry("host-passthrough should be accepted with arm64", "host-passthrough", 0),
			Entry("empty model should be accepted with arm64", "", 0),
		)

		It("should reject setting sound device", func() {
			vmi.Spec.Domain.Devices.Sound = &v1.SoundDevice{
				Name:  "test-audio-device",
				Model: "ich9",
			}
			causes := webhooks.ValidateVirtualMachineInstanceArm64Setting(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.sound"))
			Expect(causes[0].Message).To(Equal("Arm64 not support sound device"))
		})
	})

	Context("with realtime", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.CPU = &v1.CPU{Realtime: &v1.Realtime{}, Cores: 4}
		})

		It("should reject the realtime knob without DedicatedCPUPlacement", func() {
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{PageSize: "2Mi"}}
			vmi.Spec.Domain.CPU.NUMA = &v1.NUMA{GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{}}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(ContainElement(metav1.StatusCause{Type: metav1.CauseTypeFieldValueRequired, Field: "fake.domain.cpu.dedicatedCpuPlacement", Message: "fake.domain.cpu.dedicatedCpuPlacement must be set to true when fake.domain.cpu.realtime is used"}))
		})

		It("should reject the realtime knob when NUMA Guest Mapping Passthrough is not defined", func() {
			vmi.Spec.Domain.CPU.DedicatedCPUPlacement = true
			vmi.Spec.Domain.CPU.NUMA = &v1.NUMA{}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(ContainElement(metav1.StatusCause{Type: metav1.CauseTypeFieldValueRequired, Field: "fake.domain.cpu.numa.guestMappingPassthrough", Message: "fake.domain.cpu.numa.guestMappingPassthrough must be defined when fake.domain.cpu.realtime is used"}))
		})

		It("should reject the realtime knob when NUMA is nil", func() {
			vmi.Spec.Domain.CPU.DedicatedCPUPlacement = true
			vmi.Spec.Domain.CPU.NUMA = nil
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(ContainElement(metav1.StatusCause{Type: metav1.CauseTypeFieldValueRequired, Field: "fake.domain.cpu.numa.guestMappingPassthrough", Message: "fake.domain.cpu.numa.guestMappingPassthrough must be defined when fake.domain.cpu.realtime is used"}))
		})
	})

	Context("with AMD SEV LaunchSecurity", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{
				SEV: &v1.SEV{},
			}
			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					EFI: &v1.EFI{
						SecureBoot: pointer.P(false),
					},
				},
			}
			enableFeatureGates(featuregate.WorkloadEncryptionSEV)
		})

		It("should accept when the feature gate is enabled and OVMF is configured", func() {
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should reject when the feature gate is disabled", func() {
			disableFeatureGates()
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring(fmt.Sprintf("%s feature gate is not enabled", featuregate.WorkloadEncryptionSEV)))
		})

		It("should reject when UEFI is not configured", func() {
			vmi.Spec.Domain.Firmware.Bootloader.EFI = nil
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("SEV requires OVMF"))
			vmi.Spec.Domain.Firmware.Bootloader = nil
			causes = ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("SEV requires OVMF"))
			vmi.Spec.Domain.Firmware = nil
			causes = ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("SEV requires OVMF"))
		})

		It("should reject when SecureBoot is enabled", func() {
			vmi.Spec.Domain.Features = &v1.Features{
				SMM: &v1.FeatureState{
					Enabled: pointer.P(true),
				},
			}
			vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot = pointer.P(true)
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("SEV does not work along with SecureBoot"))
			vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot = nil
			causes = ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("SEV does not work along with SecureBoot"))
		})

		It("should reject when there are bootable NICs", func() {
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			bootOrder := uint(1)
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
				{Name: vmi.Spec.Networks[0].Name, BootOrder: &bootOrder},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(len(vmi.Spec.Domain.Devices.Interfaces)))
		})

		It("should accept SEV attestation with start strategy 'Paused'", func() {
			startStrategy := v1.StartStrategyPaused
			vmi.Spec.Domain.LaunchSecurity.SEV.Attestation = &v1.SEVAttestation{}
			vmi.Spec.StartStrategy = &startStrategy
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should reject SEV attestation without start strategy 'Paused'", func() {
			vmi.Spec.Domain.LaunchSecurity.SEV.Attestation = &v1.SEVAttestation{}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(ContainSubstring("launchSecurity"))
		})
	})

	Context("with Secure Execution LaunchSecurity", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{}
			vmi.Spec.Architecture = "s390x"
		})

		It("should accept when the feature gate is enabled", func() {
			enableFeatureGates(featuregate.MultiArchitecture, featuregate.SecureExecution)
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty())
		})

		It("should reject when the feature gate is disabled", func() {
			enableFeatureGates(featuregate.MultiArchitecture)

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring(fmt.Sprintf("%s feature gate is not enabled", featuregate.SecureExecution)))
		})
	})

	Context("with vsocks defined", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
			enableFeatureGates(featuregate.VSOCKGate)
		})

		Context("feature gate enabled", func() {
			It("should accept vmi with no vsocks defined", func() {
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(causes).To(BeEmpty())
			})

			It("should accept vmi with vsocks defined", func() {
				vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.P(true)
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(causes).To(BeEmpty())
			})
		})

		Context("feature gate disabled", func() {
			It("should reject when the feature gate is disabled", func() {
				disableFeatureGates()
				vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.P(true)
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(causes).To(HaveLen(1))
				Expect(causes[0].Message).To(ContainSubstring(fmt.Sprintf("%s feature gate is not enabled", featuregate.VSOCKGate)))
			})
		})
	})

	Context("with affinity checks", func() {
		var vmi *v1.VirtualMachineInstance
		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
			vmi.Spec.Architecture = runtime.GOARCH
			vmi.Spec.Affinity = &k8sv1.Affinity{}
		})

		It("Allow to create when spec.affinity set to nil", func() {
			vmi.Spec.Affinity = nil

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("(PodAffinity) Allowed PreferredDuringSchedulingIgnoredDuringExecution and RequiredDuringSchedulingIgnoredDuringExecution both are not set", func() {
			vmi.Spec.Affinity.PodAffinity = &k8sv1.PodAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: nil,
				RequiredDuringSchedulingIgnoredDuringExecution:  nil,
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("(PodAffinity) Should reject when validation failed due to TopologyKey is set to empty", func() {
			vmi.Spec.Affinity.PodAffinity = &k8sv1.PodAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []k8sv1.PodAffinityTerm{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "key1",
									Operator: metav1.LabelSelectorOpExists,
								},
							},
						},
						TopologyKey: "",
					},
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(3))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey"))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey: Required value: can not be empty"))
			Expect(resp.Result.Details.Causes[1].Field).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey"))
			Expect(resp.Result.Details.Causes[1].Message).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey: Invalid value: \"\": name part must be non-empty"))
			Expect(resp.Result.Details.Causes[2].Field).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey"))
			Expect(resp.Result.Details.Causes[2].Message).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey: Invalid value: \"\": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')"))
		})

		It("(PodAffinity) Should reject when validation failed due to first element of Values slice is set to empty as well as TopologyKey", func() {
			vmi.Spec.Affinity.PodAffinity = &k8sv1.PodAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []k8sv1.PodAffinityTerm{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "key1",
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{""},
								},
							},
						},
						TopologyKey: "",
					},
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(3))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey"))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey: Required value: can not be empty"))
			Expect(resp.Result.Details.Causes[1].Field).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey"))
			Expect(resp.Result.Details.Causes[1].Message).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey: Invalid value: \"\": name part must be non-empty"))
			Expect(resp.Result.Details.Causes[2].Field).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey"))
			Expect(resp.Result.Details.Causes[2].Message).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey: Invalid value: \"\": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')"))
		})

		It("(PodAffinity) Should reject when validation failed due to values of MatchExpressions is set to empty and TopologyKey value is not valid", func() {
			vmi.Spec.Affinity.PodAffinity = &k8sv1.PodAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []k8sv1.PodAffinityTerm{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "key1",
									Operator: metav1.LabelSelectorOpIn,
									Values:   nil,
								},
							},
						},
						TopologyKey: "hostname=host1",
					},
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(2))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].labelSelector.matchExpressions[0].values"))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].labelSelector.matchExpressions[0].values: Required value: must be specified when `operator` is 'In' or 'NotIn'"))
			Expect(resp.Result.Details.Causes[1].Field).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey"))
			Expect(resp.Result.Details.Causes[1].Message).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey: Invalid value: \"hostname=host1\": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')"))
		})

		It("(PodAffinity) Should reject when validation failed due to values of MatchExpressions and TopologyKey are both set to empty", func() {
			vmi.Spec.Affinity.PodAffinity = &k8sv1.PodAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []k8sv1.PodAffinityTerm{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "key1",
									Operator: metav1.LabelSelectorOpIn,
									Values:   nil,
								},
							},
						},
						TopologyKey: "",
					},
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(4))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].labelSelector.matchExpressions[0].values"))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].labelSelector.matchExpressions[0].values: Required value: must be specified when `operator` is 'In' or 'NotIn'"))
			Expect(resp.Result.Details.Causes[1].Field).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey"))
			Expect(resp.Result.Details.Causes[1].Message).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey: Required value: can not be empty"))
			Expect(resp.Result.Details.Causes[2].Field).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey"))
			Expect(resp.Result.Details.Causes[2].Message).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey: Invalid value: \"\": name part must be non-empty"))
			Expect(resp.Result.Details.Causes[3].Field).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey"))
			Expect(resp.Result.Details.Causes[3].Message).To(Equal("spec.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey: Invalid value: \"\": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')"))
		})

		It("(PodAffinity) Should reject when affinity.PodAffinity validation failed due to value of weight is not valid", func() {
			vmi.Spec.Affinity.PodAffinity = &k8sv1.PodAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []k8sv1.WeightedPodAffinityTerm{
					{
						Weight: 255,
						PodAffinityTerm: k8sv1.PodAffinityTerm{
							LabelSelector: nil,
							TopologyKey:   "test",
						},
					},
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].weight"))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("spec.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].weight: Invalid value: 255: must be in the range 1-100"))
		})

		It("(PodAntiAffinity) Allowed both RequiredDuringSchedulingIgnoredDuringExecution and PreferredDuringSchedulingIgnoredDuringExecution are set to empty", func() {
			vmi.Spec.Affinity.PodAntiAffinity = &k8sv1.PodAntiAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: nil,
				RequiredDuringSchedulingIgnoredDuringExecution:  nil,
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("(PodAntiAffinity) Should reject when scheduler validation failed due to TopologyKey is empty", func() {
			vmi.Spec.Affinity.PodAntiAffinity = &k8sv1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []k8sv1.PodAffinityTerm{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "key1",
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{""},
								},
							},
						},
						TopologyKey: "",
					},
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(3))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey"))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("spec.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey: Required value: can not be empty"))
			Expect(resp.Result.Details.Causes[1].Field).To(Equal("spec.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey"))
			Expect(resp.Result.Details.Causes[1].Message).To(Equal("spec.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey: Invalid value: \"\": name part must be non-empty"))
			Expect(resp.Result.Details.Causes[2].Field).To(Equal("spec.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey"))
			Expect(resp.Result.Details.Causes[2].Message).To(Equal("spec.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey: Invalid value: \"\": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')"))
		})

		It("(PodAntiAffinity) Should be ok with only PreferredDuringSchedulingIgnoredDuringExecution set with proper values", func() {
			vmi.Spec.Affinity.PodAntiAffinity = &k8sv1.PodAntiAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []k8sv1.WeightedPodAffinityTerm{
					{
						Weight: 86,
						PodAffinityTerm: k8sv1.PodAffinityTerm{
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      "key1",
										Operator: metav1.LabelSelectorOpIn,
										Values:   []string{"a"},
									},
								},
							},
							TopologyKey: "test",
						},
					},
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("(PodAntiAffinity) Should reject when validation failed due to values of MatchExpressions is set to empty and TopologyKey is not valid", func() {
			vmi.Spec.Affinity.PodAntiAffinity = &k8sv1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []k8sv1.PodAffinityTerm{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "key1",
									Operator: metav1.LabelSelectorOpIn,
									Values:   nil,
								},
							},
						},
						TopologyKey: "hostname=host1",
					},
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(2))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].labelSelector.matchExpressions[0].values"))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("spec.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].labelSelector.matchExpressions[0].values: Required value: must be specified when `operator` is 'In' or 'NotIn'"))
			Expect(resp.Result.Details.Causes[1].Field).To(Equal("spec.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey"))
			Expect(resp.Result.Details.Causes[1].Message).To(Equal("spec.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey: Invalid value: \"hostname=host1\": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')"))
		})

		It("(PodAntiAffinity) Should reject when scheduler validation failed due to values of MatchExpressions and TopologyKey are both set to empty", func() {
			vmi.Spec.Affinity.PodAntiAffinity = &k8sv1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []k8sv1.PodAffinityTerm{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "key1",
									Operator: metav1.LabelSelectorOpIn,
									Values:   nil,
								},
							},
						},
						TopologyKey: "",
					},
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(4))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].labelSelector.matchExpressions[0].values"))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("spec.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].labelSelector.matchExpressions[0].values: Required value: must be specified when `operator` is 'In' or 'NotIn'"))
			Expect(resp.Result.Details.Causes[1].Field).To(Equal("spec.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey"))
			Expect(resp.Result.Details.Causes[1].Message).To(Equal("spec.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey: Required value: can not be empty"))
			Expect(resp.Result.Details.Causes[2].Field).To(Equal("spec.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey"))
			Expect(resp.Result.Details.Causes[2].Message).To(Equal("spec.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey: Invalid value: \"\": name part must be non-empty"))
			Expect(resp.Result.Details.Causes[3].Field).To(Equal("spec.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey"))
			Expect(resp.Result.Details.Causes[3].Message).To(Equal("spec.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey: Invalid value: \"\": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')"))
		})

		It("(NodeAffinity) Allowed both RequiredDuringSchedulingIgnoredDuringExecution and PreferredDuringSchedulingIgnoredDuringExecution are set to empty", func() {
			vmi.Spec.Affinity.NodeAffinity = &k8sv1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution:  nil,
				PreferredDuringSchedulingIgnoredDuringExecution: nil,
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("(NodeAffinity) Should reject when scheduler validation failed due to NodeSelectorTerms set to empty", func() {
			vmi.Spec.Affinity.NodeAffinity = &k8sv1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
					NodeSelectorTerms: nil,
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			// webhookutils.ValidateSchema will take over so result will be only a message
			Expect(resp.Result.Details.Causes[0].Field).To(Equal(""))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms in body must be of type array: \"null\""))
		})

		It("(NodeAffinity) Allowed both MatchExpressions and MatchFields are set to empty", func() {
			vmi.Spec.Affinity.NodeAffinity = &k8sv1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
					NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
						{
							MatchExpressions: nil,
							MatchFields:      nil,
						},
					},
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("(NodeAffinity) Should be ok with only MatchExpressions set", func() {
			vmi.Spec.Affinity.NodeAffinity = &k8sv1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
					NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sv1.NodeSelectorRequirement{
								{
									Key:      "key1",
									Operator: k8sv1.NodeSelectorOpExists,
									Values:   nil,
								},
							},
						},
					},
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("(NodeAffinity) Should reject when scheduler validation failed due to NodeSelectorTerms value of key is not valid", func() {
			vmi.Spec.Affinity.NodeAffinity = &k8sv1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
					NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
						{
							MatchFields: []k8sv1.NodeSelectorRequirement{
								{
									Key:      "key",
									Operator: k8sv1.NodeSelectorOpIn,
									Values:   []string{"value1"},
								},
							},
						},
					},
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchFields[0].key"))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("spec.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchFields[0].key: Invalid value: \"key\": not a valid field selector key"))
		})

		It("(NodeAffinity) Should reject when scheduler validation failed due no element in Values slice", func() {
			vmi.Spec.Affinity.NodeAffinity = &k8sv1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
					NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
						{
							MatchFields: []k8sv1.NodeSelectorRequirement{
								{
									Key:      "metadata.name",
									Operator: k8sv1.NodeSelectorOpIn,
									Values:   []string{""},
								},
							},
						},
					},
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchFields[0].values[0]"))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("spec.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchFields[0].values[0]: Invalid value: \"\": a lowercase RFC 1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character (e.g. 'example.com', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')"))
		})

		It("(NodeAffinity) Should be ok with only PreferredDuringSchedulingIgnoredDuringExecution set with proper values", func() {
			vmi.Spec.Affinity.NodeAffinity = &k8sv1.NodeAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []k8sv1.PreferredSchedulingTerm{
					{
						Weight: 20,
						Preference: k8sv1.NodeSelectorTerm{
							MatchExpressions: []k8sv1.NodeSelectorRequirement{
								{
									Key:      "key1",
									Operator: k8sv1.NodeSelectorOpExists,
									Values:   nil,
								},
							},
						},
					},
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

	})

	Context("with topologySpreadConstraints checks", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
			vmi.Spec.Architecture = runtime.GOARCH
		})

		It("Allow to create when spec.topologySpreadConstraints set to nil", func() {
			vmi.Spec.TopologySpreadConstraints = nil
			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("Allowed LabelSelector is not set", func() {
			vmi.Spec.TopologySpreadConstraints = []k8sv1.TopologySpreadConstraint{
				{
					MaxSkew:           1,
					TopologyKey:       k8sv1.LabelHostname,
					WhenUnsatisfiable: k8sv1.DoNotSchedule,
					LabelSelector:     nil,
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("Allowed with valid LabelSelector is set", func() {
			vmi.Spec.TopologySpreadConstraints = []k8sv1.TopologySpreadConstraint{
				{
					MaxSkew:           1,
					TopologyKey:       k8sv1.LabelHostname,
					WhenUnsatisfiable: k8sv1.DoNotSchedule,
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "kubernetes.io/zone",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{"zone1"},
							},
						},
					},
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("Should reject when TopologyKey is empty", func() {
			vmi.Spec.TopologySpreadConstraints = []k8sv1.TopologySpreadConstraint{
				{
					MaxSkew:           1,
					TopologyKey:       "",
					WhenUnsatisfiable: k8sv1.DoNotSchedule,
					LabelSelector:     nil,
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.topologySpreadConstraints[0].topologyKey"))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("spec.topologySpreadConstraints[0].topologyKey: Required value: can not be empty"))
		})

		It("Should reject when TopologyKey is not valid", func() {
			vmi.Spec.TopologySpreadConstraints = []k8sv1.TopologySpreadConstraint{
				{
					MaxSkew:           1,
					TopologyKey:       "hostname=host1",
					WhenUnsatisfiable: k8sv1.DoNotSchedule,
					LabelSelector:     nil,
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.topologySpreadConstraints[0].topologyKey"))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("spec.topologySpreadConstraints[0].topologyKey: Invalid value: \"hostname=host1\": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')"))
		})

		It("Should reject MaxSkew is not valid", func() {
			vmi.Spec.TopologySpreadConstraints = []k8sv1.TopologySpreadConstraint{
				{
					MaxSkew:           -1,
					TopologyKey:       k8sv1.LabelHostname,
					WhenUnsatisfiable: k8sv1.DoNotSchedule,
					LabelSelector:     nil,
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.topologySpreadConstraints[0].maxSkew"))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("spec.topologySpreadConstraints[0].maxSkew: Invalid value: -1: must be greater than zero"))
		})

		It("Should reject when validation failed due to values of MatchExpressions is set to nil", func() {
			vmi.Spec.TopologySpreadConstraints = []k8sv1.TopologySpreadConstraint{
				{
					MaxSkew:           1,
					TopologyKey:       k8sv1.LabelHostname,
					WhenUnsatisfiable: k8sv1.DoNotSchedule,
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      k8sv1.LabelHostname,
								Operator: metav1.LabelSelectorOpIn,
								Values:   nil,
							},
						},
					},
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.topologySpreadConstraints.labelSelector.matchExpressions[0].values"))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("spec.topologySpreadConstraints.labelSelector.matchExpressions[0].values: Required value: must be specified when `operator` is 'In' or 'NotIn'"))
		})
	})

	Context("with persistent reservation defined", func() {
		var vmi *v1.VirtualMachineInstance
		addLunDiskWithPersistentReservation := func(vmi *v1.VirtualMachineInstance) {
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks,
				v1.Disk{
					Name: "testdisk",
					DiskDevice: v1.DiskDevice{
						LUN: &v1.LunTarget{
							Reservation: true,
						},
					},
				},
			)
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testdisk",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: testutils.NewFakePersistentVolumeSource(),
				},
			})
		}

		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
			enableFeatureGates(featuregate.PersistentReservation)
		})

		Context("feature gate enabled", func() {
			It("should accept vmi with no persistent reservation defined", func() {
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(causes).To(BeEmpty())
			})

			It("should accept vmi with persistent reservation defined", func() {
				addLunDiskWithPersistentReservation(vmi)
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(causes).To(BeEmpty())
			})
		})

		Context("feature gate disabled", func() {
			It("should reject when the feature gate is disabled", func() {
				disableFeatureGates()
				addLunDiskWithPersistentReservation(vmi)
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
				Expect(causes).To(HaveLen(1))
				Expect(causes[0].Message).To(ContainSubstring(fmt.Sprintf("%s feature gate is not enabled", featuregate.PersistentReservation)))
			})
		})
	})

	Context("with CPU hotplug", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
			vmi.Spec.Architecture = runtime.GOARCH
		})

		When("number of sockets higher than maxSockets", func() {
			It("deny VMI creation", func() {
				vmi.Spec.Domain.CPU = &v1.CPU{
					MaxSockets: 8,
					Sockets:    16,
				}

				ar, err := newAdmissionReviewForVMICreation(vmi)
				Expect(err).ToNot(HaveOccurred())

				resp := vmiCreateAdmitter.Admit(context.Background(), ar)
				Expect(resp.Allowed).To(BeFalse())
				Expect(resp.Result.Details.Causes).To(HaveLen(1))
				Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.domain.cpu.sockets"))

			})
		})
	})

	Context("hyperV passthrough", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
			vmi.Spec.Architecture = runtime.GOARCH
		})

		const useExplicitHyperV, useHyperVPassthrough = true, true
		const doNotUseExplicitHyperV, doNotUseHyperVPassthrough = false, false

		DescribeTable("Use of hyperV combined with hyperV passthrough is forbidden", func(explicitHyperv, hypervPassthrough, expectValid bool) {
			vmi.Spec.Domain.Features = &v1.Features{}

			if explicitHyperv {
				vmi.Spec.Domain.Features.Hyperv = &v1.FeatureHyperv{}
			}
			if hypervPassthrough {
				vmi.Spec.Domain.Features.HypervPassthrough = &v1.HyperVPassthrough{}
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())
			resp := vmiCreateAdmitter.Admit(context.Background(), ar)

			if expectValid {
				Expect(resp.Allowed).To(BeTrue())
			} else {
				Expect(resp.Allowed).To(BeFalse())
				Expect(resp.Result.Details.Causes).To(HaveLen(1))
				Expect(resp.Result.Details.Causes[0].Field).To(ContainSubstring("hyperv"))
			}
		},
			Entry("explicit + passthrough", useExplicitHyperV, useHyperVPassthrough, false),
			Entry("passthrough only", doNotUseExplicitHyperV, useHyperVPassthrough, true),
			Entry("explicit only", useExplicitHyperV, doNotUseHyperVPassthrough, true),
			Entry("no hyperv use", doNotUseExplicitHyperV, doNotUseHyperVPassthrough, true),
		)

	})

	Context("Watchdog device validation", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
		})

		DescribeTable("validate for amd64",
			func(watchdog *v1.Watchdog, expectedMessage string, shouldReject bool) {
				vmi.Spec.Domain.Devices.Watchdog = watchdog
				causes := webhooks.ValidateVirtualMachineInstanceAmd64Setting(k8sfield.NewPath("fake"), &vmi.Spec)

				if shouldReject {
					Expect(causes).To(HaveLen(1))
					Expect(causes[0].Field).To(Equal("fake.domain.devices.watchdog"))
					Expect(causes[0].Message).To(Equal(expectedMessage))
				} else {
					Expect(causes).To(BeEmpty())
				}
			},
			Entry("I6300ESB is accepted", &v1.Watchdog{
				Name: "w1",
				WatchdogDevice: v1.WatchdogDevice{
					I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionPoweroff},
				},
			}, "", false),

			Entry("Diag288 is rejected", &v1.Watchdog{
				Name: "w2",
				WatchdogDevice: v1.WatchdogDevice{
					Diag288: &v1.Diag288Watchdog{Action: v1.WatchdogActionPoweroff},
				},
			}, "amd64 only supports I6300ESB watchdog device", true),

			Entry("no watchdog configured", nil, "", false),
		)

		DescribeTable("validate for s390x",
			func(watchdog *v1.Watchdog, expectedMessage string, shouldReject bool) {
				vmi.Spec.Domain.Devices.Watchdog = watchdog
				causes := webhooks.ValidateVirtualMachineInstanceS390XSetting(k8sfield.NewPath("fake"), &vmi.Spec)

				if shouldReject {
					Expect(causes).To(HaveLen(1))
					Expect(causes[0].Field).To(Equal("fake.domain.devices.watchdog"))
					Expect(causes[0].Message).To(Equal(expectedMessage))
				} else {
					Expect(causes).To(BeEmpty())
				}
			},
			Entry("Diag288 is accepted", &v1.Watchdog{
				Name: "w3",
				WatchdogDevice: v1.WatchdogDevice{
					Diag288: &v1.Diag288Watchdog{Action: v1.WatchdogActionPoweroff},
				},
			}, "", false),

			Entry("I6300ESB is rejected", &v1.Watchdog{
				Name: "w4",
				WatchdogDevice: v1.WatchdogDevice{
					I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionPoweroff},
				},
			}, "s390x only supports Diag288 watchdog device", true),

			Entry("no watchdog configured", nil, "", false),
		)

		DescribeTable("validate for arm64",
			func(watchdog *v1.Watchdog, expectedMessage string, shouldReject bool) {
				vmi.Spec.Domain.Devices.Watchdog = watchdog
				causes := webhooks.ValidateVirtualMachineInstanceArm64Setting(k8sfield.NewPath("fake"), &vmi.Spec)

				if shouldReject {
					Expect(causes).To(HaveLen(1))
					Expect(causes[0].Field).To(Equal("fake.domain.devices.watchdog"))
					Expect(causes[0].Message).To(Equal(expectedMessage))
				} else {
					Expect(causes).To(BeEmpty())
				}
			},
			Entry("I6300ESB is rejected", &v1.Watchdog{
				Name: "w5",
				WatchdogDevice: v1.WatchdogDevice{
					I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionPoweroff},
				},
			}, "Arm64 not support Watchdog device", true),

			Entry("Diag288 is rejected", &v1.Watchdog{
				Name: "w6",
				WatchdogDevice: v1.WatchdogDevice{
					Diag288: &v1.Diag288Watchdog{Action: v1.WatchdogActionPoweroff},
				},
			}, "Arm64 not support Watchdog device", true),

			Entry("no watchdog configured", nil, "", false),
		)
	})

	Context("with VideoConfig", func() {
		var vmi *v1.VirtualMachineInstance
		BeforeEach(func() {
			enableFeatureGates(featuregate.VideoConfig)
			vmi = libvmi.New(libvmi.WithArchitecture(runtime.GOARCH), libvmi.WithVideo(v1.VirtIO))
		})

		It("should accept video configuration with feature gate enabled", func() {
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty(), "should accept video configuration with valid setup")
		})

		It("should reject when the feature gate is disabled", func() {
			disableFeatureGates()
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(causes[0].Message).To(Equal(fmt.Sprintf("Video configuration is specified but the %s feature gate is not enabled", featuregate.VideoConfig)))
			Expect(causes[0].Field).To(Equal("fake.video"))
		})

		It("should reject when autoattachGraphicsDevice is set to false", func() {
			vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = pointer.P(false)
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(causes[0].Message).To(Equal("Video configuration is not allowed when autoattachGraphicsDevice is set to false"))
			Expect(causes[0].Field).To(Equal("fake.video"))
		})

		It("should accept when autoattachGraphicsDevice is unset", func() {
			vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = nil
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec, config)
			Expect(causes).To(BeEmpty(), "should accept video configuration when autoattachGraphicsDevice is unset")
		})

		DescribeTable("should accept supported video models per architecture", func(arch, videoType string) {
			vmi.Spec.Domain.Devices.Video.Type = videoType
			vmi.Spec.Architecture = arch
			causes := ValidateVirtualMachineInstancePerArch(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(causes).To(BeEmpty(), fmt.Sprintf("expected video type %s to be valid on arch %s", videoType, arch))
		},
			Entry("amd64 allows vga", "amd64", "vga"),
			Entry("amd64 allows cirrus", "amd64", "cirrus"),
			Entry("amd64 allows virtio", "amd64", "virtio"),
			Entry("amd64 allows ramfb", "amd64", "ramfb"),
			Entry("amd64 allows bochs", "amd64", "bochs"),

			Entry("arm64 allows virtio", "arm64", "virtio"),
			Entry("arm64 allows bochs", "arm64", "ramfb"),

			Entry("s390x allows virtio", "s390x", "virtio"),

			Entry("ppc64le allows virtio", "ppc64le", "virtio"),
			Entry("ppc64le allows bochs", "ppc64le", "bochs"),
			Entry("ppc64le allows vga", "ppc64le", "vga"),
			Entry("ppc64le allows cirrus", "ppc64le", "cirrus"),
		)

		DescribeTable("should reject unsupported video models per architecture", func(arch, videoType string) {
			vmi.Spec.Domain.Devices.Video.Type = videoType
			vmi.Spec.Architecture = arch
			causes := ValidateVirtualMachineInstancePerArch(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(causes).ToNot(BeEmpty(), fmt.Sprintf("expected video type %s to be invalid on arch %s", videoType, arch))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.video.type"))
		},
			Entry("amd64 rejects qxl", "amd64", "qxl"),
			Entry("amd64 rejects vmvga", "amd64", "vmvga"),
			Entry("amd64 rejects xenfb", "amd64", "xenfb"),
			Entry("amd64 rejects none", "amd64", "none"),
			Entry("amd64 rejects invalid model", "amd64", "invalidmodel"),

			Entry("arm64 rejects vga", "arm64", "vga"),
			Entry("arm64 rejects cirrus", "arm64", "cirrus"),
			Entry("arm64 rejects bochs", "arm64", "bochs"),
			Entry("arm64 rejects qxl", "arm64", "qxl"),
			Entry("arm64 rejects vmvga", "arm64", "vmvga"),
			Entry("arm64 rejects xenfb", "arm64", "xenfb"),
			Entry("arm64 rejects none", "arm64", "none"),
			Entry("arm64 rejects invalid model", "arm64", "invalidmodel"),

			Entry("s390x rejects vga", "s390x", "vga"),
			Entry("s390x rejects cirrus", "s390x", "cirrus"),
			Entry("s390x rejects bochs", "s390x", "bochs"),
			Entry("s390x rejects ramfb", "s390x", "ramfb"),
			Entry("s390x rejects qxl", "s390x", "qxl"),
			Entry("s390x rejects vmvga", "s390x", "vmvga"),
			Entry("s390x rejects xenfb", "s390x", "xenfb"),
			Entry("s390x rejects none", "s390x", "none"),
			Entry("s390x rejects invalid model", "s390x", "invalidmodel"),

			Entry("ppc64le rejects ramfb", "ppc64le", "ramfb"),
			Entry("ppc64le rejects qxl", "ppc64le", "qxl"),
			Entry("ppc64le rejects vmvga", "ppc64le", "vmvga"),
			Entry("ppc64le rejects xenfb", "ppc64le", "xenfb"),
			Entry("ppc64le rejects none", "ppc64le", "none"),
			Entry("ppc64le rejects invalid model", "ppc64le", "invalidmodel"),
		)
	})

	Context("with DRA GPUs", func() {
		It("Should require deviceName without DRA", func() {
			vmi := libvmi.New(
				libvmi.WithArchitecture(runtime.GOARCH),
				libvmi.WithResourceMemory("128M"),
			)
			vmi.Spec.Domain.Devices.GPUs = append(vmi.Spec.Domain.Devices.GPUs,
				v1.GPU{
					Name: "rejected-gpu",
				},
			)

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(ContainSubstring("contains GPUs without deviceName"))
		})

		It("should reject a GPU that sets both deviceName and claimRequest", func() {
			enableFeatureGates(featuregate.GPUsWithDRAGate)
			vmi := libvmi.New(
				libvmi.WithArchitecture(runtime.GOARCH),
				libvmi.WithResourceMemory("128M"),
			)
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{
					Name:       "gpu",
					DeviceName: "nvidia-gpu",
					ClaimRequest: &v1.ClaimRequest{
						ClaimName:   pointer.P("my-gpu-claim"),
						RequestName: pointer.P("request-1"),
					},
				},
			}
			vmi.Spec.ResourceClaims = []k8sv1.PodResourceClaim{
				{Name: "my-gpu-claim"},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(ContainSubstring("contains GPUs with both deviceName and claimRequest"))
		})

		It("should reject a DRA-GPU when the feature-gate is NOT enabled", func() {
			vmi := libvmi.New()
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{
					Name: "gpu",
					ClaimRequest: &v1.ClaimRequest{
						ClaimName:   pointer.P("my-gpu-claim"),
						RequestName: pointer.P("request-1"),
					},
				},
			}

			vmi.Spec.ResourceClaims = []k8sv1.PodResourceClaim{{Name: "my-gpu-claim"}}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(ContainSubstring("DRA enabled GPUs but feature gate is not enabled"))
		})

		It("should reject a DRA-GPU if its claim is missing from spec.resourceClaims", func() {
			enableFeatureGates(featuregate.GPUsWithDRAGate)

			vmi := libvmi.New()
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{
					Name: "gpu",
					ClaimRequest: &v1.ClaimRequest{
						ClaimName:   pointer.P("my-gpu-claim"),
						RequestName: pointer.P("request-1"),
					},
				},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(ContainSubstring("vmi.spec.resourceClaims must specify all claims"))
		})

		It("should accept a DRA-GPU when the gate is enabled and the claim is listed", func() {
			enableFeatureGates(featuregate.GPUsWithDRAGate)

			vmi := libvmi.New(
				libvmi.WithArchitecture(runtime.GOARCH),
				libvmi.WithResourceMemory("128M"),
			)
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{
					Name: "gpu",
					ClaimRequest: &v1.ClaimRequest{
						ClaimName:   pointer.P("my-gpu-claim"),
						RequestName: pointer.P("request-1"),
					},
				},
			}
			vmi.Spec.ResourceClaims = []k8sv1.PodResourceClaim{
				{Name: "my-gpu-claim"},
			}

			ar, err := newAdmissionReviewForVMICreation(vmi)
			Expect(err).ToNot(HaveOccurred())

			resp := vmiCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue(), fmt.Sprint(resp.Result))
		})
	})
})

var _ = Describe("additional tests", func() {
	config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})

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
		Expect(causes).To(BeEmpty())
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
		Expect(causes).To(BeEmpty())
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

	It("should accept a valid serial number", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		sn := "6a1a24a1-4061-4607-8bf4-a3963d0c5895"

		spec.Domain.Firmware = &v1.Firmware{Serial: sn}

		causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), spec, config)
		Expect(causes).To(BeEmpty())
	})

	It("Should validate VMIs without HyperV configuration", func() {
		vmi := api.NewMinimalVMI("testvmi")
		Expect(vmi.Spec.Domain.Features).To(BeNil())
		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(causes).To(BeEmpty())
	})

	It("Should validate VMIs with empty HyperV configuration", func() {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{},
		}
		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(causes).To(BeEmpty())
	})

	It("Should validate VMIs with hyperv configuration without deps", func() {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				Relaxed: &v1.FeatureState{
					Enabled: pointer.P(true),
				},
				Runtime: &v1.FeatureState{
					Enabled: pointer.P(true),
				},
				Reset: &v1.FeatureState{
					Enabled: pointer.P(true),
				},
			},
		}
		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(causes).To(BeEmpty())
	})

	It("Should validate VMIs with hyperv EVMCS configuration without deps and detect multiple issues", func() {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				EVMCS: &v1.FeatureState{
					Enabled: pointer.P(true),
				},
			},
		}
		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(causes).To(HaveLen(2), "should return error")
		Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid), "type should equal")
		Expect(causes[0].Field).To(Equal("spec.domain.features.hyperv.evmcs"), "field should equal")
		Expect(causes[1].Type).To(Equal(metav1.CauseTypeFieldValueRequired), "type should equal")
		Expect(causes[1].Field).To(Equal("spec.domain.cpu.features"), "field should equal")
	})

	It("Should validate VMIs with hyperv EVMCS configuration without deps", func() {
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
					Enabled: pointer.P(true),
				},
			},
		}
		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(causes).To(HaveLen(1), "should return error")
		Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid), "type should equal")
		Expect(causes[0].Field).To(Equal("spec.domain.features.hyperv.evmcs"), "field should equal")
	})

	It("Should validate VMIs with hyperv EVMCS configuration with hyperv deps, but without vmx cpu feature", func() {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				EVMCS: &v1.FeatureState{
					Enabled: pointer.P(true),
				},
				VAPIC: &v1.FeatureState{
					Enabled: pointer.P(true),
				},
			},
		}
		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(causes).To(HaveLen(1), "should return error")
		Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueRequired), "type should equal")
		Expect(causes[0].Field).To(Equal("spec.domain.cpu.features"), "field should equal")
	})

	It("Should validate VMIs with hyperv EVMCS configuration with vmx forbid", func() {
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
					Enabled: pointer.P(true),
				},
				VAPIC: &v1.FeatureState{
					Enabled: pointer.P(true),
				},
			},
		}
		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(causes).To(HaveLen(1), "should return error")
		Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid), "type should equal")
		Expect(causes[0].Field).To(Equal("spec.domain.cpu.features[0].policy"), "field should equal")
	})

	It("Should validate VMIs with hyperv EVMCS configuration with wrong vmx policy", func() {
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
					Enabled: pointer.P(true),
				},
				VAPIC: &v1.FeatureState{
					Enabled: pointer.P(true),
				},
			},
		}
		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(causes).To(BeEmpty(), "should not return error")
	})

	It("Should not validate VMIs with broken hyperv deps", func() {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				Relaxed: &v1.FeatureState{
					Enabled: pointer.P(true),
				},
				SyNIC: &v1.FeatureState{
					Enabled: pointer.P(true),
				},
				SyNICTimer: &v1.SyNICTimer{
					Enabled: pointer.P(true),
				},
			},
		}
		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(causes).ToNot(BeEmpty())
	})

	It("Should validate VMIs with correct hyperv deps", func() {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				Relaxed: &v1.FeatureState{
					Enabled: pointer.P(true),
				},
				VPIndex: &v1.FeatureState{
					Enabled: pointer.P(true),
				},
				SyNIC: &v1.FeatureState{
					Enabled: pointer.P(true),
				},
				SyNICTimer: &v1.SyNICTimer{
					Enabled: pointer.P(true),
				},
			},
		}

		path := k8sfield.NewPath("spec")
		causes := webhooks.ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(causes).To(BeEmpty())
	})
})

func newBaseVmi(opts ...libvmi.Option) *v1.VirtualMachineInstance {
	opts = append(opts,
		libvmi.WithMemoryRequest("512Mi"),
		libvmi.WithArchitecture(runtime.GOARCH),
	)
	return libvmi.New(opts...)
}

func newAdmissionReviewForVMICreation(vmi *v1.VirtualMachineInstance) (*admissionv1.AdmissionReview, error) {
	vmiBytes, err := json.Marshal(vmi)
	if err != nil {
		return nil, err
	}

	return &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
			Object: k8sruntime.RawExtension{
				Raw: vmiBytes,
			},
			Operation: admissionv1.Create,
		},
	}, err
}

func withDomainClock(clock *v1.Clock) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Clock = clock
	}
}

func withReadinessProbe(probe *v1.Probe) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.ReadinessProbe = probe
	}
}

func withLivenessProbe(probe *v1.Probe) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.LivenessProbe = probe
	}
}

func newValidateStub(statusCauses ...metav1.StatusCause) SpecValidator {
	return func(_ *k8sfield.Path, _ *v1.VirtualMachineInstanceSpec, _ *virtconfig.ClusterConfig) []metav1.StatusCause {
		return statusCauses
	}
}
