package machinehealthcheck

import (
	"context"
	golangerrors "errors"
	"fmt"
	"time"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	mrv1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"
	disruption "kubevirt.io/machine-remediation-operator/pkg/controllers/machinedisruptionbudget"
	"kubevirt.io/machine-remediation-operator/pkg/utils/conditions"

	mapiv1 "sigs.k8s.io/cluster-api/pkg/apis/machine/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	machineAnnotationKey           = "machine.openshift.io/machine"
	ownerControllerKind            = "MachineSet"
	disableRemediationAnotationKey = "healthchecking.openshift.io/disabled"
)

var _ reconcile.Reconciler = &ReconcileMachineHealthCheck{}

// Add creates a new MachineHealthCheck Controller and adds it to the Manager. The Manager will set fields on the Controller
// and start it when the Manager is started.
func Add(mgr manager.Manager, opts manager.Options) error {
	r := newReconciler(mgr, opts)
	return add(mgr, r, r.nodeRequestsFromMachineHealthCheck)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, opts manager.Options) *ReconcileMachineHealthCheck {
	return &ReconcileMachineHealthCheck{
		client:    mgr.GetClient(),
		namespace: opts.Namespace,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler, mapFn handler.ToRequestsFunc) error {
	// Create a new controller
	c, err := controller.New("machinehealthcheck-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch MachineHealthChecks and enqueue reconcile.Request for the backed nodes.
	// This is useful to trigger remediation when a machineHealCheck is created against
	// a node which is already unhealthy and is not able to receive status updates.
	err = c.Watch(&source.Kind{Type: &mrv1.MachineHealthCheck{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn})
	if err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &corev1.Node{}}, &handler.EnqueueRequestForObject{})
}

func (r *ReconcileMachineHealthCheck) nodeRequestsFromMachineHealthCheck(o handler.MapObject) []reconcile.Request {
	glog.V(3).Infof("Watched machineHealthCheck event, finding nodes to reconcile.Request...")
	key := client.ObjectKey{
		Namespace: o.Meta.GetNamespace(),
		Name:      o.Meta.GetName(),
	}
	mhc := &mrv1.MachineHealthCheck{}
	if err := r.client.Get(context.Background(), key, mhc); err != nil {
		glog.Errorf("No-op: Unable to retrieve mhc %s/%s from store: %v", o.Meta.GetNamespace(), o.Meta.GetName(), err)
		return nil
	}

	if mhc.DeletionTimestamp != nil {
		glog.V(3).Infof("No-op: mhc %q is being deleted", o.Meta.GetName())
		return nil
	}

	// get nodes covered by then mhc
	nodeNames, err := r.getNodeNamesForMHC(mhc)
	if err != nil {
		glog.Errorf("No-op: failed to get nodes for mhc %q", o.Meta.GetName())
		return nil
	}

	if nodeNames != nil {
		var requests []reconcile.Request
		for _, nodeName := range nodeNames {
			// convert to namespacedName to satisfy type Request struct
			nodeNamespacedName := client.ObjectKey{Name: string(nodeName)}
			requests = append(requests, reconcile.Request{NamespacedName: nodeNamespacedName})
		}
		return requests
	}
	return nil
}

func (r *ReconcileMachineHealthCheck) getNodeNamesForMHC(mhc *mrv1.MachineHealthCheck) ([]types.NodeName, error) {
	machineList := &mapiv1.MachineList{}
	selector, err := metav1.LabelSelectorAsSelector(&mhc.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("failed to build selector")
	}
	options := &client.ListOptions{
		Namespace:     mhc.Namespace,
		LabelSelector: selector,
	}

	if err := r.client.List(context.Background(), machineList, client.UseListOptions(options)); err != nil {
		return nil, fmt.Errorf("failed to list machines: %v", err)
	}

	if len(machineList.Items) < 1 {
		return nil, nil
	}

	var nodeNames []types.NodeName
	for _, machine := range machineList.Items {
		if machine.Status.NodeRef != nil {
			nodeNames = append(nodeNames, types.NodeName(machine.Status.NodeRef.Name))
		}
	}
	if len(nodeNames) < 1 {
		return nil, nil
	}
	return nodeNames, nil
}

// ReconcileMachineHealthCheck reconciles a MachineHealthCheck object
type ReconcileMachineHealthCheck struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client    client.Client
	namespace string
}

