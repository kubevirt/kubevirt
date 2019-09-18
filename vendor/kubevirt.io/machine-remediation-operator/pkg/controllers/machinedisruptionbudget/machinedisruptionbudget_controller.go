package disruption

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"

	v1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"

	mrv1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"
	machineutil "kubevirt.io/machine-remediation-operator/pkg/utils/machines"

	mapiv1 "sigs.k8s.io/cluster-api/pkg/apis/machine/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// DeletionTimeout sets maximum time from the moment a machine is added to DisruptedMachines in MDB.Status
	// to the time when the machine is expected to be seen by MDB controller as having been marked for deletion.
	// If the machine was not marked for deletion during that time it is assumed that it won't be deleted at
	// all and the corresponding entry can be removed from mdb.Status.DisruptedMachines. It is assumed that
	// machine/mdb apiserver to controller latency is relatively small (like 1-2sec) so the below value should
	// be more than enough.
	// If the controller is running on a different node it is important that the two nodes have synced
	// clock (via ntp for example). Otherwise MachineDisruptionBudget controller may not provide enough
	// protection against unwanted machine disruptions.
	DeletionTimeout = 2 * time.Minute
	// maxDisruptedMachinSize is the max size of MachineDisruptionBudgetStatus.DisruptedMachines.
	// MachineHealthCheck will refuse to delete machine covered by the corresponding MDB
	// if the size of the map exceeds this value.
	maxDisruptedMachinSize = 50
)

// updateMDBRetry is the retry for a conflict where multiple clients
// are making changes to the same resource.
var updateMDBRetry = wait.Backoff{
	Steps:    20,
	Duration: 500 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

// Add creates a new MachineDisruption Controller and adds it to the Manager. The Manager will set fields on the Controller
// and start it when the Manager is started.
func Add(mgr manager.Manager, opts manager.Options) error {
	r := newReconciler(mgr, opts)
	return add(mgr, r, r.machineToMachineDisruptionBudget)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, opts manager.Options) *ReconcileMachineDisruption {
	return &ReconcileMachineDisruption{
		client:   mgr.GetClient(),
		recorder: mgr.GetEventRecorderFor("machine-disruption-controller"),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler, mapFn handler.ToRequestsFunc) error {
	// Create a new controller
	c, err := controller.New("MachineDisruption-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &mapiv1.Machine{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn}); err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &mrv1.MachineDisruptionBudget{}}, &handler.EnqueueRequestForObject{})
}

var _ reconcile.Reconciler = &ReconcileMachineDisruption{}

// ReconcileMachineDisruption reconciles a MachineDisruption object
type ReconcileMachineDisruption struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client   client.Client
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for MachineDisruptionBudget and machine objects and makes changes based on labels under
// MachineDisruptionBudget or machine objects
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileMachineDisruption) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	glog.V(4).Infof("Reconciling MachineDisruption triggered by %s/%s\n", request.Namespace, request.Name)

	// Get machine from request
	mdb := &mrv1.MachineDisruptionBudget{}
	err := r.client.Get(context.TODO(), request.NamespacedName, mdb)
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
	result, err := r.reconcile(mdb)
	if err != nil {
		glog.Errorf("Failed to reconcile mdb %s/%s: %v", mdb.Namespace, mdb.Name, err)
		err = r.failSafe(mdb)
	}
	return result, err
}

func (r *ReconcileMachineDisruption) reconcile(mdb *mrv1.MachineDisruptionBudget) (reconcile.Result, error) {
	machines, err := r.getMachinesForMachineDisruptionBudget(mdb)
	if err != nil {
		r.recorder.Eventf(mdb, v1.EventTypeWarning, "NoMachines", "Failed to get machines: %v", err)
		return reconcile.Result{}, err
	}

	if len(machines) == 0 {
		r.recorder.Eventf(mdb, v1.EventTypeNormal, "NoMachines", "No matching machines found")
	}

	total, desiredHealthy := r.getTotalAndDesiredMachinesCount(mdb, machines)

	currentTime := time.Now()
	disruptedMachines, recheckTime := r.buildDisruptedMachineMap(machines, mdb, currentTime)

	currentHealthy, err := r.countHealthyMachines(machines, disruptedMachines, currentTime)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.updateMachineDisruptionBudgetStatus(mdb, currentHealthy, desiredHealthy, total, disruptedMachines)
	if err != nil {
		return reconcile.Result{}, err
	}

	if recheckTime != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: recheckTime.Sub(currentTime)}, nil
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileMachineDisruption) getTotalAndDesiredMachinesCount(mdb *mrv1.MachineDisruptionBudget, machines []mapiv1.Machine) (total, desiredHealthy int32) {
	total = r.getTotalMachinesCount(mdb, machines)
	if mdb.Spec.MaxUnavailable != nil {
		desiredHealthy = total - int32(*mdb.Spec.MaxUnavailable)
		if desiredHealthy < 0 {
			desiredHealthy = 0
		}
	} else if mdb.Spec.MinAvailable != nil {
		desiredHealthy = *mdb.Spec.MinAvailable
	}
	return
}

