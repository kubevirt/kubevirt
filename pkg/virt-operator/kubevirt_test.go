/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package virt_operator

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"

	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	secv1 "github.com/openshift/api/security/v1"
	secv1fake "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1/fake"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	k8sv1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extclientfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	v1 "kubevirt.io/client-go/api/v1"
	promclientfake "kubevirt.io/client-go/generated/prometheus-operator/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/version"
	kubecontroller "kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/apply"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	install "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/rbac"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

const (
	Added   = "added"
	Updated = "updated"
	Patched = "patched"
	Deleted = "deleted"

	NAMESPACE = "kubevirt-test"

	resourceCount = 53
	patchCount    = 34
	updateCount   = 20
)

type KubeVirtTestData struct {
	ctrl             *gomock.Controller
	kvInterface      *kubecli.MockKubeVirtInterface
	kvSource         *framework.FakeControllerSource
	kvInformer       cache.SharedIndexInformer
	apiServiceClient *install.MockAPIServiceInterface

	serviceAccountSource           *framework.FakeControllerSource
	clusterRoleSource              *framework.FakeControllerSource
	clusterRoleBindingSource       *framework.FakeControllerSource
	roleSource                     *framework.FakeControllerSource
	roleBindingSource              *framework.FakeControllerSource
	crdSource                      *framework.FakeControllerSource
	serviceSource                  *framework.FakeControllerSource
	deploymentSource               *framework.FakeControllerSource
	daemonSetSource                *framework.FakeControllerSource
	validatingWebhookSource        *framework.FakeControllerSource
	mutatingWebhookSource          *framework.FakeControllerSource
	apiserviceSource               *framework.FakeControllerSource
	sccSource                      *framework.FakeControllerSource
	installStrategyConfigMapSource *framework.FakeControllerSource
	installStrategyJobSource       *framework.FakeControllerSource
	infrastructurePodSource        *framework.FakeControllerSource
	podDisruptionBudgetSource      *framework.FakeControllerSource
	serviceMonitorSource           *framework.FakeControllerSource
	namespaceSource                *framework.FakeControllerSource
	prometheusRuleSource           *framework.FakeControllerSource
	secretsSource                  *framework.FakeControllerSource
	configMapSource                *framework.FakeControllerSource

	stop       chan struct{}
	controller *KubeVirtController

	recorder *record.FakeRecorder

	mockQueue  *testutils.MockWorkQueue
	virtClient *kubecli.MockKubevirtClient
	kubeClient *fake.Clientset
	secClient  *secv1fake.FakeSecurityV1
	extClient  *extclientfake.Clientset
	promClient *promclientfake.Clientset

	informers util.Informers
	stores    util.Stores

	totalAdds       int
	totalUpdates    int
	totalPatches    int
	totalDeletions  int
	resourceChanges map[string]map[string]int

	deleteFromCache bool
	addToCache      bool

	defaultConfig *util.KubeVirtDeploymentConfig
}

func (k *KubeVirtTestData) BeforeTest() {

	k.defaultConfig = getConfig("", "")

	k.totalAdds = 0
	k.totalUpdates = 0
	k.totalPatches = 0
	k.totalDeletions = 0
	k.resourceChanges = make(map[string]map[string]int)
	k.deleteFromCache = true
	k.addToCache = true

	k.stop = make(chan struct{})
	k.ctrl = gomock.NewController(GinkgoT())
	k.virtClient = kubecli.NewMockKubevirtClient(k.ctrl)
	k.kvInterface = kubecli.NewMockKubeVirtInterface(k.ctrl)
	k.apiServiceClient = install.NewMockAPIServiceInterface(k.ctrl)

	k.kvInformer, k.kvSource = testutils.NewFakeInformerFor(&v1.KubeVirt{})
	k.recorder = record.NewFakeRecorder(100)

	k.informers.ServiceAccount, k.serviceAccountSource = testutils.NewFakeInformerFor(&k8sv1.ServiceAccount{})
	k.stores.ServiceAccountCache = k.informers.ServiceAccount.GetStore()

	k.informers.ClusterRole, k.clusterRoleSource = testutils.NewFakeInformerFor(&rbacv1.ClusterRole{})
	k.stores.ClusterRoleCache = k.informers.ClusterRole.GetStore()

	k.informers.ClusterRoleBinding, k.clusterRoleBindingSource = testutils.NewFakeInformerFor(&rbacv1.ClusterRoleBinding{})
	k.stores.ClusterRoleBindingCache = k.informers.ClusterRoleBinding.GetStore()

	k.informers.Role, k.roleSource = testutils.NewFakeInformerFor(&rbacv1.Role{})
	k.stores.RoleCache = k.informers.Role.GetStore()

	k.informers.RoleBinding, k.roleBindingSource = testutils.NewFakeInformerFor(&rbacv1.RoleBinding{})
	k.stores.RoleBindingCache = k.informers.RoleBinding.GetStore()

	k.informers.Crd, k.crdSource = testutils.NewFakeInformerFor(&extv1.CustomResourceDefinition{})
	k.stores.CrdCache = k.informers.Crd.GetStore()

	k.informers.Service, k.serviceSource = testutils.NewFakeInformerFor(&k8sv1.Service{})
	k.stores.ServiceCache = k.informers.Service.GetStore()

	k.informers.Deployment, k.deploymentSource = testutils.NewFakeInformerFor(&appsv1.Deployment{})
	k.stores.DeploymentCache = k.informers.Deployment.GetStore()

	k.informers.DaemonSet, k.daemonSetSource = testutils.NewFakeInformerFor(&appsv1.DaemonSet{})
	k.stores.DaemonSetCache = k.informers.DaemonSet.GetStore()

	k.informers.ValidationWebhook, k.validatingWebhookSource = testutils.NewFakeInformerFor(&admissionregistrationv1.ValidatingWebhookConfiguration{})
	k.stores.ValidationWebhookCache = k.informers.ValidationWebhook.GetStore()
	k.informers.MutatingWebhook, k.mutatingWebhookSource = testutils.NewFakeInformerFor(&admissionregistrationv1.MutatingWebhookConfiguration{})
	k.stores.MutatingWebhookCache = k.informers.MutatingWebhook.GetStore()
	k.informers.APIService, k.apiserviceSource = testutils.NewFakeInformerFor(&apiregv1.APIService{})
	k.stores.APIServiceCache = k.informers.APIService.GetStore()

	k.informers.SCC, k.sccSource = testutils.NewFakeInformerFor(&secv1.SecurityContextConstraints{})
	k.stores.SCCCache = k.informers.SCC.GetStore()

	k.informers.InstallStrategyConfigMap, k.installStrategyConfigMapSource = testutils.NewFakeInformerFor(&k8sv1.ConfigMap{})
	k.stores.InstallStrategyConfigMapCache = k.informers.InstallStrategyConfigMap.GetStore()

	k.informers.InstallStrategyJob, k.installStrategyJobSource = testutils.NewFakeInformerFor(&batchv1.Job{})
	k.stores.InstallStrategyJobCache = k.informers.InstallStrategyJob.GetStore()

	k.informers.InfrastructurePod, k.infrastructurePodSource = testutils.NewFakeInformerFor(&k8sv1.Pod{})
	k.stores.InfrastructurePodCache = k.informers.InfrastructurePod.GetStore()

	k.informers.PodDisruptionBudget, k.podDisruptionBudgetSource = testutils.NewFakeInformerFor(&policyv1beta1.PodDisruptionBudget{})
	k.stores.PodDisruptionBudgetCache = k.informers.PodDisruptionBudget.GetStore()

	k.informers.Namespace, k.namespaceSource = testutils.NewFakeInformerWithIndexersFor(
		&k8sv1.Namespace{}, cache.Indexers{
			"namespace_name": func(obj interface{}) ([]string, error) {
				return []string{obj.(*k8sv1.Namespace).GetName()}, nil
			},
		})
	k.stores.NamespaceCache = k.informers.Namespace.GetStore()

	// test OpenShift components
	k.stores.IsOnOpenshift = true

	k.informers.ServiceMonitor, k.serviceMonitorSource = testutils.NewFakeInformerFor(&promv1.ServiceMonitor{Spec: promv1.ServiceMonitorSpec{}})
	k.stores.ServiceMonitorCache = k.informers.ServiceMonitor.GetStore()
	k.stores.ServiceMonitorEnabled = true

	k.informers.PrometheusRule, k.prometheusRuleSource = testutils.NewFakeInformerFor(&promv1.PrometheusRule{Spec: promv1.PrometheusRuleSpec{}})
	k.stores.PrometheusRuleCache = k.informers.PrometheusRule.GetStore()
	k.stores.PrometheusRulesEnabled = true

	k.informers.Secrets, k.secretsSource = testutils.NewFakeInformerFor(&k8sv1.Secret{})
	k.stores.SecretCache = k.informers.Secrets.GetStore()
	k.informers.ConfigMap, k.configMapSource = testutils.NewFakeInformerFor(&k8sv1.ConfigMap{})
	k.stores.ConfigMapCache = k.informers.ConfigMap.GetStore()

	k.controller = NewKubeVirtController(k.virtClient, k.apiServiceClient, k.kvInformer, k.recorder, k.stores, k.informers, NAMESPACE)
	k.controller.delayedQueueAdder = func(key interface{}, queue workqueue.RateLimitingInterface) {
		// no delay to speed up tests
		queue.Add(key)
	}

	// Wrap our workqueue to have a way to detect when we are done processing updates
	k.mockQueue = testutils.NewMockWorkQueue(k.controller.queue)
	k.controller.queue = k.mockQueue

	// Set up mock client
	k.virtClient.EXPECT().KubeVirt(NAMESPACE).Return(k.kvInterface).AnyTimes()
	k.kubeClient = fake.NewSimpleClientset()
	k.secClient = &secv1fake.FakeSecurityV1{
		Fake: &fake.NewSimpleClientset().Fake,
	}
	k.extClient = extclientfake.NewSimpleClientset()

	k.promClient = promclientfake.NewSimpleClientset()

	k.virtClient.EXPECT().AdmissionregistrationV1().Return(k.kubeClient.AdmissionregistrationV1()).AnyTimes()
	k.virtClient.EXPECT().CoreV1().Return(k.kubeClient.CoreV1()).AnyTimes()
	k.virtClient.EXPECT().BatchV1().Return(k.kubeClient.BatchV1()).AnyTimes()
	k.virtClient.EXPECT().RbacV1().Return(k.kubeClient.RbacV1()).AnyTimes()
	k.virtClient.EXPECT().AppsV1().Return(k.kubeClient.AppsV1()).AnyTimes()
	k.virtClient.EXPECT().SecClient().Return(k.secClient).AnyTimes()
	k.virtClient.EXPECT().ExtensionsClient().Return(k.extClient).AnyTimes()
	k.virtClient.EXPECT().PolicyV1beta1().Return(k.kubeClient.PolicyV1beta1()).AnyTimes()
	k.virtClient.EXPECT().PrometheusClient().Return(k.promClient).AnyTimes()

	// Make sure that all unexpected calls to kubeClient will fail
	k.kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		if action.GetVerb() == "get" && action.GetResource().Resource == "secrets" {
			return true, nil, errors.NewNotFound(schema.GroupResource{Group: "", Resource: "secrets"}, "whatever")
		}
		if action.GetVerb() == "get" && action.GetResource().Resource == "validatingwebhookconfigurations" {
			return true, nil, errors.NewNotFound(schema.GroupResource{Group: "", Resource: "validatingwebhookconfigurations"}, "whatever")
		}
		if action.GetVerb() == "get" && action.GetResource().Resource == "mutatingwebhookconfigurations" {
			return true, nil, errors.NewNotFound(schema.GroupResource{Group: "", Resource: "mutatingwebhookconfigurations"}, "whatever")
		}
		if action.GetVerb() == "get" && action.GetResource().Resource == "serviceaccounts" {
			return true, nil, errors.NewNotFound(schema.GroupResource{Group: "", Resource: "serviceaccounts"}, "whatever")
		}
		if action.GetVerb() != "get" || action.GetResource().Resource != "namespaces" {
			Expect(action).To(BeNil())
		}
		return true, nil, nil
	})
	k.apiServiceClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, errors.NewNotFound(schema.GroupResource{Group: "", Resource: "apiservices"}, "whatever"))
	k.secClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		Expect(action).To(BeNil())
		return true, nil, nil
	})
	k.extClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		Expect(action).To(BeNil())
		return true, nil, nil
	})
	k.promClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		Expect(action).To(BeNil())
		return true, nil, nil
	})

	syncCaches(k.stop, k.kvInformer, k.informers)

	// add the privileged SCC without KubeVirt accounts
	scc := getSCC()
	k.sccSource.Add(&scc)

	k.deleteFromCache = true
	k.addToCache = true
}

