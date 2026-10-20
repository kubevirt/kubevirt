/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package controlplane_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	vap "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components/validatingadmissionpolicies"
	"kubevirt.io/kubevirt/tests/integration/framework"
)

const (
	nodeRestrictionUser = "system:serviceaccount:kubevirt:kubevirt-virt-handler"
	ownedNodeName       = "node-restriction-owned"
	otherNodeName       = "node-restriction-other"
	otherUser           = "node-restriction-other-user"

	notAllowedLabelPath      = "/metadata/labels/other.io~1notAllowedLabel"
	notAllowedAnnotationPath = "/metadata/annotations/other.io~1notAllowedAnnotation"
	allowedLabelPath         = "/metadata/labels/kubevirt.io~1allowedLabel"
	allowedAnnotationPath    = "/metadata/annotations/kubevirt.io~1allowedAnnotation"
)

var _ = Describe("CEL node restrictions", Ordered, ContinueOnFailure, func() {
	var f *framework.Framework
	var handlerClient kubernetes.Interface

	BeforeAll(func(ctx SpecContext) {
		f = framework.New()
		f.Start(context.Background())
		handlerClient = newNodeRestrictionClient(ctx, f)
		installNodeRestrictionPolicy(ctx, f)
	})

	BeforeEach(func(ctx SpecContext) {
		for _, name := range []string{ownedNodeName, otherNodeName} {
			_, err := f.K8sClient().CoreV1().Nodes().Create(ctx, &k8sv1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					Labels: map[string]string{
						"other.io/notAllowedLabel": "value",
						"kubevirt.io/allowedLabel": "value",
					},
					Annotations: map[string]string{
						"other.io/notAllowedAnnotation": "value",
						"kubevirt.io/allowedAnnotation": "value",
					},
				},
			}, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func(ctx SpecContext) {
				Expect(f.K8sClient().CoreV1().Nodes().Delete(ctx, name, metav1.DeleteOptions{})).To(Succeed())
			})
		}

		By("waiting for CEL admission to reject a dry-run spec change")
		Eventually(ctx, func(g Gomega) {
			_, err := patchNode(ctx, handlerClient, ownedNodeName,
				patch.WithAdd("/spec/unschedulable", true), metav1.PatchOptions{DryRun: []string{metav1.DryRunAll}})
			g.Expect(err).To(MatchError(errors.IsForbidden, "IsForbidden"))
			g.Expect(err).To(MatchError(ContainSubstring(vap.NodeRestrictionErrModifySpec)))
		}, time.Minute, 100*time.Millisecond).Should(Succeed())
	})

	DescribeTable("should reject forbidden changes to the handler's own node",
		func(ctx SpecContext, change patch.PatchOption, expectedError string) {
			_, err := patchNode(ctx, handlerClient, ownedNodeName, change, metav1.PatchOptions{})
			Expect(err).To(MatchError(errors.IsForbidden, "IsForbidden"))
			Expect(err).To(MatchError(ContainSubstring(expectedError)))
		},
		Entry("spec change", patch.WithAdd("/spec/unschedulable", true), vap.NodeRestrictionErrModifySpec),
		Entry("metadata change", patch.WithAdd("/metadata/finalizers", []string{"kubernetes.io/evil-finalizer"}),
			vap.NodeRestrictionErrChangeMetadataFields),
		Entry("label addition", patch.WithAdd("/metadata/labels/other.io~1newNotAllowedLabel", "value"),
			vap.NodeRestrictionErrAddDeleteLabels),
		Entry("label update", patch.WithReplace(notAllowedLabelPath, "other-value"), vap.NodeRestrictionErrUpdateLabels),
		Entry("label removal", patch.WithRemove(notAllowedLabelPath), vap.NodeRestrictionErrAddDeleteLabels),
		Entry("annotation addition", patch.WithAdd("/metadata/annotations/other.io~1newNotAllowedAnnotation", "value"),
			vap.NodeRestrictionErrAddDeleteAnnotations),
		Entry("annotation update", patch.WithReplace(notAllowedAnnotationPath, "other-value"), vap.NodeRestrictionErrUpdateAnnotations),
		Entry("annotation removal", patch.WithRemove(notAllowedAnnotationPath), vap.NodeRestrictionErrAddDeleteAnnotations),
	)

	It("should allow KubeVirt metadata changes to the handler's own node", func(ctx SpecContext) {
		for _, change := range kubeVirtNodePatches() {
			By(change.description)
			_, err := patchNode(ctx, handlerClient, ownedNodeName, change.option, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())
		}
	})

	DescribeTable("should reject KubeVirt metadata changes to another node",
		func(ctx SpecContext, change patch.PatchOption) {
			_, err := patchNode(ctx, handlerClient, otherNodeName, change, metav1.PatchOptions{})
			Expect(err).To(MatchError(errors.IsForbidden, "IsForbidden"))
			Expect(err).To(MatchError(ContainSubstring(vap.NodeRestrictionErrModifyAnother)))
		},
		Entry("label addition", patch.WithAdd(allowedLabelPath+"new", "value")),
		Entry("label update", patch.WithReplace(allowedLabelPath, "other-value")),
		Entry("label removal", patch.WithRemove(allowedLabelPath)),
		Entry("annotation addition", patch.WithAdd(allowedAnnotationPath+"new", "value")),
		Entry("annotation update", patch.WithReplace(allowedAnnotationPath, "other-value")),
		Entry("annotation removal", patch.WithRemove(allowedAnnotationPath)),
		Entry("cpumanager label addition", patch.WithAdd("/metadata/labels/cpumanager", "true")),
	)

	DescribeTable("should reject a handler request without the node-name extra",
		func(ctx SpecContext, extra map[string][]string) {
			client := impersonateNodeRestrictionUser(f, nodeRestrictionUser, extra)
			_, err := patchNode(ctx, client, ownedNodeName, patch.WithAdd(allowedLabelPath+"new", "value"), metav1.PatchOptions{})
			Expect(err).To(MatchError(errors.IsForbidden, "IsForbidden"))
			Expect(err).To(MatchError(ContainSubstring(vap.NodeRestrictionErrModifyAnother)))
		},
		Entry("without any extras", nil),
		Entry("with other pod-bound token information",
			map[string][]string{"authentication.kubernetes.io/pod-name": {"virt-handler"}}),
	)

	It("should allow a non-handler user to change a node's spec", func(ctx SpecContext) {
		client := impersonateNodeRestrictionUser(f, otherUser, nil)
		_, err := patchNode(ctx, client, ownedNodeName, patch.WithAdd("/spec/unschedulable", true), metav1.PatchOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should allow adding, updating and removing the cpumanager label on the handler's node", func(ctx SpecContext) {
		for _, change := range []patch.PatchOption{
			patch.WithAdd("/metadata/labels/cpumanager", "true"),
			patch.WithReplace("/metadata/labels/cpumanager", "false"),
			patch.WithRemove("/metadata/labels/cpumanager"),
		} {
			_, err := patchNode(ctx, handlerClient, ownedNodeName, change, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())
		}
	})
})

