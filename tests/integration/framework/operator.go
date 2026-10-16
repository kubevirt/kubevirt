package framework

import (
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	"kubevirt.io/kubevirt/pkg/controller"
	virtoperator "kubevirt.io/kubevirt/pkg/virt-operator"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

// Operator runs the real operator against the framework's API server.
type Operator struct {
	Leases          cache.SharedIndexInformer
	reconciliations atomic.Int64
}

// Reconciliations counts executions, including those that leave resources unchanged.
func (o *Operator) Reconciliations() int64 {
	return o.reconciliations.Load()
}

func (f *Framework) StartOperator() *Operator {
	GinkgoHelper()
	aggregator, err := aggregatorclient.NewForConfig(f.virtClient.Config())
	Expect(err).NotTo(HaveOccurred())
	factory := controller.NewKubeInformerFactory(f.virtClient.RestClient(), f.virtClient, f.k8sClient, aggregator, testNamespace)
	o := &Operator{Leases: factory.Leases()}
	informers := util.Informers{
		KubeVirt:                         &reconciliationInformer{SharedIndexInformer: factory.KubeVirt(), count: &o.reconciliations},
		ServiceAccount:                   factory.OperatorServiceAccount(),
		ClusterRole:                      factory.OperatorClusterRole(),
		ClusterRoleBinding:               factory.OperatorClusterRoleBinding(),
		Role:                             factory.OperatorRole(),
		RoleBinding:                      factory.OperatorRoleBinding(),
		OperatorCrd:                      factory.OperatorCRD(),
		Service:                          factory.OperatorService(),
		Deployment:                       factory.OperatorDeployment(),
		DaemonSet:                        factory.OperatorDaemonSet(),
		ValidationWebhook:                factory.OperatorValidationWebhook(),
		MutatingWebhook:                  factory.OperatorMutatingWebhook(),
		APIService:                       factory.OperatorAPIService(),
		InstallStrategyConfigMap:         factory.OperatorInstallStrategyConfigMaps(),
		InstallStrategyJob:               factory.OperatorInstallStrategyJob(),
		InfrastructurePod:                factory.OperatorPod(),
		PodDisruptionBudget:              factory.OperatorPodDisruptionBudget(),
		Namespace:                        factory.Namespace(),
		Secrets:                          factory.Secrets(),
		ConfigMap:                        factory.OperatorConfigMap(),
		ClusterInstancetype:              factory.VirtualMachineClusterInstancetype(),
		ClusterPreference:                factory.VirtualMachineClusterPreference(),
		Leases:                           o.Leases,
		SCC:                              factory.DummyOperatorSCC(),
		Route:                            factory.DummyOperatorRoute(),
		ServiceMonitor:                   factory.DummyOperatorServiceMonitor(),
		PrometheusRule:                   factory.DummyOperatorPrometheusRule(),
		ValidatingAdmissionPolicy:        factory.DummyOperatorValidatingAdmissionPolicy(),
		ValidatingAdmissionPolicyBinding: factory.DummyOperatorValidatingAdmissionPolicyBinding(),
	}
	ctrl, err := virtoperator.NewKubeVirtController(f.virtClient, f.k8sClient,
		aggregator.ApiregistrationV1().APIServices(), new(record.FakeRecorder), util.OperatorConfig{}, informers, testNamespace)
	Expect(err).NotTo(HaveOccurred())

	var syncs []cache.InformerSynced
	for _, informer := range []cache.SharedIndexInformer{
		informers.KubeVirt, informers.ServiceAccount, informers.ClusterRole, informers.ClusterRoleBinding,
		informers.Role, informers.RoleBinding, informers.OperatorCrd, informers.Service,
		informers.Deployment, informers.DaemonSet, informers.ValidationWebhook, informers.MutatingWebhook,
		informers.APIService, informers.InstallStrategyConfigMap, informers.InstallStrategyJob,
		informers.InfrastructurePod, informers.PodDisruptionBudget, informers.Namespace,
		informers.Secrets, informers.ConfigMap, informers.ClusterInstancetype, informers.ClusterPreference,
		informers.Leases, informers.SCC, informers.Route, informers.ServiceMonitor, informers.PrometheusRule,
		informers.ValidatingAdmissionPolicy, informers.ValidatingAdmissionPolicyBinding,
	} {
		syncs = append(syncs, informer.HasSynced)
		f.wg.Go(func() { informer.RunWithContext(f.ctx) })
	}
	Expect(cache.WaitForCacheSync(f.ctx.Done(), syncs...)).To(BeTrue())
	// Run owns queue shutdown. Manage the worker separately so Stop also joins
	// any in-flight reconciliation before shutting down the API server.
	f.wg.Go(func() { ctrl.Run(0, f.ctx.Done()) })
	f.wg.Go(func() {
		for ctrl.Execute() {
		}
	})
	return o
}

type reconciliationInformer struct {
	cache.SharedIndexInformer
	count *atomic.Int64
}

func (i *reconciliationInformer) GetStore() cache.Store {
	return &reconciliationStore{Store: i.SharedIndexInformer.GetStore(), count: i.count}
}

type reconciliationStore struct {
	cache.Store
	count *atomic.Int64
}

func (s *reconciliationStore) GetByKey(key string) (item interface{}, exists bool, err error) {
	// execute reads the KubeVirt from this store once per queue item. Preserve
	// the real cache and reconciliation behavior while observing every run.
	s.count.Add(1)
	return s.Store.GetByKey(key)
}