func (k *KubeVirtTestData) AfterTest() {
	close(k.stop)

	// Ensure that we add checks for expected events to every test
	Expect(k.recorder.Events).To(BeEmpty())
	k.ctrl.Finish()
}

func (k *KubeVirtTestData) shouldExpectKubeVirtUpdate(times int) {
	update := k.kvInterface.EXPECT().Update(gomock.Any())
	update.Do(func(kv *v1.KubeVirt) {
		k.kvInformer.GetStore().Update(kv)
		update.Return(kv, nil)
	}).Times(times)
}

func (k *KubeVirtTestData) shouldExpectKubeVirtUpdateStatus(times int) {
	update := k.kvInterface.EXPECT().UpdateStatus(gomock.Any())
	update.Do(func(kv *v1.KubeVirt) {
		k.kvInformer.GetStore().Update(kv)
		update.Return(kv, nil)
	}).Times(times)
}

func (k *KubeVirtTestData) shouldExpectKubeVirtUpdateStatusVersion(times int, config *util.KubeVirtDeploymentConfig) {
	update := k.kvInterface.EXPECT().UpdateStatus(gomock.Any())
	update.Do(func(kv *v1.KubeVirt) {

		Expect(kv.Status.TargetKubeVirtVersion).To(Equal(config.GetKubeVirtVersion()))
		Expect(kv.Status.ObservedKubeVirtVersion).To(Equal(config.GetKubeVirtVersion()))
		k.kvInformer.GetStore().Update(kv)
		update.Return(kv, nil)
	}).Times(times)
}

func (k *KubeVirtTestData) shouldExpectKubeVirtUpdateStatusFailureCondition(reason string) {
	update := k.kvInterface.EXPECT().UpdateStatus(gomock.Any())
	update.Do(func(kv *v1.KubeVirt) {
		Expect(len(kv.Status.Conditions)).To(Equal(1))
		Expect(kv.Status.Conditions[0].Reason).To(Equal(reason))
		k.kvInformer.GetStore().Update(kv)
		update.Return(kv, nil)
	}).Times(1)
}

func (k *KubeVirtTestData) addKubeVirt(kv *v1.KubeVirt) {
	k.mockQueue.ExpectAdds(1)
	k.kvSource.Add(kv)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) getLatestKubeVirt(kv *v1.KubeVirt) *v1.KubeVirt {
	if obj, exists, _ := k.kvInformer.GetStore().GetByKey(kv.GetNamespace() + "/" + kv.GetName()); exists {
		if kvLatest, ok := obj.(*v1.KubeVirt); ok {
			return kvLatest
		}
	}
	return nil
}

func (k *KubeVirtTestData) shouldExpectDeletions() {
	genericDeleteFunc := k.genericDeleteFunc()
	k.kubeClient.Fake.PrependReactor("delete", "serviceaccounts", genericDeleteFunc)
	k.kubeClient.Fake.PrependReactor("delete", "clusterroles", genericDeleteFunc)
	k.kubeClient.Fake.PrependReactor("delete", "clusterrolebindings", genericDeleteFunc)
	k.kubeClient.Fake.PrependReactor("delete", "roles", genericDeleteFunc)
	k.kubeClient.Fake.PrependReactor("delete", "rolebindings", genericDeleteFunc)
	k.extClient.Fake.PrependReactor("delete", "customresourcedefinitions", genericDeleteFunc)
	k.kubeClient.Fake.PrependReactor("delete", "services", genericDeleteFunc)
	k.kubeClient.Fake.PrependReactor("delete", "deployments", genericDeleteFunc)
	k.kubeClient.Fake.PrependReactor("delete", "daemonsets", genericDeleteFunc)
	k.kubeClient.Fake.PrependReactor("delete", "validatingwebhookconfigurations", genericDeleteFunc)
	k.kubeClient.Fake.PrependReactor("delete", "mutatingwebhookconfigurations", genericDeleteFunc)
	k.kubeClient.Fake.PrependReactor("delete", "secrets", genericDeleteFunc)
	k.kubeClient.Fake.PrependReactor("delete", "configmaps", genericDeleteFunc)
	k.kubeClient.Fake.PrependReactor("delete", "poddisruptionbudgets", genericDeleteFunc)
	k.secClient.Fake.PrependReactor("delete", "securitycontextconstraints", genericDeleteFunc)
	k.promClient.Fake.PrependReactor("delete", "servicemonitors", genericDeleteFunc)
	k.promClient.Fake.PrependReactor("delete", "prometheusrules", genericDeleteFunc)
	k.apiServiceClient.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Do(func(ctx context.Context, name string, options interface{}) {
		genericDeleteFunc(&testing.DeleteActionImpl{ActionImpl: testing.ActionImpl{Resource: schema.GroupVersionResource{Resource: "apiservices"}}, Name: name})
	})
}

func (k *KubeVirtTestData) genericDeleteFunc() func(action testing.Action) (handled bool, obj runtime.Object, err error) {
	return func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		deleted, ok := action.(testing.DeleteAction)
		Expect(ok).To(BeTrue())
		k.totalDeletions++
		var key string
		if len(deleted.GetNamespace()) > 0 {
			key = deleted.GetNamespace() + "/"
		}
		key += deleted.GetName()
		if k.deleteFromCache {
			k.deleteResource(deleted.GetResource().Resource, key)
		}
		return true, nil, nil
	}
}

func (k *KubeVirtTestData) deleteResource(resource string, key string) {
	switch resource {
	case "serviceaccounts":
		k.deleteServiceAccount(key)
	case "clusterroles":
		k.deleteClusterRole(key)
	case "clusterrolebindings":
		k.deleteClusterRoleBinding(key)
	case "roles":
		k.deleteRole(key)
	case "rolebindings":
		k.deleteRoleBinding(key)
	case "customresourcedefinitions":
		k.deleteCrd(key)
	case "services":
		k.deleteService(key)
	case "deployments":
		k.deleteDeployment(key)
	case "daemonsets":
		k.deleteDaemonset(key)
	case "validatingwebhookconfigurations":
		k.deleteValidationWebhook(key)
	case "mutatingwebhookconfigurations":
		k.deleteMutatingWebhook(key)
	case "apiservices":
		k.deleteAPIService(key)
	case "jobs":
		k.deleteInstallStrategyJob(key)
	case "configmaps":
		k.deleteConfigMap(key)
	case "poddisruptionbudgets":
		k.deletePodDisruptionBudget(key)
	case "secrets":
		k.deleteSecret(key)
	case "securitycontextconstraints":
		k.deleteSCC(key)
	case "servicemonitors":
		k.deleteServiceMonitor(key)
	case "prometheusrules":
		k.deletePrometheusRule(key)
	default:
		Fail(fmt.Sprintf("unknown resource type %+v", resource))
	}
	if _, ok := k.resourceChanges[resource]; !ok {
		k.resourceChanges[resource] = make(map[string]int)
	}
	k.resourceChanges[resource][Deleted]++
}

