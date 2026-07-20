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

package libpodmutator

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	virtLauncherMatchCondition = `has(object.metadata.labels) && "kubevirt.io" in object.metadata.labels && object.metadata.labels["kubevirt.io"] == "virt-launcher"`
	envInjectionPolicyTimeout  = 30 * time.Second
)

var (
	mutatingAdmissionPolicyGVR = schema.GroupVersionResource{
		Group:    "admissionregistration.k8s.io",
		Version:  "v1",
		Resource: "mutatingadmissionpolicies",
	}
	mutatingAdmissionPolicyBindingGVR = schema.GroupVersionResource{
		Group:    "admissionregistration.k8s.io",
		Version:  "v1",
		Resource: "mutatingadmissionpolicybindings",
	}
)

type EnvInjectionPolicy struct {
	PolicyName  string
	BindingName string
}

type EnvInjectionPolicyOptions struct {
	Name          string
	ConfigMapName string
	Namespace     string
}

func (opts EnvInjectionPolicyOptions) namespace() string {
	if opts.Namespace != "" {
		return opts.Namespace
	}
	return testsuite.GetTestNamespace(nil)
}

func (opts EnvInjectionPolicyOptions) bindingName() string {
	return fmt.Sprintf("%s-%s", opts.Name, opts.namespace())
}

func envFromJSONPatchExpression(configMapName string) string {
	computeIdx := `string(object.spec.containers.map(c, c.name).indexOf("compute"))`
	compute := `object.spec.containers.filter(c, c.name == "compute")[0]`
	envFrom := compute + `.?envFrom.orValue([])`
	alreadyHas := fmt.Sprintf(
		`%s.exists(e, has(e.configMapRef) && e.configMapRef.name == %q)`,
		envFrom, configMapName,
	)

	return fmt.Sprintf(
		`!object.spec.containers.exists(c, c.name == "compute") || %s ? [] :`+
			` size(%s) == 0 ?`+
			` [JSONPatch{op: "add", path: "/spec/containers/" + %s + "/envFrom",`+
			` value: [{"configMapRef": {"name": %q}}]}]`+
			` : [JSONPatch{op: "add", path: "/spec/containers/" + %s + "/envFrom/-",`+
			` value: {"configMapRef": {"name": %q}}}]`,
		alreadyHas, envFrom, computeIdx, configMapName, computeIdx, configMapName,
	)
}

// SetupEnvInjectionPolicy creates a MutatingAdmissionPolicy + Binding that injects
// the named ConfigMap as envFrom into the virt-launcher compute container. Unlike the
// webhook-based approach, this runs entirely in the API server.
func SetupEnvInjectionPolicy(opts EnvInjectionPolicyOptions) *EnvInjectionPolicy {
	Expect(opts.Name).ToNot(BeEmpty(), "EnvInjectionPolicyOptions.Name is required")
	Expect(opts.ConfigMapName).ToNot(BeEmpty(), "EnvInjectionPolicyOptions.ConfigMapName is required")

	cleanupStaleEnvInjectionPolicy(opts)

	virtClient := kubevirt.Client()
	testNamespace := opts.namespace()
	dynamic := virtClient.DynamicClient()

	By("Creating MutatingAdmissionPolicy for env injection")
	policy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "admissionregistration.k8s.io/v1",
			"kind":       "MutatingAdmissionPolicy",
			"metadata": map[string]interface{}{
				"name": opts.Name,
			},
			"spec": map[string]interface{}{
				"failurePolicy":      "Fail",
				"reinvocationPolicy": "Never",
				"matchConstraints": map[string]interface{}{
					"resourceRules": []interface{}{
						map[string]interface{}{
							"operations":  []interface{}{"CREATE"},
							"apiGroups":   []interface{}{""},
							"apiVersions": []interface{}{"v1"},
							"resources":   []interface{}{"pods"},
						},
					},
				},
				"matchConditions": []interface{}{
					map[string]interface{}{
						"name":       "is-virt-launcher",
						"expression": virtLauncherMatchCondition,
					},
				},
				"mutations": []interface{}{
					map[string]interface{}{
						"patchType": "JSONPatch",
						"jsonPatch": map[string]interface{}{
							"expression": envFromJSONPatchExpression(opts.ConfigMapName),
						},
					},
				},
			},
		},
	}
	_, err := dynamic.Resource(mutatingAdmissionPolicyGVR).Create(
		context.Background(), policy, metav1.CreateOptions{},
	)
	Expect(err).ToNot(HaveOccurred())

	By("Creating MutatingAdmissionPolicyBinding scoped to test namespace")
	binding := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "admissionregistration.k8s.io/v1",
			"kind":       "MutatingAdmissionPolicyBinding",
			"metadata": map[string]interface{}{
				"name": opts.bindingName(),
			},
			"spec": map[string]interface{}{
				"policyName": opts.Name,
				"matchResources": map[string]interface{}{
					"namespaceSelector": map[string]interface{}{
						"matchLabels": map[string]interface{}{
							"kubernetes.io/metadata.name": testNamespace,
						},
					},
				},
			},
		},
	}
	_, err = dynamic.Resource(mutatingAdmissionPolicyBindingGVR).Create(
		context.Background(), binding, metav1.CreateOptions{},
	)
	Expect(err).ToNot(HaveOccurred())

	waitEnvInjectionPolicyReady(opts)

	return &EnvInjectionPolicy{
		PolicyName:  opts.Name,
		BindingName: opts.bindingName(),
	}
}

