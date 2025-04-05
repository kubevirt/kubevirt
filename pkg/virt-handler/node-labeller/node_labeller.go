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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package nodelabeller

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"k8s.io/client-go/tools/record"
	"libvirt.org/go/libvirtxml"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	k8scli "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/workqueue"

	kubevirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var nodeLabellerLabels = []string{
	kubevirtv1.CPUFeatureLabel,
	kubevirtv1.CPUModelLabel,
	kubevirtv1.SupportedHostModelMigrationCPU,
	kubevirtv1.CPUTimerLabel,
	kubevirtv1.HypervLabel,
	kubevirtv1.RealtimeLabel,
	kubevirtv1.SEVLabel,
	kubevirtv1.SEVESLabel,
	kubevirtv1.HostModelCPULabel,
	kubevirtv1.HostModelRequiredFeaturesLabel,
	kubevirtv1.NodeHostModelIsObsoleteLabel,
	kubevirtv1.SupportedMachineTypeLabel,
}

// NodeLabeller struct holds information needed to run node-labeller
type NodeLabeller struct {
	recorder                record.EventRecorder
	nodeClient              k8scli.NodeInterface
	host                    string
	logger                  *log.FilteredLogger
	clusterConfig           *virtconfig.ClusterConfig
	hypervFeatures          supportedFeatures
	hostCapabilities        supportedFeatures
	queue                   workqueue.TypedRateLimitingInterface[string]
	supportedFeatures       []string
	cpuModelVendor          string
	volumePath              string
	domCapabilitiesFileName string
	cpuCounter              *libvirtxml.CapsHostCPUCounter
	guestCaps               []libvirtxml.CapsGuest
	hostCPUModel            hostCPUModel
	SEV                     SEVConfiguration
	arch                    string
}

func NewNodeLabeller(clusterConfig *virtconfig.ClusterConfig, nodeClient k8scli.NodeInterface, host string, recorder record.EventRecorder, cpuCounter *libvirtxml.CapsHostCPUCounter, guestCaps []libvirtxml.CapsGuest) (*NodeLabeller, error) {
	return newNodeLabeller(clusterConfig, nodeClient, host, NodeLabellerVolumePath, recorder, cpuCounter, guestCaps)

}
func newNodeLabeller(clusterConfig *virtconfig.ClusterConfig, nodeClient k8scli.NodeInterface, host, volumePath string, recorder record.EventRecorder, cpuCounter *libvirtxml.CapsHostCPUCounter, guestCaps []libvirtxml.CapsGuest) (*NodeLabeller, error) {
	n := &NodeLabeller{
		recorder:      recorder,
		nodeClient:    nodeClient,
		host:          host,
		logger:        log.DefaultLogger(),
		clusterConfig: clusterConfig,
		queue: workqueue.NewTypedRateLimitingQueueWithConfig[string](
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "virt-handler-node-labeller"},
		),
		volumePath:              volumePath,
		domCapabilitiesFileName: "virsh_domcapabilities.xml",
		cpuCounter:              cpuCounter,
		guestCaps:               guestCaps,
		hostCPUModel:            hostCPUModel{requiredFeatures: make(map[string]bool, 0)},
		arch:                    runtime.GOARCH,
	}

	err := n.loadAll()
	if err != nil {
		return n, err
	}
	return n, nil
}

// Run runs node-labeller
func (n *NodeLabeller) Run(threadiness int, stop chan struct{}) {
	defer n.queue.ShutDown()

	n.logger.Infof("node-labeller is running")

	if !n.hasTSCCounter() {
		n.logger.Error("failed to get tsc cpu frequency, will continue without the tsc frequency label")
	}

	n.clusterConfig.SetConfigModifiedCallback(func() {
		n.queue.Add(n.host)
	})

	interval := 3 * time.Minute
	go wait.JitterUntil(func() { n.queue.Add(n.host) }, interval, 1.2, true, stop)

	for i := 0; i < threadiness; i++ {
		go wait.Until(n.runWorker, time.Second, stop)
	}
	<-stop
}

func (n *NodeLabeller) runWorker() {
	for n.execute() {
	}
}