func (k *KubeVirtTestData) deleteServiceAccount(key string) {
	k.mockQueue.ExpectAdds(1)
	if obj, exists, _ := k.informers.ServiceAccount.GetStore().GetByKey(key); exists {
		k.serviceAccountSource.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) deleteClusterRole(key string) {
	k.mockQueue.ExpectAdds(1)
	if obj, exists, _ := k.informers.ClusterRole.GetStore().GetByKey(key); exists {
		k.clusterRoleSource.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) deleteClusterRoleBinding(key string) {
	k.mockQueue.ExpectAdds(1)
	if obj, exists, _ := k.informers.ClusterRoleBinding.GetStore().GetByKey(key); exists {
		k.clusterRoleBindingSource.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) deleteRole(key string) {
	k.mockQueue.ExpectAdds(1)
	if obj, exists, _ := k.informers.Role.GetStore().GetByKey(key); exists {
		k.roleSource.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) deleteRoleBinding(key string) {
	k.mockQueue.ExpectAdds(1)
	if obj, exists, _ := k.informers.RoleBinding.GetStore().GetByKey(key); exists {
		k.roleBindingSource.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) deleteCrd(key string) {
	k.mockQueue.ExpectAdds(1)
	if obj, exists, _ := k.informers.Crd.GetStore().GetByKey(key); exists {
		k.crdSource.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) deleteService(key string) {
	k.mockQueue.ExpectAdds(1)
	if obj, exists, _ := k.informers.Service.GetStore().GetByKey(key); exists {
		k.serviceSource.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) deleteDeployment(key string) {
	k.mockQueue.ExpectAdds(1)
	if obj, exists, _ := k.informers.Deployment.GetStore().GetByKey(key); exists {
		k.deploymentSource.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) deleteDaemonset(key string) {
	k.mockQueue.ExpectAdds(1)
	if obj, exists, _ := k.informers.DaemonSet.GetStore().GetByKey(key); exists {
		k.daemonSetSource.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) deleteValidationWebhook(key string) {
	k.mockQueue.ExpectAdds(1)
	if obj, exists, _ := k.informers.ValidationWebhook.GetStore().GetByKey(key); exists {
		k.validatingWebhookSource.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) deleteMutatingWebhook(key string) {
	k.mockQueue.ExpectAdds(1)
	if obj, exists, _ := k.informers.MutatingWebhook.GetStore().GetByKey(key); exists {
		k.mutatingWebhookSource.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) deleteAPIService(key string) {
	k.mockQueue.ExpectAdds(1)
	if obj, exists, _ := k.informers.APIService.GetStore().GetByKey(key); exists {
		k.apiserviceSource.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) deleteInstallStrategyJob(key string) {
	k.mockQueue.ExpectAdds(1)
	if obj, exists, _ := k.informers.InstallStrategyJob.GetStore().GetByKey(key); exists {
		k.installStrategyJobSource.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) deletePodDisruptionBudget(key string) {
	k.mockQueue.ExpectAdds(1)
	if obj, exists, _ := k.informers.PodDisruptionBudget.GetStore().GetByKey(key); exists {
		k.podDisruptionBudgetSource.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) deleteSecret(key string) {
	k.mockQueue.ExpectAdds(1)
	if obj, exists, _ := k.informers.Secrets.GetStore().GetByKey(key); exists {
		k.secretsSource.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) deleteConfigMap(key string) {
	k.mockQueue.ExpectAdds(1)
	if obj, exists, _ := k.informers.ConfigMap.GetStore().GetByKey(key); exists {
		configMap := obj.(*k8sv1.ConfigMap)
		k.configMapSource.Delete(configMap)
	} else if obj, exists, _ := k.informers.InstallStrategyConfigMap.GetStore().GetByKey(key); exists {
		configMap := obj.(*k8sv1.ConfigMap)
		k.installStrategyConfigMapSource.Delete(configMap)
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) deleteSCC(key string) {
	k.mockQueue.ExpectAdds(1)
	if obj, exists, _ := k.informers.SCC.GetStore().GetByKey(key); exists {
		k.sccSource.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) deleteServiceMonitor(key string) {
	k.mockQueue.ExpectAdds(1)
	if obj, exists, _ := k.informers.ServiceMonitor.GetStore().GetByKey(key); exists {
		k.serviceMonitorSource.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) deletePrometheusRule(key string) {
	k.mockQueue.ExpectAdds(1)
	if obj, exists, _ := k.informers.PrometheusRule.GetStore().GetByKey(key); exists {
		k.prometheusRuleSource.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) shouldExpectPatchesAndUpdates() {
	genericPatchFunc := k.genericPatchFunc()
	genericUpdateFunc := k.genericUpdateFunc()
	webhookValidationPatchFunc := k.webhookValidationPatchFunc()
	webhookMutatingPatchFunc := k.webhookMutatingPatchFunc()
	daemonsetPatchFunc := k.daemonsetPatchFunc()
	deploymentPatchFunc := k.deploymentPatchFunc()
	podDisruptionBudgetPatchFunc := k.podDisruptionBudgetPatchFunc()
	k.extClient.Fake.PrependReactor("patch", "customresourcedefinitions", k.crdPatchFunc())
	k.kubeClient.Fake.PrependReactor("patch", "serviceaccounts", genericPatchFunc)
	k.kubeClient.Fake.PrependReactor("update", "clusterroles", genericUpdateFunc)
	k.kubeClient.Fake.PrependReactor("update", "clusterrolebindings", genericUpdateFunc)
	k.kubeClient.Fake.PrependReactor("update", "roles", genericUpdateFunc)
	k.kubeClient.Fake.PrependReactor("update", "rolebindings", genericUpdateFunc)
	k.kubeClient.Fake.PrependReactor("patch", "validatingwebhookconfigurations", webhookValidationPatchFunc)
	k.kubeClient.Fake.PrependReactor("patch", "mutatingwebhookconfigurations", webhookMutatingPatchFunc)
	k.kubeClient.Fake.PrependReactor("patch", "secrets", genericPatchFunc)
	k.kubeClient.Fake.PrependReactor("patch", "configmaps", genericPatchFunc)

	k.kubeClient.Fake.PrependReactor("patch", "services", genericPatchFunc)
	k.kubeClient.Fake.PrependReactor("patch", "daemonsets", daemonsetPatchFunc)
	k.kubeClient.Fake.PrependReactor("patch", "deployments", deploymentPatchFunc)
	k.kubeClient.Fake.PrependReactor("patch", "poddisruptionbudgets", podDisruptionBudgetPatchFunc)
	k.secClient.Fake.PrependReactor("update", "securitycontextconstraints", genericUpdateFunc)
	k.promClient.Fake.PrependReactor("patch", "servicemonitors", genericPatchFunc)
	k.promClient.Fake.PrependReactor("patch", "prometheusrules", genericPatchFunc)
	k.apiServiceClient.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Do(func(args ...interface{}) {
		genericPatchFunc(&testing.PatchActionImpl{ActionImpl: testing.ActionImpl{Resource: schema.GroupVersionResource{Resource: "apiservices"}}})
	})
}

func (k *KubeVirtTestData) genericPatchFunc() func(action testing.Action) (handled bool, obj runtime.Object, err error) {
	return func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		_, ok := action.(testing.PatchAction)
		Expect(ok).To(BeTrue())
		k.totalPatches++
		resource := action.GetResource().Resource
		if _, ok := k.resourceChanges[resource]; !ok {
			k.resourceChanges[resource] = make(map[string]int)
		}
		k.resourceChanges[resource][Patched]++

		return true, nil, nil
	}
}

func (k *KubeVirtTestData) genericUpdateFunc() func(action testing.Action) (handled bool, obj runtime.Object, err error) {
	return func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		update, ok := action.(testing.UpdateAction)
		Expect(ok).To(BeTrue(), "genericUpdateFunction testing ok")
		k.totalUpdates++

		resource := action.GetResource().Resource
		if _, ok := k.resourceChanges[resource]; !ok {
			k.resourceChanges[resource] = make(map[string]int)
		}
		k.resourceChanges[resource][Updated]++

		return true, update.GetObject(), nil
	}
}

func (k *KubeVirtTestData) webhookValidationPatchFunc() func(action testing.Action) (handled bool, obj runtime.Object, err error) {
	return func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		k.genericPatchFunc()(action)

		return true, &admissionregistrationv1.ValidatingWebhookConfiguration{}, nil
	}
}

func (k *KubeVirtTestData) webhookMutatingPatchFunc() func(action testing.Action) (handled bool, obj runtime.Object, err error) {
	return func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		k.genericPatchFunc()(action)

		return true, &admissionregistrationv1.MutatingWebhookConfiguration{}, nil
	}
}

func (k *KubeVirtTestData) deploymentPatchFunc() func(action testing.Action) (handled bool, obj runtime.Object, err error) {
	return func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		k.genericPatchFunc()(action)

		return true, &appsv1.Deployment{}, nil
	}
}

func (k *KubeVirtTestData) daemonsetPatchFunc() func(action testing.Action) (handled bool, obj runtime.Object, err error) {
	return func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		k.genericPatchFunc()(action)

		return true, &appsv1.DaemonSet{}, nil
	}
}

func (k *KubeVirtTestData) podDisruptionBudgetPatchFunc() func(action testing.Action) (handled bool, obj runtime.Object, err error) {
	return func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		k.genericPatchFunc()(action)

		return true, &policyv1beta1.PodDisruptionBudget{}, nil
	}
}

func (k *KubeVirtTestData) crdPatchFunc() func(action testing.Action) (handled bool, obj runtime.Object, err error) {
	return func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		k.genericPatchFunc()(action)

		return true, &extv1.CustomResourceDefinition{}, nil
	}
}

func (k *KubeVirtTestData) shouldExpectCreations() {
	genericCreateFunc := k.genericCreateFunc()
	k.kubeClient.Fake.PrependReactor("create", "serviceaccounts", genericCreateFunc)
	k.kubeClient.Fake.PrependReactor("create", "clusterroles", genericCreateFunc)
	k.kubeClient.Fake.PrependReactor("create", "clusterrolebindings", genericCreateFunc)
	k.kubeClient.Fake.PrependReactor("create", "roles", genericCreateFunc)
	k.kubeClient.Fake.PrependReactor("create", "rolebindings", genericCreateFunc)
	k.extClient.Fake.PrependReactor("create", "customresourcedefinitions", genericCreateFunc)
	k.kubeClient.Fake.PrependReactor("create", "services", genericCreateFunc)
	k.kubeClient.Fake.PrependReactor("create", "deployments", genericCreateFunc)
	k.kubeClient.Fake.PrependReactor("create", "daemonsets", genericCreateFunc)
	k.kubeClient.Fake.PrependReactor("create", "validatingwebhookconfigurations", genericCreateFunc)
	k.kubeClient.Fake.PrependReactor("create", "mutatingwebhookconfigurations", genericCreateFunc)
	k.kubeClient.Fake.PrependReactor("create", "secrets", genericCreateFunc)
	k.kubeClient.Fake.PrependReactor("create", "configmaps", genericCreateFunc)
	k.kubeClient.Fake.PrependReactor("create", "poddisruptionbudgets", genericCreateFunc)
	k.secClient.Fake.PrependReactor("create", "securitycontextconstraints", genericCreateFunc)
	k.promClient.Fake.PrependReactor("create", "servicemonitors", genericCreateFunc)
	k.promClient.Fake.PrependReactor("create", "prometheusrules", genericCreateFunc)
	k.apiServiceClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Do(func(ctx context.Context, obj runtime.Object, opts metav1.CreateOptions) {
		genericCreateFunc(&testing.CreateActionImpl{Object: obj})
	})
}

func (k *KubeVirtTestData) genericCreateFunc() func(action testing.Action) (handled bool, obj runtime.Object, err error) {
	return func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())
		k.totalAdds++
		if k.addToCache {
			k.addResource(create.GetObject(), nil, nil)
		}
		return true, create.GetObject(), nil
	}
}

func (k *KubeVirtTestData) addResource(obj runtime.Object, config *util.KubeVirtDeploymentConfig, kv *v1.KubeVirt) {
	switch resource := obj.(type) {
	case *k8sv1.ServiceAccount:
		injectMetadata(&obj.(*k8sv1.ServiceAccount).ObjectMeta, config)
		k.addServiceAccount(resource)
	case *rbacv1.ClusterRole:
		injectMetadata(&obj.(*rbacv1.ClusterRole).ObjectMeta, config)
		k.addClusterRole(resource)
	case *rbacv1.ClusterRoleBinding:
		injectMetadata(&obj.(*rbacv1.ClusterRoleBinding).ObjectMeta, config)
		k.addClusterRoleBinding(resource)
	case *rbacv1.Role:
		injectMetadata(&obj.(*rbacv1.Role).ObjectMeta, config)
		k.addRole(resource)
	case *rbacv1.RoleBinding:
		injectMetadata(&obj.(*rbacv1.RoleBinding).ObjectMeta, config)
		k.addRoleBinding(resource)
	case *extv1.CustomResourceDefinition:
		injectMetadata(&obj.(*extv1.CustomResourceDefinition).ObjectMeta, config)
		k.addCrd(resource, kv)
	case *k8sv1.Service:
		injectMetadata(&obj.(*k8sv1.Service).ObjectMeta, config)
		k.addService(resource)
	case *appsv1.Deployment:
		injectMetadata(&obj.(*appsv1.Deployment).ObjectMeta, config)
		k.addDeployment(resource, kv)
	case *appsv1.DaemonSet:
		injectMetadata(&obj.(*appsv1.DaemonSet).ObjectMeta, config)
		k.addDaemonset(resource, kv)
	case *admissionregistrationv1.ValidatingWebhookConfiguration:
		injectMetadata(&obj.(*admissionregistrationv1.ValidatingWebhookConfiguration).ObjectMeta, config)
		k.addValidatingWebhook(resource, kv)
	case *admissionregistrationv1.MutatingWebhookConfiguration:
		injectMetadata(&obj.(*admissionregistrationv1.MutatingWebhookConfiguration).ObjectMeta, config)
		k.addMutatingWebhook(resource, kv)
	case *apiregv1.APIService:
		injectMetadata(&obj.(*apiregv1.APIService).ObjectMeta, config)
		k.addAPIService(resource)
	case *batchv1.Job:
		injectMetadata(&obj.(*batchv1.Job).ObjectMeta, config)
		k.addInstallStrategyJob(resource)
	case *k8sv1.ConfigMap:
		injectMetadata(&obj.(*k8sv1.ConfigMap).ObjectMeta, config)
		k.addConfigMap(resource)
	case *k8sv1.Pod:
		injectMetadata(&obj.(*k8sv1.Pod).ObjectMeta, config)
		k.addPod(resource)
	case *policyv1beta1.PodDisruptionBudget:
		injectMetadata(&obj.(*policyv1beta1.PodDisruptionBudget).ObjectMeta, config)
		k.addPodDisruptionBudget(resource, kv)
	case *k8sv1.Secret:
		injectMetadata(&obj.(*k8sv1.Secret).ObjectMeta, config)
		k.addSecret(resource)
	case *secv1.SecurityContextConstraints:
		injectMetadata(&obj.(*secv1.SecurityContextConstraints).ObjectMeta, config)
		k.addSCC(resource)
	case *promv1.ServiceMonitor:
		injectMetadata(&obj.(*promv1.ServiceMonitor).ObjectMeta, config)
		k.addServiceMonitor(resource)
	case *promv1.PrometheusRule:
		injectMetadata(&obj.(*promv1.PrometheusRule).ObjectMeta, config)
		k.addPrometheusRule(resource)
	default:
		Fail("unknown resource type")
	}
	split := strings.Split(fmt.Sprintf("%T", obj), ".")
	resourceKey := strings.ToLower(split[len(split)-1]) + "s"
	if _, ok := k.resourceChanges[resourceKey]; !ok {
		k.resourceChanges[resourceKey] = make(map[string]int)
	}
	k.resourceChanges[resourceKey][Added]++
}

func (k *KubeVirtTestData) addServiceAccount(sa *k8sv1.ServiceAccount) {
	k.mockQueue.ExpectAdds(1)
	k.serviceAccountSource.Add(sa)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) addClusterRole(cr *rbacv1.ClusterRole) {
	k.mockQueue.ExpectAdds(1)
	k.clusterRoleSource.Add(cr)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) addClusterRoleBinding(crb *rbacv1.ClusterRoleBinding) {
	k.mockQueue.ExpectAdds(1)
	k.clusterRoleBindingSource.Add(crb)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) addRole(role *rbacv1.Role) {
	k.mockQueue.ExpectAdds(1)
	k.roleSource.Add(role)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) addRoleBinding(rb *rbacv1.RoleBinding) {
	k.mockQueue.ExpectAdds(1)
	k.roleBindingSource.Add(rb)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) addCrd(crd *extv1.CustomResourceDefinition, kv *v1.KubeVirt) {
	k.mockQueue.ExpectAdds(1)
	if kv != nil {
		apply.SetGeneration(&kv.Status.Generations, crd)
	}
	k.crdSource.Add(crd)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) addService(svc *k8sv1.Service) {
	k.mockQueue.ExpectAdds(1)
	k.serviceSource.Add(svc)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) addDeployment(depl *appsv1.Deployment, kv *v1.KubeVirt) {
	k.mockQueue.ExpectAdds(1)
	if kv != nil {
		apply.SetGeneration(&kv.Status.Generations, depl)
	}

	k.deploymentSource.Add(depl)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) addDaemonset(ds *appsv1.DaemonSet, kv *v1.KubeVirt) {
	k.mockQueue.ExpectAdds(1)
	if kv != nil {
		apply.SetGeneration(&kv.Status.Generations, ds)
	}

	k.daemonSetSource.Add(ds)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) addMutatingWebhook(wh *admissionregistrationv1.MutatingWebhookConfiguration, kv *v1.KubeVirt) {
	k.mockQueue.ExpectAdds(1)
	if kv != nil {
		apply.SetGeneration(&kv.Status.Generations, wh)
	}

	k.mutatingWebhookSource.Add(wh)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) addAPIService(wh *apiregv1.APIService) {
	k.mockQueue.ExpectAdds(1)
	k.apiserviceSource.Add(wh)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) addInstallStrategyJob(job *batchv1.Job) {
	k.mockQueue.ExpectAdds(1)
	k.installStrategyJobSource.Add(job)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) addPod(pod *k8sv1.Pod) {
	k.mockQueue.ExpectAdds(1)
	k.infrastructurePodSource.Add(pod)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) addPodDisruptionBudget(podDisruptionBudget *policyv1beta1.PodDisruptionBudget, kv *v1.KubeVirt) {
	k.mockQueue.ExpectAdds(1)
	if kv != nil {
		apply.SetGeneration(&kv.Status.Generations, podDisruptionBudget)
	}

	k.podDisruptionBudgetSource.Add(podDisruptionBudget)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) addSecret(secret *k8sv1.Secret) {
	k.mockQueue.ExpectAdds(1)
	k.secretsSource.Add(secret)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) addConfigMap(configMap *k8sv1.ConfigMap) {
	k.mockQueue.ExpectAdds(1)
	if _, ok := configMap.Labels[v1.InstallStrategyLabel]; ok {
		k.installStrategyConfigMapSource.Add(configMap)
	} else {
		k.configMapSource.Add(configMap)
	}
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) addSCC(scc *secv1.SecurityContextConstraints) {
	k.mockQueue.ExpectAdds(1)
	k.sccSource.Add(scc)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) addServiceMonitor(serviceMonitor *promv1.ServiceMonitor) {
	k.mockQueue.ExpectAdds(1)
	k.serviceMonitorSource.Add(serviceMonitor)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) addPrometheusRule(prometheusRule *promv1.PrometheusRule) {
	k.mockQueue.ExpectAdds(1)
	k.prometheusRuleSource.Add(prometheusRule)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) generateRandomResources() int {
	version := fmt.Sprintf("rand-%s", rand.String(10))
	registry := fmt.Sprintf("rand-%s", rand.String(10))
	config := getConfig(registry, version)

	all := make([]runtime.Object, 0)
	all = append(all, &k8sv1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("rand-%s", rand.String(10)),
		},
	})
	all = append(all, &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("rand-%s", rand.String(10)),
		},
	})
	all = append(all, &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("rand-%s", rand.String(10)),
		},
	})
	all = append(all, &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("rand-%s", rand.String(10)),
		},
	})
	all = append(all, &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("rand-%s", rand.String(10)),
		},
	})
	all = append(all, &extv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("rand-%s", rand.String(10)),
		},
	})

	all = append(all, &k8sv1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("rand-%s", rand.String(10)),
		},
	})
	all = append(all, &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("rand-%s", rand.String(10)),
		},
	})
	all = append(all, &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("rand-%s", rand.String(10)),
		},
	})
	all = append(all, &secv1.SecurityContextConstraints{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "security.openshift.io/v1",
			Kind:       "SecurityContextConstraints",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("rand-%s", rand.String(10)),
		},
	})
	for _, obj := range all {
		k.addResource(obj, config, nil)
	}
	return len(all)
}

