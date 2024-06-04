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

package validatingaddmisionpolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	"k8s.io/client-go/rest"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnode"
)

var _ = Describe("[Serial][sig-compute] virt-handler node restrictions via validatingAdmissionPolicy", decorators.SigCompute, decorators.Kubernetes130, Serial, func() {

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		tests.EnableFeatureGate(virtconfig.NodeRestrictionGate)
		isValidatingAdmissionPolicyEnabled, err := util.IsValidatingAdmissionPolicyEnabled(virtClient)
		Expect(err).ToNot(HaveOccurred())
		Expect(isValidatingAdmissionPolicyEnabled).To(BeTrue(), "ValidatingAdmissionPolicy should be enabled")
		Eventually(func(g Gomega) {
			_, err := virtClient.AdmissionregistrationV1().ValidatingAdmissionPolicies().Get(context.Background(), "kubevirt-node-restriction-policy", metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred(), "validating admission policy should appear")
			_, err = virtClient.AdmissionregistrationV1().ValidatingAdmissionPolicyBindings().Get(context.Background(), "kubevirt-node-restriction-binding", metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred(), "validating admission policy binding should appear")
		}).WithPolling(5 * time.Second).WithTimeout(360 * time.Second).Should(Succeed())

		nodesList, err := virtClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())

		for _, node := range nodesList.Items {
			patchBytes, err := patch.New(
				patch.WithAdd("/metadata/labels/other.io~1notAllowedLabel", "value"),
				patch.WithAdd("/metadata/annotations/other.io~1notAllowedAnnotation", "value"),
				patch.WithAdd("/metadata/labels/kubevirt.io~1allowedLabel", "value"),
				patch.WithAdd("/metadata/annotations/kubevirt.io~1allowedAnnotation", "value"),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			_, err = virtClient.CoreV1().Nodes().Patch(context.Background(), node.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
		}
	})

	AfterEach(func() {
		nodesList, err := virtClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())

		for _, node := range nodesList.Items {
			old, err := json.Marshal(node)
			Expect(err).ToNot(HaveOccurred())
			newNode := node.DeepCopy()
			delete(newNode.Labels, "other.io/notAllowedLabel")
			delete(newNode.Annotations, "other.io/notAllowedAnnotation")
			delete(newNode.Labels, "kubevirt.io/allowedLabel")
			delete(newNode.Annotations, "kubevirt.io/allowedAnnotation")

			newJSON, err := json.Marshal(newNode)
			Expect(err).ToNot(HaveOccurred())

			patchBytes, err := strategicpatch.CreateTwoWayMergePatch(old, newJSON, node)
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.CoreV1().Nodes().Patch(
				context.Background(), node.Name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		tests.DisableFeatureGate(virtconfig.NodeRestrictionGate)
		Eventually(func(g Gomega) {
			_, err := virtClient.AdmissionregistrationV1().ValidatingAdmissionPolicies().Get(context.Background(), "kubevirt-node-restriction-policy", metav1.GetOptions{})
			g.Expect(err).To(MatchError(k8serrors.IsNotFound, "IsNotFound"), "validating admission policy should disappear")
			_, err = virtClient.AdmissionregistrationV1().ValidatingAdmissionPolicyBindings().Get(context.Background(), "kubevirt-node-restriction-binding", metav1.GetOptions{})
			g.Expect(err).To(MatchError(k8serrors.IsNotFound, "IsNotFound"), "validating admission policy binding should disappear")
		}).WithPolling(5 * time.Second).WithTimeout(360 * time.Second).Should(Succeed())
	})

	type testPatchMap struct {
		patchSet      *patch.PatchSet
		expectedError string
	}

	It("reject not allowed patches to node", func() {
		patchSetList := map[string]testPatchMap{
			"patch spec":          {patch.New(patch.WithAdd("/spec/unschedulable", true)), components.NodeRestrictionErrModifySpec},
			"metadata patch":      {patch.New(patch.WithAdd("/metadata/finalizers", []string{"kubernetes.io/evil-finalizer"})), components.NodeRestrictionErrChangeMetadataFields},
			"label addition":      {patch.New(patch.WithAdd("/metadata/labels/other.io~1newNotAllowedLabel", "value")), components.NodeRestrictionErrAddDeleteLabels},
			"label update":        {patch.New(patch.WithReplace("/metadata/labels/other.io~1notAllowedLabel", "other-value")), components.NodeRestrictionErrUpdateLabels},
			"label removal":       {patch.New(patch.WithRemove("/metadata/labels/other.io~1notAllowedLabel")), components.NodeRestrictionErrAddDeleteLabels},
			"annotation addition": {patch.New(patch.WithAdd("/metadata/annotations/other.io~1newNotAllowedAnnotation", "value")), components.NodeRestrictionErrAddDeleteAnnotations},
			"annotation update":   {patch.New(patch.WithReplace("/metadata/annotations/other.io~1notAllowedAnnotation", "other-value")), components.NodeRestrictionErrUpdateAnnotations},
			"annotation removal":  {patch.New(patch.WithRemove("/metadata/annotations/other.io~1notAllowedAnnotation")), components.NodeRestrictionErrAddDeleteAnnotations},
		}
		nodesList, err := virtClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(nodesList.Items).ToNot(BeEmpty())
		node := nodesList.Items[0]
		// give virt-handler time to be reconciled
		Eventually(func(g Gomega) {
			pod, err := libnode.GetVirtHandlerPod(virtClient, node.Name)
			Expect(err).ToNot(HaveOccurred())
			_, err = exec.ExecuteCommandOnPod(
				pod,
				"virt-handler",
				[]string{"cat",
					"/var/run/secrets/kubernetes.io/serviceaccount/token",
				},
			)
			g.Expect(err).ToNot(HaveOccurred())
		}).WithPolling(5 * time.Second).WithTimeout(240 * time.Second).Should(Succeed())

		pod, err := libnode.GetVirtHandlerPod(virtClient, node.Name)
		Expect(err).ToNot(HaveOccurred())

		token, err := exec.ExecuteCommandOnPod(
			pod,
			"virt-handler",
			[]string{"cat",
				"/var/run/secrets/kubernetes.io/serviceaccount/token",
			},
		)

		Expect(err).ToNot(HaveOccurred())
		handlerClient, err := kubecli.GetKubevirtClientFromRESTConfig(&rest.Config{
			Host: virtClient.Config().Host,
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: true,
			},
			BearerToken: token,
		})
		Expect(err).ToNot(HaveOccurred())

		for description, patchItem := range patchSetList {
			nodePatch, err := patchItem.patchSet.GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() error {
				_, err := handlerClient.CoreV1().Nodes().Patch(context.TODO(), node.Name, types.JSONPatchType, nodePatch, metav1.PatchOptions{})
				return err
			}, 10*time.Second, time.Second).Should(MatchError(
				ContainSubstring(patchItem.expectedError), fmt.Sprintf("%s should fail on node specific node restriction", description),
			))
		}
	})

	It("allow kubevirt related patches to node", func() {
		patchSetList := map[string]*patch.PatchSet{
			"kubevirt.io label addition":      patch.New(patch.WithAdd("/metadata/labels/kubevirt.io~1newAllowedLabel", "value")),
			"kubevirt.io label update":        patch.New(patch.WithReplace("/metadata/labels/kubevirt.io~1allowedLabel", "other-value")),
			"kubevirt.io label removal":       patch.New(patch.WithRemove("/metadata/labels/kubevirt.io~1allowedLabel")),
			"kubevirt.io annotation addition": patch.New(patch.WithAdd("/metadata/annotations/kubevirt.io~1newAllowedAnnotation", "value")),
			"kubevirt.io annotation update":   patch.New(patch.WithReplace("/metadata/annotations/kubevirt.io~1allowedAnnotation", "other-value")),
			"kubevirt.io annotation removal":  patch.New(patch.WithRemove("/metadata/annotations/kubevirt.io~1allowedAnnotation")),
		}
		nodesList, err := virtClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(nodesList.Items).ToNot(BeEmpty())
		node := nodesList.Items[0]
		// give virt-handler time to be reconciled
		Eventually(func(g Gomega) {
			pod, err := libnode.GetVirtHandlerPod(virtClient, node.Name)
			Expect(err).ToNot(HaveOccurred())
			_, err = exec.ExecuteCommandOnPod(
				pod,
				"virt-handler",
				[]string{"cat",
					"/var/run/secrets/kubernetes.io/serviceaccount/token",
				},
			)
			g.Expect(err).ToNot(HaveOccurred())
		}).WithPolling(5 * time.Second).WithTimeout(240 * time.Second).Should(Succeed())

		pod, err := libnode.GetVirtHandlerPod(virtClient, node.Name)
		Expect(err).ToNot(HaveOccurred())

		token, err := exec.ExecuteCommandOnPod(
			pod,
			"virt-handler",
			[]string{"cat",
				"/var/run/secrets/kubernetes.io/serviceaccount/token",
			},
		)

		Expect(err).ToNot(HaveOccurred())
		handlerClient, err := kubecli.GetKubevirtClientFromRESTConfig(&rest.Config{
			Host: virtClient.Config().Host,
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: true,
			},
			BearerToken: token,
		})
		Expect(err).ToNot(HaveOccurred())

		for description, patchSet := range patchSetList {
			nodePatch, err := patchSet.GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() error {
				_, err := handlerClient.CoreV1().Nodes().Patch(context.TODO(), node.Name, types.JSONPatchType, nodePatch, metav1.PatchOptions{})
				return err
			}, 10*time.Second, time.Second).Should(Not(HaveOccurred()), fmt.Sprintf("%s should not fail on node specific node restriction", description))
		}
	})
})
