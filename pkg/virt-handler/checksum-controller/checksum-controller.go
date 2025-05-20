package checksum_controller

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/util/checksum"
)

const (
	controllerName            = "ChecksumController"
	AnnotationChecksum        = "integrity.virtualization.deckhouse.io/core-spec-checksum"
	AnnotationChecksumApplied = "integrity.virtualization.deckhouse.io/core-spec-checksum-applied"
)

type VMIChecksumGetter interface {
	GetAppliedVMIChecksum() (string, error)
}

func NewController(vmiSourceInformer cache.SharedIndexInformer, clientset kubecli.KubevirtClient) *Controller {
	queue := workqueue.NewRateLimitingQueueWithConfig(
		workqueue.DefaultControllerRateLimiter(),
		workqueue.RateLimitingQueueConfig{Name: controllerName})

	return &Controller{
		vmiSourceInformer: vmiSourceInformer,
		clientset:         clientset,
		queue:             queue,
		log:               log.DefaultLogger().With("controller", controllerName),
		objects:           make(map[types.NamespacedName]VMIControl),
	}
}

type Controller struct {
	vmiSourceInformer cache.SharedIndexInformer
	clientset         kubecli.KubevirtClient
	queue             workqueue.RateLimitingInterface

	log *log.FilteredLogger

	objects map[types.NamespacedName]VMIControl
	mu      sync.RWMutex
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	defer c.queue.ShutDown()
	c.log.Info("Starting checksum controller")

	go wait.Until(func() {
		for key := range c.objects {
			c.queue.Add(key)
		}
	}, time.Minute, stopCh)

	wait.Until(c.runWorker, time.Second, stopCh)
}

func (c *Controller) runWorker() {
	for c.Execute() {
	}
}

func (c *Controller) Execute() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	if err := c.execute(key.(types.NamespacedName)); err != nil {
		c.log.Reason(err).Infof("re-enqueuing VirtualMachineInstance %v", key)
		c.queue.AddRateLimited(key)
	} else {
		c.log.V(4).Infof("processed VirtualMachineInstance %v", key)
		c.queue.Forget(key)
	}
	return true
}

func (c *Controller) execute(key types.NamespacedName) error {
	vmi, exist, err := c.getVMIFromCache(key)
	if err != nil {
		return fmt.Errorf("could not get VirtualMachine instance %v: %w", key, err)
	}
	if !exist || !vmi.DeletionTimestamp.IsZero() {
		c.delete(key)
		return nil
	}
	control, found := c.get(key)
	if !found {
		return nil
	}

	sum, err := control.checksumGetter.GetAppliedVMIChecksum()
	if err != nil {
		return fmt.Errorf("could not get checksum for VirtualMachine instance %v: %w", key, err)
	}
	err = c.patchVMI(vmi, control.Checksum, sum)
	if err != nil {
		return fmt.Errorf("could not patch VirtualMachine instance %v: %w", key, err)
	}

	return nil
}

func (c *Controller) getVMIFromCache(key types.NamespacedName) (vmi *v1.VirtualMachineInstance, exists bool, err error) {
	obj, exists, err := c.vmiSourceInformer.GetStore().GetByKey(key.String())
	if err != nil {
		return nil, false, err
	}

	if exists {
		vmi = obj.(*v1.VirtualMachineInstance).DeepCopy()
	}
	return vmi, exists, nil
}

func (c *Controller) patchVMI(vmi *v1.VirtualMachineInstance, handlerSum, launcherSum string) error {
	patchset := patch.New()

	addPatch := func(anno, needValue string) {
		value, exist := vmi.Annotations[anno]
		path := fmt.Sprintf("/metadata/annotations/%s", EscapeJSONPointer(anno))

		if !exist {
			patchset.AddOption(patch.WithAdd(path, needValue))
		} else if value != needValue {
			patchset.AddOption(patch.WithReplace(path, needValue))
		}
	}

	addPatch(AnnotationChecksum, handlerSum)
	addPatch(AnnotationChecksumApplied, launcherSum)

	if patchset.IsEmpty() {
		return nil
	}

	patchBytes, err := patchset.GeneratePayload()
	if err != nil {
		return err
	}
	_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})

	return err
}

func (c *Controller) Set(control VMIControl) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.objects[control.NamespacedName] = control
}

func (c *Controller) delete(key types.NamespacedName) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.objects, key)
}

func (c *Controller) get(key types.NamespacedName) (VMIControl, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	control, ok := c.objects[key]
	return control, ok
}

func NewVMIControl(vmi *v1.VirtualMachineInstance, checksumGetter VMIChecksumGetter) (VMIControl, error) {
	if vmi == nil {
		return VMIControl{}, fmt.Errorf("vmi is nil")
	}
	sum, err := checksum.FromVMISpec(&vmi.Spec)
	if err != nil {
		return VMIControl{}, err
	}
	return VMIControl{
		NamespacedName: types.NamespacedName{
			Name:      vmi.Name,
			Namespace: vmi.Namespace,
		},
		UID:            vmi.UID,
		Checksum:       sum,
		checksumGetter: checksumGetter,
	}, nil
}

type VMIControl struct {
	NamespacedName types.NamespacedName
	UID            types.UID
	Checksum       string

	checksumGetter VMIChecksumGetter
}

func EscapeJSONPointer(path string) string {
	return strings.ReplaceAll(path, "/", "~1")
}