func (k *KubeVirtTestData) addAll(config *util.KubeVirtDeploymentConfig, kv *v1.KubeVirt) {
	c, _ := apply.NewCustomizer(kv.Spec.CustomizeComponents)

	all := make([]runtime.Object, 0)

	// rbac
	all = append(all, rbac.GetAllCluster()...)
	all = append(all, rbac.GetAllApiServer(NAMESPACE)...)
	all = append(all, rbac.GetAllHandler(NAMESPACE)...)
	all = append(all, rbac.GetAllController(NAMESPACE)...)
	// crds
	functions := []func() (*extv1.CustomResourceDefinition, error){
		components.NewVirtualMachineInstanceCrd, components.NewPresetCrd, components.NewReplicaSetCrd,
		components.NewVirtualMachineCrd, components.NewVirtualMachineInstanceMigrationCrd,
		components.NewVirtualMachineSnapshotCrd, components.NewVirtualMachineSnapshotContentCrd,
		components.NewVirtualMachineRestoreCrd,
	}
	for _, f := range functions {
		crd, err := f()
		if err != nil {
			panic(fmt.Errorf("This should not happen, %v", err))
		}
		all = append(all, crd)
	}
	// cr
	all = append(all, components.NewPrometheusRuleCR(config.GetNamespace(), config.WorkloadUpdatesEnabled()))
	// sccs
	all = append(all, components.NewKubeVirtControllerSCC(NAMESPACE))
	all = append(all, components.NewKubeVirtHandlerSCC(NAMESPACE))
	// services and deployments
	all = append(all, components.NewOperatorWebhookService(NAMESPACE))
	all = append(all, components.NewPrometheusService(NAMESPACE))
	all = append(all, components.NewApiServerService(NAMESPACE))

	apiDeployment, _ := components.NewApiServerDeployment(NAMESPACE, config.GetImageRegistry(), config.GetImagePrefix(), config.GetApiVersion(), "", "", config.GetImagePullPolicy(), config.GetVerbosity(), config.GetExtraEnv())
	apiDeploymentPdb := components.NewPodDisruptionBudgetForDeployment(apiDeployment)
	controller, _ := components.NewControllerDeployment(NAMESPACE, config.GetImageRegistry(), config.GetImagePrefix(), config.GetControllerVersion(), config.GetLauncherVersion(), "", "", config.GetImagePullPolicy(), config.GetVerbosity(), config.GetExtraEnv())
	controllerPdb := components.NewPodDisruptionBudgetForDeployment(controller)
	handler, _ := components.NewHandlerDaemonSet(NAMESPACE, config.GetImageRegistry(), config.GetImagePrefix(), config.GetHandlerVersion(), "", "", config.GetLauncherVersion(), config.GetImagePullPolicy(), config.GetVerbosity(), config.GetExtraEnv())

	all = append(all, apiDeployment, apiDeploymentPdb, controller, controllerPdb, handler)

	all = append(all, rbac.GetAllServiceMonitor(NAMESPACE, config.GetMonitorNamespaces()[0], config.GetMonitorServiceAccount())...)
	all = append(all, components.NewServiceMonitorCR(NAMESPACE, config.GetMonitorNamespaces()[0], true))

	// ca certificate
	caSecret := components.NewCACertSecret(NAMESPACE)
	components.PopulateSecretWithCertificate(caSecret, nil, &metav1.Duration{Duration: apply.Duration7d})
	caCert, _ := components.LoadCertificates(caSecret)
	caBundle := cert.EncodeCertPEM(caCert.Leaf)
	all = append(all, caSecret)

	caConfigMap := components.NewKubeVirtCAConfigMap(NAMESPACE)
	caConfigMap.Data = map[string]string{components.CABundleKey: string(caBundle)}
	all = append(all, caConfigMap)

	// webhooks and apiservice
	validatingWebhook := components.NewVirtAPIValidatingWebhookConfiguration(config.GetNamespace())
	for i := range validatingWebhook.Webhooks {
		validatingWebhook.Webhooks[i].ClientConfig.CABundle = caBundle
	}
	all = append(all, validatingWebhook)

	mutatingWebhook := components.NewVirtAPIMutatingWebhookConfiguration(config.GetNamespace())
	for i := range mutatingWebhook.Webhooks {
		mutatingWebhook.Webhooks[i].ClientConfig.CABundle = caBundle
	}
	all = append(all, mutatingWebhook)

	apiServices := components.NewVirtAPIAPIServices(config.GetNamespace())
	for _, apiService := range apiServices {
		apiService.Spec.CABundle = caBundle
		all = append(all, apiService)
	}

	validatingWebhook = components.NewOpertorValidatingWebhookConfiguration(NAMESPACE)
	for i := range validatingWebhook.Webhooks {
		validatingWebhook.Webhooks[i].ClientConfig.CABundle = caBundle
	}
	all = append(all, validatingWebhook)

	secrets := components.NewCertSecrets(NAMESPACE, config.GetNamespace())
	for _, secret := range secrets {
		components.PopulateSecretWithCertificate(secret, caCert, &metav1.Duration{Duration: apply.Duration1d})
		all = append(all, secret)
	}

	for _, obj := range all {
		m := obj.(metav1.Object)
		a := m.GetAnnotations()
		if len(a) == 0 {
			a = map[string]string{}
		}

		a[v1.KubeVirtCustomizeComponentAnnotationHash] = c.Hash()
		m.SetAnnotations(a)

		k.addResource(obj, config, kv)
	}
}

func (k *KubeVirtTestData) shouldExpectJobCreation() {
	k.kubeClient.Fake.PrependReactor("create", "jobs", k.genericCreateFunc())
}

func (k *KubeVirtTestData) shouldExpectRbacBackupCreations() {
	genericCreateFunc := k.genericCreateFunc()
	k.kubeClient.Fake.PrependReactor("create", "clusterroles", genericCreateFunc)
	k.kubeClient.Fake.PrependReactor("create", "clusterrolebindings", genericCreateFunc)
	k.kubeClient.Fake.PrependReactor("create", "roles", genericCreateFunc)
	k.kubeClient.Fake.PrependReactor("create", "rolebindings", genericCreateFunc)
}

func (k *KubeVirtTestData) shouldExpectJobDeletion() {
	k.kubeClient.Fake.PrependReactor("delete", "jobs", k.genericDeleteFunc())
}

func (k *KubeVirtTestData) shouldExpectInstallStrategyDeletion() {
	k.kubeClient.Fake.PrependReactor("delete", "configmaps", func(action testing.Action) (handled bool, obj runtime.Object, err error) {

		deleted, ok := action.(testing.DeleteAction)
		Expect(ok).To(BeTrue())
		if deleted.GetName() == "kubevirt-ca" {
			return false, nil, nil
		}
		var key string
		if len(deleted.GetNamespace()) > 0 {
			key = deleted.GetNamespace() + "/"
		}
		key += deleted.GetName()
		k.deleteResource(deleted.GetResource().Resource, key)
		return true, nil, nil
	})
}

func (k *KubeVirtTestData) makeApiAndControllerReady() {
	makeDeploymentReady := func(item interface{}) {
		depl, _ := item.(*appsv1.Deployment)
		deplNew := depl.DeepCopy()
		var replicas int32 = 1
		if depl.Spec.Replicas != nil {
			replicas = *depl.Spec.Replicas
		}
		deplNew.Status.Replicas = replicas
		deplNew.Status.ReadyReplicas = replicas
		k.mockQueue.ExpectAdds(1)
		k.deploymentSource.Modify(deplNew)
		k.mockQueue.Wait()
	}

	for _, name := range []string{"/virt-api", "/virt-controller"} {
		exists := false
		var obj interface{}
		// we need to wait until the deployment exists
		for !exists {
			obj, exists, _ = k.controller.stores.DeploymentCache.GetByKey(NAMESPACE + name)
			if exists {
				makeDeploymentReady(obj)
			}
		}
	}

	k.makePodDisruptionBudgetsReady()
}

