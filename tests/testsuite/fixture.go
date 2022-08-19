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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"

	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/util"
)

const (
	defaultEventuallyTimeout         = 5 * time.Second
	defaultEventuallyPollingInterval = 1 * time.Second
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
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
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

func EnsureKubevirtInfra() {
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
	kv := util.GetCurrentKv(virtClient)

	timeout := 180 * time.Second
	interval := 1 * time.Second

	deployments := []string{
		"virt-operator",
		components.VirtAPIName,
		components.VirtControllerName,
	}

	ensureDeployment := func(deploymentName string) {
		deployment, err := virtClient.
			AppsV1().
			Deployments(kv.Namespace).
			Get(context.Background(), deploymentName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		EventuallyWithOffset(
			1,
			matcher.ThisDeploymentWith(kv.Namespace, deploymentName),
			timeout,
			interval).
			Should(matcher.HaveReadyReplicasNumerically("==", *deployment.Spec.Replicas),
				"waiting for %s deployment to be ready", deploymentName)
	}

	for _, deploymentName := range deployments {
		ensureDeployment(deploymentName)
	}

	//TODO: implement matcher for Daemonset in test infra
	Eventually(func() bool {
		ds, err := virtClient.
			AppsV1().
			DaemonSets(kv.Namespace).
			Get(context.Background(), components.VirtHandlerName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		return ds.Status.DesiredNumberScheduled == ds.Status.NumberReady
	}, timeout, interval).Should(BeTrue(), "waiting for virt-handler daemonSet to be ready")

}

func EnsureKVMPresent() {
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

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

	waitForAllPodsReady(3*time.Minute, metav1.ListOptions{})
}

func DeployTestingInfrastructure() {
	deployOrWipeTestingInfrastrucure(ApplyRawManifest)
}

func WipeTestingInfrastructure() {
	deployOrWipeTestingInfrastrucure(DeleteRawManifest)
}

func waitForAllPodsReady(timeout time.Duration, listOptions metav1.ListOptions) {
	checkForPodsToBeReady := func() []string {
		podsNotReady := make([]string, 0)
		virtClient, err := kubecli.GetKubevirtClient()
		util.PanicOnError(err)

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
