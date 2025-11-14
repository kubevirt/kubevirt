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

package tests_test

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/k8s"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
)

var _ = Describe("[sig-operator]virt-handler canary upgrade", Serial, decorators.SigOperator, func() {

	var originalKV *v1.KubeVirt
	var virtCli kubecli.KubevirtClient
	var dsInformer cache.SharedIndexInformer
	var stopCh chan struct{}
	var lastObservedEvent string
	var updateTimeout time.Duration

	const (
		e2eCanaryTestAnnotation = "e2e-canary-test"
		canaryTestNodeTimeout   = 60
	)

	BeforeEach(func() {
		virtCli = kubevirt.Client()

		originalKV = libkubevirt.GetCurrentKv(virtCli)

		stopCh = make(chan struct{})
		lastObservedEvent = ""

		informerFactory := informers.NewSharedInformerFactoryWithOptions(k8s.Client(), 0, informers.WithNamespace(flags.KubeVirtInstallNamespace), informers.WithTweakListOptions(func(opts *metav1.ListOptions) {
			opts.LabelSelector = "kubevirt.io=virt-handler"
		}))
		dsInformer = informerFactory.Apps().V1().DaemonSets().Informer()

		ds, err := k8s.Client().AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		nodesToUpdate := ds.Status.DesiredNumberScheduled
		Expect(nodesToUpdate).To(BeNumerically(">", 0))
		updateTimeout = time.Duration(canaryTestNodeTimeout * nodesToUpdate)
	})

	AfterEach(func() {
		close(stopCh)
		patchPayload, err := patch.New(patch.WithReplace("/spec/customizeComponents", originalKV.Spec.CustomizeComponents)).GeneratePayload()
		Expect(err).ToNot(HaveOccurred())
		_, err = virtCli.KubeVirt(flags.KubeVirtInstallNamespace).Patch(context.Background(), originalKV.Name, types.JSONPatchType, patchPayload, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func(g Gomega) {
			ds, err := k8s.Client().AppsV1().DaemonSets(originalKV.Namespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(ds.Status.DesiredNumberScheduled).To(Equal(ds.Status.NumberReady))
			g.Expect(ds.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable.IntValue()).To(Equal(1))
		}, updateTimeout*time.Second, 1*time.Second).Should(Succeed(), "waiting for virt-handler to be ready")
	})

	updateVirtHandler := func() error {
		testPatch := fmt.Sprintf(`{"spec": { "template": {"metadata": {"annotations": {"%s": "test"}}}}}`,
			e2eCanaryTestAnnotation)

		ccs := v1.CustomizeComponents{
			Patches: []v1.CustomizeComponentsPatch{
				{
					ResourceName: "virt-handler",
					ResourceType: "DaemonSet",
					Type:         v1.StrategicMergePatchType,
					Patch:        testPatch,
				},
			},
		}

		patchPayload, err := patch.New(patch.WithReplace("/spec/customizeComponents", ccs)).GeneratePayload()
		if err != nil {
			return err
		}
		_, err = virtCli.KubeVirt(flags.KubeVirtInstallNamespace).Patch(context.Background(), originalKV.Name, types.JSONPatchType, patchPayload, metav1.PatchOptions{})
		return err
	}

	removeExpectedAtHead := func(eventsQueue []string, expectedEvent string) []string {
		defer GinkgoRecover()

		if expectedEvent == lastObservedEvent {
			return eventsQueue
		}
		if len(eventsQueue) == 0 {
			return eventsQueue
		}

		head := eventsQueue[0]
		ExpectWithOffset(1, head).To(Equal(expectedEvent), fmt.Sprintf("was expecting event %s but got %s instead", expectedEvent, head))
		lastObservedEvent = expectedEvent
		return eventsQueue[1:]
	}

	processDsEvent := func(ds *appsv1.DaemonSet, eventsQueue []string) []string {
		update := ds.Spec.UpdateStrategy.RollingUpdate
		if update == nil {
			return eventsQueue
		}
		maxUnavailable := update.MaxUnavailable
		if maxUnavailable == nil {
			return eventsQueue
		}

		if maxUnavailable.String() == "10%" {
			return removeExpectedAtHead(eventsQueue, "maxUnavailable=10%")
		}

		if maxUnavailable.IntValue() == 1 {
			return removeExpectedAtHead(eventsQueue, "maxUnavailable=1")
		}

		return eventsQueue
	}

	It("should successfully upgrade virt-handler", decorators.RequiresTwoSchedulableNodes, func() {
		var expectedEventsLock sync.Mutex
		expectedEvents := []string{
			"maxUnavailable=1",
			"maxUnavailable=10%",
			"maxUnavailable=1",
		}

		go dsInformer.Run(stopCh)
		cache.WaitForCacheSync(stopCh, dsInformer.HasSynced)

		dsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(old, curr interface{}) {
				ds := curr.(*appsv1.DaemonSet)
				expectedEventsLock.Lock()
				defer expectedEventsLock.Unlock()
				expectedEvents = processDsEvent(ds, expectedEvents)
			},
		})

		err := updateVirtHandler()
		Expect(err).ToNot(HaveOccurred())

		Eventually(func(g Gomega) {
			expectedEventsLock.Lock()
			defer expectedEventsLock.Unlock()
			g.Expect(expectedEvents).To(BeEmpty())
		}, updateTimeout*time.Second, 1*time.Second).Should(Succeed(), fmt.Sprintf("events %v were expected but did not occur", expectedEvents))

		Eventually(func(g Gomega) {
			ds, err := k8s.Client().AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(ds.Status.DesiredNumberScheduled).To(Equal(ds.Status.NumberReady))
			g.Expect(ds.Spec.Template.Annotations).To(HaveKey(e2eCanaryTestAnnotation))

			podList, err := k8s.Client().CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "kubevirt.io=virt-handler"})
			g.Expect(err).ToNot(HaveOccurred())

			updatedPods := slices.DeleteFunc(podList.Items, func(pod corev1.Pod) bool {
				_, exists := pod.Annotations[e2eCanaryTestAnnotation]
				return !exists
			})
			g.Expect(updatedPods).To(HaveLen(int(ds.Status.DesiredNumberScheduled)))
		}, updateTimeout*time.Second, 1*time.Second).Should(Succeed())
	})
})