// getTotalMachinesCount returns total number of machines that monitored by the MDB, if the machine has owner controller,
// it will get number of desired replicas from the controller and add it to the total number.
func (r *ReconcileMachineDisruption) getTotalMachinesCount(mdb *mrv1.MachineDisruptionBudget, machines []mapiv1.Machine) int32 {
	// When the user specifies a fraction of machines that must be available, we
	// use as the fraction's denominator
	// SUM_{all c in C} scale(c)
	// where C is the union of C_m1, C_m2, ..., C_mN
	// and each C_mi is the set of controllers controlling the machine mi

	// A mapping from controllers to their scale.
	controllerScale := map[types.UID]int32{}

	// 1. Find the controller for each machine. If any machine has 0 controllers,
	// it will add map item with machine.UID as a key and 1 as a value.
	// With ControllerRef, a machine can only have 1 controller.
	for _, machine := range machines {
		foundController := false
		for _, finder := range r.finders() {
			controllerNScale := finder(&machine)
			if controllerNScale != nil {
				if _, ok := controllerScale[controllerNScale.UID]; !ok {
					controllerScale[controllerNScale.UID] = controllerNScale.scale
				}
				foundController = true
				break
			}
		}
		if !foundController {
			controllerScale[machine.UID] = 1
		}
	}

	// 2. Sum up all relevant machine scales to get the expected number
	var total int32
	for _, count := range controllerScale {
		total += count
	}
	return total
}

type controllerAndScale struct {
	types.UID
	scale int32
}

// machineControllerFinder is a function type that maps a machine to a list of
// controllers and their scale.
type machineControllerFinder func(*mapiv1.Machine) *controllerAndScale

var (
	controllerKindMachineSet        = mapiv1.SchemeGroupVersion.WithKind("MachineSet")
	controllerKindMachineDeployment = mapiv1.SchemeGroupVersion.WithKind("MachineDeployment")
)

func (r *ReconcileMachineDisruption) finders() []machineControllerFinder {
	return []machineControllerFinder{r.getMachineSetFinder, r.getMachineDeploymentFinder}
}

func (r *ReconcileMachineDisruption) getMachineMachineSet(machine *mapiv1.Machine) *mapiv1.MachineSet {
	controllerRef := metav1.GetControllerOf(machine)
	if controllerRef == nil {
		glog.Infof("machine %s does not have owner reference", machine.Name)
		return nil
	}
	if controllerRef.Kind != controllerKindMachineSet.Kind {
		// Skip MachineSet if the machine controlled by different controller
		return nil
	}

	machineSet := &mapiv1.MachineSet{}
	key := client.ObjectKey{Namespace: machine.Namespace, Name: controllerRef.Name}
	err := r.client.Get(context.TODO(), key, machineSet)
	if err != nil {
		glog.Infof("failed to get machine set object for machine %s", machine.Name)
		return nil
	}

	if machineSet.UID != controllerRef.UID {
		glog.Infof("machine %s owner reference UID is different from machines set %s UID", machine.Name, machineSet.Name)
		return nil
	}

	return machineSet
}

