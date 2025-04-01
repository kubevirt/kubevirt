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
	"os"
	"path/filepath"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
)

const (
	defaultEventuallyTimeout         = 5 * time.Second
	defaultEventuallyPollingInterval = 1 * time.Second
	defaultKubevirtReadyTimeout      = 5 * time.Minute
	defaultKWOKNodeCount             = 100
)

const HostPathBase = "/tmp/hostImages"

var (
	HostPathAlpine       string
	HostPathAlpineNoPriv string
	HostPathCustom       string
)

var Arch string

func SynchronizedAfterTestSuiteCleanup() {
	RestoreKubeVirtResource()

	if flags.DeployFakeKWOKNodesFlag {
		deleteFakeKWOKNodes()
	}

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

	if flags.DeployFakeKWOKNodesFlag {
		createFakeKWOKNodes()
	}

	EnsureKVMPresent()
	AdjustKubeVirtResource()
	EnsureKubevirtReady()

	return nil
}

func addTestAnnotation(vmi *v1.VirtualMachineInstance) {
	if vmi.Annotations == nil {
		vmi.Annotations = map[string]string{}
	}
	vmi.Annotations["kubevirt.io/created-by-test"] = GinkgoT().Name()
}

func nfsAntiAffinity(vmi *v1.VirtualMachineInstance) {
	for _, s := range CurrentSpecReport().Labels() {
		if s == "RequiresRWXFsVMStateStorageClass" {
			if vmi.Spec.Affinity == nil {
				vmi.Spec.Affinity = &k8sv1.Affinity{}
			}
			if vmi.Spec.Affinity.PodAntiAffinity == nil {
				vmi.Spec.Affinity.PodAntiAffinity = &k8sv1.PodAntiAffinity{}
			}
			vmi.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution =
				append(vmi.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
					k8sv1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "nfs-server"}},
						Namespaces:    []string{"nfs-csi"},
						TopologyKey:   k8sv1.LabelHostname,
					})
			break
		}
	}
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
	HostPathAlpineNoPriv = filepath.Join(HostPathBase, fmt.Sprintf("%s%v", "alpine-nopriv", worker))
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

	libvmifact.RegisterArchitecture(Arch)
	libvmi.RegisterDefaultOption(addTestAnnotation)
	libvmi.RegisterDefaultOption(libvmi.WithAutoattachGraphicsDevice(false))
	rwxfssc, found := libstorage.GetRWXFileSystemStorageClass()
	if found && rwxfssc == "nfs-csi" {
		// For tests that use nfs-csi as a storage class,
		// we need to ensure VMIs won't be scheduled on the same node as the NFS server
		libvmi.RegisterDefaultOption(nfsAntiAffinity)
	}
}

func EnsureKubevirtReady() {
	EnsureDefaultKubevirtReadyWithTimeout(defaultKubevirtReadyTimeout)
}

func EnsureDefaultKubevirtReadyWithTimeout(timeout time.Duration) {
	kv := libkubevirt.GetCurrentKv(kubevirt.Client())
	EnsureKubevirtReadyWithTimeout(kv, timeout)
}

func EnsureKubevirtReadyWithTimeout(kv *v1.KubeVirt, timeout time.Duration) {
	virtClient := kubevirt.Client()

	Eventually(matcher.ThisDeploymentWith(flags.KubeVirtInstallNamespace, "virt-operator"), 180*time.Second, 1*time.Second).
		Should(matcher.HaveReadyReplicasNumerically(">", 0),
			"virt-operator deployment is not ready")

	Eventually(func(g Gomega) *v1.KubeVirt {
		foundKV, err := virtClient.KubeVirt(kv.Namespace).Get(context.Background(), kv.Name, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		return foundKV
	}, timeout, 1*time.Second).Should(
		SatisfyAll(
			matcher.HaveConditionTrue(v1.KubeVirtConditionAvailable),
			matcher.HaveConditionFalse(v1.KubeVirtConditionProgressing),
			matcher.HaveConditionFalse(v1.KubeVirtConditionDegraded),
			matcher.HaveConditionTrue(v1.KubeVirtConditionCreated),
			Satisfy(func(kv *v1.KubeVirt) bool {
				return kv.ObjectMeta.Generation == *kv.Status.ObservedGeneration
			}),
		), "One of the Kubevirt control-plane components is not ready.")
}

func shouldAllowEmulation(virtClient kubecli.KubevirtClient) bool {
	allowEmulation := false

	kv := libkubevirt.GetCurrentKv(virtClient)
	if kv.Spec.Configuration.DeveloperConfiguration != nil {
		allowEmulation = kv.Spec.Configuration.DeveloperConfiguration.UseEmulation
	}

	return allowEmulation
}

func EnsureKVMPresent() {
	virtClient := kubevirt.Client()

	if !shouldAllowEmulation(virtClient) {
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

func deleteFakeKWOKNodes() {
	err := kubevirt.Client().CoreV1().Nodes().DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "type=kwok"})
	Expect(err).NotTo(HaveOccurred(), "failed to delete fake nodes")
}

// setup fake nodes for KWOK performance test
func createFakeKWOKNodes() {
	By("create fake Nodes")
	nodeCount := getKWOKNodeCount()
	for i := 1; i <= nodeCount; i++ {
		nodeName := fmt.Sprintf("kwok-node-%d", i)
		node := newFakeKWOKNode(nodeName)

		_, err := kubevirt.Client().CoreV1().Nodes().Create(context.TODO(), node, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to create node %s", nodeName))
	}

	By("Get the list of nodes")
	nodeList, err := kubevirt.Client().CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: "type=kwok"})
	Expect(err).NotTo(HaveOccurred(), "Failed to list fake nodes")
	Expect(nodeList.Items).To(HaveLen(nodeCount))
}