func newNodeRestrictionClient(ctx context.Context, f *framework.Framework) kubernetes.Interface {
	GinkgoHelper()
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: "node-restriction-handler"},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"nodes"},
			Verbs:     []string{"patch"},
		}},
	}
	_, err := f.K8sClient().RbacV1().ClusterRoles().Create(ctx, role, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	_, err = f.K8sClient().RbacV1().ClusterRoleBindings().Create(ctx, &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: role.Name},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     role.Name,
		},
		Subjects: []rbacv1.Subject{
			{APIGroup: rbacv1.GroupName, Kind: "User", Name: nodeRestrictionUser},
			{APIGroup: rbacv1.GroupName, Kind: "User", Name: otherUser},
		},
	}, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	// Supply the user information normally carried by a pod-bound handler token.
	return impersonateNodeRestrictionUser(f, nodeRestrictionUser,
		map[string][]string{"authentication.kubernetes.io/node-name": {ownedNodeName}})
}

func impersonateNodeRestrictionUser(f *framework.Framework, user string, extra map[string][]string) kubernetes.Interface {
	GinkgoHelper()
	cfg := rest.CopyConfig(f.VirtClient().Config())
	cfg.Impersonate = rest.ImpersonationConfig{UserName: user, Extra: extra}
	client, err := kubernetes.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())
	return client
}

func installNodeRestrictionPolicy(ctx context.Context, f *framework.Framework) {
	GinkgoHelper()
	client := f.K8sClient().AdmissionregistrationV1()
	_, err := client.ValidatingAdmissionPolicies().Create(ctx,
		vap.NewHandlerV1ValidatingAdmissionPolicy(nodeRestrictionUser), metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	_, err = client.ValidatingAdmissionPolicyBindings().Create(ctx,
		vap.NewHandlerV1ValidatingAdmissionPolicyBinding(), metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
}

type nodeRestrictionPatch struct {
	description string
	option      patch.PatchOption
}

// Keep updates before removals so every JSON Patch target exists when used.
func kubeVirtNodePatches() []nodeRestrictionPatch {
	return []nodeRestrictionPatch{
		{"KubeVirt label addition", patch.WithAdd(allowedLabelPath+"new", "value")},
		{"KubeVirt label update", patch.WithReplace(allowedLabelPath, "other-value")},
		{"KubeVirt label removal", patch.WithRemove(allowedLabelPath)},
		{"KubeVirt annotation addition", patch.WithAdd(allowedAnnotationPath+"new", "value")},
		{"KubeVirt annotation update", patch.WithReplace(allowedAnnotationPath, "other-value")},
		{"KubeVirt annotation removal", patch.WithRemove(allowedAnnotationPath)},
	}
}

func patchNode(
	ctx context.Context,
	client kubernetes.Interface,
	name string,
	change patch.PatchOption,
	options metav1.PatchOptions,
) (*k8sv1.Node, error) {
	GinkgoHelper()
	payload, err := patch.New(change).GeneratePayload()
	Expect(err).NotTo(HaveOccurred())
	return client.CoreV1().Nodes().Patch(ctx, name, types.JSONPatchType, payload, options)
}
