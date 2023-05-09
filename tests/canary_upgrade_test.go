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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"

	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/util"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[Serial][sig-operator]virt-handler canary upgrade", Serial, decorators.SigOperator, func() {

	var originalKV *v1.KubeVirt
	var virtCli kubecli.KubevirtClient
	var dsInformer cache.SharedIndexInformer
	var stopCh chan struct{}
	var lastObservedEvent string

	const (
		e2eCanaryTestAnnotation = "e2e-canary-test"
		canaryTestNodeTimeout   = 60
	)

	BeforeEach(func() {
		if !checks.HasAtLeastTwoNodes() {
			Skip("this test requires at least 2 nodes")
		}

		virtCli = kubevirt.Client()

		originalKV = util.GetCurrentKv(virtCli).DeepCopy()

		stopCh = make(chan struct{})
		lastObservedEvent = ""

		informerFactory := informers.NewSharedInformerFactoryWithOptions(virtCli, 0, informers.WithNamespace(flags.KubeVirtInstallNamespace), informers.WithTweakListOptions(func(opts *metav1.ListOptions) {
			opts.LabelSelector = "kubevirt.io=virt-handler"
		}))
		dsInformer = informerFactory.Apps().V1().DaemonSets().Informer()
	})

	AfterEach(func() {
		close(stopCh)

		retry.RetryOnConflict(retry.DefaultRetry, func() error {
			_, err := virtCli.KubeVirt(flags.KubeVirtInstallNamespace).Update(originalKV)
			return err
		})

		Eventually(func() bool {
			ds, err := virtCli.AppsV1().DaemonSets(originalKV.Namespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return ds.Status.DesiredNumberScheduled == ds.Status.NumberReady && ds.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable.IntValue() == 1
		}, 60*time.Second, 1*time.Second).Should(BeTrue(), "waiting for virt-handler to be ready")
	})

	getVirtHandler := func() *appsv1.DaemonSet {
		daemonSet, err := virtCli.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		return daemonSet
	}

	updateVirtHandler := func() error {
		kv := util.GetCurrentKv(virtCli)

		patch := fmt.Sprintf(`{"spec": { "template": {"metadata": {"annotations": {"%s": "test"}}}}}`,
			e2eCanaryTestAnnotation)
		kv.Spec.CustomizeComponents = v1.CustomizeComponents{
			Patches: []v1.CustomizeComponentsPatch{
				{
					ResourceName: "virt-handler",
					ResourceType: "DaemonSet",
					Type:         v1.StrategicMergePatchType,
					Patch:        patch,
				},
			},
		}
		return retry.RetryOnConflict(retry.DefaultRetry, func() error {
			_, err := virtCli.KubeVirt(flags.KubeVirtInstallNamespace).Update(kv)
			return err
		})
	}

	isVirtHandlerUpdated := func(ds *appsv1.DaemonSet) bool {
		_, exists := ds.Spec.Template.Annotations[e2eCanaryTestAnnotation]
		return exists
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
			eventsQueue = removeExpectedAtHead(eventsQueue, "maxUnavailable=10%")
		}
		if maxUnavailable.IntValue() == 1 {
			eventsQueue = removeExpectedAtHead(eventsQueue, "maxUnavailable=1")
		}
		if ds.Status.DesiredNumberScheduled == ds.Status.NumberReady {
			pods, err := virtCli.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "kubevirt.io=virt-handler"})
			Expect(err).ToNot(HaveOccurred())

			var updatedPods int32
			for i := range pods.Items {
				if _, exists := pods.Items[i].Annotations[e2eCanaryTestAnnotation]; exists {
					updatedPods++
				}
			}

			if updatedPods > 0 && updatedPods == ds.Status.DesiredNumberScheduled {
				eventsQueue = removeExpectedAtHead(eventsQueue, "virt-handler=ready")
			}
		}
		return eventsQueue
	}

	It("should successfully upgrade virt-handler", func() {
		var expectedEventsLock sync.Mutex
		expectedEvents := []string{
			"maxUnavailable=1",
			"maxUnavailable=10%",
			"virt-handler=ready",
			"maxUnavailable=1",
		}

		ds, err := virtCli.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		nodesToUpdate := ds.Status.DesiredNumberScheduled
		Expect(nodesToUpdate).To(BeNumerically(">", 0))

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

		err = updateVirtHandler()
		Expect(err).ToNot(HaveOccurred())

		updateTimeout := time.Duration(canaryTestNodeTimeout * nodesToUpdate)
		Eventually(func() bool {
			expectedEventsLock.Lock()
			defer expectedEventsLock.Unlock()
			return len(expectedEvents) == 0
		}, updateTimeout*time.Second, 1*time.Second).Should(BeTrue(), fmt.Sprintf("events %v were expected but did not occur", expectedEvents))

		Expect(isVirtHandlerUpdated(getVirtHandler())).To(BeTrue())
	})
})
