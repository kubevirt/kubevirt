package watch

import (
	"github.com/jeevatkm/go-model"
	"k8s.io/client-go/kubernetes"
	kubeapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/types"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	corev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

func scheduledVMPodSelector() kubeapi.ListOptions {
	fieldSelector := fields.ParseSelectorOrDie("status.phase=" + string(kubeapi.PodRunning))
	labelSelector, err := labels.Parse(corev1.AppLabel + " in (virt-launcher)")
	if err != nil {
		panic(err)
	}
	return kubeapi.ListOptions{FieldSelector: fieldSelector, LabelSelector: labelSelector}
}

func NewPodController(vmCache cache.Store, recorder record.EventRecorder, clientset *kubernetes.Clientset, restClient *rest.RESTClient, vmService services.VMService) (cache.Store, *kubecli.Controller) {

	selector := scheduledVMPodSelector()
	lw := kubecli.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", kubeapi.NamespaceDefault, selector.FieldSelector, selector.LabelSelector)
	return NewPodControllerWithListWatch(vmCache, recorder, lw, restClient, vmService, clientset)
}

func NewPodControllerWithListWatch(vmCache cache.Store, _ record.EventRecorder, lw cache.ListerWatcher, restClient *rest.RESTClient, vmService services.VMService, clientset *kubernetes.Clientset) (cache.Store, *kubecli.Controller) {

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	return kubecli.NewController(lw, queue, &v1.Pod{}, func(store cache.Store, queue workqueue.RateLimitingInterface) bool {
		key, quit := queue.Get()
		if quit {
			return false
		}
		defer queue.Done(key)

		// Fetch the latest Vm state from cache
		obj, exists, err := store.GetByKey(key.(string))

		if err != nil {
			queue.AddRateLimited(key)
			return true
		}

		if !exists {
			// Do nothing
			return true
		}
		pod := obj.(*v1.Pod)

		vmObj, exists, err := vmCache.GetByKey(kubeapi.NamespaceDefault + "/" + pod.GetLabels()[corev1.DomainLabel])
		if err != nil {
			queue.AddRateLimited(key)
			return true
		}
		if !exists {
			// Do nothing, the pod will timeout.
			return true
		}
		vm := vmObj.(*corev1.VM)
		if vm.GetObjectMeta().GetUID() != types.UID(pod.GetLabels()[corev1.UIDLabel]) {
			// Obviously the pod of an outdated VM object, do nothing
			return true
		}
		// This is basically a hack, so that virt-handler can completely focus on the VM object and does not have to care about pods
		if vm.Status.Phase == corev1.Scheduling {
			// deep copy the VM to allow manipulations
			vmCopy := corev1.VM{}
			model.Copy(&vmCopy, vm)

			vmCopy.Status.Phase = corev1.Pending
			// FIXME we store this in the metadata since field selctors are currently not working for TPRs
			if vmCopy.GetObjectMeta().GetLabels() == nil {
				vmCopy.ObjectMeta.Labels = map[string]string{}
			}
			vmCopy.ObjectMeta.Labels[corev1.NodeNameLabel] = pod.Spec.NodeName
			vmCopy.Status.NodeName = pod.Spec.NodeName
			// Update the VM
			logger := logging.DefaultLogger()
			if _, err := putVm(&vmCopy, restClient, queue); err != nil {
				logger.V(3).Info().Msg("Enqueuing VM again.")
				queue.AddRateLimited(key)
				return true
			}
			logger.Info().Msgf("VM successfully scheduled to %s.", vmCopy.Status.NodeName)
		} else if vm.Status.Phase == corev1.Running {
			logger := logging.DefaultLogger()
			obj, err := kubeapi.Scheme.Copy(vm)
			if err != nil {
				logger.Error().Reason(err).Msg("could not copy vm object")
				queue.AddRateLimited(key)
				return true
			}

			// Set target node on VM if necessary
			vmCopy := obj.(*corev1.VM)
			if vmCopy.Status.MigrationNodeName != pod.Spec.NodeName {
				vmCopy.Status.MigrationNodeName = pod.Spec.NodeName
				if vmCopy, err = putVm(vmCopy, restClient, queue); err != nil {
					logger.V(3).Info().Msg("Enqueuing VM again.")
					queue.AddRateLimited(key)
					return true
				}
			}

			// Let's check if the job already exists, it can already exist in case we could not update the VM object in a previous run
			if _, exists, err := vmService.GetMigrationJob(vmCopy); err != nil {
				logger.Error().Reason(err).Msg("Checking for an existing migration job failed.")
				queue.AddRateLimited(key)
				return true
			} else if !exists {
				// Job does not yet exist, create it.
				sourceNode, err := clientset.CoreV1().Nodes().Get(vmCopy.Status.NodeName, metav1.GetOptions{})
				if err != nil {
					logger.Error().Reason(err).Msgf("Fetching source node %s failed.", vmCopy.Status.NodeName)
					queue.AddRateLimited(key)
					return true
				}
				targetNode, err := clientset.CoreV1().Nodes().Get(vmCopy.Status.MigrationNodeName, metav1.GetOptions{})
				if err != nil {
					logger.Error().Reason(err).Msgf("Fetching target node %s failed.", vmCopy.Status.MigrationNodeName)
					queue.AddRateLimited(key)
					return true
				}
				if err := vmService.StartMigration(vmCopy, sourceNode, targetNode); err != nil {
					logger.Error().Reason(err).Msg("Starting the migration job failed.")
					queue.AddRateLimited(key)
					return true
				}
			}

			// Update VM phase after successfull job creation to migrating
			vmCopy.Status.Phase = corev1.Migrating
			if vmCopy, err = putVm(vmCopy, restClient, queue); err != nil {
				logger.V(3).Info().Msg("Enqueuing VM again.")
				queue.AddRateLimited(key)
				return true
			} else if vmCopy == nil {
				return true
			}
			logger.Info().Msgf("Scheduled VM migration to node %s.", vmCopy.Status.NodeName)
		}
		return true
	})
}

// synchronously put updated VM object to API server.
func putVm(vm *corev1.VM, restClient *rest.RESTClient, queue workqueue.RateLimitingInterface) (*corev1.VM, error) {
	logger := logging.DefaultLogger().Object(vm)
	obj, err := restClient.Put().Resource("vms").Body(vm).Name(vm.ObjectMeta.Name).Namespace(kubeapi.NamespaceDefault).Do().Get()
	if err != nil {
		logger.Error().Reason(err).Msg("Setting the VM state failed.")
		return nil, err
	}
	return obj.(*corev1.VM), nil
}
