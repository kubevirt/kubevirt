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
	"reflect"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"

	kubevirtv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	device_manager "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"
	util "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
)

const (
	kvmPath = "/dev/kvm"
)

//NodeLabeller struct holds informations needed to run node-labeller
type NodeLabeller struct {
	kvmController     *device_manager.DeviceController
	clientset         kubecli.KubevirtClient
	host              string
	namespace         string
	logger            *log.FilteredLogger
	clusterConfig     *virtconfig.ClusterConfig
	hypervFeatures    supportedFeatures
	hostCapabilities  supportedFeatures
	queue             workqueue.RateLimitingInterface
	supportedFeatures []string
	cpuInfo           cpuInfo
}

func NewNodeLabeller(kvmController *device_manager.DeviceController, clusterConfig *virtconfig.ClusterConfig, clientset kubecli.KubevirtClient, host, namespace string) (*NodeLabeller, error) {
	n := &NodeLabeller{
		kvmController: kvmController,
		clientset:     clientset,
		host:          host,
		namespace:     namespace,
		logger:        log.DefaultLogger(),
		clusterConfig: clusterConfig,
		queue:         workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
	err := n.loadCPUInfo()
	if err != nil {
		n.logger.Errorf("node-labeller could not load cpu info: " + err.Error())
		return nil, err
	}

	err = n.loadHostSupportedFeatures()
	if err != nil {
		n.logger.Errorf("node-labeller could not load supported features: " + err.Error())
		return nil, err
	}

	err = n.loadHostCapabilities()
	if err != nil {
		n.logger.Errorf("node-labeller could not load host capabilities: " + err.Error())
		return nil, err
	}

	n.loadHypervFeatures()

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

	if !n.kvmController.NodeHasDevice(kvmPath) {
		n.logger.Errorf("node-labeller cannot work without KVM device.")
		return true
	}

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
	var (
		cpuFeatures map[string]bool
		cpuModels   []string
	)

	//parse all informations
	cpuModels, cpuFeatures = n.getCPUInfo()

	originalNode, err := n.clientset.CoreV1().Nodes().Get(context.Background(), n.host, metav1.GetOptions{})
	if err != nil {
		return err
	}

	node := originalNode.DeepCopy()

	//prepare new labels
	newLabels := n.prepareLabels(cpuModels, cpuFeatures)
	//remove old labeller labels
	n.removeLabellerLabels(node)
	//add new labels
	n.addLabellerLabels(node, newLabels)

	err = n.patchNode(originalNode, node)

	return err
}

func (n *NodeLabeller) patchNode(originalNode, node *v1.Node) error {
	type payload struct {
		Op    string            `json:"op"`
		Path  string            `json:"path"`
		Value map[string]string `json:"value"`
	}

	p := make([]payload, 0)
	if !reflect.DeepEqual(originalNode.Labels, node.Labels) {
		p = append(p, payload{
			Op:    "test",
			Path:  "/metadata/labels",
			Value: originalNode.Labels,
		}, payload{
			Op:    "replace",
			Path:  "/metadata/labels",
			Value: node.Labels,
		})
	}

	if !reflect.DeepEqual(originalNode.Annotations, node.Annotations) {
		p = append(p, payload{
			Op:    "test",
			Path:  "/metadata/annotations",
			Value: originalNode.Annotations,
		}, payload{
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
func (n *NodeLabeller) prepareLabels(cpuModels []string, cpuFeatures cpuFeatures) map[string]string {
	newLabels := make(map[string]string)
	for key := range cpuFeatures {
		newLabels[kubevirtv1.CPUFeatureLabel+key] = "true"
	}

	for _, value := range cpuModels {
		newLabels[kubevirtv1.CPUModelLabel+value] = "true"
	}

	for _, key := range n.hypervFeatures.items {
		newLabels[kubevirtv1.HypervLabel+key] = "true"
	}
	return newLabels
}

// addNodeLabels adds labels and special annotation to node.
// annotations are needed because we need to know which labels were set by kubevirt.
func (n *NodeLabeller) addLabellerLabels(node *v1.Node, labels map[string]string) {
	for label := range labels {
		node.Labels[label] = "true"
	}
}

// removeLabellerLabels removes labels from node
func (n *NodeLabeller) removeLabellerLabels(node *v1.Node) {
	for label := range node.Labels {
		if strings.Contains(label, util.DeprecatedLabelNamespace+util.DeprecatedcpuModelPrefix) ||
			strings.Contains(label, util.DeprecatedLabelNamespace+util.DeprecatedcpuFeaturePrefix) ||
			strings.Contains(label, util.DeprecatedLabelNamespace+util.DeprecatedHyperPrefix) ||
			strings.Contains(label, kubevirtv1.CPUFeatureLabel) ||
			strings.Contains(label, kubevirtv1.CPUModelLabel) ||
			strings.Contains(label, kubevirtv1.HypervLabel) {
			delete(node.Labels, label)
		}
	}

	for annotation := range node.Annotations {
		if strings.Contains(annotation, util.DeprecatedLabellerNamespaceAnnotation) {
			delete(node.Annotations, annotation)
		}
	}
}
