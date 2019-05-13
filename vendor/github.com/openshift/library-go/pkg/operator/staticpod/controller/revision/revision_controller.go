package revision

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/management"
	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
)

const operatorStatusRevisionControllerDegraded = "RevisionControllerDegraded"
const revisionControllerWorkQueueKey = "key"

// RevisionController is a controller that watches a set of configmaps and secrets and them against a revision snapshot
// of them. If the original resources changes, the revision counter is increased, stored in LatestAvailableRevision
// field of the operator config and new snapshots suffixed by the revision are created.
type RevisionController struct {
	targetNamespace string
	// configMaps is the list of configmaps that are directly copied.A different actor/controller modifies these.
	// the first element should be the configmap that contains the static pod manifest
	configMaps []RevisionResource
	// secrets is a list of secrets that are directly copied for the current values.  A different actor/controller modifies these.
	secrets []RevisionResource

	operatorClient  v1helpers.StaticPodOperatorClient
	configMapGetter corev1client.ConfigMapsGetter
	secretGetter    corev1client.SecretsGetter

	cachesToSync  []cache.InformerSynced
	queue         workqueue.RateLimitingInterface
	eventRecorder events.Recorder
}

type RevisionResource struct {
	Name     string
	Optional bool
}

// NewRevisionController create a new revision controller.
func NewRevisionController(
	targetNamespace string,
	configMaps []RevisionResource,
	secrets []RevisionResource,
	kubeInformersForTargetNamespace informers.SharedInformerFactory,
	operatorClient v1helpers.StaticPodOperatorClient,
	configMapGetter corev1client.ConfigMapsGetter,
	secretGetter corev1client.SecretsGetter,
	eventRecorder events.Recorder,
) *RevisionController {
	c := &RevisionController{
		targetNamespace: targetNamespace,
		configMaps:      configMaps,
		secrets:         secrets,

		operatorClient:  operatorClient,
		configMapGetter: configMapGetter,
		secretGetter:    secretGetter,
		eventRecorder:   eventRecorder.WithComponentSuffix("revision-controller"),

		queue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "RevisionController"),
	}

	operatorClient.Informer().AddEventHandler(c.eventHandler())
	kubeInformersForTargetNamespace.Core().V1().ConfigMaps().Informer().AddEventHandler(c.eventHandler())
	kubeInformersForTargetNamespace.Core().V1().Secrets().Informer().AddEventHandler(c.eventHandler())

	c.cachesToSync = append(c.cachesToSync, operatorClient.Informer().HasSynced)
	c.cachesToSync = append(c.cachesToSync, kubeInformersForTargetNamespace.Core().V1().ConfigMaps().Informer().HasSynced)
	c.cachesToSync = append(c.cachesToSync, kubeInformersForTargetNamespace.Core().V1().Secrets().Informer().HasSynced)

	return c
}

// createRevisionIfNeeded takes care of creating content for the static pods to use.
// returns whether or not requeue and if an error happened when updating status.  Normally it updates status itself.
func (c RevisionController) createRevisionIfNeeded(operatorSpec *operatorv1.StaticPodOperatorSpec, operatorStatusOriginal *operatorv1.StaticPodOperatorStatus, resourceVersion string) (bool, error) {
	operatorStatus := operatorStatusOriginal.DeepCopy()

	latestRevision := operatorStatus.LatestAvailableRevision
	isLatestRevisionCurrent, reason := c.isLatestRevisionCurrent(latestRevision)

	// check to make sure that the latestRevision has the exact content we expect.  No mutation here, so we start creating the next Revision only when it is required
	if isLatestRevisionCurrent {
		return false, nil
	}

	nextRevision := latestRevision + 1
	c.eventRecorder.Eventf("RevisionTriggered", "new revision %d triggered by %q", nextRevision, reason)
	if err := c.createNewRevision(nextRevision); err != nil {
		cond := operatorv1.OperatorCondition{
			Type:    "RevisionControllerDegraded",
			Status:  operatorv1.ConditionTrue,
			Reason:  "ContentCreationError",
			Message: err.Error(),
		}
		if _, _, updateError := v1helpers.UpdateStaticPodStatus(c.operatorClient, v1helpers.UpdateStaticPodConditionFn(cond)); updateError != nil {
			c.eventRecorder.Warningf("RevisionCreateFailed", "Failed to create revision %d: %v", nextRevision, err.Error())
			return true, updateError
		}
		return true, nil
	}

	cond := operatorv1.OperatorCondition{
		Type:   "RevisionControllerDegraded",
		Status: operatorv1.ConditionFalse,
	}
	if _, updated, updateError := v1helpers.UpdateStaticPodStatus(c.operatorClient, v1helpers.UpdateStaticPodConditionFn(cond), func(operatorStatus *operatorv1.StaticPodOperatorStatus) error {
		if operatorStatus.LatestAvailableRevision == nextRevision {
			klog.Warningf("revision %d is unexpectedly already the latest available revision. This is a possible race!", nextRevision)
			return fmt.Errorf("conflicting latestAvailableRevision %d", operatorStatus.LatestAvailableRevision)
		}
		operatorStatus.LatestAvailableRevision = nextRevision
		return nil
	}); updateError != nil {
		return true, updateError
	} else if updated {
		c.eventRecorder.Eventf("RevisionCreate", "Revision %d created because %s", operatorStatus.LatestAvailableRevision, reason)
	}

	return false, nil
}