func (k *KubeVirtTestData) makePodDisruptionBudgetsReady() {
	for _, pdbname := range []string{"/virt-api-pdb", "/virt-controller-pdb"} {
		exists := false
		// we need to wait until the pdb exists
		for !exists {
			_, exists, _ = k.stores.PodDisruptionBudgetCache.GetByKey(NAMESPACE + pdbname)
			if !exists {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

func (k *KubeVirtTestData) makeHandlerReady() {
	exists := false
	var obj interface{}
	// we need to wait until the daemonset exists
	for !exists {
		obj, exists, _ = k.controller.stores.DaemonSetCache.GetByKey(NAMESPACE + "/virt-handler")
		if exists {
			handler, _ := obj.(*appsv1.DaemonSet)
			handlerNew := handler.DeepCopy()
			handlerNew.Status.DesiredNumberScheduled = 1
			handlerNew.Status.NumberReady = 1
			k.mockQueue.ExpectAdds(1)
			k.daemonSetSource.Modify(handlerNew)
			k.mockQueue.Wait()
		}
	}
}

func (k *KubeVirtTestData) addDummyValidationWebhook() {
	version := fmt.Sprintf("rand-%s", rand.String(10))
	registry := fmt.Sprintf("rand-%s", rand.String(10))
	config := getConfig(registry, version)

	validationWebhook := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "virt-operator-tmp-webhook",
		},
	}

	injectMetadata(&validationWebhook.ObjectMeta, config)
	k.addValidatingWebhook(validationWebhook, nil)
}

func (k *KubeVirtTestData) addValidatingWebhook(wh *admissionregistrationv1.ValidatingWebhookConfiguration, kv *v1.KubeVirt) {
	k.mockQueue.ExpectAdds(1)
	if kv != nil {
		apply.SetGeneration(&kv.Status.Generations, wh)
	}

	k.validatingWebhookSource.Add(wh)
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) addInstallStrategy(config *util.KubeVirtDeploymentConfig) {
	// install strategy config
	resource, _ := install.NewInstallStrategyConfigMap(config, "openshift-monitoring", NAMESPACE)

	resource.Name = fmt.Sprintf("%s-%s", resource.Name, rand.String(10))

	injectMetadata(&resource.ObjectMeta, config)
	k.addConfigMap(resource)
}

func (k *KubeVirtTestData) addPodDisruptionBudgets(config *util.KubeVirtDeploymentConfig, apiDeployment *appsv1.Deployment, controller *appsv1.Deployment, kv *v1.KubeVirt) {
	minAvailable := intstr.FromInt(int(1))
	apiPodDisruptionBudget := &policyv1beta1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: apiDeployment.Namespace,
			Name:      apiDeployment.Name + "-pdb",
			Labels:    apiDeployment.Labels,
		},
		Spec: policyv1beta1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
			Selector:     apiDeployment.Spec.Selector,
		},
	}
	injectMetadata(&apiPodDisruptionBudget.ObjectMeta, config)
	k.addPodDisruptionBudget(apiPodDisruptionBudget, kv)
	controllerPodDisruptionBudget := &policyv1beta1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: controller.Namespace,
			Name:      controller.Name + "-pdb",
			Labels:    controller.Labels,
		},
		Spec: policyv1beta1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
			Selector:     controller.Spec.Selector,
		},
	}
	injectMetadata(&controllerPodDisruptionBudget.ObjectMeta, config)
	k.addPodDisruptionBudget(controllerPodDisruptionBudget, kv)
}

func (k *KubeVirtTestData) fakeNamespaceModificationEvent() {
	// Add modification event for namespace w/o the labels we need
	k.mockQueue.ExpectAdds(1)
	k.namespaceSource.Modify(&k8sv1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind: "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: NAMESPACE,
		},
	})
	k.mockQueue.Wait()
}

func (k *KubeVirtTestData) shouldExpectNamespacePatch() {
	k.kubeClient.Fake.PrependReactor("patch", "namespaces", k.genericPatchFunc())
}

func (k *KubeVirtTestData) addPodsWithIndividualConfigs(config *util.KubeVirtDeploymentConfig,
	configController *util.KubeVirtDeploymentConfig,
	configHandler *util.KubeVirtDeploymentConfig,
	shouldAddPodDisruptionBudgets bool,
	kv *v1.KubeVirt) {
	// we need at least one active pod for
	// virt-api
	// virt-controller
	// virt-handler
	apiDeployment, _ := components.NewApiServerDeployment(NAMESPACE, config.GetImageRegistry(), config.GetImagePrefix(), config.GetApiVersion(), "", "", config.GetImagePullPolicy(), config.GetVerbosity(), config.GetExtraEnv())

	pod := &k8sv1.Pod{
		ObjectMeta: apiDeployment.Spec.Template.ObjectMeta,
		Spec:       apiDeployment.Spec.Template.Spec,
		Status: k8sv1.PodStatus{
			Phase: k8sv1.PodRunning,
			ContainerStatuses: []k8sv1.ContainerStatus{
				{Ready: true, Name: "somecontainer"},
			},
		},
	}
	injectMetadata(&pod.ObjectMeta, config)
	pod.Name = "virt-api-xxxx"
	k.addPod(pod)

	controller, _ := components.NewControllerDeployment(NAMESPACE, configController.GetImageRegistry(), configController.GetImagePrefix(), configController.GetControllerVersion(), configController.GetLauncherVersion(), "", "", configController.GetImagePullPolicy(), configController.GetVerbosity(), configController.GetExtraEnv())
	pod = &k8sv1.Pod{
		ObjectMeta: controller.Spec.Template.ObjectMeta,
		Spec:       controller.Spec.Template.Spec,
		Status: k8sv1.PodStatus{
			Phase: k8sv1.PodRunning,
			ContainerStatuses: []k8sv1.ContainerStatus{
				{Ready: true, Name: "somecontainer"},
			},
		},
	}
	pod.Name = "virt-controller-xxxx"
	injectMetadata(&pod.ObjectMeta, configController)
	k.addPod(pod)

	handler, _ := components.NewHandlerDaemonSet(NAMESPACE, configHandler.GetImageRegistry(), configHandler.GetImagePrefix(), configHandler.GetHandlerVersion(), "", "", configController.GetLauncherVersion(), configHandler.GetImagePullPolicy(), configHandler.GetVerbosity(), configHandler.GetExtraEnv())
	pod = &k8sv1.Pod{
		ObjectMeta: handler.Spec.Template.ObjectMeta,
		Spec:       handler.Spec.Template.Spec,
		Status: k8sv1.PodStatus{
			Phase: k8sv1.PodRunning,
			ContainerStatuses: []k8sv1.ContainerStatus{
				{Ready: true, Name: "somecontainer"},
			},
		},
	}
	injectMetadata(&pod.ObjectMeta, configHandler)
	pod.Name = "virt-handler-xxxx"
	k.addPod(pod)

	if shouldAddPodDisruptionBudgets {
		k.addPodDisruptionBudgets(config, apiDeployment, controller, kv)
	}
}

func (k *KubeVirtTestData) addPodsWithOptionalPodDisruptionBudgets(config *util.KubeVirtDeploymentConfig, shouldAddPodDisruptionBudgets bool, kv *v1.KubeVirt) {
	k.addPodsWithIndividualConfigs(config, config, config, shouldAddPodDisruptionBudgets, kv)
}

func (k *KubeVirtTestData) addPodsAndPodDisruptionBudgets(config *util.KubeVirtDeploymentConfig, kv *v1.KubeVirt) {
	k.addPodsWithOptionalPodDisruptionBudgets(config, true, kv)
}

