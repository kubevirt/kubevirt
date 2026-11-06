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
// pass and hands to exactly one controller. Reading it in a single place is what
// keeps the three controllers from disagreeing about whether a migration is in
// flight: they no longer each resolve the key against their own cache at their own
// moment in time.
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

// Arbiter is the single entry point for VMI processing in virt-handler.
//
// virt-handler has three controllers that act on the same VMIs: the VM controller
// drives the lifecycle of a local guest, the migration-source controller drives the
// outgoing side of a migration and the migration-target controller the incoming
// side. They must never act on the same VMI at the same time.
//
// Previously each of them had its own workqueue, its own worker pool and its own
// "is a migration in flight?" predicate over its own informer cache, so exclusion
// was a convention that three independent goroutines could, and did, violate. The
// Arbiter replaces that with structural exclusion:
//
//   - one workqueue, which already guarantees that a key is handed to at most one
//     worker at a time;
//   - one read of the VMI and the domain per pass;
//   - one role decision (see migrationRole) derived from that single snapshot.
//
// Two controllers acting on a VMI concurrently is therefore not something the
// controllers have to avoid, it is something they cannot express.
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

	vmi, vmiExists, err := a.getVMI(key)
	if err != nil {
		return nil, err
	}
	state.vmi, state.vmiExists = vmi, vmiExists

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

func (a *Arbiter) addVMIFunc(obj interface{}) {
	a.enqueueVMI(obj)
}

func (a *Arbiter) deleteVMIFunc(obj interface{}) {
	a.enqueueVMI(obj)
}

func (a *Arbiter) updateVMIFunc(_, new interface{}) {
	a.enqueueVMI(new)
}

func (a *Arbiter) enqueueVMI(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err != nil {
		return
	}
	// Observing the VMI is what satisfies an expectation set by whichever controller
	// updated it last.
	a.vmiExpectations.SetExpectations(key, 0, 0)
	a.queue.Add(key)
}

func (a *Arbiter) addDomainFunc(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err == nil {
		a.queue.Add(key)
	}
}

func (a *Arbiter) deleteDomainFunc(obj interface{}) {
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

func (a *Arbiter) updateDomainFunc(old, new interface{}) {
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
