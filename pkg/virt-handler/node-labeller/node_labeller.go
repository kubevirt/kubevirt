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
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"k8s.io/client-go/tools/record"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	k8scli "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/workqueue"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/api"
)

var nodeLabellerLabels = []string{
	v1.CPUFeatureLabel,
	v1.CPUModelLabel,
	v1.SupportedHostModelMigrationCPU,
	v1.CPUTimerLabel,
	v1.HypervLabel,
	v1.RealtimeLabel,
	v1.SEVLabel,
	v1.SEVESLabel,
	v1.HostModelCPULabel,
	v1.HostModelRequiredFeaturesLabel,
	v1.NodeHostModelIsObsoleteLabel,
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
	queue                   workqueue.RateLimitingInterface
	supportedFeatures       []string
	cpuInfo                 cpuInfo
	cpuModelVendor          string
	volumePath              string
	domCapabilitiesFileName string
	capabilities            *api.Capabilities
	hostCPUModel            hostCPUModel
	SEV                     SEVConfiguration
}

func NewNodeLabeller(clusterConfig *virtconfig.ClusterConfig, nodeClient k8scli.NodeInterface, host string, recorder record.EventRecorder) (*NodeLabeller, error) {
	return newNodeLabeller(clusterConfig, nodeClient, host, nodeLabellerVolumePath, recorder)

}
func newNodeLabeller(clusterConfig *virtconfig.ClusterConfig, nodeClient k8scli.NodeInterface, host, volumePath string, recorder record.EventRecorder) (*NodeLabeller, error) {
	n := &NodeLabeller{
		recorder:                recorder,
		nodeClient:              nodeClient,
		host:                    host,
		logger:                  log.DefaultLogger(),
		clusterConfig:           clusterConfig,
		queue:                   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-handler-node-labeller"),
		volumePath:              volumePath,
		domCapabilitiesFileName: "virsh_domcapabilities.xml",
		hostCPUModel:            hostCPUModel{requiredFeatures: make(map[string]bool, 0)},
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
	err := n.loadCPUInfo()
	if err != nil {
		n.logger.Errorf("node-labeller could not load cpu info: " + err.Error())
		return err
	}

	// host supported features is only available on AMD64 nodes.
	// This is because hypervisor-cpu-baseline virsh command doesnt work for ARM64 architecture.
	if virtconfig.IsAMD64(runtime.GOARCH) {
		err = n.loadHostSupportedFeatures()
		if err != nil {
			n.logger.Errorf("node-labeller could not load supported features: " + err.Error())
			return err
		}
	}

	err = n.loadDomCapabilities()
	if err != nil {
		n.logger.Errorf("node-labeller could not load host dom capabilities: " + err.Error())
		return err
	}

	err = n.loadHostCapabilities()
	if err != nil {
		n.logger.Errorf("node-labeller could not load host capabilities: " + err.Error())
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

func skipNodeLabelling(node *k8sv1.Node) bool {
	_, exists := node.Annotations[v1.LabellerSkipNodeAnnotation]
	return exists
}

func (n *NodeLabeller) patchNode(originalNode, node *k8sv1.Node) error {
	p := make([]patch.PatchOperation, 0)
	if !equality.Semantic.DeepEqual(originalNode.Labels, node.Labels) {
		p = append(p, patch.PatchOperation{
			Op:    "test",
			Path:  "/metadata/labels",
			Value: originalNode.Labels,
		}, patch.PatchOperation{
			Op:    "replace",
			Path:  "/metadata/labels",
			Value: node.Labels,
		})
	}

	// patch node only if there is change in labels
	if len(p) > 0 {
		payloadBytes, err := json.Marshal(p)
		if err != nil {
			return err
		}
		_, err = n.nodeClient.Patch(context.Background(), node.Name, types.JSONPatchType, payloadBytes, metav1.PatchOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *NodeLabeller) loadHypervFeatures() {
	n.hypervFeatures.items = getCapLabels()
}

// prepareLabels converts cpu models, features, hyperv features to map[string]string format
// e.g. "cpu-feature.node.kubevirt.io/Penryn": "true"
func (n *NodeLabeller) prepareLabels(node *k8sv1.Node, cpuModels []string, cpuFeatures cpuFeatures, hostCpuModel hostCPUModel, obsoleteCPUsx86 map[string]bool) map[string]string {
	newLabels := make(map[string]string)
	for key := range cpuFeatures {
		newLabels[v1.CPUFeatureLabel+key] = "true"
	}

	for _, value := range cpuModels {
		if !n.shouldAddCPUModelLabel(value, &hostCpuModel, newLabels) {
			continue
		}

		newLabels[v1.CPUModelLabel+value] = "true"
		newLabels[v1.SupportedHostModelMigrationCPU+value] = "true"
	}

	if _, hostModelObsolete := obsoleteCPUsx86[hostCpuModel.Name]; !hostModelObsolete {
		newLabels[v1.SupportedHostModelMigrationCPU+hostCpuModel.Name] = "true"
	}

	for _, key := range n.hypervFeatures.items {
		newLabels[v1.HypervLabel+key] = "true"
	}

	if c, err := n.capabilities.GetTSCCounter(); err == nil && c != nil {
		newLabels[v1.CPUTimerLabel+"tsc-frequency"] = fmt.Sprintf("%d", c.Frequency)
		newLabels[v1.CPUTimerLabel+"tsc-scalable"] = fmt.Sprintf("%v", c.Scaling)
	} else if err != nil {
		n.logger.Reason(err).Error("failed to get tsc cpu frequency, will continue without the tsc frequency label")
	}

	for feature := range hostCpuModel.requiredFeatures {
		newLabels[v1.HostModelRequiredFeaturesLabel+feature] = "true"
	}
	if _, obsolete := obsoleteCPUsx86[hostCpuModel.Name]; obsolete {
		newLabels[v1.NodeHostModelIsObsoleteLabel] = "true"
		err := n.alertIfHostModelIsObsolete(node, hostCpuModel.Name, obsoleteCPUsx86)
		if err != nil {
			n.logger.Reason(err).Error(err.Error())
		}
	}

	newLabels[v1.CPUModelVendorLabel+n.cpuModelVendor] = "true"
	newLabels[v1.HostModelCPULabel+hostCpuModel.Name] = "true"

	capable, err := isNodeRealtimeCapable()
	if err != nil {
		n.logger.Reason(err).Error("failed to identify if a node is capable of running realtime workloads")
	}
	if capable {
		newLabels[v1.RealtimeLabel] = ""
	}

	if n.SEV.Supported == "yes" {
		newLabels[v1.SEVLabel] = ""
	}

	if n.SEV.SupportedES == "yes" {
		newLabels[v1.SEVESLabel] = ""
	}

	return newLabels
}

// addNodeLabels adds labels to node.
func (n *NodeLabeller) addLabellerLabels(node *k8sv1.Node, labels map[string]string) {
	for key, value := range labels {
		node.Labels[key] = value
	}
}

func (n *NodeLabeller) HostCapabilities() *api.Capabilities {
	return n.capabilities
}

// removeLabellerLabels removes labels from node
func (n *NodeLabeller) removeLabellerLabels(node *k8sv1.Node) {
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

func (n *NodeLabeller) alertIfHostModelIsObsolete(originalNode *k8sv1.Node, hostModel string, ObsoleteCPUModels map[string]bool) error {
	warningMsg := fmt.Sprintf("This node has %v host-model cpu that is included in ObsoleteCPUModels: %v", hostModel, ObsoleteCPUModels)
	n.recorder.Eventf(originalNode, k8sv1.EventTypeWarning, "HostModelIsObsolete", warningMsg)
	return nil
}

func (n *NodeLabeller) shouldAddCPUModelLabel(
	cpuModelName string,
	hostCpuModel *hostCPUModel,
	featureLabels map[string]string,
) bool {
	if cpuModelName == hostCpuModel.Name {
		return true
	}
	// The logic below is necessary to handle the scenarios when libvirt's definition of a
	// particular CPU model differs from hypervisor's definition.
	// E.g. currently Opteron_G2 requires svm by libvirt:
	//     /usr/share/libvirt/cpu_map/x86_Opteron_G2.xml
	// But libvirt marks it as Usable:yes even without svm because it is usable by qemu:
	//     /var/lib/kubevirt-node-labeller/virsh_domcapabilities.xml
	// For more information refer to https://wiki.qemu.org/Features/CPUModels, "Getting
	// information about CPU models" section.
	// Another similar issue:
	//     https://gitlab.com/libvirt/libvirt/-/issues/304
	requiredFeatures, ok := n.cpuInfo.usableModels[cpuModelName]
	if !ok {
		n.logger.Warningf("The list of required features for CPU model %s is not defined", cpuModelName)
		return false
	}
	missingFeatures := make([]string, 0)
	for f := range requiredFeatures {
		if _, isFeatureSupported := featureLabels[v1.CPUFeatureLabel+f]; !isFeatureSupported {
			missingFeatures = append(missingFeatures, f)
		}
	}
	return len(missingFeatures) == 0
}