func (n *NodeLabeller) execute() bool {
	key, quit := n.queue.Get()
	if quit {
		return false
	}
	defer n.queue.Done(key)

	err := n.run()

	if err != nil {
		n.logger.Errorf("node-labeller sync error encountered: %v", err)
		n.queue.AddRateLimited(key)
	} else {
		n.queue.Forget(key)
	}
	return true
}

func (n *NodeLabeller) loadAll() error {
	// host supported features is only available on AMD64 and S390X nodes.
	// This is because hypervisor-cpu-baseline virsh command doesnt work for ARM64 architecture.
	if virtconfig.IsAMD64(n.arch) || virtconfig.IsS390X(n.arch) {
		err := n.loadHostSupportedFeatures()
		if err != nil {
			n.logger.Errorf("node-labeller could not load supported features: " + err.Error())
			return err
		}
	}

	err := n.loadDomCapabilities()
	if err != nil {
		n.logger.Errorf("node-labeller could not load host dom capabilities: " + err.Error())
		return err
	}

	n.loadHypervFeatures()

	return nil
}

func (n *NodeLabeller) run() error {
	obsoleteCPUsx86 := n.clusterConfig.GetObsoleteCPUModels()
	cpuModels := n.getSupportedCpuModels(obsoleteCPUsx86)
	cpuFeatures := n.getSupportedCpuFeatures()
	hostCPUModel := n.GetHostCpuModel()

	originalNode, err := n.nodeClient.Get(context.Background(), n.host, metav1.GetOptions{})
	if err != nil {
		return err
	}

	node := originalNode.DeepCopy()

	if !skipNodeLabelling(node) {
		//prepare new labels
		newLabels := n.prepareLabels(node, cpuModels, cpuFeatures, hostCPUModel, obsoleteCPUsx86)
		//remove old labeller labels
		n.removeLabellerLabels(node)
		//add new labels
		n.addLabellerLabels(node, newLabels)
	}

	err = n.patchNode(originalNode, node)

	return err
}

func skipNodeLabelling(node *v1.Node) bool {
	_, exists := node.Annotations[kubevirtv1.LabellerSkipNodeAnnotation]
	return exists
}

