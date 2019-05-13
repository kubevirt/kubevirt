package backingresource

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/openshift/library-go/pkg/operator/management"
	"github.com/openshift/library-go/pkg/operator/v1helpers"

	"k8s.io/klog"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	rbaclisterv1 "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/library-go/pkg/assets"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	"github.com/openshift/library-go/pkg/operator/staticpod/controller/backingresource/bindata"
)

const (
	operatorStatusBackingResourceControllerDegraded = "BackingResourceControllerDegraded"
	controllerWorkQueueKey                          = "key"
	manifestDir                                     = "pkg/operator/staticpod/controller/backingresource"
)

// BackingResourceController is a controller that watches the operator config and updates
// service accounts and RBAC rules in the target namespace according to the bindata manifests
// (templated with the config) if they differ.
type BackingResourceController struct {
	targetNamespace string

	operatorClient           v1helpers.OperatorClient
	saLister                 corelisterv1.ServiceAccountLister
	clusterRoleBindingLister rbaclisterv1.ClusterRoleBindingLister
	kubeClient               kubernetes.Interface

	cachesToSync  []cache.InformerSynced
	queue         workqueue.RateLimitingInterface
	eventRecorder events.Recorder
}

// NewBackingResourceController creates a new backing resource controller.
func NewBackingResourceController(
	targetNamespace string,
	operatorClient v1helpers.OperatorClient,
	kubeInformersForTargetNamespace informers.SharedInformerFactory,
	kubeClient kubernetes.Interface,
	eventRecorder events.Recorder,
) *BackingResourceController {
	c := &BackingResourceController{
		targetNamespace: targetNamespace,
		operatorClient:  operatorClient,

		saLister:                 kubeInformersForTargetNamespace.Core().V1().ServiceAccounts().Lister(),
		clusterRoleBindingLister: kubeInformersForTargetNamespace.Rbac().V1().ClusterRoleBindings().Lister(),
		kubeClient:               kubeClient,

		queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "BackingResourceController"),
		eventRecorder: eventRecorder.WithComponentSuffix("backing-resource-controller"),
	}

	operatorClient.Informer().AddEventHandler(c.eventHandler())
	kubeInformersForTargetNamespace.Core().V1().ServiceAccounts().Informer().AddEventHandler(c.eventHandler())
	kubeInformersForTargetNamespace.Rbac().V1().ClusterRoleBindings().Informer().AddEventHandler(c.eventHandler())

	c.cachesToSync = append(c.cachesToSync, operatorClient.Informer().HasSynced)
	c.cachesToSync = append(c.cachesToSync, kubeInformersForTargetNamespace.Core().V1().ServiceAccounts().Informer().HasSynced)
	c.cachesToSync = append(c.cachesToSync, kubeInformersForTargetNamespace.Rbac().V1().ClusterRoleBindings().Informer().HasSynced)

	return c
}

func (c BackingResourceController) mustTemplateAsset(name string) ([]byte, error) {
	config := struct {
		TargetNamespace string
	}{
		TargetNamespace: c.targetNamespace,
	}
	return assets.MustCreateAssetFromTemplate(name, bindata.MustAsset(filepath.Join(manifestDir, name)), config).Data, nil
}

func (c BackingResourceController) sync() error {
	operatorSpec, _, _, err := c.operatorClient.GetOperatorState()
	if err != nil {
		return err
	}

	if !management.IsOperatorManaged(operatorSpec.ManagementState) {
		return nil
	}

	directResourceResults := resourceapply.ApplyDirectly(c.kubeClient, c.eventRecorder, c.mustTemplateAsset,
		"manifests/installer-sa.yaml",
		"manifests/installer-cluster-rolebinding.yaml",
	)

	errs := []error{}
	for _, currResult := range directResourceResults {
		if currResult.Error != nil {
			errs = append(errs, fmt.Errorf("%q (%T): %v", currResult.File, currResult.Type, currResult.Error))
		}
	}
	err = v1helpers.NewMultiLineAggregate(errs)

	// update failing condition
	cond := operatorv1.OperatorCondition{
		Type:   operatorStatusBackingResourceControllerDegraded,
		Status: operatorv1.ConditionFalse,
	}
	if err != nil {
		cond.Status = operatorv1.ConditionTrue
		cond.Reason = "Error"
		cond.Message = err.Error()
	}
	if _, _, updateError := v1helpers.UpdateStatus(c.operatorClient, v1helpers.UpdateConditionFn(cond)); updateError != nil {
		if err == nil {
			return updateError
		}
	}

	return err
}

// Run starts the kube-apiserver and blocks until stopCh is closed.
func (c *BackingResourceController) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Infof("Starting BackingResourceController")
	defer klog.Infof("Shutting down BackingResourceController")
	if !cache.WaitForCacheSync(stopCh, c.cachesToSync...) {
		return
	}

	// doesn't matter what workers say, only start one.
	go wait.Until(c.runWorker, time.Second, stopCh)

	<-stopCh
}

func (c *BackingResourceController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *BackingResourceController) processNextWorkItem() bool {
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

	utilruntime.HandleError(fmt.Errorf("%v failed with : %v", dsKey, err))
	c.queue.AddRateLimited(dsKey)

	return true
}

// eventHandler queues the operator to check spec and status
func (c *BackingResourceController) eventHandler() cache.ResourceEventHandler {
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { c.queue.Add(controllerWorkQueueKey) },
		UpdateFunc: func(old, new interface{}) { c.queue.Add(controllerWorkQueueKey) },
		DeleteFunc: func(obj interface{}) { c.queue.Add(controllerWorkQueueKey) },
	}
}