func (r *ReconcileMachineDisruption) getMachineSetFinder(machine *mapiv1.Machine) *controllerAndScale {
	machineSet := r.getMachineMachineSet(machine)
	if machineSet == nil {
		return nil
	}

	controllerRef := metav1.GetControllerOf(machineSet)
	if controllerRef != nil && controllerRef.Kind == controllerKindMachineDeployment.Kind {
		// Skip MachineSet if it's controlled by a Deployment.
		return nil
	}
	return &controllerAndScale{machineSet.UID, *(machineSet.Spec.Replicas)}
}

func (r *ReconcileMachineDisruption) getMachineDeploymentFinder(machine *mapiv1.Machine) *controllerAndScale {
	machineSet := r.getMachineMachineSet(machine)
	if machineSet == nil {
		return nil
	}

	controllerRef := metav1.GetControllerOf(machineSet)
	if controllerRef == nil {
		return nil
	}
	if controllerRef.Kind != controllerKindMachineDeployment.Kind {
		return nil
	}
	machineDeployment := &mapiv1.MachineDeployment{}
	key := client.ObjectKey{Namespace: machine.Namespace, Name: controllerRef.Name}
	err := r.client.Get(context.TODO(), key, machineDeployment)
	if err != nil {
		// The only possible error is NotFound, which is ok here.
		return nil
	}
	if machineDeployment.UID != controllerRef.UID {
		return nil
	}
	return &controllerAndScale{machineDeployment.UID, *(machineDeployment.Spec.Replicas)}
}

func (r *ReconcileMachineDisruption) countHealthyMachines(machines []mapiv1.Machine, disruptedMachines map[string]metav1.Time, currentTime time.Time) (int32, error) {
	var currentHealthy int32

	for _, machine := range machines {
		// Machine is being deleted.
		if machine.DeletionTimestamp != nil {
			continue
		}
		// Machine is expected to be deleted soon.
		if disruptionTime, found := disruptedMachines[machine.Name]; found && disruptionTime.Time.Add(DeletionTimeout).After(currentTime) {
			continue
		}

		healthy, err := machineutil.IsMachineHealthy(r.client, &machine)
		if err != nil {
			return currentHealthy, err
		}
		if healthy {
			currentHealthy++
		}
	}
	return currentHealthy, nil
}

func (r *ReconcileMachineDisruption) updateMachineDisruptionBudgetStatus(
	mdb *mrv1.MachineDisruptionBudget,
	currentHealthy,
	desiredHealthy,
	total int32,
	disruptedMachines map[string]metav1.Time) error {

	// we add one because we do not want to respect disruption budget when expected and healthy are equal
	disruptionsAllowed := currentHealthy - desiredHealthy + 1

	// We require expectedCount to be > 0 so that MDBs which currently match no
	// machines are in a safe state when their first machines appear but this controller
	// has not updated their status yet.  This isn't the only race, but it's a
	// common one that's easy to detect.
	if total <= 0 || disruptionsAllowed <= 0 {
		disruptionsAllowed = 0
	}

	if mdb.Status.CurrentHealthy == currentHealthy &&
		mdb.Status.DesiredHealthy == desiredHealthy &&
		mdb.Status.Total == total &&
		mdb.Status.MachineDisruptionsAllowed == disruptionsAllowed &&
		apiequality.Semantic.DeepEqual(mdb.Status.DisruptedMachines, disruptedMachines) &&
		mdb.Status.ObservedGeneration == mdb.Generation {
		return nil
	}

	newMdb := mdb.DeepCopy()
	newMdb.Status = mrv1.MachineDisruptionBudgetStatus{
		CurrentHealthy:            currentHealthy,
		DesiredHealthy:            desiredHealthy,
		Total:                     total,
		MachineDisruptionsAllowed: disruptionsAllowed,
		DisruptedMachines:         disruptedMachines,
		ObservedGeneration:        mdb.Generation,
	}

	return r.client.Status().Update(context.TODO(), newMdb)
}

