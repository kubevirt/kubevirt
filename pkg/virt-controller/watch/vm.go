package watch

import (
	"fmt"
	"strings"

	"github.com/jeevatkm/go-model"
	"k8s.io/client-go/kubernetes"
	kubeapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	k8sv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/types"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"time"

	"k8s.io/client-go/pkg/util/wait"

	kubev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

func NewVMController(vmService services.VMService, recorder record.EventRecorder, restClient *rest.RESTClient, clientset *kubernetes.Clientset) *VMController {
	lw := cache.NewListWatchFromClient(restClient, "vms", kubeapi.NamespaceDefault, fields.Everything())
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	indexer, informer := cache.NewIndexerInformer(lw, &kubev1.VM{}, 0, kubecli.NewResourceEventHandlerFuncsForWorkqueue(queue), cache.Indexers{})
	return &VMController{
		restClient: restClient,
		vmService:  vmService,
		queue:      queue,
		store:      indexer,
		informer:   informer,
		recorder:   recorder,
		clientset:  clientset,
	}
}

type VMController struct {
	restClient *rest.RESTClient
	vmService  services.VMService
	clientset  *kubernetes.Clientset
	queue      workqueue.RateLimitingInterface
	store      cache.Store
	informer   cache.ControllerInterface
	recorder   record.EventRecorder
}

func (c *VMController) Run(threadiness int, stopCh chan struct{}) {
	defer kubecli.HandlePanic()
	defer c.queue.ShutDown()
	logging.DefaultLogger().Info().Msg("Starting controller.")

	// Start all informers/controllers and wait for the cache sync
	// TODO, change controllers to informers
	_, podController := NewPodController(c.store, nil, c.clientset, c.restClient, c.vmService)
	podController.StartInformer(stopCh)
	podController.WaitForSync(stopCh)
	go c.informer.Run(stopCh)

	// Wait for cache sync before we start the pod controller
	cache.WaitForCacheSync(stopCh, c.informer.HasSynced)

	// Start the actual work
	go podController.Run(threadiness, stopCh)
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	logging.DefaultLogger().Info().Msg("Stopping controller.")
}

func (c *VMController) runWorker() {
	for c.Execute() {
	}
}

func (c *VMController) Execute() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	if err := c.execute(key.(string)); err != nil {
		logging.DefaultLogger().Info().Reason(err).Msgf("reenqueuing VM %v", key)
		c.queue.AddRateLimited(key)
	} else {
		logging.DefaultLogger().Info().V(4).Msgf("processed VM %v", key)
		c.queue.Forget(key)
	}
	return true
}

