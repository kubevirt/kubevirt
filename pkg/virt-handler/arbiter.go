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
 * Copyright The KubeVirt Authors.
 *
 */

package virthandler

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// vmiState is the snapshot of a VMI and its domain that the Arbiter reads once per
// pass and hands to exactly one controller. Reading it in a single place keeps the
// three controllers from disagreeing about whether a migration is in flight.
type vmiState struct {
	key             string
	vmi             *v1.VirtualMachineInstance
	vmiExists       bool
	domain          *api.Domain
	domainExists    bool
	domainCachedUID types.UID
}

// vmiController is the part of a virt-handler controller the Arbiter drives.
type vmiController interface {
	execute(state *vmiState) error
}

type role int

const (
	// roleNone means no controller may act on the key in this pass.
	roleNone role = iota
	roleVM
	roleSource
	roleTarget
)

func (r role) String() string {
	switch r {
	case roleVM:
		return "vm"
	case roleSource:
		return "migration-source"
	case roleTarget:
		return "migration-target"
	default:
		return "none"
	}
}

// migrationRole decides which controller owns a VMI on this node for one pass of
// the Arbiter: the target while this node holds the migration target claim, the
// source while an outgoing migration is in flight, and the VM controller
// otherwise - including for every VMI that is terminal or going away, and for the
// local cleanup of a domain whose VMI is gone.
//
// migrationTargetOwns and migrationSourceOwns are mutually exclusive by
// construction - virt-controller writes the target node label and
// MigrationState.SourceNode in the same update, and they never name this host at
// once - so their order below only decides precedence against the VM controller.
//
// Hand-over points are chosen so that the controller giving up ownership is the one
// that publishes the release.
func migrationRole(vmi *v1.VirtualMachineInstance, vmiExists bool, domain *api.Domain, host string) role {
	// No VMI left: all that remains is local cleanup of a leftover domain and its
	// ghost record, which only the VM controller does. vmi is never nil here:
	// getVMI only returns a nil VMI together with an error, and execute rejects
	// the broken invariant before the role is decided.
	if !vmiExists {
		return roleVM
	}

	if migrationTargetOwns(vmi, host) {
		return roleTarget
	}

	if migrationSourceOwns(vmi, host) {
		return roleSource
	}

	// A VMI that is going away always goes to the VM controller: local teardown and
	// releasing the ghost record are nobody else's job. This must come before the
	// in-progress check below, because a VMI that dies mid-migration leaves a
	// migration state that never reaches an end timestamp, and standing everybody
	// down there would leak the domain and the ghost record.
	if vmi.DeletionTimestamp != nil || vmi.IsFinal() {
		return roleVM
	}

	// Last line of defence: the cached VMI may not reflect a migration that the
	// domain already knows about. Nobody acts until the VMI catches up - the key is
	// re-queued by the next VMI or domain event.
	if isMigrationInProgress(vmi, domain) {
		return roleNone
	}

	return roleVM
}

// migrationTargetOwns reports whether the migration target controller owns vmi on
// this node.
//
// Ownership starts when virt-controller labels the VMI with the node its target pod
// was scheduled on, and ends when the target controller itself drops that label,
// which it does as the very last step of finalCleanup. Because the owner publishes
// its own release, the VM controller cannot take over while the target is still
// finalizing - the window that EndTimestamp-based predicates leave wide open.
func migrationTargetOwns(vmi *v1.VirtualMachineInstance, host string) bool {
	if vmi.Labels[v1.MigrationTargetNodeNameLabel] != host {
		return false
	}

	// A terminal or terminating VMI goes to the VM controller, whose teardown is a
	// superset of this controller's best-effort cleanup. That holds both when
	// ackMigrationCompletion has already moved the VMI onto this node (an ordinary
	// local VMI), and when the guest died on the source mid-migration: there this
	// controller never gets past its best-effort cleanup, so it could not drop the
	// label itself, and holding the claim would keep the VM controller away from
	// the leftover domain and ghost record.
	if vmi.DeletionTimestamp != nil || vmi.IsFinal() {
		return false
	}

	return true
}

// migrationSourceOwns reports whether the migration source controller owns vmi on
// this node. It is deliberately the exact condition under which the source
// controller has work to do, so that the VM controller's own window is its
// complement rather than a second, independently drifting predicate.
func migrationSourceOwns(vmi *v1.VirtualMachineInstance, host string) bool {
	migrationState := vmi.Status.MigrationState
	if migrationState == nil || migrationState.SourceNode != host {
		return false
	}

	// A VMI that is going away is shut down and cleaned up by the VM controller, even
	// mid-migration. The source controller has nothing to contribute there.
	if vmi.DeletionTimestamp != nil {
		return false
	}

	// A terminal VMI is torn down locally by the VM controller, which is also the only
	// one that releases the ghost record. A decentralized source is the one exception,
	// and only for as long as it takes to record on the migration state a failure that
	// something else caused: nothing on this side of a decentralized migration would
	// otherwise do it, and that single update always makes progress, so ownership moves
	// on right after.
	if vmi.IsFinal() {
		return vmi.IsDecentralizedMigration() &&
			vmi.IsMigrationSource() &&
			vmi.Status.Phase == v1.Failed &&
			!isMigrationDone(migrationState)
	}

	if vmi.IsDecentralizedMigration() {
		// Before the two sides have synchronized there is nothing for the source
		// controller to do and the guest is simply running here.
		return vmi.IsMigrationSource()
	}

	// Once the migration is done the source domain is shut off with reason "migrated"
	// and the VM controller reaps it through its orphan path.
	return vmi.Status.NodeName == host && !isMigrationDone(migrationState)
}

