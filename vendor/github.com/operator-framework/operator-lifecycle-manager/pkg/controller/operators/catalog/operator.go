package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	registryclient "github.com/operator-framework/operator-registry/pkg/client"
	errorwrap "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	v1beta1ext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	validation "k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	extinf "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilclock "k8s.io/apimachinery/pkg/util/clock"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/reference"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/informers/externalversions"
	olmerrors "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/errors"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/controller/operators/catalog/subscription"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/controller/registry/reconciler"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/controller/registry/resolver"
	index "github.com/operator-framework/operator-lifecycle-manager/pkg/lib/index"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorclient"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorlister"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/queueinformer"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/scoped"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/metrics"
)

const (
	crdKind                = "CustomResourceDefinition"
	secretKind             = "Secret"
	clusterRoleKind        = "ClusterRole"
	clusterRoleBindingKind = "ClusterRoleBinding"
	serviceAccountKind     = "ServiceAccount"
	serviceKind            = "Service"
	roleKind               = "Role"
	roleBindingKind        = "RoleBinding"
	generatedByKey         = "olm.generated-by"
)

// Operator represents a Kubernetes operator that executes InstallPlans by
// resolving dependencies in a catalog.
type Operator struct {
	queueinformer.Operator

	logger                 *logrus.Logger
	clock                  utilclock.Clock
	opClient               operatorclient.ClientInterface
	client                 versioned.Interface
	dynamicClient          dynamic.Interface
	lister                 operatorlister.OperatorLister
	catsrcQueueSet         *queueinformer.ResourceQueueSet
	subQueueSet            *queueinformer.ResourceQueueSet
	ipQueueSet             *queueinformer.ResourceQueueSet
	nsResolveQueue         workqueue.RateLimitingInterface
	namespace              string
	sources                map[resolver.CatalogKey]resolver.SourceRef
	sourcesLock            sync.RWMutex
	sourcesLastUpdate      metav1.Time
	resolver               resolver.Resolver
	reconciler             reconciler.RegistryReconcilerFactory
	csvProvidedAPIsIndexer map[string]cache.Indexer
	clientAttenuator       *scoped.ClientAttenuator
	serviceAccountQuerier  *scoped.UserDefinedServiceAccountQuerier
}

