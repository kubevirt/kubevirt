package catalog

import (
	"encoding/json"
	"errors"
	"fmt"

	"sync"
	"time"

	olmerrors "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1beta1ext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/informers/externalversions"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/controller/registry"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/controller/registry/resolver"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/queueinformer"
)

const (
	crdKind         = "CustomResourceDefinition"
	secretKind      = "Secret"
	clusterRoleKind = "ClusterRole"
)

//for test stubbing and for ensuring standardization of timezones to UTC
var timeNow = func() metav1.Time { return metav1.NewTime(time.Now().UTC()) }

// Operator represents a Kubernetes operator that executes InstallPlans by
// resolving dependencies in a catalog.
type Operator struct {
	*queueinformer.Operator
	client             versioned.Interface
	namespace          string
	sources            map[registry.ResourceKey]registry.Source
	sourcesLock        sync.RWMutex
	sourcesLastUpdate  metav1.Time
	dependencyResolver resolver.DependencyResolver
	subQueue           workqueue.RateLimitingInterface
}

// NewOperator creates a new Catalog Operator.
func NewOperator(kubeconfigPath string, wakeupInterval time.Duration, operatorNamespace string, watchedNamespaces ...string) (*Operator, error) {
	// Default to watching all namespaces.
	if watchedNamespaces == nil {
		watchedNamespaces = []string{metav1.NamespaceAll}
	}

	// Create a new client for ALM types (CRs)
	crClient, err := client.NewClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	// Create an informer for each watched namespace.
	ipSharedIndexInformers := []cache.SharedIndexInformer{}
	subSharedIndexInformers := []cache.SharedIndexInformer{}
	for _, namespace := range watchedNamespaces {
		nsInformerFactory := externalversions.NewSharedInformerFactoryWithOptions(crClient, wakeupInterval, externalversions.WithNamespace(namespace))
		ipSharedIndexInformers = append(ipSharedIndexInformers, nsInformerFactory.Operators().V1alpha1().InstallPlans().Informer())
		subSharedIndexInformers = append(subSharedIndexInformers, nsInformerFactory.Operators().V1alpha1().Subscriptions().Informer())
	}

	// Create an informer for each catalog namespace
	catsrcSharedIndexInformers := []cache.SharedIndexInformer{}
	for _, namespace := range []string{operatorNamespace} {
		nsInformerFactory := externalversions.NewSharedInformerFactoryWithOptions(crClient, wakeupInterval, externalversions.WithNamespace(namespace))
		catsrcSharedIndexInformers = append(catsrcSharedIndexInformers, nsInformerFactory.Operators().V1alpha1().CatalogSources().Informer())
	}

	// Create a new queueinformer-based operator.
	queueOperator, err := queueinformer.NewOperator(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	// Allocate the new instance of an Operator.
	op := &Operator{
		Operator:           queueOperator,
		client:             crClient,
		namespace:          operatorNamespace,
		sources:            make(map[registry.ResourceKey]registry.Source),
		dependencyResolver: &resolver.MultiSourceResolver{},
	}

	// Register CatalogSource informers.
	catsrcQueue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "catalogsources")
	catsrcQueueInformer := queueinformer.New(
		catsrcQueue,
		catsrcSharedIndexInformers,
		op.syncCatalogSources,
		nil,
		"catsrc",
	)
	for _, informer := range catsrcQueueInformer {
		op.RegisterQueueInformer(informer)
	}

	// Register InstallPlan informers.
	ipQueue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "installplans")
	ipQueueInformers := queueinformer.New(
		ipQueue,
		ipSharedIndexInformers,
		op.syncInstallPlans,
		nil,
		"installplan",
	)
	for _, informer := range ipQueueInformers {
		op.RegisterQueueInformer(informer)
	}

	// Register Subscription informers.
	subscriptionQueue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "subscriptions")
	subscriptionQueueInformers := queueinformer.New(
		subscriptionQueue,
		subSharedIndexInformers,
		op.syncSubscriptions,
		nil,
		"subscription",
	)
	op.subQueue = subscriptionQueue
	for _, informer := range subscriptionQueueInformers {
		op.RegisterQueueInformer(informer)
	}

	return op, nil
}

