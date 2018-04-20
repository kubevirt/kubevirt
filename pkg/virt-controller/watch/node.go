package watch

import (
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"strings"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

type NodeController struct {
	clientset        kubecli.KubevirtClient
	Queue            workqueue.RateLimitingInterface
	nodeInformer     cache.SharedIndexInformer
	vmInformer       cache.SharedIndexInformer
	recorder         record.EventRecorder
	heartBeatTimeout time.Duration
	recheckInterval  time.Duration
}

func NewNodeController(clientset kubecli.KubevirtClient, nodeInformer cache.SharedIndexInformer, vmInformer cache.SharedIndexInformer, recorder record.EventRecorder) *NodeController {
	c := &NodeController{
		clientset:        clientset,
		Queue:            workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		nodeInformer:     nodeInformer,
		vmInformer:       vmInformer,
		recorder:         recorder,
		heartBeatTimeout: 5 * time.Minute,
		recheckInterval:  1 * time.Minute,
	}

	c.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addNode,
		DeleteFunc: c.deleteNode,
		UpdateFunc: c.updateNode,
	})

	c.vmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachine,
		DeleteFunc: func(_ interface{}) {}, // nothing to do
		UpdateFunc: c.updateVirtualMachine,
	})

	return c
}

func (c *NodeController) addNode(obj interface{}) {
	c.enqueueNode(obj)
}

func (c *NodeController) deleteNode(obj interface{}) {
	c.enqueueNode(obj)
}

func (c *NodeController) updateNode(old, curr interface{}) {
	c.enqueueNode(curr)
}

func (c *NodeController) enqueueNode(obj interface{}) {
	logger := log.Log
	node := obj.(*v1.Node)
	key, err := controller.KeyFunc(node)
	if err != nil {
		logger.Object(node).Reason(err).Error("Failed to extract key from node.")
	}
	c.Queue.Add(key)
}

func (c *NodeController) addVirtualMachine(obj interface{}) {
	vm := obj.(*virtv1.VirtualMachine)
	if vm.Status.NodeName != "" {
		c.Queue.Add(vm.Status.NodeName)
	}
}

func (c *NodeController) updateVirtualMachine(old, curr interface{}) {
	currVM := curr.(*virtv1.VirtualMachine)
	if currVM.Status.NodeName != "" {
		c.Queue.Add(currVM.Status.NodeName)
	}
}