// NewOperator creates a new Catalog Operator.
func NewOperator(ctx context.Context, kubeconfigPath string, clock utilclock.Clock, logger *logrus.Logger, resyncPeriod time.Duration, configmapRegistryImage, operatorNamespace string, watchedNamespaces ...string) (*Operator, error) {
	// Default to watching all namespaces.
	if len(watchedNamespaces) == 0 {
		watchedNamespaces = []string{metav1.NamespaceAll}
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}

	// Create a new client for OLM types (CRs)
	crClient, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Create a new client for dynamic types (CRs)
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Create a new queueinformer-based operator.
	opClient := operatorclient.NewClientFromConfig(kubeconfigPath, logger)
	queueOperator, err := queueinformer.NewOperator(opClient.KubernetesInterface().Discovery(), queueinformer.WithOperatorLogger(logger))
	if err != nil {
		return nil, err
	}

	// Create an OperatorLister
	lister := operatorlister.NewLister()

	// Allocate the new instance of an Operator.
	op := &Operator{
		Operator:               queueOperator,
		logger:                 logger,
		clock:                  clock,
		opClient:               opClient,
		dynamicClient:          dynamicClient,
		client:                 crClient,
		lister:                 lister,
		namespace:              operatorNamespace,
		sources:                make(map[resolver.CatalogKey]resolver.SourceRef),
		resolver:               resolver.NewOperatorsV1alpha1Resolver(lister),
		catsrcQueueSet:         queueinformer.NewEmptyResourceQueueSet(),
		subQueueSet:            queueinformer.NewEmptyResourceQueueSet(),
		csvProvidedAPIsIndexer: map[string]cache.Indexer{},
		serviceAccountQuerier:  scoped.NewUserDefinedServiceAccountQuerier(logger, crClient),
		clientAttenuator:       scoped.NewClientAttenuator(logger, config, opClient, crClient),
	}
	op.reconciler = reconciler.NewRegistryReconcilerFactory(lister, opClient, configmapRegistryImage, op.now)

	// Set up syncing for namespace-scoped resources
	for _, namespace := range watchedNamespaces {
		// Wire OLM CR informers
		crInformerFactory := externalversions.NewSharedInformerFactoryWithOptions(op.client, resyncPeriod, externalversions.WithNamespace(namespace))

		// Wire CSVs
		csvInformer := crInformerFactory.Operators().V1alpha1().ClusterServiceVersions()
		op.lister.OperatorsV1alpha1().RegisterClusterServiceVersionLister(namespace, csvInformer.Lister())
		op.RegisterInformer(csvInformer.Informer())

		csvInformer.Informer().AddIndexers(cache.Indexers{index.ProvidedAPIsIndexFuncKey: index.ProvidedAPIsIndexFunc})
		csvIndexer := csvInformer.Informer().GetIndexer()
		op.csvProvidedAPIsIndexer[namespace] = csvIndexer

		// TODO: Add namespace resolve sync

		// Wire InstallPlans
		ipInformer := crInformerFactory.Operators().V1alpha1().InstallPlans()
		op.lister.OperatorsV1alpha1().RegisterInstallPlanLister(namespace, ipInformer.Lister())
		ipQueueInformer, err := queueinformer.NewQueueInformer(
			ctx,
			queueinformer.WithMetricsProvider(metrics.NewMetricsInstallPlan(op.client)),
			queueinformer.WithLogger(op.logger),
			queueinformer.WithInformer(ipInformer.Informer()),
			queueinformer.WithSyncer(queueinformer.LegacySyncHandler(op.syncInstallPlans).ToSyncer()),
		)
		if err != nil {
			return nil, err
		}
		op.RegisterQueueInformer(ipQueueInformer)

		// Wire CatalogSources
		catsrcInformer := crInformerFactory.Operators().V1alpha1().CatalogSources()
		op.lister.OperatorsV1alpha1().RegisterCatalogSourceLister(namespace, catsrcInformer.Lister())
		catsrcQueue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), fmt.Sprintf("%s/catsrcs", namespace))
		op.catsrcQueueSet.Set(namespace, catsrcQueue)
		catsrcQueueInformer, err := queueinformer.NewQueueInformer(
			ctx,
			queueinformer.WithMetricsProvider(metrics.NewMetricsCatalogSource(op.client)),
			queueinformer.WithLogger(op.logger),
			queueinformer.WithQueue(catsrcQueue),
			queueinformer.WithInformer(catsrcInformer.Informer()),
			queueinformer.WithSyncer(queueinformer.LegacySyncHandler(op.syncCatalogSources).ToSyncerWithDelete(op.handleCatSrcDeletion)),
		)
		if err != nil {
			return nil, err
		}
		op.RegisterQueueInformer(catsrcQueueInformer)

		// Wire Subscriptions
		subInformer := crInformerFactory.Operators().V1alpha1().Subscriptions()
		op.lister.OperatorsV1alpha1().RegisterSubscriptionLister(namespace, subInformer.Lister())
		subQueue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), fmt.Sprintf("%s/subs", namespace))
		op.subQueueSet.Set(namespace, subQueue)
		subSyncer, err := subscription.NewSyncer(
			ctx,
			subscription.WithLogger(op.logger),
			subscription.WithClient(op.client),
			subscription.WithOperatorLister(op.lister),
			subscription.WithSubscriptionInformer(subInformer.Informer()),
			subscription.WithCatalogInformer(catsrcInformer.Informer()),
			subscription.WithInstallPlanInformer(ipInformer.Informer()),
			subscription.WithSubscriptionQueue(subQueue),
			subscription.WithAppendedReconcilers(subscription.ReconcilerFromLegacySyncHandler(op.syncSubscriptions, nil)),
			subscription.WithRegistryReconcilerFactory(op.reconciler),
			subscription.WithGlobalCatalogNamespace(op.namespace),
		)
		if err != nil {
			return nil, err
		}
		subQueueInformer, err := queueinformer.NewQueueInformer(
			ctx,
			queueinformer.WithMetricsProvider(metrics.NewMetricsSubscription(op.client)),
			queueinformer.WithLogger(op.logger),
			queueinformer.WithQueue(subQueue),
			queueinformer.WithInformer(subInformer.Informer()),
			queueinformer.WithSyncer(subSyncer),
		)
		if err != nil {
			return nil, err
		}
		op.RegisterQueueInformer(subQueueInformer)

		// Wire k8s informers
		k8sInformerFactory := informers.NewSharedInformerFactoryWithOptions(op.opClient.KubernetesInterface(), resyncPeriod, informers.WithNamespace(namespace))
		informers := []cache.SharedIndexInformer{}

		// Wire Roles
		roleInformer := k8sInformerFactory.Rbac().V1().Roles()
		op.lister.RbacV1().RegisterRoleLister(namespace, roleInformer.Lister())
		informers = append(informers, roleInformer.Informer())

		// Wire RoleBindings
		roleBindingInformer := k8sInformerFactory.Rbac().V1().RoleBindings()
		op.lister.RbacV1().RegisterRoleBindingLister(namespace, roleBindingInformer.Lister())
		informers = append(informers, roleBindingInformer.Informer())

		// Wire ServiceAccounts
		serviceAccountInformer := k8sInformerFactory.Core().V1().ServiceAccounts()
		op.lister.CoreV1().RegisterServiceAccountLister(namespace, serviceAccountInformer.Lister())
		informers = append(informers, serviceAccountInformer.Informer())

		// Wire Services
		serviceInformer := k8sInformerFactory.Core().V1().Services()
		op.lister.CoreV1().RegisterServiceLister(namespace, serviceInformer.Lister())
		informers = append(informers, serviceInformer.Informer())

		// Wire Pods
		podInformer := k8sInformerFactory.Core().V1().Pods()
		op.lister.CoreV1().RegisterPodLister(namespace, podInformer.Lister())
		informers = append(informers, podInformer.Informer())

		// Wire ConfigMaps
		configMapInformer := k8sInformerFactory.Core().V1().ConfigMaps()
		op.lister.CoreV1().RegisterConfigMapLister(namespace, configMapInformer.Lister())
		informers = append(informers, configMapInformer.Informer())

		// Generate and register QueueInformers for k8s resources
		k8sSyncer := queueinformer.LegacySyncHandler(op.syncObject).ToSyncerWithDelete(op.handleDeletion)
		for _, informer := range informers {
			queueInformer, err := queueinformer.NewQueueInformer(
				ctx,
				queueinformer.WithLogger(op.logger),
				queueinformer.WithInformer(informer),
				queueinformer.WithSyncer(k8sSyncer),
			)
			if err != nil {
				return nil, err
			}

			if err := op.RegisterQueueInformer(queueInformer); err != nil {
				return nil, err
			}
		}

	}

	// Register CustomResourceDefinition QueueInformer
	crdInformer := extinf.NewSharedInformerFactory(op.opClient.ApiextensionsV1beta1Interface(), resyncPeriod).Apiextensions().V1beta1().CustomResourceDefinitions()
	op.lister.APIExtensionsV1beta1().RegisterCustomResourceDefinitionLister(crdInformer.Lister())
	crdQueueInformer, err := queueinformer.NewQueueInformer(
		ctx,
		queueinformer.WithLogger(op.logger),
		queueinformer.WithInformer(crdInformer.Informer()),
		queueinformer.WithSyncer(queueinformer.LegacySyncHandler(op.syncObject).ToSyncerWithDelete(op.handleDeletion)),
	)
	if err != nil {
		return nil, err
	}
	op.RegisterQueueInformer(crdQueueInformer)

	// Namespace sync for resolving subscriptions
	namespaceInformer := informers.NewSharedInformerFactory(op.opClient.KubernetesInterface(), resyncPeriod).Core().V1().Namespaces()
	op.lister.CoreV1().RegisterNamespaceLister(namespaceInformer.Lister())
	op.nsResolveQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "resolver")
	namespaceQueueInformer, err := queueinformer.NewQueueInformer(
		ctx,
		queueinformer.WithLogger(op.logger),
		queueinformer.WithQueue(op.nsResolveQueue),
		queueinformer.WithInformer(namespaceInformer.Informer()),
		queueinformer.WithSyncer(queueinformer.LegacySyncHandler(op.syncResolvingNamespace).ToSyncer()),
	)
	if err != nil {
		return nil, err
	}
	op.RegisterQueueInformer(namespaceQueueInformer)

	return op, nil
}

func (o *Operator) now() metav1.Time {
	return metav1.NewTime(o.clock.Now().UTC())
}