func newFakeKWOKNode(nodeName string) *k8sv1.Node {
	return &k8sv1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				"beta.kubernetes.io/arch":       "amd64",
				"beta.kubernetes.io/os":         "linux",
				"kubernetes.io/arch":            "amd64",
				"kubernetes.io/hostname":        nodeName,
				"kubernetes.io/os":              "linux",
				"kubernetes.io/role":            "agent",
				"node-role.kubernetes.io/agent": "",
				"kubevirt.io/schedulable":       "true",
				"type":                          "kwok",
			},
			Annotations: map[string]string{
				"node.alpha.kubernetes.io/ttl": "0",
				"kwok.x-k8s.io/node":           "fake",
			},
		},
		Spec: k8sv1.NodeSpec{
			Taints: []k8sv1.Taint{
				{
					Key:    "kwok.x-k8s.io/node",
					Value:  "fake",
					Effect: "NoSchedule",
				},
				{
					Key:    "CriticalAddonsOnly",
					Effect: k8sv1.TaintEffectNoSchedule,
				},
			},
		},
		Status: k8sv1.NodeStatus{
			Allocatable: k8sv1.ResourceList{
				k8sv1.ResourceCPU:               resource.MustParse("32"),
				k8sv1.ResourceMemory:            resource.MustParse("256Gi"),
				k8sv1.ResourceEphemeralStorage:  resource.MustParse("100Gi"),
				k8sv1.ResourcePods:              resource.MustParse("110"),
				"devices.kubevirt.io/kvm":       resource.MustParse("1k"),
				"devices.kubevirt.io/tun":       resource.MustParse("1k"),
				"devices.kubevirt.io/vhost-net": resource.MustParse("1k"),
			},
			Capacity: k8sv1.ResourceList{
				k8sv1.ResourceCPU:               resource.MustParse("32"),
				k8sv1.ResourceMemory:            resource.MustParse("256Gi"),
				k8sv1.ResourceEphemeralStorage:  resource.MustParse("100Gi"),
				k8sv1.ResourcePods:              resource.MustParse("110"),
				"devices.kubevirt.io/kvm":       resource.MustParse("1k"),
				"devices.kubevirt.io/tun":       resource.MustParse("1k"),
				"devices.kubevirt.io/vhost-net": resource.MustParse("1k"),
			},
		},
	}
}

func getKWOKNodeCount() int {
	vmCountString := os.Getenv("KWOK_NODE_COUNT")
	if vmCountString == "" {
		return defaultKWOKNodeCount
	}

	vmCount, err := strconv.Atoi(vmCountString)
	if err != nil {
		return defaultKWOKNodeCount
	}

	return vmCount
}

func deployOrWipeTestingInfrastrucure(actionOnObject func(unstructured.Unstructured) error) {
	// Deploy / delete test infrastructure / dependencies
	manifests := GetListOfManifests(flags.PathToTestingInfrastrucureManifests)
	for _, manifest := range manifests {
		objects := ReadManifestYamlFile(manifest)
		for _, obj := range objects {
			Expect(actionOnObject(obj)).To(Succeed())
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
		Expect(err).ToNot(HaveOccurred())
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
		Expect(err).ToNot(HaveOccurred())
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
