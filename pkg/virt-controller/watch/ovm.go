package watch

import (
	"fmt"
	"log"
	"reflect"
	"time"

	machineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	machinerylabels "k8s.io/apimachinery/pkg/labels"
	runtime "k8s.io/apimachinery/pkg/util/runtime"
	wait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	workqueue "k8s.io/client-go/util/workqueue"

	virtapiv1 "kubevirt.io/kubevirt/pkg/api/v1"
	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	virtcli "kubevirt.io/kubevirt/pkg/kubecli"
)

const controllerName = "offlinevm-controller"

// OVMController watches the offline virtual machines and keeps them in
// sync with the Virtual Machine. The Virtual Machine is created by this
// controller if needed
type OVMController struct {
	// client which connects to cluster with all kubevirt added
	client virtcli.KubevirtClient

	ovmInformer cache.SharedIndexInformer
	vmInformer  cache.SharedIndexInformer

	// workque is used to buffer and limit work ammount. We do not want to be
	// too fast
	workqueue workqueue.RateLimitingInterface

	// expectations are used to controll creation behaviour
	// it helps us limit creation to only one vm and deleting
	// only one mv
	vmExpectations *virtcontroller.UIDTrackingControllerExpectations

	// recorder and broadcaster handle events from and to cluster
	eventRecorder record.EventRecorder
}

// ******************
//   event handlers
// ******************

// NewOVMController creates instance of OVM controller and setups the required stuff
func NewOVMController(
	clientset virtcli.KubevirtClient,
	recorder record.EventRecorder,
	ovmInformer cache.SharedIndexInformer,
	vmInformer cache.SharedIndexInformer,
) *OVMController {

	controller := &OVMController{
		client:         clientset,
		ovmInformer:    ovmInformer,
		vmInformer:     vmInformer,
		workqueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ovms"),
		vmExpectations: virtcontroller.NewUIDTrackingControllerExpectations(virtcontroller.NewControllerExpectations()),
		eventRecorder:  recorder,
	}

	// Register the listener for OVM kind events in the cluster
	ovmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			log.Print("EVENT ADD OVM")
			controller.addToQueue(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			log.Print("EVENT UPDATE OVM")
			// TODO: Add check for changes. Update only when object really
			//       changes. Kubernetes fires update event more than only
			//       for real update
			//       Question: How the change is defined?
			controller.addToQueue(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			log.Print("EVENT DELETE OVM")
			controller.addToQueue(obj)
		},
	})

	// Register informer also for VM since this controller reflects the status
	// of VM in the OVM status
	vmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			log.Print("EVENT ADD VM")
			controller.addVMHandler(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			log.Print("EVENT UPDATE VM")

		},
		DeleteFunc: func(obj interface{}) {
			log.Print("EVENT DELETE VM")
			controller.deleteVMHandler(obj)
		},
	})

	return controller
}

// addVMHandler uses controllerRef to find the original OfflineVirtualMachine
// and enques that OVM for a change
// it is called when VM ADD event is raised
func (c *OVMController) addVMHandler(obj interface{}) {
	vm := obj.(*virtapiv1.VirtualMachine)

	if vm.DeletionTimestamp != nil {
		// the vm is being deleted lets do a proper delete
		c.deleteVMHandler(obj)
		return
	}

	if controllerRef := machineryv1.GetControllerOf(vm); controllerRef != nil {
		// VM has reference inside, lookup its controller and enque it
		ovm := c.resolveControllerRef(vm.ObjectMeta.Namespace, controllerRef)
		if ovm == nil {
			runtime.HandleError(fmt.Errorf("No OfflineVirtualMachine found for VM %s", vm.Name))
		}

		key, err := cache.MetaNamespaceKeyFunc(ovm)
		if err != nil {
			runtime.HandleError(err)
			return
		}

		// controller should be waiting for creation event, signal it
		c.vmExpectations.CreationObserved(key)

		log.Printf("Adding the key for ovm: %s", key)
		c.workqueue.AddRateLimited(key)
	}

	// its an orphan, notify OfflineVirtualMachine that matches this
	// VirtualMachine if it wants to adopt it
}

// deleteVMHandler is called for any raised VM DELETE event
// it looks up the VM reference and link it to its owning
// OVM.
func (c *OVMController) deleteVMHandler(obj interface{}) {
	vm, ok := obj.(*virtapiv1.VirtualMachine)

	log.Printf("Handling deletion of VM: %s", vm.Name)

	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}

		vm, ok = tombstone.Obj.(*virtapiv1.VirtualMachine)
		if !ok {
			return
		}
	}

	if controllerRef := machineryv1.GetControllerOf(vm); controllerRef != nil {
		log.Printf("It has ref: %s", controllerRef.Name)

		ovm := c.resolveControllerRef(vm.Namespace, controllerRef)
		if ovm == nil {
			log.Print("Cannot locate the ovm")
			return
		}

		log.Printf("Found ovm: %s", ovm.Name)

		key, err := cache.MetaNamespaceKeyFunc(ovm)
		if err != nil {
			runtime.HandleError(err)
			return
		}

		// vm has been deleted so lets report it
		c.vmExpectations.DeletionObserved(key, virtcontroller.VirtualMachineKey(vm))

		// notify this controller
		c.workqueue.AddRateLimited(key)
	}

	return
}