// failSafe is an attempt to at least update the MachineDisruptionsAllowed field to
// 0 if everything else has failed.  This is one place we
// implement the  "fail open" part of the design since if we manage to update
// this field correctly, we will prevent the deletion when it may be unsafe to do
func (r *ReconcileMachineDisruption) failSafe(mdb *mrv1.MachineDisruptionBudget) error {
	newMdb := mdb.DeepCopy()
	mdb.Status.MachineDisruptionsAllowed = 0
	return r.client.Status().Update(context.TODO(), newMdb)
}

func (r *ReconcileMachineDisruption) getMachineDisruptionBudgetForMachine(machine *mapiv1.Machine) *mrv1.MachineDisruptionBudget {
	// GetMachineMachineDisruptionBudgets returns an error only if no
	// MachineDisruptionBudgets are found.  We don't return that as an error to the
	// caller.
	mdbs, err := machineutil.GetMachineMachineDisruptionBudgets(r.client, machine)
	if err != nil {
		glog.V(4).Infof("No MachineDisruptionBudgets found for machine %v, MachineDisruptionBudget controller will avoid syncing.", machine.Name)
		return nil
	}

	if len(mdbs) == 0 {
		glog.V(4).Infof("Could not find MachineDisruptionBudget for machine %s in namespace %s with labels: %v", machine.Name, machine.Namespace, machine.Labels)
		return nil
	}

	if len(mdbs) > 1 {
		msg := fmt.Sprintf("Machine %q/%q matches multiple MachineDisruptionBudgets.  Chose %q arbitrarily.", machine.Namespace, machine.Name, mdbs[0].Name)
		glog.Warning(msg)
		r.recorder.Event(machine, v1.EventTypeWarning, "MultipleMachineDisruptionBudgets", msg)
	}
	return mdbs[0]
}

// This function returns machines using the MachineDisruptionBudget object.
func (r *ReconcileMachineDisruption) getMachinesForMachineDisruptionBudget(mdb *mrv1.MachineDisruptionBudget) ([]mapiv1.Machine, error) {
	sel, err := metav1.LabelSelectorAsSelector(mdb.Spec.Selector)
	if err != nil {
		return nil, err
	}
	if sel.Empty() {
		return nil, nil
	}

	machines := &mapiv1.MachineList{}
	listOptions := &client.ListOptions{
		Namespace:     mdb.Namespace,
		LabelSelector: sel,
	}
	err = r.client.List(context.TODO(), machines, client.UseListOptions(listOptions))
	if err != nil {
		return nil, err
	}
	return machines.Items, nil
}

// Builds new MachineDisruption map, possibly removing items that refer to non-existing, already deleted
// or not-deleted at all items. Also returns an information when this check should be repeated.
func (r *ReconcileMachineDisruption) buildDisruptedMachineMap(machines []mapiv1.Machine, mdb *mrv1.MachineDisruptionBudget, currentTime time.Time) (map[string]metav1.Time, *time.Time) {
	disruptedMachines := mdb.Status.DisruptedMachines
	result := make(map[string]metav1.Time)
	var recheckTime *time.Time

	if disruptedMachines == nil || len(disruptedMachines) == 0 {
		return result, recheckTime
	}
	for _, machine := range machines {
		if machine.DeletionTimestamp != nil {
			// Already being deleted.
			continue
		}
		disruptionTime, found := disruptedMachines[machine.Name]
		if !found {
			// Machine not on the list.
			continue
		}
		expectedDeletion := disruptionTime.Time.Add(DeletionTimeout)
		if expectedDeletion.Before(currentTime) {
			glog.V(1).Infof("Machine %s/%s was expected to be deleted at %s but it wasn't, updating mdb %s/%s",
				machine.Namespace, machine.Name, disruptionTime.String(), mdb.Namespace, mdb.Name)
			r.recorder.Eventf(&machine, v1.EventTypeWarning, "NotDeleted", "Machine was expected by MDB %s/%s to be deleted but it wasn't",
				mdb.Namespace, mdb.Namespace)
		} else {
			if recheckTime == nil || expectedDeletion.Before(*recheckTime) {
				recheckTime = &expectedDeletion
			}
			result[machine.Name] = disruptionTime
		}
	}
	return result, recheckTime
}