func (c *VMController) execute(key string) error {

	// Fetch the latest Vm state from cache
	obj, exists, err := c.store.GetByKey(key)

	if err != nil {
		return err
	}

	// Retrieve the VM
	var vm *kubev1.VM
	if !exists {
		_, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return err
		}
		vm = kubev1.NewVMReferenceFromName(name)
	} else {
		vm = obj.(*kubev1.VM)
	}
	logger := logging.DefaultLogger().Object(vm)

	if !exists {
		// Delete VM Pods
		err := c.vmService.DeleteVMPod(vm)
		if err != nil {
			logger.Error().Reason(err).Msg("Deleting VM target Pod failed.")
			return err
		}
		logger.Info().Msg("Deleting VM target Pod succeeded.")
	} else if vm.Status.Phase == kubev1.VmPhaseUnset {
		// Schedule the VM
		vmCopy := kubev1.VM{}

		// Deep copy the object, so that we can safely manipulate it
		model.Copy(&vmCopy, vm)
		logger := logging.DefaultLogger().Object(&vmCopy)

		// Create a pod for the specified VM
		//Three cases where this can fail:
		// 1) VM pods exist from old definition // 2) VM pods exist from previous start attempt and updating the VM definition failed
		//    below
		// 3) Technical difficulties, we can't reach the apiserver
		// For case (1) this loop is not responsible. virt-handler or another loop is
		// responsible.
		// For case (2) we want to delete the VM first and then start over again.

		// TODO move defaulting to virt-api
		// TODO move constants to virt-handler
		if vmCopy.Spec.Domain == nil {
			spec := kubev1.NewMinimalDomainSpec(vmCopy.GetObjectMeta().GetName())
			vmCopy.Spec.Domain = spec
		}
		vmCopy.Spec.Domain.UUID = string(vmCopy.GetObjectMeta().GetUID())
		vmCopy.Spec.Domain.Devices.Emulator = "/usr/local/bin/qemu-x86_64"
		vmCopy.Spec.Domain.Name = vmCopy.GetObjectMeta().GetName()

		// TODO when we move this to virt-api, we have to block that they are set on POST or changed on PUT
		graphics := vmCopy.Spec.Domain.Devices.Graphics
		for i, _ := range graphics {
			if strings.ToLower(graphics[i].Type) == "spice" {
				graphics[i].Port = int32(4000) + int32(i)
				graphics[i].Listen = kubev1.Listen{
					Address: "0.0.0.0",
					Type:    "address",
				}

			}
		}

		// TODO get rid of these service calls
		if err := c.vmService.StartVMPod(&vmCopy); err != nil {
			logger.Error().Reason(err).Msg("Defining a target pod for the VM.")
			pl, err := c.vmService.GetRunningVMPods(&vmCopy)
			if err != nil {
				logger.Error().Reason(err).Msg("Getting all running Pods for the VM failed.")
				return err
			}
			for _, p := range pl.Items {
				if p.GetObjectMeta().GetLabels()["kubevirt.io/vmUID"] == string(vmCopy.GetObjectMeta().GetUID()) {
					// Pod from incomplete initialization detected, cleaning up
					logger.Error().Msgf("Found orphan pod with name '%s' for VM.", p.GetName())
					err = c.vmService.DeleteVMPod(&vmCopy)
					if err != nil {
						logger.Critical().Reason(err).Msgf("Deleting orphaned pod with name '%s' for VM failed.", p.GetName())
						return err
					}
				} else {
					// TODO virt-api should make sure this does not happen. For now don't ask and clean up.
					// Pod from old VM object detected,
					logger.Error().Msgf("Found orphan pod with name '%s' for deleted VM.", p.GetName())
					err = c.vmService.DeleteVMPod(&vmCopy)
					if err != nil {
						logger.Critical().Reason(err).Msgf("Deleting orphaned pod with name '%s' for VM failed.", p.GetName())
						return err
					}
				}
			}
			return err
		}
		// Mark the VM as "initialized". After the created Pod above is scheduled by
		// kubernetes, virt-handler can take over.
		//Three cases where this can fail:
		// 1) VM spec got deleted
		// 2) VM  spec got updated by the user
		// 3) Technical difficulties, we can't reach the apiserver
		// For (1) we don't want to retry, the pods will time out and fail. For (2) another
		// object got enqueued already. It will fail above until the created pods time out.
		// For (3) we want to enqueue again. If we don't do that the created pods will time out and we will
		// not get any updates
		vmCopy.Status.Phase = kubev1.Scheduling
		if err := c.restClient.Put().Resource("vms").Body(&vmCopy).Name(vmCopy.ObjectMeta.Name).Namespace(kubeapi.NamespaceDefault).Do().Error(); err != nil {
			logger.Error().Reason(err).Msg("Updating the VM state to 'Scheduling' failed.")
			if errors.IsNotFound(err) || errors.IsConflict(err) {
				// Nothing to do for us, VM got either deleted in the meantime or a newer version is enqueued already
				return nil
			}
			return err
		}
		logger.Info().Msg("Handing over the VM to the scheduler succeeded.")
	}
	return nil
}