func nameFor(name string, revision int32) string {
	return fmt.Sprintf("%s-%d", name, revision)
}

// isLatestRevisionCurrent returns whether the latest revision is up to date and an optional reason
func (c RevisionController) isLatestRevisionCurrent(revision int32) (bool, string) {
	configChanges := []string{}
	for _, cm := range c.configMaps {
		requiredData := map[string]string{}
		existingData := map[string]string{}

		required, err := c.configMapGetter.ConfigMaps(c.targetNamespace).Get(cm.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) && !cm.Optional {
			return false, err.Error()
		}
		existing, err := c.configMapGetter.ConfigMaps(c.targetNamespace).Get(nameFor(cm.Name, revision), metav1.GetOptions{})
		if apierrors.IsNotFound(err) && !cm.Optional {
			return false, err.Error()
		}
		if required != nil {
			requiredData = required.Data
		}
		if existing != nil {
			existingData = existing.Data
		}
		if !equality.Semantic.DeepEqual(existingData, requiredData) {
			if klog.V(4) {
				klog.Infof("configmap %q changes for revision %d: %s", cm.Name, revision, resourceapply.JSONPatch(existing, required))
			}
			configChanges = append(configChanges, fmt.Sprintf("configmap/%s has changed", cm.Name))
		}
	}

	secretChanges := []string{}
	for _, s := range c.secrets {
		requiredData := map[string][]byte{}
		existingData := map[string][]byte{}

		required, err := c.secretGetter.Secrets(c.targetNamespace).Get(s.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) && !s.Optional {
			return false, err.Error()
		}
		existing, err := c.secretGetter.Secrets(c.targetNamespace).Get(nameFor(s.Name, revision), metav1.GetOptions{})
		if apierrors.IsNotFound(err) && !s.Optional {
			return false, err.Error()
		}
		if required != nil {
			requiredData = required.Data
		}
		if existing != nil {
			existingData = existing.Data
		}
		if !equality.Semantic.DeepEqual(existingData, requiredData) {
			if klog.V(4) {
				klog.Infof("secret %q changes for revision %d: %s", s.Name, revision, resourceapply.JSONPatch(existing, required))
			}
			secretChanges = append(secretChanges, fmt.Sprintf("secret/%s has changed", s.Name))
		}
	}

	if len(secretChanges) > 0 || len(configChanges) > 0 {
		return false, strings.Join(append(secretChanges, configChanges...), ",")
	}

	return true, ""
}

