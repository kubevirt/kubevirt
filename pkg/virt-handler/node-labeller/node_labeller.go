package nodelabeller

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	device_manager "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"
)

const (
	labelNamespace    = "feature.node.kubernetes.io"
	labellerNamespace = "node-labeller"
)

var nodeLabellerVolumePath = "/var/lib/kubevirt-node-labeller"

//NodeLabeller struct holds informations needed to run node-labeller
type NodeLabeller struct {
	kvmController     *device_manager.DeviceController
	configMapInformer cache.SharedIndexInformer
	nodeInformer      cache.SharedIndexInformer
	clientset         kubecli.KubevirtClient
	Queue             workqueue.RateLimitingInterface
	host              string
	namespace         string
	logger            *log.FilteredLogger
}

func NewNodeLabeller(kvmController *device_manager.DeviceController, nodeInformer, configMapInformer cache.SharedIndexInformer, clientset kubecli.KubevirtClient, host, namespace string) *NodeLabeller {
	return &NodeLabeller{
		kvmController:     kvmController,
		configMapInformer: configMapInformer,
		nodeInformer:      nodeInformer,
		clientset:         clientset,
		Queue:             workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),

		host:      host,
		namespace: namespace,
		logger:    log.DefaultLogger(),
	}
}

//Run runs node-labeller
func (n *NodeLabeller) Run(threadiness int, stop chan struct{}) {
	defer n.Queue.ShutDown()
	n.logger.Infof("node-labeller is running")

	if !n.kvmController.NodeHasDevice(device_manager.KVMPath) {
		n.logger.Infof("Node-labeller cannot work without KVM device.")
		return
	}

	n.configMapInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(_ interface{}) {
			n.Queue.Add(n.host)
		},
		DeleteFunc: func(_ interface{}) {
			n.Queue.Add(n.host)
		},
		UpdateFunc: func(_, _ interface{}) {
			n.Queue.Add(n.host)
		},
	})

	go n.configMapInformer.Run(stop)

	n.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, _ interface{}) {
			n.Queue.Add(n.host)
		},
	})

	go n.nodeInformer.Run(stop)
	cache.WaitForCacheSync(stop, n.nodeInformer.HasSynced, n.configMapInformer.HasSynced)

	for i := 0; i < threadiness; i++ {
		go wait.Until(n.runWorker, time.Second, stop)
	}
	<-stop
}
func (n *NodeLabeller) runWorker() {
	for n.Execute() {
	}
}

func (n *NodeLabeller) Execute() bool {
	key, quit := n.Queue.Get()
	if quit {
		return false
	}
	defer n.Queue.Done(key)

	err := n.run()

	if err != nil {
		n.logger.Errorf("node-labeller sync error encountered: %v", err)
		n.Queue.AddRateLimited(key)
	} else {
		n.Queue.Forget(key)
	}
	return true
}

func (n *NodeLabeller) run() error {
	cpuFeatures := make(map[string]bool)
	cpuModels := make([]string, 0)

	//parse all informations
	cpuModels, cpuFeatures, err := n.getCPUInfo()
	if err != nil {
		n.logger.Infof("node-labeller cannot get new labels %s\n", err.Error())
		return err
	}

	//load node
	nodeObj, exists, err := n.nodeInformer.GetStore().GetByKey(n.host)
	if err != nil || !exists {
		n.logger.Infof("node-labeller cannot get node %s", n.host)
		return err
	}
	var (
		ok           bool
		originalNode *v1.Node
	)
	if originalNode, ok = nodeObj.(*v1.Node); !ok {
		n.logger.Infof("node-labeller cannot convert node " + n.host)
		return fmt.Errorf("Could not convert node " + n.host)
	}

	node := originalNode.DeepCopy()

	//prepare new labels
	newLabels := n.prepareLabels(cpuModels, cpuFeatures)
	//remove old labeller labels
	n.removeCPULabels(node, n.getNodeLabellerLabels(node))
	//add new labels
	n.addNodeLabels(node, newLabels)
	//patch node only if there is change in labels
	err = n.patchNode(originalNode, node)
	return err
}

func (n *NodeLabeller) patchNode(originalNode, node *v1.Node) error {
	originalLabelsBytes, err := json.Marshal(originalNode.Labels)
	if err != nil {
		return err
	}

	originalAnnotationsBytes, err := json.Marshal(originalNode.Annotations)
	if err != nil {
		return err
	}

	labelsBytes, err := json.Marshal(node.Labels)
	if err != nil {
		return err
	}

	annotationsBytes, err := json.Marshal(node.Annotations)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(node.Labels, originalNode.Labels) {
		patchTestLabels := fmt.Sprintf(`{ "op": "test", "path": "/metadata/labels", "value": %s}`, string(originalLabelsBytes))
		patchTestAnnotations := fmt.Sprintf(`{ "op": "test", "path": "/metadata/annotations", "value": %s}`, string(originalAnnotationsBytes))
		patchLabels := fmt.Sprintf(`{ "op": "replace", "path": "/metadata/labels", "value": %s}`, string(labelsBytes))
		patchAnnotations := fmt.Sprintf(`{ "op": "replace", "path": "/metadata/annotations", "value": %s}`, string(annotationsBytes))
		data := []byte(fmt.Sprintf("[ %s, %s, %s, %s ]", patchTestLabels, patchLabels, patchTestAnnotations, patchAnnotations))
		_, err = n.clientset.CoreV1().Nodes().Patch(node.Name, types.JSONPatchType, data)
		if err != nil {
			return err
		}
	}

	return nil
}

// prepareLabels converts cpu models + features to map[string]string format
// e.g. "/cpu-model-Penryn": "true"
func (n *NodeLabeller) prepareLabels(cpuModels []string, cpuFeatures map[string]bool) map[string]string {
	newLabels := make(map[string]string)
	for key, value := range cpuFeatures {
		newLabels["/cpu-feature-"+key] = strconv.FormatBool(value)
	}
	for _, value := range cpuModels {
		newLabels["/cpu-model-"+value] = "true"
	}
	return newLabels
}

// addNodeLabels adds labels and special annotation to node.
// annotations are needed because we need to know which labels were set by kubevirt.
func (n *NodeLabeller) addNodeLabels(node *v1.Node, labels map[string]string) {
	for name := range labels {
		node.Labels[labelNamespace+name] = "true"
		node.Annotations[labellerNamespace+"-"+labelNamespace+name] = "true"
	}
}

// getNodeLabellerLabels gets all labels which were created by kubevirt-node-labeller
func (n *NodeLabeller) getNodeLabellerLabels(node *v1.Node) map[string]bool {
	labellerLabels := make(map[string]bool)
	for key := range node.Annotations {
		if strings.Contains(key, labellerNamespace) {
			delete(node.Annotations, key)
			labellerLabels[key] = true
		}
	}
	return labellerLabels
}

// removeCPULabels removes labels from node
func (n *NodeLabeller) removeCPULabels(node *v1.Node, oldLabels map[string]bool) {
	for label := range node.Labels {
		if ok := oldLabels[label]; ok || strings.Contains(label, labelNamespace+"/cpu-model-") || strings.Contains(label, labelNamespace+"/cpu-feature-") {
			delete(node.Labels, label)
		}
	}
}