func (o *Operator) syncCatalogSources(obj interface{}) (syncError error) {
	catsrc, ok := obj.(*v1alpha1.CatalogSource)
	if !ok {
		log.Debugf("wrong type: %#v", obj)
		return fmt.Errorf("casting CatalogSource failed")
	}

	// Get the catalog source's config map
	configMap, err := o.OpClient.KubernetesInterface().CoreV1().ConfigMaps(catsrc.GetNamespace()).Get(catsrc.Spec.ConfigMap, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get catalog config map %s when updating status: %s", catsrc.Spec.ConfigMap, err)
	}

	o.sourcesLock.Lock()
	defer o.sourcesLock.Unlock()
	sourceKey := registry.ResourceKey{Name: catsrc.GetName(), Namespace: catsrc.GetNamespace()}
	_, ok = o.sources[sourceKey]

	// Check for catalog source changes
	if ok && catsrc.Status.ConfigMapResource != nil && catsrc.Status.ConfigMapResource.Name == configMap.GetName() && catsrc.Status.ConfigMapResource.ResourceVersion == configMap.GetResourceVersion() {
		return nil
	}

	// Update status subresource
	out := catsrc.DeepCopy()
	out.Status.ConfigMapResource = &v1alpha1.ConfigMapResourceReference{
		Name:            configMap.GetName(),
		Namespace:       configMap.GetNamespace(),
		UID:             configMap.GetUID(),
		ResourceVersion: configMap.GetResourceVersion(),
	}
	out.Status.LastSync = timeNow()

	_, err = o.client.OperatorsV1alpha1().CatalogSources(out.GetNamespace()).UpdateStatus(out)
	if err != nil {
		return fmt.Errorf("failed to update catalog source %s status: %s", out.GetName(), err)
	}

	// Create a new in-mem registry
	src, err := registry.NewInMemoryFromConfigMap(o.OpClient, out.GetNamespace(), out.Spec.ConfigMap)
	if err != nil {
		return fmt.Errorf("failed to create catalog source from ConfigMap %s: %s", out.Spec.ConfigMap, err)
	}

	// Update sources map
	o.sources[sourceKey] = src
	o.sourcesLastUpdate = timeNow()

	return nil
}

func (o *Operator) syncSubscriptions(obj interface{}) (syncError error) {
	sub, ok := obj.(*v1alpha1.Subscription)
	if !ok {
		log.Debugf("wrong type: %#v", obj)
		return fmt.Errorf("casting Subscription failed")
	}

	logger := log.WithFields(log.Fields{
		"sub":       sub.GetName(),
		"namespace": sub.GetNamespace(),
		"source":    sub.Spec.CatalogSource,
		"pkg":       sub.Spec.Package,
		"channel":   sub.Spec.Channel,
	})

	logger.Infof("syncing")

	var updatedSub *v1alpha1.Subscription
	updatedSub, syncError = o.syncSubscription(sub)

	if updatedSub == nil || updatedSub.Status.State == sub.Status.State {
		return
	}
	if syncError != nil {
		logger = logger.WithField("syncError", syncError)
	}

	updatedSub.Status.LastUpdated = timeNow()
	// Update Subscription with status of transition. Log errors if we can't write them to the status.
	if _, err := o.client.OperatorsV1alpha1().Subscriptions(updatedSub.GetNamespace()).UpdateStatus(updatedSub); err != nil {
		logger = logger.WithField("updateError", err.Error())
		updateErr := errors.New("error updating Subscription status: " + err.Error())
		if syncError == nil {
			logger.Info("error updating Subscription status")
			return updateErr
		}
		logger.Info("error transitioning Subscription")
		syncError = fmt.Errorf("error transitioning Subscription: %s and error updating Subscription status: %s", syncError, updateErr)
	}

	return
}

