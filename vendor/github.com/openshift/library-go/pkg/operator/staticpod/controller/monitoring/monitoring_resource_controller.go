package monitoring

import (
	"fmt"
	"path/filepath"
	"time"

	"k8s.io/klog"

	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	rbaclisterv1 "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/openshift/library-go/pkg/assets"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/management"
	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	"github.com/openshift/library-go/pkg/operator/staticpod/controller/monitoring/bindata"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
)

const (
	operatorStatusMonitoringResourceControllerDegraded = "MonitoringResourceControllerDegraded"
	controllerWorkQueueKey                             = "key"
	manifestDir                                        = "pkg/operator/staticpod/controller/monitoring"
)

var syntheticRequeueError = fmt.Errorf("synthetic requeue request")

type MonitoringResourceController struct {
	targetNamespace    string
	serviceMonitorName string

	clusterRoleBindingLister rbaclisterv1.ClusterRoleBindingLister
	kubeClient               kubernetes.Interface
	dynamicClient            dynamic.Interface
	operatorClient           v1helpers.StaticPodOperatorClient

	cachesToSync  []cache.InformerSynced
	queue         workqueue.RateLimitingInterface
	eventRecorder events.Recorder
}

// NewMonitoringResourceController creates a new backing resource controller.
func NewMonitoringResourceController(
	targetNamespace string,
	serviceMonitorName string,
	operatorClient v1helpers.StaticPodOperatorClient,
	kubeInformersForTargetNamespace informers.SharedInformerFactory,
	kubeClient kubernetes.Interface,
	dynamicClient dynamic.Interface,
	eventRecorder events.Recorder,
) *MonitoringResourceController {
	c := &MonitoringResourceController{
		targetNamespace:    targetNamespace,
		operatorClient:     operatorClient,
		eventRecorder:      eventRecorder.WithComponentSuffix("monitoring-resource-controller"),
		serviceMonitorName: serviceMonitorName,

		clusterRoleBindingLister: kubeInformersForTargetNamespace.Rbac().V1().ClusterRoleBindings().Lister(),
		cachesToSync: []cache.InformerSynced{
			kubeInformersForTargetNamespace.Core().V1().ServiceAccounts().Informer().HasSynced,
			operatorClient.Informer().HasSynced,
		},

		queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "MonitoringResourceController"),
		kubeClient:    kubeClient,
		dynamicClient: dynamicClient,
	}

	operatorClient.Informer().AddEventHandler(c.eventHandler())
	// TODO: We need a dynamic informer here to observe changes to ServiceMonitor resource.
	kubeInformersForTargetNamespace.Rbac().V1().ClusterRoleBindings().Informer().AddEventHandler(c.eventHandler())

	c.cachesToSync = append(c.cachesToSync, operatorClient.Informer().HasSynced)
	c.cachesToSync = append(c.cachesToSync, kubeInformersForTargetNamespace.Rbac().V1().ClusterRoleBindings().Informer().HasSynced)

	return c
}

func (c MonitoringResourceController) mustTemplateAsset(name string) ([]byte, error) {
	config := struct {
		TargetNamespace string
	}{
		TargetNamespace: c.targetNamespace,
	}
	return assets.MustCreateAssetFromTemplate(name, bindata.MustAsset(filepath.Join(manifestDir, name)), config).Data, nil
}

func (c MonitoringResourceController) sync() error {
	operatorSpec, _, _, err := c.operatorClient.GetStaticPodOperatorState()
	if err != nil {
		return err
	}

	if !management.IsOperatorManaged(operatorSpec.ManagementState) {
		return nil
	}

	directResourceResults := resourceapply.ApplyDirectly(c.kubeClient, c.eventRecorder, c.mustTemplateAsset,
		"manifests/prometheus-role.yaml",
		"manifests/prometheus-role-binding.yaml",
	)

	errs := []error{}
	for _, currResult := range directResourceResults {
		if currResult.Error != nil {
			errs = append(errs, fmt.Errorf("%q (%T): %v", currResult.File, currResult.Type, currResult.Error))
		}
	}

	serviceMonitorBytes, err := c.mustTemplateAsset("manifests/service-monitor.yaml")
	if err != nil {
		errs = append(errs, fmt.Errorf("manifests/service-monitor.yaml: %v", err))
	} else {
		_, serviceMonitorErr := resourceapply.ApplyServiceMonitor(c.dynamicClient, c.eventRecorder, serviceMonitorBytes)
		// This is to handle 'the server could not find the requested resource' which occurs when the CRD is not available
		// yet (the CRD is provided by prometheus operator). This produce noise and plenty of events.
		if errors.IsNotFound(serviceMonitorErr) {
			klog.V(4).Infof("Unable to apply service monitor: %v", err)
			return syntheticRequeueError
		} else if serviceMonitorErr != nil {
			errs = append(errs, serviceMonitorErr)
		}
	}

	err = v1helpers.NewMultiLineAggregate(errs)

	// NOTE: Failing to create the monitoring resources should not lead to operator failed state.
	cond := operatorv1.OperatorCondition{
		Type:   operatorStatusMonitoringResourceControllerDegraded,
		Status: operatorv1.ConditionFalse,
	}
	if err != nil {
		// this is not a typo.  We will not have failing status on our operator for missing servicemonitor since servicemonitoring
		// is not a prereq.
		cond.Status = operatorv1.ConditionFalse
		cond.Reason = "Error"
		cond.Message = err.Error()
	}
	if _, _, updateError := v1helpers.UpdateStaticPodStatus(c.operatorClient, v1helpers.UpdateStaticPodConditionFn(cond)); updateError != nil {
		if err == nil {
			return updateError
		}
	}

	return err
}

func (c *MonitoringResourceController) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Infof("Starting MonitoringResourceController")
	defer klog.Infof("Shutting down MonitoringResourceController")
	if !cache.WaitForCacheSync(stopCh, c.cachesToSync...) {
		return
	}

	// doesn't matter what workers say, only start one.
	go wait.Until(c.runWorker, time.Second, stopCh)

	<-stopCh
}

func (c *MonitoringResourceController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *MonitoringResourceController) processNextWorkItem() bool {
	dsKey, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(dsKey)

	err := c.sync()
	if err == nil {
		c.queue.Forget(dsKey)
		return true
	}

	if err != syntheticRequeueError {
		utilruntime.HandleError(fmt.Errorf("%v failed with : %v", dsKey, err))
	}

	c.queue.AddRateLimited(dsKey)

	return true
}

// eventHandler queues the operator to check spec and status
func (c *MonitoringResourceController) eventHandler() cache.ResourceEventHandler {
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { c.queue.Add(controllerWorkQueueKey) },
		UpdateFunc: func(old, new interface{}) { c.queue.Add(controllerWorkQueueKey) },
		DeleteFunc: func(obj interface{}) { c.queue.Add(controllerWorkQueueKey) },
	}
}
