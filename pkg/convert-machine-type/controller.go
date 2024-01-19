package convertmachinetype

import (
	"context"
	"fmt"
	"path"
	"time"

	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/util/status"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var (
	Testing = false
)

type JobController struct {
	vmInformer      cache.SharedIndexInformer
	vmiInformer     cache.SharedIndexInformer
	virtClient      kubecli.KubevirtClient
	Queue           workqueue.RateLimitingInterface
	statusUpdater   *status.VMStatusUpdater
	exitJobChan     chan struct{}
	machineTypeGlob string
	restartRequired bool
}

func NewJobController(
	vmInformer, vmiInformer cache.SharedIndexInformer,
	virtClient kubecli.KubevirtClient,
	machineTypeGlob string,
	restartRequired bool,
) (*JobController, error) {
	c := &JobController{
		vmInformer:      vmInformer,
		vmiInformer:     vmiInformer,
		virtClient:      virtClient,
		Queue:           workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		statusUpdater:   status.NewVMStatusUpdater(virtClient),
		exitJobChan:     make(chan struct{}),
		machineTypeGlob: machineTypeGlob,
		restartRequired: restartRequired,
	}

	_, err := vmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.vmHandler,
		UpdateFunc: func(_, newObj interface{}) { c.vmHandler(newObj) },
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *JobController) exitJob() {
	vms := c.vmInformer.GetStore().List()

	for _, obj := range vms {
		vm := obj.(*v1.VirtualMachine)
		updated, err := c.isMachineTypeUpdated(vm)
		if err != nil {
			fmt.Println(err)
			return
		}
		running, err := isVMRunning(vm)
		if err != nil {
			fmt.Println(err)
			return
		}

		if !updated {
			return
		} else if vm.Status.MachineTypeRestartRequired {
			return
		} else if running {
			updated, err = c.isVMIUpdated(vm)
			if err != nil {
				fmt.Println(err)
				return
			}
			if !updated {
				return
			}
		}
	}

	close(c.exitJobChan)
}

func (c *JobController) run(stopCh <-chan struct{}) {
	defer c.Queue.ShutDown()
	informerStopCh := make(chan struct{})

	fmt.Println("Starting job controller")
	go c.vmInformer.Run(informerStopCh)
	go c.vmiInformer.Run(informerStopCh)

	if !cache.WaitForCacheSync(informerStopCh, c.vmInformer.HasSynced, c.vmiInformer.HasSynced) {
		fmt.Println("Timed out waiting for caches to sync")
		return
	}

	vmKeys := c.vmInformer.GetStore().ListKeys()
	for _, k := range vmKeys {
		c.Queue.Add(k)
	}

	wait.Until(c.runWorker, time.Second, stopCh)
}

func (c *JobController) runWorker() {
	for c.Execute() {
	}
}

func (c *JobController) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		close(c.exitJobChan)
		return false
	}

	defer c.Queue.Done(key)

	if err := c.execute(key.(string)); err != nil {
		c.Queue.AddRateLimited(key)
	} else {
		c.Queue.Forget(key)
		c.exitJob()
	}

	return true
}

func (c *JobController) vmHandler(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		c.Queue.Add(key)
	}
}

func (c *JobController) execute(key string) error {
	obj, exists, err := c.vmInformer.GetStore().GetByKey(key)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("VM does not exist")
	}

	vm := obj.(*v1.VirtualMachine)

	// check if VM is running
	isRunning, err := isVMRunning(vm)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// check if VM machine type was updated
	updated, err := c.isMachineTypeUpdated(vm)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// update VMs that require update
	if !updated {
		err := c.patchMachineType(vm)
		if err != nil {
			return err
		}

		// don't need to do anything else to stopped VMs
		if !isRunning {
			return nil
		}

		// if force restart flag is set, restart running VMs immediately
		// don't apply warning label to VMs being restarted
		if c.restartRequired {
			return c.virtClient.VirtualMachine(vm.Namespace).Restart(context.Background(), vm.Name, &v1.RestartOptions{})
		}

		// adding the warning label to the running VMs to indicate to the user
		// they must manually be restarted
		patchString := `[{ "op": "add", "path": "/status/machineTypeRestartRequired", "value": true }]`
		err = c.statusUpdater.PatchStatus(vm, types.JSONPatchType, []byte(patchString), &metav1.PatchOptions{})
		if err != nil {
			return err
		}
	}

	if isRunning {
		// check if VMI machine type has been updated
		updated, err = c.isVMIUpdated(vm)
		if err != nil {
			fmt.Println(err)
			return err
		}

		if !updated {
			fmt.Println("vmi machine type has not been updated")
			return nil
		}
	}

	// mark MachineTypeRestartRequired as false
	patchString := `[{ "op": "replace", "path": "/status/machineTypeRestartRequired", "value": false }]`
	err = c.statusUpdater.PatchStatus(vm, types.JSONPatchType, []byte(patchString), &k8sv1.PatchOptions{})
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func isVMRunning(vm *v1.VirtualMachine) (bool, error) {
	runStrategy, err := vm.RunStrategy()
	if err != nil {
		return false, err
	}

	if runStrategy == v1.RunStrategyAlways {
		return true, nil
	}

	return false, nil
}

func (c *JobController) isVMIUpdated(vm *v1.VirtualMachine) (bool, error) {
	// get VMI from cache
	vmKey, err := cache.MetaNamespaceKeyFunc(vm)
	if err != nil {
		return false, err
	}

	obj, exists, err := c.vmiInformer.GetStore().GetByKey(vmKey)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, fmt.Errorf("VMI does not exist")
	}

	vmi := obj.(*v1.VirtualMachineInstance)

	specMachine := vmi.Spec.Domain.Machine
	statusMachine := vmi.Status.Machine
	if specMachine == nil || statusMachine == nil {
		return false, fmt.Errorf("vmi machine type is not set properly")
	}
	matchesGlob, err := c.matchMachineType(statusMachine.Type)
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	return specMachine.Type == virtconfig.DefaultAMD64MachineType && !matchesGlob, nil
}

func (c *JobController) matchMachineType(machineType string) (bool, error) {
	matchMachineType, err := path.Match(c.machineTypeGlob, machineType)
	if !matchMachineType || err != nil {
		return false, err
	}

	return true, nil
}

func (c *JobController) patchMachineType(vm *v1.VirtualMachine) error {
	// removing the machine type field from the VM spec reverts it to
	// the default machine type of the VM's arch
	updateMachineType := `[{"op": "remove", "path": "/spec/template/spec/domain/machine"}]`

	_, err := c.virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, []byte(updateMachineType), &metav1.PatchOptions{})
	return err
}

func (c *JobController) isMachineTypeUpdated(vm *v1.VirtualMachine) (bool, error) {
	machine := vm.Spec.Template.Spec.Domain.Machine
	matchesGlob := false
	var err error

	// when running unit tests, updating the machine type
	// does not update it to the aliased machine type when
	// setting it to nil. This is to account for that for now
	if machine == nil && Testing {
		return true, nil
	}

	if machine.Type == virtconfig.DefaultAMD64MachineType {
		return true, nil
	}

	matchesGlob, err = c.matchMachineType(machine.Type)
	return !matchesGlob, err
}
