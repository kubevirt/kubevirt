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
	nodeInformer     cache.SharedIndexInformer
	recorder         record.EventRecorder
	heartBeatTimeout time.Duration
	recheckInterval  time.Duration
}

// NewNodeController creates a new instance of the NodeController struct.
func NewNodeController(clientset kubecli.KubevirtClient, nodeInformer cache.SharedIndexInformer, recorder record.EventRecorder) *NodeController {
	c := &NodeController{
		clientset:        clientset,
		nodeInformer:     nodeInformer,
		recorder:         recorder,
		heartBeatTimeout: 5 * time.Minute,
		recheckInterval:  1 * time.Minute,
	}
	return c
}

func (c *NodeController) Run(stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	log.Log.Info("Starting node controller.")
	interval := 1 * time.Minute
	wait.JitterUntil(c.execute, interval, -1, true, stopCh)
}

func (c *NodeController) execute() {
	logger := log.DefaultLogger()
	nodes := c.nodeInformer.GetStore().List()

	for _, nodeToConvert := range nodes {
		node := nodeToConvert.(*v1.Node)
		unresponsive, err := isNodeUnresponsive(node, c.heartBeatTimeout)
		if err != nil {
			logger.Reason(err).Error(fmt.Sprintf("NodeController Failed to determine if node %v is responsive", node.Name))
			continue
		}

		if unresponsive && nodeIsSchedulable(node) {
			if err := c.markNodeAsUnresponsive(node, logger); err != nil {
				continue
			}
		} else if unresponsive {
			err = c.checkNodeForOrphanedAndErroredVMIs(node, logger)
			if err != nil {
				continue
			}
		}
	}
}

func nodeIsSchedulable(node *v1.Node) bool {
	if node == nil {
		return false
	}

	return node.Labels[virtv1.NodeSchedulable] == "true"
}

func (c *NodeController) checkNodeForOrphanedAndErroredVMIs(node *v1.Node, logger *log.FilteredLogger) error {
	vmis, err := lookup.ActiveVirtualMachinesOnNode(c.clientset, node.Name)
	if err != nil {
		logger.Reason(err).Error("Failed fetching vmis for node")
		return err
	}

	if len(vmis) == 0 {
		return nil
	}

	err = c.createEventIfNodeHasOrphanedVMIs(node, vmis)
	if err != nil {
		logger.Reason(err).Error("checking virt-handler for node")
		return err
	}

	return c.checkVirtLauncherPodsAndUpdateVMIStatus(node.Name, vmis, logger)
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
	// if a virt-handler pod will be run on this node
	if !running {
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
