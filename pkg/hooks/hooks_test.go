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

package hooks_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/hooks"
)

var _ = Describe("HooksAPI", func() {
	Context("test HookSidecarsList structure and helper functions", func() {
		It("by unmarshalling of VM annotations", func() {
			expectedHookSidecarList := hooks.HookSidecarList{
				hooks.HookSidecar{
					Image:           "some-image:v1",
					ImagePullPolicy: "IfNotPresent",
				},
				hooks.HookSidecar{
					Image:           "another-image:v1",
					ImagePullPolicy: "Always",
				},
			}
			vmiHookObject := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						hooks.HookSidecarListAnnotationName: `
[
  {
    "image": "some-image:v1",
    "imagePullPolicy": "IfNotPresent"
  },
  {
    "image": "another-image:v1",
    "imagePullPolicy": "Always"
  }
]
`,
					},
				},
			}
			hookSidecarList, err := hooks.UnmarshalHookSidecarList(vmiHookObject)
			Expect(err).ToNot(HaveOccurred())
			Expect(equality.Semantic.DeepEqual(hookSidecarList, expectedHookSidecarList)).To(BeTrue())
		})
	})
})