func (o *Operator) requeueOwners(obj metav1.Object) {
	namespace := obj.GetNamespace()
	logger := o.logger.WithFields(logrus.Fields{
		"name":      obj.GetName(),
		"namespace": namespace,
	})

	for _, owner := range obj.GetOwnerReferences() {
		var queueSet *queueinformer.ResourceQueueSet
		switch kind := owner.Kind; kind {
		case v1alpha1.CatalogSourceKind:
			if err := o.catsrcQueueSet.Requeue(namespace, owner.Name); err != nil {
				logger.Warn(err.Error())
			}
			queueSet = o.catsrcQueueSet
		case v1alpha1.SubscriptionKind:
			if err := o.catsrcQueueSet.Requeue(namespace, owner.Name); err != nil {
				logger.Warn(err.Error())
			}
			queueSet = o.subQueueSet
		default:
			logger.WithField("kind", kind).Trace("untracked owner kind")
		}

		if queueSet != nil {
			logger.WithField("ref", owner).Trace("requeuing owner")
			queueSet.Requeue(namespace, owner.Name)
		}
	}
}

func (o *Operator) syncObject(obj interface{}) (syncError error) {
	// Assert as metav1.Object
	metaObj, ok := obj.(metav1.Object)
	if !ok {
		syncError = errors.New("casting to metav1 object failed")
		o.logger.Warn(syncError.Error())
		return
	}

	o.requeueOwners(metaObj)

	return
}

func (o *Operator) handleDeletion(obj interface{}) {
	metaObj, ok := obj.(metav1.Object)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("couldn't get object from tombstone %#v", obj))
			return
		}

		metaObj, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a metav1 object %#v", obj))
			return
		}
	}

	o.logger.WithFields(logrus.Fields{
		"name":      metaObj.GetName(),
		"namespace": metaObj.GetNamespace(),
	}).Debug("handling object deletion")

	o.requeueOwners(metaObj)

	return
}

func (o *Operator) handleCatSrcDeletion(obj interface{}) {
	catsrc, ok := obj.(metav1.Object)
	if !ok {
		if !ok {
			tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
			if !ok {
				utilruntime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
				return
			}

			catsrc, ok = tombstone.Obj.(metav1.Object)
			if !ok {
				utilruntime.HandleError(fmt.Errorf("Tombstone contained object that is not a Namespace %#v", obj))
				return
			}
		}
	}
	sourceKey := resolver.CatalogKey{Name: catsrc.GetName(), Namespace: catsrc.GetNamespace()}
	func() {
		o.sourcesLock.Lock()
		defer o.sourcesLock.Unlock()
		if s, ok := o.sources[sourceKey]; ok {
			if err := s.Client.Close(); err != nil {
				o.logger.WithError(err).Warn("error closing client")
			}
		}
		delete(o.sources, sourceKey)
	}()
	o.logger.WithField("source", sourceKey).Info("removed client for deleted catalogsource")
}

func (o *Operator) syncCatalogSources(obj interface{}) (syncError error) {
	catsrc, ok := obj.(*v1alpha1.CatalogSource)
	if !ok {
		o.logger.Debugf("wrong type: %#v", obj)
		return fmt.Errorf("casting CatalogSource failed")
	}

	logger := o.logger.WithFields(logrus.Fields{
		"source": catsrc.GetName(),
		"id":     queueinformer.NewLoopID(),
	})
	logger.Debug("syncing catsrc")
	out := catsrc.DeepCopy()
	sourceKey := resolver.CatalogKey{Name: catsrc.GetName(), Namespace: catsrc.GetNamespace()}

	if catsrc.Spec.SourceType == v1alpha1.SourceTypeInternal || catsrc.Spec.SourceType == v1alpha1.SourceTypeConfigmap {
		logger.Debug("checking catsrc configmap state")

		// Get the catalog source's config map
		configMap, err := o.lister.CoreV1().ConfigMapLister().ConfigMaps(catsrc.GetNamespace()).Get(catsrc.Spec.ConfigMap)
		if err != nil {
			return fmt.Errorf("failed to get catalog config map %s: %s", catsrc.Spec.ConfigMap, err)
		}

		if wasOwned := ownerutil.EnsureOwner(configMap, catsrc); !wasOwned {
			configMap, err = o.opClient.KubernetesInterface().CoreV1().ConfigMaps(configMap.GetNamespace()).Update(configMap)
			if err != nil {
				return fmt.Errorf("unable to write owner onto catalog source configmap")
			}
			logger.Debug("adopted configmap")
		}

		if catsrc.Status.ConfigMapResource == nil || catsrc.Status.ConfigMapResource.UID != configMap.GetUID() || catsrc.Status.ConfigMapResource.ResourceVersion != configMap.GetResourceVersion() {
			logger.Debug("updating catsrc configmap state")
			// configmap ref nonexistent or updated, write out the new configmap ref to status and exit
			out.Status.ConfigMapResource = &v1alpha1.ConfigMapResourceReference{
				Name:            configMap.GetName(),
				Namespace:       configMap.GetNamespace(),
				UID:             configMap.GetUID(),
				ResourceVersion: configMap.GetResourceVersion(),
			}

			out.Status.LastSync = o.now()
			if _, err := o.client.OperatorsV1alpha1().CatalogSources(out.GetNamespace()).UpdateStatus(out); err != nil {
				return err
			}

			return nil
		}
	}

	srcReconciler := o.reconciler.ReconcilerForSource(catsrc)
	if srcReconciler == nil {
		// TODO: Add failure status on catalogsource and remove from sources
		return fmt.Errorf("no reconciler for source type %s", catsrc.Spec.SourceType)
	}

	healthy, err := srcReconciler.CheckRegistryServer(catsrc)
	if err != nil {
		return err
	}
	logger.Debugf("check registry server healthy: %t", healthy)

	// If registry pod hasn't been created or hasn't been updated since the last configmap update, recreate it
	if !healthy || catsrc.Status.RegistryServiceStatus == nil {
		return func() error {
			o.sourcesLock.Lock()
			defer o.sourcesLock.Unlock()

			logger.Debug("ensuring registry server")
			if err := srcReconciler.EnsureRegistryServer(out); err != nil {
				logger.WithError(err).Warn("couldn't ensure registry server")
				return err
			}
			logger.Debug("ensured registry server")

			if s, ok := o.sources[sourceKey]; ok {
				if err := s.Client.Close(); err != nil {
					logger.WithError(err).Debug("error closing client connection")
				}
			}
			delete(o.sources, sourceKey)
			o.sourcesLastUpdate = out.Status.LastSync

			logger.Debug("updating catsrc status")
			if _, err := o.client.OperatorsV1alpha1().CatalogSources(out.GetNamespace()).UpdateStatus(out); err != nil {
				return err
			}
			logger.Debug("registry server recreated")

			return nil
		}()
	}
	logger.Debug("registry state good")

	// update operator's view of sources
	sourcesUpdated := false
	func() {
		o.sourcesLock.Lock()
		defer o.sourcesLock.Unlock()
		address := catsrc.Address()
		currentSource, ok := o.sources[sourceKey]
		logger = logger.WithField("currentSource", sourceKey)

		connect := false

		// this connection is out of date, close and reconnect
		if ok && (currentSource.Address != address || catsrc.Status.LastSync.After(currentSource.LastConnect.Time)) {
			logger.Info("rebuilding connection to registry")
			if currentSource.Client != nil {
				if err := currentSource.Client.Close(); err != nil {
					logger.WithError(err).Warn("couldn't close outdated connection to registry")
					return
				}
			}
			delete(o.sources, sourceKey)
			o.sourcesLastUpdate = o.now()

			connect = true
		} else if !ok {
			// have never made a connection, so need to build a new one
			connect = true
		}

		logger := logger.WithField("address", address)
		if connect {
			logger.Info("building connection to registry")
			c, err := registryclient.NewClient(address)
			if err != nil {
				logger.WithError(err).Warn("couldn't connect to registry")
			}
			sourceRef := resolver.SourceRef{
				Address:     address,
				Client:      c,
				LastConnect: o.now(),
				LastHealthy: metav1.Time{}, // haven't detected healthy yet
			}
			o.sources[sourceKey] = sourceRef
			currentSource = sourceRef
			sourcesUpdated = true
			o.sourcesLastUpdate = sourceRef.LastConnect
		}

		if currentSource.LastHealthy.IsZero() {
			logger.Info("client hasn't yet become healthy, attempt a health check")
			healthy, err := currentSource.Client.HealthCheck(context.TODO(), 2*time.Second)
			if err != nil || !healthy {
				if registryclient.IsErrorUnrecoverable(err) {
					logger.Debug("state didn't change, trigger reconnect. this may happen when cached dns is wrong.")
					if err := currentSource.Client.Close(); err != nil {
						logger.WithError(err).Warn("couldn't close outdated connection to registry")
						return
					}
					delete(o.sources, sourceKey)
					o.sourcesLastUpdate = o.now()
				}
				if err := o.catsrcQueueSet.Requeue(sourceKey.Namespace, sourceKey.Name); err != nil {
					logger.WithError(err).Debug("error requeuing")
				}
				return
			}

			logger.Debug("client has become healthy!")
			currentSource.LastHealthy = currentSource.LastConnect
			o.sourcesLastUpdate = currentSource.LastHealthy
			o.sources[sourceKey] = currentSource
			sourcesUpdated = true
		}
	}()

	if !sourcesUpdated {
		return nil
	}

	// record that we've done work here onto the status
	out.Status.LastSync = o.now()
	if _, err := o.client.OperatorsV1alpha1().CatalogSources(out.GetNamespace()).UpdateStatus(out); err != nil {
		return err
	}

	// Trigger a resolve, will pick up any subscriptions that depend on the catalog
	o.nsResolveQueue.Add(out.GetNamespace())

	return nil
}