var _ = Describe("KubeVirt Operator", func() {

	BeforeEach(func() {
		err := os.Setenv(util.OperatorImageEnvName, fmt.Sprintf("%s/virt-operator:%s", "someregistry", "v9.9.9"))
		Expect(err).NotTo(HaveOccurred())
	})

	Context("On valid KubeVirt object", func() {

		It("Should not patch kubevirt namespace when labels are already defined", func(done Done) {
			defer close(done)

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			// Add fake namespace with labels predefined
			err := kvTestData.informers.Namespace.GetStore().Add(&k8sv1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind: "Namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: NAMESPACE,
					Labels: map[string]string{
						"openshift.io/cluster-monitoring": "true",
					},
				},
			})
			Expect(err).To(Not(HaveOccurred()), "could not add fake namespace to the store")
			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Generation: int64(1),
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Status: v1.KubeVirtStatus{
					Phase: v1.KubeVirtPhaseDeleted,
				},
			}
			// Add kubevirt deployment and mark everything as ready
			kvTestData.addKubeVirt(kv)
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.shouldExpectCreations()
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.addAll(kvTestData.defaultConfig, kv)
			kvTestData.addPodsAndPodDisruptionBudgets(kvTestData.defaultConfig, kv)
			kvTestData.makeApiAndControllerReady()
			kvTestData.makeHandlerReady()
			kvTestData.shouldExpectPatchesAndUpdates()

			// Now when the controller runs, if the namespace will be patched, the test will fail
			// because the patch is not expected here.
			kvTestData.controller.Execute()
		}, 60)

		It("should delete install strategy configmap once kubevirt install is deleted", func(done Done) {
			defer close(done)

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Status: v1.KubeVirtStatus{
					Phase: v1.KubeVirtPhaseDeleted,
				},
			}
			kv.DeletionTimestamp = now()
			util.UpdateConditionsDeleting(kv)

			kvTestData.shouldExpectInstallStrategyDeletion()

			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.shouldExpectKubeVirtUpdate(1)
			kvTestData.controller.Execute()
			kv = kvTestData.getLatestKubeVirt(kv)
			Expect(len(kv.ObjectMeta.Finalizers)).To(Equal(0))
		}, 30)

		It("should observe custom image tag in status during deploy", func(done Done) {
			defer close(done)
			defer GinkgoRecover()

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
					Generation: int64(1),
				},
				Spec: v1.KubeVirtSpec{
					ImageTag: "custom.tag",
				},
				Status: v1.KubeVirtStatus{
					Phase:           v1.KubeVirtPhaseDeployed,
					OperatorVersion: version.Get().String(),
				},
			}

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			customConfig := getConfig(kvTestData.defaultConfig.GetImageRegistry(), "custom.tag")

			kvTestData.fakeNamespaceModificationEvent()
			kvTestData.shouldExpectNamespacePatch()
			kvTestData.shouldExpectPatchesAndUpdates()
			kvTestData.addAll(customConfig, kv)
			// install strategy config
			kvTestData.addInstallStrategy(customConfig)
			kvTestData.addPodsAndPodDisruptionBudgets(customConfig, kv)

			kvTestData.makeApiAndControllerReady()
			kvTestData.makeHandlerReady()

			kvTestData.shouldExpectKubeVirtUpdateStatusVersion(1, customConfig)
			kvTestData.controller.Execute()
			kv = kvTestData.getLatestKubeVirt(kv)
			shouldExpectHCOConditions(kv, k8sv1.ConditionTrue, k8sv1.ConditionFalse, k8sv1.ConditionFalse)

		}, 60)

		It("delete temporary validation webhook once virt-api is deployed", func(done Done) {
			defer close(done)

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
					Generation: int64(1),
				},
				Status: v1.KubeVirtStatus{
					Phase:           v1.KubeVirtPhaseDeployed,
					OperatorVersion: version.Get().String(),
				},
			}
			kvTestData.defaultConfig.SetTargetDeploymentConfig(kv)
			kvTestData.defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)
			kvTestData.deleteFromCache = false

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addDummyValidationWebhook()
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.addAll(kvTestData.defaultConfig, kv)
			kvTestData.addPodsAndPodDisruptionBudgets(kvTestData.defaultConfig, kv)
			kvTestData.makeApiAndControllerReady()
			kvTestData.makeHandlerReady()

			kvTestData.shouldExpectDeletions()
			kvTestData.fakeNamespaceModificationEvent()
			kvTestData.shouldExpectNamespacePatch()
			kvTestData.shouldExpectPatchesAndUpdates()
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)

			kvTestData.controller.Execute()
			Expect(kvTestData.totalDeletions).To(Equal(1))

		}, 60)

		It("should do nothing if KubeVirt object is deployed", func(done Done) {
			defer close(done)

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
					Generation: int64(1),
				},
				Status: v1.KubeVirtStatus{
					Phase:           v1.KubeVirtPhaseDeployed,
					OperatorVersion: version.Get().String(),
				},
			}
			kvTestData.defaultConfig.SetTargetDeploymentConfig(kv)
			kvTestData.defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.addAll(kvTestData.defaultConfig, kv)
			kvTestData.addPodsAndPodDisruptionBudgets(kvTestData.defaultConfig, kv)
			kvTestData.makeApiAndControllerReady()
			kvTestData.makeHandlerReady()

			kvTestData.fakeNamespaceModificationEvent()
			kvTestData.shouldExpectNamespacePatch()
			kvTestData.shouldExpectPatchesAndUpdates()
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)

			kvTestData.controller.Execute()

		}, 60)

		It("should update KubeVirt object if generation IDs do not match", func(done Done) {
			defer close(done)

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
					Generation: int64(1),
				},
				Status: v1.KubeVirtStatus{
					Phase:           v1.KubeVirtPhaseDeployed,
					OperatorVersion: version.Get().String(),
				},
			}
			kvTestData.defaultConfig.SetTargetDeploymentConfig(kv)
			kvTestData.defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)

			kvTestData.defaultConfig.SetTargetDeploymentConfig(kv)
			kvTestData.defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.addAll(kvTestData.defaultConfig, kv)
			kvTestData.addPodsAndPodDisruptionBudgets(kvTestData.defaultConfig, kv)
			kvTestData.makeApiAndControllerReady()
			kvTestData.makeHandlerReady()

			kvTestData.fakeNamespaceModificationEvent()
			kvTestData.shouldExpectNamespacePatch()
			kvTestData.shouldExpectPatchesAndUpdates()
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)

			// invalidate all lastGeneration versions
			numGenerations := len(kv.Status.Generations)
			for i := range kv.Status.Generations {
				kv.Status.Generations[i].LastGeneration = -1
			}

			kvTestData.controller.Execute()

			// add one for the namespace
			Expect(kvTestData.totalPatches).To(Equal(numGenerations + 1))

			// all these resources should be tracked by there generation so everyone that has been added should now be patched
			// since they where the `lastGeneration` was set to -1 on the KubeVirt CR
			Expect(kvTestData.resourceChanges["mutatingwebhookconfigurations"][Patched]).To(Equal(kvTestData.resourceChanges["mutatingwebhookconfigurations"][Added]))
			Expect(kvTestData.resourceChanges["validatingwebhookconfigurations"][Patched]).To(Equal(kvTestData.resourceChanges["validatingwebhookconfigurations"][Added]))
			Expect(kvTestData.resourceChanges["deployements"][Patched]).To(Equal(kvTestData.resourceChanges["deployements"][Added]))
			Expect(kvTestData.resourceChanges["daemonsets"][Patched]).To(Equal(kvTestData.resourceChanges["daemonsets"][Added]))
		}, 60)

		It("should delete operator managed resources not in the deployed installstrategy", func() {
			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			defer GinkgoRecover()
			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
					Generation: int64(1),
				},
				Status: v1.KubeVirtStatus{
					Phase:           v1.KubeVirtPhaseDeployed,
					OperatorVersion: version.Get().String(),
				},
			}
			kvTestData.defaultConfig.SetTargetDeploymentConfig(kv)
			kvTestData.defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsDeploying(kv)
			util.UpdateConditionsCreated(kv)

			kvTestData.deleteFromCache = false

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.addAll(kvTestData.defaultConfig, kv)
			numResources := kvTestData.generateRandomResources()
			kvTestData.addPodsAndPodDisruptionBudgets(kvTestData.defaultConfig, kv)

			kvTestData.makeApiAndControllerReady()
			kvTestData.makeHandlerReady()

			kvTestData.shouldExpectDeletions()
			kvTestData.fakeNamespaceModificationEvent()
			kvTestData.shouldExpectNamespacePatch()
			kvTestData.shouldExpectPatchesAndUpdates()
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)

			kvTestData.controller.Execute()
			Expect(kvTestData.totalDeletions).To(Equal(numResources))
		}, 60)

		It("should fail if KubeVirt object already exists", func() {

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kv1 := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install-1",
					Namespace:  NAMESPACE,
					UID:        "11111111111",
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Status: v1.KubeVirtStatus{
					Phase:           v1.KubeVirtPhaseDeployed,
					OperatorVersion: "v0.0.0-master+$Format:%h$",
				},
			}

			kv2 := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-install-2",
					Namespace: NAMESPACE,
					UID:       "123123123",
				},
				Status: v1.KubeVirtStatus{},
			}

			kubecontroller.SetLatestApiVersionAnnotation(kv1)
			util.UpdateConditionsCreated(kv1)
			util.UpdateConditionsAvailable(kv1)
			kvTestData.addKubeVirt(kv1)
			kubecontroller.SetLatestApiVersionAnnotation(kv2)
			kvTestData.addKubeVirt(kv2)

			kvTestData.shouldExpectKubeVirtUpdateStatusFailureCondition(util.ConditionReasonDeploymentFailedExisting)

			kvTestData.controller.execute(fmt.Sprintf("%s/%s", kv2.Namespace, kv2.Name))

		}, 30)

		It("should generate install strategy creation job for update version", func(done Done) {
			defer close(done)

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			updatedVersion := "1.1.1"
			updatedRegistry := "otherregistry"

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Spec: v1.KubeVirtSpec{
					ImageTag:      updatedVersion,
					ImageRegistry: updatedRegistry,
				},
				Status: v1.KubeVirtStatus{
					Phase:           v1.KubeVirtPhaseDeployed,
					OperatorVersion: version.Get().String(),
				},
			}
			kvTestData.defaultConfig.SetTargetDeploymentConfig(kv)
			kvTestData.defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)

			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.shouldExpectJobCreation()
			kvTestData.controller.Execute()

		}, 30)

		It("should create an install strategy creation job with passthrough env vars, if provided in config", func(done Done) {
			defer close(done)

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			config := getConfig("registry", "v1.1.1")
			envKey := rand.String(10)
			envVal := rand.String(10)
			config.PassthroughEnvVars = map[string]string{envKey: envVal}
			job, err := kvTestData.controller.generateInstallStrategyJob(config)

			Expect(err).ToNot(HaveOccurred())
			Expect(job.Spec.Template.Spec.Containers[0].Env).To(ContainElement(k8sv1.EnvVar{Name: envKey, Value: envVal}))
		}, 30)

		It("should create an api server deployment with passthrough env vars, if provided in config", func(done Done) {
			defer close(done)
			config := getConfig("registry", "v1.1.1")
			envKey := rand.String(10)
			envVal := rand.String(10)
			config.PassthroughEnvVars = map[string]string{envKey: envVal}

			apiDeployment, err := components.NewApiServerDeployment(NAMESPACE, config.GetImageRegistry(), config.GetImagePrefix(), config.GetApiVersion(), "", "", config.GetImagePullPolicy(), config.GetVerbosity(), config.GetExtraEnv())

			Expect(err).ToNot(HaveOccurred())
			Expect(apiDeployment.Spec.Template.Spec.Containers[0].Env).To(ContainElement(k8sv1.EnvVar{Name: envKey, Value: envVal}))
		}, 30)

		It("should create a controller deployment with passthrough env vars, if provided in config", func(done Done) {
			defer close(done)
			config := getConfig("registry", "v1.1.1")
			envKey := rand.String(10)
			envVal := rand.String(10)
			config.PassthroughEnvVars = map[string]string{envKey: envVal}

			controllerDeployment, err := components.NewControllerDeployment(NAMESPACE, config.GetImageRegistry(), config.GetImagePrefix(), config.GetControllerVersion(), config.GetLauncherVersion(), "", "", config.GetImagePullPolicy(), config.GetVerbosity(), config.GetExtraEnv())

			Expect(err).ToNot(HaveOccurred())
			Expect(controllerDeployment.Spec.Template.Spec.Containers[0].Env).To(ContainElement(k8sv1.EnvVar{Name: envKey, Value: envVal}))
		}, 30)

		It("should create a handler daemonset with passthrough env vars, if provided in config", func(done Done) {
			defer close(done)
			config := getConfig("registry", "v1.1.1")
			envKey := rand.String(10)
			envVal := rand.String(10)
			config.PassthroughEnvVars = map[string]string{envKey: envVal}

			handlerDaemonset, err := components.NewHandlerDaemonSet(NAMESPACE, config.GetImageRegistry(), config.GetImagePrefix(), config.GetHandlerVersion(), "", "", config.GetLauncherVersion(), config.GetImagePullPolicy(), config.GetVerbosity(), config.GetExtraEnv())

			Expect(err).ToNot(HaveOccurred())
			Expect(handlerDaemonset.Spec.Template.Spec.Containers[0].Env).To(ContainElement(k8sv1.EnvVar{Name: envKey, Value: envVal}))
		}, 30)

		It("should generate install strategy creation job if no install strategy exists", func(done Done) {
			defer close(done)

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Status: v1.KubeVirtStatus{},
			}

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.shouldExpectJobCreation()
			kvTestData.controller.Execute()

		}, 30)

		It("should label install strategy creation job", func(done Done) {
			defer close(done)

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Status: v1.KubeVirtStatus{},
			}

			job, err := kvTestData.controller.generateInstallStrategyJob(util.GetTargetConfigFromKV(kv))
			Expect(err).ToNot(HaveOccurred())

			Expect(job.Spec.Template.ObjectMeta.Labels).Should(HaveKeyWithValue(v1.AppLabel, virtOperatorJobAppLabel))
		}, 30)

		It("should delete install strategy creation job if job has failed", func(done Done) {
			defer close(done)

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Status: v1.KubeVirtStatus{},
			}

			job, err := kvTestData.controller.generateInstallStrategyJob(util.GetTargetConfigFromKV(kv))
			Expect(err).ToNot(HaveOccurred())

			// will only create a new job after 10 seconds has passed.
			// this is just a simple mechanism to prevent spin loops
			// in the event that jobs are fast failing for some unknown reason.
			completionTime := time.Now().Add(time.Duration(-10) * time.Second)
			job.Status.CompletionTime = &metav1.Time{Time: completionTime}

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategyJob(job)

			kvTestData.shouldExpectJobDeletion()
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)

			kvTestData.controller.Execute()

		}, 30)

		It("should not delete completed install strategy creation job if job has failed less that 10 seconds ago", func(done Done) {
			defer GinkgoRecover()
			defer close(done)

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Status: v1.KubeVirtStatus{},
			}

			job, err := kvTestData.controller.generateInstallStrategyJob(util.GetTargetConfigFromKV(kv))
			Expect(err).ToNot(HaveOccurred())

			job.Status.CompletionTime = now()

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategyJob(job)

			kvTestData.shouldExpectKubeVirtUpdateStatus(1)

			kvTestData.controller.Execute()

		}, 30)

		It("should add resources on create", func() {

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-install",
					Namespace: NAMESPACE,
				},
			}
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)

			job, err := kvTestData.controller.generateInstallStrategyJob(util.GetTargetConfigFromKV(kv))
			Expect(err).ToNot(HaveOccurred())

			job.Status.CompletionTime = now()
			kvTestData.addInstallStrategyJob(job)

			// ensure completed jobs are garbage collected once install strategy
			// is loaded
			kvTestData.deleteFromCache = false
			kvTestData.shouldExpectJobDeletion()
			kvTestData.shouldExpectKubeVirtUpdate(1)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.shouldExpectCreations()

			kvTestData.controller.Execute()

			kv = kvTestData.getLatestKubeVirt(kv)
			Expect(kv.Status.Phase).To(Equal(v1.KubeVirtPhaseDeploying))
			Expect(len(kv.Status.Conditions)).To(Equal(3))
			Expect(len(kv.ObjectMeta.Finalizers)).To(Equal(1))
			shouldExpectHCOConditions(kv, k8sv1.ConditionFalse, k8sv1.ConditionTrue, k8sv1.ConditionFalse)

			// 3 in total are yet missing at this point
			// because waiting on controller, controller's PDB and virt-handler daemonset until API server deploys successfully
			expectedUncreatedResources := 3

			// 1 because a temporary validation webhook is created to block new CRDs until api server is deployed
			expectedTemporaryResources := 1

			Expect(kvTestData.totalAdds).To(Equal(resourceCount - expectedUncreatedResources + expectedTemporaryResources))

			Expect(len(kvTestData.controller.stores.ServiceAccountCache.List())).To(Equal(3))
			Expect(len(kvTestData.controller.stores.ClusterRoleCache.List())).To(Equal(7))
			Expect(len(kvTestData.controller.stores.ClusterRoleBindingCache.List())).To(Equal(5))
			Expect(len(kvTestData.controller.stores.RoleCache.List())).To(Equal(3))
			Expect(len(kvTestData.controller.stores.RoleBindingCache.List())).To(Equal(3))
			Expect(len(kvTestData.controller.stores.CrdCache.List())).To(Equal(8))
			Expect(len(kvTestData.controller.stores.ServiceCache.List())).To(Equal(3))
			Expect(len(kvTestData.controller.stores.DeploymentCache.List())).To(Equal(1))
			Expect(len(kvTestData.controller.stores.DaemonSetCache.List())).To(Equal(0))
			Expect(len(kvTestData.controller.stores.ValidationWebhookCache.List())).To(Equal(3))
			Expect(len(kvTestData.controller.stores.PodDisruptionBudgetCache.List())).To(Equal(1))
			Expect(len(kvTestData.controller.stores.SCCCache.List())).To(Equal(3))
			Expect(len(kvTestData.controller.stores.ServiceMonitorCache.List())).To(Equal(1))
			Expect(len(kvTestData.controller.stores.PrometheusRuleCache.List())).To(Equal(1))

			Expect(kvTestData.resourceChanges["poddisruptionbudgets"][Added]).To(Equal(1))

		}, 60)

		It("should pause rollback until api server is rolled over.", func(done Done) {
			defer close(done)
			defer GinkgoRecover()

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			rollbackConfig := getConfig("otherregistry", "9.9.7")

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Spec: v1.KubeVirtSpec{
					ImageTag:      rollbackConfig.GetKubeVirtVersion(),
					ImageRegistry: rollbackConfig.GetImageRegistry(),
				},
				Status: v1.KubeVirtStatus{
					Phase:           v1.KubeVirtPhaseDeployed,
					OperatorVersion: version.Get().String(),
				},
			}
			kvTestData.defaultConfig.SetTargetDeploymentConfig(kv)
			kvTestData.defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.addInstallStrategy(rollbackConfig)

			kvTestData.addAll(kvTestData.defaultConfig, kv)
			kvTestData.addPodsAndPodDisruptionBudgets(kvTestData.defaultConfig, kv)

			kvTestData.makeApiAndControllerReady()
			kvTestData.makeHandlerReady()

			kvTestData.addToCache = false
			kvTestData.shouldExpectRbacBackupCreations()
			kvTestData.shouldExpectPatchesAndUpdates()
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)

			kvTestData.controller.Execute()

			kv = kvTestData.getLatestKubeVirt(kv)
			// conditions should reflect an ongoing update
			shouldExpectHCOConditions(kv, k8sv1.ConditionTrue, k8sv1.ConditionTrue, k8sv1.ConditionTrue)

			// on rollback or create, api server must be online first before controllers and daemonset.
			// On rollback this prevents someone from posting invalid specs to
			// the cluster from newer versions when an older version is being deployed.
			// On create this prevents invalid specs from entering the cluster
			// while controllers are available to process them.

			// 4 because 2 for virt-controller service and deployment,
			// 1 because of the pdb of virt-controller
			// and another 1 because of the namespace was not patched yet.
			Expect(kvTestData.totalPatches).To(Equal(patchCount - 4))
			// 2 for virt-controller and pdb
			Expect(kvTestData.totalUpdates).To(Equal(updateCount))

			Expect(kvTestData.resourceChanges["poddisruptionbudgets"][Patched]).To(Equal(1))
		}, 60)

		It("should pause update after daemonsets are rolled over", func(done Done) {
			defer close(done)

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			updatedConfig := getConfig("otherregistry", "9.9.10")

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Spec: v1.KubeVirtSpec{
					ImageTag:      updatedConfig.GetKubeVirtVersion(),
					ImageRegistry: updatedConfig.GetImageRegistry(),
				},
				Status: v1.KubeVirtStatus{
					Phase:           v1.KubeVirtPhaseDeployed,
					OperatorVersion: version.Get().String(),
				},
			}
			kvTestData.defaultConfig.SetTargetDeploymentConfig(kv)
			kvTestData.defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.addInstallStrategy(updatedConfig)

			kvTestData.addAll(kvTestData.defaultConfig, kv)
			kvTestData.addPodsAndPodDisruptionBudgets(kvTestData.defaultConfig, kv)

			kvTestData.makeApiAndControllerReady()
			kvTestData.makeHandlerReady()

			kvTestData.addToCache = false
			kvTestData.shouldExpectRbacBackupCreations()
			kvTestData.shouldExpectPatchesAndUpdates()
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)

			kvTestData.controller.Execute()

			kv = kvTestData.getLatestKubeVirt(kv)
			// conditions should reflect an ongoing update
			shouldExpectHCOConditions(kv, k8sv1.ConditionTrue, k8sv1.ConditionTrue, k8sv1.ConditionTrue)

			Expect(kvTestData.totalUpdates).To(Equal(updateCount))

			// daemonset, controller and apiserver pods are updated in this order.
			// this prevents the new API from coming online until the controllers can manage it.
			// The PDBs will prevent updated pods from getting "ready", so update should pause after
			//   daemonsets and before controller and namespace

			// 5 because virt-controller, virt-api, PDBs and the namespace are not patched
			Expect(kvTestData.totalPatches).To(Equal(patchCount - 5))

			// Make sure the 5 unpatched are as expected
			Expect(kvTestData.resourceChanges["deployments"][Patched]).To(Equal(0))          // virt-controller and virt-api unpatched
			Expect(kvTestData.resourceChanges["poddisruptionbudgets"][Patched]).To(Equal(0)) // PDBs unpatched
			Expect(kvTestData.resourceChanges["namespace"][Patched]).To(Equal(0))            // namespace unpatched
		}, 60)

		It("should pause update after controllers are rolled over", func(done Done) {
			defer close(done)

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			updatedConfig := getConfig("otherregistry", "9.9.10")

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Spec: v1.KubeVirtSpec{
					ImageTag:      updatedConfig.GetKubeVirtVersion(),
					ImageRegistry: updatedConfig.GetImageRegistry(),
				},
				Status: v1.KubeVirtStatus{
					Phase:           v1.KubeVirtPhaseDeployed,
					OperatorVersion: version.Get().String(),
				},
			}
			kvTestData.defaultConfig.SetTargetDeploymentConfig(kv)
			kvTestData.defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.addInstallStrategy(updatedConfig)

			kvTestData.addAll(kvTestData.defaultConfig, kv)
			// Create virt-api and virt-controller under kvTestData.defaultConfig,
			// but use updatedConfig for virt-handler (hack) to avoid pausing after daemonsets
			kvTestData.addPodsWithIndividualConfigs(kvTestData.defaultConfig, kvTestData.defaultConfig, updatedConfig, true, kv)

			kvTestData.makeApiAndControllerReady()
			kvTestData.makeHandlerReady()

			kvTestData.addToCache = false
			kvTestData.shouldExpectRbacBackupCreations()
			kvTestData.shouldExpectPatchesAndUpdates()
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)

			kvTestData.controller.Execute()

			kv = kvTestData.getLatestKubeVirt(kv)
			// conditions should reflect an ongoing update
			shouldExpectHCOConditions(kv, k8sv1.ConditionTrue, k8sv1.ConditionTrue, k8sv1.ConditionTrue)

			Expect(kvTestData.totalUpdates).To(Equal(updateCount))

			// The update was hacked to avoid pausing after rolling out the daemonsets (virt-handler)
			// That will allow both daemonset and controller pods to get patched before the pause.

			// 3 because virt-api, PDB and the namespace should not be patched
			Expect(kvTestData.totalPatches).To(Equal(patchCount - 3))

			// Make sure the 3 unpatched are as expected
			Expect(kvTestData.resourceChanges["deployments"][Patched]).To(Equal(1))          // virt-operator patched, virt-api unpatched
			Expect(kvTestData.resourceChanges["poddisruptionbudgets"][Patched]).To(Equal(1)) // 1 of 2 PDBs patched
			Expect(kvTestData.resourceChanges["namespace"][Patched]).To(Equal(0))            // namespace unpatched
		}, 60)

		It("should update kubevirt resources when Operator version changes if no imageTag and imageRegistry is explicitly set.", func() {
			os.Setenv(util.OperatorImageEnvName, fmt.Sprintf("%s/virt-operator:%s", "otherregistry", "1.1.1"))
			updatedConfig := getConfig("", "")

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Spec: v1.KubeVirtSpec{},
				Status: v1.KubeVirtStatus{
					Phase:           v1.KubeVirtPhaseDeployed,
					OperatorVersion: version.Get().String(),
				},
			}
			kvTestData.defaultConfig.SetTargetDeploymentConfig(kv)
			kvTestData.defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.addInstallStrategy(updatedConfig)

			kvTestData.addAll(kvTestData.defaultConfig, kv)
			kvTestData.addPodsAndPodDisruptionBudgets(kvTestData.defaultConfig, kv)

			// pods for the new version are added so this test won't
			// wait for daemonsets to rollover before updating/patching
			// all resources.
			kvTestData.addPodsWithOptionalPodDisruptionBudgets(updatedConfig, false, kv)

			kvTestData.makeApiAndControllerReady()
			kvTestData.makeHandlerReady()

			kvTestData.shouldExpectPatchesAndUpdates()
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.fakeNamespaceModificationEvent()
			kvTestData.shouldExpectNamespacePatch()

			kvTestData.controller.Execute()

			kv = kvTestData.getLatestKubeVirt(kv)
			// conditions should reflect a successful update
			shouldExpectHCOConditions(kv, k8sv1.ConditionTrue, k8sv1.ConditionFalse, k8sv1.ConditionFalse)

			Expect(kvTestData.totalPatches).To(Equal(patchCount))
			Expect(kvTestData.totalUpdates).To(Equal(updateCount))

			// ensure every resource is either patched or updated
			// + 1 is for the namespace patch which we don't consider as a resource we own.
			Expect(kvTestData.totalUpdates + kvTestData.totalPatches).To(Equal(resourceCount + 1))

			Expect(kvTestData.resourceChanges["poddisruptionbudgets"][Patched]).To(Equal(2))

		}, 60)

		It("should update resources when changing KubeVirt version.", func() {
			updatedConfig := getConfig("otherregistry", "1.1.1")

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Spec: v1.KubeVirtSpec{
					ImageTag:      updatedConfig.GetKubeVirtVersion(),
					ImageRegistry: updatedConfig.GetImageRegistry(),
				},
				Status: v1.KubeVirtStatus{
					Phase:           v1.KubeVirtPhaseDeployed,
					OperatorVersion: version.Get().String(),
				},
			}
			kvTestData.defaultConfig.SetTargetDeploymentConfig(kv)
			kvTestData.defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.addInstallStrategy(updatedConfig)

			kvTestData.addAll(kvTestData.defaultConfig, kv)
			kvTestData.addPodsAndPodDisruptionBudgets(kvTestData.defaultConfig, kv)

			// pods for the new version are added so this test won't
			// wait for daemonsets to rollover before updating/patching
			// all resources.
			kvTestData.addPodsWithOptionalPodDisruptionBudgets(updatedConfig, false, kv)

			kvTestData.makeApiAndControllerReady()
			kvTestData.makeHandlerReady()

			kvTestData.shouldExpectPatchesAndUpdates()
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.fakeNamespaceModificationEvent()
			kvTestData.shouldExpectNamespacePatch()

			kvTestData.controller.Execute()

			kv = kvTestData.getLatestKubeVirt(kv)
			// conditions should reflect a successful update
			shouldExpectHCOConditions(kv, k8sv1.ConditionTrue, k8sv1.ConditionFalse, k8sv1.ConditionFalse)

			Expect(kvTestData.totalPatches).To(Equal(patchCount))
			Expect(kvTestData.totalUpdates).To(Equal(updateCount))

			// ensure every resource is either patched or updated
			// + 1 is for the namespace patch which we don't consider as a resource we own.
			Expect(kvTestData.totalUpdates + kvTestData.totalPatches).To(Equal(resourceCount + 1))

		}, 60)

		It("should patch poddisruptionbudgets when changing KubeVirt version.", func() {
			updatedConfig := getConfig("otherregistry", "1.1.1")

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Spec: v1.KubeVirtSpec{
					ImageTag:      updatedConfig.GetKubeVirtVersion(),
					ImageRegistry: updatedConfig.GetImageRegistry(),
				},
				Status: v1.KubeVirtStatus{
					Phase: v1.KubeVirtPhaseDeployed,
					Conditions: []v1.KubeVirtCondition{
						{
							Type:    v1.KubeVirtConditionCreated,
							Status:  k8sv1.ConditionTrue,
							Reason:  util.ConditionReasonDeploymentCreated,
							Message: "All resources were created.",
						},
						{
							Type:    v1.KubeVirtConditionAvailable,
							Status:  k8sv1.ConditionTrue,
							Reason:  util.ConditionReasonDeploymentReady,
							Message: "All components are ready.",
						},
					},
					OperatorVersion: version.Get().String(),
				},
			}
			kvTestData.defaultConfig.SetTargetDeploymentConfig(kv)
			kvTestData.defaultConfig.SetObservedDeploymentConfig(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.addInstallStrategy(updatedConfig)

			kvTestData.addAll(kvTestData.defaultConfig, kv)
			kvTestData.addPodsAndPodDisruptionBudgets(kvTestData.defaultConfig, kv)

			// pods for the new version are added so this test won't
			// wait for daemonsets to rollover before updating/patching
			// all resources.
			kvTestData.addPodsWithOptionalPodDisruptionBudgets(updatedConfig, false, kv)

			kvTestData.makeApiAndControllerReady()
			kvTestData.makeHandlerReady()

			kvTestData.shouldExpectPatchesAndUpdates()
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)

			kvTestData.controller.Execute()

			Expect(kvTestData.resourceChanges["poddisruptionbudgets"][Patched]).To(Equal(2))

		}, 60)

		It("should remove resources on deletion", func() {

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-install",
					Namespace: NAMESPACE,
				},
			}
			kv.DeletionTimestamp = now()
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)

			// create all resources which should be deleted
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.addAll(kvTestData.defaultConfig, kv)

			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.shouldExpectDeletions()
			kvTestData.shouldExpectInstallStrategyDeletion()

			kvTestData.controller.Execute()

			// Note: in real life during the first execution loop very probably only CRDs are deleted,
			// because that takes some time (see the check that the crd store is empty before going on with deletions)
			// But in this test the deletion succeeds immediately, so everything is deleted on first try
			Expect(kvTestData.totalDeletions).To(Equal(resourceCount))

			kv = kvTestData.getLatestKubeVirt(kv)
			Expect(kv.Status.Phase).To(Equal(v1.KubeVirtPhaseDeleted))
			Expect(len(kv.Status.Conditions)).To(Equal(3))
			shouldExpectHCOConditions(kv, k8sv1.ConditionFalse, k8sv1.ConditionFalse, k8sv1.ConditionTrue)
		}, 60)

		It("should remove poddisruptionbudgets on deletion", func() {

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-install",
					Namespace: NAMESPACE,
				},
			}
			kv.DeletionTimestamp = now()
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)

			// create all resources which should be deleted
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.addAll(kvTestData.defaultConfig, kv)

			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.shouldExpectDeletions()
			kvTestData.shouldExpectInstallStrategyDeletion()

			kvTestData.controller.Execute()

			Expect(kvTestData.resourceChanges["poddisruptionbudgets"][Deleted]).To(Equal(2))
		}, 60)
	})

	Context("when the monitor namespace does not exist", func() {
		It("should not create ServiceMonitor resources", func() {

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
			}
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)

			// install strategy config
			resource, _ := install.NewInstallStrategyConfigMap(kvTestData.defaultConfig, "", NAMESPACE)
			resource.Name = fmt.Sprintf("%s-%s", resource.Name, rand.String(10))
			kvTestData.addResource(resource, kvTestData.defaultConfig, nil)

			job, err := kvTestData.controller.generateInstallStrategyJob(util.GetTargetConfigFromKV(kv))
			Expect(err).ToNot(HaveOccurred())

			job.Status.CompletionTime = now()
			kvTestData.addInstallStrategyJob(job)

			// ensure completed jobs are garbage collected once install strategy
			// is loaded
			kvTestData.deleteFromCache = false
			kvTestData.shouldExpectJobDeletion()
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.shouldExpectCreations()

			kvTestData.controller.Execute()

			Expect(len(kvTestData.controller.stores.RoleCache.List())).To(Equal(2))
			Expect(len(kvTestData.controller.stores.RoleBindingCache.List())).To(Equal(2))
			Expect(len(kvTestData.controller.stores.ServiceMonitorCache.List())).To(Equal(0))
		}, 30)
	})

	Context("On install strategy dump", func() {
		It("should generate latest install strategy and post as config map", func(done Done) {
			defer close(done)

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			config, err := util.GetConfigFromEnv()
			Expect(err).ToNot(HaveOccurred())

			kvTestData.kubeClient.Fake.PrependReactor("create", "configmaps", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				create, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())

				configMap := create.GetObject().(*k8sv1.ConfigMap)
				Expect(configMap.GenerateName).To(Equal("kubevirt-install-strategy-"))

				version, ok := configMap.ObjectMeta.Annotations[v1.InstallStrategyVersionAnnotation]
				Expect(ok).To(BeTrue())
				Expect(version).To(Equal(config.GetKubeVirtVersion()))

				registry, ok := configMap.ObjectMeta.Annotations[v1.InstallStrategyRegistryAnnotation]
				Expect(ok).To(BeTrue())
				Expect(registry).To(Equal(config.GetImageRegistry()))

				id, ok := configMap.ObjectMeta.Annotations[v1.InstallStrategyIdentifierAnnotation]
				Expect(ok).To(BeTrue())
				Expect(id).To(Equal(config.GetDeploymentID()))

				_, ok = configMap.Data["manifests"]
				Expect(ok).To(BeTrue())

				return true, create.GetObject(), nil
			})

			// This generates and posts the install strategy config map
			install.DumpInstallStrategyToConfigMap(kvTestData.virtClient, NAMESPACE)
		}, 30)
	})
})

