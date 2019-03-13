package watch

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

const (
	// NodeUnresponsiveReason is in various places as reason to indicate that
	// an action was taken because virt-handler became unresponsive.
	NodeUnresponsiveReason = "NodeUnresponsive"
)

// NodeController is the main NodeController struct.
type NodeController struct {
	clientset        kubecli.KubevirtClient
	Queue            workqueue.RateLimitingInterface
	nodeInformer     cache.SharedIndexInformer
	vmiInformer      cache.SharedIndexInformer
	recorder         record.EventRecorder
	heartBeatTimeout time.Duration
	recheckInterval  time.Duration
}

// NewNodeController creates a new instance of the NodeController struct.
func NewNodeController(clientset kubecli.KubevirtClient, nodeInformer cache.SharedIndexInformer, vmiInformer cache.SharedIndexInformer, recorder record.EventRecorder) *NodeController {
	c := &NodeController{
		clientset:        clientset,
		Queue:            workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		nodeInformer:     nodeInformer,
		vmiInformer:      vmiInformer,
		recorder:         recorder,
		heartBeatTimeout: 5 * time.Minute,
		recheckInterval:  1 * time.Minute,
	}

	c.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addNode,
		DeleteFunc: c.deleteNode,
		UpdateFunc: c.updateNode,
	})

	c.vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
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
	vmi := obj.(*virtv1.VirtualMachineInstance)
	if vmi.Status.NodeName != "" {
		c.Queue.Add(vmi.Status.NodeName)
	}
}

func (c *NodeController) updateVirtualMachine(old, curr interface{}) {
	currVMI := curr.(*virtv1.VirtualMachineInstance)
	if currVMI.Status.NodeName != "" {
		c.Queue.Add(currVMI.Status.NodeName)
	}
}

// Run runs the passed in NodeController.
func (c *NodeController) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting node controller.")

	// Wait for cache sync before we start the node controller
	cache.WaitForCacheSync(stopCh, c.nodeInformer.HasSynced, c.vmiInformer.HasSynced)

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

// Execute runs commands from the controller queue, if there is
// an error it requeues the command. Returns false if the queue
// is empty.
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
			c.recorder.Event(node, v1.EventTypeNormal, NodeUnresponsiveReason, "virt-handler is not responsive, marking node as unresponsive")
			data := []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "false"}}}`, virtv1.NodeSchedulable))
			_, err = c.clientset.CoreV1().Nodes().Patch(nodeName, types.StrategicMergePatchType, data)
			if err != nil {
				logger.Reason(err).Error("Failed to mark node as unschedulable")
				return fmt.Errorf("failed to mark node %s as unschedulable: %v", nodeName, err)
			}
		}
		vmis, err := c.virtualMachinesOnNode(nodeName)
		if err != nil {
			logger.Reason(err).Error("Failed fetch vmis for node")
			return err
		} else if len(vmis) == 0 {
			if nodeExists {
				c.Queue.AddAfter(key, c.recheckInterval)
			}
			return nil
		}
		pods, err := c.alivePodsOnNode(nodeName)
		if err != nil {
			logger.Reason(err).Error("Failed fetch pods for node")
			return err
		}

		vmis = filterStuckVirtualMachinesWithoutPods(vmis, pods)

		errs := []string{}
		// Do sequential updates, we don't want to create update storms in situations where something might already be wrong
		for _, vmi := range vmis {
			c.recorder.Event(vmi, v1.EventTypeNormal, NodeUnresponsiveReason, fmt.Sprintf("virt-handler on node %s is not responsive, marking VMI as failed", vmi.Status.NodeName))
			logger.V(2).Infof("Moving vmi %s in namespace %s on unresponsive node to failed state", vmi.Name, vmi.Namespace)
			phasePatch := fmt.Sprintf(`{ "op": "replace", "path": "/status/phase", "value": "%s" }`, virtv1.Failed)
			operation := "add"
			if vmi.Status.Reason != "" {
				operation = "replace"
			}
			reasonPatch := fmt.Sprintf(`{ "op": "%s", "path": "/status/reason", "value": "%s" }`, operation, NodeUnresponsiveReason)
			_, err := c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(vmi.Name, types.JSONPatchType, []byte(fmt.Sprintf("[%s, %s]", phasePatch, reasonPatch)))
			if err != nil {
				errs = append(errs, fmt.Sprintf("failed to move vmi %s in namespace %s to final state: %v", vmi.Name, vmi.Namespace, err))
				logger.Reason(err).Errorf("Failed to move vmi %s in namespace %s to final state", vmi.Name, vmi.Namespace)
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

func (c *NodeController) virtualMachinesOnNode(nodeName string) ([]*virtv1.VirtualMachineInstance, error) {
	labelSelector, err := labels.Parse(fmt.Sprintf("%s in (%s)", virtv1.NodeNameLabel, nodeName))
	if err != nil {
		return nil, err
	}
	list, err := c.clientset.VirtualMachineInstance(v1.NamespaceAll).List(&metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})

	if err != nil {
		return nil, err
	}

	vmis := []*virtv1.VirtualMachineInstance{}

	for i := range list.Items {
		vmis = append(vmis, &list.Items[i])
	}
	return vmis, nil
}

func (c *NodeController) alivePodsOnNode(nodeName string) ([]*v1.Pod, error) {
	handlerNodeSelector := fields.ParseSelectorOrDie("spec.nodeName=" + nodeName)
	list, err := c.clientset.CoreV1().Pods(v1.NamespaceAll).List(metav1.ListOptions{
		FieldSelector: handlerNodeSelector.String(),
	})
	if err != nil {
		return nil, err
	}

	pods := []*v1.Pod{}

	for i := range list.Items {
		pod := &list.Items[i]
		if controllerRef := controller.GetControllerOf(pod); !isControlledByVMI(controllerRef) {
			continue
		}
		phase := pod.Status.Phase
		if phase != v1.PodFailed && phase != v1.PodSucceeded {
			pods = append(pods, pod)
		}
	}
	return pods, nil
}

func filterStuckVirtualMachinesWithoutPods(vmis []*virtv1.VirtualMachineInstance, pods []*v1.Pod) []*virtv1.VirtualMachineInstance {
	podsPerNamespace := map[string]map[string]*v1.Pod{}

	for _, pod := range pods {
		podsForVMI, ok := podsPerNamespace[pod.Namespace]
		if !ok {
			podsForVMI = map[string]*v1.Pod{}
		}
		if controllerRef := controller.GetControllerOf(pod); isControlledByVMI(controllerRef) {
			podsForVMI[string(controllerRef.UID)] = pod
			podsPerNamespace[pod.Namespace] = podsForVMI
		}
	}

	filtered := []*virtv1.VirtualMachineInstance{}
	for _, vmi := range vmis {
		if vmi.IsScheduled() || vmi.IsRunning() {
			if podsForVMI, exists := podsPerNamespace[vmi.Namespace]; exists {
				if _, exists := podsForVMI[string(vmi.UID)]; exists {
					continue
				}
			}
			filtered = append(filtered, vmi)
		}
	}
	return filtered
}

func isControlledByVMI(controllerRef *metav1.OwnerReference) bool {
	return controllerRef != nil && controllerRef.Kind == virtv1.VirtualMachineInstanceGroupVersionKind.Kind
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