// Arbiter is the single entry point for VMI processing in virt-handler. It drives
// the VM, migration-source and migration-target controllers and guarantees that
// only one of them ever acts on a given VMI at a time: one workqueue (which hands
// a key to at most one worker at a time), one read of the VMI and its domain per
// pass, and one role decision (see migrationRole) derived from that single
// snapshot. Two controllers acting on a VMI concurrently is therefore not
// something the controllers have to avoid, it is something they cannot express.
type Arbiter struct {
	logger *log.FilteredLogger
	host   string
	queue  workqueue.TypedRateLimitingInterface[string]

	// A VMI on this node is served by the source-scoped informer, the target-scoped
	// informer, or - while a hand-over is in progress and neither label points here
	// yet - only by the cluster-wide one.
	vmiSourceStore cache.Store
	vmiTargetStore cache.Store
	vmiGlobalStore cache.Store
	domainStore    cache.Store

	vmiExpectations *controller.UIDTrackingControllerExpectations

	hasSynced func() bool

	vmController     vmiController
	sourceController vmiController
	targetController vmiController
}

func NewArbiter(
	host string,
	queue workqueue.TypedRateLimitingInterface[string],
	vmiExpectations *controller.UIDTrackingControllerExpectations,
	vmiSourceInformer cache.SharedIndexInformer,
	vmiTargetInformer cache.SharedIndexInformer,
	domainInformer cache.SharedInformer,
	vmiGlobalStore cache.Store,
	vmController *VirtualMachineController,
	sourceController *MigrationSourceController,
	targetController *MigrationTargetController,
) (*Arbiter, error) {

	a := &Arbiter{
		logger:           log.Log.With("controller", "arbiter"),
		host:             host,
		queue:            queue,
		vmiExpectations:  vmiExpectations,
		vmiSourceStore:   vmiSourceInformer.GetStore(),
		vmiTargetStore:   vmiTargetInformer.GetStore(),
		vmiGlobalStore:   vmiGlobalStore,
		domainStore:      domainInformer.GetStore(),
		vmController:     vmController,
		sourceController: sourceController,
		targetController: targetController,
		hasSynced: func() bool {
			return vmiSourceInformer.HasSynced() &&
				vmiTargetInformer.HasSynced() &&
				domainInformer.HasSynced()
		},
	}

	for _, informer := range []cache.SharedIndexInformer{vmiSourceInformer, vmiTargetInformer} {
		_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    a.addVMIFunc,
			DeleteFunc: a.deleteVMIFunc,
			UpdateFunc: a.updateVMIFunc,
		})
		if err != nil {
			return nil, err
		}
	}

	_, err := domainInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    a.addDomainFunc,
		DeleteFunc: a.deleteDomainFunc,
		UpdateFunc: a.updateDomainFunc,
	})
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (a *Arbiter) Run(threadiness int, stopCh chan struct{}) {
	defer a.queue.ShutDown()
	a.logger.Info("Starting virt-handler arbiter.")

	cache.WaitForCacheSync(stopCh, a.hasSynced)

	a.enqueueUnclaimedDomains()

	for range threadiness {
		go wait.Until(a.runWorker, time.Second, stopCh)
	}

	<-stopCh
	a.logger.Info("Stopping virt-handler arbiter.")
}

// enqueueUnclaimedDomains queues the domains on this host that neither node-scoped
// informer serves, so local cleanup still happens for VMIs that went away, or
// migrated away, while virt-handler was down.
//
// It deliberately does not consult the cluster-wide store: a VMI that finished
// migrating off this node is still in it, and the source domain it left behind is
// precisely what has to be reaped here.
func (a *Arbiter) enqueueUnclaimedDomains() {
	for _, obj := range a.domainStore.List() {
		d := obj.(*api.Domain)
		vmiRef := v1.NewVMIReferenceWithUUID(
			d.ObjectMeta.Namespace,
			d.ObjectMeta.Name,
			d.Spec.Metadata.KubeVirt.UID)

		key := controller.VirtualMachineInstanceKey(vmiRef)

		_, onSource, _ := a.vmiSourceStore.GetByKey(key)
		_, onTarget, _ := a.vmiTargetStore.GetByKey(key)
		if !onSource && !onTarget {
			a.queue.Add(key)
		}
	}
}

func (a *Arbiter) runWorker() {
	for a.Execute() {
	}
}

