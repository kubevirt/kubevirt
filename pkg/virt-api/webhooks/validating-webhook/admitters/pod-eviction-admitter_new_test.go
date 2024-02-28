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

package admitters_test

import (
	"github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sadmissionv1 "k8s.io/api/admission/v1"
	k8scorev1 "k8s.io/api/core/v1"
	k8spolicyv1 "k8s.io/api/policy/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/client-go/kubernetes/fake"

	kvirtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/validating-webhook/admitters"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Pod eviction admitter", func() {
	const (
		testNamespace = "test-ns"
		testNodeName  = "node01"
	)

	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	When("an AdmissionReview request for the eviction of a regular pod is admitted", func() {
		It("should be allowed", func() {
			const evictedPodName = "my-pod"

			evictedPod := newPod(testNamespace, evictedPodName, testNodeName)
			kubeClient := fake.NewSimpleClientset(evictedPod)

			virtClient := kubecli.NewMockKubevirtClient(ctrl)
			virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

			admitter := admitters.PodEvictionAdmitter{
				ClusterConfig: newClusterConfig(nil),
				VirtClient:    virtClient,
			}

			actualAdmissionResponse := admitter.Admit(
				newAdmissionReview(evictedPod.Namespace, evictedPod.Name, nil),
			)

			Expect(actualAdmissionResponse).To(Equal(allowedAdmissionResponse()))
			Expect(kubeClient.Fake.Actions()).To(HaveLen(1))
		})
	})

	When("the admitter cannot fetch the pod from the AdmissionReview request", func() {
		It("should allow the request", func() {
			kubeClient := fake.NewSimpleClientset()
			Expect(kubeClient.Fake.Resources).To(BeEmpty())

			virtClient := kubecli.NewMockKubevirtClient(ctrl)
			virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

			admitter := admitters.PodEvictionAdmitter{
				ClusterConfig: newClusterConfig(nil),
				VirtClient:    virtClient,
			}

			actualAdmissionResponse := admitter.Admit(
				newAdmissionReview(testNamespace, "does-not-exist", nil),
			)

			Expect(actualAdmissionResponse).To(Equal(allowedAdmissionResponse()))
			Expect(kubeClient.Fake.Actions()).To(HaveLen(1))
		})
	})
})

func newClusterConfig(clusterWideEvictionStrategy *kvirtv1.EvictionStrategy) *virtconfig.ClusterConfig {
	const (
		kubevirtCRName    = "kubevirt"
		kubevirtNamespace = "kubevirt"
	)

	kv := kubecli.NewMinimalKubeVirt(kubevirtCRName)
	kv.Namespace = kubevirtNamespace

	kv.Spec.Configuration.EvictionStrategy = clusterWideEvictionStrategy

	clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKV(kv)
	return clusterConfig
}

func newPod(namespace, name, nodeName string) *k8scorev1.Pod {
	return &k8scorev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: k8scorev1.PodSpec{
			NodeName: nodeName,
		},
	}
}

func newAdmissionReview(evictedPodNamespace, evictedPodName string, isDryRun *bool) *k8sadmissionv1.AdmissionReview {
	return &k8sadmissionv1.AdmissionReview{
		Request: &k8sadmissionv1.AdmissionRequest{
			Namespace: evictedPodNamespace,
			Name:      evictedPodName,
			DryRun:    isDryRun,
			Kind: metav1.GroupVersionKind{
				Group:   "policy",
				Version: "v1",
				Kind:    "Eviction",
			},
			Resource: metav1.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "pods",
			},
			SubResource: "eviction",
			RequestKind: &metav1.GroupVersionKind{
				Group:   "policy",
				Version: "v1",
				Kind:    "Eviction",
			},
			RequestResource: &metav1.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "pods",
			},
			RequestSubResource: "eviction",
			Operation:          "CREATE",
			Object: runtime.RawExtension{
				Object: &k8spolicyv1.Eviction{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "policy/v1",
						Kind:       "Eviction",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: evictedPodNamespace,
						Name:      evictedPodName,
					},
				},
			},
		},
	}
}

func allowedAdmissionResponse() *k8sadmissionv1.AdmissionResponse {
	return &k8sadmissionv1.AdmissionResponse{
		Allowed: true,
	}
}
