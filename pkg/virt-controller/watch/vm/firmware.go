package vm

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/clock"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

type FirmwareController struct {
	clientset kubecli.KubevirtClient
	queue     workqueue.TypedRateLimitingInterface[string]
	vmIndexer cache.Indexer
	recorder  record.EventRecorder
}

func NewFirmwareController(vmInformer cache.SharedIndexInformer, clientset kubecli.KubevirtClient, recorder record.EventRecorder) (*FirmwareController, error) {
	c := &FirmwareController{
		clientset: clientset,
		queue: workqueue.NewTypedRateLimitingQueueWithConfig[string](
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{
				Name:  "firmware-uuid-controller",
				Clock: clock.RealClock{},
			},
		),
		vmIndexer: vmInformer.GetIndexer(),
		recorder:  recorder,
	}

	_, err := vmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.enqueueVM,
		UpdateFunc: func(_, new any) { c.enqueueVM(new) },
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *FirmwareController) enqueueVM(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Log.Errorf("Failed to get key for VM: %v", err)
		return
	}
	c.queue.Add(key)
}

func (c *FirmwareController) Run(threadiness int, stopCh <-chan struct{}) {
	defer c.queue.ShutDown()
	log.Log.Info("Starting FirmwareController")

	for range threadiness {
		go c.runWorker()
	}

	<-stopCh
	log.Log.Info("Stopping FirmwareController")
}

func (c *FirmwareController) runWorker() {
	for c.Execute() {
	}
}

func (c *FirmwareController) Execute() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	if err := c.execute(key); err != nil {
		log.Log.Reason(err).Infof("re-enqueuing VirtualMachine %v", key)
		c.queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachine %v", key)
		c.queue.Forget(key)
	}
	return true
}

func (c *FirmwareController) execute(key string) error {
	obj, exists, err := c.vmIndexer.GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	vm := obj.(*virtv1.VirtualMachine)
	vmCopy := vm.DeepCopy()

	patchOps, err := c.generateFirmwarePatch(vmCopy)
	if err != nil {
		return err
	}

	if patchOps.IsEmpty() {
		return nil
	}

	patchBytes, err := patchOps.GeneratePayload()
	if err != nil {
		return fmt.Errorf("failed to generate patch payload: %w", err)
	}

	_, err = c.clientset.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		if errors.IsConflict(err) {
			log.Log.Warningf("Conflict while patching VM firmware UUID, retrying: %s", vm.Name)
			c.queue.AddRateLimited(key)
			return nil
		}
		return fmt.Errorf("failed to patch VM firmware UUID: %w", err)
	}

	log.Log.Infof("Patched firmware UUID for VM %s", vm.Name)
	return nil
}

// generateFirmwarePatch handles the firmware UUID patching logic and returns patch operations
func (c *FirmwareController) generateFirmwarePatch(vm *virtv1.VirtualMachine) (patch.PatchSet, error) {

	const (
		firmwarePatchPath     = "/spec/template/spec/domain/firmware"
		firmwareUUIDPatchPath = "/spec/template/spec/domain/firmware/uuid"
	)

	var patchOps patch.PatchSet
	legacyUUID := CalculateLegacyUUID(vm.Name)

	firmware := vm.Spec.Template.Spec.Domain.Firmware
	switch {
	case firmware == nil:
		vm.Spec.Template.Spec.Domain.Firmware = &virtv1.Firmware{
			UUID: legacyUUID,
		}
		patchOps.AddOption(patch.WithAdd(
			firmwarePatchPath,
			vm.Spec.Template.Spec.Domain.Firmware,
		))

	case firmware.UUID == "":
		vm.Spec.Template.Spec.Domain.Firmware.UUID = legacyUUID
		patchOps.AddOption(patch.WithTest(firmwareUUIDPatchPath, ""))
		patchOps.AddOption(patch.WithReplace(
			firmwarePatchPath,
			vm.Spec.Template.Spec.Domain.Firmware,
		))
	}

	return patchOps, nil
}

const magicUUID = "6a1a24a1-4061-4607-8bf4-a3963d0c5895"

var firmwareUUIDns = uuid.MustParse(magicUUID
var firmwareUUIDns = uuid.MustParse(magicUUID)

func CalculateLegacyUUID(name string) types.UID {
	return types.UID(uuid.NewSHA1(firmwareUUIDns, []byte(name)).String())
}