func (a *Arbiter) Execute() bool {
	key, quit := a.queue.Get()
	if quit {
		return false
	}
	defer a.queue.Done(key)

	if err := a.execute(key); err != nil {
		a.logger.Reason(err).Infof("re-enqueuing VirtualMachineInstance %v", key)
		a.queue.AddRateLimited(key)
	} else {
		a.logger.V(4).Infof("processed VirtualMachineInstance %v", key)
		a.queue.Forget(key)
	}
	return true
}

func (a *Arbiter) execute(key string) error {
	state, err := a.snapshot(key)
	if err != nil {
		return err
	}

	// getVMI fabricates a VMI when the key resolves to nothing, so it never
	// returns a nil one without an error. A VMI that exists but is nil is a
	// broken invariant, not a state any controller could act on.
	if state.vmiExists && state.vmi == nil {
		return fmt.Errorf("internal error: no VMI found for %v", key)
	}

	if !state.vmiExists {
		a.vmiExpectations.DeleteExpectations(key)
	} else if !a.vmiExpectations.SatisfiedExpectations(key) {
		a.logger.V(4).Infof("waiting for expectations to be satisfied for %v", key)
		return nil
	}

	owner := migrationRole(state.vmi, state.vmiExists, state.domain, a.host)
	a.logger.V(4).Infof("handing %v to the %s controller", key, owner)

	switch owner {
	case roleVM:
		return a.vmController.execute(state)
	case roleSource:
		return a.sourceController.execute(state)
	case roleTarget:
		return a.targetController.execute(state)
	default:
		return nil
	}
}

// snapshot resolves the VMI and its domain exactly once per pass.
func (a *Arbiter) snapshot(key string) (*vmiState, error) {
	state := &vmiState{key: key}

	var err error
	state.vmi, state.vmiExists, err = a.getVMI(key)
	if err != nil {
		return nil, err
	}

	obj, domainExists, err := a.domainStore.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if domainExists {
		domain := obj.(*api.Domain)
		state.domainCachedUID = domain.Spec.Metadata.KubeVirt.UID
		// A DeletionTimestamp is how the domain informer signals a deleted domain,
		// instead of sending a DELETE watch event.
		if domain.ObjectMeta.DeletionTimestamp == nil {
			state.domain, state.domainExists = domain, true
		}
	}

	return state, nil
}

func (a *Arbiter) getVMI(key string) (*v1.VirtualMachineInstance, bool, error) {
	// The target-scoped informer selects on the migration target node label, which is
	// what the role decision hinges on, so its copy wins when it has one. The
	// cluster-wide store is only a fallback for the moment between a VMI losing one
	// node-scoped label and gaining the other.
	for _, store := range []cache.Store{a.vmiTargetStore, a.vmiSourceStore, a.vmiGlobalStore} {
		obj, exists, err := store.GetByKey(key)
		if err != nil {
			return nil, false, err
		}
		if exists {
			return obj.(*v1.VirtualMachineInstance), true, nil
		}
	}

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// Invalid keys will be retried forever, but the queue has no reason to contain any
		return nil, false, err
	}
	return libvmi.New(libvmi.WithName(name), libvmi.WithNamespace(namespace)), false, nil
}

func (a *Arbiter) addVMIFunc(obj any) {
	a.enqueueVMI(obj)
}

func (a *Arbiter) deleteVMIFunc(obj any) {
	a.enqueueVMI(obj)
}

func (a *Arbiter) updateVMIFunc(_, new any) {
	a.enqueueVMI(new)
}

func (a *Arbiter) enqueueVMI(obj any) {
	key, err := controller.KeyFunc(obj)
	if err != nil {
		return
	}
	// Observing the VMI is what satisfies an expectation set by whichever controller
	// updated it last.
	a.vmiExpectations.SetExpectations(key, 0, 0)
	a.queue.Add(key)
}

func (a *Arbiter) addDomainFunc(obj any) {
	key, err := controller.KeyFunc(obj)
	if err == nil {
		a.queue.Add(key)
	}
}

func (a *Arbiter) deleteDomainFunc(obj any) {
	domain, ok := obj.(*api.Domain)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			a.logger.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error("Failed to process delete notification")
			return
		}
		domain, ok = tombstone.Obj.(*api.Domain)
		if !ok {
			a.logger.Reason(fmt.Errorf("tombstone contained object that is not a domain %#v", obj)).Error("Failed to process delete notification")
			return
		}
	}
	a.logger.V(3).Object(domain).Info("Domain deleted")

	key, err := controller.KeyFunc(obj)
	if err == nil {
		a.queue.Add(key)
	}
}

func (a *Arbiter) updateDomainFunc(old, new any) {
	newDomain, ok := new.(*api.Domain)
	if !ok {
		return
	}
	if oldDomain, ok := old.(*api.Domain); ok {
		if oldDomain.Status.Status != newDomain.Status.Status || oldDomain.Status.Reason != newDomain.Status.Reason {
			a.logger.Object(newDomain).Infof("Domain is in state %s reason %s", newDomain.Status.Status, newDomain.Status.Reason)
		}
	}
	if newDomain.ObjectMeta.DeletionTimestamp != nil {
		a.logger.Object(newDomain).Info("Domain is marked for deletion")
	}

	key, err := controller.KeyFunc(new)
	if err == nil {
		a.queue.Add(key)
	}
}
