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
 */

package webhooks

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8stesting "k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

var _ = Describe("Webhook", func() {
	var admitter *KubeVirtDeletionAdmitter
	var fakeClient *kubevirtfake.Clientset

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		kv := &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "test",
			},
			Status: v1.KubeVirtStatus{Phase: v1.KubeVirtPhaseDeployed},
		}
		fakeClient = kubevirtfake.NewSimpleClientset(kv)
		kubeCli := kubecli.NewMockKubevirtClient(ctrl)
		admitter = &KubeVirtDeletionAdmitter{kubeCli}
		kubeCli.
			EXPECT().
			KubeVirt("test").
			Return(fakeClient.KubevirtV1().KubeVirts("test")).
			AnyTimes()
		kubeCli.
			EXPECT().
			ReplicaSet(k8sv1.NamespaceAll).
			Return(fakeClient.KubevirtV1().VirtualMachineInstanceReplicaSets(k8sv1.NamespaceAll)).
			AnyTimes()
		kubeCli.
			EXPECT().
			VirtualMachineInstance(k8sv1.NamespaceAll).
			Return(fakeClient.KubevirtV1().VirtualMachineInstances(k8sv1.NamespaceAll)).
			AnyTimes()
		kubeCli.
			EXPECT().
			VirtualMachine(k8sv1.NamespaceAll).
			Return(fakeClient.KubevirtV1().VirtualMachines(k8sv1.NamespaceAll)).
			AnyTimes()
	})

	Context("if uninstall strategy is BlockUninstallIfWorkloadExists", func() {
		BeforeEach(func() {
			setKV(fakeClient, v1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist, v1.KubeVirtPhaseDeployed)
		})

		It("should allow the deletion if no workload exists", func() {
			response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeTrue())
		})

		It("should deny the deletion if a VMI exists", func() {
			_, err := fakeClient.KubevirtV1().VirtualMachineInstances(k8sv1.NamespaceDefault).Create(context.TODO(), &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "vmi"}}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if a VM exists", func() {
			_, err := fakeClient.KubevirtV1().VirtualMachines(k8sv1.NamespaceDefault).Create(context.TODO(), &v1.VirtualMachine{ObjectMeta: metav1.ObjectMeta{Name: "vm"}}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if a VMIRS exists", func() {
			_, err := fakeClient.KubevirtV1().VirtualMachineInstanceReplicaSets(k8sv1.NamespaceDefault).Create(context.TODO(), &v1.VirtualMachineInstanceReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "rs"}}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if checking VMIs fails", func() {
			fakeClient.PrependReactor("list", "virtualmachineinstances", func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, &v1.VirtualMachineInstanceList{}, fmt.Errorf("whatever")
			})

			response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if checking VMs fails", func() {
			fakeClient.PrependReactor("list", "virtualmachines", func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, &v1.VirtualMachineList{}, fmt.Errorf("whatever")
			})

			response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if checking VMIRS fails", func() {
			fakeClient.PrependReactor("list", "virtualmachineinstancereplicasets", func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, &v1.VirtualMachineInstanceReplicaSetList{}, fmt.Errorf("whatever")
			})

			response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})
	})

	It("should allow the deletion if the strategy is empty", func() {
		response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
		Expect(response.Allowed).To(BeTrue())
	})

	It("should allow the deletion if the strategy is set to RemoveWorkloads", func() {
		setKV(fakeClient, v1.KubeVirtUninstallStrategyRemoveWorkloads, v1.KubeVirtPhaseDeployed)
		response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
		Expect(response.Allowed).To(BeTrue())
	})

	It("should allow the deletion of namespaces, where it gets an admission request without a resource name", func() {
		setKV(fakeClient, v1.KubeVirtUninstallStrategyRemoveWorkloads, v1.KubeVirtPhaseDeployed)
		response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: ""}})
		Expect(response.Allowed).To(BeTrue())
	})

	DescribeTable("should not check for workloads if kubevirt phase is", func(phase v1.KubeVirtPhase) {
		setKV(fakeClient, v1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist, phase)
		response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
		Expect(response.Allowed).To(BeTrue())
	},
		Entry("unset", v1.KubeVirtPhase("")),
		Entry("deploying", v1.KubeVirtPhaseDeploying),
		Entry("deleting", v1.KubeVirtPhaseDeleting),
		Entry("deleted", v1.KubeVirtPhaseDeleted),
	)
})

func setKV(fakeClient *kubevirtfake.Clientset, strategy v1.KubeVirtUninstallStrategy, phase v1.KubeVirtPhase) {
	patchBytes, err := patch.New(
		patch.WithReplace("/spec/uninstallStrategy", strategy),
		patch.WithReplace("/status/phase", phase),
	).GeneratePayload()
	Expect(err).NotTo(HaveOccurred())
	_, err = fakeClient.KubevirtV1().KubeVirts("test").Patch(context.TODO(), "kubevirt", types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	Expect(err).ToNot(HaveOccurred())
}
