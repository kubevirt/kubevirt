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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package testsuite

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"kubevirt.io/kubevirt/tests/framework/matcher"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"k8s.io/apimachinery/pkg/api/errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/util"
)

const (
	defaultEventuallyTimeout         = 5 * time.Second
	defaultEventuallyPollingInterval = 1 * time.Second
	defaultKubevirtReadyTimeout      = 180 * time.Second
)

const HostPathBase = "/tmp/hostImages"

var (
	HostPathAlpine string
	HostPathCustom string
)

var Arch string

func SynchronizedAfterTestSuiteCleanup() {
	RestoreKubeVirtResource()

	libnode.CleanNodes()
}

func AfterTestSuiteCleanup() {

	cleanupServiceAccounts()
	CleanNamespaces()

	removeNamespaces()
}

func SynchronizedBeforeTestSetup() []byte {
	var err error
	libstorage.Config, err = libstorage.LoadConfig()
	Expect(err).ToNot(HaveOccurred())

	if flags.KubeVirtInstallNamespace == "" {
		detectInstallNamespace()
	}

	if flags.DeployTestingInfrastructureFlag {
		WipeTestingInfrastructure()
		DeployTestingInfrastructure()
	}

	EnsureKVMPresent()
	AdjustKubeVirtResource()
	EnsureKubevirtReady()

	return nil
}

func BeforeTestSuiteSetup(_ []byte) {

	worker := GinkgoParallelProcess()
	rand.Seed(int64(worker))
	log.InitializeLogging("tests")
	log.Log.SetIOWriter(GinkgoWriter)
	var err error
	libstorage.Config, err = libstorage.LoadConfig()
	Expect(err).ToNot(HaveOccurred())
	Arch = libnode.GetArch()

	// Customize host disk paths
	// Right now we support three nodes. More image copying needs to happen
	// TODO link this somehow with the image provider which we run upfront

	HostPathAlpine = filepath.Join(HostPathBase, fmt.Sprintf("%s%v", "alpine", worker))
	HostPathCustom = filepath.Join(HostPathBase, fmt.Sprintf("%s%v", "custom", worker))

	// Wait for schedulable nodes
	virtClient := kubevirt.Client()
	initRunConfiguration(virtClient)
	Eventually(func() int {
		nodes := libnode.GetAllSchedulableNodes(virtClient)
		if len(nodes.Items) > 0 {
			idx := rand.Intn(len(nodes.Items))
			libnode.SchedulableNode = nodes.Items[idx].Name
		}
		return len(nodes.Items)
	}, 5*time.Minute, 10*time.Second).ShouldNot(BeZero(), "no schedulable nodes found")

	createNamespaces()
	createServiceAccounts()

	SetDefaultEventuallyTimeout(defaultEventuallyTimeout)
	SetDefaultEventuallyPollingInterval(defaultEventuallyPollingInterval)
}

func EnsureKubevirtReady() {
	EnsureKubevirtReadyWithTimeout(defaultKubevirtReadyTimeout)
}