func (o *Operator) requeueInstallPlan(name, namespace string) {
	// we can build the key directly, will need to change if queue uses different key scheme
	key := fmt.Sprintf("%s/%s", namespace, name)
	o.subQueue.AddRateLimited(key)
	return
}

func (o *Operator) syncInstallPlans(obj interface{}) (syncError error) {
	plan, ok := obj.(*v1alpha1.InstallPlan)
	if !ok {
		log.Debugf("wrong type: %#v", obj)
		return fmt.Errorf("casting InstallPlan failed")
	}

	logger := log.WithFields(log.Fields{
		"ip":        plan.GetName(),
		"namespace": plan.GetNamespace(),
		"phase":     plan.Status.Phase,
	})

	logger.Info("syncing")
	outInstallPlan, syncError := transitionInstallPlanState(o, *plan)

	if syncError != nil {
		logger = logger.WithField("syncError", syncError)
	}

	// no changes in status, don't update
	if outInstallPlan.Status.Phase == plan.Status.Phase {
		return
	}

	// notify subscription loop of installplan changes
	if ownerutil.IsOwnedByKind(outInstallPlan, v1alpha1.SubscriptionKind) {
		oref := ownerutil.GetOwnerByKind(outInstallPlan, v1alpha1.SubscriptionKind)
		o.requeueInstallPlan(oref.Name, outInstallPlan.GetNamespace())
	}

	// Update InstallPlan with status of transition. Log errors if we can't write them to the status.
	if _, err := o.client.OperatorsV1alpha1().InstallPlans(plan.GetNamespace()).UpdateStatus(outInstallPlan); err != nil {
		logger = logger.WithField("updateError", err.Error())
		updateErr := errors.New("error updating InstallPlan status: " + err.Error())
		if syncError == nil {
			logger.Info("error updating InstallPlan status")
			return updateErr
		}
		logger.Info("error transitioning InstallPlan")
		syncError = fmt.Errorf("error transitioning InstallPlan: %s and error updating InstallPlan status: %s", syncError, updateErr)
	}
	return
}

type installPlanTransitioner interface {
	ResolvePlan(*v1alpha1.InstallPlan) error
	ExecutePlan(*v1alpha1.InstallPlan) error
}

var _ installPlanTransitioner = &Operator{}

func transitionInstallPlanState(transitioner installPlanTransitioner, in v1alpha1.InstallPlan) (*v1alpha1.InstallPlan, error) {
	logger := log.WithFields(log.Fields{
		"ip":        in.GetName(),
		"namespace": in.GetNamespace(),
		"phase":     in.Status.Phase,
	})

	out := in.DeepCopy()

	switch in.Status.Phase {
	case v1alpha1.InstallPlanPhaseNone:
		logger.Debugf("setting phase to %s", v1alpha1.InstallPlanPhasePlanning)
		out.Status.Phase = v1alpha1.InstallPlanPhasePlanning
		return out, nil

	case v1alpha1.InstallPlanPhasePlanning:
		logger.Debug("attempting to resolve")
		if err := transitioner.ResolvePlan(out); err != nil {
			out.Status.SetCondition(v1alpha1.ConditionFailed(v1alpha1.InstallPlanResolved,
				v1alpha1.InstallPlanReasonInstallCheckFailed, err))
			out.Status.Phase = v1alpha1.InstallPlanPhaseFailed
			return out, err
		}
		out.Status.SetCondition(v1alpha1.ConditionMet(v1alpha1.InstallPlanResolved))

		if out.Spec.Approval == v1alpha1.ApprovalManual && out.Spec.Approved != true {
			out.Status.Phase = v1alpha1.InstallPlanPhaseRequiresApproval
		} else {
			out.Status.Phase = v1alpha1.InstallPlanPhaseInstalling
		}
		return out, nil

	case v1alpha1.InstallPlanPhaseRequiresApproval:
		if out.Spec.Approved {
			logger.Debugf("approved, setting to %s", v1alpha1.InstallPlanPhasePlanning)
			out.Status.Phase = v1alpha1.InstallPlanPhaseInstalling
		} else {
			logger.Debug("not approved, skipping sync")
		}
		return out, nil

	case v1alpha1.InstallPlanPhaseInstalling:
		logger.Debug("attempting to install")
		if err := transitioner.ExecutePlan(out); err != nil {
			out.Status.SetCondition(v1alpha1.ConditionFailed(v1alpha1.InstallPlanInstalled,
				v1alpha1.InstallPlanReasonComponentFailed, err))
			out.Status.Phase = v1alpha1.InstallPlanPhaseFailed
			return out, err
		}
		out.Status.SetCondition(v1alpha1.ConditionMet(v1alpha1.InstallPlanInstalled))
		out.Status.Phase = v1alpha1.InstallPlanPhaseComplete
		return out, nil
	default:
		return out, nil
	}
}

