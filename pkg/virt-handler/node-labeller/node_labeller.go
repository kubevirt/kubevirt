package nodelabeller

import (
	goerror "errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	clientset         kubecli.KubevirtClient
	Queue             workqueue.RateLimitingInterface
	host              string
	namespace         string
}

func NewNodeLabeller(kvmController *device_manager.DeviceController, configMapInformer cache.SharedIndexInformer, clientset kubecli.KubevirtClient, host, namespace string) *NodeLabeller {
	return &NodeLabeller{
		kvmController:     kvmController,
		configMapInformer: configMapInformer,
		clientset:         clientset,
		Queue:             workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),

		host:      host,
		namespace: namespace,
	}
}

//Run runs node-labeller
func (n *NodeLabeller) Run(threadiness int, stop chan struct{}) {
	defer n.Queue.ShutDown()
	logger := log.DefaultLogger()
	logger.Infof("node-labeller is running")

	n.configMapInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			n.Queue.AddRateLimited(obj)
		},
		DeleteFunc: func(obj interface{}) {
			n.Queue.AddRateLimited(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			n.Queue.AddRateLimited(new)
		},
	})

	go n.configMapInformer.Run(stop)
	cache.WaitForCacheSync(stop, n.configMapInformer.HasSynced)

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
		n.Queue.AddRateLimited(key)
	} else {
		n.Queue.Forget(key)
	}
	return true
}

func (n *NodeLabeller) run() error {
	logger := log.DefaultLogger()

	if !n.kvmController.NodeHasDevice(device_manager.KVMPath) {
		return goerror.New(fmt.Sprintf("Node-labeller cannot work without KVM device."))
	}

	cpuFeatures := make(map[string]bool)
	cpuModels := make([]string, 0)
	//parse all informations
	cpuModels, cpuFeatures, err := n.getCPUInfo()
	if err != nil {
		logger.Infof("node-labeller cannot get new labels %s\n", err.Error())
		return err
	}
	//get node on which virt-handler is running
	node, err := n.clientset.CoreV1().Nodes().Get(n.host, metav1.GetOptions{})
	if err != nil {
		logger.Infof("node-labeller cannot get node %s", n.host)
		return err
	}

	//prepare new labels
	newLabels := n.prepareLabels(cpuModels, cpuFeatures)
	//remove old labeller labels
	n.removeCPULabels(node, n.getNodeLabellerLabels(node))
	//add new labels
	n.addNodeLabels(node, newLabels)
	//patch node with new labels
	data := []byte(fmt.Sprintf(`[{"op": "replace", "path": "/metadata/labels", "value": {%s}}, {"op": "replace", "path": "/metadata/annotations", "value": {%s}}]`, convertMapToText(node.Labels), convertMapToText(node.Annotations)))
	_, err = n.clientset.CoreV1().Nodes().Patch(node.Name, types.JSONPatchType, data)
	if err != nil {
		logger.Infof("error during node %s update. %s\n", node.Name, err)
		return err
	}
	return nil
}

func convertMapToText(m map[string]string) string {
	text := ""
	lenMap := len(m)
	i := 0
	for key, value := range m {
		if strings.Contains(key, string("\"")) {
			key = strings.ReplaceAll(key, string("\""), string("\\\""))
		}
		if strings.Contains(value, string("\"")) {
			value = strings.ReplaceAll(value, string("\""), string("\\\""))
		}
		text += fmt.Sprintf(`"%s":"%s"`, key, value)

		if i != (lenMap - 1) {
			text += ", "
		}
		i++
	}
	return text
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

// UpdateNode updates node
func (n *NodeLabeller) updateNode(node *v1.Node) error {
	_, err := n.clientset.CoreV1().Nodes().Update(node)
	if err != nil {
		return err
	}

	return nil
}