func (o *Operator) syncResolvingNamespace(obj interface{}) error {
	ns, ok := obj.(*corev1.Namespace)
	if !ok {
		o.logger.Debugf("wrong type: %#v", obj)
		return fmt.Errorf("casting Namespace failed")
	}
	namespace := ns.GetName()

	logger := o.logger.WithFields(logrus.Fields{
		"namespace": namespace,
		"id":        queueinformer.NewLoopID(),
	})

	// get the set of sources that should be used for resolution and best-effort get their connections working
	resolverSources := o.ensureResolverSources(logger, namespace)
	logger.Debugf("resolved sources: %#v", resolverSources)
	querier := resolver.NewNamespaceSourceQuerier(resolverSources)

	logger.Debug("checking if subscriptions need update")

	subs, err := o.lister.OperatorsV1alpha1().SubscriptionLister().Subscriptions(namespace).List(labels.Everything())
	if err != nil {
		logger.WithError(err).Debug("couldn't list subscriptions")
		return err
	}

	// TODO: parallel
	subscriptionUpdated := false
	for _, sub := range subs {
		logger := logger.WithFields(logrus.Fields{
			"sub":     sub.GetName(),
			"source":  sub.Spec.CatalogSource,
			"pkg":     sub.Spec.Package,
			"channel": sub.Spec.Channel,
		})

		// ensure the installplan reference is correct
		sub, changedIP, err := o.ensureSubscriptionInstallPlanState(logger, sub)
		if err != nil {
			return err
		}
		subscriptionUpdated = subscriptionUpdated || changedIP

		// record the current state of the desired corresponding CSV in the status. no-op if we don't know the csv yet.
		sub, changedCSV, err := o.ensureSubscriptionCSVState(logger, sub, querier)
		if err != nil {
			return err
		}

		subscriptionUpdated = subscriptionUpdated || changedCSV
	}
	if subscriptionUpdated {
		logger.Debug("subscriptions were updated, wait for a new resolution")
		return nil
	}

	shouldUpdate := false
	for _, sub := range subs {
		shouldUpdate = shouldUpdate || !o.nothingToUpdate(logger, sub)
	}
	if !shouldUpdate {
		logger.Debug("all subscriptions up to date")
		return nil
	}

	logger.Debug("resolving subscriptions in namespace")

	// resolve a set of steps to apply to a cluster, a set of subscriptions to create/update, and any errors
	steps, updatedSubs, err := o.resolver.ResolveSteps(namespace, querier)
	if err != nil {
		return err
	}

	// create installplan if anything updated
	if len(updatedSubs) > 0 {
		logger.Debug("resolution caused subscription changes, creating installplan")
		// any subscription in the namespace with manual approval will force generated installplans to be manual
		// TODO: this is an odd artifact of the older resolver, and will probably confuse users. approval mode could be on the operatorgroup?
		installPlanApproval := v1alpha1.ApprovalAutomatic
		for _, sub := range subs {
			if sub.Spec.InstallPlanApproval == v1alpha1.ApprovalManual {
				installPlanApproval = v1alpha1.ApprovalManual
				break
			}
		}

		installPlanReference, err := o.ensureInstallPlan(logger, namespace, subs, installPlanApproval, steps)
		if err != nil {
			logger.WithError(err).Debug("error ensuring installplan")
			return err
		}
		if err := o.updateSubscriptionStatus(namespace, updatedSubs, installPlanReference); err != nil {
			logger.WithError(err).Debug("error ensuring subscription installplan state")
			return err
		}
		return nil
	}

	return nil
}

func (o *Operator) syncSubscriptions(obj interface{}) error {
	sub, ok := obj.(*v1alpha1.Subscription)
	if !ok {
		o.logger.Debugf("wrong type: %#v", obj)
		return fmt.Errorf("casting Subscription failed")
	}

	o.nsResolveQueue.Add(sub.GetNamespace())

	return nil
}

