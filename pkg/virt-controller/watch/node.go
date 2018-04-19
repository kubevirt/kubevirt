package watch

import (
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/api/core/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	v12 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

type NodeController struct {
	clientset        kubecli.KubevirtClient
	Queue            workqueue.RateLimitingInterface
	nodeInformer     cache.SharedIndexInformer
	recorder         record.EventRecorder
	heartBeatTimeout time.Duration
	recheckInterval  time.Duration
}

func NewNodeController(clientset kubecli.KubevirtClient, nodeInformer cache.SharedIndexInformer, recorder record.EventRecorder) *NodeController {
	c := &NodeController{
		clientset:        clientset,
		Queue:            workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		nodeInformer:     nodeInformer,
		recorder:         recorder,
		heartBeatTimeout: 5 * time.Minute,
		recheckInterval:  1 * time.Minute,
	}

	c.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addNode,
		DeleteFunc: c.deleteNode,
		UpdateFunc: c.updateNode,
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

func (c *NodeController) Run(threadiness int, stopCh chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting node controller.")

	// Wait for cache sync before we start the pod controller
	cache.WaitForCacheSync(stopCh, c.nodeInformer.HasSynced)

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

	if err != nil {
		return err
	}

	nodeName := key
	var node *v1.Node
	if nodeExists {
		node = obj.(*v1.Node)
	}

	if unresponsive, err := isNodeUnresponsive(node, c.heartBeatTimeout); err != nil {
		return fmt.Errorf("failed to determine if node %s is responsive: %v", nodeName, err)
	} else if unresponsive {
		if nodeExists && node.Labels[v12.NodeSchedulable] == "true" {
			data := []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "false"}}}`, v12.NodeSchedulable))
			_, err = c.clientset.CoreV1().Nodes().Patch(nodeName, types.StrategicMergePatchType, data)
			if err != nil {
				return fmt.Errorf("failed to mark node %s as unschedulable: %v", nodeName, err)
			}
		}
		vms, err := c.virtalMachinesOnNode(nodeName)
		if err != nil || len(vms) == 0 {
			return err
		}
		pods, err := c.podsOnNode(nodeName)
		if err != nil {
			return err
		}
		vms = filterStuckVirtualMachinesWithoutPods(vms, pods)

		for _, vm := range vms {
			// FIXME don't stop on first error
			_, err := c.clientset.VM(vm.Namespace).Patch(vm.Name, types.JSONPatchType, []byte(fmt.Sprintf("[{ \"op\": \"replace\", \"path\": \"/status/phase\", \"value\": \"%s\" }]", v12.Failed)))
			if err != nil {
				return fmt.Errorf("failed to move vm %s to final state: %v", vm.Name, err)
			}
		}
	}
	c.Queue.AddAfter(key, c.recheckInterval)
	return nil
}

func (c *NodeController) virtalMachinesOnNode(nodeName string) ([]*v12.VirtualMachine, error) {
	labelSelector, err := labels.Parse(fmt.Sprintf("%s in (%s)", v12.NodeNameLabel, nodeName))
	if err != nil {
		return nil, err
	}
	list, err := c.clientset.VM(v1.NamespaceAll).List(v13.ListOptions{
		LabelSelector: labelSelector.String(),
	})

	if err != nil {
		return nil, err
	}

	vms := []*v12.VirtualMachine{}

	for i, _ := range list.Items {
		vms = append(vms, &list.Items[i])
	}
	return vms, nil
}

func (c *NodeController) podsOnNode(nodeName string) ([]*v1.Pod, error) {
	labelSelector, err := labels.Parse(v12.DomainLabel)
	handlerNodeSelector := fields.ParseSelectorOrDie("spec.nodeName=" + nodeName)
	if err != nil {
		return nil, err
	}
	list, err := c.clientset.CoreV1().Pods(v1.NamespaceAll).List(v13.ListOptions{
		LabelSelector: labelSelector.String(),
		FieldSelector: handlerNodeSelector.String(),
	})

	pods := []*v1.Pod{}

	for i, _ := range list.Items {
		pods = append(pods, &list.Items[i])
	}
	return pods, nil
}

func filterStuckVirtualMachinesWithoutPods(vms []*v12.VirtualMachine, pods []*v1.Pod) []*v12.VirtualMachine {
	podsPerNamespace := map[string]map[string]*v1.Pod{}

	for _, pod := range pods {
		podsForVM, ok := podsPerNamespace[pod.Namespace]
		if !ok {
			podsForVM = map[string]*v1.Pod{}
		}
		name := pod.Labels[v12.DomainLabel]
		if len(name) == 0 {
			continue
		}
		podsForVM[name] = pod
		podsPerNamespace[pod.Namespace] = podsForVM
	}

	filtered := []*v12.VirtualMachine{}
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
	if lastHeartBeat, exists := node.Annotations[v12.VirtHandlerHeartbeat]; exists {

		timestamp := v13.Time{}
		if err := json.Unmarshal([]byte(`"`+lastHeartBeat+`"`), &timestamp); err != nil {
			return false, err
		}
		if timestamp.Time.Before(v13.Now().Add(-timeout)) {
			return true, nil
		}
	}
	return false, nil
}