func now() *metav1.Time {
	now := metav1.Now()
	return &now
}

func getSCC() secv1.SecurityContextConstraints {
	return secv1.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{
			Name: "privileged",
		},
		Users: []string{
			"someUser",
		},
	}
}

func getConfig(registry, version string) *util.KubeVirtDeploymentConfig {
	return util.GetTargetConfigFromKV(&v1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: NAMESPACE,
		},
		Spec: v1.KubeVirtSpec{
			ImageRegistry: registry,
			ImageTag:      version,
		},
	})
}

func syncCaches(stop chan struct{}, kvInformer cache.SharedIndexInformer, informers util.Informers) {
	go kvInformer.Run(stop)
	go informers.ServiceAccount.Run(stop)
	go informers.ClusterRole.Run(stop)
	go informers.ClusterRoleBinding.Run(stop)
	go informers.Role.Run(stop)
	go informers.RoleBinding.Run(stop)
	go informers.Crd.Run(stop)
	go informers.Service.Run(stop)
	go informers.Deployment.Run(stop)
	go informers.DaemonSet.Run(stop)
	go informers.ValidationWebhook.Run(stop)
	go informers.MutatingWebhook.Run(stop)
	go informers.APIService.Run(stop)
	go informers.SCC.Run(stop)
	go informers.InstallStrategyJob.Run(stop)
	go informers.InstallStrategyConfigMap.Run(stop)
	go informers.InfrastructurePod.Run(stop)
	go informers.PodDisruptionBudget.Run(stop)
	go informers.ServiceMonitor.Run(stop)
	go informers.Namespace.Run(stop)
	go informers.PrometheusRule.Run(stop)
	go informers.Secrets.Run(stop)
	go informers.ConfigMap.Run(stop)

	Expect(cache.WaitForCacheSync(stop, kvInformer.HasSynced)).To(BeTrue())

	cache.WaitForCacheSync(stop, informers.ServiceAccount.HasSynced)
	cache.WaitForCacheSync(stop, informers.ClusterRole.HasSynced)
	cache.WaitForCacheSync(stop, informers.ClusterRoleBinding.HasSynced)
	cache.WaitForCacheSync(stop, informers.Role.HasSynced)
	cache.WaitForCacheSync(stop, informers.RoleBinding.HasSynced)
	cache.WaitForCacheSync(stop, informers.Crd.HasSynced)
	cache.WaitForCacheSync(stop, informers.Service.HasSynced)
	cache.WaitForCacheSync(stop, informers.Deployment.HasSynced)
	cache.WaitForCacheSync(stop, informers.DaemonSet.HasSynced)
	cache.WaitForCacheSync(stop, informers.ValidationWebhook.HasSynced)
	cache.WaitForCacheSync(stop, informers.MutatingWebhook.HasSynced)
	cache.WaitForCacheSync(stop, informers.APIService.HasSynced)
	cache.WaitForCacheSync(stop, informers.SCC.HasSynced)
	cache.WaitForCacheSync(stop, informers.InstallStrategyJob.HasSynced)
	cache.WaitForCacheSync(stop, informers.InstallStrategyConfigMap.HasSynced)
	cache.WaitForCacheSync(stop, informers.InfrastructurePod.HasSynced)
	cache.WaitForCacheSync(stop, informers.PodDisruptionBudget.HasSynced)
	cache.WaitForCacheSync(stop, informers.ServiceMonitor.HasSynced)
	cache.WaitForCacheSync(stop, informers.Namespace.HasSynced)
	cache.WaitForCacheSync(stop, informers.PrometheusRule.HasSynced)
	cache.WaitForCacheSync(stop, informers.Secrets.HasSynced)
	cache.WaitForCacheSync(stop, informers.ConfigMap.HasSynced)
}