// Reconcile reads that state of the cluster for MachineHealthCheck, machine and nodes objects and makes changes based on the state read
// and what is in the MachineHealthCheck.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileMachineHealthCheck) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	glog.Infof("Reconciling MachineHealthCheck triggered by %s/%s\n", request.Namespace, request.Name)

	// Get node from request
	node := &corev1.Node{}
	err := r.client.Get(context.TODO(), request.NamespacedName, node)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	machineKey, ok := node.Annotations[machineAnnotationKey]
	if !ok {
		glog.Warningf("No machine annotation for node %s", node.Name)
		return reconcile.Result{}, nil
	}

	glog.Infof("Node %s is annotated with machine %s", node.Name, machineKey)
	machine := &mapiv1.Machine{}
	namespace, machineName, err := cache.SplitMetaNamespaceKey(machineKey)
	if err != nil {
		return reconcile.Result{}, err
	}
	key := &types.NamespacedName{
		Namespace: namespace,
		Name:      machineName,
	}
	err = r.client.Get(context.TODO(), *key, machine)
	if err != nil {
		if errors.IsNotFound(err) {
			glog.Warningf("machine %s not found", machineKey)
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		glog.Errorf("error getting machine %s. Error: %v. Requeuing...", machineKey, err)
		return reconcile.Result{}, err
	}

	if hasRemediationDisabledAnnotation(*machine) {
		glog.Infof("Machine %q has a matching %s annotation set to <true>, remediation is skipped.", machine.Name, disableRemediationAnotationKey)
		return reconcile.Result{}, nil
	}

	// If the current machine matches any existing MachineHealthCheck CRD
	allMachineHealthChecks := &mrv1.MachineHealthCheckList{}
	err = r.client.List(context.Background(), allMachineHealthChecks)
	if err != nil {
		glog.Errorf("failed to list MachineHealthChecks, %v", err)
		return reconcile.Result{}, err
	}

	for _, hc := range allMachineHealthChecks.Items {

		if hasMatchingLabels(&hc, machine) {
			glog.V(4).Infof("Machine %s has a matching machineHealthCheck: %s", machineKey, hc.Name)
			return remediate(r, hc.Spec.RemediationStrategy, machine)
		}
	}

	glog.Infof("Machine %s has no MachineHealthCheck associated", machineName)
	return reconcile.Result{}, nil
}

// This is set so the fake client can be used for unit test. See:
// https://github.com/kubernetes-sigs/controller-runtime/issues/168
func getMachineHealthCheckListOptions() *client.ListOptions {
	return &client.ListOptions{
		Raw: &metav1.ListOptions{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "healthchecking.openshift.io/v1alpha1",
				Kind:       "MachineHealthCheck",
			},
		},
	}
}

