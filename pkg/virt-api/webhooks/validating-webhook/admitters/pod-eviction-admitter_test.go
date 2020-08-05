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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package admitters

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/admission/v1beta1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	virtv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Pod eviction admitter", func() {

	newClusterConfigWithFeatureGate := func(featureGate string) *virtconfig.ClusterConfig {
		clusterConfig, cmInformer, _, _ := testutils.NewFakeClusterConfig(&k8sv1.ConfigMap{})
		testutils.UpdateFakeClusterConfig(cmInformer, &k8sv1.ConfigMap{
			Data: map[string]string{virtconfig.FeatureGatesKey: featureGate},
		})

		return clusterConfig
	}

	var ctrl *gomock.Controller

	var kubeClient *fake.Clientset
	var virtClient *kubecli.MockKubevirtClient

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		kubeClient = fake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

		// Make sure that any unexpected call to the client will fail
		kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
	})

	Context("Live migration enabled", func() {

		It("Should allow review requests that are not on a virt-launcher pod", func() {

			By("Composing a dummy admission request on a virt-launcher pod")
			testns := "test"

			pod := &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testns,
					Name:      "foo",
				},
			}

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Name:      pod.Name,
					Namespace: pod.Namespace,
				},
			}

			clusterConfig := newClusterConfigWithFeatureGate(virtconfig.LiveMigrationGate)

			podEvictionAdmitter := PodEvictionAdmitter{
				ClusterConfig: clusterConfig,
				VirtClient:    virtClient,
			}

			kubeClient.Fake.PrependReactor("get", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				get, ok := action.(testing.GetAction)
				Expect(ok).To(BeTrue())
				Expect(pod.Namespace).To(Equal(get.GetNamespace()))
				Expect(pod.Name).To(Equal(get.GetName()))
				return true, pod, nil
			})

			resp := podEvictionAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("Should deny review requests that are on a virt-launcher pod", func() {

			By("Composing a dummy admission request on a virt-launcher pod")
			testns := "test"

			pod := &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testpod",
					Namespace: testns,
					Labels: map[string]string{
						virtv1.AppLabel: "virt-launcher",
					},
				},
				Spec:   k8sv1.PodSpec{},
				Status: k8sv1.PodStatus{},
			}

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Name:      pod.Name,
					Namespace: pod.Namespace,
				},
			}

			kubeClient.Fake.PrependReactor("get", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				get, ok := action.(testing.GetAction)
				Expect(ok).To(BeTrue())
				Expect(pod.Namespace).To(Equal(get.GetNamespace()))
				Expect(pod.Name).To(Equal(get.GetName()))
				return true, pod, nil
			})

			kubeClient.Fake.PrependReactor("update", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.UpdateAction)
				Expect(ok).To(BeTrue())
				Expect(pod.Namespace).To(Equal(update.GetNamespace()))

				updatedPod, ok := update.GetObject().(*k8sv1.Pod)
				Expect(ok).To(BeTrue())
				Expect(pod.Name).To(Equal(updatedPod.Name))
				found := false
				for _, status := range updatedPod.Status.Conditions {
					if status.Type == virtv1.LauncherMarkedForEviction && status.Status == k8sv1.ConditionTrue {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue(), "expected condition update on the launcher pod")
				return true, nil, nil
			})
			clusterConfig := newClusterConfigWithFeatureGate(virtconfig.LiveMigrationGate)

			podEvictionAdmitter := PodEvictionAdmitter{
				ClusterConfig: clusterConfig,
				VirtClient:    virtClient,
			}
			resp := podEvictionAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Code).To(Equal(int32(429)))
			Expect(kubeClient.Fake.Actions()).To(HaveLen(2))
		})

	})

	Context("Live migration disabled", func() {

		clusterConfig, _, _, _ := testutils.NewFakeClusterConfig(&k8sv1.ConfigMap{})
		podEvictionAdmitter := PodEvictionAdmitter{
			ClusterConfig: clusterConfig,
		}

		It("Should allow any review request", func() {
			resp := podEvictionAdmitter.Admit(&v1beta1.AdmissionReview{})
			Expect(resp.Allowed).To(BeTrue())
		})

	})

})
