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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("Validating KubeVirtUpdate Admitter", func() {
	test := field.NewPath("test")
	vmProfileField := test.Child("virtualMachineInstanceProfile")

	DescribeTable("validateVirtTemplateDeployment", func(kvSpec v1.KubeVirtSpec, expectError bool) {
		causes := validateVirtTemplateDeployment(&kvSpec.Configuration)
		if expectError {
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(causes[0].Field).To(Equal("spec.configuration.virtTemplateDeployment.enabled"))
		} else {
			Expect(causes).To(BeEmpty())
		}
	},
		Entry("should reject when VirtTemplateDeployment enabled without Template feature gate",
			v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					VirtTemplateDeployment: &v1.VirtTemplateDeployment{
						Enabled: pointer.P(true),
					},
				},
			},
			true,
		),
		Entry("should allow when VirtTemplateDeployment enabled with Template feature gate",
			v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						FeatureGates: []string{featuregate.Template},
					},
					VirtTemplateDeployment: &v1.VirtTemplateDeployment{
						Enabled: pointer.P(true),
					},
				},
			},
			false,
		),
		Entry("should allow when VirtTemplateDeployment is nil",
			v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{},
			},
			false,
		),
		Entry("should allow when VirtTemplateDeployment.Enabled is nil",
			v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					VirtTemplateDeployment: &v1.VirtTemplateDeployment{
						Enabled: nil,
					},
				},
			},
			false,
		),
		Entry("should allow when VirtTemplateDeployment.Enabled is false",
			v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					VirtTemplateDeployment: &v1.VirtTemplateDeployment{
						Enabled: pointer.P(false),
					},
				},
			},
			false,
		),
	)

	DescribeTable("validateRoleAggregationStrategy", func(kvSpec v1.KubeVirtSpec, expectError bool) {
		causes := validateRoleAggregationStrategy(&kvSpec.Configuration)
		if expectError {
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(causes[0].Field).To(Equal("spec.configuration.roleAggregationStrategy"))
		} else {
			Expect(causes).To(BeEmpty())
		}
	},
		Entry("should reject when RoleAggregationStrategy is Manual without OptOutRoleAggregation feature gate",
			v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					RoleAggregationStrategy: pointer.P(v1.RoleAggregationStrategyManual),
				},
			},
			true,
		),
		Entry("should allow when RoleAggregationStrategy is Manual with OptOutRoleAggregation feature gate",
			v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						FeatureGates: []string{featuregate.OptOutRoleAggregation},
					},
					RoleAggregationStrategy: pointer.P(v1.RoleAggregationStrategyManual),
				},
			},
			false,
		),
		Entry("should allow when RoleAggregationStrategy is nil",
			v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{},
			},
			false,
		),
		Entry("should allow when RoleAggregationStrategy is AggregateToDefault without feature gate",
			v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					RoleAggregationStrategy: pointer.P(v1.RoleAggregationStrategyAggregateToDefault),
				},
			},
			false,
		),
	)

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

	Context("with WorkerPools", func() {
		It("should reject pools when feature gate is not enabled", func() {
			causes := validateWorkerPools(&v1.KubeVirtSpec{
				WorkerPools: []v1.WorkerPoolConfig{
					{
						Name:             "test",
						VirtHandlerImage: "img",
						NodeSelector:     map[string]string{"k": "v"},
						Selector:         v1.WorkerPoolSelector{DeviceNames: []string{"dev"}},
					},
				},
			})
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("feature gate"))
		})

		It("should reject duplicate pool names", func() {
			causes := validateWorkerPools(&v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						FeatureGates: []string{"WorkerPools"},
					},
				},
				WorkerPools: []v1.WorkerPoolConfig{
					{Name: "dup", VirtHandlerImage: "img", NodeSelector: map[string]string{"k": "v"}, Selector: v1.WorkerPoolSelector{DeviceNames: []string{"dev"}}},
					{Name: "dup", VirtLauncherImage: "img", NodeSelector: map[string]string{"k2": "v2"}, Selector: v1.WorkerPoolSelector{DeviceNames: []string{"dev2"}}},
				},
			})
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("duplicate"))
		})

		It("should reject pools with no image override", func() {
			causes := validateWorkerPools(&v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						FeatureGates: []string{"WorkerPools"},
					},
				},
				WorkerPools: []v1.WorkerPoolConfig{
					{Name: "no-img", NodeSelector: map[string]string{"k": "v"}, Selector: v1.WorkerPoolSelector{DeviceNames: []string{"dev"}}},
				},
			})
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("virtHandlerImage or virtLauncherImage"))
		})

		It("should reject pools with no selector criteria", func() {
			causes := validateWorkerPools(&v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						FeatureGates: []string{"WorkerPools"},
					},
				},
				WorkerPools: []v1.WorkerPoolConfig{
					{Name: "no-sel", VirtHandlerImage: "img", NodeSelector: map[string]string{"k": "v"}, Selector: v1.WorkerPoolSelector{}},
				},
			})
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("selector"))
		})

		It("should accept valid pool configuration", func() {
			causes := validateWorkerPools(&v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						FeatureGates: []string{"WorkerPools"},
					},
				},
				WorkerPools: []v1.WorkerPoolConfig{
					{Name: "valid", VirtLauncherImage: "img", NodeSelector: map[string]string{"k": "v"}, Selector: v1.WorkerPoolSelector{DeviceNames: []string{"dev"}}},
				},
			})
			Expect(causes).To(BeEmpty())
		})
	})

	Context("validateWorkerPoolRemoval", func() {
		var (
			ctrl      *gomock.Controller
			client    *kubecli.MockKubevirtClient
			vmiClient *kubecli.MockVirtualMachineInstanceInterface
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			client = kubecli.NewMockKubevirtClient(ctrl)
			vmiClient = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
			client.EXPECT().VirtualMachineInstance("").Return(vmiClient).AnyTimes()
		})

		It("should allow pool removal when no VMIs match", func() {
			vmiClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1.VirtualMachineInstanceList{}, nil)
			causes := validateWorkerPoolRemoval(context.Background(),
				&v1.KubeVirtSpec{
					WorkerPools: []v1.WorkerPoolConfig{
						{Name: "gpu-pool", VirtLauncherImage: "img", NodeSelector: map[string]string{"gpu": "true"}, Selector: v1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/T4"}}},
					},
				},
				&v1.KubeVirtSpec{},
				client,
			)
			Expect(causes).To(BeEmpty())
		})

		It("should reject pool removal when running VMIs match by device", func() {
			vmiClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1.VirtualMachineInstanceList{
				Items: []v1.VirtualMachineInstance{
					{
						Status: v1.VirtualMachineInstanceStatus{Phase: v1.Running},
						Spec: v1.VirtualMachineInstanceSpec{
							Domain: v1.DomainSpec{
								Devices: v1.Devices{
									GPUs: []v1.GPU{{Name: "gpu1", DeviceName: "nvidia.com/T4"}},
								},
							},
						},
					},
				},
			}, nil)
			causes := validateWorkerPoolRemoval(context.Background(),
				&v1.KubeVirtSpec{
					WorkerPools: []v1.WorkerPoolConfig{
						{Name: "gpu-pool", VirtLauncherImage: "img", NodeSelector: map[string]string{"gpu": "true"}, Selector: v1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/T4"}}},
					},
				},
				&v1.KubeVirtSpec{},
				client,
			)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("cannot remove pool"))
			Expect(causes[0].Message).To(ContainSubstring("gpu-pool"))
		})

		It("should reject pool removal when running VMIs match by label", func() {
			vmiClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1.VirtualMachineInstanceList{
				Items: []v1.VirtualMachineInstance{
					{
						ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"workload": "secure"}},
						Status:     v1.VirtualMachineInstanceStatus{Phase: v1.Running},
					},
				},
			}, nil)
			causes := validateWorkerPoolRemoval(context.Background(),
				&v1.KubeVirtSpec{
					WorkerPools: []v1.WorkerPoolConfig{
						{Name: "secure-pool", VirtLauncherImage: "img", NodeSelector: map[string]string{"zone": "secure"}, Selector: v1.WorkerPoolSelector{VMLabels: &v1.WorkerPoolVMLabels{MatchLabels: map[string]string{"workload": "secure"}}}},
					},
				},
				&v1.KubeVirtSpec{},
				client,
			)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("secure-pool"))
		})

		It("should allow pool removal when matching VMIs are in final state", func() {
			vmiClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1.VirtualMachineInstanceList{
				Items: []v1.VirtualMachineInstance{
					{
						Status: v1.VirtualMachineInstanceStatus{Phase: v1.Succeeded},
						Spec: v1.VirtualMachineInstanceSpec{
							Domain: v1.DomainSpec{
								Devices: v1.Devices{
									GPUs: []v1.GPU{{Name: "gpu1", DeviceName: "nvidia.com/T4"}},
								},
							},
						},
					},
				},
			}, nil)
			causes := validateWorkerPoolRemoval(context.Background(),
				&v1.KubeVirtSpec{
					WorkerPools: []v1.WorkerPoolConfig{
						{Name: "gpu-pool", VirtLauncherImage: "img", NodeSelector: map[string]string{"gpu": "true"}, Selector: v1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/T4"}}},
					},
				},
				&v1.KubeVirtSpec{},
				client,
			)
			Expect(causes).To(BeEmpty())
		})

		It("should not check VMIs when no pools are removed", func() {
			pool := v1.WorkerPoolConfig{Name: "gpu-pool", VirtLauncherImage: "img", NodeSelector: map[string]string{"gpu": "true"}, Selector: v1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/T4"}}}
			causes := validateWorkerPoolRemoval(context.Background(),
				&v1.KubeVirtSpec{WorkerPools: []v1.WorkerPoolConfig{pool}},
				&v1.KubeVirtSpec{WorkerPools: []v1.WorkerPoolConfig{pool}},
				client,
			)
			Expect(causes).To(BeEmpty())
		})
	})

	Context("warnOverlappingWorkerPools", func() {
		It("should warn when pools share a deviceName", func() {
			warnings := warnOverlappingWorkerPools([]v1.WorkerPoolConfig{
				{Name: "pool-a", NodeSelector: map[string]string{"a": "1"}, Selector: v1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/T4"}}},
				{Name: "pool-b", NodeSelector: map[string]string{"b": "1"}, Selector: v1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/T4"}}},
			})
			Expect(warnings).To(HaveLen(1))
			Expect(warnings[0]).To(ContainSubstring("overlapping deviceName"))
		})

		It("should warn when one pool's vmLabels is a subset of another", func() {
			warnings := warnOverlappingWorkerPools([]v1.WorkerPoolConfig{
				{Name: "pool-a", NodeSelector: map[string]string{"a": "1"}, Selector: v1.WorkerPoolSelector{VMLabels: &v1.WorkerPoolVMLabels{MatchLabels: map[string]string{"env": "prod"}}}},
				{Name: "pool-b", NodeSelector: map[string]string{"b": "1"}, Selector: v1.WorkerPoolSelector{VMLabels: &v1.WorkerPoolVMLabels{MatchLabels: map[string]string{"env": "prod", "tier": "web"}}}},
			})
			Expect(warnings).To(HaveLen(1))
			Expect(warnings[0]).To(ContainSubstring("overlapping vmLabels"))
		})

		It("should warn when pools have identical nodeSelector", func() {
			warnings := warnOverlappingWorkerPools([]v1.WorkerPoolConfig{
				{Name: "pool-a", NodeSelector: map[string]string{"gpu": "true"}, Selector: v1.WorkerPoolSelector{DeviceNames: []string{"dev-a"}}},
				{Name: "pool-b", NodeSelector: map[string]string{"gpu": "true"}, Selector: v1.WorkerPoolSelector{DeviceNames: []string{"dev-b"}}},
			})
			Expect(warnings).To(HaveLen(1))
			Expect(warnings[0]).To(ContainSubstring("identical nodeSelector"))
		})

		It("should warn when one pool's nodeSelector is a subset of another", func() {
			warnings := warnOverlappingWorkerPools([]v1.WorkerPoolConfig{
				{Name: "pool-a", NodeSelector: map[string]string{"gpu": "true"}, Selector: v1.WorkerPoolSelector{DeviceNames: []string{"dev-a"}}},
				{Name: "pool-b", NodeSelector: map[string]string{"gpu": "true", "zone": "a"}, Selector: v1.WorkerPoolSelector{DeviceNames: []string{"dev-b"}}},
			})
			Expect(warnings).To(HaveLen(1))
			Expect(warnings[0]).To(ContainSubstring("overlapping nodeSelector"))
		})

		It("should not warn when pools are disjoint", func() {
			warnings := warnOverlappingWorkerPools([]v1.WorkerPoolConfig{
				{Name: "pool-a", NodeSelector: map[string]string{"a": "1"}, Selector: v1.WorkerPoolSelector{DeviceNames: []string{"dev-a"}}},
				{Name: "pool-b", NodeSelector: map[string]string{"b": "1"}, Selector: v1.WorkerPoolSelector{DeviceNames: []string{"dev-b"}}},
			})
			Expect(warnings).To(BeEmpty())
		})
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
