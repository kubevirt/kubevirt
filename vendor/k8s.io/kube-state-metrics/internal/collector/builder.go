/*
Copyright 2018 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package collector

import (
	"sort"
	"strings"

	"k8s.io/klog"

	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v2beta1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	certv1beta1 "k8s.io/api/certificates/v1beta1"
	v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	policy "k8s.io/api/policy/v1beta1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	coll "k8s.io/kube-state-metrics/pkg/collector"
	"k8s.io/kube-state-metrics/pkg/metric"
	metricsstore "k8s.io/kube-state-metrics/pkg/metrics_store"
	"k8s.io/kube-state-metrics/pkg/options"
)

type whiteBlackLister interface {
	IsIncluded(string) bool
	IsExcluded(string) bool
}

// Builder helps to build collector. It follows the builder pattern
// (https://en.wikipedia.org/wiki/Builder_pattern).
type Builder struct {
	kubeClient        clientset.Interface
	namespaces        options.NamespaceList
	ctx               context.Context
	enabledCollectors []string
	whiteBlackList    whiteBlackLister
}

// NewBuilder returns a new builder.
func NewBuilder(
	ctx context.Context,
) *Builder {
	return &Builder{
		ctx: ctx,
	}
}

// WithEnabledCollectors sets the enabledCollectors property of a Builder.
func (b *Builder) WithEnabledCollectors(c []string) {
	var copy []string
	copy = append(copy, c...)

	sort.Strings(copy)

	b.enabledCollectors = copy
}

// WithNamespaces sets the namespaces property of a Builder.
func (b *Builder) WithNamespaces(n options.NamespaceList) {
	b.namespaces = n
}

// WithKubeClient sets the kubeClient property of a Builder.
func (b *Builder) WithKubeClient(c clientset.Interface) {
	b.kubeClient = c
}

// WithWhiteBlackList configures the white or blacklisted metric to be exposed
// by the collector build by the Builder
func (b *Builder) WithWhiteBlackList(l whiteBlackLister) {
	b.whiteBlackList = l
}

// Build initializes and registers all enabled collectors.
func (b *Builder) Build() []*coll.Collector {
	if b.whiteBlackList == nil {
		panic("whiteBlackList should not be nil")
	}

	collectors := []*coll.Collector{}
	activeCollectorNames := []string{}

	for _, c := range b.enabledCollectors {
		constructor, ok := availableCollectors[c]
		if ok {
			collector := constructor(b)
			activeCollectorNames = append(activeCollectorNames, c)
			collectors = append(collectors, collector)
		}
	}

	klog.Infof("Active collectors: %s", strings.Join(activeCollectorNames, ","))

	return collectors
}

var availableCollectors = map[string]func(f *Builder) *coll.Collector{
	"certificatesigningrequests": func(b *Builder) *coll.Collector { return b.buildCsrCollector() },
	"configmaps":                 func(b *Builder) *coll.Collector { return b.buildConfigMapCollector() },
	"cronjobs":                   func(b *Builder) *coll.Collector { return b.buildCronJobCollector() },
	"daemonsets":                 func(b *Builder) *coll.Collector { return b.buildDaemonSetCollector() },
	"deployments":                func(b *Builder) *coll.Collector { return b.buildDeploymentCollector() },
	"endpoints":                  func(b *Builder) *coll.Collector { return b.buildEndpointsCollector() },
	"horizontalpodautoscalers":   func(b *Builder) *coll.Collector { return b.buildHPACollector() },
	"ingresses":                  func(b *Builder) *coll.Collector { return b.buildIngressCollector() },
	"jobs":                       func(b *Builder) *coll.Collector { return b.buildJobCollector() },
	"limitranges":                func(b *Builder) *coll.Collector { return b.buildLimitRangeCollector() },
	"namespaces":                 func(b *Builder) *coll.Collector { return b.buildNamespaceCollector() },
	"nodes":                      func(b *Builder) *coll.Collector { return b.buildNodeCollector() },
	"persistentvolumeclaims":     func(b *Builder) *coll.Collector { return b.buildPersistentVolumeClaimCollector() },
	"persistentvolumes":          func(b *Builder) *coll.Collector { return b.buildPersistentVolumeCollector() },
	"poddisruptionbudgets":       func(b *Builder) *coll.Collector { return b.buildPodDisruptionBudgetCollector() },
	"pods":                       func(b *Builder) *coll.Collector { return b.buildPodCollector() },
	"replicasets":                func(b *Builder) *coll.Collector { return b.buildReplicaSetCollector() },
	"replicationcontrollers":     func(b *Builder) *coll.Collector { return b.buildReplicationControllerCollector() },
	"resourcequotas":             func(b *Builder) *coll.Collector { return b.buildResourceQuotaCollector() },
	"secrets":                    func(b *Builder) *coll.Collector { return b.buildSecretCollector() },
	"services":                   func(b *Builder) *coll.Collector { return b.buildServiceCollector() },
	"statefulsets":               func(b *Builder) *coll.Collector { return b.buildStatefulSetCollector() },
}

func (b *Builder) buildConfigMapCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, configMapMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.ConfigMap{}, store, b.namespaces, createConfigMapListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildCronJobCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, cronJobMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &batchv1beta1.CronJob{}, store, b.namespaces, createCronJobListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildDaemonSetCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, daemonSetMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &appsv1.DaemonSet{}, store, b.namespaces, createDaemonSetListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildDeploymentCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, deploymentMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &appsv1.Deployment{}, store, b.namespaces, createDeploymentListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildEndpointsCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, endpointMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.Endpoints{}, store, b.namespaces, createEndpointsListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildHPACollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, hpaMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &autoscaling.HorizontalPodAutoscaler{}, store, b.namespaces, createHPAListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildIngressCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, ingressMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &extensions.Ingress{}, store, b.namespaces, createIngressListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildJobCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, jobMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &batchv1.Job{}, store, b.namespaces, createJobListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildLimitRangeCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, limitRangeMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.LimitRange{}, store, b.namespaces, createLimitRangeListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildNamespaceCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, namespaceMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.Namespace{}, store, b.namespaces, createNamespaceListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildNodeCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, nodeMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.Node{}, store, b.namespaces, createNodeListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildPersistentVolumeClaimCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, persistentVolumeClaimMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.PersistentVolumeClaim{}, store, b.namespaces, createPersistentVolumeClaimListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildPersistentVolumeCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, persistentVolumeMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.PersistentVolume{}, store, b.namespaces, createPersistentVolumeListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildPodDisruptionBudgetCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, podDisruptionBudgetMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &policy.PodDisruptionBudget{}, store, b.namespaces, createPodDisruptionBudgetListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildReplicaSetCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, replicaSetMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &extensions.ReplicaSet{}, store, b.namespaces, createReplicaSetListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildReplicationControllerCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, replicationControllerMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.ReplicationController{}, store, b.namespaces, createReplicationControllerListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildResourceQuotaCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, resourceQuotaMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.ResourceQuota{}, store, b.namespaces, createResourceQuotaListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildSecretCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, secretMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.Secret{}, store, b.namespaces, createSecretListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildServiceCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, serviceMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.Service{}, store, b.namespaces, createServiceListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildStatefulSetCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, statefulSetMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &appsv1.StatefulSet{}, store, b.namespaces, createStatefulSetListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildPodCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, podMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.Pod{}, store, b.namespaces, createPodListWatch)

	return coll.NewCollector(store)
}

func (b *Builder) buildCsrCollector() *coll.Collector {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, csrMetricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &certv1beta1.CertificateSigningRequest{}, store, b.namespaces, createCSRListWatch)

	return coll.NewCollector(store)
}

// reflectorPerNamespace creates a Kubernetes client-go reflector with the given
// listWatchFunc for each given namespace and registers it with the given store.
func reflectorPerNamespace(
	ctx context.Context,
	kubeClient clientset.Interface,
	expectedType interface{},
	store cache.Store,
	namespaces []string,
	listWatchFunc func(kubeClient clientset.Interface, ns string) cache.ListWatch,
) {
	for _, ns := range namespaces {
		lw := listWatchFunc(kubeClient, ns)
		reflector := cache.NewReflector(&lw, expectedType, store, 0)
		go reflector.Run(ctx.Done())
	}
}