func (o *Operator) ensureResolverSources(logger *logrus.Entry, namespace string) map[resolver.CatalogKey]registryclient.Interface {
	// TODO: record connection status onto an object
	resolverSources := map[resolver.CatalogKey]registryclient.Interface{}
	func() {
		o.sourcesLock.RLock()
		defer o.sourcesLock.RUnlock()
		for k, ref := range o.sources {
			if ref.LastHealthy.IsZero() {
				logger = logger.WithField("source", k)
				logger.Debug("omitting source, hasn't yet become healthy")
				if err := o.catsrcQueueSet.Requeue(k.Namespace, k.Name); err != nil {
					logger.Warn("error requeueing")
				}
				continue
			}
			// only resolve in namespace local + global catalogs
			if k.Namespace == namespace || k.Namespace == o.namespace {
				resolverSources[k] = ref.Client
			}
		}
	}()

	for k, s := range resolverSources {
		logger = logger.WithField("resolverSource", k)
		if healthy, err := s.HealthCheck(context.TODO(), 2*time.Second); err != nil || !healthy {
			logger.WithError(err).Debug("omitting unhealthy source")
			if err := o.catsrcQueueSet.Requeue(k.Namespace, k.Name); err != nil {
				logger.Warn("error requeueing")
			}
			delete(resolverSources, k)
		}
	}

	return resolverSources
}

func (o *Operator) nothingToUpdate(logger *logrus.Entry, sub *v1alpha1.Subscription) bool {
	o.sourcesLock.RLock()
	defer o.sourcesLock.RUnlock()

	// Only sync if catalog has been updated since last sync time
	if o.sourcesLastUpdate.Before(&sub.Status.LastUpdated) && sub.Status.State != v1alpha1.SubscriptionStateNone && sub.Status.State != v1alpha1.SubscriptionStateUpgradeAvailable {
		logger.Debugf("skipping update: no new updates to catalog since last sync at %s", sub.Status.LastUpdated.String())
		return true
	}
	if sub.Status.InstallPlanRef != nil && sub.Status.State == v1alpha1.SubscriptionStateUpgradePending {
		logger.Debugf("skipping update: installplan already created")
		return true
	}
	return false
}

func (o *Operator) ensureSubscriptionInstallPlanState(logger *logrus.Entry, sub *v1alpha1.Subscription) (*v1alpha1.Subscription, bool, error) {
	if sub.Status.InstallPlanRef != nil {
		return sub, false, nil
	}

	logger.Debug("checking for existing installplan")

	// check if there's an installplan that created this subscription (only if it doesn't have a reference yet)
	// this indicates it was newly resolved by another operator, and we should reference that installplan in the status
	ipName, ok := sub.GetAnnotations()[generatedByKey]
	if !ok {
		return sub, false, nil
	}

	ip, err := o.lister.OperatorsV1alpha1().InstallPlanLister().InstallPlans(sub.GetNamespace()).Get(ipName)
	if err != nil {
		logger.WithField("installplan", ipName).Warn("unable to get installplan from cache")
		return nil, false, err
	}
	logger.WithField("installplan", ipName).Debug("found installplan that generated subscription")

	out := sub.DeepCopy()
	ref, err := reference.GetReference(ip)
	if err != nil {
		logger.WithError(err).Warn("unable to generate installplan reference")
		return nil, false, err
	}
	out.Status.InstallPlanRef = ref
	out.Status.Install = v1alpha1.NewInstallPlanReference(ref)
	out.Status.State = v1alpha1.SubscriptionStateUpgradePending
	out.Status.CurrentCSV = out.Spec.StartingCSV
	out.Status.LastUpdated = o.now()

	updated, err := o.client.OperatorsV1alpha1().Subscriptions(sub.GetNamespace()).UpdateStatus(out)
	if err != nil {
		return nil, false, err
	}

	return updated, true, nil
}

func (o *Operator) ensureSubscriptionCSVState(logger *logrus.Entry, sub *v1alpha1.Subscription, querier resolver.SourceQuerier) (*v1alpha1.Subscription, bool, error) {
	if sub.Status.CurrentCSV == "" {
		return sub, false, nil
	}

	csv, err := o.client.OperatorsV1alpha1().ClusterServiceVersions(sub.GetNamespace()).Get(sub.Status.CurrentCSV, metav1.GetOptions{})
	out := sub.DeepCopy()
	if err != nil {
		logger.WithError(err).WithField("currentCSV", sub.Status.CurrentCSV).Debug("error fetching csv listed in subscription status")
		out.Status.State = v1alpha1.SubscriptionStateUpgradePending
	} else {
		// Check if an update is available for the current csv
		if err := querier.Queryable(); err != nil {
			return nil, false, err
		}
		bundle, _, _ := querier.FindReplacement(&csv.Spec.Version.Version, sub.Status.CurrentCSV, sub.Spec.Package, sub.Spec.Channel, resolver.CatalogKey{Name: sub.Spec.CatalogSource, Namespace: sub.Spec.CatalogSourceNamespace})
		if bundle != nil {
			o.logger.Tracef("replacement %s bundle found for current bundle %s", bundle.Name, sub.Status.CurrentCSV)
			out.Status.State = v1alpha1.SubscriptionStateUpgradeAvailable
		} else {
			out.Status.State = v1alpha1.SubscriptionStateAtLatest
		}

		out.Status.InstalledCSV = sub.Status.CurrentCSV
	}

	if sub.Status.State == out.Status.State {
		// The subscription status represents the cluster state
		return sub, false, nil
	}
	out.Status.LastUpdated = o.now()

	// Update Subscription with status of transition. Log errors if we can't write them to the status.
	updatedSub, err := o.client.OperatorsV1alpha1().Subscriptions(out.GetNamespace()).UpdateStatus(out)
	if err != nil {
		logger.WithError(err).Info("error updating subscription status")
		return nil, false, fmt.Errorf("error updating Subscription status: " + err.Error())
	}

	// subscription status represents cluster state
	return updatedSub, true, nil
}

func (o *Operator) updateSubscriptionStatus(namespace string, subs []*v1alpha1.Subscription, installPlanRef *corev1.ObjectReference) error {
	// TODO: parallel, sync waitgroup
	var err error
	for _, sub := range subs {
		sub.Status.LastUpdated = o.now()
		if installPlanRef != nil {
			sub.Status.InstallPlanRef = installPlanRef
			sub.Status.Install = v1alpha1.NewInstallPlanReference(installPlanRef)
			sub.Status.State = v1alpha1.SubscriptionStateUpgradePending
		}
		if _, subErr := o.client.OperatorsV1alpha1().Subscriptions(namespace).UpdateStatus(sub); subErr != nil {
			err = subErr
		}
	}
	return err
}