// updateVMHandler is called every time the update event is raised
// it checks what happend and eventually notify this controller
func (c *OVMController) updateVMHandler(oldObj, newObj interface{}) {
	newVM := newObj.(*virtapiv1.VirtualMachine)
	oldVM := oldObj.(*virtapiv1.VirtualMachine)
	if newVM.ResourceVersion != oldVM.ResourceVersion {
		// every version present in the system gets updated
		// skip for non matching version
		return
	}

	if newVM.DeletionTimestamp != nil {
		// the VM is being deleted, no need to do updates
		c.deleteVMHandler(newObj)
		return
	}

	newControllerRef := machineryv1.GetControllerOf(newVM)
	oldControllerRef := machineryv1.GetControllerOf(oldVM)
	controllerRefChanged := !reflect.DeepEqual(newControllerRef, oldControllerRef)
	if controllerRefChanged && oldControllerRef != nil {
		// changes occured, notify the old controller. VM changed the owner
		// and old controller have to know he no longer owns this vm
		if ovm := c.resolveControllerRef(oldVM.Namespace, oldControllerRef); ovm != nil {
			c.addToQueue(ovm)
		}
	}

	if newControllerRef != nil {
		// the new controller has to be updated any way if exists
		if ovm := c.resolveControllerRef(newVM.Namespace, newControllerRef); ovm != nil {
			c.addToQueue(newVM)
			return
		}
	}

	// TODO handle orphans
}

// addToQueue takes an object reported by handler, lookup its namespace and key
// test wheter it is not deleted - if so ignore and if not add it for processing
func (c *OVMController) addToQueue(obj interface{}) {
	var (
		key string
		err error
	)

	if key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj); err != nil {
		// The error happened but it is not critical.
		// Let kubernetes handle and continue.
		runtime.HandleError(err)
		return
	}

	c.workqueue.AddRateLimited(key)
}

// ********************************
//       controller runtime
// ********************************

// Run starts the controller. It setups the caches and waits for them to sync.
// It spaws all workers and blocks until the stopCh closes.
func (c *OVMController) Run(workers int, stopCh <-chan struct{}) error {
	// Not defering the CrashHandling. It should be safe for the controller
	// to crash, be restarted and continue work
	//defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	log.Print("Starting the OVM controller")

	log.Print("Syncing cache")
	if ok := cache.WaitForCacheSync(stopCh, c.ovmInformer.HasSynced, c.vmInformer.HasSynced); !ok {
		return fmt.Errorf("Caches cannot be synced")
	}

	log.Print("Spawning OVM controller workers")
	for i := 0; i < workers; i++ {
		go wait.Until(c.worker, time.Second, stopCh)
	}

	log.Print("Workers started")
	<-stopCh
	log.Print("Stopped OVM controller")

	return nil
}

// worker is an infinite loop. Takes one item from queue and process it.
// Terminates when the queue is marker for shutdown.
func (c *OVMController) worker() {
	for c.processNextItem() {
	}
}

// processNextItem takes one item from the workqueue and process it.
// it return false when the queue is marker for shutdown
func (c *OVMController) processNextItem() bool {

	obj, quit := c.workqueue.Get()
	if quit {
		return false
	}
	defer c.workqueue.Done(obj)

	var (
		key string
		ok  bool
	)

	if key, ok = obj.(string); !ok {
		// no string key, so forget it and handle the error in kubernetes
		c.workqueue.Forget(obj)
		runtime.HandleError(fmt.Errorf("Expected string key, but got %#v", obj))
	}

	if err := c.processItem(key); err != nil {
		runtime.HandleError(err)
	}

	log.Printf("Processing done for key: %s", key)
	c.workqueue.Forget(obj)

	return true
}