func (c RevisionController) createNewRevision(revision int32) error {
	statusConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: c.targetNamespace,
			Name:      nameFor("revision-status", revision),
		},
		Data: map[string]string{
			"status":   "InProgress",
			"revision": fmt.Sprintf("%d", revision),
		},
	}
	statusConfigMap, _, err := resourceapply.ApplyConfigMap(c.configMapGetter, c.eventRecorder, statusConfigMap)
	if err != nil {
		return err
	}
	ownerRefs := []metav1.OwnerReference{{
		APIVersion: "v1",
		Kind:       "ConfigMap",
		Name:       statusConfigMap.Name,
		UID:        statusConfigMap.UID,
	}}

	for _, cm := range c.configMaps {
		obj, _, err := resourceapply.SyncConfigMap(c.configMapGetter, c.eventRecorder, c.targetNamespace, cm.Name, c.targetNamespace, nameFor(cm.Name, revision), ownerRefs)
		if err != nil {
			return err
		}
		if obj == nil && !cm.Optional {
			return apierrors.NewNotFound(corev1.Resource("configmaps"), cm.Name)
		}
	}
	for _, s := range c.secrets {
		obj, _, err := resourceapply.SyncSecret(c.secretGetter, c.eventRecorder, c.targetNamespace, s.Name, c.targetNamespace, nameFor(s.Name, revision), ownerRefs)
		if err != nil {
			return err
		}
		if obj == nil && !s.Optional {
			return apierrors.NewNotFound(corev1.Resource("secrets"), s.Name)
		}
	}

	return nil
}

// getLatestAvailableRevision returns the latest known revision to the operator
// This is either the LatestAvailableRevision in the status or by checking revision status configmaps
func (c RevisionController) getLatestAvailableRevision(operatorStatus *operatorv1.StaticPodOperatorStatus) (int32, error) {
	configMaps, err := c.configMapGetter.ConfigMaps(c.targetNamespace).List(metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	var latestRevision int32
	for _, configMap := range configMaps.Items {
		if !strings.HasPrefix(configMap.Name, "revision-status-") {
			continue
		}
		if revision, ok := configMap.Data["revision"]; ok {
			revisionNumber, err := strconv.Atoi(revision)
			if err != nil {
				return 0, err
			}
			if int32(revisionNumber) > latestRevision {
				latestRevision = int32(revisionNumber)
			}
		}
	}
	// If there are no configmaps, then this should actually be revision 0
	return latestRevision, nil
}

func (c RevisionController) sync() error {
	operatorSpec, originalOperatorStatus, resourceVersion, err := c.operatorClient.GetStaticPodOperatorStateWithQuorum()
	if err != nil {
		return err
	}
	operatorStatus := originalOperatorStatus.DeepCopy()

	if !management.IsOperatorManaged(operatorSpec.ManagementState) {
		return nil
	}

	// If the operator status has 0 as its latest available revision, this is either the first revision
	// or possibly the operator resource was deleted and reset back to 0, which is not what we want so check configmaps
	if operatorStatus.LatestAvailableRevision == 0 {
		// Check to see if current revision is accurate and if not, search through configmaps for latest revision
		latestRevision, err := c.getLatestAvailableRevision(operatorStatus)
		if err != nil {
			return err
		}
		if latestRevision != 0 {
			// Then make sure that revision number is what's in the operator status
			_, _, err = v1helpers.UpdateStaticPodStatus(c.operatorClient, func(status *operatorv1.StaticPodOperatorStatus) error {
				status.LatestAvailableRevision = latestRevision
				return nil
			})
			// If we made a change return and requeue with the correct status
			return fmt.Errorf("synthetic requeue request (err: %v)", err)
		}
	}

	requeue, syncErr := c.createRevisionIfNeeded(operatorSpec, operatorStatus, resourceVersion)
	if requeue && syncErr == nil {
		return fmt.Errorf("synthetic requeue request (err: %v)", syncErr)
	}
	err = syncErr

	// update failing condition
	cond := operatorv1.OperatorCondition{
		Type:   operatorStatusRevisionControllerDegraded,
		Status: operatorv1.ConditionFalse,
	}
	if err != nil {
		cond.Status = operatorv1.ConditionTrue
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

// Run starts the kube-apiserver and blocks until stopCh is closed.
func (c *RevisionController) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Infof("Starting RevisionController")
	defer klog.Infof("Shutting down RevisionController")
	if !cache.WaitForCacheSync(stopCh, c.cachesToSync...) {
		return
	}

	// doesn't matter what workers say, only start one.
	go wait.Until(c.runWorker, time.Second, stopCh)

	<-stopCh
}

func (c *RevisionController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *RevisionController) processNextWorkItem() bool {
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
func (c *RevisionController) eventHandler() cache.ResourceEventHandler {
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { c.queue.Add(revisionControllerWorkQueueKey) },
		UpdateFunc: func(old, new interface{}) { c.queue.Add(revisionControllerWorkQueueKey) },
		DeleteFunc: func(obj interface{}) { c.queue.Add(revisionControllerWorkQueueKey) },
	}
}
