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
 * Copyright The KubeVirt Authors
 *
 */

package object_test

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tools/perfscale-load-generator/object"
)

var _ = Describe("CleanupNamespaces", func() {
	const workloadLabel = "workload=uuid"

	listOpts := &metav1.ListOptions{LabelSelector: workloadLabel}

	namespaceDeletes := func(client *fake.Clientset) []testing.Action {
		var deletes []testing.Action
		for _, action := range client.Actions() {
			if action.GetVerb() == "delete" && action.GetResource().Resource == "namespaces" {
				deletes = append(deletes, action)
			}
		}
		return deletes
	}

	Describe("with a fake client", func() {
		var (
			virtClient *kubecli.MockKubevirtClient
			kubeClient *fake.Clientset
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			kubeClient = fake.NewSimpleClientset()
			virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		})

		DescribeTable("returns the list error without deleting namespaces", func(list runtime.Object) {
			listErr := errors.New("namespace list failed")
			kubeClient.PrependReactor("list", "namespaces", func(testing.Action) (bool, runtime.Object, error) {
				return true, list, listErr
			})

			err := object.CleanupNamespaces(virtClient, time.Minute, listOpts)
			Expect(err).To(MatchError(listErr))
			Expect(namespaceDeletes(kubeClient)).To(BeEmpty())
		},
			Entry("when the reactor returns no object", nil),
			Entry("when the reactor returns namespaces alongside the error", &corev1.NamespaceList{
				Items: []corev1.Namespace{{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "should-not-delete",
						Labels: map[string]string{"workload": "uuid"},
					},
				}},
			}),
		)

		It("deletes namespaces that match the selector and leaves the rest", func() {
			ctx := context.Background()
			matching := map[string]string{"workload": "uuid"}
			for _, ns := range []*corev1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: "keep-ns"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "remove-a", Labels: matching}},
				{ObjectMeta: metav1.ObjectMeta{Name: "remove-b", Labels: matching}},
			} {
				_, err := kubeClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
			kubeClient.ClearActions()

			err := object.CleanupNamespaces(virtClient, time.Minute, listOpts)
			Expect(err).ToNot(HaveOccurred())

			for _, name := range []string{"remove-a", "remove-b"} {
				_, getErr := kubeClient.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
				Expect(apierrors.IsNotFound(getErr)).To(BeTrue())
			}
			_, err = kubeClient.CoreV1().Namespaces().Get(ctx, "keep-ns", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("continues cleanup when a namespace delete returns NotFound", func() {
			ctx := context.Background()
			matching := map[string]string{"workload": "uuid"}
			for _, name := range []string{"already-gone", "still-there"} {
				_, err := kubeClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: name, Labels: matching},
				}, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
			kubeClient.ClearActions()

			kubeClient.PrependReactor("delete", "namespaces", func(action testing.Action) (bool, runtime.Object, error) {
				if action.(testing.DeleteAction).GetName() == "already-gone" {
					return true, nil, apierrors.NewNotFound(corev1.Resource("namespaces"), "already-gone")
				}
				return false, nil, nil
			})

			err := object.CleanupNamespaces(virtClient, time.Minute, listOpts)
			Expect(err).ToNot(HaveOccurred())

			var deleted []string
			for _, action := range namespaceDeletes(kubeClient) {
				deleted = append(deleted, action.(testing.DeleteAction).GetName())
			}
			Expect(deleted).To(ConsistOf("already-gone", "still-there"))
			_, err = kubeClient.CoreV1().Namespaces().Get(ctx, "still-there", metav1.GetOptions{})
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})

		It("returns a delete error without deleting the remaining namespaces", func() {
			ctx := context.Background()
			matching := map[string]string{"workload": "uuid"}
			for _, name := range []string{"remove-a", "remove-b"} {
				_, err := kubeClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: name, Labels: matching},
				}, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
			kubeClient.ClearActions()

			deleteErr := errors.New("delete failed")
			kubeClient.PrependReactor("delete", "namespaces", func(testing.Action) (bool, runtime.Object, error) {
				return true, nil, deleteErr
			})

			err := object.CleanupNamespaces(virtClient, time.Minute, listOpts)
			Expect(err).To(MatchError(deleteErr))
			Expect(namespaceDeletes(kubeClient)).To(HaveLen(1))
			for _, name := range []string{"remove-a", "remove-b"} {
				_, getErr := kubeClient.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
				Expect(getErr).ToNot(HaveOccurred())
			}
		})
	})
})
