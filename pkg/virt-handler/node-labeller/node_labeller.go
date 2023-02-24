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
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"

	kubevirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	utiltype "kubevirt.io/kubevirt/pkg/util/types"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/api"
	"kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
)

//NodeLabeller struct holds informations needed to run node-labeller
type NodeLabeller struct {
	clientset               kubecli.KubevirtClient
	host                    string
	namespace               string
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

func NewNodeLabeller(clusterConfig *virtconfig.ClusterConfig, clientset kubecli.KubevirtClient, host, namespace string) (*NodeLabeller, error) {
	return newNodeLabeller(clusterConfig, clientset, host, namespace, nodeLabellerVolumePath)

}
func newNodeLabeller(clusterConfig *virtconfig.ClusterConfig, clientset kubecli.KubevirtClient, host, namespace string, volumePath string) (*NodeLabeller, error) {
	n := &NodeLabeller{
		clientset:               clientset,
		host:                    host,
		namespace:               namespace,
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

//Run runs node-labeller
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

	err = n.loadHostSupportedFeatures()
	if err != nil {
		n.logger.Errorf("node-labeller could not load supported features: " + err.Error())
		return err
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
	hostCPUModel := n.getHostCpuModel()

	originalNode, err := n.clientset.CoreV1().Nodes().Get(context.Background(), n.host, metav1.GetOptions{})
	if err != nil {
		return err
	}

	node := originalNode.DeepCopy()

	if skipNode(node) {
		return nil
	}

	//prepare new labels
	newLabels := n.prepareLabels(cpuModels, cpuFeatures, hostCPUModel, obsoleteCPUsx86)
	//remove old labeller labels
	n.removeLabellerLabels(node)
	//add new labels
	n.addLabellerLabels(node, newLabels)

	err = n.patchNode(originalNode, node)

	return err
}

func skipNode(node *v1.Node) bool {
	_, exists := node.Annotations[kubevirtv1.LabellerSkipNodeAnnotation]
	return exists
}

func (n *NodeLabeller) patchNode(originalNode, node *v1.Node) error {
	p := make([]utiltype.PatchOperation, 0)
	if !equality.Semantic.DeepEqual(originalNode.Labels, node.Labels) {
		p = append(p, utiltype.PatchOperation{
			Op:    "test",
			Path:  "/metadata/labels",
			Value: originalNode.Labels,
		}, utiltype.PatchOperation{
			Op:    "replace",
			Path:  "/metadata/labels",
			Value: node.Labels,
		})
	}

	if !equality.Semantic.DeepEqual(originalNode.Annotations, node.Annotations) {
		p = append(p, utiltype.PatchOperation{
			Op:    "test",
			Path:  "/metadata/annotations",
			Value: originalNode.Annotations,
		}, utiltype.PatchOperation{
			Op:    "replace",
			Path:  "/metadata/annotations",
			Value: node.Annotations,
		},
		)
	}

	//patch node only if there is change in labels or annotations
	if len(p) > 0 {
		payloadBytes, err := json.Marshal(p)
		if err != nil {
			return err
		}
		_, err = n.clientset.CoreV1().Nodes().Patch(context.Background(), node.Name, types.JSONPatchType, payloadBytes, metav1.PatchOptions{})
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
func (n *NodeLabeller) prepareLabels(cpuModels []string, cpuFeatures cpuFeatures, hostCpuModel hostCPUModel, obsoleteCPUsx86 map[string]bool) map[string]string {
	newLabels := make(map[string]string)
	for key := range cpuFeatures {
		newLabels[kubevirtv1.CPUFeatureLabel+key] = "true"
	}
	_, svmIsSupported := newLabels[kubevirtv1.CPUFeatureLabel+"svm"]

	for _, value := range cpuModels {

		//This workaround is necessary because currently Opteron_G2 require svm by libvirt (see /usr/share/libvirt/cpu_map/x86_Opteron_G2.xml )
		//But libvirt still marks it as Usable:yes even without `svm` because it is usable by qemu (/var/lib/kubevirt-node-labeller/virsh_domcapabilities.xml)
		//For more information : https://wiki.qemu.org/Features/CPUModels in "Getting information about CPU models" section
		//TODO: Delete this workaround once libvirt resolve the disagreement with qemu
		if value == "Opteron_G2" && svmIsSupported == false {
			continue
		}
		if !n.shouldAddCPUModelLabel(value, &hostCpuModel, newLabels) {
			continue
		}
		newLabels[kubevirtv1.CPUModelLabel+value] = "true"
		newLabels[kubevirtv1.SupportedHostModelMigrationCPU+value] = "true"
	}

	if _, hostModelObsolete := obsoleteCPUsx86[hostCpuModel.name]; !hostModelObsolete {
		newLabels[kubevirtv1.SupportedHostModelMigrationCPU+hostCpuModel.name] = "true"
	}

	for _, key := range n.hypervFeatures.items {
		newLabels[kubevirtv1.HypervLabel+key] = "true"
	}

	if c, err := n.capabilities.GetTSCCounter(); err == nil && c != nil {
		newLabels[kubevirtv1.CPUTimerLabel+"tsc-frequency"] = fmt.Sprintf("%d", c.Frequency)
		newLabels[kubevirtv1.CPUTimerLabel+"tsc-scalable"] = fmt.Sprintf("%v", c.Scaling)
	} else if err != nil {
		n.logger.Reason(err).Error("failed to get tsc cpu frequency, will continue without the tsc frequency label")
	}

	for feature, _ := range hostCpuModel.requiredFeatures {
		newLabels[kubevirtv1.HostModelRequiredFeaturesLabel+feature] = "true"
	}

	newLabels[kubevirtv1.CPUModelVendorLabel+n.cpuModelVendor] = "true"
	newLabels[kubevirtv1.HostModelCPULabel+hostCpuModel.name] = "true"

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

	return newLabels
}

// addNodeLabels adds labels and special annotation to node.
// annotations are needed because we need to know which labels were set by kubevirt.
func (n *NodeLabeller) addLabellerLabels(node *v1.Node, labels map[string]string) {
	for key, value := range labels {
		node.Labels[key] = value
	}
}

func (n *NodeLabeller) HostCapabilities() *api.Capabilities {
	return n.capabilities
}

// removeLabellerLabels removes labels from node
func (n *NodeLabeller) removeLabellerLabels(node *v1.Node) {
	for label := range node.Labels {
		if strings.Contains(label, util.DeprecatedLabelNamespace+util.DeprecatedcpuModelPrefix) ||
			strings.Contains(label, util.DeprecatedLabelNamespace+util.DeprecatedcpuFeaturePrefix) ||
			strings.Contains(label, util.DeprecatedLabelNamespace+util.DeprecatedHyperPrefix) ||
			strings.Contains(label, kubevirtv1.CPUFeatureLabel) ||
			strings.Contains(label, kubevirtv1.CPUModelLabel) ||
			strings.Contains(label, kubevirtv1.SupportedHostModelMigrationCPU) ||
			strings.Contains(label, kubevirtv1.HostModelCPULabel) ||
			strings.Contains(label, kubevirtv1.HostModelRequiredFeaturesLabel) ||
			strings.Contains(label, kubevirtv1.CPUTimerLabel) ||
			strings.Contains(label, kubevirtv1.HypervLabel) ||
			strings.Contains(label, kubevirtv1.RealtimeLabel) ||
			strings.Contains(label, kubevirtv1.SEVLabel) {
			delete(node.Labels, label)
		}
	}

	for annotation := range node.Annotations {
		if strings.Contains(annotation, util.DeprecatedLabellerNamespaceAnnotation) {
			delete(node.Annotations, annotation)
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

func (n *NodeLabeller) shouldAddCPUModelLabel(
	cpuModelName string,
	hostCpuModel *hostCPUModel,
	featureLabels map[string]string,
) bool {
	if cpuModelName == hostCpuModel.name {
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
	for f, _ := range requiredFeatures {
		if _, isFeatureSupported := featureLabels[kubevirtv1.CPUFeatureLabel+f]; !isFeatureSupported {
			missingFeatures = append(missingFeatures, f)
		}
	}
	return len(missingFeatures) == 0
}
