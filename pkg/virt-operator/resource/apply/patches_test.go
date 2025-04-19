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

package apply

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

var _ = Describe("Patches", func() {

	namespace := "fake-namespace"

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      components.VirtControllerName,
		},
		Spec: appsv1.DeploymentSpec{},
	}

	flags := &v1.Flags{
		Controller: map[string]string{
			"v": "4",
		},
	}

	getCustomizer := func() *Customizer {
		c, _ := NewCustomizer(v1.CustomizeComponents{
			Patches: []v1.CustomizeComponentsPatch{
				{
					ResourceName: components.VirtControllerName,
					ResourceType: "Deployment",
					Patch:        `{"metadata":{"labels":{"new-key":"added-this-label"}}}`,
					Type:         v1.StrategicMergePatchType,
				},
				{
					ResourceName: "*",
					ResourceType: "Deployment",
					Patch:        `{"spec":{"template":{"spec":{"imagePullSecrets":[{"name":"image-pull"}]}}}}`,
					Type:         v1.StrategicMergePatchType,
				},
			},
			Flags: flags,
		})

		return c
	}

	config := getCustomizer()

	Context("generically apply patches", func() {

		It("should apply to deployments", func() {
			deployments := []*appsv1.Deployment{
				deployment,
			}

			fmt.Printf("%+v", config.GetPatches())
			err := config.GenericApplyPatches(deployments)
			Expect(err).ToNot(HaveOccurred())
			Expect(deployment.ObjectMeta.Labels["new-key"]).To(Equal("added-this-label"))
			Expect(deployment.Spec.Template.Spec.ImagePullSecrets[0].Name).To(Equal("image-pull"))
			// check flags are applied
			expectedFlags := []string{components.VirtControllerName}
			expectedFlags = append(expectedFlags, flagsToArray(flags.Controller)...)
			Expect(deployment.Spec.Template.Spec.Containers[0].Command).To(Equal(expectedFlags))

			err = config.GenericApplyPatches([]string{"string"})
			Expect(err).To(HaveOccurred())
		})
	})

	Context("apply patch", func() {

		It("should not error on empty patch", func() {
			err := applyPatch(nil, v1.CustomizeComponentsPatch{})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("get hash", func() {

		c1 := v1.CustomizeComponents{
			Patches: []v1.CustomizeComponentsPatch{
				{
					ResourceName: components.VirtControllerName,
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
					ResourceName: components.VirtControllerName,
					ResourceType: "Deployment",
					Patch:        `{"metadata":{"annotation":{"key":"value"}}}`,
					Type:         v1.StrategicMergePatchType,
				},
			},
		}

		c2 := v1.CustomizeComponents{
			Patches: []v1.CustomizeComponentsPatch{
				{
					ResourceName: "virt-api",
					ResourceType: "Deployment",
					Patch:        `{"metadata":{"labels":{"my-custom-label":"custom-label"}}}`,
					Type:         v1.StrategicMergePatchType,
				},
				{
					ResourceName: components.VirtControllerName,
					ResourceType: "Deployment",
					Patch:        `{"metadata":{"labels":{"new-key":"added-this-label"}}}`,
					Type:         v1.StrategicMergePatchType,
				},
				{
					ResourceName: components.VirtControllerName,
					ResourceType: "Deployment",
					Patch:        `{"metadata":{"annotation":{"key":"value"}}}`,
					Type:         v1.StrategicMergePatchType,
				},
			},
		}

		flags1 := &v1.Flags{
			API: map[string]string{
				"v": "4",
			},
		}

		flags2 := &v1.Flags{
			API: map[string]string{
				"v": "1",
			},
		}

		It("should be equal", func() {
			h1, err := getHash(c1)
			Expect(err).ToNot(HaveOccurred())
			h2, err := getHash(c2)
			Expect(err).ToNot(HaveOccurred())

			Expect(h1).To(Equal(h2))
		})

		It("should not be equal", func() {
			c1.Flags = flags1
			c2.Flags = flags2

			h1, err := getHash(c1)
			Expect(err).ToNot(HaveOccurred())
			h2, err := getHash(c2)
			Expect(err).ToNot(HaveOccurred())

			Expect(h1).ToNot(Equal(h2))
		})
	})

	DescribeTable("valueMatchesKey", func(value, key string, expected bool) {

		matches := valueMatchesKey(value, key)
		Expect(matches).To(Equal(expected))

	},
		Entry("should match wildcard", "*", "Deployment", true),
		Entry("should match with different cases", "deployment", "Deployment", true),
		Entry("should not match", "Service", "Deployment", false),
	)

	Describe("Config controller flags", func() {
		flags := map[string]string{
			"flag-one":  "1",
			"flag":      "3",
			"bool-flag": "",
		}
		resource := "Deployment"

		It("should return flags in the proper format", func() {
			fa := flagsToArray(flags)
			Expect(fa).To(HaveLen(5))

			Expect(strings.Join(fa, " ")).To(ContainSubstring("--flag-one 1"))
			Expect(strings.Join(fa, " ")).To(ContainSubstring("--flag 3"))
			Expect(strings.Join(fa, " ")).To(ContainSubstring("--bool-flag"))
		})

		It("should add flag patch", func() {
			patches := addFlagsPatch(components.VirtAPIName, resource, flags, []v1.CustomizeComponentsPatch{})
			Expect(patches).To(HaveLen(1))
			patch := patches[0]

			Expect(patch.ResourceName).To(Equal(components.VirtAPIName))
			Expect(patch.ResourceType).To(Equal(resource))
		})

		It("should return empty patch", func() {
			patches := addFlagsPatch(components.VirtAPIName, resource, map[string]string{}, []v1.CustomizeComponentsPatch{})
			Expect(patches).To(BeEmpty())
		})

		It("should chain patches", func() {
			patches := addFlagsPatch(components.VirtAPIName, resource, flags, []v1.CustomizeComponentsPatch{})
			Expect(patches).To(HaveLen(1))

			patches = addFlagsPatch(components.VirtControllerName, resource, flags, patches)
			Expect(patches).To(HaveLen(2))
		})

		It("should return all flag patches", func() {
			f := &v1.Flags{
				API: flags,
			}

			patches := flagsToPatches(f)
			Expect(patches).To(HaveLen(1))
		})
	})

})