// ResolvePlan modifies an InstallPlan to contain a Plan in its Status field.
func (o *Operator) ResolvePlan(plan *v1alpha1.InstallPlan) error {
	if plan.Status.Phase != v1alpha1.InstallPlanPhasePlanning {
		panic("attempted to create a plan that wasn't in the planning phase")
	}

	if len(o.sources) == 0 {
		return fmt.Errorf("cannot resolve InstallPlan without any Catalog Sources")
	}

	// Take a snapshot of the included catalog sources
	includedNamespaces := map[string]struct{}{
		o.namespace:    {},
		plan.Namespace: {},
	}
	sourcesSnapshot := o.getSourcesSnapshot(plan, includedNamespaces)

	// Take a snapshot of the existing CRD owners
	existingCRDOwners, err := o.getExistingCRDOwners(plan.Namespace)
	if err != nil {
		return err
	}

	// Attempt to resolve the InstallPlan
	steps, usedSources, err := o.dependencyResolver.ResolveInstallPlan(sourcesSnapshot, existingCRDOwners, CatalogLabel, plan)
	if err != nil {
		return err
	}

	// Set the resolved steps
	plan.Status.Plan = steps
	plan.Status.CatalogSources = []string{}

	// Add secrets for each used catalog source
	for _, sourceKey := range usedSources {
		// Append the used catalog source
		plan.Status.CatalogSources = append(plan.Status.CatalogSources, sourceKey.Name)

		// Get the catalog source
		catsrc, err := o.client.OperatorsV1alpha1().CatalogSources(sourceKey.Namespace).Get(sourceKey.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		for _, secretName := range catsrc.Spec.Secrets {
			// Attempt to look up the secret
			_, err := o.OpClient.KubernetesInterface().CoreV1().Secrets(sourceKey.Namespace).Get(secretName, metav1.GetOptions{})
			status := v1alpha1.StepStatusUnknown
			if k8serrors.IsNotFound(err) {
				status = v1alpha1.StepStatusNotPresent
			} else if err == nil {
				status = v1alpha1.StepStatusPresent
			} else {
				return err
			}

			// Prepend any required secrets to the plan for that catalog source
			plan.Status.Plan = append([]v1alpha1.Step{{
				Resolving: "",
				Resource: v1alpha1.StepResource{
					Name:    secretName,
					Kind:    "Secret",
					Group:   "",
					Version: "v1",
				},
				Status: status,
			}}, plan.Status.Plan...)
		}
	}

	return nil
}

// ExecutePlan applies a planned InstallPlan to a namespace.
func (o *Operator) ExecutePlan(plan *v1alpha1.InstallPlan) error {
	if plan.Status.Phase != v1alpha1.InstallPlanPhaseInstalling {
		panic("attempted to install a plan that wasn't in the installing phase")
	}

	// Get the set of initial installplan csv names
	initialCSVNames := getCSVNameSet(plan)
	// Get pre-existing CRD owners to make decisions about applying resolved CSVs
	existingCRDOwners, err := o.getExistingCRDOwners(plan.GetNamespace())
	if err != nil {
		return err
	}

	for i, step := range plan.Status.Plan {
		switch step.Status {
		case v1alpha1.StepStatusPresent, v1alpha1.StepStatusCreated:
			continue

		case v1alpha1.StepStatusUnknown, v1alpha1.StepStatusNotPresent:
			log.Debugf("resource kind: %s", step.Resource.Kind)
			log.Debugf("resource name: %s", step.Resource.Name)
			switch step.Resource.Kind {
			case crdKind:
				// Marshal the manifest into a CRD instance.
				var crd v1beta1ext.CustomResourceDefinition
				err := json.Unmarshal([]byte(step.Resource.Manifest), &crd)
				if err != nil {
					return err
				}

				// TODO: check that names are accepted
				// Attempt to create the CRD.
				_, err = o.OpClient.ApiextensionsV1beta1Interface().ApiextensionsV1beta1().CustomResourceDefinitions().Create(&crd)
				if k8serrors.IsAlreadyExists(err) {
					// If it already existed, mark the step as Present.
					plan.Status.Plan[i].Status = v1alpha1.StepStatusPresent
					continue
				} else if err != nil {
					return err
				} else {
					// If no error occured, mark the step as Created.
					plan.Status.Plan[i].Status = v1alpha1.StepStatusCreated
					continue
				}

			case v1alpha1.ClusterServiceVersionKind:
				// Marshal the manifest into a CSV instance.
				var csv v1alpha1.ClusterServiceVersion
				err := json.Unmarshal([]byte(step.Resource.Manifest), &csv)
				if err != nil {
					return err
				}

				// Check if the resolved CSV is in the initial set
				if _, ok := initialCSVNames[csv.GetName()]; !ok {
					// Check for pre-existing CSVs that own the same CRDs
					competingOwners, err := competingCRDOwnersExist(plan.GetNamespace(), &csv, existingCRDOwners)
					if err != nil {
						return err
					}

					// TODO: decide on fail/continue logic for pre-existing dependent CSVs that own the same CRD(s)
					if competingOwners {
						// For now, error out
						return fmt.Errorf("Pre-existing CRD owners found for owned CRD(s) of dependent CSV %s", csv.GetName())
					}
				}

				// Attempt to create the CSV.
				_, err = o.client.OperatorsV1alpha1().ClusterServiceVersions(csv.GetNamespace()).Create(&csv)
				if k8serrors.IsAlreadyExists(err) {
					// If it already existed, mark the step as Present.
					plan.Status.Plan[i].Status = v1alpha1.StepStatusPresent
				} else if err != nil {
					return err
				} else {
					// If no error occurred, mark the step as Created.
					plan.Status.Plan[i].Status = v1alpha1.StepStatusCreated
				}

			case secretKind:
				// Get the pre-existing secret.
				secret, err := o.OpClient.KubernetesInterface().CoreV1().Secrets(o.namespace).Get(step.Resource.Name, metav1.GetOptions{})
				if k8serrors.IsNotFound(err) {
					return fmt.Errorf("secret %s does not exist", step.Resource.Name)
				} else if err != nil {
					return err
				}

				// Set the namespace to the InstallPlan's namespace and attempt to
				// create a new secret.
				secret.Namespace = plan.Namespace
				_, err = o.OpClient.KubernetesInterface().CoreV1().Secrets(plan.Namespace).Create(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secret.Name,
						Namespace: plan.Namespace,
					},
					Data: secret.Data,
					Type: secret.Type,
				})
				if k8serrors.IsAlreadyExists(err) {
					// If it already existed, mark the step as Present.
					plan.Status.Plan[i].Status = v1alpha1.StepStatusPresent
				} else if err != nil {
					return err
				} else {
					// If no error occured, mark the step as Created.
					plan.Status.Plan[i].Status = v1alpha1.StepStatusCreated
				}

			case clusterRoleKind:
				// Marshal the manifest into a ClusterRole instance.
				var cr rbacv1.ClusterRole
				err := json.Unmarshal([]byte(step.Resource.Manifest), &cr)
				if err != nil {
					return err
				}

				// Attempt to create the ClusterRole.
				_, err = o.OpClient.KubernetesInterface().RbacV1().ClusterRoles().Create(&cr)
				if k8serrors.IsAlreadyExists(err) {
					// If it already existed, mark the step as Present.
					plan.Status.Plan[i].Status = v1alpha1.StepStatusPresent
				} else if err != nil {
					return err
				} else {
					// If no error occurred, mark the step as Created.
					plan.Status.Plan[i].Status = v1alpha1.StepStatusCreated
				}

			default:
				return v1alpha1.ErrInvalidInstallPlan
			}

		default:
			return v1alpha1.ErrInvalidInstallPlan
		}
	}

	// Loop over one final time to check and see if everything is good.
	for _, step := range plan.Status.Plan {
		switch step.Status {
		case v1alpha1.StepStatusCreated, v1alpha1.StepStatusPresent:
		default:
			return nil
		}
	}

	return nil
}

