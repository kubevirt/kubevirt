package watch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/util/lookup"
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
func NewNodeController(clientset kubecli.KubevirtClient, nodeInformer cache.SharedIndexInformer, vmiInformer cache.SharedIndexInformer, recorder record.EventRecorder) (*NodeController, error) {
	c := &NodeController{
		clientset:        clientset,
		Queue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-node"),
		nodeInformer:     nodeInformer,
		vmiInformer:      vmiInformer,
		recorder:         recorder,
		heartBeatTimeout: 5 * time.Minute,
		recheckInterval:  1 * time.Minute,
	}

	_, err := c.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addNode,
		DeleteFunc: c.deleteNode,
		UpdateFunc: c.updateNode,
	})
	if err != nil {
		return nil, err
	}

	_, err = c.vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachine,
		DeleteFunc: func(_ interface{}) { /* nothing to do */ },
		UpdateFunc: c.updateVirtualMachine,
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *NodeController) addNode(obj interface{}) {
	c.enqueueNode(obj)
}

func (c *NodeController) deleteNode(obj interface{}) {
	c.enqueueNode(obj)
}

func (c *NodeController) updateNode(_, curr interface{}) {
	c.enqueueNode(curr)
}

func (c *NodeController) enqueueNode(obj interface{}) {
	logger := log.Log
	node := obj.(*v1.Node)
	key, err := controller.KeyFunc(node)
	if err != nil {
		logger.Object(node).Reason(err).Error("Failed to extract key from node.")
		return
	}
	c.Queue.Add(key)
}

func (c *NodeController) addVirtualMachine(obj interface{}) {
	vmi := obj.(*virtv1.VirtualMachineInstance)
	if vmi.Status.NodeName != "" {
		c.Queue.Add(vmi.Status.NodeName)
	}
}

func (c *NodeController) updateVirtualMachine(_, curr interface{}) {
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
	logger := log.DefaultLogger()

	obj, nodeExists, err := c.nodeInformer.GetStore().GetByKey(key)
	if err != nil {
		return err
	}

	var node *v1.Node
	if nodeExists {
		node = obj.(*v1.Node)
		logger = logger.Object(node)
	} else {
		namespace, name, err := cache.SplitMetaNamespaceKey(key)
		if err == nil {
			params := []string{}
			if namespace != "" {
				params = append(params, "namespace", namespace)

			}
			params = append(params, "name", name)
			params = append(params, "kind", "Node")
			logger = logger.With(params)
		}

	}

	unresponsive, err := isNodeUnresponsive(node, c.heartBeatTimeout)
	if err != nil {
		logger.Reason(err).Error("Failed to determine if node is responsive, will not reenqueue")
		return nil
	}

	if unresponsive {
		if nodeIsSchedulable(node) {
			if err := c.markNodeAsUnresponsive(node, logger); err != nil {
				return err
			}
		}

		err = c.checkNodeForOrphanedAndErroredVMIs(key, node, logger)
		if err != nil {
			return err
		}
	}

	c.requeueIfExists(key, node)

	return nil
}

func nodeIsSchedulable(node *v1.Node) bool {
	if node == nil {
		return false
	}

	return node.Labels[virtv1.NodeSchedulable] == "true"
}

func (c *NodeController) checkNodeForOrphanedAndErroredVMIs(nodeName string, node *v1.Node, logger *log.FilteredLogger) error {
	vmis, err := lookup.ActiveVirtualMachinesOnNode(c.clientset, nodeName)
	if err != nil {
		logger.Reason(err).Error("Failed fetching vmis for node")
		return err
	}

	if len(vmis) == 0 {
		c.requeueIfExists(nodeName, node)
		return nil
	}

	err = c.createEventIfNodeHasOrphanedVMIs(node, vmis)
	if err != nil {
		logger.Reason(err).Error("checking virt-handler for node")
		return err
	}

	return c.checkVirtLauncherPodsAndUpdateVMIStatus(nodeName, vmis, logger)
}

func (c *NodeController) checkVirtLauncherPodsAndUpdateVMIStatus(nodeName string, vmis []*virtv1.VirtualMachineInstance, logger *log.FilteredLogger) error {
	pods, err := c.alivePodsOnNode(nodeName)
	if err != nil {
		logger.Reason(err).Error("Failed fetch pods for node")
		return err
	}

	vmis = filterStuckVirtualMachinesWithoutPods(vmis, pods)

	return c.updateVMIWithFailedStatus(vmis, logger)
}

func (c *NodeController) updateVMIWithFailedStatus(vmis []*virtv1.VirtualMachineInstance, logger *log.FilteredLogger) error {
	errs := []string{}
	// Do sequential updates, we don't want to create update storms in situations where something might already be wrong
	for _, vmi := range vmis {
		err := c.createAndApplyFailedVMINodeUnresponsivePatch(vmi, logger)
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to move vmi %s in namespace %s to final state: %v", vmi.Name, vmi.Namespace, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%v", strings.Join(errs, "; "))
	}

	return nil
}

