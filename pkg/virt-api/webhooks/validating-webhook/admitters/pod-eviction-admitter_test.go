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
	"errors"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	virtv1 "kubevirt.io/api/core/v1"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
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

	const isDryRun = true

	It("should allow the request when it refers to a non virt-launcher pod", func() {
		virtClient := kubevirtfake.NewSimpleClientset()
		Expect(virtClient.Fake.Resources).To(BeEmpty())

		const evictedPodName = "my-pod"

		evictedPod := newPod(testNamespace, evictedPodName, testNodeName)
		kubeClient := fake.NewSimpleClientset(evictedPod)

		admitter := admitters.NewPodEvictionAdmitter(
			newClusterConfig(nil),
			kubeClient,
			virtClient,
		)

		actualAdmissionResponse := admitter.Admit(
			newAdmissionReview(evictedPod.Namespace, evictedPod.Name, !isDryRun),
		)

		Expect(actualAdmissionResponse).To(Equal(allowedAdmissionResponse()))
		Expect(kubeClient.Fake.Actions()).To(HaveLen(1))
		Expect(virtClient.Fake.Actions()).To(BeEmpty())
	})

	It("should allow the request when the admitter cannot fetch the pod", func() {
		virtClient := kubevirtfake.NewSimpleClientset()
		Expect(virtClient.Fake.Resources).To(BeEmpty())

		kubeClient := fake.NewSimpleClientset()
		Expect(kubeClient.Fake.Resources).To(BeEmpty())

		admitter := admitters.NewPodEvictionAdmitter(
			newClusterConfig(nil),
			kubeClient,
			virtClient,
		)

		actualAdmissionResponse := admitter.Admit(
			newAdmissionReview(testNamespace, "does-not-exist", !isDryRun),
		)

		Expect(actualAdmissionResponse).To(Equal(allowedAdmissionResponse()))
		Expect(kubeClient.Fake.Actions()).To(HaveLen(1))
		Expect(virtClient.Fake.Actions()).To(BeEmpty())
	})

	DescribeTable("should trigger VMI Evacuation and deny the request", func(clusterWideEvictionStrategy *virtv1.EvictionStrategy, vmiOptions ...vmiOption) {
		vmi := newVMI(testNamespace, testVMIName, testNodeName, vmiOptions...)
		virtClient := kubevirtfake.NewSimpleClientset(vmi)

		evictedVirtLauncherPod := newVirtLauncherPod(vmi.Namespace, vmi.Name, vmi.Status.NodeName)
		kubeClient := fake.NewSimpleClientset(evictedVirtLauncherPod)

		admitter := admitters.NewPodEvictionAdmitter(
			newClusterConfig(clusterWideEvictionStrategy),
			kubeClient,
			virtClient,
		)

		expectedAdmissionResponse := newDeniedAdmissionResponse(
			fmt.Sprintf("Eviction triggered evacuation of VMI \"%s/%s\"", vmi.Namespace, vmi.Name),
		)

		actualAdmissionResponse := admitter.Admit(
			newAdmissionReview(evictedVirtLauncherPod.Namespace, evictedVirtLauncherPod.Name, !isDryRun),
		)

		Expect(actualAdmissionResponse).To(Equal(expectedAdmissionResponse))
		Expect(kubeClient.Fake.Actions()).To(HaveLen(1))

		patchBytes, err := patch.New(patch.WithAdd("/status/evacuationNodeName", vmi.Status.NodeName)).GeneratePayload()
		Expect(err).To(Not(HaveOccurred()))
		Expect(virtClient.Actions()).To(ContainElement(newExpectedJSONPatchToVMI(vmi, patchBytes)))
	},
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is LiveMigrate and VMI is migratable",
			nil,
			withEvictionStrategy(pointer.P(virtv1.EvictionStrategyLiveMigrate)),
			withLiveMigratableCondition(),
		),
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is LiveMigrateIfPossible and VMI is migratable",
			nil,
			withEvictionStrategy(pointer.P(virtv1.EvictionStrategyLiveMigrateIfPossible)),
			withLiveMigratableCondition(),
		),
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is External and VMI is not migratable",
			nil,
			withEvictionStrategy(pointer.P(virtv1.EvictionStrategyExternal)),
		),
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is External and VMI is migratable",
			nil,
			withEvictionStrategy(pointer.P(virtv1.EvictionStrategyExternal)),
			withLiveMigratableCondition(),
		),
		Entry("When cluster-wide eviction strategy is LiveMigrate, VMI eviction strategy is missing and VMI is migratable",
			pointer.P(virtv1.EvictionStrategyLiveMigrate),
			withEvictionStrategy(nil),
			withLiveMigratableCondition(),
		),
		Entry("When cluster-wide eviction strategy is LiveMigrateIfPossible, VMI eviction strategy is missing and VMI is migratable",
			pointer.P(virtv1.EvictionStrategyLiveMigrateIfPossible),
			withEvictionStrategy(nil),
			withLiveMigratableCondition(),
		),
		Entry("When cluster-wide eviction strategy is External, VMI eviction strategy is missing and VMI is not migratable",
			pointer.P(virtv1.EvictionStrategyExternal),
			withEvictionStrategy(nil),
		),
		Entry("When cluster-wide eviction strategy is External, VMI eviction strategy is missing and VMI is migratable",
			pointer.P(virtv1.EvictionStrategyExternal),
			withEvictionStrategy(nil),
			withLiveMigratableCondition(),
		),
	)

	DescribeTable("should allow the request without triggering VMI evacuation", func(clusterWideEvictionStrategy *virtv1.EvictionStrategy, vmiOptions ...vmiOption) {
		vmi := newVMI(testNamespace, testVMIName, testNodeName, vmiOptions...)
		virtClient := kubevirtfake.NewSimpleClientset(vmi)

		evictedVirtLauncherPod := newVirtLauncherPod(vmi.Namespace, vmi.Name, vmi.Status.NodeName)
		kubeClient := fake.NewSimpleClientset(evictedVirtLauncherPod)

		admitter := admitters.NewPodEvictionAdmitter(
			newClusterConfig(clusterWideEvictionStrategy),
			kubeClient,
			virtClient,
		)

		actualAdmissionResponse := admitter.Admit(
			newAdmissionReview(evictedVirtLauncherPod.Namespace, evictedVirtLauncherPod.Name, !isDryRun),
		)

		Expect(actualAdmissionResponse).To(Equal(allowedAdmissionResponse()))
		Expect(kubeClient.Fake.Actions()).To(HaveLen(1))
		Expect(virtClient.Fake.Actions()).To(HaveLen(1))
	},
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is missing and VMI is not migratable",
			nil,
			withEvictionStrategy(nil),
		),
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is missing and VMI is migratable",
			nil,
			withEvictionStrategy(nil),
			withLiveMigratableCondition(),
		),
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is None and VMI is not migratable",
			nil,
			withEvictionStrategy(pointer.P(virtv1.EvictionStrategyNone)),
		),
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is None and VMI is migratable",
			nil,
			withEvictionStrategy(pointer.P(virtv1.EvictionStrategyNone)),
			withLiveMigratableCondition(),
		),
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is LiveMigrateIfPossible and VMI is not migratable",
			nil,
			withEvictionStrategy(pointer.P(virtv1.EvictionStrategyLiveMigrateIfPossible)),
		),
		Entry("When cluster-wide eviction strategy is None, VMI eviction strategy is missing and VMI is not migratable",
			pointer.P(virtv1.EvictionStrategyNone),
			withEvictionStrategy(nil),
		),
		Entry("When cluster-wide eviction strategy is None, VMI eviction strategy is missing and VMI is migratable",
			pointer.P(virtv1.EvictionStrategyNone),
			withEvictionStrategy(nil),
			withLiveMigratableCondition(),
		),
		Entry("When cluster-wide eviction strategy is LiveMigrateIfPossible, VMI eviction strategy is missing and VMI is not migratable",
			pointer.P(virtv1.EvictionStrategyLiveMigrateIfPossible),
			withEvictionStrategy(nil),
		),
	)

	DescribeTable("should deny the request without triggering VMI evacuation", func(clusterWideEvictionStrategy *virtv1.EvictionStrategy, vmiOptions ...vmiOption) {
		vmi := newVMI(testNamespace, testVMIName, testNodeName, vmiOptions...)
		virtClient := kubevirtfake.NewSimpleClientset(vmi)

		evictedVirtLauncherPod := newVirtLauncherPod(vmi.Namespace, vmi.Name, vmi.Status.NodeName)
		kubeClient := fake.NewSimpleClientset(evictedVirtLauncherPod)

		admitter := admitters.NewPodEvictionAdmitter(
			newClusterConfig(clusterWideEvictionStrategy),
			kubeClient,
			virtClient,
		)

		expectedAdmissionResponse := newDeniedAdmissionResponse(
			fmt.Sprintf("VMI %s is configured with an eviction strategy but is not live-migratable", vmi.Name),
		)

		actualAdmissionResponse := admitter.Admit(
			newAdmissionReview(evictedVirtLauncherPod.Namespace, evictedVirtLauncherPod.Name, !isDryRun),
		)

		Expect(actualAdmissionResponse).To(Equal(expectedAdmissionResponse))
		Expect(kubeClient.Fake.Actions()).To(HaveLen(1))
		Expect(virtClient.Fake.Actions()).To(HaveLen(1))
	},
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is LiveMigrate and VMI is not migratable",
			nil,
			withEvictionStrategy(pointer.P(virtv1.EvictionStrategyLiveMigrate)),
		),
		Entry("When cluster-wide eviction strategy is LiveMigrate, VMI eviction strategy is missing and VMI is not migratable",
			pointer.P(virtv1.EvictionStrategyLiveMigrate),
			withEvictionStrategy(nil),
		),
	)

	It("should deny the request when the admitter fails to fetch the VMI", func() {
		vmi := newVMI(testNamespace, testVMIName, testNodeName)
		virtClient := kubevirtfake.NewSimpleClientset(vmi)

		expectedError := errors.New("some error")
		virtClient.PrependReactor("get", "virtualmachineinstances", func(_ testing.Action) (bool, runtime.Object, error) {
			return true, nil, expectedError
		})

		evictedVirtLauncherPod := newVirtLauncherPod(vmi.Namespace, vmi.Name, vmi.Status.NodeName)
		kubeClient := fake.NewSimpleClientset(evictedVirtLauncherPod)

		admitter := admitters.NewPodEvictionAdmitter(
			newClusterConfig(nil),
			kubeClient,
			virtClient,
		)

		expectedAdmissionResponse := newDeniedAdmissionResponse(
			fmt.Sprintf("kubevirt failed getting the vmi: %s", expectedError.Error()),
		)

		actualAdmissionResponse := admitter.Admit(
			newAdmissionReview(evictedVirtLauncherPod.Namespace, evictedVirtLauncherPod.Name, !isDryRun),
		)

		Expect(actualAdmissionResponse).To(Equal(expectedAdmissionResponse))
		Expect(kubeClient.Fake.Actions()).To(HaveLen(1))
		Expect(virtClient.Fake.Actions()).To(HaveLen(1))
	})

	It("should deny the request when the admitter fails to patch the VMI", func() {
		evictionStrategy := virtv1.EvictionStrategyLiveMigrate
		vmiOptions := []vmiOption{withEvictionStrategy(&evictionStrategy), withLiveMigratableCondition()}

		migratableVMI := newVMI(testNamespace, testVMIName, testNodeName, vmiOptions...)
		virtClient := kubevirtfake.NewSimpleClientset(migratableVMI)

		expectedError := errors.New("some error")
		virtClient.PrependReactor("patch", "virtualmachineinstances", func(_ testing.Action) (bool, runtime.Object, error) {
			return true, nil, expectedError
		})

		evictedVirtLauncherPod := newVirtLauncherPod(migratableVMI.Namespace, migratableVMI.Name, migratableVMI.Status.NodeName)
		kubeClient := fake.NewSimpleClientset(evictedVirtLauncherPod)

		admitter := admitters.NewPodEvictionAdmitter(
			newClusterConfig(nil),
			kubeClient,
			virtClient,
		)

		expectedAdmissionResponse := newDeniedAdmissionResponse(
			fmt.Sprintf("kubevirt failed marking the vmi for eviction: %s", expectedError.Error()),
		)

		actualAdmissionResponse := admitter.Admit(
			newAdmissionReview(evictedVirtLauncherPod.Namespace, evictedVirtLauncherPod.Name, !isDryRun),
		)

		Expect(actualAdmissionResponse).To(Equal(expectedAdmissionResponse))
		Expect(kubeClient.Fake.Actions()).To(HaveLen(1))

		patchBytes, err := patch.New(patch.WithAdd("/status/evacuationNodeName", migratableVMI.Status.NodeName)).GeneratePayload()
		Expect(err).To(Not(HaveOccurred()))
		Expect(virtClient.Actions()).To(ContainElement(newExpectedJSONPatchToVMI(migratableVMI, patchBytes)))
	})

	It("should allow the request and not mark the VMI again when the VMI is already marked for evacuation", func() {
		evictionStrategy := virtv1.EvictionStrategyLiveMigrate
		vmiOptions := []vmiOption{withEvictionStrategy(&evictionStrategy), withLiveMigratableCondition(), withEvacuationNodeName(testNodeName)}

		migratableVMI := newVMI(testNamespace, testVMIName, testNodeName, vmiOptions...)
		virtClient := kubevirtfake.NewSimpleClientset(migratableVMI)

		evictedVirtLauncherPod := newVirtLauncherPod(migratableVMI.Namespace, migratableVMI.Name, migratableVMI.Status.NodeName)
		kubeClient := fake.NewSimpleClientset(evictedVirtLauncherPod)

		admitter := admitters.NewPodEvictionAdmitter(
			newClusterConfig(nil),
			kubeClient,
			virtClient,
		)

		actualAdmissionResponse := admitter.Admit(
			newAdmissionReview(evictedVirtLauncherPod.Namespace, evictedVirtLauncherPod.Name, !isDryRun),
		)

		Expect(actualAdmissionResponse).To(Equal(allowedAdmissionResponse()))
		Expect(kubeClient.Fake.Actions()).To(HaveLen(1))
		Expect(virtClient.Fake.Actions()).To(HaveLen(1))
	})

	It("should deny the request and perform a dryRun patch on the VMI when the request is a dry run", func() {
		evictionStrategy := virtv1.EvictionStrategyLiveMigrate
		vmiOptions := []vmiOption{withEvictionStrategy(&evictionStrategy), withLiveMigratableCondition()}

		migratableVMI := newVMI(testNamespace, testVMIName, testNodeName, vmiOptions...)
		virtClient := kubevirtfake.NewSimpleClientset(migratableVMI)

		evictedVirtLauncherPod := newVirtLauncherPod(migratableVMI.Namespace, migratableVMI.Name, migratableVMI.Status.NodeName)
		kubeClient := fake.NewSimpleClientset(evictedVirtLauncherPod)

		expectedAdmissionResponse := newDeniedAdmissionResponse(
			fmt.Sprintf("Eviction triggered evacuation of VMI \"%s/%s\"", migratableVMI.Namespace, migratableVMI.Name),
		)

		admitter := admitters.NewPodEvictionAdmitter(
			newClusterConfig(nil),
			kubeClient,
			virtClient,
		)

		actualAdmissionResponse := admitter.Admit(
			newAdmissionReview(evictedVirtLauncherPod.Namespace, evictedVirtLauncherPod.Name, isDryRun),
		)

		Expect(actualAdmissionResponse).To(Equal(expectedAdmissionResponse))
		Expect(kubeClient.Fake.Actions()).To(HaveLen(1))

		patchBytes, err := patch.New(patch.WithAdd("/status/evacuationNodeName", migratableVMI.Status.NodeName)).GeneratePayload()
		Expect(err).To(Not(HaveOccurred()))
		Expect(virtClient.Actions()).To(ContainElement(newExpectedJSONPatchToVMI(migratableVMI, patchBytes)))
	})
})