func (o *Operator) getSourcesSnapshot(plan *v1alpha1.InstallPlan, includedNamespaces map[string]struct{}) []registry.SourceRef {
	o.sourcesLock.RLock()
	defer o.sourcesLock.RUnlock()
	sourcesSnapshot := []registry.SourceRef{}

	for key, source := range o.sources {
		// Only copy catalog sources in included namespaces
		if _, ok := includedNamespaces[key.Namespace]; ok {
			ref := registry.SourceRef{
				Source:    source,
				SourceKey: key,
			}
			if key.Name == plan.Spec.CatalogSource && key.Namespace == plan.Spec.CatalogSourceNamespace {
				// Prepend preffered catalog source
				sourcesSnapshot = append([]registry.SourceRef{ref}, sourcesSnapshot...)
			} else {
				// Append the catalog source
				sourcesSnapshot = append(sourcesSnapshot, ref)
			}
		}
	}

	return sourcesSnapshot
}

// getExistingCRDOwners creates a map of CRD names to existing owner CSVs in the given namespace
func (o *Operator) getExistingCRDOwners(namespace string) (map[string][]string, error) {
	// Get a list of CSV CRs in the namespace
	csvList, err := o.client.OperatorsV1alpha1().ClusterServiceVersions(namespace).List(metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	// Map CRD names to existing owner CSV CRs in the namespace
	owners := make(map[string][]string)
	for _, csv := range csvList.Items {
		for _, crd := range csv.Spec.CustomResourceDefinitions.Owned {
			owners[crd.Name] = append(owners[crd.Name], csv.GetName())
		}
	}

	return owners, nil
}

// competingCRDOwnersExist returns true if there exists a CSV that owns at least one of the given CSVs owned CRDs (that's not the given CSV)
func competingCRDOwnersExist(namespace string, csv *v1alpha1.ClusterServiceVersion, existingOwners map[string][]string) (bool, error) {
	// Attempt to find a pre-existing owner in the namespace for any owned crd
	for _, crdDesc := range csv.Spec.CustomResourceDefinitions.Owned {
		crdOwners := existingOwners[crdDesc.Name]
		l := len(crdOwners)
		switch {
		case l == 1:
			// One competing owner found
			if crdOwners[0] != csv.GetName() {
				return true, nil
			}
		case l > 1:
			return true, olmerrors.NewMultipleExistingCRDOwnersError(crdOwners, crdDesc.Name, namespace)
		}
	}

	return false, nil
}

// getCSVNameSet returns a set of the given installplan's csv names
func getCSVNameSet(plan *v1alpha1.InstallPlan) map[string]struct{} {
	csvNameSet := make(map[string]struct{})
	for _, name := range plan.Spec.ClusterServiceVersionNames {
		csvNameSet[name] = struct{}{}
	}

	return csvNameSet
}