func scheduledVMPodSelector() kubeapi.ListOptions {
	fieldSelectionQuery := fmt.Sprintf("status.phase=%s", string(kubeapi.PodRunning))
	fieldSelector := fields.ParseSelectorOrDie(fieldSelectionQuery)
	labelSelectorQuery := fmt.Sprintf("!%s, %s in (virt-launcher)", string(kubev1.MigrationLabel), kubev1.AppLabel)
	labelSelector, err := labels.Parse(labelSelectorQuery)
	if err != nil {
		panic(err)
	}
	return kubeapi.ListOptions{FieldSelector: fieldSelector, LabelSelector: labelSelector}
}

func NewPodController(vmCache cache.Store, recorder record.EventRecorder, clientset *kubernetes.Clientset, restClient *rest.RESTClient, vmService services.VMService) (cache.Store, *kubecli.Controller) {

	selector := scheduledVMPodSelector()
	lw := kubecli.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", kubeapi.NamespaceDefault, selector.FieldSelector, selector.LabelSelector)
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	return kubecli.NewController(lw, queue, &k8sv1.Pod{}, NewPodControllerDispatch(vmCache, restClient, vmService, clientset))
}

func NewPodControllerDispatch(vmCache cache.Store, restClient *rest.RESTClient, vmService services.VMService, clientset *kubernetes.Clientset) kubecli.ControllerDispatch {
	dispatch := podDispatch{
		vmCache:    vmCache,
		restClient: restClient,
		vmService:  vmService,
		clientset:  clientset,
	}
	return &dispatch
}

type podDispatch struct {
	vmCache    cache.Store
	restClient *rest.RESTClient
	vmService  services.VMService
	clientset  *kubernetes.Clientset
}

func (pd *podDispatch) Execute(podStore cache.Store, podQueue workqueue.RateLimitingInterface, key interface{}) {
	// Fetch the latest Vm state from cache
	obj, exists, err := podStore.GetByKey(key.(string))

	if err != nil {
		podQueue.AddRateLimited(key)
		return
	}

	if !exists {
		// Do nothing
		return
	}
	pod := obj.(*k8sv1.Pod)

	vmObj, exists, err := pd.vmCache.GetByKey(kubeapi.NamespaceDefault + "/" + pod.GetLabels()[kubev1.DomainLabel])
	if err != nil {
		podQueue.AddRateLimited(key)
		return
	}
	if !exists {
		// Do nothing, the pod will timeout.
		return
	}
	vm := vmObj.(*kubev1.VM)
	if vm.GetObjectMeta().GetUID() != types.UID(pod.GetLabels()[kubev1.VMUIDLabel]) {
		// Obviously the pod of an outdated VM object, do nothing
		return
	}
	if vm.Status.Phase == kubev1.Scheduling {
		// This is basically a hack, so that virt-handler can completely focus on the VM object and does not have to care about pods
		pd.handleScheduling(podQueue, key, vm, pod)
	}
	return
}

func (pd *podDispatch) handleScheduling(podQueue workqueue.RateLimitingInterface, key interface{}, vm *kubev1.VM, pod *k8sv1.Pod) {
	// deep copy the VM to allow manipulations
	vmCopy := kubev1.VM{}
	model.Copy(&vmCopy, vm)

	vmCopy.Status.Phase = kubev1.Pending
	// FIXME we store this in the metadata since field selctors are currently not working for TPRs
	if vmCopy.GetObjectMeta().GetLabels() == nil {
		vmCopy.ObjectMeta.Labels = map[string]string{}
	}
	vmCopy.ObjectMeta.Labels[kubev1.NodeNameLabel] = pod.Spec.NodeName
	vmCopy.Status.NodeName = pod.Spec.NodeName
	// Update the VM
	logger := logging.DefaultLogger()
	if _, err := pd.vmService.PutVm(&vmCopy); err != nil {
		logger.V(3).Info().Msg("Enqueuing VM again.")
		podQueue.AddRateLimited(key)
		return
	}
	logger.Info().Msgf("VM successfully scheduled to %s.", vmCopy.Status.NodeName)
}
