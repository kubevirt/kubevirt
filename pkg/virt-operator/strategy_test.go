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

package virt_operator

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/apply"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	operatorutil "kubevirt.io/kubevirt/pkg/virt-operator/util"
)

var _ = Describe("Install Strategy Cache Idempotency", func() {
	It("should not mutate cached strategy across multiple patch reconciliations", func() {
		controller := &KubeVirtController{}
		kv := &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "test-kubevirt",
				Namespace:  "kubevirt",
				Generation: 1,
			},
			Spec: v1.KubeVirtSpec{
				ImageRegistry: "quay.io/kubevirt",
				ImageTag:      "v1.0.0",
				CustomizeComponents: v1.CustomizeComponents{
					Patches: []v1.CustomizeComponentsPatch{
						{
							ResourceName: components.VirtHandlerName,
							ResourceType: "DaemonSet",
							Type:         v1.JSONPatchType,
							Patch: `[
								{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--custom-flag"}
							]`,
						},
					},
				},
			},
		}

		config := operatorutil.GetTargetConfigFromKV(kv)
		baseStrategy, err := install.GenerateCurrentInstallStrategy(config, "", "kubevirt")
		Expect(err).ToNot(HaveOccurred())

		// Cache the generated strategy
		controller.cacheInstallStrategy(baseStrategy, config, kv.Generation)

		// First reconciliation: fetch from cache and apply patches
		cached1, ok := controller.getCachedInstallStrategy(config, kv.Generation)
		Expect(ok).To(BeTrue())
		Expect(cached1).ToNot(BeNil())

		customizer1, err := apply.NewCustomizer(kv.Spec.CustomizeComponents)
		Expect(err).ToNot(HaveOccurred())
		Expect(customizer1.Apply(cached1)).To(Succeed())

		// Second reconciliation: fetch from cache again and apply patches
		cached2, ok := controller.getCachedInstallStrategy(config, kv.Generation)
		Expect(ok).To(BeTrue())
		Expect(cached2).ToNot(BeNil())

		customizer2, err := apply.NewCustomizer(kv.Spec.CustomizeComponents)
		Expect(err).ToNot(HaveOccurred())
		Expect(customizer2.Apply(cached2)).To(Succeed())

		// Count occurrences of "--custom-flag" in cached2 virt-handler daemonSet
		var customFlagCount int
		for _, ds := range cached2.DaemonSets() {
			if ds.Name == components.VirtHandlerName {
				for _, arg := range ds.Spec.Template.Spec.Containers[0].Args {
					if arg == "--custom-flag" {
						customFlagCount++
					}
				}
			}
		}
		Expect(customFlagCount).To(Equal(1), "custom flag should appear exactly once, proving patch application is idempotent")
	})
})
