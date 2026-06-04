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

package mutators

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/migrations"
	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("MigrationPolicy Mutator", func() {
	var mutator *MigrationPolicyMutator
	var clusterConfig *virtconfig.ClusterConfig

	BeforeEach(func() {
		clusterConfig, _, _ = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
		mutator = &MigrationPolicyMutator{ClusterConfig: clusterConfig}
	})

	It("should clear experimental section if feature gate is disabled", func() {
		policy := &migrationsv1.MigrationPolicy{
			Spec: migrationsv1.MigrationPolicySpec{
				VMMigrationConfiguration: v1.VMMigrationConfiguration{
					AdvancedMigrationOptions: &v1.AdvancedMigrationOptions{
						StallDetector: &v1.StallDetectorOptions{
							StallMargin: pointer.P(float64(0.04)),
						},
					},
				},
			},
		}

		policyBytes, _ := json.Marshal(policy)
		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Operation: admissionv1.Create,
				Resource: metav1.GroupVersionResource{
					Group:    migrationsv1.MigrationPolicyKind.Group,
					Resource: migrations.ResourceMigrationPolicies,
				},
				Object: runtime.RawExtension{
					Raw: policyBytes,
				},
			},
		}

		resp := mutator.Mutate(ar)
		Expect(resp.Allowed).To(BeTrue())
		Expect(string(resp.Patch)).ToNot(ContainSubstring(`"experimental"`))
	})

	It("should not clear experimental section if feature gate is enabled", func() {
		clusterConfig, _, _ = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			DeveloperConfiguration: &v1.DeveloperConfiguration{
				FeatureGates: []string{featuregate.AdvancedMigrationOptions},
			},
		})
		mutator.ClusterConfig = clusterConfig

		policy := &migrationsv1.MigrationPolicy{
			Spec: migrationsv1.MigrationPolicySpec{
				VMMigrationConfiguration: v1.VMMigrationConfiguration{
					AdvancedMigrationOptions: &v1.AdvancedMigrationOptions{
						StallDetector: &v1.StallDetectorOptions{
							StallMargin: pointer.P(float64(0.04)),
						},
					},
				},
			},
		}

		policyBytes, _ := json.Marshal(policy)
		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Operation: admissionv1.Create,
				Resource: metav1.GroupVersionResource{
					Group:    migrationsv1.MigrationPolicyKind.Group,
					Resource: migrations.ResourceMigrationPolicies,
				},
				Object: runtime.RawExtension{
					Raw: policyBytes,
				},
			},
		}

		resp := mutator.Mutate(ar)
		Expect(resp.Allowed).To(BeTrue())
		Expect(string(resp.Patch)).ToNot(ContainSubstring(`"experimental":null`))
	})

	It("should preserve existing experimental section on update if feature gate is disabled", func() {
		oldPolicy := &migrationsv1.MigrationPolicy{
			Spec: migrationsv1.MigrationPolicySpec{
				VMMigrationConfiguration: v1.VMMigrationConfiguration{
					AdvancedMigrationOptions: &v1.AdvancedMigrationOptions{
						StallDetector: &v1.StallDetectorOptions{
							StallMargin: pointer.P(float64(0.04)),
						},
					},
				},
			},
		}
		oldPolicyBytes, _ := json.Marshal(oldPolicy)

		policy := &migrationsv1.MigrationPolicy{
			Spec: migrationsv1.MigrationPolicySpec{
				VMMigrationConfiguration: v1.VMMigrationConfiguration{
					AdvancedMigrationOptions: &v1.AdvancedMigrationOptions{
						StallDetector: &v1.StallDetectorOptions{
							StallMargin: pointer.P(float64(0.05)),
						},
					},
				},
			},
		}
		policyBytes, _ := json.Marshal(policy)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Operation: admissionv1.Update,
				Resource: metav1.GroupVersionResource{
					Group:    migrationsv1.MigrationPolicyKind.Group,
					Resource: migrations.ResourceMigrationPolicies,
				},
				Object: runtime.RawExtension{
					Raw: policyBytes,
				},
				OldObject: runtime.RawExtension{
					Raw: oldPolicyBytes,
				},
			},
		}

		resp := mutator.Mutate(ar)
		Expect(resp.Allowed).To(BeTrue())
		// Should revert to old value
		Expect(string(resp.Patch)).To(ContainSubstring(`"stallMargin":0.04`))
	})
})
