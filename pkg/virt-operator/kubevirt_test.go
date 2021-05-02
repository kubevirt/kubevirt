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
)

var _ = Describe("KubeVirt Operator", func() {

	var ctrl *gomock.Controller
	var kvInterface *kubecli.MockKubeVirtInterface
	var kvSource *framework.FakeControllerSource
	var kvInformer cache.SharedIndexInformer
	var apiServiceClient *install.MockAPIServiceInterface

	var serviceAccountSource *framework.FakeControllerSource
	var clusterRoleSource *framework.FakeControllerSource
	var clusterRoleBindingSource *framework.FakeControllerSource
	var roleSource *framework.FakeControllerSource
	var roleBindingSource *framework.FakeControllerSource
	var crdSource *framework.FakeControllerSource
	var serviceSource *framework.FakeControllerSource
	var deploymentSource *framework.FakeControllerSource
	var daemonSetSource *framework.FakeControllerSource
	var validatingWebhookSource *framework.FakeControllerSource
	var mutatingWebhookSource *framework.FakeControllerSource
	var apiserviceSource *framework.FakeControllerSource
	var sccSource *framework.FakeControllerSource
	var installStrategyConfigMapSource *framework.FakeControllerSource
	var installStrategyJobSource *framework.FakeControllerSource
	var infrastructurePodSource *framework.FakeControllerSource
	var podDisruptionBudgetSource *framework.FakeControllerSource
	var serviceMonitorSource *framework.FakeControllerSource
	var namespaceSource *framework.FakeControllerSource
	var prometheusRuleSource *framework.FakeControllerSource
	var secretsSource *framework.FakeControllerSource
	var configMapSource *framework.FakeControllerSource

	var stop chan struct{}
	var controller *KubeVirtController

	var recorder *record.FakeRecorder

	var mockQueue *testutils.MockWorkQueue
	var virtClient *kubecli.MockKubevirtClient
	var kubeClient *fake.Clientset
	var secClient *secv1fake.FakeSecurityV1
	var extClient *extclientfake.Clientset
	var promClient *promclientfake.Clientset

	var informers util.Informers
	var stores util.Stores

	NAMESPACE := "kubevirt-test"

	getConfig := func(registry, version string) *util.KubeVirtDeploymentConfig {
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

	var totalAdds int
	var totalUpdates int
	var totalPatches int
	var totalDeletions int
	var resourceChanges map[string]map[string]int

	resourceCount := 53
	patchCount := 34
	updateCount := 20

	deleteFromCache := true
	addToCache := true

	syncCaches := func(stop chan struct{}) {
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

	getSCC := func() secv1.SecurityContextConstraints {
		return secv1.SecurityContextConstraints{
			ObjectMeta: metav1.ObjectMeta{
				Name: "privileged",
			},
			Users: []string{
				"someUser",
			},
		}
	}

	var defaultConfig *util.KubeVirtDeploymentConfig
	BeforeEach(func() {

		err := os.Setenv(util.OperatorImageEnvName, fmt.Sprintf("%s/virt-operator:%s", "someregistry", "v9.9.9"))
		Expect(err).NotTo(HaveOccurred())
		defaultConfig = getConfig("", "")

		totalAdds = 0
		totalUpdates = 0
		totalPatches = 0
		totalDeletions = 0
		resourceChanges = make(map[string]map[string]int)
		deleteFromCache = true
		addToCache = true

		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		kvInterface = kubecli.NewMockKubeVirtInterface(ctrl)
		apiServiceClient = install.NewMockAPIServiceInterface(ctrl)

		kvInformer, kvSource = testutils.NewFakeInformerFor(&v1.KubeVirt{})
		recorder = record.NewFakeRecorder(100)

		informers.ServiceAccount, serviceAccountSource = testutils.NewFakeInformerFor(&k8sv1.ServiceAccount{})
		stores.ServiceAccountCache = informers.ServiceAccount.GetStore()

		informers.ClusterRole, clusterRoleSource = testutils.NewFakeInformerFor(&rbacv1.ClusterRole{})
		stores.ClusterRoleCache = informers.ClusterRole.GetStore()

		informers.ClusterRoleBinding, clusterRoleBindingSource = testutils.NewFakeInformerFor(&rbacv1.ClusterRoleBinding{})
		stores.ClusterRoleBindingCache = informers.ClusterRoleBinding.GetStore()

		informers.Role, roleSource = testutils.NewFakeInformerFor(&rbacv1.Role{})
		stores.RoleCache = informers.Role.GetStore()

		informers.RoleBinding, roleBindingSource = testutils.NewFakeInformerFor(&rbacv1.RoleBinding{})
		stores.RoleBindingCache = informers.RoleBinding.GetStore()

		informers.Crd, crdSource = testutils.NewFakeInformerFor(&extv1.CustomResourceDefinition{})
		stores.CrdCache = informers.Crd.GetStore()

		informers.Service, serviceSource = testutils.NewFakeInformerFor(&k8sv1.Service{})
		stores.ServiceCache = informers.Service.GetStore()

		informers.Deployment, deploymentSource = testutils.NewFakeInformerFor(&appsv1.Deployment{})
		stores.DeploymentCache = informers.Deployment.GetStore()

		informers.DaemonSet, daemonSetSource = testutils.NewFakeInformerFor(&appsv1.DaemonSet{})
		stores.DaemonSetCache = informers.DaemonSet.GetStore()

		informers.ValidationWebhook, validatingWebhookSource = testutils.NewFakeInformerFor(&admissionregistrationv1.ValidatingWebhookConfiguration{})
		stores.ValidationWebhookCache = informers.ValidationWebhook.GetStore()
		informers.MutatingWebhook, mutatingWebhookSource = testutils.NewFakeInformerFor(&admissionregistrationv1.MutatingWebhookConfiguration{})
		stores.MutatingWebhookCache = informers.MutatingWebhook.GetStore()
		informers.APIService, apiserviceSource = testutils.NewFakeInformerFor(&apiregv1.APIService{})
		stores.APIServiceCache = informers.APIService.GetStore()

		informers.SCC, sccSource = testutils.NewFakeInformerFor(&secv1.SecurityContextConstraints{})
		stores.SCCCache = informers.SCC.GetStore()

		informers.InstallStrategyConfigMap, installStrategyConfigMapSource = testutils.NewFakeInformerFor(&k8sv1.ConfigMap{})
		stores.InstallStrategyConfigMapCache = informers.InstallStrategyConfigMap.GetStore()

		informers.InstallStrategyJob, installStrategyJobSource = testutils.NewFakeInformerFor(&batchv1.Job{})
		stores.InstallStrategyJobCache = informers.InstallStrategyJob.GetStore()

		informers.InfrastructurePod, infrastructurePodSource = testutils.NewFakeInformerFor(&k8sv1.Pod{})
		stores.InfrastructurePodCache = informers.InfrastructurePod.GetStore()

		informers.PodDisruptionBudget, podDisruptionBudgetSource = testutils.NewFakeInformerFor(&policyv1beta1.PodDisruptionBudget{})
		stores.PodDisruptionBudgetCache = informers.PodDisruptionBudget.GetStore()

		informers.Namespace, namespaceSource = testutils.NewFakeInformerWithIndexersFor(
			&k8sv1.Namespace{}, cache.Indexers{
				"namespace_name": func(obj interface{}) ([]string, error) {
					return []string{obj.(*k8sv1.Namespace).GetName()}, nil
				},
			})
		stores.NamespaceCache = informers.Namespace.GetStore()

		// test OpenShift components
		stores.IsOnOpenshift = true

		informers.ServiceMonitor, serviceMonitorSource = testutils.NewFakeInformerFor(&promv1.ServiceMonitor{Spec: promv1.ServiceMonitorSpec{}})
		stores.ServiceMonitorCache = informers.ServiceMonitor.GetStore()
		stores.ServiceMonitorEnabled = true

		informers.PrometheusRule, prometheusRuleSource = testutils.NewFakeInformerFor(&promv1.PrometheusRule{Spec: promv1.PrometheusRuleSpec{}})
		stores.PrometheusRuleCache = informers.PrometheusRule.GetStore()
		stores.PrometheusRulesEnabled = true

		informers.Secrets, secretsSource = testutils.NewFakeInformerFor(&k8sv1.Secret{})
		stores.SecretCache = informers.Secrets.GetStore()
		informers.ConfigMap, configMapSource = testutils.NewFakeInformerFor(&k8sv1.ConfigMap{})
		stores.ConfigMapCache = informers.ConfigMap.GetStore()

		controller = NewKubeVirtController(virtClient, apiServiceClient, kvInformer, recorder, stores, informers, NAMESPACE)

		// Wrap our workqueue to have a way to detect when we are done processing updates
		mockQueue = testutils.NewMockWorkQueue(controller.queue)
		controller.queue = mockQueue

		// Set up mock client
		virtClient.EXPECT().KubeVirt(NAMESPACE).Return(kvInterface).AnyTimes()
		kubeClient = fake.NewSimpleClientset()
		secClient = &secv1fake.FakeSecurityV1{
			Fake: &fake.NewSimpleClientset().Fake,
		}
		extClient = extclientfake.NewSimpleClientset()

		promClient = promclientfake.NewSimpleClientset()

		virtClient.EXPECT().AdmissionregistrationV1().Return(kubeClient.AdmissionregistrationV1()).AnyTimes()
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().BatchV1().Return(kubeClient.BatchV1()).AnyTimes()
		virtClient.EXPECT().RbacV1().Return(kubeClient.RbacV1()).AnyTimes()
		virtClient.EXPECT().AppsV1().Return(kubeClient.AppsV1()).AnyTimes()
		virtClient.EXPECT().SecClient().Return(secClient).AnyTimes()
		virtClient.EXPECT().ExtensionsClient().Return(extClient).AnyTimes()
		virtClient.EXPECT().PolicyV1beta1().Return(kubeClient.PolicyV1beta1()).AnyTimes()
		virtClient.EXPECT().PrometheusClient().Return(promClient).AnyTimes()

		// Make sure that all unexpected calls to kubeClient will fail
		kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			if action.GetVerb() == "get" && action.GetResource().Resource == "secrets" {
				return true, nil, errors.NewNotFound(schema.GroupResource{Group: "", Resource: "secrets"}, "whatever")
			}
			if action.GetVerb() == "get" && action.GetResource().Resource == "validatingwebhookconfigurations" {
				return true, nil, errors.NewNotFound(schema.GroupResource{Group: "", Resource: "validatingwebhookconfigurations"}, "whatever")
			}
			if action.GetVerb() == "get" && action.GetResource().Resource == "mutatingwebhookconfigurations" {
				return true, nil, errors.NewNotFound(schema.GroupResource{Group: "", Resource: "mutatingwebhookconfigurations"}, "whatever")
			}
			if action.GetVerb() != "get" || action.GetResource().Resource != "namespaces" {
				Expect(action).To(BeNil())
			}
			return true, nil, nil
		})
		apiServiceClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, errors.NewNotFound(schema.GroupResource{Group: "", Resource: "apiservices"}, "whatever"))
		secClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
		extClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
		promClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})

		syncCaches(stop)

		// add the privileged SCC without KubeVirt accounts
		scc := getSCC()
		sccSource.Add(&scc)

	})

	AfterEach(func() {
		close(stop)
		// Ensure that we add checks for expected events to every test
		Expect(recorder.Events).To(BeEmpty())
		ctrl.Finish()
	})

	injectMetadata := func(objectMeta *metav1.ObjectMeta, config *util.KubeVirtDeploymentConfig) {

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

	addKubeVirt := func(kv *v1.KubeVirt) {
		mockQueue.ExpectAdds(1)
		kvSource.Add(kv)
		mockQueue.Wait()
	}

	addServiceAccount := func(sa *k8sv1.ServiceAccount) {
		mockQueue.ExpectAdds(1)
		serviceAccountSource.Add(sa)
		mockQueue.Wait()
	}

	addClusterRole := func(cr *rbacv1.ClusterRole) {
		mockQueue.ExpectAdds(1)
		clusterRoleSource.Add(cr)
		mockQueue.Wait()
	}

	addClusterRoleBinding := func(crb *rbacv1.ClusterRoleBinding) {
		mockQueue.ExpectAdds(1)
		clusterRoleBindingSource.Add(crb)
		mockQueue.Wait()
	}

	addRole := func(role *rbacv1.Role) {
		mockQueue.ExpectAdds(1)
		roleSource.Add(role)
		mockQueue.Wait()
	}

	addRoleBinding := func(rb *rbacv1.RoleBinding) {
		mockQueue.ExpectAdds(1)
		roleBindingSource.Add(rb)
		mockQueue.Wait()
	}

	addCrd := func(crd *extv1.CustomResourceDefinition, kv *v1.KubeVirt) {
		mockQueue.ExpectAdds(1)
		if kv != nil {
			apply.SetGeneration(&kv.Status.Generations, crd)
		}

		crdSource.Add(crd)
		mockQueue.Wait()
	}

	addService := func(svc *k8sv1.Service) {
		mockQueue.ExpectAdds(1)
		serviceSource.Add(svc)
		mockQueue.Wait()
	}

	addDeployment := func(depl *appsv1.Deployment, kv *v1.KubeVirt) {
		mockQueue.ExpectAdds(1)
		if kv != nil {
			apply.SetGeneration(&kv.Status.Generations, depl)
		}

		deploymentSource.Add(depl)
		mockQueue.Wait()
	}

	addDaemonset := func(ds *appsv1.DaemonSet, kv *v1.KubeVirt) {
		mockQueue.ExpectAdds(1)
		if kv != nil {
			apply.SetGeneration(&kv.Status.Generations, ds)
		}

		daemonSetSource.Add(ds)
		mockQueue.Wait()
	}

	addValidatingWebhook := func(wh *admissionregistrationv1.ValidatingWebhookConfiguration, kv *v1.KubeVirt) {
		mockQueue.ExpectAdds(1)
		if kv != nil {
			apply.SetGeneration(&kv.Status.Generations, wh)
		}

		validatingWebhookSource.Add(wh)
		mockQueue.Wait()
	}

	addMutatingWebhook := func(wh *admissionregistrationv1.MutatingWebhookConfiguration, kv *v1.KubeVirt) {
		mockQueue.ExpectAdds(1)
		if kv != nil {
			apply.SetGeneration(&kv.Status.Generations, wh)
		}

		mutatingWebhookSource.Add(wh)
		mockQueue.Wait()
	}

	addAPIService := func(wh *apiregv1.APIService) {
		mockQueue.ExpectAdds(1)
		apiserviceSource.Add(wh)
		mockQueue.Wait()
	}

	addInstallStrategyJob := func(job *batchv1.Job) {
		mockQueue.ExpectAdds(1)
		installStrategyJobSource.Add(job)
		mockQueue.Wait()
	}

	addPod := func(pod *k8sv1.Pod) {
		mockQueue.ExpectAdds(1)
		infrastructurePodSource.Add(pod)
		mockQueue.Wait()
	}

	addPodDisruptionBudget := func(podDisruptionBudget *policyv1beta1.PodDisruptionBudget, kv *v1.KubeVirt) {
		mockQueue.ExpectAdds(1)
		if kv != nil {
			apply.SetGeneration(&kv.Status.Generations, podDisruptionBudget)
		}

		podDisruptionBudgetSource.Add(podDisruptionBudget)
		mockQueue.Wait()
	}

	addSecret := func(secret *k8sv1.Secret) {
		mockQueue.ExpectAdds(1)
		secretsSource.Add(secret)
		mockQueue.Wait()
	}

	addConfigMap := func(configMap *k8sv1.ConfigMap) {
		mockQueue.ExpectAdds(1)
		if _, ok := configMap.Labels[v1.InstallStrategyLabel]; ok {
			installStrategyConfigMapSource.Add(configMap)
		} else {
			configMapSource.Add(configMap)
		}
		mockQueue.Wait()
	}

	addSCC := func(scc *secv1.SecurityContextConstraints) {
		mockQueue.ExpectAdds(1)
		sccSource.Add(scc)
		mockQueue.Wait()
	}

	addServiceMonitor := func(serviceMonitor *promv1.ServiceMonitor) {
		mockQueue.ExpectAdds(1)
		serviceMonitorSource.Add(serviceMonitor)
		mockQueue.Wait()
	}

	addPrometheusRule := func(prometheusRule *promv1.PrometheusRule) {
		mockQueue.ExpectAdds(1)
		prometheusRuleSource.Add(prometheusRule)
		mockQueue.Wait()
	}

	addResource := func(obj runtime.Object, config *util.KubeVirtDeploymentConfig, kv *v1.KubeVirt) {
		switch resource := obj.(type) {
		case *k8sv1.ServiceAccount:
			injectMetadata(&obj.(*k8sv1.ServiceAccount).ObjectMeta, config)
			addServiceAccount(resource)
		case *rbacv1.ClusterRole:
			injectMetadata(&obj.(*rbacv1.ClusterRole).ObjectMeta, config)
			addClusterRole(resource)
		case *rbacv1.ClusterRoleBinding:
			injectMetadata(&obj.(*rbacv1.ClusterRoleBinding).ObjectMeta, config)
			addClusterRoleBinding(resource)
		case *rbacv1.Role:
			injectMetadata(&obj.(*rbacv1.Role).ObjectMeta, config)
			addRole(resource)
		case *rbacv1.RoleBinding:
			injectMetadata(&obj.(*rbacv1.RoleBinding).ObjectMeta, config)
			addRoleBinding(resource)
		case *extv1.CustomResourceDefinition:
			injectMetadata(&obj.(*extv1.CustomResourceDefinition).ObjectMeta, config)
			addCrd(resource, kv)
		case *k8sv1.Service:
			injectMetadata(&obj.(*k8sv1.Service).ObjectMeta, config)
			addService(resource)
		case *appsv1.Deployment:
			injectMetadata(&obj.(*appsv1.Deployment).ObjectMeta, config)
			addDeployment(resource, kv)
		case *appsv1.DaemonSet:
			injectMetadata(&obj.(*appsv1.DaemonSet).ObjectMeta, config)
			addDaemonset(resource, kv)
		case *admissionregistrationv1.ValidatingWebhookConfiguration:
			injectMetadata(&obj.(*admissionregistrationv1.ValidatingWebhookConfiguration).ObjectMeta, config)
			addValidatingWebhook(resource, kv)
		case *admissionregistrationv1.MutatingWebhookConfiguration:
			injectMetadata(&obj.(*admissionregistrationv1.MutatingWebhookConfiguration).ObjectMeta, config)
			addMutatingWebhook(resource, kv)
		case *apiregv1.APIService:
			injectMetadata(&obj.(*apiregv1.APIService).ObjectMeta, config)
			addAPIService(resource)
		case *batchv1.Job:
			injectMetadata(&obj.(*batchv1.Job).ObjectMeta, config)
			addInstallStrategyJob(resource)
		case *k8sv1.ConfigMap:
			injectMetadata(&obj.(*k8sv1.ConfigMap).ObjectMeta, config)
			addConfigMap(resource)
		case *k8sv1.Pod:
			injectMetadata(&obj.(*k8sv1.Pod).ObjectMeta, config)
			addPod(resource)
		case *policyv1beta1.PodDisruptionBudget:
			injectMetadata(&obj.(*policyv1beta1.PodDisruptionBudget).ObjectMeta, config)
			addPodDisruptionBudget(resource, kv)
		case *k8sv1.Secret:
			injectMetadata(&obj.(*k8sv1.Secret).ObjectMeta, config)
			addSecret(resource)
		case *secv1.SecurityContextConstraints:
			injectMetadata(&obj.(*secv1.SecurityContextConstraints).ObjectMeta, config)
			addSCC(resource)
		case *promv1.ServiceMonitor:
			injectMetadata(&obj.(*promv1.ServiceMonitor).ObjectMeta, config)
			addServiceMonitor(resource)
		case *promv1.PrometheusRule:
			injectMetadata(&obj.(*promv1.PrometheusRule).ObjectMeta, config)
			addPrometheusRule(resource)
		default:
			Fail("unknown resource type")
		}
		split := strings.Split(fmt.Sprintf("%T", obj), ".")
		resourceKey := strings.ToLower(split[len(split)-1]) + "s"
		if _, ok := resourceChanges[resourceKey]; !ok {
			resourceChanges[resourceKey] = make(map[string]int)
		}
		resourceChanges[resourceKey][Added]++
	}

	addInstallStrategy := func(config *util.KubeVirtDeploymentConfig) {
		// install strategy config
		resource, _ := install.NewInstallStrategyConfigMap(config, true, NAMESPACE)

		resource.Name = fmt.Sprintf("%s-%s", resource.Name, rand.String(10))

		injectMetadata(&resource.ObjectMeta, config)
		addConfigMap(resource)
	}

	addPodDisruptionBudgets := func(config *util.KubeVirtDeploymentConfig, apiDeployment *appsv1.Deployment, controller *appsv1.Deployment, kv *v1.KubeVirt) {
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
		addPodDisruptionBudget(apiPodDisruptionBudget, kv)
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
		addPodDisruptionBudget(controllerPodDisruptionBudget, kv)
	}

	addPodsWithIndividualConfigs := func(config *util.KubeVirtDeploymentConfig,
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
		addPod(pod)

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
		addPod(pod)

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
		addPod(pod)

		if shouldAddPodDisruptionBudgets {
			addPodDisruptionBudgets(config, apiDeployment, controller, kv)
		}
	}

	addPodsWithOptionalPodDisruptionBudgets := func(config *util.KubeVirtDeploymentConfig, shouldAddPodDisruptionBudgets bool, kv *v1.KubeVirt) {
		addPodsWithIndividualConfigs(config, config, config, shouldAddPodDisruptionBudgets, kv)
	}

	addPodsAndPodDisruptionBudgets := func(config *util.KubeVirtDeploymentConfig, kv *v1.KubeVirt) {
		addPodsWithOptionalPodDisruptionBudgets(config, true, kv)
	}

	generateRandomResources := func() int {
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
			addResource(obj, config, nil)
		}
		return len(all)
	}

	addDummyValidationWebhook := func() {
		version := fmt.Sprintf("rand-%s", rand.String(10))
		registry := fmt.Sprintf("rand-%s", rand.String(10))
		config := getConfig(registry, version)

		validationWebhook := &admissionregistrationv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "virt-operator-tmp-webhook",
			},
		}

		injectMetadata(&validationWebhook.ObjectMeta, config)
		addValidatingWebhook(validationWebhook, nil)
	}

	addAll := func(config *util.KubeVirtDeploymentConfig, kv *v1.KubeVirt) {
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

		all = append(all, rbac.GetAllServiceMonitor(NAMESPACE, config.GetMonitorNamespace(), config.GetMonitorServiceAccount())...)
		all = append(all, components.NewServiceMonitorCR(NAMESPACE, config.GetMonitorNamespace(), true))

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

			addResource(obj, config, kv)
		}
	}

	makePodDisruptionBudgetsReady := func() {
		for _, pdbname := range []string{"/virt-api-pdb", "/virt-controller-pdb"} {
			exists := false
			// we need to wait until the pdb exists
			for !exists {
				_, exists, _ = stores.PodDisruptionBudgetCache.GetByKey(NAMESPACE + pdbname)
				if !exists {
					time.Sleep(time.Second)
				}
			}
		}
	}

	makeApiAndControllerReady := func() {
		makeDeploymentReady := func(item interface{}) {
			depl, _ := item.(*appsv1.Deployment)
			deplNew := depl.DeepCopy()
			var replicas int32 = 1
			if depl.Spec.Replicas != nil {
				replicas = *depl.Spec.Replicas
			}
			deplNew.Status.Replicas = replicas
			deplNew.Status.ReadyReplicas = replicas
			deploymentSource.Modify(deplNew)
		}

		for _, name := range []string{"/virt-api", "/virt-controller"} {
			exists := false
			var obj interface{}
			// we need to wait until the deployment exists
			for !exists {
				obj, exists, _ = controller.stores.DeploymentCache.GetByKey(NAMESPACE + name)
				if exists {
					makeDeploymentReady(obj)
				}
				time.Sleep(time.Second)
			}
		}

		makePodDisruptionBudgetsReady()
	}

	makeHandlerReady := func() {
		exists := false
		var obj interface{}
		// we need to wait until the daemonset exists
		for !exists {
			obj, exists, _ = controller.stores.DaemonSetCache.GetByKey(NAMESPACE + "/virt-handler")
			if exists {
				handler, _ := obj.(*appsv1.DaemonSet)
				handlerNew := handler.DeepCopy()
				handlerNew.Status.DesiredNumberScheduled = 1
				handlerNew.Status.NumberReady = 1
				daemonSetSource.Modify(handlerNew)
			}
			time.Sleep(time.Second)
		}
	}

	deleteServiceAccount := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.ServiceAccount.GetStore().GetByKey(key); exists {
			serviceAccountSource.Delete(obj.(runtime.Object))
		}
		mockQueue.Wait()
	}

	deleteClusterRole := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.ClusterRole.GetStore().GetByKey(key); exists {
			clusterRoleSource.Delete(obj.(runtime.Object))
		}
		mockQueue.Wait()
	}

	deleteClusterRoleBinding := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.ClusterRoleBinding.GetStore().GetByKey(key); exists {
			clusterRoleBindingSource.Delete(obj.(runtime.Object))
		}
		mockQueue.Wait()
	}

	deleteRole := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.Role.GetStore().GetByKey(key); exists {
			roleSource.Delete(obj.(runtime.Object))
		}
		mockQueue.Wait()
	}

	deleteRoleBinding := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.RoleBinding.GetStore().GetByKey(key); exists {
			roleBindingSource.Delete(obj.(runtime.Object))
		}
		mockQueue.Wait()
	}

	deleteCrd := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.Crd.GetStore().GetByKey(key); exists {
			crdSource.Delete(obj.(runtime.Object))
		}
		mockQueue.Wait()
	}

	deleteService := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.Service.GetStore().GetByKey(key); exists {
			serviceSource.Delete(obj.(runtime.Object))
		}
		mockQueue.Wait()
	}

	deleteDeployment := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.Deployment.GetStore().GetByKey(key); exists {
			deploymentSource.Delete(obj.(runtime.Object))
		}
		mockQueue.Wait()
	}

	deleteDaemonset := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.DaemonSet.GetStore().GetByKey(key); exists {
			daemonSetSource.Delete(obj.(runtime.Object))
		}
		mockQueue.Wait()
	}

	deleteValidationWebhook := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.ValidationWebhook.GetStore().GetByKey(key); exists {
			validatingWebhookSource.Delete(obj.(runtime.Object))
		}
		mockQueue.Wait()
	}

	deleteMutatingWebhook := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.MutatingWebhook.GetStore().GetByKey(key); exists {
			mutatingWebhookSource.Delete(obj.(runtime.Object))
		}
		mockQueue.Wait()
	}

	deleteAPIService := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.APIService.GetStore().GetByKey(key); exists {
			apiserviceSource.Delete(obj.(runtime.Object))
		}
		mockQueue.Wait()
	}

	deleteInstallStrategyJob := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.InstallStrategyJob.GetStore().GetByKey(key); exists {
			installStrategyJobSource.Delete(obj.(runtime.Object))
		}
		mockQueue.Wait()
	}

	deletePodDisruptionBudget := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.PodDisruptionBudget.GetStore().GetByKey(key); exists {
			podDisruptionBudgetSource.Delete(obj.(runtime.Object))
		}
		mockQueue.Wait()
	}

	deleteSecret := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.Secrets.GetStore().GetByKey(key); exists {
			secretsSource.Delete(obj.(runtime.Object))
		}
		mockQueue.Wait()
	}

	deleteConfigMap := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.ConfigMap.GetStore().GetByKey(key); exists {
			configMap := obj.(*k8sv1.ConfigMap)
			configMapSource.Delete(configMap)
		} else if obj, exists, _ := informers.InstallStrategyConfigMap.GetStore().GetByKey(key); exists {
			configMap := obj.(*k8sv1.ConfigMap)
			installStrategyConfigMapSource.Delete(configMap)
		}
		mockQueue.Wait()
	}

	deleteSCC := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.SCC.GetStore().GetByKey(key); exists {
			sccSource.Delete(obj.(runtime.Object))
		}
		mockQueue.Wait()
	}

	deleteServiceMonitor := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.ServiceMonitor.GetStore().GetByKey(key); exists {
			serviceMonitorSource.Delete(obj.(runtime.Object))
		}
		mockQueue.Wait()
	}

	deletePrometheusRule := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.PrometheusRule.GetStore().GetByKey(key); exists {
			prometheusRuleSource.Delete(obj.(runtime.Object))
		}
		mockQueue.Wait()
	}

	deleteResource := func(resource string, key string) {
		switch resource {
		case "serviceaccounts":
			deleteServiceAccount(key)
		case "clusterroles":
			deleteClusterRole(key)
		case "clusterrolebindings":
			deleteClusterRoleBinding(key)
		case "roles":
			deleteRole(key)
		case "rolebindings":
			deleteRoleBinding(key)
		case "customresourcedefinitions":
			deleteCrd(key)
		case "services":
			deleteService(key)
		case "deployments":
			deleteDeployment(key)
		case "daemonsets":
			deleteDaemonset(key)
		case "validatingwebhookconfigurations":
			deleteValidationWebhook(key)
		case "mutatingwebhookconfigurations":
			deleteMutatingWebhook(key)
		case "apiservices":
			deleteAPIService(key)
		case "jobs":
			deleteInstallStrategyJob(key)
		case "configmaps":
			deleteConfigMap(key)
		case "poddisruptionbudgets":
			deletePodDisruptionBudget(key)
		case "secrets":
			deleteSecret(key)
		case "securitycontextconstraints":
			deleteSCC(key)
		case "servicemonitors":
			deleteServiceMonitor(key)
		case "prometheusrules":
			deletePrometheusRule(key)
		default:
			Fail(fmt.Sprintf("unknown resource type %+v", resource))
		}
		if _, ok := resourceChanges[resource]; !ok {
			resourceChanges[resource] = make(map[string]int)
		}
		resourceChanges[resource][Deleted]++
	}

	genericUpdateFunc := func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		update, ok := action.(testing.UpdateAction)
		Expect(ok).To(BeTrue(), "genericUpdateFunction testing ok")
		totalUpdates++
		resource := action.GetResource().Resource
		if _, ok := resourceChanges[resource]; !ok {
			resourceChanges[resource] = make(map[string]int)
		}
		resourceChanges[resource][Updated]++

		return true, update.GetObject(), nil
	}

	genericPatchFunc := func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		_, ok := action.(testing.PatchAction)
		Expect(ok).To(BeTrue())
		totalPatches++
		resource := action.GetResource().Resource
		if _, ok := resourceChanges[resource]; !ok {
			resourceChanges[resource] = make(map[string]int)
		}
		resourceChanges[resource][Patched]++

		return true, nil, nil
	}

	webhookValidationPatchFunc := func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		genericPatchFunc(action)

		return true, &admissionregistrationv1.ValidatingWebhookConfiguration{}, nil
	}

	webhookMutatingPatchFunc := func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		genericPatchFunc(action)

		return true, &admissionregistrationv1.MutatingWebhookConfiguration{}, nil
	}

	deploymentPatchFunc := func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		genericPatchFunc(action)

		return true, &appsv1.Deployment{}, nil
	}

	daemonsetPatchFunc := func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		genericPatchFunc(action)

		return true, &appsv1.DaemonSet{}, nil
	}

	podDisruptionBudgetPatchFunc := func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		genericPatchFunc(action)

		return true, &policyv1beta1.PodDisruptionBudget{}, nil
	}

	crdPatchFunc := func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		genericPatchFunc(action)

		return true, &extv1.CustomResourceDefinition{}, nil
	}

	genericCreateFunc := func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())
		totalAdds++
		if addToCache {
			addResource(create.GetObject(), nil, nil)
		}
		return true, create.GetObject(), nil
	}

	genericDeleteFunc := func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		deleted, ok := action.(testing.DeleteAction)
		Expect(ok).To(BeTrue())
		totalDeletions++
		var key string
		if len(deleted.GetNamespace()) > 0 {
			key = deleted.GetNamespace() + "/"
		}
		key += deleted.GetName()
		if deleteFromCache {
			deleteResource(deleted.GetResource().Resource, key)
		}
		return true, nil, nil
	}

	shouldExpectInstallStrategyDeletion := func() {
		kubeClient.Fake.PrependReactor("delete", "configmaps", func(action testing.Action) (handled bool, obj runtime.Object, err error) {

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
			deleteResource(deleted.GetResource().Resource, key)
			return true, nil, nil
		})
	}

	shouldExpectDeletions := func() {
		kubeClient.Fake.PrependReactor("delete", "serviceaccounts", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "clusterroles", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "clusterrolebindings", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "roles", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "rolebindings", genericDeleteFunc)
		extClient.Fake.PrependReactor("delete", "customresourcedefinitions", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "services", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "deployments", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "daemonsets", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "validatingwebhookconfigurations", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "mutatingwebhookconfigurations", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "secrets", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "configmaps", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "poddisruptionbudgets", genericDeleteFunc)
		secClient.Fake.PrependReactor("delete", "securitycontextconstraints", genericDeleteFunc)
		promClient.Fake.PrependReactor("delete", "servicemonitors", genericDeleteFunc)
		promClient.Fake.PrependReactor("delete", "prometheusrules", genericDeleteFunc)
		apiServiceClient.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Do(func(ctx context.Context, name string, options interface{}) {
			genericDeleteFunc(&testing.DeleteActionImpl{ActionImpl: testing.ActionImpl{Resource: schema.GroupVersionResource{Resource: "apiservices"}}, Name: name})
		})
	}

	shouldExpectJobDeletion := func() {
		kubeClient.Fake.PrependReactor("delete", "jobs", genericDeleteFunc)
	}

	shouldExpectJobCreation := func() {
		kubeClient.Fake.PrependReactor("create", "jobs", genericCreateFunc)
	}

	shouldExpectPatchesAndUpdates := func() {
		extClient.Fake.PrependReactor("patch", "customresourcedefinitions", crdPatchFunc)
		kubeClient.Fake.PrependReactor("patch", "serviceaccounts", genericPatchFunc)
		kubeClient.Fake.PrependReactor("update", "clusterroles", genericUpdateFunc)
		kubeClient.Fake.PrependReactor("update", "clusterrolebindings", genericUpdateFunc)
		kubeClient.Fake.PrependReactor("update", "roles", genericUpdateFunc)
		kubeClient.Fake.PrependReactor("update", "rolebindings", genericUpdateFunc)
		kubeClient.Fake.PrependReactor("patch", "validatingwebhookconfigurations", webhookValidationPatchFunc)
		kubeClient.Fake.PrependReactor("patch", "mutatingwebhookconfigurations", webhookMutatingPatchFunc)
		kubeClient.Fake.PrependReactor("patch", "secrets", genericPatchFunc)
		kubeClient.Fake.PrependReactor("patch", "configmaps", genericPatchFunc)

		kubeClient.Fake.PrependReactor("patch", "services", genericPatchFunc)
		kubeClient.Fake.PrependReactor("patch", "daemonsets", daemonsetPatchFunc)
		kubeClient.Fake.PrependReactor("patch", "deployments", deploymentPatchFunc)
		kubeClient.Fake.PrependReactor("patch", "poddisruptionbudgets", podDisruptionBudgetPatchFunc)
		secClient.Fake.PrependReactor("update", "securitycontextconstraints", genericUpdateFunc)
		promClient.Fake.PrependReactor("patch", "servicemonitors", genericPatchFunc)
		promClient.Fake.PrependReactor("patch", "prometheusrules", genericPatchFunc)
		apiServiceClient.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Do(func(args ...interface{}) {
			genericPatchFunc(&testing.PatchActionImpl{ActionImpl: testing.ActionImpl{Resource: schema.GroupVersionResource{Resource: "apiservices"}}})
		})
	}

	shouldExpectRbacBackupCreations := func() {
		kubeClient.Fake.PrependReactor("create", "clusterroles", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "clusterrolebindings", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "roles", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "rolebindings", genericCreateFunc)
	}

	shouldExpectCreations := func() {
		kubeClient.Fake.PrependReactor("create", "serviceaccounts", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "clusterroles", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "clusterrolebindings", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "roles", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "rolebindings", genericCreateFunc)
		extClient.Fake.PrependReactor("create", "customresourcedefinitions", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "services", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "deployments", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "daemonsets", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "validatingwebhookconfigurations", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "mutatingwebhookconfigurations", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "secrets", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "configmaps", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "poddisruptionbudgets", genericCreateFunc)
		secClient.Fake.PrependReactor("create", "securitycontextconstraints", genericCreateFunc)
		promClient.Fake.PrependReactor("create", "servicemonitors", genericCreateFunc)
		promClient.Fake.PrependReactor("create", "prometheusrules", genericCreateFunc)
		apiServiceClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Do(func(ctx context.Context, obj runtime.Object, opts metav1.CreateOptions) {
			genericCreateFunc(&testing.CreateActionImpl{Object: obj})
		})
	}

	shouldExpectKubeVirtUpdate := func(times int) {
		update := kvInterface.EXPECT().Update(gomock.Any())
		update.Do(func(kv *v1.KubeVirt) {
			kvInformer.GetStore().Update(kv)
			update.Return(kv, nil)
		}).Times(times)
	}

	shouldExpectKubeVirtUpdateStatus := func(times int) {
		update := kvInterface.EXPECT().UpdateStatus(gomock.Any())
		update.Do(func(kv *v1.KubeVirt) {
			kvInformer.GetStore().Update(kv)
			update.Return(kv, nil)
		}).Times(times)
	}

	shouldExpectKubeVirtUpdateStatusVersion := func(times int, config *util.KubeVirtDeploymentConfig) {
		update := kvInterface.EXPECT().UpdateStatus(gomock.Any())
		update.Do(func(kv *v1.KubeVirt) {

			Expect(kv.Status.TargetKubeVirtVersion).To(Equal(config.GetKubeVirtVersion()))
			Expect(kv.Status.ObservedKubeVirtVersion).To(Equal(config.GetKubeVirtVersion()))
			kvInformer.GetStore().Update(kv)
			update.Return(kv, nil)
		}).Times(times)
	}

	shouldExpectKubeVirtUpdateStatusFailureCondition := func(reason string) {
		update := kvInterface.EXPECT().UpdateStatus(gomock.Any())
		update.Do(func(kv *v1.KubeVirt) {
			Expect(len(kv.Status.Conditions)).To(Equal(1))
			Expect(kv.Status.Conditions[0].Reason).To(Equal(reason))
			kvInformer.GetStore().Update(kv)
			update.Return(kv, nil)
		}).Times(1)
	}

	getLatestKubeVirt := func(kv *v1.KubeVirt) *v1.KubeVirt {
		if obj, exists, _ := kvInformer.GetStore().GetByKey(kv.GetNamespace() + "/" + kv.GetName()); exists {
			if kvLatest, ok := obj.(*v1.KubeVirt); ok {
				return kvLatest
			}
		}
		return nil
	}

	shouldExpectHCOConditions := func(kv *v1.KubeVirt, available k8sv1.ConditionStatus, progressing k8sv1.ConditionStatus, degraded k8sv1.ConditionStatus) {
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

	fakeNamespaceModificationEvent := func() {
		// Add modification event for namespace w/o the labels we need
		mockQueue.ExpectAdds(1)
		namespaceSource.Modify(&k8sv1.Namespace{
			TypeMeta: metav1.TypeMeta{
				Kind: "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: NAMESPACE,
			},
		})
		mockQueue.Wait()
	}

	shouldExpectNamespacePatch := func() {
		kubeClient.Fake.PrependReactor("patch", "namespaces", genericPatchFunc)
	}

	Context("On valid KubeVirt object", func() {

		It("Should not patch kubevirt namespace when labels are already defined", func(done Done) {
			defer close(done)

			// Add fake namespace with labels predefined
			err := informers.Namespace.GetStore().Add(&k8sv1.Namespace{
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
			addKubeVirt(kv)
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			shouldExpectKubeVirtUpdateStatus(1)
			shouldExpectCreations()
			addInstallStrategy(defaultConfig)
			addAll(defaultConfig, kv)
			addPodsAndPodDisruptionBudgets(defaultConfig, kv)
			makeHandlerReady()
			makeApiAndControllerReady()
			makeHandlerReady()
			shouldExpectPatchesAndUpdates()

			// Now when the controller runs, if the namespace will be patched, the test will fail
			// because the patch is not expected here.
			controller.Execute()
		}, 15)

		It("should delete install strategy configmap once kubevirt install is deleted", func(done Done) {
			defer close(done)

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

			shouldExpectInstallStrategyDeletion()

			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)
			shouldExpectKubeVirtUpdate(1)
			controller.Execute()
			kv = getLatestKubeVirt(kv)
			Expect(len(kv.ObjectMeta.Finalizers)).To(Equal(0))
		}, 15)

		It("should observe custom image tag in status during deploy", func(done Done) {
			defer close(done)
			defer GinkgoRecover()

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
			addKubeVirt(kv)
			customConfig := getConfig(defaultConfig.GetImageRegistry(), "custom.tag")

			fakeNamespaceModificationEvent()
			shouldExpectNamespacePatch()
			shouldExpectPatchesAndUpdates()
			addAll(customConfig, kv)
			// install strategy config
			addInstallStrategy(customConfig)
			addPodsAndPodDisruptionBudgets(customConfig, kv)

			makeApiAndControllerReady()
			makeHandlerReady()

			shouldExpectKubeVirtUpdateStatusVersion(1, customConfig)
			controller.Execute()
			kv = getLatestKubeVirt(kv)
			shouldExpectHCOConditions(kv, k8sv1.ConditionTrue, k8sv1.ConditionFalse, k8sv1.ConditionFalse)

		}, 15)

		It("delete temporary validation webhook once virt-api is deployed", func(done Done) {
			defer close(done)

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
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)
			deleteFromCache = false

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addDummyValidationWebhook()
			addInstallStrategy(defaultConfig)
			addAll(defaultConfig, kv)
			addPodsAndPodDisruptionBudgets(defaultConfig, kv)
			makeApiAndControllerReady()
			makeHandlerReady()

			shouldExpectDeletions()
			fakeNamespaceModificationEvent()
			shouldExpectNamespacePatch()
			shouldExpectPatchesAndUpdates()
			shouldExpectKubeVirtUpdateStatus(1)

			controller.Execute()
			Expect(totalDeletions).To(Equal(1))

		}, 15)

		It("should do nothing if KubeVirt object is deployed", func(done Done) {
			defer close(done)

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
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)
			addAll(defaultConfig, kv)
			addPodsAndPodDisruptionBudgets(defaultConfig, kv)
			makeApiAndControllerReady()
			makeHandlerReady()

			fakeNamespaceModificationEvent()
			shouldExpectNamespacePatch()
			shouldExpectPatchesAndUpdates()
			shouldExpectKubeVirtUpdateStatus(1)

			controller.Execute()

		}, 15)

		It("should update KubeVirt object if generation IDs do not match", func(done Done) {
			defer close(done)

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
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)

			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)
			addAll(defaultConfig, kv)
			addPodsAndPodDisruptionBudgets(defaultConfig, kv)
			makeApiAndControllerReady()
			makeHandlerReady()

			fakeNamespaceModificationEvent()
			shouldExpectNamespacePatch()
			shouldExpectPatchesAndUpdates()
			shouldExpectKubeVirtUpdateStatus(1)

			// invalidate all lastGeneration versions
			numGenerations := len(kv.Status.Generations)
			for i := range kv.Status.Generations {
				kv.Status.Generations[i].LastGeneration = -1
			}

			controller.Execute()

			// add one for the namespace
			Expect(totalPatches).To(Equal(numGenerations + 1))

			// all these resources should be tracked by there generation so everyone that has been added should now be patched
			// since they where the `lastGeneration` was set to -1 on the KubeVirt CR
			Expect(resourceChanges["mutatingwebhookconfigurations"][Patched]).To(Equal(resourceChanges["mutatingwebhookconfigurations"][Added]))
			Expect(resourceChanges["validatingwebhookconfigurations"][Patched]).To(Equal(resourceChanges["validatingwebhookconfigurations"][Added]))
			Expect(resourceChanges["deployements"][Patched]).To(Equal(resourceChanges["deployements"][Added]))
			Expect(resourceChanges["daemonsets"][Patched]).To(Equal(resourceChanges["daemonsets"][Added]))
			Expect(resourceChanges["poddisruptionbudgets"][Patched]).To(Equal(resourceChanges["poddisruptionbudgets"][Added]))
		}, 150)

		It("should delete operator managed resources not in the deployed installstrategy", func() {
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
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsDeploying(kv)
			util.UpdateConditionsCreated(kv)

			deleteFromCache = false

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)
			addAll(defaultConfig, kv)
			numResources := generateRandomResources()
			addPodsAndPodDisruptionBudgets(defaultConfig, kv)

			makeApiAndControllerReady()
			makeHandlerReady()

			shouldExpectDeletions()
			fakeNamespaceModificationEvent()
			shouldExpectNamespacePatch()
			shouldExpectPatchesAndUpdates()
			shouldExpectKubeVirtUpdateStatus(1)

			controller.Execute()
			Expect(totalDeletions).To(Equal(numResources))
		}, 15)

		It("should fail if KubeVirt object already exists", func() {
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
			addKubeVirt(kv1)
			kubecontroller.SetLatestApiVersionAnnotation(kv2)
			addKubeVirt(kv2)

			shouldExpectKubeVirtUpdateStatusFailureCondition(util.ConditionReasonDeploymentFailedExisting)

			controller.execute(fmt.Sprintf("%s/%s", kv2.Namespace, kv2.Name))

		}, 15)

		It("should generate install strategy creation job for update version", func(done Done) {
			defer close(done)

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
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)

			shouldExpectKubeVirtUpdateStatus(1)
			shouldExpectJobCreation()
			controller.Execute()

		}, 15)

		It("should create an install strategy creation job with passthrough env vars, if provided in config", func(done Done) {
			defer close(done)
			config := getConfig("registry", "v1.1.1")
			envKey := rand.String(10)
			envVal := rand.String(10)
			config.PassthroughEnvVars = map[string]string{envKey: envVal}
			job, err := controller.generateInstallStrategyJob(config)

			Expect(err).ToNot(HaveOccurred())
			Expect(job.Spec.Template.Spec.Containers[0].Env).To(ContainElement(k8sv1.EnvVar{Name: envKey, Value: envVal}))
		}, 15)

		It("should create an api server deployment with passthrough env vars, if provided in config", func(done Done) {
			defer close(done)
			config := getConfig("registry", "v1.1.1")
			envKey := rand.String(10)
			envVal := rand.String(10)
			config.PassthroughEnvVars = map[string]string{envKey: envVal}

			apiDeployment, err := components.NewApiServerDeployment(NAMESPACE, config.GetImageRegistry(), config.GetImagePrefix(), config.GetApiVersion(), "", "", config.GetImagePullPolicy(), config.GetVerbosity(), config.GetExtraEnv())

			Expect(err).ToNot(HaveOccurred())
			Expect(apiDeployment.Spec.Template.Spec.Containers[0].Env).To(ContainElement(k8sv1.EnvVar{Name: envKey, Value: envVal}))
		}, 15)

		It("should create a controller deployment with passthrough env vars, if provided in config", func(done Done) {
			defer close(done)
			config := getConfig("registry", "v1.1.1")
			envKey := rand.String(10)
			envVal := rand.String(10)
			config.PassthroughEnvVars = map[string]string{envKey: envVal}

			controllerDeployment, err := components.NewControllerDeployment(NAMESPACE, config.GetImageRegistry(), config.GetImagePrefix(), config.GetControllerVersion(), config.GetLauncherVersion(), "", "", config.GetImagePullPolicy(), config.GetVerbosity(), config.GetExtraEnv())

			Expect(err).ToNot(HaveOccurred())
			Expect(controllerDeployment.Spec.Template.Spec.Containers[0].Env).To(ContainElement(k8sv1.EnvVar{Name: envKey, Value: envVal}))
		}, 15)

		It("should create a handler daemonset with passthrough env vars, if provided in config", func(done Done) {
			defer close(done)
			config := getConfig("registry", "v1.1.1")
			envKey := rand.String(10)
			envVal := rand.String(10)
			config.PassthroughEnvVars = map[string]string{envKey: envVal}

			handlerDaemonset, err := components.NewHandlerDaemonSet(NAMESPACE, config.GetImageRegistry(), config.GetImagePrefix(), config.GetHandlerVersion(), "", "", config.GetLauncherVersion(), config.GetImagePullPolicy(), config.GetVerbosity(), config.GetExtraEnv())

			Expect(err).ToNot(HaveOccurred())
			Expect(handlerDaemonset.Spec.Template.Spec.Containers[0].Env).To(ContainElement(k8sv1.EnvVar{Name: envKey, Value: envVal}))
		}, 15)

		It("should generate install strategy creation job if no install strategy exists", func(done Done) {
			defer close(done)

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
			addKubeVirt(kv)
			shouldExpectKubeVirtUpdateStatus(1)
			shouldExpectJobCreation()
			controller.Execute()

		}, 15)

		It("should label install strategy creation job", func(done Done) {
			defer close(done)

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Status: v1.KubeVirtStatus{},
			}

			job, err := controller.generateInstallStrategyJob(util.GetTargetConfigFromKV(kv))
			Expect(err).ToNot(HaveOccurred())

			Expect(job.Spec.Template.ObjectMeta.Labels).Should(HaveKeyWithValue(v1.AppLabel, virtOperatorJobAppLabel))
		}, 15)

		It("should delete install strategy creation job if job has failed", func(done Done) {
			defer close(done)

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Status: v1.KubeVirtStatus{},
			}

			job, err := controller.generateInstallStrategyJob(util.GetTargetConfigFromKV(kv))
			Expect(err).ToNot(HaveOccurred())

			// will only create a new job after 10 seconds has passed.
			// this is just a simple mechanism to prevent spin loops
			// in the event that jobs are fast failing for some unknown reason.
			completionTime := time.Now().Add(time.Duration(-10) * time.Second)
			job.Status.CompletionTime = &metav1.Time{Time: completionTime}

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategyJob(job)

			shouldExpectJobDeletion()
			shouldExpectKubeVirtUpdateStatus(1)

			controller.Execute()

		}, 15)

		It("should not delete completed install strategy creation job if job has failed less that 10 seconds ago", func(done Done) {
			defer GinkgoRecover()
			defer close(done)

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Status: v1.KubeVirtStatus{},
			}

			job, err := controller.generateInstallStrategyJob(util.GetTargetConfigFromKV(kv))
			Expect(err).ToNot(HaveOccurred())

			job.Status.CompletionTime = now()

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategyJob(job)

			shouldExpectKubeVirtUpdateStatus(1)

			controller.Execute()

		}, 15)

		It("should add resources on create", func() {
			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-install",
					Namespace: NAMESPACE,
				},
			}
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)

			job, err := controller.generateInstallStrategyJob(util.GetTargetConfigFromKV(kv))
			Expect(err).ToNot(HaveOccurred())

			job.Status.CompletionTime = now()
			addInstallStrategyJob(job)

			// ensure completed jobs are garbage collected once install strategy
			// is loaded
			deleteFromCache = false
			shouldExpectJobDeletion()
			shouldExpectKubeVirtUpdate(1)
			shouldExpectKubeVirtUpdateStatus(1)
			shouldExpectCreations()

			controller.Execute()

			kv = getLatestKubeVirt(kv)
			Expect(kv.Status.Phase).To(Equal(v1.KubeVirtPhaseDeploying))
			Expect(len(kv.Status.Conditions)).To(Equal(3))
			Expect(len(kv.ObjectMeta.Finalizers)).To(Equal(1))
			shouldExpectHCOConditions(kv, k8sv1.ConditionFalse, k8sv1.ConditionTrue, k8sv1.ConditionFalse)

			// 3 in total are yet missing at this point
			// because waiting on controller, controller's PDB and virt-handler daemonset until API server deploys successfully
			expectedUncreatedResources := 3

			// 1 because a temporary validation webhook is created to block new CRDs until api server is deployed
			expectedTemporaryResources := 1

			Expect(totalAdds).To(Equal(resourceCount - expectedUncreatedResources + expectedTemporaryResources))

			Expect(len(controller.stores.ServiceAccountCache.List())).To(Equal(3))
			Expect(len(controller.stores.ClusterRoleCache.List())).To(Equal(7))
			Expect(len(controller.stores.ClusterRoleBindingCache.List())).To(Equal(5))
			Expect(len(controller.stores.RoleCache.List())).To(Equal(3))
			Expect(len(controller.stores.RoleBindingCache.List())).To(Equal(3))
			Expect(len(controller.stores.CrdCache.List())).To(Equal(8))
			Expect(len(controller.stores.ServiceCache.List())).To(Equal(3))
			Expect(len(controller.stores.DeploymentCache.List())).To(Equal(1))
			Expect(len(controller.stores.DaemonSetCache.List())).To(Equal(0))
			Expect(len(controller.stores.ValidationWebhookCache.List())).To(Equal(3))
			Expect(len(controller.stores.PodDisruptionBudgetCache.List())).To(Equal(1))
			Expect(len(controller.stores.SCCCache.List())).To(Equal(3))
			Expect(len(controller.stores.ServiceMonitorCache.List())).To(Equal(1))
			Expect(len(controller.stores.PrometheusRuleCache.List())).To(Equal(1))

			Expect(resourceChanges["poddisruptionbudgets"][Added]).To(Equal(1))

		}, 15)

		Context("when the monitor namespace does not exist", func() {
			It("should not create ServiceMonitor resources", func() {
				kv := &v1.KubeVirt{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-install",
						Namespace:  NAMESPACE,
						Finalizers: []string{util.KubeVirtFinalizer},
					},
				}
				kubecontroller.SetLatestApiVersionAnnotation(kv)
				addKubeVirt(kv)

				// install strategy config
				resource, _ := install.NewInstallStrategyConfigMap(defaultConfig, false, NAMESPACE)
				resource.Name = fmt.Sprintf("%s-%s", resource.Name, rand.String(10))
				addResource(resource, defaultConfig, nil)

				job, err := controller.generateInstallStrategyJob(util.GetTargetConfigFromKV(kv))
				Expect(err).ToNot(HaveOccurred())

				job.Status.CompletionTime = now()
				addInstallStrategyJob(job)

				// ensure completed jobs are garbage collected once install strategy
				// is loaded
				deleteFromCache = false
				shouldExpectJobDeletion()
				shouldExpectKubeVirtUpdateStatus(1)
				shouldExpectCreations()

				controller.Execute()

				Expect(len(controller.stores.RoleCache.List())).To(Equal(2))
				Expect(len(controller.stores.RoleBindingCache.List())).To(Equal(2))
				Expect(len(controller.stores.ServiceMonitorCache.List())).To(Equal(0))
			}, 15)
		})

		It("should pause rollback until api server is rolled over.", func(done Done) {
			defer close(done)
			defer GinkgoRecover()

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
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)
			addInstallStrategy(rollbackConfig)

			addAll(defaultConfig, kv)
			addPodsAndPodDisruptionBudgets(defaultConfig, kv)

			makeApiAndControllerReady()
			makeHandlerReady()

			addToCache = false
			shouldExpectRbacBackupCreations()
			shouldExpectPatchesAndUpdates()
			shouldExpectKubeVirtUpdateStatus(1)

			controller.Execute()

			kv = getLatestKubeVirt(kv)
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
			Expect(totalPatches).To(Equal(patchCount - 4))
			// 2 for virt-controller and pdb
			Expect(totalUpdates).To(Equal(updateCount))

			Expect(resourceChanges["poddisruptionbudgets"][Patched]).To(Equal(1))
		}, 15)

		It("should pause update after daemonsets are rolled over", func(done Done) {
			defer close(done)

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
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)
			addInstallStrategy(updatedConfig)

			addAll(defaultConfig, kv)
			addPodsAndPodDisruptionBudgets(defaultConfig, kv)

			makeApiAndControllerReady()
			makeHandlerReady()

			addToCache = false
			shouldExpectRbacBackupCreations()
			shouldExpectPatchesAndUpdates()
			shouldExpectKubeVirtUpdateStatus(1)

			controller.Execute()

			kv = getLatestKubeVirt(kv)
			// conditions should reflect an ongoing update
			shouldExpectHCOConditions(kv, k8sv1.ConditionTrue, k8sv1.ConditionTrue, k8sv1.ConditionTrue)

			Expect(totalUpdates).To(Equal(updateCount))

			// daemonset, controller and apiserver pods are updated in this order.
			// this prevents the new API from coming online until the controllers can manage it.
			// The PDBs will prevent updated pods from getting "ready", so update should pause after
			//   daemonsets and before controller and namespace

			// 5 because virt-controller, virt-api, PDBs and the namespace are not patched
			Expect(totalPatches).To(Equal(patchCount - 5))

			// Make sure the 5 unpatched are as expected
			Expect(resourceChanges["deployments"][Patched]).To(Equal(0))          // virt-controller and virt-api unpatched
			Expect(resourceChanges["poddisruptionbudgets"][Patched]).To(Equal(0)) // PDBs unpatched
			Expect(resourceChanges["namespace"][Patched]).To(Equal(0))            // namespace unpatched
		}, 15)

		It("should pause update after controllers are rolled over", func(done Done) {
			defer close(done)

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
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)
			addInstallStrategy(updatedConfig)

			addAll(defaultConfig, kv)
			// Create virt-api and virt-controller under defaultConfig,
			// but use updatedConfig for virt-handler (hack) to avoid pausing after daemonsets
			addPodsWithIndividualConfigs(defaultConfig, defaultConfig, updatedConfig, true, kv)

			makeApiAndControllerReady()
			makeHandlerReady()

			addToCache = false
			shouldExpectRbacBackupCreations()
			shouldExpectPatchesAndUpdates()
			shouldExpectKubeVirtUpdateStatus(1)

			controller.Execute()

			kv = getLatestKubeVirt(kv)
			// conditions should reflect an ongoing update
			shouldExpectHCOConditions(kv, k8sv1.ConditionTrue, k8sv1.ConditionTrue, k8sv1.ConditionTrue)

			Expect(totalUpdates).To(Equal(updateCount))

			// The update was hacked to avoid pausing after rolling out the daemonsets (virt-handler)
			// That will allow both daemonset and controller pods to get patched before the pause.

			// 3 because virt-api, PDB and the namespace should not be patched
			Expect(totalPatches).To(Equal(patchCount - 3))

			// Make sure the 3 unpatched are as expected
			Expect(resourceChanges["deployments"][Patched]).To(Equal(1))          // virt-operator patched, virt-api unpatched
			Expect(resourceChanges["poddisruptionbudgets"][Patched]).To(Equal(1)) // 1 of 2 PDBs patched
			Expect(resourceChanges["namespace"][Patched]).To(Equal(0))            // namespace unpatched
		}, 15)

		It("should update kubevirt resources when Operator version changes if no imageTag and imageRegistry is explicitly set.", func() {
			os.Setenv(util.OperatorImageEnvName, fmt.Sprintf("%s/virt-operator:%s", "otherregistry", "1.1.1"))
			updatedConfig := getConfig("", "")

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
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)
			addInstallStrategy(updatedConfig)

			addAll(defaultConfig, kv)
			addPodsAndPodDisruptionBudgets(defaultConfig, kv)

			// pods for the new version are added so this test won't
			// wait for daemonsets to rollover before updating/patching
			// all resources.
			addPodsWithOptionalPodDisruptionBudgets(updatedConfig, false, kv)

			makeApiAndControllerReady()
			makeHandlerReady()

			shouldExpectPatchesAndUpdates()
			shouldExpectKubeVirtUpdateStatus(1)
			fakeNamespaceModificationEvent()
			shouldExpectNamespacePatch()

			controller.Execute()

			kv = getLatestKubeVirt(kv)
			// conditions should reflect a successful update
			shouldExpectHCOConditions(kv, k8sv1.ConditionTrue, k8sv1.ConditionFalse, k8sv1.ConditionFalse)

			Expect(totalPatches).To(Equal(patchCount))
			Expect(totalUpdates).To(Equal(updateCount))

			// ensure every resource is either patched or updated
			// + 1 is for the namespace patch which we don't consider as a resource we own.
			Expect(totalUpdates + totalPatches).To(Equal(resourceCount + 1))

			Expect(resourceChanges["poddisruptionbudgets"][Patched]).To(Equal(2))

		}, 15)

		It("should update resources when changing KubeVirt version.", func() {
			updatedConfig := getConfig("otherregistry", "1.1.1")

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
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)
			addInstallStrategy(updatedConfig)

			addAll(defaultConfig, kv)
			addPodsAndPodDisruptionBudgets(defaultConfig, kv)

			// pods for the new version are added so this test won't
			// wait for daemonsets to rollover before updating/patching
			// all resources.
			addPodsWithOptionalPodDisruptionBudgets(updatedConfig, false, kv)

			makeApiAndControllerReady()
			makeHandlerReady()

			shouldExpectPatchesAndUpdates()
			shouldExpectKubeVirtUpdateStatus(1)
			fakeNamespaceModificationEvent()
			shouldExpectNamespacePatch()

			controller.Execute()

			kv = getLatestKubeVirt(kv)
			// conditions should reflect a successful update
			shouldExpectHCOConditions(kv, k8sv1.ConditionTrue, k8sv1.ConditionFalse, k8sv1.ConditionFalse)

			Expect(totalPatches).To(Equal(patchCount))
			Expect(totalUpdates).To(Equal(updateCount))

			// ensure every resource is either patched or updated
			// + 1 is for the namespace patch which we don't consider as a resource we own.
			Expect(totalUpdates + totalPatches).To(Equal(resourceCount + 1))

		}, 15)

		It("should patch poddisruptionbudgets when changing KubeVirt version.", func() {
			updatedConfig := getConfig("otherregistry", "1.1.1")

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
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)
			addInstallStrategy(updatedConfig)

			addAll(defaultConfig, kv)
			addPodsAndPodDisruptionBudgets(defaultConfig, kv)

			// pods for the new version are added so this test won't
			// wait for daemonsets to rollover before updating/patching
			// all resources.
			addPodsWithOptionalPodDisruptionBudgets(updatedConfig, false, kv)

			makeApiAndControllerReady()
			makeHandlerReady()

			shouldExpectPatchesAndUpdates()
			shouldExpectKubeVirtUpdateStatus(1)

			controller.Execute()

			Expect(resourceChanges["poddisruptionbudgets"][Patched]).To(Equal(2))

		}, 15)

		It("should remove resources on deletion", func() {
			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-install",
					Namespace: NAMESPACE,
				},
			}
			kv.DeletionTimestamp = now()
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)

			// create all resources which should be deleted
			addInstallStrategy(defaultConfig)
			addAll(defaultConfig, kv)

			shouldExpectKubeVirtUpdateStatus(1)
			shouldExpectDeletions()
			shouldExpectInstallStrategyDeletion()

			controller.Execute()

			// Note: in real life during the first execution loop very probably only CRDs are deleted,
			// because that takes some time (see the check that the crd store is empty before going on with deletions)
			// But in this test the deletion succeeds immediately, so everything is deleted on first try
			Expect(totalDeletions).To(Equal(resourceCount))

			kv = getLatestKubeVirt(kv)
			Expect(kv.Status.Phase).To(Equal(v1.KubeVirtPhaseDeleted))
			Expect(len(kv.Status.Conditions)).To(Equal(3))
			shouldExpectHCOConditions(kv, k8sv1.ConditionFalse, k8sv1.ConditionFalse, k8sv1.ConditionTrue)
		}, 15)

		It("should remove poddisruptionbudgets on deletion", func() {
			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-install",
					Namespace: NAMESPACE,
				},
			}
			kv.DeletionTimestamp = now()
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)

			// create all resources which should be deleted
			addInstallStrategy(defaultConfig)
			addAll(defaultConfig, kv)

			shouldExpectKubeVirtUpdateStatus(1)
			shouldExpectDeletions()
			shouldExpectInstallStrategyDeletion()

			controller.Execute()

			Expect(resourceChanges["poddisruptionbudgets"][Deleted]).To(Equal(2))
		}, 15)
	})

	Context("On install strategy dump", func() {
		It("should generate latest install strategy and post as config map", func(done Done) {
			defer close(done)

			config, err := util.GetConfigFromEnv()
			Expect(err).ToNot(HaveOccurred())

			kubeClient.Fake.PrependReactor("create", "configmaps", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
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
			install.DumpInstallStrategyToConfigMap(virtClient, NAMESPACE)
		}, 15)
	})
})

func now() *metav1.Time {
	now := metav1.Now()
	return &now
}
