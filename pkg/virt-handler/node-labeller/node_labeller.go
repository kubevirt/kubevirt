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

package nodelabeller

import (
	"context"
	"fmt"
	"os/exec"
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

var DefaultObsoleteCPUModels = map[string]bool{
	"486":           true,
	"486-v1":        true,
	"pentium":       true,
	"pentium-v1":    true,
	"pentium2":      true,
	"pentium2-v1":   true,
	"pentium3":      true,
	"pentium3-v1":   true,
	"pentiumpro":    true,
	"pentiumpro-v1": true,
	"coreduo":       true,
	"coreduo-v1":    true,
	"n270":          true,
	"n270-v1":       true,
	"core2duo":      true,
	"core2duo-v1":   true,
	"Conroe":        true,
	"Conroe-v1":     true,
	"athlon":        true,
	"athlon-v1":     true,
	"phenom":        true,
	"phenom-v1":     true,
	"qemu64":        true,
	"qemu64-v1":     true,
	"qemu32":        true,
	"qemu32-v1":     true,
	"kvm64":         true,
	"kvm64-v1":      true,
	"kvm32":         true,
	"kvm32-v1":      true,
	"Opteron_G1":    true,
	"Opteron_G1-v1": true,
	"Opteron_G2":    true,
	"Opteron_G2-v1": true,
}

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
	recorder            record.EventRecorder
	nodeClient          k8scli.NodeInterface
	host                string
	logger              *log.FilteredLogger
	clusterConfig       *virtconfig.ClusterConfig
	queue               workqueue.RateLimitingInterface
	supportedFeatures   []string
	cpuCounter          *libvirtxml.CapsHostCPUCounter
	guestCaps           []libvirtxml.CapsGuest
	hostCPUModel        string
	cpuModelVendor      string
	usableModels        []string
	cpuRequiredFeatures []string
	sevSupported        bool
	sevSupportedES      bool
	hypervFeatures      []string
}

func NewNodeLabeller(
	clusterConfig *virtconfig.ClusterConfig,
	nodeClient k8scli.NodeInterface,
	host string,
	recorder record.EventRecorder,
	supportedFeatures []string,
	cpuCounter *libvirtxml.CapsHostCPUCounter,
	guestCaps []libvirtxml.CapsGuest,
	hostCPUModel string,
	cpuModelVendor string,
	usableModels []string,
	cpuRequiredFeatures []string,
	sevSupported bool,
	sevSupportedES bool,
	hypervFeatures []string,
) *NodeLabeller {
	return &NodeLabeller{
		recorder:            recorder,
		nodeClient:          nodeClient,
		host:                host,
		logger:              log.DefaultLogger(),
		clusterConfig:       clusterConfig,
		queue:               workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-handler-node-labeller"),
		supportedFeatures:   supportedFeatures,
		cpuCounter:          cpuCounter,
		guestCaps:           guestCaps,
		hostCPUModel:        hostCPUModel,
		cpuModelVendor:      cpuModelVendor,
		usableModels:        usableModels,
		cpuRequiredFeatures: cpuRequiredFeatures,
		sevSupported:        sevSupported,
		sevSupportedES:      sevSupportedES,
		hypervFeatures:      hypervFeatures,
	}
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

func (n *NodeLabeller) run() error {
	originalNode, err := n.nodeClient.Get(context.Background(), n.host, metav1.GetOptions{})
	if err != nil {
		return err
	}

	node := originalNode.DeepCopy()

	if !skipNodeLabelling(node) {
		//prepare new labels
		newLabels := n.prepareLabels(node)
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

// prepareLabels converts cpu models, features, hyperv features to map[string]string format
// e.g. "cpu-feature.node.kubevirt.io/Penryn": "true"
func (n *NodeLabeller) prepareLabels(node *v1.Node) map[string]string {
	obsoleteCPUsx86 := n.clusterConfig.GetObsoleteCPUModels()
	newLabels := make(map[string]string)

	for _, value := range n.supportedFeatures {
		newLabels[kubevirtv1.CPUFeatureLabel+value] = "true"
	}

	for _, value := range n.supportedCPUModels(obsoleteCPUsx86) {
		newLabels[kubevirtv1.CPUModelLabel+value] = "true"
		newLabels[kubevirtv1.SupportedHostModelMigrationCPU+value] = "true"
	}

	for _, machine := range n.getSupportedMachines() {
		labelKey := kubevirtv1.SupportedMachineTypeLabel + machine.Name
		newLabels[labelKey] = "true"
	}

	for _, value := range n.hypervFeatures {
		newLabels[kubevirtv1.HypervLabel+value] = "true"
	}

	if n.hasTSCCounter() {
		newLabels[kubevirtv1.CPUTimerLabel+"tsc-frequency"] = fmt.Sprintf("%d", n.cpuCounter.Frequency)
		newLabels[kubevirtv1.CPUTimerLabel+"tsc-scalable"] = fmt.Sprintf("%t", n.cpuCounter.Scaling == "yes")
	}

	if n.hostCPUModel != "" {
		if _, hostModelObsolete := obsoleteCPUsx86[n.hostCPUModel]; !hostModelObsolete {
			newLabels[kubevirtv1.SupportedHostModelMigrationCPU+n.hostCPUModel] = "true"
		} else {
			newLabels[kubevirtv1.NodeHostModelIsObsoleteLabel] = "true"
			err := n.alertIfHostModelIsObsolete(node, n.hostCPUModel, obsoleteCPUsx86)
			if err != nil {
				n.logger.Reason(err).Error(err.Error())
			}
		}
		for _, feature := range n.cpuRequiredFeatures {
			newLabels[kubevirtv1.HostModelRequiredFeaturesLabel+feature] = "true"
		}

		newLabels[kubevirtv1.CPUModelVendorLabel+n.cpuModelVendor] = "true"
		newLabels[kubevirtv1.HostModelCPULabel+n.hostCPUModel] = "true"
	}

	capable, err := isNodeRealtimeCapable()
	if err != nil {
		n.logger.Reason(err).Error("failed to identify if a node is capable of running realtime workloads")
	}
	if capable {
		newLabels[kubevirtv1.RealtimeLabel] = ""
	}

	if n.sevSupported {
		newLabels[kubevirtv1.SEVLabel] = ""
	}

	if n.sevSupportedES {
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

func (n *NodeLabeller) supportedCPUModels(obsoleteCPUsx86 map[string]bool) []string {

	supportedCPUModels := make([]string, 0)

	if obsoleteCPUsx86 == nil {
		obsoleteCPUsx86 = DefaultObsoleteCPUModels
	}

	for _, model := range n.usableModels {
		if _, ok := obsoleteCPUsx86[model]; ok {
			continue
		}
		supportedCPUModels = append(supportedCPUModels, model)
	}

	return supportedCPUModels
}