// processItem takes the key, loads the item from cluster.
func (c *OVMController) processItem(key string) error {
	ovm, err := c.getOvm(key)
	if err != nil {
		return err
	}

	// find matching VM if any in the same namespace as the ovm
	// currently the created VM have to be in the same namespace
	vms, err := c.listVMsFromNamespace(ovm.Namespace)

	// filter only to ovm owned vm if any
	myVms := filterOwnedVms(vms, ovm)
	var myVM *virtapiv1.VirtualMachine

	if len(myVms) > 1 {
		return fmt.Errorf("OfflineVirtualMachine owns more than one VM: %d", len(myVms))
	} else if len(myVms) == 1 {
		myVM = myVms[0]
		log.Printf("Found new VM %s", myVM.Name)
	}

	doSomething := c.vmExpectations.SatisfiedExpectations(key)
	log.Printf("This controller is scheduled to do some action for %s", key)

	// There is a VirtualMachine update the OfflineVirtualMachine Status to
	// Reflects the VirtualMachine state
	c.updateStatus(ovm, myVM)

	if doSomething && ovm.ObjectMeta.DeletionTimestamp == nil && ovm.Spec.Running == true && myVM == nil {
		c.createVM(ovm)
	}

	if doSomething && ovm.ObjectMeta.DeletionTimestamp == nil && ovm.Spec.Running == false && myVM != nil {
		c.deleteVM(ovm, myVM)
	}

	// TODO when updating only status, update the status as subresource
	c.client.OfflineVirtualMachine(ovm.ObjectMeta.Namespace).Update(ovm)

	return nil
}

// *************************************
//          Utility functions
// *************************************

// getOvm returns the OfflineVirtualMachine object from the cache store
// To get the freshest instance, use the restclient
func (c *OVMController) getOvm(key string) (*virtapiv1.OfflineVirtualMachine, error) {
	obj, exists, err := c.ovmInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		// the controller has been deleted in the meantime
		// the operations for recreated controller would be invalind
		// therefore delete all of the expetations and give new
		// controller a clean slate
		c.vmExpectations.DeleteExpectations(key)
		return nil, fmt.Errorf("Key: %s not found", key)
	}

	// Do the deepcopy, otherwise the controller messes with the cache and
	// results in undefined behaviour
	ovm := obj.(*virtapiv1.OfflineVirtualMachine).DeepCopy()

	return ovm, nil
}

// listOVMsFromNamespace lookup in the cluster in the given namespace and return all OfflineVirtualMachines
// that matches the specified selector
func (c *OVMController) listOVMsFromNamespace(namespace string) []*virtapiv1.OfflineVirtualMachine {
	objs, err := c.ovmInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		// error happened but this is not the place to handle it
		// delegate handling to runtime
		runtime.HandleError(err)
		return nil
	}

	ovms := []*virtapiv1.OfflineVirtualMachine{}
	for _, obj := range objs {
		ovms = append(ovms, obj.(*virtapiv1.OfflineVirtualMachine))
	}

	return ovms
}

// findMatchingOvms return every OfflineVirtualMachine from the cluster
// in the given namespace that matches the selector
func (c *OVMController) findMatchingOvms(namespace string, selector *machineryv1.LabelSelector) []*virtapiv1.OfflineVirtualMachine {
	ovms := c.listOVMsFromNamespace(namespace)
	if ovms == nil {
		// nothing found, no need to do antoher check
		return nil
	}

	foundOVMs := []*virtapiv1.OfflineVirtualMachine{}
	for _, ovm := range ovms {
		selector, err := machineryv1.LabelSelectorAsSelector(selector)
		if err != nil {
			log.Printf("Failed to parse the label selector. %s", err.Error())
			continue
		}

		if selector.Matches(machinerylabels.Set(ovm.ObjectMeta.Labels)) {
			foundOVMs = append(foundOVMs, ovm)
		}

	}
	return foundOVMs
}

// updateStatus updates status fields of the OfflineVirtualMachine according to VirtualMachine state
// the update is done in place
func (c *OVMController) updateStatus(offlinevm *virtapiv1.OfflineVirtualMachine, vm *virtapiv1.VirtualMachine) error {
	if vm == nil {
		// No VM, only quick update
		offlinevm.Status.VirtualMachineName = ""
		offlinevm.Status.Ready = false
		offlinevm.Status.Running = false

		return nil
	}

	offlinevm.Status.VirtualMachineName = vm.ObjectMeta.Name
	offlinevm.Status.Ready = vm.IsReady()
	offlinevm.Status.Running = vm.IsRunning()

	return nil
}

// createVM takes the OfflineVirtualMachine, builds the VM from the Specification
// and uses RESTclient to create new resource in the cluster
func (c *OVMController) createVM(offlinevm *virtapiv1.OfflineVirtualMachine) error {
	ovmKey, err := virtcontroller.KeyFunc(offlinevm)
	if err != nil {
		return err
	}

	// the offlineVm holds every information needed to create new VM
	// it holds the metadata and the full spec
	// put it together and create VM
	vm := buildVM(offlinevm)

	// before vm is created, increment the counter for exptected result
	c.vmExpectations.ExpectCreations(ovmKey, 1)

	// TODO handle accidents during vm creation, expectation manager
	vm, err = c.client.VM(offlinevm.Namespace).Create(vm)
	if err != nil {
		// since the creation process crashed, decrease the counter
		// this controller is no longer expecting to see the creation of new
		// VirtualMachine
		c.vmExpectations.CreationObserved(ovmKey)
		return err
	}

	return nil
}

