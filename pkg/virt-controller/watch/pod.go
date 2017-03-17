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
	return kubecli.NewController(lw, queue, &v1.Pod{}, NewPodControllerFunc(vmCache, restClient, vmService, clientset))
}

func NewPodControllerFunc(vmCache cache.Store, restClient *rest.RESTClient, vmService services.VMService, clientset *kubernetes.Clientset) kubecli.ControllerFunc {

	return func(store cache.Store, queue workqueue.RateLimitingInterface, key interface{}) bool {
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
		if vm.GetObjectMeta().GetUID() != types.UID(pod.GetLabels()[corev1.VMUIDLabel]) {
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
		} else if _, isMigrationPod := pod.Labels[corev1.MigrationLabel]; vm.Status.Phase == corev1.Running && isMigrationPod {
			logger := logging.DefaultLogger()

			// Get associated migration
			obj, err := restClient.Get().Resource("migrations").Namespace(v1.NamespaceDefault).Name(pod.Labels[corev1.MigrationLabel]).Do().Get()
			if err != nil {
				logger.Error().Reason(err).Msgf("Fetching migration %s failed.", pod.Labels[corev1.MigrationLabel])
				queue.AddRateLimited(key)
				return true
			}
			migration := obj.(*corev1.Migration)
			if migration.Status.Phase == corev1.MigrationUnknown {
				logger.Info().Msg("migration not yet in right state, backing off")
				queue.AddRateLimited(key)
				return true
			}

			obj, err = kubeapi.Scheme.Copy(vm)
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
			if _, exists, err := vmService.GetMigrationJob(migration); err != nil {
				logger.Error().Reason(err).Msg("Checking for an existing migration job failed.")
				queue.AddRateLimited(key)
				return true
			} else if !exists {
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

				if err := vmService.StartMigration(migration, vmCopy, sourceNode, targetNode); err != nil {
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
	}
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