func (o *Operator) ensureInstallPlan(logger *logrus.Entry, namespace string, subs []*v1alpha1.Subscription, installPlanApproval v1alpha1.Approval, steps []*v1alpha1.Step) (*corev1.ObjectReference, error) {
	if len(steps) == 0 {
		return nil, nil
	}

	// Check if any existing installplans are creating the same resources
	installPlans, err := o.lister.OperatorsV1alpha1().InstallPlanLister().InstallPlans(namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	for _, installPlan := range installPlans {
		if installPlan.Status.CSVManifestsMatch(steps) {
			logger.Infof("found InstallPlan with matching manifests: %s", installPlan.GetName())
			return reference.GetReference(installPlan)
		}
	}
	logger.Warn("no installplan found with matching manifests, creating new one")

	return o.createInstallPlan(namespace, subs, installPlanApproval, steps)
}

func (o *Operator) createInstallPlan(namespace string, subs []*v1alpha1.Subscription, installPlanApproval v1alpha1.Approval, steps []*v1alpha1.Step) (*corev1.ObjectReference, error) {
	if len(steps) == 0 {
		return nil, nil
	}

	csvNames := []string{}
	catalogSourceMap := map[string]struct{}{}
	for _, s := range steps {
		if s.Resource.Kind == "ClusterServiceVersion" {
			csvNames = append(csvNames, s.Resource.Name)
		}
		catalogSourceMap[s.Resource.CatalogSource] = struct{}{}
	}
	catalogSources := []string{}
	for s := range catalogSourceMap {
		catalogSources = append(catalogSources, s)
	}

	phase := v1alpha1.InstallPlanPhaseInstalling
	if installPlanApproval == v1alpha1.ApprovalManual {
		phase = v1alpha1.InstallPlanPhaseRequiresApproval
	}
	ip := &v1alpha1.InstallPlan{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "install-",
			Namespace:    namespace,
		},
		Spec: v1alpha1.InstallPlanSpec{
			ClusterServiceVersionNames: csvNames,
			Approval:                   installPlanApproval,
			Approved:                   installPlanApproval == v1alpha1.ApprovalAutomatic,
		},
	}
	for _, sub := range subs {
		ownerutil.AddNonBlockingOwner(ip, sub)
	}

	res, err := o.client.OperatorsV1alpha1().InstallPlans(namespace).Create(ip)
	if err != nil {
		return nil, err
	}

	res.Status = v1alpha1.InstallPlanStatus{
		Phase:          phase,
		Plan:           steps,
		CatalogSources: catalogSources,
	}
	res, err = o.client.OperatorsV1alpha1().InstallPlans(namespace).UpdateStatus(res)
	if err != nil {
		return nil, err
	}

	return reference.GetReference(res)
}