func (r *ReconcileMachineDisruption) machineToMachineDisruptionBudget(o handler.MapObject) []reconcile.Request {
	machine := &mapiv1.Machine{}
	key := client.ObjectKey{Namespace: o.Meta.GetNamespace(), Name: o.Meta.GetName()}
	if err := r.client.Get(context.TODO(), key, machine); err != nil {
		glog.V(4).Infof("Unable to retrieve Machine %v from store, uses a dummy machine to get MDB object: %v", key, err)
		machine.Name = o.Meta.GetName()
		machine.Namespace = o.Meta.GetNamespace()
		machine.Labels = o.Meta.GetLabels()
	}

	mdb := r.getMachineDisruptionBudgetForMachine(machine)
	if mdb == nil {
		glog.Errorf("Unable to find MachineDisruptionBudget for machine %s", machine.Name)
		return nil
	}

	name := client.ObjectKey{Namespace: mdb.Namespace, Name: mdb.Name}
	return []reconcile.Request{{NamespacedName: name}}
}

// isMachineDisruptionAllowed returns true if the provided MachineDisruptionBudget allows any disruption
func isMachineDisruptionAllowed(mdb *mrv1.MachineDisruptionBudget, maxDisruptedMachinSize int) bool {
	if mdb.Status.ObservedGeneration < mdb.Generation {
		glog.Warningf("The machine disruption budget %s is still being processed by the server", mdb.Name)
		return false
	}
	if mdb.Status.MachineDisruptionsAllowed < 0 {
		glog.Warningf("The machine disruption budget %s MachineDisruptionsAllowed is negative", mdb.Name)
		return false
	}
	if len(mdb.Status.DisruptedMachines) > maxDisruptedMachinSize {
		glog.Warningf("The machine disruption budget %s DisruptedMachines map too big - too many deletions not confirmed by MDB controller", mdb.Name)
		return false
	}
	if mdb.Status.MachineDisruptionsAllowed == 0 {
		glog.Warningf("Cannot remediate machine as it would violate the machine's disruption budget %s", mdb.Name)
		return false
	}

	return true
}

func decrementMachineDisruptionsAllowed(c client.Client, machineName string, mdb *mrv1.MachineDisruptionBudget) error {
	if mdb.Status.DisruptedMachines == nil {
		mdb.Status.DisruptedMachines = make(map[string]metav1.Time)
	}

	if _, exists := mdb.Status.DisruptedMachines[machineName]; exists {
		return nil
	}

	mdb.Status.MachineDisruptionsAllowed--

	// MachineHealthCheck controller needs to inform the MDB controller that it is about to remediate a machine
	// so it should not consider it as available in calculations when updating MachineDisruptions allowed.
	// If the machine is not remediated within a reasonable time limit MDB controller will assume that it won't
	// be remediated at all and remove it from DisruptedMachines map.
	mdb.Status.DisruptedMachines[machineName] = metav1.Time{Time: time.Now()}
	return c.Status().Update(context.TODO(), mdb)
}

// RetryDecrementMachineDisruptionsAllowed validates if the disruption is allowed, when it allowed it will decrement
// MDB MachineDisruptionsAllowed parameter and update the status of the MDB, in case when update failed
// on the conflict it will try again with the backoff
func RetryDecrementMachineDisruptionsAllowed(c client.Client, machine *mapiv1.Machine) error {
	var mdb *mrv1.MachineDisruptionBudget
	err := retry.RetryOnConflict(updateMDBRetry, func() error {
		mdbs, err := machineutil.GetMachineMachineDisruptionBudgets(c, machine)
		if err != nil {
			return err
		}

		if len(mdbs) > 1 {
			return fmt.Errorf("machine %q has more than one MachineDisruptionBudget, which is not supported", machine.Name)
		}

		if len(mdbs) == 1 {
			mdb = mdbs[0]

			if !isMachineDisruptionAllowed(mdb, maxDisruptedMachinSize) {
				return fmt.Errorf("machine disruption is not allowed")
			}

			return decrementMachineDisruptionsAllowed(c, machine.Name, mdb)
		}
		return nil
	})

	if err == wait.ErrWaitTimeout {
		err = fmt.Errorf("couldn't update MachineDisruptionBudget %q due to conflicts", mdb.Name)
	}

	return err
}