func injectMetadata(objectMeta *metav1.ObjectMeta, config *util.KubeVirtDeploymentConfig) {
	if config == nil {
		return
	}
	if objectMeta.Labels == nil {
		objectMeta.Labels = make(map[string]string)
	}
	objectMeta.Labels[v1.ManagedByLabel] = v1.ManagedByLabelOperatorValue

	if config.GetProductVersion() != "" {
		objectMeta.Labels[v1.AppVersionLabel] = config.GetProductVersion()
	}
	if config.GetProductName() != "" {
		objectMeta.Labels[v1.AppPartOfLabel] = config.GetProductName()
	}

	if objectMeta.Annotations == nil {
		objectMeta.Annotations = make(map[string]string)
	}
	objectMeta.Annotations[v1.InstallStrategyVersionAnnotation] = config.GetKubeVirtVersion()
	objectMeta.Annotations[v1.InstallStrategyRegistryAnnotation] = config.GetImageRegistry()
	objectMeta.Annotations[v1.InstallStrategyIdentifierAnnotation] = config.GetDeploymentID()
	objectMeta.Annotations[v1.KubeVirtGenerationAnnotation] = "1"

	objectMeta.Labels[v1.AppComponentLabel] = v1.AppComponent
}

func shouldExpectHCOConditions(kv *v1.KubeVirt, available k8sv1.ConditionStatus, progressing k8sv1.ConditionStatus, degraded k8sv1.ConditionStatus) {
	getType := func(c v1.KubeVirtCondition) v1.KubeVirtConditionType { return c.Type }
	getStatus := func(c v1.KubeVirtCondition) k8sv1.ConditionStatus { return c.Status }
	Expect(kv.Status.Conditions).To(ContainElement(
		And(
			WithTransform(getType, Equal(v1.KubeVirtConditionAvailable)),
			WithTransform(getStatus, Equal(available)),
		),
	))
	Expect(kv.Status.Conditions).To(ContainElement(
		And(
			WithTransform(getType, Equal(v1.KubeVirtConditionProgressing)),
			WithTransform(getStatus, Equal(progressing)),
		),
	))
	Expect(kv.Status.Conditions).To(ContainElement(
		And(
			WithTransform(getType, Equal(v1.KubeVirtConditionDegraded)),
			WithTransform(getStatus, Equal(degraded)),
		),
	))
}