// deleteVM takes the VM and deletes it from the cluster
func (c *OVMController) deleteVM(ovm *virtapiv1.OfflineVirtualMachine, vm *virtapiv1.VirtualMachine) error {
	ovmKey, err := virtcontroller.KeyFunc(ovm)
	if err != nil {
		return err
	}

	// increment the counter for expected results, also store refrence for
	// what this controller expect to see being deleted
	c.vmExpectations.ExpectDeletions(ovmKey, []string{virtcontroller.VirtualMachineKey(vm)})

	err = c.client.VM(vm.Namespace).Delete(vm.Name, &machineryv1.DeleteOptions{})
	if err != nil {
		// error happened, remove expectation this controller no longer
		// expectiong to see the deletion
		c.vmExpectations.DeletionObserved(ovmKey, virtcontroller.VirtualMachineKey(vm))

		return err
	}

	// everything went well
	return nil
}

// listVMsFromNamespace takes a namespace and returns all VMs from the VM cache which run in this namespace
func (c *OVMController) listVMsFromNamespace(namespace string) ([]*virtapiv1.VirtualMachine, error) {
	// TODO figure out a way to move this to shared space. This code is duplicated
	// from kubevirt.io/kubevirt/pkg/virt-controller/watch/replicaset.go
	// More parts of kubevirt will require this functionality
	objs, err := c.vmInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	vms := []*virtapiv1.VirtualMachine{}
	for _, obj := range objs {
		vms = append(vms, obj.(*virtapiv1.VirtualMachine))
	}
	return vms, nil
}

// resolveControllerRef lookup the controller OfflineVirtualMachine for the VirtualMachine
// the lookup is done from the cache. It can happen that the cache holds deleted object
func (c *OVMController) resolveControllerRef(namespace string, controllerRef *machineryv1.OwnerReference) *virtapiv1.OfflineVirtualMachine {
	if controllerRef.Kind != virtapiv1.OfflineVirtualMachineGroupVersionKind.Kind {
		log.Printf("The controllRef kind does not match: %s != %s", controllerRef.Kind, virtapiv1.OfflineVirtualMachineGroupVersionKind.Kind)
		return nil
	}

	obj, exists, err := c.ovmInformer.GetStore().GetByKey(namespace + "/" + controllerRef.Name)
	if !exists {
		log.Printf("The OVM does not exist: %s/%s", namespace, controllerRef.Name)
		return nil
	}

	if err != nil {
		log.Printf("Error when retrieving the ovm: %s", err.Error())
		return nil
	}

	ovm := obj.(*virtapiv1.OfflineVirtualMachine)
	if ovm.ObjectMeta.UID != controllerRef.UID {
		return nil
	}

	return ovm
}

// buildVM exports the VirtualMachine spec from OfflineVirtualMachine and use it
// to build new VirtualMachine object
func buildVM(offlinevm *virtapiv1.OfflineVirtualMachine) *virtapiv1.VirtualMachine {
	vm := virtapiv1.NewVMReferenceFromNameWithNS(offlinevm.ObjectMeta.Namespace, "")

	vm.ObjectMeta = offlinevm.Spec.Template.ObjectMeta
	vm.ObjectMeta.Name = ""
	vm.ObjectMeta.GenerateName = offlinevm.ObjectMeta.Name
	vm.ObjectMeta.Labels = offlinevm.Spec.Template.ObjectMeta.Labels

	vm.Spec = offlinevm.Spec.Template.Spec

	ovk := virtapiv1.OfflineVirtualMachineGroupVersionKind
	t := true
	// OwnerReferences are used to locate the object of origin
	owr := machineryv1.OwnerReference{
		APIVersion: ovk.GroupVersion().String(),
		Kind:       ovk.Kind,
		Name:       offlinevm.ObjectMeta.Name,
		UID:        offlinevm.ObjectMeta.UID,
		Controller: &t,
	}
	vm.ObjectMeta.OwnerReferences = []machineryv1.OwnerReference{owr}

	return vm
}

// filterOwnedVms takes a list of vms and offlinevm and filter out the vms not owned by the offlinevm
func filterOwnedVms(vms []*virtapiv1.VirtualMachine, ovm *virtapiv1.OfflineVirtualMachine) []*virtapiv1.VirtualMachine {
	return filter(vms, func(vm *virtapiv1.VirtualMachine) bool {
		vmControllerRef := machineryv1.GetControllerOf(vm)
		return vmControllerRef.UID == ovm.ObjectMeta.UID
	})
}