func (c *NodeController) createAndApplyFailedVMINodeUnresponsivePatch(vmi *virtv1.VirtualMachineInstance, logger *log.FilteredLogger) error {
	c.recorder.Event(vmi, v1.EventTypeNormal, NodeUnresponsiveReason, fmt.Sprintf("virt-handler on node %s is not responsive, marking VMI as failed", vmi.Status.NodeName))
	logger.V(2).Infof("Moving vmi %s in namespace %s on unresponsive node to failed state", vmi.Name, vmi.Namespace)

	patchBytes, err := generateFailedVMIPatch(vmi.Status.Reason)
	if err != nil {
		return err
	}
	_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, &metav1.PatchOptions{})
	if err != nil {
		logger.Reason(err).Errorf("Failed to move vmi %s in namespace %s to final state", vmi.Name, vmi.Namespace)
		return err
	}

	return nil
}

func generateFailedVMIPatch(reason string) ([]byte, error) {
	reasonOp := "add"
	if reason != "" {
		reasonOp = "replace"
	}

	return patch.GeneratePatchPayload(
		patch.PatchOperation{
			Op:    patch.PatchReplaceOp,
			Path:  "/status/phase",
			Value: virtv1.Failed,
		},
		patch.PatchOperation{
			Op:    reasonOp,
			Path:  "/status/reason",
			Value: NodeUnresponsiveReason,
		},
	)
}

func (c *NodeController) requeueIfExists(key string, node *v1.Node) {
	if node == nil {
		return
	}

	c.Queue.AddAfter(key, c.recheckInterval)
}

func (c *NodeController) markNodeAsUnresponsive(node *v1.Node, logger *log.FilteredLogger) error {
	c.recorder.Event(node, v1.EventTypeNormal, NodeUnresponsiveReason, "virt-handler is not responsive, marking node as unresponsive")
	logger.V(4).Infof("Marking node %s as unresponsive", node.Name)

	data := []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "false"}}}`, virtv1.NodeSchedulable))
	_, err := c.clientset.CoreV1().Nodes().Patch(context.Background(), node.Name, types.StrategicMergePatchType, data, metav1.PatchOptions{})
	if err != nil {
		logger.Reason(err).Error("Failed to mark node as unschedulable")
		return fmt.Errorf("failed to mark node %s as unschedulable: %v", node.Name, err)
	}

	return nil
}

func (c *NodeController) createEventIfNodeHasOrphanedVMIs(node *v1.Node, vmis []*virtv1.VirtualMachineInstance) error {
	// node is not running any vmis so we don't need to check anything else
	if len(vmis) == 0 || node == nil {
		return nil
	}

	// query for a virt-handler pod on the node
	handlerNodeSelector := fields.ParseSelectorOrDie("spec.nodeName=" + node.GetName())
	virtHandlerSelector := fields.ParseSelectorOrDie("kubevirt.io=virt-handler")
	pods, err := c.clientset.CoreV1().Pods(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{
		FieldSelector: handlerNodeSelector.String(),
		LabelSelector: virtHandlerSelector.String(),
	})

	if err != nil {
		return err
	}

	// node is running the virt-handler
	if len(pods.Items) != 0 {
		return nil
	}

	running, err := checkDaemonSetStatus(c.clientset, virtHandlerSelector)
	if err != nil {
		return err
	}

	// the virt-handler DaemonsSet is not running as expect so we can't know for sure
	// if a virt-handler pod will be ran on this node
	if !running {
		c.requeueIfExists(node.GetName(), node)
		return nil
	}

	c.recorder.Event(node, v1.EventTypeWarning, NodeUnresponsiveReason, "virt-handler is not present, there are orphaned vmis on this node. Run virt-handler on this node to migrate or remove them.")

	return nil
}

func checkDaemonSetStatus(clientset kubecli.KubevirtClient, selector fields.Selector) (bool, error) {
	dss, err := clientset.AppsV1().DaemonSets(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})

	if err != nil {
		return false, err
	}

	if len(dss.Items) != 1 {
		return false, fmt.Errorf("shouuld only be running one virt-handler DaemonSet")
	}

	ds := dss.Items[0]
	desired, scheduled, ready := ds.Status.DesiredNumberScheduled, ds.Status.CurrentNumberScheduled, ds.Status.NumberReady
	if desired != scheduled && desired != ready {
		return false, nil
	}

	return true, nil
}

func (c *NodeController) alivePodsOnNode(nodeName string) ([]*v1.Pod, error) {
	handlerNodeSelector := fields.ParseSelectorOrDie("spec.nodeName=" + nodeName)
	list, err := c.clientset.CoreV1().Pods(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{
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

		// Some pods get stuck in a pending Termination during shutdown
		// due to virt-handler not being available to unmount container disk
		// mount propagation. A pod with all containers terminated is not
		// considered alive
		allContainersTerminated := false
		if len(pod.Status.ContainerStatuses) > 0 {
			allContainersTerminated = true
			for _, status := range pod.Status.ContainerStatuses {
				if status.State.Terminated == nil {
					allContainersTerminated = false
					break
				}
			}
		}

		phase := pod.Status.Phase
		toAppendPod := !allContainersTerminated && phase != v1.PodFailed && phase != v1.PodSucceeded
		if toAppendPod {
			pods = append(pods, pod)
			continue
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
		if podsForVMI, exists := podsPerNamespace[vmi.Namespace]; exists {
			if _, exists := podsForVMI[string(vmi.UID)]; exists {
				continue
			}
		}
		filtered = append(filtered, vmi)
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
