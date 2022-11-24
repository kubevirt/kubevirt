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

	util2 "kubevirt.io/kubevirt/pkg/virt-operator/util"

	"kubevirt.io/kubevirt/tests/framework/matcher"

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
	//var lastObservedEvent string

	type canaryUpgradePhase string

	var phase canaryUpgradePhase
	var transitionPhaseObserved bool
	var podUpdatedAndReady bool
	var phaseDescription string
	var onlyOnce sync.Once

	const (
		InProgressNotSwitchedToPercent canaryUpgradePhase = "InProgressNotSwitchedToPercent"
		InProgressSwitchedToPercent                       = "InProgressSwitchedToPercent"
		FinishedNotRestored                               = "FinishedNotRestored"
		FinishedRestored                                  = "FinishedRestored"
		NotHealthy                                        = "NotHealthy"
		NonRelevantPhase                                  = "NonRelevantPhase"
	)

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
		//lastObservedEvent = ""

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

		patch := fmt.Sprintf(`{"spec": { "template": {"metadata": {"annotations": {"%s": "test"}}}, "minReadySeconds": 10 }}`,
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

	checkIfPodUpdated := func() {
		pods, err := virtCli.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "kubevirt.io=virt-handler"})
		Expect(err).ToNot(HaveOccurred())

		for i := range pods.Items {
			if _, exists := pods.Items[i].Annotations[e2eCanaryTestAnnotation]; exists {
				if util2.PodIsReady(&pods.Items[i]) {
					podUpdatedAndReady = true
					break
				}
			}
		}
	}

	processDsEvent := func(ds *appsv1.DaemonSet) (phase canaryUpgradePhase, msg string) {
		maxUnavailable := ds.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable

		maxUnavailableStr := maxUnavailable.String()
		maxUnavailableInt := maxUnavailable.IntValue()
		updatedNumSched := ds.Status.UpdatedNumberScheduled
		desiredNumSched := ds.Status.DesiredNumberScheduled
		numReady := ds.Status.NumberReady

		switch {
		case maxUnavailableInt == 1:
			switch {
			case updatedNumSched == desiredNumSched && numReady == desiredNumSched:
				phase = FinishedRestored
				msg = fmt.Sprintf("Rollout finished and values restored")
			case updatedNumSched == desiredNumSched && numReady < desiredNumSched:
				phase = NotHealthy
				msg = fmt.Sprintf("Either operator restored the maxUnavailable too soon, or one of the pods isn't ready")
			case updatedNumSched < desiredNumSched:
				phase = InProgressNotSwitchedToPercent
				msg = fmt.Sprintf("this is the very beginning of the rollout, the first few rolled-out pods are not ready yet")
			default:
				phase = NonRelevantPhase
				msg = fmt.Sprintf("Should not end with this phase. maxUnavailableInt:1 updatedNumSched:%d desiredNumSched:%d numReady:%d",
					updatedNumSched, desiredNumSched, numReady)
			}
		case maxUnavailableStr == "10%":
			transitionPhaseObserved = true
			onlyOnce.Do(checkIfPodUpdated)
			switch {
			case updatedNumSched == desiredNumSched && numReady == desiredNumSched:
				phase = FinishedNotRestored
				msg = fmt.Sprintf("DS rollout finished but operator is still restoring maxUn")
			case numReady < desiredNumSched:
				phase = InProgressSwitchedToPercent
				msg = fmt.Sprintf("Rollout in progress, canary upgrade takes place")
			default:
				phase = NonRelevantPhase
				msg = fmt.Sprintf("Should not end with this phase. maxUnavailableStr:10-percent updatedNumSched:%d desiredNumSched:%d numReady:%d",
					updatedNumSched, desiredNumSched, numReady)
			}
		default:
			phase = NonRelevantPhase
			msg = fmt.Sprintf("virt-handler RollingUpdate.MaxUnavailable doesn't match any of test epxectations. maxUnavailableStr:%s maxUnavailableInt: %d",
				maxUnavailableStr, maxUnavailableInt)
		}

		return
	}

	It("should successfully upgrade virt-handler", func() {
		var expectedPhasesLock sync.Mutex

		By("Checking whether the virt-handler Daemonset is ready")
		ds, err := virtCli.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		nodesToUpdate := ds.Status.DesiredNumberScheduled
		Expect(nodesToUpdate).To(BeNumerically(">", 0), "There must be at least one candidate for upgrade")

		Expect(ds.Status.NumberAvailable).To(BeNumerically("==", ds.Status.DesiredNumberScheduled),
			"all the virt-handler pods should be ready on all the relevant nodes")

		if ds.Status.UpdatedNumberScheduled != 0 {
			Expect(ds.Status.UpdatedNumberScheduled).To(BeNumerically("==", ds.Status.DesiredNumberScheduled),
				"all the updated virt-handler pods should be ready")
		}

		Expect(ds.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable.IntValue()).To(BeNumerically("==", 1),
			"should start with rolling update maxUnavailable set to 1")

		By("Register Daemonset client-cache update callback")
		go dsInformer.Run(stopCh)
		cache.WaitForCacheSync(stopCh, dsInformer.HasSynced)
		dsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(old, curr interface{}) {
				ds := curr.(*appsv1.DaemonSet)
				expectedPhasesLock.Lock()
				defer expectedPhasesLock.Unlock()
				if ds.Spec.UpdateStrategy.RollingUpdate != nil {
					phase, phaseDescription = processDsEvent(ds)
				}
			},
		})

		By("Patching the virt-handler DS via customizeComponents")
		err = updateVirtHandler()
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for virt-operator to start the rollout")
		Eventually(func() *v1.KubeVirt {
			kv, err := virtCli.KubeVirt(originalKV.Namespace).Get(originalKV.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "Should get Kubevirt CR from API server")

			return kv
		}, 15*time.Second, 1*time.Second).Should(matcher.HaveConditionTrue(v1.KubeVirtConditionProgressing))

		updateTimeout := time.Duration(canaryTestNodeTimeout * nodesToUpdate)
		Eventually(func() bool {
			expectedPhasesLock.Lock()
			defer expectedPhasesLock.Unlock()
			return phase == FinishedRestored
		}, updateTimeout*time.Second, 1*time.Second).Should(BeTrue(),
			fmt.Sprintf("stuck at phase %s: %s", phase, phaseDescription))

		Expect(isVirtHandlerUpdated(getVirtHandler())).To(BeTrue())
		Expect(transitionPhaseObserved).To(BeTrue(), "RollingUpdate.maxUnavailable should be changed to 10% during the test")
		Expect(podUpdatedAndReady).To(BeTrue(), "RollingUpdate.maxUnavailable changed to 10% but updated pod wasn't in ready state")
	})
})
