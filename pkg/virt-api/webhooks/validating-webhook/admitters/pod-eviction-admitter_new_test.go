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
	"context"

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

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/validating-webhook/admitters"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Pod eviction admitter", func() {
	const (
		testNamespace = "test-ns"
		testNodeName  = "node01"
		testVMIName   = "my-vmi"
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

	DescribeTable("The admitter should approve the request and not trigger VMI evacuation", func(clusterWideEvictionStrategy, vmiEvictionStrategy *kvirtv1.EvictionStrategy, isVMIMigratable bool) {
		vmiOptions := []vmiOption{
			withEvictionStrategy(vmiEvictionStrategy),
		}

		if isVMIMigratable {
			vmiOptions = append(vmiOptions, withLiveMigratableCondition())
		}

		vmi := newVMI(testNamespace, testVMIName, testNodeName, vmiOptions...)

		evictedVirtLauncherPod := newVirtLauncherPod(vmi.Namespace, vmi.Name, vmi.Status.NodeName)
		kubeClient := fake.NewSimpleClientset(evictedVirtLauncherPod)

		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

		vmiClient := kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		virtClient.EXPECT().VirtualMachineInstance(testNamespace).Return(vmiClient).AnyTimes()
		vmiClient.EXPECT().Get(context.Background(), vmi.Name, metav1.GetOptions{}).Return(vmi, nil)

		admitter := admitters.PodEvictionAdmitter{
			ClusterConfig: newClusterConfig(clusterWideEvictionStrategy),
			VirtClient:    virtClient,
		}

		actualAdmissionResponse := admitter.Admit(
			newAdmissionReview(evictedVirtLauncherPod.Namespace, evictedVirtLauncherPod.Name, nil),
		)

		Expect(actualAdmissionResponse).To(Equal(allowedAdmissionResponse()))
		Expect(kubeClient.Fake.Actions()).To(HaveLen(1))
	},
		Entry("When cluster-wide eviction strategy is nil, VMI eviction strategy is nil and VMI is not migratable",
			nil,
			nil,
			false,
		),
		Entry("When cluster-wide eviction strategy is nil, VMI eviction strategy is nil and VMI is migratable",
			nil,
			nil,
			true,
		),
		Entry("When cluster-wide eviction strategy is nil, VMI eviction strategy is None and VMI is not migratable",
			nil,
			pointer.P(kvirtv1.EvictionStrategyNone),
			false,
		),
		Entry("When cluster-wide eviction strategy is nil, VMI eviction strategy is None and VMI is migratable",
			nil,
			pointer.P(kvirtv1.EvictionStrategyNone),
			true,
		),
		Entry("When cluster-wide eviction strategy is nil, VMI eviction strategy is LiveMigrateIfPossible and VMI is not migratable",
			nil,
			pointer.P(kvirtv1.EvictionStrategyLiveMigrateIfPossible),
			false,
		),
		Entry("When cluster-wide eviction strategy is None, VMI eviction strategy is nil and VMI is not migratable",
			pointer.P(kvirtv1.EvictionStrategyNone),
			nil,
			false,
		),
		Entry("When cluster-wide eviction strategy is None, VMI eviction strategy is nil and VMI is migratable",
			pointer.P(kvirtv1.EvictionStrategyNone),
			nil,
			true,
		),
		Entry("When cluster-wide eviction strategy is LiveMigrateIfPossible, VMI eviction strategy is nil and VMI is not migratable",
			pointer.P(kvirtv1.EvictionStrategyLiveMigrateIfPossible),
			nil,
			false,
		),
	)
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

func newVirtLauncherPod(namespace, vmiName, nodeName string) *k8scorev1.Pod {
	podName := "virt-launcher" + vmiName
	virtLauncher := newPod(namespace, podName, nodeName)

	virtLauncher.Annotations = map[string]string{
		kvirtv1.DomainAnnotation: vmiName,
	}

	virtLauncher.Labels = map[string]string{
		kvirtv1.AppLabel: "virt-launcher",
	}

	return virtLauncher
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

type vmiOption func(vmi *kvirtv1.VirtualMachineInstance)

func newVMI(namespace, name, nodeName string, options ...vmiOption) *kvirtv1.VirtualMachineInstance {
	vmi := &kvirtv1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Status: kvirtv1.VirtualMachineInstanceStatus{
			NodeName: nodeName,
		},
	}

	for _, optionFunc := range options {
		optionFunc(vmi)
	}

	return vmi
}

func withEvictionStrategy(evictionStrategy *kvirtv1.EvictionStrategy) vmiOption {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		vmi.Spec.EvictionStrategy = evictionStrategy
	}
}

func withLiveMigratableCondition() vmiOption {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		vmi.Status.Conditions = append(vmi.Status.Conditions, kvirtv1.VirtualMachineInstanceCondition{
			Type:   kvirtv1.VirtualMachineInstanceIsMigratable,
			Status: k8scorev1.ConditionTrue,
		})
	}
}
