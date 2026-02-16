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

package webhooks

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("Validating KubeVirtUpdate Admitter", func() {
	test := field.NewPath("test")
	vmProfileField := test.Child("virtualMachineInstanceProfile")

	DescribeTable("validateSeccompConfiguration", func(seccompConfiguration *v1.SeccompConfiguration, expectedFields []string) {
		causes := validateSeccompConfiguration(test, seccompConfiguration)
		Expect(causes).To(HaveLen(len(expectedFields)))
		for _, cause := range causes {
			Expect(cause.Field).To(BeElementOf(expectedFields))
		}
	},
		Entry("don't specifying custom ", &v1.SeccompConfiguration{
			VirtualMachineInstanceProfile: &v1.VirtualMachineInstanceProfile{
				CustomProfile: nil,
			},
		}, []string{vmProfileField.Child("customProfile").String()}),

		Entry("having custom local and runtimeDefault Profile", &v1.SeccompConfiguration{
			VirtualMachineInstanceProfile: &v1.VirtualMachineInstanceProfile{
				CustomProfile: &v1.CustomProfile{
					RuntimeDefaultProfile: true,
					LocalhostProfile:      pointer.P("somethingNotImportant"),
				},
			},
		}, []string{vmProfileField.Child("customProfile", "runtimeDefaultProfile").String(), vmProfileField.Child("customProfile", "localhostProfile").String()}),
	)

	DescribeTable("test validateCustomizeComponents", func(cc v1.CustomizeComponents, expectedCauses int) {
		causes := validateCustomizeComponents(cc)
		Expect(causes).To(HaveLen(expectedCauses))
	},
		Entry("invalid values rejected", v1.CustomizeComponents{
			Patches: []v1.CustomizeComponentsPatch{
				{
					ResourceName: "virt-api",
					ResourceType: "Deployment",
					Type:         v1.StrategicMergePatchType,
					Patch:        `{"json: "not valid"}`,
				},
			},
		}, 1),
		Entry("empty patch field rejected", v1.CustomizeComponents{
			Patches: []v1.CustomizeComponentsPatch{
				{
					ResourceName: "virt-api",
					ResourceType: "Deployment",
					Type:         v1.StrategicMergePatchType,
					Patch:        "",
				},
			},
		}, 1),
		Entry("valid values accepted", v1.CustomizeComponents{
			Patches: []v1.CustomizeComponentsPatch{
				{
					ResourceName: "virt-api",
					ResourceType: "Deployment",
					Type:         v1.StrategicMergePatchType,
					Patch:        `{}`,
				},
			},
		}, 0),
	)

	Context("with TLSConfiguration", func() {
		DescribeTable("should reject", func(tlsConfiguration *v1.TLSConfiguration, expectedErrorMessage string, indexInField int) {
			causes := validateTLSConfiguration(tlsConfiguration)

			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(Equal(expectedErrorMessage))
			field := "spec.configuration.tlsConfiguration.ciphers"
			if indexInField != -1 {
				field = fmt.Sprintf("%s#%d", field, indexInField)
			}
			Expect(causes[0].Field).To(Equal(field))
		},
			Entry("with unspecified minTLSVersion but non empty ciphers",
				&v1.TLSConfiguration{Ciphers: []string{tls.CipherSuiteName(tls.TLS_AES_256_GCM_SHA384)}},
				"You cannot specify ciphers when spec.configuration.tlsConfiguration.minTLSVersion is empty or VersionTLS13",
				-1,
			),
			Entry("with specified ciphers and minTLSVersion = 1.3",
				&v1.TLSConfiguration{Ciphers: []string{tls.CipherSuiteName(tls.TLS_AES_256_GCM_SHA384)}, MinTLSVersion: v1.VersionTLS13},
				"You cannot specify ciphers when spec.configuration.tlsConfiguration.minTLSVersion is empty or VersionTLS13",
				-1,
			),
			Entry("with unknown cipher in the list",
				&v1.TLSConfiguration{
					MinTLSVersion: v1.VersionTLS12,
					Ciphers:       []string{tls.CipherSuiteName(tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256), "NOT_VALID_CIPHER"},
				},
				"NOT_VALID_CIPHER is not a valid cipher",
				1,
			),
		)
	})

	Context("with AdditionalGuestMemoryOverheadRatio", func() {
		DescribeTable("the ratio must be parsable to float", func(unparsableRatio string) {
			causes := validateGuestToRequestHeadroom(&unparsableRatio)
			Expect(causes).To(HaveLen(1))
		},
			Entry("not a number", "abcdefg"),
			Entry("number with bad formatting", "1.fd3ggx"),
		)

		DescribeTable("the ratio must be larger than 1", func(lessThanOneRatio string) {
			causes := validateGuestToRequestHeadroom(&lessThanOneRatio)
			Expect(causes).ToNot(BeEmpty())
		},
			Entry("0.999", "0.999"),
			Entry("negative number", "-1.3"),
		)

		DescribeTable("valid values", func(validRatio string) {
		},
			Entry("1.0", "1.0"),
			Entry("5", "5"),
			Entry("1.123", "1.123"),
		)
	})

	Context("validateCPUModel", func() {
		var (
			ctrl       *gomock.Controller
			mockClient *kubecli.MockKubevirtClient
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			mockClient = kubecli.NewMockKubevirtClient(ctrl)
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		DescribeTable("should accept valid values without querying nodes", func(cpuModel string) {
			causes, warning := validateCPUModel(context.Background(), cpuModel, mockClient)
			Expect(causes).To(BeEmpty())
			Expect(warning).To(BeEmpty())
		},
			Entry("empty string", ""),
			Entry("host-passthrough", v1.CPUModeHostPassthrough),
			Entry("host-model", v1.CPUModeHostModel),
		)

		It("should accept a CPU model that exists on all cluster nodes", func() {
			kubeClient := k8sfake.NewSimpleClientset(
				&k8sv1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
						Labels: map[string]string{
							v1.CPUModelLabel + "Haswell": "true",
						},
					},
				},
				&k8sv1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node2",
						Labels: map[string]string{
							v1.CPUModelLabel + "Haswell": "true",
						},
					},
				},
			)
			mockClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

			causes, warning := validateCPUModel(context.Background(), "Haswell", mockClient)
			Expect(causes).To(BeEmpty())
			Expect(warning).To(BeEmpty())
		})

		It("should warn when CPU model exists only on subset of nodes", func() {
			kubeClient := k8sfake.NewSimpleClientset(
				&k8sv1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
						Labels: map[string]string{
							v1.CPUModelLabel + "Haswell": "true",
						},
					},
				},
				&k8sv1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node2",
						Labels: map[string]string{
							v1.CPUModelLabel + "Cascadelake-Server": "true",
						},
					},
				},
				&k8sv1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node3",
						Labels: map[string]string{
							v1.CPUModelLabel + "Cascadelake-Server": "true",
						},
					},
				},
			)
			mockClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

			causes, warning := validateCPUModel(context.Background(), "Haswell", mockClient)
			Expect(causes).To(BeEmpty())
			Expect(warning).ToNot(BeEmpty())
			Expect(warning).To(ContainSubstring("Haswell"))
			Expect(warning).To(ContainSubstring("1 of 3"))
			Expect(warning).To(ContainSubstring("33%"))
			Expect(warning).To(ContainSubstring("For cluster-wide VM scheduling"))
		})

		It("should reject a CPU model not supported by any node", func() {
			kubeClient := k8sfake.NewSimpleClientset()
			mockClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

			causes, warning := validateCPUModel(context.Background(), "Homer-Simpson", mockClient)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(causes[0].Field).To(Equal("spec.configuration.cpuModel"))
			Expect(causes[0].Message).To(ContainSubstring("Homer-Simpson"))
			Expect(warning).To(BeEmpty())
		})

		It("should reject a misspelled CPU model", func() {
			kubeClient := k8sfake.NewSimpleClientset(&k8sv1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Labels: map[string]string{
						v1.CPUModelLabel + "Cascadelake-Server": "true",
					},
				},
			})
			mockClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

			causes, warning := validateCPUModel(context.Background(), "CascadelakeServer", mockClient)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("CascadelakeServer"))
			Expect(warning).To(BeEmpty())
		})

		It("should not fail when client is nil", func() {
			causes, warning := validateCPUModel(context.Background(), "SomeModel", nil)
			Expect(causes).To(BeEmpty())
			Expect(warning).To(BeEmpty())
		})
	})

	Context("deprecations", func() {
		var admitter *KubeVirtUpdateAdmitter

		BeforeEach(func() {
			clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
			admitter = NewKubeVirtUpdateAdmitter(nil, clusterConfig)
		})

		admit := func(kubevirt v1.KubeVirt) *admissionv1.AdmissionResponse {
			return admitKVUpdate(admitter, &kubevirt, &kubevirt)
		}

		const warn = true
		const warnNotExpected = false

		DescribeTable("usage of mediatedDevicesTypes", func(shouldWarn bool, conf *v1.MediatedDevicesConfiguration) {
			kvObject := v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						MediatedDevicesConfiguration: conf,
					},
				},
			}

			response := admit(kvObject)
			Expect(response).NotTo(BeNil())
			if shouldWarn {
				Expect(response.Warnings).NotTo(BeEmpty())
				Expect(response.Warnings).To(ContainElement("spec.configuration.mediatedDevicesConfiguration.mediatedDevicesTypes is deprecated, use mediatedDeviceTypes"))
			} else {
				Expect(response.Warnings).To(BeEmpty())
			}
		},
			Entry("should warn if used", warn, &v1.MediatedDevicesConfiguration{
				MediatedDevicesTypes: []string{"test1", "test2"},
			}),

			Entry("should not warn if empty", warnNotExpected, &v1.MediatedDevicesConfiguration{
				MediatedDevicesTypes: []string{},
			}),
			Entry("should not warn if nil", warnNotExpected, &v1.MediatedDevicesConfiguration{
				MediatedDevicesTypes: nil,
			}),
			Entry("should not warn if configuration is nil", warnNotExpected, nil),
		)

		DescribeTable("usage of nodeMediatedDeviceTypes.mediatedDevicesTypes", func(shouldWarn bool, conf *v1.MediatedDevicesConfiguration) {
			kvObject := v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						MediatedDevicesConfiguration: conf,
					},
				},
			}

			response := admit(kvObject)
			Expect(response).NotTo(BeNil())
			if shouldWarn {
				Expect(response.Warnings).NotTo(BeEmpty())
				Expect(response.Warnings).To(ContainElement("spec.configuration.mediatedDevicesConfiguration.nodeMediatedDeviceTypes[0].mediatedDevicesTypes is deprecated, use mediatedDeviceTypes"))
			} else {
				Expect(response.Warnings).To(BeEmpty())
			}
		}, Entry("should warn if used", warn, &v1.MediatedDevicesConfiguration{
			NodeMediatedDeviceTypes: []v1.NodeMediatedDeviceTypesConfig{
				{
					NodeSelector:         map[string]string{},
					MediatedDevicesTypes: []string{"test1", "test2"},
					MediatedDeviceTypes:  []string{},
				},
			},
		}),
			Entry("should not warn if empty", warnNotExpected, &v1.MediatedDevicesConfiguration{
				NodeMediatedDeviceTypes: []v1.NodeMediatedDeviceTypesConfig{
					{
						NodeSelector:         map[string]string{},
						MediatedDevicesTypes: []string{},
						MediatedDeviceTypes:  []string{},
					},
				},
			}),
			Entry("should not warn if nil", warnNotExpected, &v1.MediatedDevicesConfiguration{
				NodeMediatedDeviceTypes: []v1.NodeMediatedDeviceTypesConfig{
					{
						NodeSelector:         map[string]string{},
						MediatedDevicesTypes: nil,
						MediatedDeviceTypes:  []string{},
					},
				},
			}),

			Entry("should not warn if configuration nil", warnNotExpected, nil),
		)

		DescribeTable("should raise warning when a deprecated feature-gate is enabled", func(featureGate, expectedWarning string) {
			kv := v1.KubeVirt{}
			kvBytes, err := json.Marshal(kv)
			Expect(err).ToNot(HaveOccurred())

			kv.Spec.Configuration.DeveloperConfiguration = &v1.DeveloperConfiguration{FeatureGates: []string{featureGate}}
			kvUpdatedBytes, err := json.Marshal(kv)
			Expect(err).ToNot(HaveOccurred())

			request := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Resource:  KubeVirtGroupVersionResource,
					Operation: admissionv1.Update,
					OldObject: runtime.RawExtension{Raw: kvBytes},
					Object:    runtime.RawExtension{Raw: kvUpdatedBytes},
				},
			}

			Expect(admitter.Admit(context.Background(), request)).To(Equal(&admissionv1.AdmissionResponse{
				Allowed: true,
				Warnings: []string{
					expectedWarning,
				},
			}))
		},
			Entry("with LiveMigration", featuregate.LiveMigrationGate, fmt.Sprintf(featuregate.WarningPattern, featuregate.LiveMigrationGate, featuregate.GA)),
			Entry("with SRIOVLiveMigration", featuregate.SRIOVLiveMigrationGate, fmt.Sprintf(featuregate.WarningPattern, featuregate.SRIOVLiveMigrationGate, featuregate.GA)),
			Entry("with NonRoot", featuregate.NonRoot, fmt.Sprintf(featuregate.WarningPattern, featuregate.NonRoot, featuregate.GA)),
			Entry("with PSA", featuregate.PSA, fmt.Sprintf(featuregate.WarningPattern, featuregate.PSA, featuregate.GA)),
			Entry("with CPUNodeDiscoveryGate", featuregate.CPUNodeDiscoveryGate, fmt.Sprintf(featuregate.WarningPattern, featuregate.CPUNodeDiscoveryGate, featuregate.GA)),
			Entry("with HotplugNICs", featuregate.HotplugNetworkIfacesGate, fmt.Sprintf(featuregate.WarningPattern, featuregate.HotplugNetworkIfacesGate, featuregate.GA)),
			Entry("with Passt", featuregate.PasstGate, featuregate.PasstDiscontinueMessage),
			Entry("with MacvtapGate", featuregate.MacvtapGate, featuregate.MacvtapDiscontinueMessage),
			Entry("with ExperimentalVirtiofsSupport", featuregate.VirtIOFSGate, featuregate.VirtioFsFeatureGateDiscontinueMessage),
			Entry("with DisableMediatedDevicesHandling", featuregate.DisableMediatedDevicesHandling, "DisableMDEVConfiguration has been deprecated since v1.8.0"),
		)

		DescribeTable("should raise warning when archConfig is set for ppc64le", func(shouldWarn bool, archConfig *v1.ArchConfiguration) {
			kv := v1.KubeVirt{
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						ArchitectureConfiguration: archConfig,
					},
				},
			}

			response := admit(kv)
			Expect(response).NotTo(BeNil())

			if shouldWarn {
				Expect(response.Warnings).NotTo(BeEmpty())
				Expect(response.Warnings).To(ContainElement("spec.configuration.architectureConfiguration.ppc64le is deprecated and no longer supported."))
			} else {
				Expect(response.Warnings).To(BeEmpty())
			}
		},
			Entry("should warn when archConfig is set for ppc64le", true, &v1.ArchConfiguration{Ppc64le: &v1.ArchSpecificConfiguration{}}),
			Entry("should not warn when archConfig is not set for ppc64le", false, &v1.ArchConfiguration{}),
		)
	})

	Context("Feature Gate Validation", func() {
		var admitter *KubeVirtUpdateAdmitter

		BeforeEach(func() {
			clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
			admitter = NewKubeVirtUpdateAdmitter(nil, clusterConfig)
		})

		admitUpdate := func(devConfig *v1.DeveloperConfiguration) *admissionv1.AdmissionResponse {
			oldKV := &v1.KubeVirt{ObjectMeta: metav1.ObjectMeta{Name: "test"}}
			newKV := oldKV.DeepCopy()
			newKV.Spec.Configuration.DeveloperConfiguration = devConfig
			return admitKVUpdate(admitter, oldKV, newKV)
		}

		DescribeTable("should reject conflicting feature gates", func(enabledGates, disabledGates []string, expectedConflictingGates ...string) {
			var devConfig *v1.DeveloperConfiguration
			if enabledGates != nil || disabledGates != nil {
				devConfig = &v1.DeveloperConfiguration{
					FeatureGates:         enabledGates,
					DisabledFeatureGates: disabledGates,
				}
			}

			response := admitUpdate(devConfig)

			if len(expectedConflictingGates) == 0 {
				Expect(response.Allowed).To(BeTrue())
			} else {
				Expect(response.Allowed).To(BeFalse())
				Expect(response.Result.Details.Causes).To(HaveLen(len(expectedConflictingGates)))
				for _, gate := range expectedConflictingGates {
					Expect(response.Result.Details.Causes).To(ContainElement(And(
						HaveField("Message", fmt.Sprintf(`feature gate "%s" exists on both "FeatureGates" and "DisabledFeatureGates"`, gate)),
						HaveField("Type", metav1.CauseTypeForbidden),
						HaveField("Field", field.NewPath("spec", "configuration", "developerConfiguration", "featureGates").String()),
					)), `Expected to find conflict for gate: %s`, gate)
				}
			}
		},
			Entry("no conflict - both lists empty",
				[]string{},
				[]string{}),

			Entry("no conflict - only enabled gates",
				[]string{"Gate1", "Gate2"},
				[]string{}),

			Entry("no conflict - only disabled gates",
				[]string{},
				[]string{"Gate1", "Gate2"}),

			Entry("no conflict - different gates",
				[]string{"EnabledGate1", "EnabledGate2"},
				[]string{"DisabledGate1", "DisabledGate2"}),

			Entry("no conflict - nil DeveloperConfiguration",
				nil, nil),

			Entry("single conflict - same gate in both lists",
				[]string{"ConflictGate", "ValidGate1"},
				[]string{"ConflictGate", "ValidGate2"},
				"ConflictGate"),

			Entry("multiple conflicts",
				[]string{"Conflict1", "Conflict2", "ValidGate"},
				[]string{"Conflict1", "Conflict2", "AnotherValid"},
				"Conflict1", "Conflict2"),

			Entry("all gates conflict",
				[]string{"Gate1", "Gate2", "Gate3"},
				[]string{"Gate1", "Gate2", "Gate3"},
				"Gate1", "Gate2", "Gate3"),
		)
	})
})

func admitKVUpdate(admitter *KubeVirtUpdateAdmitter, oldKV, newKV *v1.KubeVirt) *admissionv1.AdmissionResponse {
	oldKVBytes, err := json.Marshal(oldKV)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	newKVBytes, err := json.Marshal(newKV)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	request := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Resource:  KubeVirtGroupVersionResource,
			Operation: admissionv1.Update,
			OldObject: runtime.RawExtension{Raw: oldKVBytes},
			Object:    runtime.RawExtension{Raw: newKVBytes},
		},
	}
	return admitter.Admit(context.Background(), request)
}