func remediate(r *ReconcileMachineHealthCheck, remediationStrategy *mrv1.RemediationStrategyType, machine *mapiv1.Machine) (reconcile.Result, error) {
	glog.Infof("Initialising remediation logic for machine %s", machine.Name)
	if !hasMachineSetOwner(machine) && !isMaster(machine, r.client) {
		glog.Infof("Machine %s has no machineSet controller owner, skipping remediation", machine.Name)
		return reconcile.Result{}, nil
	}

	node, err := getNodeFromMachine(machine, r.client)
	if err != nil {
		if errors.IsNotFound(err) {
			glog.Warningf("Node %s not found for machine %s", node.Name, machine.Name)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	cmUnhealtyConditions, err := conditions.GetUnhealthyConditionsConfigMap(r.client, r.namespace)
	if err != nil {
		return reconcile.Result{}, err
	}

	nodeUnhealthyConditions, err := conditions.GetNodeUnhealthyConditions(node, cmUnhealtyConditions)
	if err != nil {
		return reconcile.Result{}, err
	}

	var result *reconcile.Result
	var minimalConditionTimeout time.Duration
	minimalConditionTimeout = 0
	for _, c := range nodeUnhealthyConditions {
		nodeCondition := conditions.GetNodeCondition(node, c.Name)
		// skip when current node condition is different from the one reported in the config map
		if nodeCondition == nil || !isConditionsStatusesEqual(nodeCondition, &c) {
			continue
		}

		conditionTimeout, err := time.ParseDuration(c.Timeout)
		if err != nil {
			return reconcile.Result{}, err
		}

		// apply remediation logic, if at least one condition last more than specified timeout
		// specific remediation logic goes here
		if unhealthyForTooLong(nodeCondition, conditionTimeout) {
			// do not fail immediatlty, but try again if the method fails because of the update conflict
			if err = disruption.RetryDecrementMachineDisruptionsAllowed(r.client, machine); err != nil {
				// if the error appears here it means that machine healthcheck operation restricted by machine
				// disruption budget, in this case we want to re-try after one minute
				glog.Warning(err)
				return reconcile.Result{Requeue: true, RequeueAfter: time.Minute}, nil
			}

			if remediationStrategy != nil && *remediationStrategy == mrv1.RemediationStrategyTypeReboot {
				return r.remediationStrategyReboot(machine, node)
			}

			if isMaster(machine, r.client) {
				glog.Infof("The machine %s is a master node, skipping remediation", machine.Name)
				return reconcile.Result{}, nil
			}
			glog.Infof("Machine %s has been unhealthy for too long, deleting", machine.Name)
			if err := r.client.Delete(context.TODO(), machine); err != nil {
				glog.Errorf("Failed to delete machine %s, requeuing referenced node", machine.Name)
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, nil
		}

		now := time.Now()
		durationUnhealthy := now.Sub(nodeCondition.LastTransitionTime.Time)
		glog.Warningf(
			"Machine %s has unhealthy node %s with the condition %s and the timeout %s for %s. Requeuing...",
			machine.Name,
			node.Name,
			nodeCondition.Type,
			c.Timeout,
			durationUnhealthy.String(),
		)

		// calculate the duration until the node will be unhealthy for too long
		// and re-queue after with this timeout, add one second just to be sure
		// that we will not enter this loop again before the node unhealthy for too long
		unhealthyTooLongTimeout := conditionTimeout - durationUnhealthy + time.Second
		// be sure that we will use timeout with the minimal value for the reconcile.Result
		if minimalConditionTimeout == 0 || minimalConditionTimeout > unhealthyTooLongTimeout {
			minimalConditionTimeout = unhealthyTooLongTimeout
		}
		result = &reconcile.Result{Requeue: true, RequeueAfter: minimalConditionTimeout}
	}

	// requeue
	if result != nil {
		return *result, nil
	}

	glog.Infof("No remediaton action was taken. Machine %s with node %v is healthy", machine.Name, node.Name)
	return reconcile.Result{}, nil
}

func (r *ReconcileMachineHealthCheck) remediationStrategyReboot(machine *mapiv1.Machine, node *corev1.Node) (reconcile.Result, error) {
	rebootInProgress, err := isRebootInProgress(r.client, machine.Name)
	if err != nil {
		return reconcile.Result{}, err
	}

	// reboot already in progress, stop reconcile
	if rebootInProgress {
		return reconcile.Result{}, nil
	}

	mr := &mrv1.MachineRemediation{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "remediation-",
			Namespace:    machine.Namespace,
		},
		Spec: mrv1.MachineRemediationSpec{
			MachineName: machine.Name,
			Type:        mrv1.RemediationTypeReboot,
		},
	}

	glog.Infof("Machine %s has been unhealthy for too long, creating machine remediation", machine.Name)
	if err = r.client.Create(context.TODO(), mr); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func isConditionsStatusesEqual(cond *corev1.NodeCondition, unhealthyCond *conditions.UnhealthyCondition) bool {
	return cond.Status == unhealthyCond.Status
}

func getNodeFromMachine(machine *mapiv1.Machine, client client.Client) (*corev1.Node, error) {
	if machine.Status.NodeRef == nil {
		glog.Errorf("node NodeRef not found in machine %s", machine.Name)
		return nil, golangerrors.New("node NodeRef not found in machine")
	}
	node := &corev1.Node{}
	nodeKey := types.NamespacedName{
		Namespace: machine.Status.NodeRef.Namespace,
		Name:      machine.Status.NodeRef.Name,
	}
	err := client.Get(context.TODO(), nodeKey, node)
	return node, err
}

func unhealthyForTooLong(nodeCondition *corev1.NodeCondition, timeout time.Duration) bool {
	now := time.Now()
	if nodeCondition.LastTransitionTime.Add(timeout).Before(now) {
		return true
	}
	return false
}

func hasMachineSetOwner(machine *mapiv1.Machine) bool {
	ownerRefs := machine.ObjectMeta.GetOwnerReferences()
	for _, or := range ownerRefs {
		if or.Kind == ownerControllerKind {
			return true
		}
	}
	return false
}

func hasRemediationDisabledAnnotation(machine mapiv1.Machine) bool {
	skipRemediation, ok := machine.Annotations[disableRemediationAnotationKey]
	if !ok {
		return false
	}
	return skipRemediation == "true"
}

func hasMatchingLabels(machineHealthCheck *mrv1.MachineHealthCheck, machine *mapiv1.Machine) bool {
	selector, err := metav1.LabelSelectorAsSelector(&machineHealthCheck.Spec.Selector)
	if err != nil {
		glog.Warningf("unable to convert selector: %v", err)
		return false
	}
	// If a deployment with a nil or empty selector creeps in, it should match nothing, not everything.
	if selector.Empty() {
		glog.V(2).Infof("%v machineHealthCheck has empty selector", machineHealthCheck.Name)
		return false
	}
	if !selector.Matches(labels.Set(machine.Labels)) {
		glog.V(4).Infof("%v machine has mismatched labels", machine.Name)
		return false
	}
	return true
}

func isMaster(machine *mapiv1.Machine, client client.Client) bool {
	masterLabels := []string{
		"node-role.kubernetes.io/master",
	}

	node, err := getNodeFromMachine(machine, client)
	if err != nil {
		glog.Warningf("Couldn't get node for machine %s", machine.Name)
		return false
	}
	nodeLabels := labels.Set(node.Labels)
	for _, masterLabel := range masterLabels {
		if nodeLabels.Has(masterLabel) {
			return true
		}
	}
	return false
}

func isRebootInProgress(c client.Client, machineName string) (bool, error) {
	machineRemediations := &mrv1.MachineRemediationList{}
	if err := c.List(context.TODO(), machineRemediations); err != nil {
		return false, err
	}

	for _, mr := range machineRemediations.Items {
		if mr.Spec.MachineName == machineName {
			if mr.Status.EndTime != nil {
				return true, nil
			}
		}
	}
	return false, nil
}