func (c *NodeController) Run(threadiness int, stopCh chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting node controller.")

	// Wait for cache sync before we start the node controller
	cache.WaitForCacheSync(stopCh, c.nodeInformer.HasSynced, c.vmInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping node controller.")
}

func (c *NodeController) runWorker() {
	for c.Execute() {
	}
}

func (c *NodeController) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}
	defer c.Queue.Done(key)
	err := c.execute(key.(string))

	if err != nil {
		log.Log.Reason(err).Infof("reenqueuing node %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed node %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func (c *NodeController) execute(key string) error {

	obj, nodeExists, err := c.nodeInformer.GetStore().GetByKey(key)
	logger := log.DefaultLogger()

	if err != nil {
		return err
	}

	nodeName := key
	var node *v1.Node
	if nodeExists {
		node = obj.(*v1.Node)
		logger = logger.Object(node)
	} else {
		logger = logger.Key(key, "Node")
	}

	if unresponsive, err := isNodeUnresponsive(node, c.heartBeatTimeout); err != nil {
		logger.Reason(err).Error("Failed to dermine if node is responsive, will not reenqueue")
		return fmt.Errorf("failed to determine if node %s is responsive: %v", nodeName, err)
	} else if unresponsive {
		if nodeExists && node.Labels[virtv1.NodeSchedulable] == "true" {
			data := []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "false"}}}`, virtv1.NodeSchedulable))
			_, err = c.clientset.CoreV1().Nodes().Patch(nodeName, types.StrategicMergePatchType, data)
			if err != nil {
				logger.Reason(err).Error("Failed to mark node as unschedulable")
				return fmt.Errorf("failed to mark node %s as unschedulable: %v", nodeName, err)
			}
		}
		vms, err := c.virtalMachinesOnNode(nodeName)
		if err != nil {
			logger.Reason(err).Error("Failed fetch vms for node")
			return err
		} else if len(vms) == 0 {
			if nodeExists {
				c.Queue.AddAfter(key, c.recheckInterval)
			}
			return nil
		}
		pods, err := c.podsOnNode(nodeName)
		if err != nil {
			logger.Reason(err).Error("Failed fetch pods for node")
			return err
		}
		vms = filterStuckVirtualMachinesWithoutPods(vms, pods)

		errs := []string{}
		// Do sequential updates, we don't want to create update storms in situations where something might already be wrong
		for _, vm := range vms {
			logger.V(2).Infof("Moving vm %s in namespace %s on unresponsive node to failed state", vm.Name, vm.Namespace)
			_, err := c.clientset.VM(vm.Namespace).Patch(vm.Name, types.JSONPatchType, []byte(fmt.Sprintf("[{ \"op\": \"replace\", \"path\": \"/status/phase\", \"value\": \"%s\" }]", virtv1.Failed)))
			if err != nil {
				errs = append(errs, fmt.Sprintf("failed to move vm %s in namespace %s to final state: %v", vm.Name, vm.Namespace, err))
				logger.Reason(err).Errorf("Failed to move vm %s in namespace %s to final state", vm.Name, vm.Namespace)
			}
		}

		if len(errs) > 0 {
			return fmt.Errorf("%v", strings.Join(errs, "; "))
		}
	}
	if nodeExists {
		c.Queue.AddAfter(key, c.recheckInterval)
	}
	return nil
}

func (c *NodeController) virtalMachinesOnNode(nodeName string) ([]*virtv1.VirtualMachine, error) {
	labelSelector, err := labels.Parse(fmt.Sprintf("%s in (%s)", virtv1.NodeNameLabel, nodeName))
	if err != nil {
		return nil, err
	}
	list, err := c.clientset.VM(v1.NamespaceAll).List(metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})

	if err != nil {
		return nil, err
	}

	vms := []*virtv1.VirtualMachine{}

	for i, _ := range list.Items {
		vms = append(vms, &list.Items[i])
	}
	return vms, nil
}

func (c *NodeController) podsOnNode(nodeName string) ([]*v1.Pod, error) {
	labelSelector, err := labels.Parse(virtv1.DomainLabel)
	handlerNodeSelector := fields.ParseSelectorOrDie("spec.nodeName=" + nodeName)
	if err != nil {
		return nil, err
	}
	list, err := c.clientset.CoreV1().Pods(v1.NamespaceAll).List(metav1.ListOptions{
		LabelSelector: labelSelector.String(),
		FieldSelector: handlerNodeSelector.String(),
	})

	pods := []*v1.Pod{}

	for i, _ := range list.Items {
		pods = append(pods, &list.Items[i])
	}
	return pods, nil
}

func filterStuckVirtualMachinesWithoutPods(vms []*virtv1.VirtualMachine, pods []*v1.Pod) []*virtv1.VirtualMachine {
	podsPerNamespace := map[string]map[string]*v1.Pod{}

	for _, pod := range pods {
		podsForVM, ok := podsPerNamespace[pod.Namespace]
		if !ok {
			podsForVM = map[string]*v1.Pod{}
		}
		name := pod.Labels[virtv1.DomainLabel]
		if len(name) == 0 {
			continue
		}
		podsForVM[name] = pod
		podsPerNamespace[pod.Namespace] = podsForVM
	}

	filtered := []*virtv1.VirtualMachine{}
	for _, vm := range vms {
		if vm.IsScheduled() || vm.IsRunning() {
			if podsForVM, exists := podsPerNamespace[vm.Namespace]; exists {
				if _, exists := podsForVM[vm.Name]; exists {
					continue
				}
			}
			filtered = append(filtered, vm)
		}
	}
	return filtered
}

func isNodeUnresponsive(node *v1.Node, timeout time.Duration) (bool, error) {
	if node == nil {
		return true, nil
	}
	if lastHeartBeat, exists := node.Annotations[virtv1.VirtHandlerHeartbeat]; exists {

		timestamp := metav1.Time{}
		if err := json.Unmarshal([]byte(`"`+lastHeartBeat+`"`), &timestamp); err != nil {
			return false, err
		}
		if timestamp.Time.Before(metav1.Now().Add(-timeout)) {
			return true, nil
		}
	}
	return false, nil
}
