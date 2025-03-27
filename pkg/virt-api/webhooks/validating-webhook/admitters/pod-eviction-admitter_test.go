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
	"encoding/json"
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
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/validating-webhook/admitters"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Pod eviction admitter", func() {
	const (
		testNamespace = "test-ns"
		testNodeName  = "node01"
	)

	defaultVMIOptions := []libvmi.Option{
		libvmi.WithNamespace(testNamespace),
		withStatusNodeName(testNodeName),
	}

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
			context.Background(),
			newAdmissionReview(evictedPod.Namespace, evictedPod.Name, &dryRunOptions{}),
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
			context.Background(),
			newAdmissionReview(testNamespace, "does-not-exist", &dryRunOptions{}),
		)

		Expect(actualAdmissionResponse).To(Equal(allowedAdmissionResponse()))
		Expect(kubeClient.Fake.Actions()).To(HaveLen(1))
		Expect(virtClient.Fake.Actions()).To(BeEmpty())
	})

	DescribeTable("should allow the request when it refers to a virt-launcher pod", func(podPhase k8sv1.PodPhase) {
		vmi := libvmi.New(defaultVMIOptions...)
		virtClient := kubevirtfake.NewSimpleClientset(vmi)

		pod := newVirtLauncherPodWithPhase(vmi.Namespace, vmi.Name, vmi.Status.NodeName, podPhase)
		kubeClient := fake.NewSimpleClientset(pod)

		admitter := admitters.NewPodEvictionAdmitter(
			newClusterConfig(nil),
			kubeClient,
			virtClient,
		)

		actualAdmissionResponse := admitter.Admit(
			context.Background(),
			newAdmissionReview(pod.Namespace, pod.Name, &dryRunOptions{}),
		)

		Expect(actualAdmissionResponse).To(Equal(allowedAdmissionResponse()))
		Expect(kubeClient.Fake.Actions()).To(HaveLen(1))
		Expect(virtClient.Fake.Actions()).To(BeEmpty())
	},
		Entry("in failed phase", k8sv1.PodFailed),
		Entry("in succeeded phase", k8sv1.PodSucceeded),
	)

	DescribeTable("should trigger VMI Evacuation and deny the request", func(clusterWideEvictionStrategy *virtv1.EvictionStrategy, additionalVMIOptions ...libvmi.Option) {
		vmiOptions := append(defaultVMIOptions, additionalVMIOptions...)

		vmi := libvmi.New(vmiOptions...)
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
			context.Background(),
			newAdmissionReview(evictedVirtLauncherPod.Namespace, evictedVirtLauncherPod.Name, &dryRunOptions{}),
		)

		Expect(actualAdmissionResponse).To(Equal(expectedAdmissionResponse))
		Expect(kubeClient.Fake.Actions()).To(HaveLen(1))

		patchBytes, err := patch.New(patch.WithAdd("/status/evacuationNodeName", vmi.Status.NodeName)).GeneratePayload()
		Expect(err).To(Not(HaveOccurred()))
		Expect(virtClient.Actions()).To(ContainElement(newExpectedJSONPatchToVMI(vmi, patchBytes, metav1.PatchOptions{})))
	},
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is LiveMigrate and VMI is migratable",
			nil,
			libvmi.WithEvictionStrategy(virtv1.EvictionStrategyLiveMigrate),
			withLiveMigratableCondition(),
		),
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is LiveMigrateIfPossible and VMI is migratable",
			nil,
			libvmi.WithEvictionStrategy(virtv1.EvictionStrategyLiveMigrateIfPossible),
			withLiveMigratableCondition(),
		),
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is External and VMI is not migratable",
			nil,
			libvmi.WithEvictionStrategy(virtv1.EvictionStrategyExternal),
		),
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is External and VMI is migratable",
			nil,
			libvmi.WithEvictionStrategy(virtv1.EvictionStrategyExternal),
			withLiveMigratableCondition(),
		),
		Entry("When cluster-wide eviction strategy is LiveMigrate, VMI eviction strategy is missing and VMI is migratable",
			pointer.P(virtv1.EvictionStrategyLiveMigrate),
			withLiveMigratableCondition(),
		),
		Entry("When cluster-wide eviction strategy is LiveMigrateIfPossible, VMI eviction strategy is missing and VMI is migratable",
			pointer.P(virtv1.EvictionStrategyLiveMigrateIfPossible),
			withLiveMigratableCondition(),
		),
		Entry("When cluster-wide eviction strategy is External, VMI eviction strategy is missing and VMI is not migratable",
			pointer.P(virtv1.EvictionStrategyExternal),
		),
		Entry("When cluster-wide eviction strategy is External, VMI eviction strategy is missing and VMI is migratable",
			pointer.P(virtv1.EvictionStrategyExternal),
			withLiveMigratableCondition(),
		),
	)

	DescribeTable("should allow the request without triggering VMI evacuation", func(clusterWideEvictionStrategy *virtv1.EvictionStrategy, additionalVMIOptions ...libvmi.Option) {
		vmiOptions := append(defaultVMIOptions, additionalVMIOptions...)

		vmi := libvmi.New(vmiOptions...)
		virtClient := kubevirtfake.NewSimpleClientset(vmi)

		evictedVirtLauncherPod := newVirtLauncherPod(vmi.Namespace, vmi.Name, vmi.Status.NodeName)
		kubeClient := fake.NewSimpleClientset(evictedVirtLauncherPod)

		admitter := admitters.NewPodEvictionAdmitter(
			newClusterConfig(clusterWideEvictionStrategy),
			kubeClient,
			virtClient,
		)

		actualAdmissionResponse := admitter.Admit(
			context.Background(),
			newAdmissionReview(evictedVirtLauncherPod.Namespace, evictedVirtLauncherPod.Name, &dryRunOptions{}),
		)

		Expect(actualAdmissionResponse).To(Equal(allowedAdmissionResponse()))
		Expect(kubeClient.Fake.Actions()).To(HaveLen(1))
		Expect(virtClient.Fake.Actions()).To(HaveLen(1))
	},
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is missing and VMI is not migratable",
			nil,
		),
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is missing and VMI is migratable",
			nil,
			withLiveMigratableCondition(),
		),
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is None and VMI is not migratable",
			nil,
			libvmi.WithEvictionStrategy(virtv1.EvictionStrategyNone),
		),
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is None and VMI is migratable",
			nil,
			libvmi.WithEvictionStrategy(virtv1.EvictionStrategyNone),
			withLiveMigratableCondition(),
		),
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is LiveMigrateIfPossible and VMI is not migratable",
			nil,
			libvmi.WithEvictionStrategy(virtv1.EvictionStrategyLiveMigrateIfPossible),
		),
		Entry("When cluster-wide eviction strategy is None, VMI eviction strategy is missing and VMI is not migratable",
			pointer.P(virtv1.EvictionStrategyNone),
		),
		Entry("When cluster-wide eviction strategy is None, VMI eviction strategy is missing and VMI is migratable",
			pointer.P(virtv1.EvictionStrategyNone),
			withLiveMigratableCondition(),
		),
		Entry("When cluster-wide eviction strategy is LiveMigrateIfPossible, VMI eviction strategy is missing and VMI is not migratable",
			pointer.P(virtv1.EvictionStrategyLiveMigrateIfPossible),
		),
	)

	DescribeTable("should deny the request without triggering VMI evacuation", func(clusterWideEvictionStrategy *virtv1.EvictionStrategy, additionalVMIOptions ...libvmi.Option) {
		vmiOptions := append(defaultVMIOptions, additionalVMIOptions...)

		vmi := libvmi.New(vmiOptions...)
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
			context.Background(),
			newAdmissionReview(evictedVirtLauncherPod.Namespace, evictedVirtLauncherPod.Name, &dryRunOptions{}),
		)

		Expect(actualAdmissionResponse).To(Equal(expectedAdmissionResponse))
		Expect(kubeClient.Fake.Actions()).To(HaveLen(1))
		Expect(virtClient.Fake.Actions()).To(HaveLen(1))
	},
		Entry("When cluster-wide eviction strategy is missing, VMI eviction strategy is LiveMigrate and VMI is not migratable",
			nil,
			libvmi.WithEvictionStrategy(virtv1.EvictionStrategyLiveMigrate),
		),
		Entry("When cluster-wide eviction strategy is LiveMigrate, VMI eviction strategy is missing and VMI is not migratable",
			pointer.P(virtv1.EvictionStrategyLiveMigrate),
		),
	)

	It("should deny the request when the admitter fails to fetch the VMI", func() {
		vmi := libvmi.New(defaultVMIOptions...)
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
			context.Background(),
			newAdmissionReview(evictedVirtLauncherPod.Namespace, evictedVirtLauncherPod.Name, &dryRunOptions{}),
		)

		Expect(actualAdmissionResponse).To(Equal(expectedAdmissionResponse))
		Expect(kubeClient.Fake.Actions()).To(HaveLen(1))
		Expect(virtClient.Fake.Actions()).To(HaveLen(1))
	})

	It("should deny the request when the admitter fails to patch the VMI", func() {
		vmiOptions := append(defaultVMIOptions,
			libvmi.WithEvictionStrategy(virtv1.EvictionStrategyLiveMigrate),
			withLiveMigratableCondition(),
		)

		migratableVMI := libvmi.New(vmiOptions...)
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
			context.Background(),
			newAdmissionReview(evictedVirtLauncherPod.Namespace, evictedVirtLauncherPod.Name, &dryRunOptions{}),
		)

		Expect(actualAdmissionResponse).To(Equal(expectedAdmissionResponse))
		Expect(kubeClient.Fake.Actions()).To(HaveLen(1))

		patchBytes, err := patch.New(patch.WithAdd("/status/evacuationNodeName", migratableVMI.Status.NodeName)).GeneratePayload()
		Expect(err).To(Not(HaveOccurred()))
		Expect(virtClient.Actions()).To(ContainElement(newExpectedJSONPatchToVMI(migratableVMI, patchBytes, metav1.PatchOptions{})))
	})

	It("should deny the request and not mark the VMI again when the VMI is already marked for evacuation", func() {
		vmiOptions := append(defaultVMIOptions,
			libvmi.WithEvictionStrategy(virtv1.EvictionStrategyLiveMigrate),
			withLiveMigratableCondition(),
			withEvacuationNodeName(testNodeName),
		)

		migratableVMI := libvmi.New(vmiOptions...)
		virtClient := kubevirtfake.NewSimpleClientset(migratableVMI)
		virtClient.AddReactor("*", "*", func(_ testing.Action) (handled bool, ret runtime.Object, err error) {
			Fail("Not rest call should be made")
			return
		})

		evictedVirtLauncherPod := newVirtLauncherPod(migratableVMI.Namespace, migratableVMI.Name, migratableVMI.Status.NodeName)
		kubeClient := fake.NewSimpleClientset(evictedVirtLauncherPod)

		admitter := admitters.NewPodEvictionAdmitter(
			newClusterConfig(nil),
			kubeClient,
			virtClient,
		)

		actualAdmissionResponse := admitter.Admit(
			context.Background(),
			newAdmissionReview(evictedVirtLauncherPod.Namespace, evictedVirtLauncherPod.Name, &dryRunOptions{}),
		)

		Expect(actualAdmissionResponse).To(Equal(newDeniedAdmissionResponse(fmt.Sprintf(`Evacuation in progress: Eviction triggered evacuation of VMI "%s/%s"`, migratableVMI.Namespace, migratableVMI.Name))))
		Expect(kubeClient.Fake.Actions()).To(HaveLen(1))
		Expect(virtClient.Fake.Actions()).To(HaveLen(1))
	})

	DescribeTable("should deny the request and perform a dryRun patch on the VMI when", func(dryRunOpts *dryRunOptions) {
		vmiOptions := append(defaultVMIOptions,
			libvmi.WithEvictionStrategy(virtv1.EvictionStrategyLiveMigrate),
			withLiveMigratableCondition(),
		)

		migratableVMI := libvmi.New(vmiOptions...)
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
			context.Background(),
			newAdmissionReview(evictedVirtLauncherPod.Namespace, evictedVirtLauncherPod.Name, dryRunOpts),
		)

		Expect(actualAdmissionResponse).To(Equal(expectedAdmissionResponse))
		Expect(kubeClient.Fake.Actions()).To(HaveLen(1))

		patchBytes, err := patch.New(patch.WithAdd("/status/evacuationNodeName", migratableVMI.Status.NodeName)).GeneratePayload()
		Expect(err).To(Not(HaveOccurred()))
		Expect(virtClient.Actions()).To(ContainElement(newExpectedJSONPatchToVMI(migratableVMI, patchBytes, metav1.PatchOptions{DryRun: []string{metav1.DryRunAll}})))
	},

		Entry("dry run is set in the request", &dryRunOptions{dryRunInRequest: true}),
		Entry("dry run is set in the object", &dryRunOptions{dryRunInObject: []string{metav1.DryRunAll}}),
	)
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