func newClusterConfig(clusterWideEvictionStrategy *virtv1.EvictionStrategy) *virtconfig.ClusterConfig {
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

func newPod(namespace, name, nodeName string) *k8sv1.Pod {
	return &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: k8sv1.PodSpec{
			NodeName: nodeName,
		},
	}
}

func newVirtLauncherPod(namespace, vmiName, nodeName string) *k8sv1.Pod {
	podName := "virt-launcher" + vmiName
	virtLauncher := newPod(namespace, podName, nodeName)

	virtLauncher.Annotations = map[string]string{
		virtv1.DomainAnnotation: vmiName,
	}

	virtLauncher.Labels = map[string]string{
		virtv1.AppLabel: "virt-launcher",
	}

	return virtLauncher
}

func newAdmissionReview(evictedPodNamespace, evictedPodName string, isDryRun bool) *admissionv1.AdmissionReview {
	return &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Namespace: evictedPodNamespace,
			Name:      evictedPodName,
			DryRun:    pointer.P(isDryRun),
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
				Object: &policyv1.Eviction{
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

func allowedAdmissionResponse() *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}

func newDeniedAdmissionResponse(message string) *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{
		Allowed: false,
		Result: &metav1.Status{
			Code:    int32(http.StatusTooManyRequests),
			Message: message,
		},
	}
}

