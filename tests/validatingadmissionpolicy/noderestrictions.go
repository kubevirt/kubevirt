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

package validatingadmissionpolicy

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"

	"kubevirt.io/kubevirt/tests/framework/k8s"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	"k8s.io/client-go/rest"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/libnode"
)

const (
	notAllowedLabelPath      = "/metadata/labels/other.io~1notAllowedLabel"
	notAllowedAnnotationPath = "/metadata/annotations/other.io~1notAllowedAnnotation"
	allowedLabelPath         = "/metadata/labels/kubevirt.io~1allowedLabel"
	allowedAnnotationPath    = "/metadata/annotations/kubevirt.io~1allowedAnnotation"
)

var _ = Describe("[sig-compute] virt-handler node restrictions via validatingAdmissionPolicy", decorators.SigCompute, Serial, func() {

	var (
		k8sClient   kubernetes.Interface
		nodeName    string
		anotherNode string
	)

	BeforeEach(func() {
		k8sClient = k8s.Client()
		isValidatingAdmissionPolicyEnabled, err := util.IsValidatingAdmissionPolicyEnabled(kubevirt.Client())
		Expect(err).ToNot(HaveOccurred())
		Expect(isValidatingAdmissionPolicyEnabled).To(BeTrue(), "ValidatingAdmissionPolicy should be enabled")
		_, err = k8sClient.AdmissionregistrationV1().ValidatingAdmissionPolicies().Get(context.Background(), "kubevirt-node-restriction-policy", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred(), "validating admission policy should appear")
		_, err = k8sClient.AdmissionregistrationV1().ValidatingAdmissionPolicyBindings().Get(context.Background(), "kubevirt-node-restriction-binding", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred(), "validating admission policy binding should appear")

		nodesList := libnode.GetAllSchedulableNodes(k8sClient)
		Expect(nodesList.Items).ToNot(BeEmpty())
		nodeName = nodesList.Items[0].Name

		prepareNode(nodeName)

		DeferCleanup(func() { cleanup(nodeName) })
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
			"label update":        {patch.New(patch.WithReplace(notAllowedLabelPath, "other-value")), components.NodeRestrictionErrUpdateLabels},
			"label removal":       {patch.New(patch.WithRemove(notAllowedLabelPath)), components.NodeRestrictionErrAddDeleteLabels},
			"annotation addition": {patch.New(patch.WithAdd("/metadata/annotations/other.io~1newNotAllowedAnnotation", "value")), components.NodeRestrictionErrAddDeleteAnnotations},
			"annotation update":   {patch.New(patch.WithReplace(notAllowedAnnotationPath, "other-value")), components.NodeRestrictionErrUpdateAnnotations},
			"annotation removal":  {patch.New(patch.WithRemove(notAllowedAnnotationPath)), components.NodeRestrictionErrAddDeleteAnnotations},
		}
		node, err := k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		pod, err := libnode.GetVirtHandlerPod(k8sClient, nodeName)
		Expect(err).ToNot(HaveOccurred())

		token, err := exec.ExecuteCommandOnPod(
			pod,
			"virt-handler",
			[]string{"cat",
				"/var/run/secrets/kubernetes.io/serviceaccount/token",
			},
		)
		Expect(err).ToNot(HaveOccurred())

		handlerClient, err := kubecli.GetK8sClientFromRESTConfig(&rest.Config{
			Host: kubevirt.Client().Config().Host,
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: true,
			},
			BearerToken: token,
		})
		Expect(err).ToNot(HaveOccurred())

		for description, patchItem := range patchSetList {
			nodePatch, err := patchItem.patchSet.GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			_, err = handlerClient.CoreV1().Nodes().Patch(context.Background(), node.Name, types.JSONPatchType, nodePatch, metav1.PatchOptions{})
			Expect(err).To(HaveOccurred(), fmt.Sprintf("%s should fail on node specific node restriction", description))
			Expect(err).To(MatchError(errors.IsForbidden, "k8serrors.IsForbidden"))
			Expect(err.Error()).To(ContainSubstring(patchItem.expectedError), fmt.Sprintf("%s should match specific error", description))
		}
	})

	It("allow kubevirt related patches to node", func() {
		patchSetList := map[string]*patch.PatchSet{
			"kubevirt.io label addition":      patch.New(patch.WithAdd("/metadata/labels/kubevirt.io~1newAllowedLabel", "value")),
			"kubevirt.io label update":        patch.New(patch.WithReplace(allowedLabelPath, "other-value")),
			"kubevirt.io label removal":       patch.New(patch.WithRemove(allowedLabelPath)),
			"kubevirt.io annotation addition": patch.New(patch.WithAdd("/metadata/annotations/kubevirt.io~1newAllowedAnnotation", "value")),
			"kubevirt.io annotation update":   patch.New(patch.WithReplace(allowedAnnotationPath, "other-value")),
			"kubevirt.io annotation removal":  patch.New(patch.WithRemove(allowedAnnotationPath)),
		}
		node, err := k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		pod, err := libnode.GetVirtHandlerPod(k8sClient, node.Name)
		Expect(err).ToNot(HaveOccurred())

		token, err := exec.ExecuteCommandOnPod(
			pod,
			"virt-handler",
			[]string{"cat",
				"/var/run/secrets/kubernetes.io/serviceaccount/token",
			},
		)

		Expect(err).ToNot(HaveOccurred())
		handlerClient, err := kubecli.GetK8sClientFromRESTConfig(&rest.Config{
			Host: kubevirt.Client().Config().Host,
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: true,
			},
			BearerToken: token,
		})
		Expect(err).ToNot(HaveOccurred())

		for description, patchSet := range patchSetList {
			nodePatch, err := patchSet.GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			_, err = handlerClient.CoreV1().Nodes().Patch(context.Background(), node.Name, types.JSONPatchType, nodePatch, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("%s should not fail on node specific node restriction", description))
		}
	})

	Context("patching another node", func() {
		BeforeEach(func() {
			nodesList := libnode.GetAllSchedulableNodes(k8sClient)
			Expect(nodesList.Items).ToNot(BeEmpty())
			Expect(len(nodesList.Items)).To(BeNumerically(">", 1))
			for _, node := range nodesList.Items {
				if nodeName != node.Name {
					anotherNode = node.Name
					break
				}
			}

			prepareNode(anotherNode)
			DeferCleanup(func() { cleanup(anotherNode) })
		})

		It("rejects kubevirt related patches", func() {
			patchSetList := map[string]*patch.PatchSet{
				"kubevirt.io label addition":      patch.New(patch.WithAdd(allowedLabelPath+"new", "value")),
				"kubevirt.io label update":        patch.New(patch.WithReplace(allowedLabelPath, "other-value")),
				"kubevirt.io label removal":       patch.New(patch.WithRemove(allowedLabelPath)),
				"kubevirt.io annotation addition": patch.New(patch.WithAdd(allowedAnnotationPath+"new", "value")),
				"kubevirt.io annotation update":   patch.New(patch.WithReplace(allowedAnnotationPath, "other-value")),
				"kubevirt.io annotation removal":  patch.New(patch.WithRemove(allowedAnnotationPath)),
			}
			node, err := k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			pod, err := libnode.GetVirtHandlerPod(k8sClient, node.Name)
			Expect(err).ToNot(HaveOccurred())

			token, err := exec.ExecuteCommandOnPod(
				pod,
				"virt-handler",
				[]string{"cat",
					"/var/run/secrets/kubernetes.io/serviceaccount/token",
				},
			)

			Expect(err).ToNot(HaveOccurred())
			handlerClient, err := kubecli.GetK8sClientFromRESTConfig(&rest.Config{
				Host: kubevirt.Client().Config().Host,
				TLSClientConfig: rest.TLSClientConfig{
					Insecure: true,
				},
				BearerToken: token,
			})
			Expect(err).ToNot(HaveOccurred())

			otherNode, err := k8sClient.CoreV1().Nodes().Get(context.Background(), anotherNode, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			for description, patchSet := range patchSetList {
				nodePatch, err := patchSet.GeneratePayload()
				Expect(err).ToNot(HaveOccurred())
				_, err = handlerClient.CoreV1().Nodes().Patch(context.Background(), otherNode.Name, types.JSONPatchType, nodePatch, metav1.PatchOptions{})
				Expect(err).To(MatchError(ContainSubstring("this user cannot modify this node")), fmt.Sprintf("%s should fail on node specific node restriction", description))
			}
		})
	})

})

func prepareNode(name string) {
	patchBytes, err := patch.New(
		patch.WithAdd(notAllowedLabelPath, "value"),
		patch.WithAdd(notAllowedAnnotationPath, "value"),
		patch.WithAdd(allowedLabelPath, "value"),
		patch.WithAdd(allowedAnnotationPath, "value"),
	).GeneratePayload()
	Expect(err).ToNot(HaveOccurred())

	_, err = k8s.Client().CoreV1().Nodes().Patch(context.Background(), name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	Expect(err).ToNot(HaveOccurred())
}

func cleanup(nodeName string) {
	node, err := k8s.Client().CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

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

	_, err = k8s.Client().CoreV1().Nodes().Patch(
		context.Background(), node.Name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	Expect(err).ToNot(HaveOccurred())
}
