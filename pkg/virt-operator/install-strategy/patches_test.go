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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package installstrategy

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/log"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
)

var _ = Describe("Patches", func() {
	log.Log.SetIOWriter(GinkgoWriter)

	namespace := "fake-namespace"

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "virt-controller",
		},
		Spec: appsv1.DeploymentSpec{},
	}

	getCustomizer := func() *Customizer {
		c, _ := NewCustomizer(v1.CustomizeComponents{
			Patches: []v1.Patch{
				{
					ResourceName: "virt-controller",
					ResourceType: "Deployment",
					Patch:        `{"metadata":{"labels":{"new-key":"added-this-label"}}}`,
					Type:         v1.StrategicMergePatchType,
				},
			},
		})

		return c
	}

	config := getCustomizer()

	Context("generically apply patches", func() {

		It("should apply to deployments", func() {
			deployments := []*appsv1.Deployment{
				deployment,
			}

			err := config.GenericApplyPatches(deployments)
			Expect(err).ToNot(HaveOccurred())
			Expect(deployment.ObjectMeta.Labels["new-key"]).To(Equal("added-this-label"))

			err = config.GenericApplyPatches([]string{"string"})
			Expect(err).To(HaveOccurred())
		})
	})

	Context("apply patch", func() {

		It("should not error on empty patch", func() {
			err := applyPatch(nil, v1.Patch{})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("get hash", func() {

		It("should be equal", func() {
			c1 := v1.CustomizeComponents{
				Patches: []v1.Patch{
					{
						ResourceName: "virt-controller",
						ResourceType: "Deployment",
						Patch:        `{"metadata":{"labels":{"new-key":"added-this-label"}}}`,
						Type:         v1.StrategicMergePatchType,
					},
					{
						ResourceName: "virt-api",
						ResourceType: "Deployment",
						Patch:        `{"metadata":{"labels":{"my-custom-label":"custom-label"}}}`,
						Type:         v1.StrategicMergePatchType,
					},
					{
						ResourceName: "virt-controller",
						ResourceType: "Deployment",
						Patch:        `{"metadata":{"annotation":{"key":"value"}}}`,
						Type:         v1.StrategicMergePatchType,
					},
				},
			}

			c2 := v1.CustomizeComponents{
				Patches: []v1.Patch{
					{
						ResourceName: "virt-api",
						ResourceType: "Deployment",
						Patch:        `{"metadata":{"labels":{"my-custom-label":"custom-label"}}}`,
						Type:         v1.StrategicMergePatchType,
					},
					{
						ResourceName: "virt-controller",
						ResourceType: "Deployment",
						Patch:        `{"metadata":{"labels":{"new-key":"added-this-label"}}}`,
						Type:         v1.StrategicMergePatchType,
					},
					{
						ResourceName: "virt-controller",
						ResourceType: "Deployment",
						Patch:        `{"metadata":{"annotation":{"key":"value"}}}`,
						Type:         v1.StrategicMergePatchType,
					},
				},
			}

			h1, err := getHash(c1)
			Expect(err).ToNot(HaveOccurred())
			h2, err := getHash(c2)
			Expect(err).ToNot(HaveOccurred())

			Expect(h1).To(Equal(h2))
		})
	})
})