func (o *Operator) syncInstallPlans(obj interface{}) (syncError error) {
	plan, ok := obj.(*v1alpha1.InstallPlan)
	if !ok {
		o.logger.Debugf("wrong type: %#v", obj)
		return fmt.Errorf("casting InstallPlan failed")
	}

	logger := o.logger.WithFields(logrus.Fields{
		"id":        queueinformer.NewLoopID(),
		"ip":        plan.GetName(),
		"namespace": plan.GetNamespace(),
		"phase":     plan.Status.Phase,
	})

	logger.Info("syncing")

	if len(plan.Status.Plan) == 0 {
		logger.Info("skip processing installplan without status - subscription sync responsible for initial status")
		return
	}

	outInstallPlan, syncError := transitionInstallPlanState(logger.Logger, o, *plan, o.now())

	if syncError != nil {
		logger = logger.WithField("syncError", syncError)
	}

	// no changes in status, don't update
	if outInstallPlan.Status.Phase == plan.Status.Phase {
		return
	}

	defer func() {
		// Notify subscription loop of installplan changes
		if owners := ownerutil.GetOwnersByKind(plan, v1alpha1.SubscriptionKind); len(owners) > 0 {
			for _, owner := range owners {
				logger.WithField("owner", owner).Debug("requeueing installplan owner")
				o.subQueueSet.Requeue(plan.GetNamespace(), owner.Name)
			}
		} else {
			logger.Trace("no installplan owner subscriptions found to requeue")
		}
	}()

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

func transitionInstallPlanState(log *logrus.Logger, transitioner installPlanTransitioner, in v1alpha1.InstallPlan, now metav1.Time) (*v1alpha1.InstallPlan, error) {
	out := in.DeepCopy()

	switch in.Status.Phase {
	case v1alpha1.InstallPlanPhaseRequiresApproval:
		if out.Spec.Approved {
			log.Debugf("approved, setting to %s", v1alpha1.InstallPlanPhasePlanning)
			out.Status.Phase = v1alpha1.InstallPlanPhaseInstalling
		} else {
			log.Debug("not approved, skipping sync")
		}
		return out, nil

	case v1alpha1.InstallPlanPhaseInstalling:
		log.Debug("attempting to install")
		if err := transitioner.ExecutePlan(out); err != nil {
			out.Status.SetCondition(v1alpha1.ConditionFailed(v1alpha1.InstallPlanInstalled,
				v1alpha1.InstallPlanReasonComponentFailed, err.Error(), &now))
			out.Status.Phase = v1alpha1.InstallPlanPhaseFailed
			return out, err
		}
		out.Status.SetCondition(v1alpha1.ConditionMet(v1alpha1.InstallPlanInstalled, &now))
		out.Status.Phase = v1alpha1.InstallPlanPhaseComplete
		return out, nil
	default:
		return out, nil
	}
}

// ResolvePlan modifies an InstallPlan to contain a Plan in its Status field.
func (o *Operator) ResolvePlan(plan *v1alpha1.InstallPlan) error {
	return nil
}

// Ensure all existing versions are present in new CRD
func ensureCRDVersions(oldCRD *v1beta1ext.CustomResourceDefinition, newCRD *v1beta1ext.CustomResourceDefinition) error {
	for _, oldVersion := range oldCRD.Spec.Versions {
		var versionPresent bool
		for _, newVersion := range newCRD.Spec.Versions {
			if oldVersion.Name == newVersion.Name {
				versionPresent = true
			}
		}
		if !versionPresent {
			return fmt.Errorf("not allowing CRD (%v) update with unincluded version %v", newCRD.GetName(), oldVersion)
		}
	}

	return nil
}

func (o *Operator) validateCustomResourceDefinition(oldCRD *v1beta1ext.CustomResourceDefinition, newCRD *v1beta1ext.CustomResourceDefinition) error {
	o.logger.Debugf("Comparing %#v to %#v", oldCRD.Spec.Validation, newCRD.Spec.Validation)
	// If validation schema is unchanged, return right away
	if reflect.DeepEqual(oldCRD.Spec.Validation, newCRD.Spec.Validation) {
		return nil
	}
	convertedCRD := &apiextensions.CustomResourceDefinition{}
	if err := v1beta1ext.Convert_v1beta1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(newCRD, convertedCRD, nil); err != nil {
		return err
	}
	for _, oldVersion := range oldCRD.Spec.Versions {
		gvr := schema.GroupVersionResource{Group: oldCRD.Spec.Group, Version: oldVersion.Name, Resource: oldCRD.Spec.Names.Plural}
		err := o.validateExistingCRs(gvr, convertedCRD)
		if err != nil {
			return err
		}
	}

	if oldCRD.Spec.Version != "" {
		gvr := schema.GroupVersionResource{Group: oldCRD.Spec.Group, Version: oldCRD.Spec.Version, Resource: oldCRD.Spec.Names.Plural}
		err := o.validateExistingCRs(gvr, convertedCRD)
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *Operator) validateExistingCRs(gvr schema.GroupVersionResource, newCRD *apiextensions.CustomResourceDefinition) error {
	crList, err := o.dynamicClient.Resource(gvr).List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing resources in GroupVersionResource %#v: %s", gvr, err)
	}
	for _, cr := range crList.Items {
		validator, _, err := validation.NewSchemaValidator(newCRD.Spec.Validation)
		if err != nil {
			return fmt.Errorf("error creating validator for schema %#v: %s", newCRD.Spec.Validation, err)
		}
		err = validation.ValidateCustomResource(cr.UnstructuredContent(), validator)
		if err != nil {
			return fmt.Errorf("error validating custom resource against new schema %#v: %s", newCRD.Spec.Validation, err)
		}
	}

	return nil
}

// ExecutePlan applies a planned InstallPlan to a namespace.
func (o *Operator) ExecutePlan(plan *v1alpha1.InstallPlan) error {
	if plan.Status.Phase != v1alpha1.InstallPlanPhaseInstalling {
		panic("attempted to install a plan that wasn't in the installing phase")
	}

	namespace := plan.GetNamespace()

	// Get the set of initial installplan csv names
	initialCSVNames := getCSVNameSet(plan)
	// Get pre-existing CRD owners to make decisions about applying resolved CSVs
	existingCRDOwners, err := o.getExistingApiOwners(plan.GetNamespace())
	if err != nil {
		return err
	}

	// Does the namespace have an operator group that specifies a user defined
	// service account? If so, then we should use a scoped client for plan
	// execution.
	getter := o.serviceAccountQuerier.NamespaceQuerier(namespace)
	kubeclient, crclient, err := o.clientAttenuator.AttenuateClient(getter)
	if err != nil {
		o.logger.Errorf("failed to get a client for plan execution- %v", err)
		return err
	}

	ensurer := newStepEnsurer(kubeclient, crclient)

	for i, step := range plan.Status.Plan {
		switch step.Status {
		case v1alpha1.StepStatusPresent, v1alpha1.StepStatusCreated:
			continue

		case v1alpha1.StepStatusUnknown, v1alpha1.StepStatusNotPresent:
			o.logger.WithFields(logrus.Fields{"kind": step.Resource.Kind, "name": step.Resource.Name}).Debug("execute resource")
			switch step.Resource.Kind {
			case crdKind:
				// Marshal the manifest into a CRD instance.
				var crd v1beta1ext.CustomResourceDefinition
				err := json.Unmarshal([]byte(step.Resource.Manifest), &crd)
				if err != nil {
					return errorwrap.Wrapf(err, "error parsing step manifest: %s", step.Resource.Name)
				}

				// TODO: check that names are accepted
				// Attempt to create the CRD.
				_, err = o.opClient.ApiextensionsV1beta1Interface().ApiextensionsV1beta1().CustomResourceDefinitions().Create(&crd)
				if k8serrors.IsAlreadyExists(err) {
					currentCRD, _ := o.lister.APIExtensionsV1beta1().CustomResourceDefinitionLister().Get(crd.GetName())
					// Compare 2 CRDs to see if it needs to be updatetd
					if !reflect.DeepEqual(crd, *currentCRD) {
						// Verify CRD ownership, only attempt to update if
						// CRD has only one owner
						// Example: provided=database.coreos.com/v1alpha1/EtcdCluster
						matchedCSV, err := index.CRDProviderNames(o.csvProvidedAPIsIndexer, crd)
						if err != nil {
							return errorwrap.Wrapf(err, "error find matched CSV: %s", step.Resource.Name)
						}
						crd.SetResourceVersion(currentCRD.GetResourceVersion())
						if len(matchedCSV) == 1 {
							o.logger.Debugf("Found one owner for CRD %v", crd)

							_, err = o.opClient.ApiextensionsV1beta1Interface().ApiextensionsV1beta1().CustomResourceDefinitions().Update(&crd)
							if err != nil {
								return errorwrap.Wrapf(err, "error updating CRD: %s", step.Resource.Name)
							}
						} else if len(matchedCSV) > 1 {
							o.logger.Debugf("Found multiple owners for CRD %v", crd)

							if err := ensureCRDVersions(currentCRD, &crd); err != nil {
								return errorwrap.Wrapf(err, "error missing existing CRD version(s) in new CRD: %s", step.Resource.Name)
							}

							if err = o.validateCustomResourceDefinition(currentCRD, &crd); err != nil {
								return errorwrap.Wrapf(err, "error validating existing CRs agains new CRD's schema: %s", step.Resource.Name)
							}

							_, err = o.opClient.ApiextensionsV1beta1Interface().ApiextensionsV1beta1().CustomResourceDefinitions().Update(&crd)
							if err != nil {
								return errorwrap.Wrapf(err, "error update CRD: %s", step.Resource.Name)
							}
						}
					}
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
					return errorwrap.Wrapf(err, "error parsing step manifest: %s", step.Resource.Name)
				}

				// Check if the resolved CSV is in the initial set
				if _, ok := initialCSVNames[csv.GetName()]; !ok {
					// Check for pre-existing CSVs that own the same CRDs
					competingOwners, err := competingCRDOwnersExist(plan.GetNamespace(), &csv, existingCRDOwners)
					if err != nil {
						return errorwrap.Wrapf(err, "error checking crd owners for: %s", csv.GetName())
					}

					// TODO: decide on fail/continue logic for pre-existing dependent CSVs that own the same CRD(s)
					if competingOwners {
						// For now, error out
						return fmt.Errorf("pre-existing CRD owners found for owned CRD(s) of dependent CSV %s", csv.GetName())
					}
				}

				// Attempt to create the CSV.
				csv.SetNamespace(namespace)

				status, err := ensurer.EnsureClusterServiceVersion(&csv)
				if err != nil {
					return err
				}

				plan.Status.Plan[i].Status = status

			case v1alpha1.SubscriptionKind:
				// Marshal the manifest into a subscription instance.
				var sub v1alpha1.Subscription
				err := json.Unmarshal([]byte(step.Resource.Manifest), &sub)
				if err != nil {
					return errorwrap.Wrapf(err, "error parsing step manifest: %s", step.Resource.Name)
				}

				// Add the InstallPlan's name as an annotation
				if annotations := sub.GetAnnotations(); annotations != nil {
					annotations[generatedByKey] = plan.GetName()
				} else {
					sub.SetAnnotations(map[string]string{generatedByKey: plan.GetName()})
				}

				// Attempt to create the Subscription
				sub.SetNamespace(namespace)

				status, err := ensurer.EnsureSubscription(&sub)
				if err != nil {
					return err
				}

				plan.Status.Plan[i].Status = status

			case secretKind:
				status, err := ensurer.EnsureSecret(o.namespace, plan.GetNamespace(), step.Resource.Name)
				if err != nil {
					return err
				}

				plan.Status.Plan[i].Status = status

			case clusterRoleKind:
				// Marshal the manifest into a ClusterRole instance.
				var cr rbacv1.ClusterRole
				err := json.Unmarshal([]byte(step.Resource.Manifest), &cr)
				if err != nil {
					return errorwrap.Wrapf(err, "error parsing step manifest: %s", step.Resource.Name)
				}

				status, err := ensurer.EnsureClusterRole(&cr, step)
				if err != nil {
					return err
				}

				plan.Status.Plan[i].Status = status

			case clusterRoleBindingKind:
				// Marshal the manifest into a RoleBinding instance.
				var rb rbacv1.ClusterRoleBinding
				err := json.Unmarshal([]byte(step.Resource.Manifest), &rb)
				if err != nil {
					return errorwrap.Wrapf(err, "error parsing step manifest: %s", step.Resource.Name)
				}

				status, err := ensurer.EnsureClusterRoleBinding(&rb, step)
				if err != nil {
					return err
				}

				plan.Status.Plan[i].Status = status

			case roleKind:
				// Marshal the manifest into a Role instance.
				var r rbacv1.Role
				err := json.Unmarshal([]byte(step.Resource.Manifest), &r)
				if err != nil {
					return errorwrap.Wrapf(err, "error parsing step manifest: %s", step.Resource.Name)
				}

				// Update UIDs on all CSV OwnerReferences
				updated, err := o.getUpdatedOwnerReferences(r.OwnerReferences, plan.Namespace)
				if err != nil {
					return errorwrap.Wrapf(err, "error generating ownerrefs for role %s", r.GetName())
				}
				r.SetOwnerReferences(updated)
				r.SetNamespace(namespace)

				status, err := ensurer.EnsureRole(plan.Namespace, &r)
				if err != nil {
					return err
				}

				plan.Status.Plan[i].Status = status

			case roleBindingKind:
				// Marshal the manifest into a RoleBinding instance.
				var rb rbacv1.RoleBinding
				err := json.Unmarshal([]byte(step.Resource.Manifest), &rb)
				if err != nil {
					return errorwrap.Wrapf(err, "error parsing step manifest: %s", step.Resource.Name)
				}

				// Update UIDs on all CSV OwnerReferences
				updated, err := o.getUpdatedOwnerReferences(rb.OwnerReferences, plan.Namespace)
				if err != nil {
					return errorwrap.Wrapf(err, "error generating ownerrefs for rolebinding %s", rb.GetName())
				}
				rb.SetOwnerReferences(updated)
				rb.SetNamespace(namespace)

				status, err := ensurer.EnsureRoleBinding(plan.Namespace, &rb)
				if err != nil {
					return err
				}

				plan.Status.Plan[i].Status = status

			case serviceAccountKind:
				// Marshal the manifest into a ServiceAccount instance.
				var sa corev1.ServiceAccount
				err := json.Unmarshal([]byte(step.Resource.Manifest), &sa)
				if err != nil {
					return errorwrap.Wrapf(err, "error parsing step manifest: %s", step.Resource.Name)
				}

				// Update UIDs on all CSV OwnerReferences
				updated, err := o.getUpdatedOwnerReferences(sa.OwnerReferences, plan.Namespace)
				if err != nil {
					return errorwrap.Wrapf(err, "error generating ownerrefs for service account: %s", sa.GetName())
				}
				sa.SetOwnerReferences(updated)
				sa.SetNamespace(namespace)

				status, err := ensurer.EnsureServiceAccount(namespace, &sa)
				if err != nil {
					return err
				}

				plan.Status.Plan[i].Status = status

			case serviceKind:
				// Marshal the manifest into a Service instance
				var s corev1.Service
				err := json.Unmarshal([]byte(step.Resource.Manifest), &s)
				if err != nil {
					return errorwrap.Wrapf(err, "error parsing step manifest: %s", step.Resource.Name)
				}

				// Update UIDs on all CSV OwnerReferences
				updated, err := o.getUpdatedOwnerReferences(s.OwnerReferences, plan.Namespace)
				if err != nil {
					return errorwrap.Wrapf(err, "error generating ownerrefs for service: %s", s.GetName())
				}
				s.SetOwnerReferences(updated)
				s.SetNamespace(namespace)

				status, err := ensurer.EnsureService(namespace, &s)
				if err != nil {
					return err
				}

				plan.Status.Plan[i].Status = status

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

// getExistingApiOwners creates a map of CRD names to existing owner CSVs in the given namespace
func (o *Operator) getExistingApiOwners(namespace string) (map[string][]string, error) {
	// Get a list of CSVs in the namespace
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
		for _, api := range csv.Spec.APIServiceDefinitions.Owned {
			owners[api.Group] = append(owners[api.Group], csv.GetName())
		}
	}

	return owners, nil
}

func (o *Operator) getUpdatedOwnerReferences(refs []metav1.OwnerReference, namespace string) ([]metav1.OwnerReference, error) {
	updated := append([]metav1.OwnerReference(nil), refs...)

	for i, owner := range refs {
		if owner.Kind == v1alpha1.ClusterServiceVersionKind {
			csv, err := o.client.OperatorsV1alpha1().ClusterServiceVersions(namespace).Get(owner.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			owner.UID = csv.GetUID()
			updated[i] = owner
		}
	}
	return updated, nil
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