func newVirtLauncherPodWithPhase(namespace, vmiName, nodeName string, phase k8sv1.PodPhase) *k8sv1.Pod {
	pod := newVirtLauncherPod(namespace, vmiName, nodeName)
	pod.Status.Phase = phase
	return pod
}

type dryRunOptions struct {
	dryRunInRequest bool
	dryRunInObject  []string
}

func newAdmissionReview(evictedPodNamespace, evictedPodName string, dryRunOpts *dryRunOptions) *admissionv1.AdmissionReview {
	obj := &policyv1.Eviction{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1",
			Kind:       "Eviction",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: evictedPodNamespace,
			Name:      evictedPodName,
		},
		DeleteOptions: &metav1.DeleteOptions{
			DryRun: dryRunOpts.dryRunInObject,
		},
	}
	rawObj, err := json.Marshal(obj)
	Expect(err).To(Not(HaveOccurred()))
	return &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Namespace: evictedPodNamespace,
			Name:      evictedPodName,
			DryRun:    pointer.P(dryRunOpts.dryRunInRequest),
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
				Object: obj,
				Raw:    rawObj,
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

func newExpectedJSONPatchToVMI(vmi *virtv1.VirtualMachineInstance, expectedJSONPatchData []byte, patchOpts metav1.PatchOptions) testing.PatchActionImpl {
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
		Name:         vmi.Name,
		PatchType:    types.JSONPatchType,
		Patch:        expectedJSONPatchData,
		PatchOptions: patchOpts,
	}
}

func withStatusNodeName(nodeName string) libvmi.Option {
	return func(vmi *virtv1.VirtualMachineInstance) {
		vmi.Status.NodeName = nodeName
	}
}

func withLiveMigratableCondition() libvmi.Option {
	return func(vmi *virtv1.VirtualMachineInstance) {
		vmi.Status.Conditions = append(vmi.Status.Conditions, virtv1.VirtualMachineInstanceCondition{
			Type:   virtv1.VirtualMachineInstanceIsMigratable,
			Status: k8sv1.ConditionTrue,
		})
	}
}

func withEvacuationNodeName(evacuationNodeName string) libvmi.Option {
	return func(vmi *virtv1.VirtualMachineInstance) {
		vmi.Status.EvacuationNodeName = evacuationNodeName
	}
}
