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

package components

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

var _ = Describe("VirtTemplate", func() {
	const testNamespace = "kubevirt-test"

	var testConfig *util.KubeVirtDeploymentConfig

	BeforeEach(func() {
		testConfig = &util.KubeVirtDeploymentConfig{
			Namespace: testNamespace,
		}
	})

	It("should successfully parse virt-template bundle", func() {
		resources, err := NewVirtTemplateResources(testConfig)
		Expect(err).ToNot(HaveOccurred())
		Expect(resources).ToNot(BeNil())

		Expect(resources.CRDs).To(ConsistOf(
			HaveField("Name", "virtualmachinetemplaterequests.template.kubevirt.io"),
			HaveField("Name", "virtualmachinetemplates.template.kubevirt.io"),
		))
		Expect(resources.ServiceAccounts).To(HaveLen(2))
		Expect(resources.Roles).To(HaveLen(1))
		Expect(resources.ClusterRoles).To(HaveLen(10))
		Expect(resources.RoleBindings).To(HaveLen(2))
		Expect(resources.ClusterRoleBindings).To(HaveLen(4))
		Expect(resources.Services).To(HaveLen(3))
		Expect(resources.Deployments).To(HaveLen(2))
		Expect(resources.ValidatingAdmissionPolicies).To(HaveLen(1))
		Expect(resources.ValidatingAdmissionPolicyBindings).To(HaveLen(1))
		Expect(resources.ValidatingWebhookConfigurations).To(HaveLen(1))
		Expect(resources.APIServices).To(HaveLen(1))
		Expect(resources.NetworkPolicies).To(HaveLen(3))
	})

	It("should update namespace correctly", func() {
		resources, err := NewVirtTemplateResources(testConfig)
		Expect(err).ToNot(HaveOccurred())

		for _, sa := range resources.ServiceAccounts {
			Expect(sa.Namespace).To(Equal(testNamespace))
		}

		for _, role := range resources.Roles {
			Expect(role.Namespace).To(Equal(testNamespace))
		}

		for _, rb := range resources.RoleBindings {
			if rb.Name == "virt-template-apiserver-auth-reader" {
				Expect(rb.Namespace).To(Equal("kube-system"))
			} else {
				Expect(rb.Namespace).To(Equal(testNamespace))
			}
			for _, subject := range rb.Subjects {
				Expect(subject.Namespace).To(Equal(testNamespace))
			}
		}

		for _, crb := range resources.ClusterRoleBindings {
			for _, subject := range crb.Subjects {
				Expect(subject.Namespace).To(Equal(testNamespace))
			}
		}

		for _, svc := range resources.Services {
			Expect(svc.Namespace).To(Equal(testNamespace))
		}

		for _, dep := range resources.Deployments {
			Expect(dep.Namespace).To(Equal(testNamespace))
		}

		for _, vwc := range resources.ValidatingWebhookConfigurations {
			for _, webhook := range vwc.Webhooks {
				Expect(webhook.ClientConfig.Service.Namespace).To(Equal(testNamespace))
			}
		}

		for _, apiSvc := range resources.APIServices {
			Expect(apiSvc.Spec.Service.Namespace).To(Equal(testNamespace))
		}

		for _, np := range resources.NetworkPolicies {
			Expect(np.Namespace).To(Equal(testNamespace))
		}
	})

	DescribeTable("should require exactly one replica in deployments",
		func(replicas *int32, expectedMatcher gomegatypes.GomegaMatcher) {
			dep := &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: replicas,
					Template: k8sv1.PodTemplateSpec{
						Spec: k8sv1.PodSpec{
							Containers: []k8sv1.Container{
								{
									Name: "container",
								},
							},
						},
					},
				},
			}
			Expect(updateDeployment(dep, testConfig)).To(expectedMatcher)
		},
		Entry("nil replicas", nil, Succeed()),
		Entry("one replicas", pointer.P(int32(1)), Succeed()),
		Entry("zero replicas", pointer.P(int32(0)), MatchError(ContainSubstring("expected exactly 1 replica"))),
		Entry("two replicas", pointer.P(int32(2)), MatchError(ContainSubstring("expected exactly 1 replica"))),
	)

	DescribeTable("should require exactly one container in deployments",
		func(count int, expectedMatcher gomegatypes.GomegaMatcher) {
			dep := &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Template: k8sv1.PodTemplateSpec{
						Spec: k8sv1.PodSpec{
							Containers: make([]k8sv1.Container, count),
						},
					},
				},
			}

			for i := range count {
				dep.Spec.Template.Spec.Containers[i] = k8sv1.Container{
					Name:  "container",
					Image: "registry.io/image:latest",
				}
			}

			Expect(updateDeployment(dep, testConfig)).To(expectedMatcher)
		},
		Entry("zero containers", 0, MatchError(ContainSubstring("expected exactly 1 container"))),
		Entry("two containers", 2, MatchError(ContainSubstring("expected exactly 1 container"))),
		Entry("one container", 1, Succeed()),
	)

	DescribeTable("should handle product labels on deployments",
		func(productName, productVersion, productComponent string, expectLabels bool) {
			config := &util.KubeVirtDeploymentConfig{
				Namespace: testNamespace,
				AdditionalProperties: map[string]string{
					util.ProductNameKey:      productName,
					util.ProductVersionKey:   productVersion,
					util.ProductComponentKey: productComponent,
				},
			}

			resources, err := NewVirtTemplateResources(config)
			Expect(err).ToNot(HaveOccurred())

			for _, dep := range resources.Deployments {
				if expectLabels {
					Expect(dep.ObjectMeta.Labels).To(HaveKeyWithValue(v1.AppPartOfLabel, productName))
					Expect(dep.ObjectMeta.Labels).To(HaveKeyWithValue(v1.AppVersionLabel, productVersion))
					Expect(dep.ObjectMeta.Labels).To(HaveKeyWithValue(v1.AppComponentLabel, productComponent))
					Expect(dep.Spec.Template.ObjectMeta.Labels).To(HaveKeyWithValue(v1.AppPartOfLabel, productName))
					Expect(dep.Spec.Template.ObjectMeta.Labels).To(HaveKeyWithValue(v1.AppVersionLabel, productVersion))
					Expect(dep.Spec.Template.ObjectMeta.Labels).To(HaveKeyWithValue(v1.AppComponentLabel, productComponent))
				} else {
					Expect(dep.ObjectMeta.Labels).ToNot(HaveKey(v1.AppPartOfLabel))
					Expect(dep.ObjectMeta.Labels).ToNot(HaveKey(v1.AppVersionLabel))
					Expect(dep.ObjectMeta.Labels).ToNot(HaveKey(v1.AppComponentLabel))
					Expect(dep.Spec.Template.ObjectMeta.Labels).ToNot(HaveKey(v1.AppPartOfLabel))
					Expect(dep.Spec.Template.ObjectMeta.Labels).ToNot(HaveKey(v1.AppVersionLabel))
					Expect(dep.Spec.Template.ObjectMeta.Labels).ToNot(HaveKey(v1.AppComponentLabel))
				}
			}
		},
		Entry("empty labels", "", "", "", false),
		Entry("invalid labels", "invalid product name", "-invalid-version", "invalid@component", false),
		Entry("valid labels", "my-product", "1.2.3", "my-component", true),
	)

	It("should set image pull secrets on deployments", func() {
		config := &util.KubeVirtDeploymentConfig{
			Namespace: testNamespace,
			AdditionalProperties: map[string]string{
				util.AdditionalPropertiesPullSecrets: `[{"name": "my-secret"}]`,
			},
		}

		resources, err := NewVirtTemplateResources(config)
		Expect(err).ToNot(HaveOccurred())

		for _, dep := range resources.Deployments {
			Expect(dep.Spec.Template.Spec.ImagePullSecrets).To(HaveLen(1))
			Expect(dep.Spec.Template.Spec.ImagePullSecrets[0].Name).To(Equal("my-secret"))
		}
	})

	It("should update deployment images with custom registry", func() {
		const registry = "my-custom-registry.io/kubevirt"
		config := &util.KubeVirtDeploymentConfig{
			Namespace: testNamespace,
			Registry:  registry,
		}

		resources, err := NewVirtTemplateResources(config)
		Expect(err).ToNot(HaveOccurred())

		for _, dep := range resources.Deployments {
			Expect(dep.Spec.Template.Spec.Containers[0].Image).To(HavePrefix(registry + "/"))
		}
	})

	It("should update deployment images with custom prefix", func() {
		prefix := "myprefix"
		config := &util.KubeVirtDeploymentConfig{
			Namespace:   testNamespace,
			ImagePrefix: prefix,
		}

		resources, err := NewVirtTemplateResources(config)
		Expect(err).ToNot(HaveOccurred())

		for _, dep := range resources.Deployments {
			Expect(dep.Spec.Template.Spec.Containers[0].Image).To(ContainSubstring("/" + prefix))
		}
	})

	It("should set image pull policy on containers", func() {
		config := &util.KubeVirtDeploymentConfig{
			Namespace: testNamespace,
			AdditionalProperties: map[string]string{
				util.AdditionalPropertiesNamePullPolicy: string(k8sv1.PullAlways),
			},
		}

		resources, err := NewVirtTemplateResources(config)
		Expect(err).ToNot(HaveOccurred())

		for _, dep := range resources.Deployments {
			Expect(dep.Spec.Template.Spec.Containers[0].ImagePullPolicy).To(Equal(k8sv1.PullAlways))
		}
	})

	It("should add extra environment variables to containers", func() {
		const (
			envName  = "MY_ENV"
			envValue = "my-value"
		)

		config := &util.KubeVirtDeploymentConfig{
			Namespace: testNamespace,
			PassthroughEnvVars: map[string]string{
				envName: envValue,
			},
		}

		resources, err := NewVirtTemplateResources(config)
		Expect(err).ToNot(HaveOccurred())

		for _, dep := range resources.Deployments {
			found := false
			for _, env := range dep.Spec.Template.Spec.Containers[0].Env {
				if env.Name == envName && env.Value == envValue {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		}
	})

	It("should append verbosity argument to containers", func() {
		resources, err := NewVirtTemplateResources(testConfig)
		Expect(err).ToNot(HaveOccurred())

		for _, dep := range resources.Deployments {
			container := dep.Spec.Template.Spec.Containers[0]
			length := len(container.Args)
			Expect(length).To(BeNumerically(">=", 2))
			Expect(container.Args[length-2]).To(Equal("-v"))
			Expect(container.Args[length-1]).To(Equal("2"))
		}
	})
})