func EnsureKubevirtReadyWithTimeout(timeout time.Duration) {
	virtClient := kubevirt.Client()
	kv := util.GetCurrentKv(virtClient)

	Eventually(matcher.ThisDeploymentWith(flags.KubeVirtInstallNamespace, "virt-operator"), 180*time.Second, 1*time.Second).
		Should(matcher.HaveReadyReplicasNumerically(">", 0),
			"virt-operator deployment is not ready")

	Eventually(func() *v1.KubeVirt {
		kv, err := virtClient.KubeVirt(kv.Namespace).Get(kv.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return kv
	}, timeout, 1*time.Second).Should(
		SatisfyAll(
			matcher.HaveConditionTrue(v1.KubeVirtConditionAvailable),
			matcher.HaveConditionFalse(v1.KubeVirtConditionProgressing),
			matcher.HaveConditionFalse(v1.KubeVirtConditionDegraded),
			WithTransform(func(kv *v1.KubeVirt) bool {
				return kv.ObjectMeta.Generation == *kv.Status.ObservedGeneration
			}, BeTrue()),
		), "One of the Kubevirt control-plane components is not ready.")

}

func EnsureKVMPresent() {
	virtClient := kubevirt.Client()

	if !ShouldAllowEmulation(virtClient) {
		listOptions := metav1.ListOptions{LabelSelector: v1.AppLabel + "=virt-handler"}
		virtHandlerPods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), listOptions)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		EventuallyWithOffset(1, func() bool {
			ready := true
			// cluster is not ready until all nodes are ready.
			for _, pod := range virtHandlerPods.Items {
				virtHandlerNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), pod.Spec.NodeName, metav1.GetOptions{})
				ExpectWithOffset(1, err).ToNot(HaveOccurred())

				kvmAllocatable, ok1 := virtHandlerNode.Status.Allocatable[services.KvmDevice]
				vhostNetAllocatable, ok2 := virtHandlerNode.Status.Allocatable[services.VhostNetDevice]
				ready = ready && ok1 && ok2
				ready = ready && (kvmAllocatable.Value() > 0) && (vhostNetAllocatable.Value() > 0)
			}
			return ready
		}, 120*time.Second, 1*time.Second).Should(BeTrue(),
			"Both KVM devices and vhost-net devices are required for testing, but are not present on cluster nodes")
	}
}

func deployOrWipeTestingInfrastrucure(actionOnObject func(unstructured.Unstructured) error) {
	// Deploy / delete test infrastructure / dependencies
	manifests := GetListOfManifests(flags.PathToTestingInfrastrucureManifests)
	for _, manifest := range manifests {
		objects := ReadManifestYamlFile(manifest)
		for _, obj := range objects {
			err := actionOnObject(obj)
			util.PanicOnError(err)
		}
	}

	waitForAllDaemonSetsReady(3 * time.Minute)
	waitForAllPodsReady(3*time.Minute, metav1.ListOptions{})
}

func DeployTestingInfrastructure() {
	deployOrWipeTestingInfrastrucure(ApplyRawManifest)
}

func WipeTestingInfrastructure() {
	deployOrWipeTestingInfrastrucure(DeleteRawManifest)
}

func waitForAllDaemonSetsReady(timeout time.Duration) {
	checkForDaemonSetsReady := func() []string {
		dsNotReady := make([]string, 0)
		virtClient := kubevirt.Client()

		dsList, err := virtClient.AppsV1().DaemonSets(k8sv1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
		util.PanicOnError(err)
		for _, ds := range dsList.Items {
			if ds.Status.DesiredNumberScheduled != ds.Status.NumberReady {
				dsNotReady = append(dsNotReady, ds.Name)
			}
		}
		return dsNotReady

	}
	Eventually(checkForDaemonSetsReady, timeout, 2*time.Second).Should(BeEmpty(), "There are daemonsets in system which are not ready.")
}

func waitForAllPodsReady(timeout time.Duration, listOptions metav1.ListOptions) {
	checkForPodsToBeReady := func() []string {
		podsNotReady := make([]string, 0)
		virtClient := kubevirt.Client()

		podsList, err := virtClient.CoreV1().Pods(k8sv1.NamespaceAll).List(context.Background(), listOptions)
		util.PanicOnError(err)
		for _, pod := range podsList.Items {
			for _, status := range pod.Status.ContainerStatuses {
				if status.State.Terminated != nil {
					break // We don't care about terminated pods
				} else if status.State.Running != nil {
					if !status.Ready { // We need to wait for this one
						podsNotReady = append(podsNotReady, pod.Name)
						break
					}
				} else {
					// It is in Waiting state, We need to wait for this one
					podsNotReady = append(podsNotReady, pod.Name)
					break
				}
			}
		}
		return podsNotReady
	}
	Eventually(checkForPodsToBeReady, timeout, 2*time.Second).Should(BeEmpty(), "There are pods in system which are not ready.")
}

func WaitExportProxyReady() {
	Eventually(func() bool {
		virtClient := kubevirt.Client()
		d, err := virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(context.TODO(), "virt-exportproxy", metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return false
		}
		Expect(err).ToNot(HaveOccurred())
		return d.Status.AvailableReplicas > 0
	}, 90*time.Second, 1*time.Second).Should(BeTrue())
}