func newExpectedJSONPatchToVMI(vmi *virtv1.VirtualMachineInstance, expectedJSONPatchData []byte) testing.PatchActionImpl {
	return testing.PatchActionImpl{
		ActionImpl: testing.ActionImpl{
			Namespace: vmi.Namespace,
			Verb:      "patch",
			Resource: schema.GroupVersionResource{
				Group:    "kubevirt.io",
				Version:  "v1",
				Resource: "virtualmachineinstances",
			},
			Subresource: "",
		},
		Name:      vmi.Name,
		PatchType: types.JSONPatchType,
		Patch:     expectedJSONPatchData,
	}
}

type vmiOption func(vmi *virtv1.VirtualMachineInstance)

func newVMI(namespace, name, nodeName string, options ...vmiOption) *virtv1.VirtualMachineInstance {
	vmi := &virtv1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Status: virtv1.VirtualMachineInstanceStatus{
			NodeName: nodeName,
		},
	}

	for _, optionFunc := range options {
		optionFunc(vmi)
	}

	return vmi
}

func withEvictionStrategy(evictionStrategy *virtv1.EvictionStrategy) vmiOption {
	return func(vmi *virtv1.VirtualMachineInstance) {
		vmi.Spec.EvictionStrategy = evictionStrategy
	}
}

func withLiveMigratableCondition() vmiOption {
	return func(vmi *virtv1.VirtualMachineInstance) {
		vmi.Status.Conditions = append(vmi.Status.Conditions, virtv1.VirtualMachineInstanceCondition{
			Type:   virtv1.VirtualMachineInstanceIsMigratable,
			Status: k8sv1.ConditionTrue,
		})
	}
}

func withEvacuationNodeName(evacuationNodeName string) vmiOption {
	return func(vmi *virtv1.VirtualMachineInstance) {
		vmi.Status.EvacuationNodeName = evacuationNodeName
	}
}