// TeardownEnvInjectionPolicy removes resources created by SetupEnvInjectionPolicy.
func TeardownEnvInjectionPolicy(eip *EnvInjectionPolicy) {
	if eip == nil {
		return
	}
	dynamic := kubevirt.Client().DynamicClient()

	if eip.BindingName != "" {
		err := dynamic.Resource(mutatingAdmissionPolicyBindingGVR).Delete(
			context.Background(), eip.BindingName, metav1.DeleteOptions{},
		)
		if !errors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}
		waitEnvInjectionPolicyBindingAbsent(eip.BindingName)
	}
	if eip.PolicyName != "" {
		err := dynamic.Resource(mutatingAdmissionPolicyGVR).Delete(
			context.Background(), eip.PolicyName, metav1.DeleteOptions{},
		)
		if !errors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}
		waitEnvInjectionPolicyAbsent(eip.PolicyName)
	}
}

func cleanupStaleEnvInjectionPolicy(opts EnvInjectionPolicyOptions) {
	dynamic := kubevirt.Client().DynamicClient()

	err := dynamic.Resource(mutatingAdmissionPolicyBindingGVR).Delete(
		context.Background(), opts.bindingName(), metav1.DeleteOptions{},
	)
	if !errors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}
	err = dynamic.Resource(mutatingAdmissionPolicyGVR).Delete(
		context.Background(), opts.Name, metav1.DeleteOptions{},
	)
	if !errors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	waitEnvInjectionPolicyBindingAbsent(opts.bindingName())
	waitEnvInjectionPolicyAbsent(opts.Name)
}

func waitEnvInjectionPolicyReady(opts EnvInjectionPolicyOptions) {
	virtClient := kubevirt.Client()
	testNamespace := opts.namespace()

	By("Waiting for MutatingAdmissionPolicy to become active")
	Eventually(func(g Gomega) {
		probePod := &k8sv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-map-readiness-probe", opts.Name),
				Namespace: testNamespace,
				Labels:    map[string]string{"kubevirt.io": "virt-launcher"},
			},
			Spec: k8sv1.PodSpec{
				Containers: []k8sv1.Container{{
					Name:  "compute",
					Image: "busybox",
					SecurityContext: &k8sv1.SecurityContext{
						AllowPrivilegeEscalation: pointer.P(false),
						RunAsNonRoot:             pointer.P(true),
						RunAsUser:                pointer.P(int64(1000)),
						SeccompProfile:           &k8sv1.SeccompProfile{Type: k8sv1.SeccompProfileTypeRuntimeDefault},
						Capabilities:             &k8sv1.Capabilities{Drop: []k8sv1.Capability{"ALL"}},
					},
				}},
			},
		}
		result, err := virtClient.CoreV1().Pods(testNamespace).Create(
			context.Background(), probePod, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}},
		)
		g.Expect(err).ToNot(HaveOccurred())

		var compute *k8sv1.Container
		for i := range result.Spec.Containers {
			if result.Spec.Containers[i].Name == "compute" {
				compute = &result.Spec.Containers[i]
				break
			}
		}
		g.Expect(compute).ToNot(BeNil(), "probe pod must have a compute container")

		var configMapNames []string
		for _, envFrom := range compute.EnvFrom {
			if envFrom.ConfigMapRef != nil {
				configMapNames = append(configMapNames, envFrom.ConfigMapRef.Name)
			}
		}
		g.Expect(configMapNames).To(ContainElement(opts.ConfigMapName),
			"MAP has not injected envFrom for ConfigMap %q yet", opts.ConfigMapName)
	}).WithTimeout(envInjectionPolicyTimeout).WithPolling(time.Second).Should(Succeed())
}

func waitEnvInjectionPolicyAbsent(name string) {
	dynamic := kubevirt.Client().DynamicClient()
	Eventually(func() bool {
		_, err := dynamic.Resource(mutatingAdmissionPolicyGVR).Get(
			context.Background(), name, metav1.GetOptions{},
		)
		return errors.IsNotFound(err)
	}, envInjectionPolicyTimeout, time.Second).Should(BeTrue())
}

func waitEnvInjectionPolicyBindingAbsent(name string) {
	dynamic := kubevirt.Client().DynamicClient()
	Eventually(func() bool {
		_, err := dynamic.Resource(mutatingAdmissionPolicyBindingGVR).Get(
			context.Background(), name, metav1.GetOptions{},
		)
		return errors.IsNotFound(err)
	}, envInjectionPolicyTimeout, time.Second).Should(BeTrue())
}