func (n *NodeLabeller) patchNode(originalNode, node *v1.Node) error {
	if equality.Semantic.DeepEqual(originalNode.Labels, node.Labels) {
		return nil
	}

	patchBytes, err := patch.New(
		patch.WithTest("/metadata/labels", originalNode.Labels),
		patch.WithReplace("/metadata/labels", node.Labels),
	).GeneratePayload()

	if err != nil {
		return err
	}

	_, err = n.nodeClient.Patch(context.Background(), node.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	return err
}

func (n *NodeLabeller) loadHypervFeatures() {
	n.hypervFeatures.items = getCapLabels()
}

// prepareLabels converts cpu models, features, hyperv features to map[string]string format
// e.g. "cpu-feature.node.kubevirt.io/Penryn": "true"
func (n *NodeLabeller) prepareLabels(node *v1.Node, cpuModels []string, cpuFeatures cpuFeatures, hostCpuModel hostCPUModel, obsoleteCPUsx86 map[string]bool) map[string]string {
	newLabels := make(map[string]string)
	for key := range cpuFeatures {
		newLabels[kubevirtv1.CPUFeatureLabel+key] = "true"
	}

	for _, value := range cpuModels {
		newLabels[kubevirtv1.CPUModelLabel+value] = "true"
		newLabels[kubevirtv1.SupportedHostModelMigrationCPU+value] = "true"
	}

	// Add labels for supported machine types
	machines := n.getSupportedMachines()

	for _, machine := range machines {
		labelKey := kubevirtv1.SupportedMachineTypeLabel + machine.Name
		newLabels[labelKey] = "true"
	}

	if _, hostModelObsolete := obsoleteCPUsx86[hostCpuModel.Name]; !hostModelObsolete {
		newLabels[kubevirtv1.SupportedHostModelMigrationCPU+hostCpuModel.Name] = "true"
	}

	for _, key := range n.hypervFeatures.items {
		newLabels[kubevirtv1.HypervLabel+key] = "true"
	}

	if n.hasTSCCounter() {
		newLabels[kubevirtv1.CPUTimerLabel+"tsc-frequency"] = fmt.Sprintf("%d", n.cpuCounter.Frequency)
		newLabels[kubevirtv1.CPUTimerLabel+"tsc-scalable"] = fmt.Sprintf("%t", n.cpuCounter.Scaling == "yes")
	}

	for feature := range hostCpuModel.requiredFeatures {
		newLabels[kubevirtv1.HostModelRequiredFeaturesLabel+feature] = "true"
	}
	if _, obsolete := obsoleteCPUsx86[hostCpuModel.Name]; obsolete {
		newLabels[kubevirtv1.NodeHostModelIsObsoleteLabel] = "true"
		err := n.alertIfHostModelIsObsolete(node, hostCpuModel.Name, obsoleteCPUsx86)
		if err != nil {
			n.logger.Reason(err).Error(err.Error())
		}
	}

	newLabels[kubevirtv1.CPUModelVendorLabel+n.cpuModelVendor] = "true"
	newLabels[kubevirtv1.HostModelCPULabel+hostCpuModel.Name] = "true"

	capable, err := isNodeRealtimeCapable()
	if err != nil {
		n.logger.Reason(err).Error("failed to identify if a node is capable of running realtime workloads")
	}
	if capable {
		newLabels[kubevirtv1.RealtimeLabel] = ""
	}

	if n.SEV.Supported == "yes" {
		newLabels[kubevirtv1.SEVLabel] = ""
	}

	if n.SEV.SupportedES == "yes" {
		newLabels[kubevirtv1.SEVESLabel] = ""
	}

	return newLabels
}

// addNodeLabels adds labels to node.
func (n *NodeLabeller) addLabellerLabels(node *v1.Node, labels map[string]string) {
	for key, value := range labels {
		node.Labels[key] = value
	}
}

// removeLabellerLabels removes labels from node
func (n *NodeLabeller) removeLabellerLabels(node *v1.Node) {
	for label := range node.Labels {
		if isNodeLabellerLabel(label) {
			delete(node.Labels, label)
		}
	}
}

const kernelSchedRealtimeRuntimeInMicrosecods = "kernel.sched_rt_runtime_us"

// isNodeRealtimeCapable Checks if a node is capable of running realtime workloads. Currently by validating if the kernel system setting value
// for `kernel.sched_rt_runtime_us` is set to allow running realtime scheduling with unlimited time (==-1)
// TODO: This part should be improved to validate against key attributes that determine best if a host is able to run realtime
// workloads at peak performance.

func isNodeRealtimeCapable() (bool, error) {
	ret, err := exec.Command("sysctl", kernelSchedRealtimeRuntimeInMicrosecods).CombinedOutput()
	if err != nil {
		return false, err
	}
	st := strings.Trim(string(ret), "\n")
	return fmt.Sprintf("%s = -1", kernelSchedRealtimeRuntimeInMicrosecods) == st, nil
}

func isNodeLabellerLabel(label string) bool {
	for _, prefix := range nodeLabellerLabels {
		if strings.HasPrefix(label, prefix) {
			return true
		}
	}

	return false
}

func (n *NodeLabeller) alertIfHostModelIsObsolete(originalNode *v1.Node, hostModel string, ObsoleteCPUModels map[string]bool) error {
	warningMsg := fmt.Sprintf("This node has %v host-model cpu that is included in ObsoleteCPUModels: %v", hostModel, ObsoleteCPUModels)
	n.recorder.Eventf(originalNode, v1.EventTypeWarning, "HostModelIsObsolete", warningMsg)
	return nil
}

func (n *NodeLabeller) hasTSCCounter() bool {
	return n.cpuCounter != nil && n.cpuCounter.Name == "tsc"
}

func (n *NodeLabeller) getSupportedMachines() []libvirtxml.CapsGuestMachine {
	var supportedMachines []libvirtxml.CapsGuestMachine
	for _, guest := range n.guestCaps {
		supportedMachines = append(supportedMachines, guest.Arch.Machines...)
	}
	return supportedMachines
}
