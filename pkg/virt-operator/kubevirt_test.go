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
 * Copyright The KubeVirt Authors.
 *
 */

package virt_operator

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	jsonpatch "github.com/evanphx/json-patch"
	routev1 "github.com/openshift/api/route/v1"
	secv1 "github.com/openshift/api/security/v1"
	routev1fake "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1/fake"
	secv1fake "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1/fake"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"go.uber.org/mock/gomock"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	k8sv1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extclientfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	v1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	promclientfake "kubevirt.io/client-go/prometheusoperator/fake"
	kvtesting "kubevirt.io/client-go/testing"
	"kubevirt.io/client-go/version"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	kubecontroller "kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/monitoring/rules"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
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

	// +1 for ContainerPathVolumes webhook (always enabled in tests)
	resourceCount = 94 + virtTemplateResourceCount
	patchCount    = 62 + virtTemplatePatchCount
	updateCount   = 33 + virtTemplateUpdateCount

	// 1 because a temporary validation webhook is created to block new CRDs until api server is deployed
	expectedTemporaryResources = 1
	externalCAConfigMapCount   = 1

	// virtTemplateResourceCount consists of 30 objects from the bundled template + 3 secrets
	virtTemplateResourceCount = 33
	virtTemplatePatchCount    = 16
	virtTemplateUpdateCount   = 17

	numVirtTemplateCRDs = 2
)

var (
	crdFunctions = []func() (*extv1.CustomResourceDefinition, error){
		components.NewVirtualMachineInstanceCrd, components.NewPresetCrd, components.NewReplicaSetCrd,
		components.NewVirtualMachineCrd, components.NewVirtualMachineInstanceMigrationCrd,
		components.NewVirtualMachineSnapshotCrd, components.NewVirtualMachineSnapshotContentCrd,
		components.NewVirtualMachineExportCrd, components.NewVirtualMachineBackupCrd,
		components.NewVirtualMachineRestoreCrd, components.NewVirtualMachineInstancetypeCrd,
		components.NewVirtualMachineClusterInstancetypeCrd, components.NewVirtualMachinePoolCrd,
		components.NewMigrationPolicyCrd, components.NewVirtualMachinePreferenceCrd,
		components.NewVirtualMachineClusterPreferenceCrd, components.NewVirtualMachineCloneCrd,
		components.NewVirtualMachineBackupTrackerCrd,
	}
	numCRDs = len(crdFunctions) + numVirtTemplateCRDs
)

type KubeVirtTestData struct {
	ctrl             *gomock.Controller
	kvInterface      *kubecli.MockKubeVirtInterface
	apiServiceClient *install.MockAPIServiceInterface

	controller *KubeVirtController

	recorder *record.FakeRecorder

	mockQueue      *testutils.MockWorkQueue[string]
	virtClient     *kubecli.MockKubevirtClient
	virtFakeClient *kubevirtfake.Clientset
	kubeClient     *fake.Clientset
	secClient      *secv1fake.FakeSecurityV1
	extClient      *extclientfake.Clientset
	promClient     *promclientfake.Clientset
	routeClient    *routev1fake.FakeRouteV1

	totalAdds      int
	totalUpdates   int
	totalPatches   int
	totalDeletions int

	resourceChanges map[string]map[string]int

	deleteFromCache bool
	addToCache      bool

	defaultConfig     *util.KubeVirtDeploymentConfig
	mockEnvVarManager util.EnvVarManager

	deploymentPatchReactionFunc testing.ReactionFunc
	daemonSetPatchReactionFunc  testing.ReactionFunc
}

func (k *KubeVirtTestData) BeforeTest() {

	k.mockEnvVarManager = &util.EnvVarManagerMock{}

	err := k.mockEnvVarManager.Setenv(util.OldOperatorImageEnvName, fmt.Sprintf("%s/virt-operator:%s", "someregistry", "v9.9.9"))
	Expect(err).NotTo(HaveOccurred())

	util.DefaultEnvVarManager = k.mockEnvVarManager

	k.defaultConfig = k.getConfig("", "")

	k.totalAdds = 0
	k.totalUpdates = 0
	k.totalPatches = 0
	k.totalDeletions = 0
	k.resourceChanges = make(map[string]map[string]int)
	k.deleteFromCache = true
	k.addToCache = true

	k.ctrl = gomock.NewController(GinkgoT())
	k.virtClient = kubecli.NewMockKubevirtClient(k.ctrl)
	k.kvInterface = kubecli.NewMockKubeVirtInterface(k.ctrl)
	k.apiServiceClient = install.NewMockAPIServiceInterface(k.ctrl)

	k.recorder = record.NewFakeRecorder(100)
	k.recorder.IncludeObject = true

	informers := util.Informers{}
	informers.KubeVirt, _ = testutils.NewFakeInformerFor(&v1.KubeVirt{})
	informers.ServiceAccount, _ = testutils.NewFakeInformerFor(&k8sv1.ServiceAccount{})
	informers.ServiceAccount, _ = testutils.NewFakeInformerFor(&k8sv1.ServiceAccount{})
	informers.ClusterRole, _ = testutils.NewFakeInformerFor(&rbacv1.ClusterRole{})
	informers.ClusterRoleBinding, _ = testutils.NewFakeInformerFor(&rbacv1.ClusterRoleBinding{})
	informers.Role, _ = testutils.NewFakeInformerFor(&rbacv1.Role{})
	informers.RoleBinding, _ = testutils.NewFakeInformerFor(&rbacv1.RoleBinding{})
	informers.OperatorCrd, _ = testutils.NewFakeInformerFor(&extv1.CustomResourceDefinition{})
	informers.Service, _ = testutils.NewFakeInformerFor(&k8sv1.Service{})
	informers.Deployment, _ = testutils.NewFakeInformerFor(&appsv1.Deployment{})
	informers.DaemonSet, _ = testutils.NewFakeInformerFor(&appsv1.DaemonSet{})
	informers.ValidationWebhook, _ = testutils.NewFakeInformerFor(&admissionregistrationv1.ValidatingWebhookConfiguration{})
	informers.MutatingWebhook, _ = testutils.NewFakeInformerFor(&admissionregistrationv1.MutatingWebhookConfiguration{})
	informers.APIService, _ = testutils.NewFakeInformerFor(&apiregv1.APIService{})
	informers.SCC, _ = testutils.NewFakeInformerFor(&secv1.SecurityContextConstraints{})
	informers.Route, _ = testutils.NewFakeInformerFor(&routev1.Route{})
	informers.InstallStrategyConfigMap, _ = testutils.NewFakeInformerFor(&k8sv1.ConfigMap{})
	informers.InstallStrategyJob, _ = testutils.NewFakeInformerFor(&batchv1.Job{})
	informers.InfrastructurePod, _ = testutils.NewFakeInformerFor(&k8sv1.Pod{})
	informers.PodDisruptionBudget, _ = testutils.NewFakeInformerFor(&policyv1.PodDisruptionBudget{})
	informers.Namespace, _ = testutils.NewFakeInformerWithIndexersFor(
		&k8sv1.Namespace{}, cache.Indexers{
			"namespace_name": func(obj interface{}) ([]string, error) {
				return []string{obj.(*k8sv1.Namespace).GetName()}, nil
			},
		})
	informers.ServiceMonitor, _ = testutils.NewFakeInformerFor(&promv1.ServiceMonitor{Spec: promv1.ServiceMonitorSpec{}})
	informers.PrometheusRule, _ = testutils.NewFakeInformerFor(&promv1.PrometheusRule{Spec: promv1.PrometheusRuleSpec{}})
	informers.Secrets, _ = testutils.NewFakeInformerFor(&k8sv1.Secret{})
	informers.ConfigMap, _ = testutils.NewFakeInformerFor(&k8sv1.ConfigMap{})
	informers.ValidatingAdmissionPolicyBinding, _ = testutils.NewFakeInformerFor(&admissionregistrationv1.ValidatingAdmissionPolicyBinding{})
	informers.ValidatingAdmissionPolicy, _ = testutils.NewFakeInformerFor(&admissionregistrationv1.ValidatingAdmissionPolicy{})
	informers.ClusterInstancetype, _ = testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterInstancetype{})
	informers.ClusterPreference, _ = testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterPreference{})
	informers.Leases, _ = testutils.NewFakeInformerFor(&coordinationv1.Lease{})

	// test OpenShift components
	config := util.OperatorConfig{
		IsOnOpenshift:                           true,
		ServiceMonitorEnabled:                   true,
		PrometheusRulesEnabled:                  true,
		ValidatingAdmissionPolicyBindingEnabled: true,
		ValidatingAdmissionPolicyEnabled:        true,
	}
	k.controller, _ = NewKubeVirtController(k.virtClient, k.apiServiceClient, k.recorder, config, informers, NAMESPACE)
	k.controller.delayedQueueAdder = func(key string, queue workqueue.TypedRateLimitingInterface[string]) {
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

	k.routeClient = &routev1fake.FakeRouteV1{
		Fake: &fake.NewSimpleClientset().Fake,
	}

	k.virtFakeClient = kubevirtfake.NewSimpleClientset()

	k.virtClient.EXPECT().AdmissionregistrationV1().Return(k.kubeClient.AdmissionregistrationV1()).AnyTimes()
	k.virtClient.EXPECT().CoreV1().Return(k.kubeClient.CoreV1()).AnyTimes()
	k.virtClient.EXPECT().BatchV1().Return(k.kubeClient.BatchV1()).AnyTimes()
	k.virtClient.EXPECT().RbacV1().Return(k.kubeClient.RbacV1()).AnyTimes()
	k.virtClient.EXPECT().AppsV1().Return(k.kubeClient.AppsV1()).AnyTimes()
	k.virtClient.EXPECT().SecClient().Return(k.secClient).AnyTimes()
	k.virtClient.EXPECT().ExtensionsClient().Return(k.extClient).AnyTimes()
	k.virtClient.EXPECT().PolicyV1().Return(k.kubeClient.PolicyV1()).AnyTimes()
	k.virtClient.EXPECT().PrometheusClient().Return(k.promClient).AnyTimes()
	k.virtClient.EXPECT().RouteClient().Return(k.routeClient).AnyTimes()
	k.virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(k.virtFakeClient.InstancetypeV1beta1().VirtualMachineClusterInstancetypes()).AnyTimes()
	k.virtClient.EXPECT().VirtualMachineClusterPreference().Return(k.virtFakeClient.InstancetypeV1beta1().VirtualMachineClusterPreferences()).AnyTimes()

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
		if action.GetVerb() == "create" && action.GetResource().Resource == "validatingadmissionpolicybindings" {
			dummyValidatingAdmissionPolicyBinding := &admissionregistrationv1.ValidatingAdmissionPolicyBinding{}
			return true, dummyValidatingAdmissionPolicyBinding, nil
		}
		if action.GetVerb() == "create" && action.GetResource().Resource == "validatingadmissionpolicies" {
			dummyValidatingAdmissionPolicy := &admissionregistrationv1.ValidatingAdmissionPolicy{}
			return true, dummyValidatingAdmissionPolicy, nil
		}

		if action.GetVerb() == "create" && action.GetResource().Resource == "configmaps" {
			dummyConfigMap := &k8sv1.ConfigMap{}
			return true, dummyConfigMap, nil
		}

		if action.GetVerb() == "update" && action.GetResource().Resource == "configmaps" {
			return true, nil, nil
		}

		if action.GetVerb() == "get" && action.GetResource().Resource == "serviceaccounts" {
			return true, nil, errors.NewNotFound(schema.GroupResource{Group: "", Resource: "serviceaccounts"}, "whatever")
		}
		if action.GetVerb() == "get" && action.GetResource().Resource == "configmaps" {
			return true, nil, errors.NewNotFound(schema.GroupResource{Group: "", Resource: "configmaps"}, "whatever")
		}
		if action.GetVerb() == "list" && action.GetResource().Resource == "nodes" {
			dummyNode := k8sv1.Node{}
			nodeList := &k8sv1.NodeList{Items: []k8sv1.Node{dummyNode, *dummyNode.DeepCopy(), *dummyNode.DeepCopy()}}
			return true, nodeList, nil
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
	k.routeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		Expect(action).To(BeNil())
		return true, nil, nil
	})

	// add the privileged SCC without KubeVirt accounts
	scc := getSCC()
	k.controller.stores.SCCCache.Add(&scc)

	k.deleteFromCache = true
	k.addToCache = true

	err = rules.SetupRules(k.defaultConfig.Namespace)
	Expect(err).ToNot(HaveOccurred())
}

func (k *KubeVirtTestData) AfterTest() {
	util.DefaultEnvVarManager = nil

	// Ensure that we add checks for expected events to every test
	Expect(k.recorder.Events).To(BeEmpty())
}

type finalizerPatch struct {
	Op    string   `json:"op"`
	Path  string   `json:"path"`
	Value []string `json:"value"`
}

func extractFinalizers(data []byte) []string {
	patches := make([]finalizerPatch, 0)
	err := json.Unmarshal(data, &patches)
	Expect(patches).To(HaveLen(1))
	patch := patches[0]
	Expect(err).ToNot(HaveOccurred())
	Expect(patch.Op).To(Equal("replace"))
	Expect(patch.Path).To(Equal("/metadata/finalizers"))
	return patch.Value
}

func (k *KubeVirtTestData) shouldExpectKubeVirtFinalizersPatch(times int) {
	patch := k.kvInterface.EXPECT().Patch(context.Background(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	patch.DoAndReturn(func(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, _ ...string) (*v1.KubeVirt, error) {
		Expect(pt).To(Equal(types.JSONPatchType))
		finalizers := extractFinalizers(data)
		obj, exists, err := k.controller.stores.KubeVirtCache.GetByKey(NAMESPACE + "/" + name)
		Expect(err).ToNot(HaveOccurred())
		Expect(exists).To(BeTrue())
		kv := obj.(*v1.KubeVirt)
		kv.Finalizers = finalizers
		err = k.controller.stores.KubeVirtCache.Update(kv)
		Expect(err).ToNot(HaveOccurred())
		return kv, nil
	}).Times(times)
}

func (k *KubeVirtTestData) shouldExpectKubeVirtUpdateStatus(times int) {
	update := k.kvInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any(), metav1.UpdateOptions{})
	update.Do(func(ctx context.Context, kv *v1.KubeVirt, options metav1.UpdateOptions) {
		k.controller.stores.KubeVirtCache.Update(kv)
		update.Return(kv, nil)
	}).Times(times)
}

func (k *KubeVirtTestData) shouldExpectKubeVirtUpdateStatusVersion(times int, config *util.KubeVirtDeploymentConfig) {
	update := k.kvInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any(), metav1.UpdateOptions{})
	update.Do(func(ctx context.Context, kv *v1.KubeVirt, options metav1.UpdateOptions) {

		Expect(kv.Status.TargetKubeVirtVersion).To(Equal(config.GetKubeVirtVersion()))
		Expect(kv.Status.ObservedKubeVirtVersion).To(Equal(config.GetKubeVirtVersion()))
		k.controller.stores.KubeVirtCache.Update(kv)
		update.Return(kv, nil)
	}).Times(times)
}

func (k *KubeVirtTestData) shouldExpectKubeVirtUpdateStatusFailureCondition(reason string) {
	update := k.kvInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any(), metav1.UpdateOptions{})
	update.Do(func(ctx context.Context, kv *v1.KubeVirt, options metav1.UpdateOptions) {
		Expect(kv.Status.Conditions).To(HaveLen(1))
		Expect(kv.Status.Conditions[0].Reason).To(Equal(reason))
		k.controller.stores.KubeVirtCache.Update(kv)
		update.Return(kv, nil)
	}).Times(1)
}

func (k *KubeVirtTestData) addKubeVirt(kv *v1.KubeVirt) {
	k.controller.stores.KubeVirtCache.Add(kv)
	key, err := kubecontroller.KeyFunc(kv)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) getLatestKubeVirt(kv *v1.KubeVirt) *v1.KubeVirt {
	if obj, exists, _ := k.controller.stores.KubeVirtCache.GetByKey(kv.GetNamespace() + "/" + kv.GetName()); exists {
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
	k.routeClient.Fake.PrependReactor("delete", "routes", genericDeleteFunc)
	k.kubeClient.Fake.PrependReactor("delete", "validatingadmissionpolicybindings", genericDeleteFunc)
	k.kubeClient.Fake.PrependReactor("delete", "validatingadmissionpolicies", genericDeleteFunc)
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
	case "validatingadmissionpolicybindings":
		k.deleteValidatingAdmissionPolicyBinding(key)
	case "validatingadmissionpolicies":
		k.deleteValidatingAdmissionPolicy(key)
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
	case "routes":
		k.deleteRoute(key)
	default:
		Fail(fmt.Sprintf("unknown resource type %+v", resource))
	}
	if _, ok := k.resourceChanges[resource]; !ok {
		k.resourceChanges[resource] = make(map[string]int)
	}
	k.resourceChanges[resource][Deleted]++
}

func (k *KubeVirtTestData) deleteServiceAccount(key string) {
	if obj, exists, _ := k.controller.stores.ServiceAccountCache.GetByKey(key); exists {
		k.controller.stores.ServiceAccountCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deleteClusterRole(key string) {
	if obj, exists, _ := k.controller.stores.ClusterRoleCache.GetByKey(key); exists {
		k.controller.stores.ClusterRoleCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deleteClusterRoleBinding(key string) {
	if obj, exists, _ := k.controller.stores.ClusterRoleBindingCache.GetByKey(key); exists {
		k.controller.stores.ClusterRoleBindingCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deleteRole(key string) {
	if obj, exists, _ := k.controller.stores.RoleCache.GetByKey(key); exists {
		k.controller.stores.RoleCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deleteRoleBinding(key string) {
	if obj, exists, _ := k.controller.stores.RoleBindingCache.GetByKey(key); exists {
		k.controller.stores.RoleBindingCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deleteCrd(key string) {
	if obj, exists, _ := k.controller.stores.OperatorCrdCache.GetByKey(key); exists {
		k.controller.stores.OperatorCrdCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deleteService(key string) {
	if obj, exists, _ := k.controller.stores.ServiceCache.GetByKey(key); exists {
		k.controller.stores.ServiceCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deleteDeployment(key string) {
	if obj, exists, _ := k.controller.stores.DeploymentCache.GetByKey(key); exists {
		k.controller.stores.DeploymentCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deleteDaemonset(key string) {
	if obj, exists, _ := k.controller.stores.DaemonSetCache.GetByKey(key); exists {
		k.controller.stores.DaemonSetCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deleteValidationWebhook(key string) {
	if obj, exists, _ := k.controller.stores.ValidationWebhookCache.GetByKey(key); exists {
		k.controller.stores.ValidationWebhookCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deleteMutatingWebhook(key string) {
	if obj, exists, _ := k.controller.stores.MutatingWebhookCache.GetByKey(key); exists {
		k.controller.stores.MutatingWebhookCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deleteAPIService(key string) {
	if obj, exists, _ := k.controller.stores.APIServiceCache.GetByKey(key); exists {
		k.controller.stores.APIServiceCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deleteInstallStrategyJob(key string) {
	if obj, exists, _ := k.controller.stores.InstallStrategyJobCache.GetByKey(key); exists {
		k.controller.stores.InstallStrategyJobCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deletePodDisruptionBudget(key string) {
	if obj, exists, _ := k.controller.stores.PodDisruptionBudgetCache.GetByKey(key); exists {
		k.controller.stores.PodDisruptionBudgetCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deleteSecret(key string) {
	if obj, exists, _ := k.controller.stores.SecretCache.GetByKey(key); exists {
		k.controller.stores.SecretCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deleteConfigMap(key string) {
	if obj, exists, _ := k.controller.stores.ConfigMapCache.GetByKey(key); exists {
		configMap := obj.(*k8sv1.ConfigMap)
		k.controller.stores.ConfigMapCache.Delete(configMap)
	} else if obj, exists, _ := k.controller.stores.InstallStrategyConfigMapCache.GetByKey(key); exists {
		configMap := obj.(*k8sv1.ConfigMap)
		k.controller.stores.InstallStrategyConfigMapCache.Delete(configMap)
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deleteValidatingAdmissionPolicyBinding(key string) {
	if obj, exists, _ := k.controller.stores.ValidatingAdmissionPolicyBindingCache.GetByKey(key); exists {
		k.controller.stores.ValidatingAdmissionPolicyBindingCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deleteValidatingAdmissionPolicy(key string) {
	if obj, exists, _ := k.controller.stores.ValidatingAdmissionPolicyCache.GetByKey(key); exists {
		k.controller.stores.ValidatingAdmissionPolicyCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deleteSCC(key string) {
	if obj, exists, _ := k.controller.stores.SCCCache.GetByKey(key); exists {
		k.controller.stores.SCCCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deleteRoute(key string) {
	if obj, exists, _ := k.controller.stores.RouteCache.GetByKey(key); exists {
		k.controller.stores.RouteCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deleteServiceMonitor(key string) {
	if obj, exists, _ := k.controller.stores.ServiceMonitorCache.GetByKey(key); exists {
		k.controller.stores.ServiceMonitorCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) deletePrometheusRule(key string) {
	if obj, exists, _ := k.controller.stores.PrometheusRuleCache.GetByKey(key); exists {
		k.controller.stores.PrometheusRuleCache.Delete(obj.(runtime.Object))
	}
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) shouldExpectPatchesAndUpdates(kv *v1.KubeVirt) {
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
	k.apiServiceClient.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Do(func(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, _ ...string) {
		genericPatchFunc(&testing.PatchActionImpl{ActionImpl: testing.ActionImpl{Resource: schema.GroupVersionResource{Resource: "apiservices"}}})
	})
	if exportProxyEnabled(kv) {
		k.routeClient.Fake.PrependReactor("patch", "routes", genericPatchFunc)
	}
	k.kubeClient.Fake.PrependReactor("patch", "validatingadmissionpolicybindings", genericPatchFunc)
	k.kubeClient.Fake.PrependReactor("patch", "validatingadmissionpolicies", genericPatchFunc)
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
	if k.deploymentPatchReactionFunc != nil {
		return k.deploymentPatchReactionFunc
	}

	return func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		k.genericPatchFunc()(action)

		patchAction, ok := action.(testing.PatchAction)

		Expect(ok).To(BeTrue())

		replicas := int32(2)
		if patchAction.GetName() == components.VirtTemplateApiserverDeploymentName ||
			patchAction.GetName() == components.VirtTemplateControllerDeploymentName {
			replicas = 1
		}

		return true, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      patchAction.GetName(),
				Namespace: patchAction.GetNamespace(),
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
			},
		}, nil
	}
}

func (k *KubeVirtTestData) daemonsetPatchFunc() func(action testing.Action) (handled bool, obj runtime.Object, err error) {
	if k.daemonSetPatchReactionFunc != nil {
		return k.daemonSetPatchReactionFunc
	}

	return func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		k.genericPatchFunc()(action)

		return true, &appsv1.DaemonSet{}, nil
	}
}

func (k *KubeVirtTestData) podDisruptionBudgetPatchFunc() func(action testing.Action) (handled bool, obj runtime.Object, err error) {
	return func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		k.genericPatchFunc()(action)

		return true, &policyv1.PodDisruptionBudget{}, nil
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
	k.kubeClient.Fake.PrependReactor("create", "validatingadmissionpolicybindings", genericCreateFunc)
	k.kubeClient.Fake.PrependReactor("create", "validatingadmissionpolicies", genericCreateFunc)
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
	resourceKey := ""
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
	case *policyv1.PodDisruptionBudget:
		injectMetadata(&obj.(*policyv1.PodDisruptionBudget).ObjectMeta, config)
		k.addPodDisruptionBudget(resource, kv)
	case *k8sv1.Secret:
		injectMetadata(&obj.(*k8sv1.Secret).ObjectMeta, config)
		k.addSecret(resource)
	case *secv1.SecurityContextConstraints:
		injectMetadata(&obj.(*secv1.SecurityContextConstraints).ObjectMeta, config)
		k.addSCC(resource)
		resourceKey = "securitycontextconstraints"
	case *promv1.ServiceMonitor:
		injectMetadata(&obj.(*promv1.ServiceMonitor).ObjectMeta, config)
		k.addServiceMonitor(resource)
	case *promv1.PrometheusRule:
		injectMetadata(&obj.(*promv1.PrometheusRule).ObjectMeta, config)
		k.addPrometheusRule(resource)
	case *routev1.Route:
		injectMetadata(&obj.(*routev1.Route).ObjectMeta, config)
		k.addRoute(resource)
	case *admissionregistrationv1.ValidatingAdmissionPolicyBinding:
		injectMetadata(&obj.(*admissionregistrationv1.ValidatingAdmissionPolicyBinding).ObjectMeta, config)
		k.addValidatingAdmissionPolicyBinding(resource)
	case *admissionregistrationv1.ValidatingAdmissionPolicy:
		injectMetadata(&obj.(*admissionregistrationv1.ValidatingAdmissionPolicy).ObjectMeta, config)
		k.addValidatingAdmissionPolicy(resource)
		resourceKey = "validatingadmissionpolicies"
	default:
		Fail("unknown resource type")
	}
	split := strings.Split(fmt.Sprintf("%T", obj), ".")
	if resourceKey == "" {
		resourceKey = strings.ToLower(split[len(split)-1]) + "s"
	}
	if _, ok := k.resourceChanges[resourceKey]; !ok {
		k.resourceChanges[resourceKey] = make(map[string]int)
	}
	k.resourceChanges[resourceKey][Added]++
}

func (k *KubeVirtTestData) addServiceAccount(sa *k8sv1.ServiceAccount) {
	k.controller.stores.ServiceAccountCache.Add(sa)
	key, err := kubecontroller.KeyFunc(sa)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addClusterRole(cr *rbacv1.ClusterRole) {
	k.controller.stores.ClusterRoleCache.Add(cr)
	key, err := kubecontroller.KeyFunc(cr)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addClusterRoleBinding(crb *rbacv1.ClusterRoleBinding) {
	k.controller.stores.ClusterRoleBindingCache.Add(crb)
	key, err := kubecontroller.KeyFunc(crb)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addRole(role *rbacv1.Role) {
	k.controller.stores.RoleCache.Add(role)
	key, err := kubecontroller.KeyFunc(role)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addRoleBinding(rb *rbacv1.RoleBinding) {
	k.controller.stores.RoleBindingCache.Add(rb)
	key, err := kubecontroller.KeyFunc(rb)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addCrd(crd *extv1.CustomResourceDefinition, kv *v1.KubeVirt) {
	if kv != nil {
		apply.SetGeneration(&kv.Status.Generations, crd)
	}
	k.controller.stores.OperatorCrdCache.Add(crd)
	key, err := kubecontroller.KeyFunc(crd)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addService(svc *k8sv1.Service) {
	k.controller.stores.ServiceCache.Add(svc)
	key, err := kubecontroller.KeyFunc(svc)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addDeployment(depl *appsv1.Deployment, kv *v1.KubeVirt) {
	if kv != nil {
		apply.SetGeneration(&kv.Status.Generations, depl)
	}
	k.controller.stores.DeploymentCache.Add(depl)
	key, err := kubecontroller.KeyFunc(depl)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addDaemonset(ds *appsv1.DaemonSet, kv *v1.KubeVirt) {
	if kv != nil {
		apply.SetGeneration(&kv.Status.Generations, ds)
	}
	k.controller.stores.DaemonSetCache.Add(ds)
	key, err := kubecontroller.KeyFunc(ds)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addMutatingWebhook(wh *admissionregistrationv1.MutatingWebhookConfiguration, kv *v1.KubeVirt) {
	if kv != nil {
		apply.SetGeneration(&kv.Status.Generations, wh)
	}
	k.controller.stores.MutatingWebhookCache.Add(wh)
	key, err := kubecontroller.KeyFunc(wh)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addAPIService(as *apiregv1.APIService) {
	k.controller.stores.APIServiceCache.Add(as)
	key, err := kubecontroller.KeyFunc(as)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addInstallStrategyJob(job *batchv1.Job) {
	k.controller.stores.InstallStrategyJobCache.Add(job)
	key, err := kubecontroller.KeyFunc(job)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addPod(pod *k8sv1.Pod) {
	k.controller.stores.InfrastructurePodCache.Add(pod)
	key, err := kubecontroller.KeyFunc(pod)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addPodDisruptionBudget(podDisruptionBudget *policyv1.PodDisruptionBudget, kv *v1.KubeVirt) {
	if kv != nil {
		apply.SetGeneration(&kv.Status.Generations, podDisruptionBudget)
	}
	k.controller.stores.PodDisruptionBudgetCache.Add(podDisruptionBudget)
	key, err := kubecontroller.KeyFunc(podDisruptionBudget)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addSecret(secret *k8sv1.Secret) {
	k.controller.stores.SecretCache.Add(secret)
	key, err := kubecontroller.KeyFunc(secret)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addValidatingAdmissionPolicyBinding(vapb *admissionregistrationv1.ValidatingAdmissionPolicyBinding) {
	k.controller.stores.ValidatingAdmissionPolicyBindingCache.Add(vapb)
	key, err := kubecontroller.KeyFunc(vapb)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addValidatingAdmissionPolicy(vap *admissionregistrationv1.ValidatingAdmissionPolicy) {
	k.controller.stores.ValidatingAdmissionPolicyCache.Add(vap)
	key, err := kubecontroller.KeyFunc(vap)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addConfigMap(configMap *k8sv1.ConfigMap) {
	if _, ok := configMap.Labels[v1.InstallStrategyLabel]; ok {
		k.controller.stores.InstallStrategyConfigMapCache.Add(configMap)
	} else {
		k.controller.stores.ConfigMapCache.Add(configMap)
	}
	key, err := kubecontroller.KeyFunc(configMap)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addSCC(scc *secv1.SecurityContextConstraints) {
	k.controller.stores.SCCCache.Add(scc)
	key, err := kubecontroller.KeyFunc(scc)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addRoute(route *routev1.Route) {
	k.controller.stores.RouteCache.Add(route)
	key, err := kubecontroller.KeyFunc(route)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addServiceMonitor(serviceMonitor *promv1.ServiceMonitor) {
	k.controller.stores.ServiceMonitorCache.Add(serviceMonitor)
	key, err := kubecontroller.KeyFunc(serviceMonitor)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addPrometheusRule(prometheusRule *promv1.PrometheusRule) {
	k.controller.stores.PrometheusRuleCache.Add(prometheusRule)
	key, err := kubecontroller.KeyFunc(prometheusRule)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) generateRandomResources() int {
	version := fmt.Sprintf("rand-%s", rand.String(10))
	registry := fmt.Sprintf("rand-%s", rand.String(10))
	config := k.getConfig(registry, version)

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

func enableExportFeatureGate(kv *v1.KubeVirt) {
	enableFeatureGate(kv, featuregate.VMExportGate)
}

func enableTemplateFeatureGate(kv *v1.KubeVirt) {
	enableFeatureGate(kv, featuregate.Template)
}

func enableFeatureGate(kv *v1.KubeVirt, fg string) {
	if kv.Spec.Configuration.DeveloperConfiguration == nil {
		kv.Spec.Configuration.DeveloperConfiguration = &v1.DeveloperConfiguration{}
	}
	kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = append(
		kv.Spec.Configuration.DeveloperConfiguration.FeatureGates,
		fg,
	)
}

func enableContainerPathVolumesFeatureGate(kv *v1.KubeVirt) {
	enableFeatureGate(kv, featuregate.ContainerPathVolumesGate)
}

func containerPathVolumesEnabled(kv *v1.KubeVirt) bool {
	return isFeatureGateEnabled(kv, featuregate.ContainerPathVolumesGate)
}

func exportProxyEnabled(kv *v1.KubeVirt) bool {
	return isFeatureGateEnabled(kv, featuregate.VMExportGate)
}

func synchronizationControllerEnabled(kv *v1.KubeVirt) bool {
	return isFeatureGateEnabled(kv, featuregate.DecentralizedLiveMigration)
}

func virtTemplateDeploymentEnabled(kv *v1.KubeVirt) bool {
	if !isFeatureGateEnabled(kv, featuregate.Template) {
		return false
	}
	virtTemplateDeployment := kv.Spec.Configuration.VirtTemplateDeployment
	return virtTemplateDeployment == nil || virtTemplateDeployment.Enabled == nil || *virtTemplateDeployment.Enabled
}

func isFeatureGateEnabled(kv *v1.KubeVirt, fg string) bool {
	if kv.Spec.Configuration.DeveloperConfiguration == nil {
		return false
	}
	for _, enabledFg := range kv.Spec.Configuration.DeveloperConfiguration.FeatureGates {
		if enabledFg == fg {
			return true
		}
	}
	return false
}

func (k *KubeVirtTestData) addAllWithExclusionMap(config *util.KubeVirtDeploymentConfig, kv *v1.KubeVirt, exclusionMap map[string]bool) {
	c, _ := apply.NewCustomizer(kv.Spec.CustomizeComponents)

	all := make([]runtime.Object, 0)

	// rbac
	all = append(all, rbac.GetAllCluster()...)
	all = append(all, rbac.GetAllApiServer(NAMESPACE)...)
	all = append(all, rbac.GetAllHandler(NAMESPACE)...)
	all = append(all, rbac.GetAllController(NAMESPACE, true)...)
	all = append(all, rbac.GetAllExportProxy(NAMESPACE)...)
	all = append(all, rbac.GetAllSynchronizationController(NAMESPACE)...)

	// crds
	for _, f := range crdFunctions {
		crd, err := f()
		if err != nil {
			panic(fmt.Errorf("This should not happen, %v", err))
		}
		all = append(all, crd)
	}
	// cr
	pr, err := rules.BuildPrometheusRule(config.GetNamespace())
	Expect(err).ToNot(HaveOccurred())
	all = append(all, pr)
	// sccs
	all = append(all, components.NewKubeVirtControllerSCC(NAMESPACE))
	all = append(all, components.NewKubeVirtHandlerSCC(NAMESPACE))
	// services and deployments
	all = append(all, components.NewOperatorWebhookService(NAMESPACE))
	all = append(all, components.NewPrometheusService(NAMESPACE))
	all = append(all, components.NewApiServerService(NAMESPACE))
	all = append(all, components.NewExportProxyService(NAMESPACE))

	config.Namespace = NAMESPACE
	apiDeployment := components.NewApiServerDeployment(config, "", "", "")
	apiDeploymentPdb := components.NewPodDisruptionBudgetForDeployment(apiDeployment)
	controller := components.NewControllerDeployment(config, "", "", "")
	controllerPdb := components.NewPodDisruptionBudgetForDeployment(controller)

	handler := components.NewHandlerDaemonSet(config, "", "", "")
	all = append(all, apiDeployment, apiDeploymentPdb, controller, controllerPdb, handler)

	if exportProxyEnabled(kv) {
		exportProxy := components.NewExportProxyDeployment(config, "", "", "")
		exportProxyPdb := components.NewPodDisruptionBudgetForDeployment(exportProxy)
		route := components.NewExportProxyRoute(NAMESPACE)
		all = append(all, exportProxy, exportProxyPdb, route)
	}

	all = append(all, rbac.GetAllServiceMonitor(NAMESPACE, config.GetPotentialMonitorNamespaces()[0], config.GetMonitorServiceAccountName())...)
	all = append(all, components.NewServiceMonitorCR(NAMESPACE, config.GetPotentialMonitorNamespaces()[0], true))

	// ca certificate
	caSecrets := components.NewCACertSecrets(NAMESPACE)
	var caSecret *k8sv1.Secret
	var caExportSecret *k8sv1.Secret
	var caBackupSecret *k8sv1.Secret
	for _, ca := range caSecrets {
		switch ca.Name {
		case components.KubeVirtCASecretName:
			caSecret = ca
		case components.KubeVirtExportCASecretName:
			caExportSecret = ca
		case components.KubeVirtBackupCASecretName:
			caBackupSecret = ca
		}
	}
	components.PopulateSecretWithCertificate(caSecret, nil, &metav1.Duration{Duration: apply.Duration7d})
	caCert, _ := components.LoadCertificates(caSecret)
	caBundle := cert.EncodeCertPEM(caCert.Leaf)
	all = append(all, caSecret)

	caExportDuration := metav1.Duration{Duration: time.Hour * 24 * 7} // 7 Days
	components.PopulateSecretWithCertificate(caExportSecret, nil, &caExportDuration)
	caExportCert, _ := components.LoadCertificates(caExportSecret)
	caExportBundle := cert.EncodeCertPEM(caExportCert.Leaf)
	all = append(all, caExportSecret)

	caBackupDuration := metav1.Duration{Duration: time.Hour * 24 * 7} // 7 Days
	components.PopulateSecretWithCertificate(caBackupSecret, nil, &caBackupDuration)
	caBackupCert, _ := components.LoadCertificates(caBackupSecret)
	caBackupBundle := cert.EncodeCertPEM(caBackupCert.Leaf)
	all = append(all, caBackupSecret)

	configMaps := components.NewCAConfigMaps(NAMESPACE)
	var caConfigMap *k8sv1.ConfigMap
	var caExportConfigMap *k8sv1.ConfigMap
	var caBackupConfigMap *k8sv1.ConfigMap
	for _, cm := range configMaps {
		switch cm.Name {
		case components.KubeVirtCASecretName:
			caConfigMap = cm
		case components.KubeVirtExportCASecretName:
			caExportConfigMap = cm
		case components.KubeVirtBackupCASecretName:
			caBackupConfigMap = cm
		}
	}

	caConfigMap.Data = map[string]string{components.CABundleKey: string(caBundle)}
	all = append(all, caConfigMap)
	caExportConfigMap.Data = map[string]string{components.CABundleKey: string(caExportBundle)}
	all = append(all, caExportConfigMap)
	caBackupConfigMap.Data = map[string]string{components.CABundleKey: string(caBackupBundle)}
	all = append(all, caBackupConfigMap)

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

	if containerPathVolumesEnabled(kv) {
		containerPathWebhook := components.NewVirtLauncherPodMutatingWebhookConfiguration(config.GetNamespace())
		for i := range containerPathWebhook.Webhooks {
			containerPathWebhook.Webhooks[i].ClientConfig.CABundle = caBundle
		}
		all = append(all, containerPathWebhook)
	}

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

	secrets := components.NewCertSecrets(config.GetNamespace(), NAMESPACE)
	if config.VirtTemplateDeploymentEnabled() {
		secrets = append(secrets, components.NewVirtTemplateCertSecrets(config.GetNamespace())...)
	}
	for _, secret := range secrets {
		components.PopulateSecretWithCertificate(secret, caCert, &metav1.Duration{Duration: apply.Duration1d})
		all = append(all, secret)
	}

	userName := fmt.Sprintf("system:serviceaccount:%s:%s", config.GetNamespace(), components.HandlerServiceAccountName)
	all = append(all, components.NewHandlerV1ValidatingAdmissionPolicy(userName), components.NewHandlerV1ValidatingAdmissionPolicyBinding())

	if config.VirtTemplateDeploymentEnabled() {
		resources, err := components.NewVirtTemplateResources(config)
		Expect(err).ToNot(HaveOccurred())
		for i := range resources.ValidatingWebhookConfigurations {
			for j := range resources.ValidatingWebhookConfigurations[i].Webhooks {
				resources.ValidatingWebhookConfigurations[i].Webhooks[j].ClientConfig.CABundle = caBundle
			}
		}
		for i := range resources.APIServices {
			resources.APIServices[i].Spec.CABundle = caBundle
		}
		all = appendVirtTemplateResources(all, resources)
	}

	for _, obj := range all {
		m := obj.(metav1.Object)

		if _, exists := exclusionMap[m.GetName()]; exists {
			continue
		}

		a := m.GetAnnotations()
		if len(a) == 0 {
			a = map[string]string{}
		}

		a[v1.KubeVirtCustomizeComponentAnnotationHash] = c.Hash()
		m.SetAnnotations(a)

		k.addResource(obj, config, kv)
	}
}

func appendVirtTemplateResources(all []runtime.Object, resources *components.VirtTemplateResources) []runtime.Object {
	for _, obj := range resources.CRDs {
		all = append(all, obj)
	}
	for _, obj := range resources.ServiceAccounts {
		all = append(all, obj)
	}
	for _, obj := range resources.Roles {
		all = append(all, obj)
	}
	for _, obj := range resources.ClusterRoles {
		all = append(all, obj)
	}
	for _, obj := range resources.RoleBindings {
		all = append(all, obj)
	}
	for _, obj := range resources.ClusterRoleBindings {
		all = append(all, obj)
	}
	for _, obj := range resources.Services {
		all = append(all, obj)
	}
	for _, obj := range resources.Deployments {
		all = append(all, obj)
	}
	for _, obj := range resources.ValidatingAdmissionPolicies {
		all = append(all, obj)
	}
	for _, obj := range resources.ValidatingAdmissionPolicyBindings {
		all = append(all, obj)
	}
	for _, obj := range resources.ValidatingWebhookConfigurations {
		all = append(all, obj)
	}
	for _, obj := range resources.APIServices {
		all = append(all, obj)
	}
	return all
}

func (k *KubeVirtTestData) addAll(config *util.KubeVirtDeploymentConfig, kv *v1.KubeVirt) {
	k.addAllWithExclusionMap(config, kv, nil)
}

func (k *KubeVirtTestData) addAllButHandler(config *util.KubeVirtDeploymentConfig, kv *v1.KubeVirt) {
	k.addAllWithExclusionMap(config, kv, map[string]bool{"virt-handler": true})
}

func (k *KubeVirtTestData) addVirtHandler(config *util.KubeVirtDeploymentConfig, kv *v1.KubeVirt) {
	config.Namespace = NAMESPACE
	handler := components.NewHandlerDaemonSet(config, "", "", "")

	c, _ := apply.NewCustomizer(kv.Spec.CustomizeComponents)

	if handler.Annotations == nil {
		handler.Annotations = make(map[string]string)
	}
	handler.Annotations[v1.InstallStrategyVersionAnnotation] = config.GetKubeVirtVersion()
	handler.Annotations[v1.InstallStrategyRegistryAnnotation] = config.GetImageRegistry()
	handler.Annotations[v1.InstallStrategyIdentifierAnnotation] = config.GetDeploymentID()
	handler.Annotations[v1.KubeVirtCustomizeComponentAnnotationHash] = c.Hash()
	handler.Annotations[v1.KubeVirtGenerationAnnotation] = strconv.FormatInt(kv.GetGeneration(), 10)

	if handler.Labels == nil {
		handler.Labels = make(map[string]string)
	}
	handler.Labels[v1.ManagedByLabel] = v1.ManagedByLabelOperatorValue
	handler.Labels[v1.AppComponentLabel] = v1.AppComponent
	if config.GetProductVersion() != "" {
		handler.Labels[v1.AppVersionLabel] = config.GetProductVersion()
	}
	if config.GetProductName() != "" {
		handler.Labels[v1.AppPartOfLabel] = config.GetProductName()
	}

	k.addDaemonset(handler, kv)
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
		switch deleted.GetName() {
		case components.KubeVirtCASecretName, components.KubeVirtExportCASecretName, components.KubeVirtBackupCASecretName:
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

func (k *KubeVirtTestData) makeDeploymentsReady(kv *v1.KubeVirt) {
	makeDeploymentReady := func(item interface{}) {
		depl, _ := item.(*appsv1.Deployment)
		deplNew := depl.DeepCopy()
		var replicas int32 = 1
		if depl.Spec.Replicas != nil {
			replicas = *depl.Spec.Replicas
		}
		deplNew.Status.Replicas = replicas
		deplNew.Status.ReadyReplicas = replicas
		k.controller.stores.DeploymentCache.Update(deplNew)
		key, err := kubecontroller.KeyFunc(deplNew)
		Expect(err).To(Not(HaveOccurred()))
		k.mockQueue.Add(key)
	}

	deployments := []string{"/virt-api", "/virt-controller"}
	if exportProxyEnabled(kv) {
		deployments = append(deployments, "/virt-exportproxy")
	}
	if synchronizationControllerEnabled(kv) {
		deployments = append(deployments, "/virt-synchronization-controller")
	}
	if virtTemplateDeploymentEnabled(kv) {
		deployments = append(
			deployments,
			"/"+components.VirtTemplateApiserverDeploymentName,
			"/"+components.VirtTemplateControllerDeploymentName,
		)
	}

	for _, name := range deployments {
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

	k.makePodDisruptionBudgetsReady(kv)
}

func (k *KubeVirtTestData) makePodDisruptionBudgetsReady(kv *v1.KubeVirt) {
	pdbs := []string{"/virt-api-pdb", "/virt-controller-pdb"}
	if exportProxyEnabled(kv) {
		pdbs = append(pdbs, "/virt-exportproxy-pdb")
	}

	for _, pdbname := range pdbs {
		exists := false
		// we need to wait until the pdb exists
		for !exists {
			_, exists, _ = k.controller.stores.PodDisruptionBudgetCache.GetByKey(NAMESPACE + pdbname)
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
			handlerNew.Status.UpdatedNumberScheduled = 1
			k.controller.stores.DaemonSetCache.Update(handlerNew)
			key, err := kubecontroller.KeyFunc(handlerNew)
			Expect(err).To(Not(HaveOccurred()))
			k.mockQueue.Add(key)
		}
	}
}

func (k *KubeVirtTestData) makeHandlerComplete() {
	exists := false
	var obj interface{}
	// we need to wait until the daemonset exists
	for !exists {
		obj, exists, _ = k.controller.stores.DaemonSetCache.GetByKey(NAMESPACE + "/virt-handler")
		if exists {
			handler, _ := obj.(*appsv1.DaemonSet)
			handlerNew := handler.DeepCopy()
			maxUnavailable := intstr.FromInt(1)
			handlerNew.Spec.UpdateStrategy.RollingUpdate = &appsv1.RollingUpdateDaemonSet{
				MaxUnavailable: &maxUnavailable,
			}
			handlerNew.Spec.Template.Spec.Containers[0].Args = append(handlerNew.Spec.Template.Spec.Containers[0].Args, "migration-cn-types")
			k.controller.stores.DaemonSetCache.Update(handlerNew)
			key, err := kubecontroller.KeyFunc(handlerNew)
			Expect(err).To(Not(HaveOccurred()))
			k.mockQueue.Add(key)
		}
	}
}

func (k *KubeVirtTestData) addDummyValidationWebhook() {
	version := fmt.Sprintf("rand-%s", rand.String(10))
	registry := fmt.Sprintf("rand-%s", rand.String(10))
	config := k.getConfig(registry, version)

	validationWebhook := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "virt-operator-tmp-webhook",
		},
	}

	injectMetadata(&validationWebhook.ObjectMeta, config)
	k.addValidatingWebhook(validationWebhook, nil)
}

func (k *KubeVirtTestData) addValidatingWebhook(wh *admissionregistrationv1.ValidatingWebhookConfiguration, kv *v1.KubeVirt) {
	if kv != nil {
		apply.SetGeneration(&kv.Status.Generations, wh)
	}
	k.controller.stores.ValidationWebhookCache.Add(wh)
	key, err := kubecontroller.KeyFunc(wh)
	Expect(err).To(Not(HaveOccurred()))
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) addInstallStrategy(config *util.KubeVirtDeploymentConfig) {
	// install strategy config
	resource, err := install.NewInstallStrategyConfigMap(config, "openshift-monitoring", NAMESPACE)
	Expect(err).ToNot(HaveOccurred())

	resource.Name = fmt.Sprintf("%s-%s", resource.Name, rand.String(10))

	injectMetadata(&resource.ObjectMeta, config)
	k.addConfigMap(resource)
}

func (k *KubeVirtTestData) addPodDisruptionBudgets(config *util.KubeVirtDeploymentConfig, deployments []*appsv1.Deployment, kv *v1.KubeVirt) {
	for _, deployment := range deployments {
		minAvailable := intstr.FromInt32(1)
		if deployment.Spec.Replicas != nil {
			minAvailable = intstr.FromInt32(*deployment.Spec.Replicas - 1)
		}
		if minAvailable.IntValue() < 1 {
			continue
		}
		pdr := &policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: deployment.Namespace,
				Name:      deployment.Name + "-pdb",
				Labels:    deployment.Labels,
			},
			Spec: policyv1.PodDisruptionBudgetSpec{
				MinAvailable: &minAvailable,
				Selector:     deployment.Spec.Selector,
			},
		}
		injectMetadata(&pdr.ObjectMeta, config)
		k.addPodDisruptionBudget(pdr, kv)
	}
}

func (k *KubeVirtTestData) fakeNamespaceModificationEvent() {
	// Add modification event for namespace w/o the labels we need
	k.controller.stores.NamespaceCache.Update(&k8sv1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind: "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: NAMESPACE,
		},
	})
	key, _ := kubecontroller.KeyFunc(NAMESPACE)
	k.mockQueue.Add(key)
}

func (k *KubeVirtTestData) shouldExpectNamespacePatch() {
	k.kubeClient.Fake.PrependReactor("patch", "namespaces", k.genericPatchFunc())
}

func (k *KubeVirtTestData) addPodsWithIndividualConfigs(config *util.KubeVirtDeploymentConfig,
	configController *util.KubeVirtDeploymentConfig,
	configExportProxy *util.KubeVirtDeploymentConfig,
	configHandler *util.KubeVirtDeploymentConfig,
	shouldAddPodDisruptionBudgets bool,
	kv *v1.KubeVirt) {
	// we need at least one active pod for
	// virt-api
	// virt-controller
	// virt-handler
	var deployments []*appsv1.Deployment
	config.Namespace = NAMESPACE
	apiDeployment := components.NewApiServerDeployment(config, "", "", "")

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
	deployments = append(deployments, apiDeployment)

	controller := components.NewControllerDeployment(config, "", "", "")
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
	deployments = append(deployments, controller)

	configHandler.Namespace = NAMESPACE
	handler := components.NewHandlerDaemonSet(configHandler, "", "", "")
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
	boolTrue := true
	pod.OwnerReferences = []metav1.OwnerReference{{Name: handler.Name, Controller: &boolTrue, UID: handler.UID}}
	pod.Name = "virt-handler-xxxx"
	k.addPod(pod)

	if exportProxyEnabled(kv) {
		configExportProxy.Namespace = NAMESPACE
		exportProxy := components.NewExportProxyDeployment(configExportProxy, "", "", "")
		pod = &k8sv1.Pod{
			ObjectMeta: exportProxy.Spec.Template.ObjectMeta,
			Spec:       exportProxy.Spec.Template.Spec,
			Status: k8sv1.PodStatus{
				Phase: k8sv1.PodRunning,
				ContainerStatuses: []k8sv1.ContainerStatus{
					{Ready: true, Name: "somecontainer"},
				},
			},
		}
		pod.Name = "virt-exportproxy-xxxx"
		injectMetadata(&pod.ObjectMeta, configExportProxy)
		k.addPod(pod)
		deployments = append(deployments, exportProxy)
	}

	if virtTemplateDeploymentEnabled(kv) {
		resources, err := components.NewVirtTemplateResources(config)
		Expect(err).ToNot(HaveOccurred())
		for _, deployment := range resources.Deployments {
			pod = &k8sv1.Pod{
				ObjectMeta: deployment.Spec.Template.ObjectMeta,
				Spec:       deployment.Spec.Template.Spec,
				Status: k8sv1.PodStatus{
					Phase: k8sv1.PodRunning,
					ContainerStatuses: []k8sv1.ContainerStatus{
						{Ready: true, Name: "somecontainer"},
					},
				},
			}
			pod.Name = deployment.Name + "-xxxx"
			injectMetadata(&pod.ObjectMeta, config)
			k.addPod(pod)
			deployments = append(deployments, deployment)
		}
	}

	if shouldAddPodDisruptionBudgets {
		k.addPodDisruptionBudgets(config, deployments, kv)
	}
}

func (k *KubeVirtTestData) addPodsWithOptionalPodDisruptionBudgets(config *util.KubeVirtDeploymentConfig, shouldAddPodDisruptionBudgets bool, kv *v1.KubeVirt) {
	k.addPodsWithIndividualConfigs(config, config, config, config, shouldAddPodDisruptionBudgets, kv)
}

func (k *KubeVirtTestData) addPodsAndPodDisruptionBudgets(config *util.KubeVirtDeploymentConfig, kv *v1.KubeVirt) {
	k.addPodsWithOptionalPodDisruptionBudgets(config, true, kv)
}

func (k *KubeVirtTestData) getConfig(registry, version string) *util.KubeVirtDeploymentConfig {
	kv := &v1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: NAMESPACE,
		},
		Spec: v1.KubeVirtSpec{
			ImageRegistry: registry,
			ImageTag:      version,
		},
	}
	enableTemplateFeatureGate(kv)
	enableContainerPathVolumesFeatureGate(kv)
	return util.GetTargetConfigFromKVWithEnvVarManager(kv, k.mockEnvVarManager)
}

var _ = Describe("KubeVirt Operator", func() {

	Context("On valid KubeVirt object", func() {

		It("Should not patch kubevirt namespace when labels are already defined", func() {

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			// Add fake namespace with labels predefined
			err := kvTestData.controller.stores.NamespaceCache.Add(&k8sv1.Namespace{
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
					Phase:              v1.KubeVirtPhaseDeleted,
					ObservedGeneration: pointer.P(int64(1)),
				},
			}
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
			// Add kubevirt deployment and mark everything as ready
			kvTestData.addKubeVirt(kv)
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.shouldExpectCreations()
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.addAll(kvTestData.defaultConfig, kv)
			kvTestData.addPodsAndPodDisruptionBudgets(kvTestData.defaultConfig, kv)
			kvTestData.makeDeploymentsReady(kv)
			kvTestData.makeHandlerReady()
			kvTestData.shouldExpectPatchesAndUpdates(kv)

			// Now when the controller runs, if the namespace will be patched, the test will fail
			// because the patch is not expected here.
			kvTestData.controller.Execute()
		})

		It("should delete install strategy configmap once kubevirt install is deleted", func() {
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
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
			kv.DeletionTimestamp = now()
			util.UpdateConditionsDeleting(kv)

			kvTestData.shouldExpectInstallStrategyDeletion()

			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.shouldExpectKubeVirtFinalizersPatch(1)
			kvTestData.controller.Execute()
			kv = kvTestData.getLatestKubeVirt(kv)
			Expect(kv.ObjectMeta.Finalizers).To(BeEmpty())
		})

		It("should observe custom image tag in status during deploy", func() {
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
					Phase:              v1.KubeVirtPhaseDeployed,
					OperatorVersion:    version.Get().String(),
					ObservedGeneration: pointer.P(int64(1)),
				},
			}
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			customConfig := kvTestData.getConfig(kvTestData.defaultConfig.GetImageRegistry(), "custom.tag")

			kvTestData.fakeNamespaceModificationEvent()
			kvTestData.shouldExpectNamespacePatch()
			kvTestData.shouldExpectPatchesAndUpdates(kv)
			kvTestData.addAll(customConfig, kv)
			// install strategy config
			kvTestData.addInstallStrategy(customConfig)
			kvTestData.addPodsAndPodDisruptionBudgets(customConfig, kv)

			kvTestData.makeDeploymentsReady(kv)
			kvTestData.makeHandlerReady()

			kvTestData.shouldExpectKubeVirtUpdateStatusVersion(1, customConfig)
			kvTestData.shouldExpectInstallStrategyDeletion()
			kvTestData.controller.Execute()
			kv = kvTestData.getLatestKubeVirt(kv)
			shouldExpectHCOConditions(kv, k8sv1.ConditionTrue, k8sv1.ConditionFalse, k8sv1.ConditionFalse)
		})

		It("delete temporary validation webhook once virt-api is deployed", func() {
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
					Phase:              v1.KubeVirtPhaseDeployed,
					OperatorVersion:    version.Get().String(),
					ObservedGeneration: pointer.P(int64(1)),
				},
			}
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
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
			kvTestData.makeDeploymentsReady(kv)
			kvTestData.makeHandlerReady()

			kvTestData.shouldExpectDeletions()
			kvTestData.fakeNamespaceModificationEvent()
			kvTestData.shouldExpectNamespacePatch()
			kvTestData.shouldExpectPatchesAndUpdates(kv)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)

			kvTestData.controller.Execute()
			Expect(kvTestData.totalDeletions).To(Equal(1))
		})

		It("should delete old serviceMonitor on update", func() {

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
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
			kvTestData.defaultConfig.SetTargetDeploymentConfig(kv)
			kvTestData.defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)
			kvTestData.deleteFromCache = false

			// create old servicemonitor (should be deleted)

			err := kvTestData.controller.stores.NamespaceCache.Add(&k8sv1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind: "Namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "openshift-monitoring",
					Labels: map[string]string{
						"openshift.io/cluster-monitoring": "true",
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			err = kvTestData.controller.stores.ServiceMonitorCache.Add(&promv1.ServiceMonitor{
				TypeMeta: metav1.TypeMeta{
					Kind: "ServiceMonitor",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "openshift-monitoring",
					Labels: map[string]string{
						v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
					},
				},
			})

			Expect(err).NotTo(HaveOccurred())

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.addAll(kvTestData.defaultConfig, kv)
			kvTestData.addPodsAndPodDisruptionBudgets(kvTestData.defaultConfig, kv)
			kvTestData.makeDeploymentsReady(kv)
			kvTestData.makeHandlerReady()

			kvTestData.shouldExpectDeletions()
			kvTestData.fakeNamespaceModificationEvent()
			kvTestData.shouldExpectNamespacePatch()
			kvTestData.shouldExpectPatchesAndUpdates(kv)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)

			kvTestData.controller.Execute()
			Expect(kvTestData.totalDeletions).To(Equal(1))
		})

		It("should do nothing if KubeVirt object is deployed", func() {
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
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
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
			kvTestData.makeDeploymentsReady(kv)
			kvTestData.makeHandlerReady()

			kvTestData.fakeNamespaceModificationEvent()
			kvTestData.shouldExpectNamespacePatch()
			kvTestData.shouldExpectPatchesAndUpdates(kv)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)

			kvTestData.controller.Execute()
		})

		It("should update KubeVirt object if generation IDs do not match", func() {
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
					Phase:              v1.KubeVirtPhaseDeployed,
					OperatorVersion:    version.Get().String(),
					ObservedGeneration: pointer.P(int64(1)),
				},
			}
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
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
			kvTestData.makeDeploymentsReady(kv)
			kvTestData.makeHandlerReady()
			kvTestData.makeHandlerComplete()

			kvTestData.fakeNamespaceModificationEvent()
			kvTestData.shouldExpectNamespacePatch()
			kvTestData.shouldExpectPatchesAndUpdates(kv)
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
			Expect(kvTestData.resourceChanges["deployments"][Patched]).To(Equal(kvTestData.resourceChanges["deployments"][Added]))
			// Expecting to drop certificate
			Expect(kvTestData.resourceChanges["daemonsets"][Patched]).To(Equal(kvTestData.resourceChanges["daemonsets"][Added]))
		})

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
					Phase:              v1.KubeVirtPhaseDeployed,
					OperatorVersion:    version.Get().String(),
					ObservedGeneration: pointer.P(int64(1)),
				},
			}
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
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

			kvTestData.makeDeploymentsReady(kv)
			kvTestData.makeHandlerReady()

			kvTestData.shouldExpectDeletions()
			kvTestData.fakeNamespaceModificationEvent()
			kvTestData.shouldExpectNamespacePatch()
			kvTestData.shouldExpectPatchesAndUpdates(kv)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)

			kvTestData.controller.Execute()
			Expect(kvTestData.totalDeletions).To(Equal(numResources))
		})

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
			enableTemplateFeatureGate(kv1)

			kv2 := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-install-2",
					Namespace: NAMESPACE,
					UID:       "123123123",
				},
				Status: v1.KubeVirtStatus{},
			}
			enableTemplateFeatureGate(kv2)

			kubecontroller.SetLatestApiVersionAnnotation(kv1)
			util.UpdateConditionsCreated(kv1)
			util.UpdateConditionsAvailable(kv1)
			kvTestData.addKubeVirt(kv1)
			kubecontroller.SetLatestApiVersionAnnotation(kv2)
			kvTestData.addKubeVirt(kv2)

			kvTestData.shouldExpectKubeVirtUpdateStatusFailureCondition(util.ConditionReasonDeploymentFailedExisting)

			kvTestData.controller.execute(fmt.Sprintf("%s/%s", kv2.Namespace, kv2.Name))

		})

		It("should generate install strategy creation job for update version", func() {
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
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
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

		})

		It("should create an install strategy creation job with passthrough env vars, if provided in config", func() {
			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			config := kvTestData.getConfig("registry", "v1.1.1")
			envKey := rand.String(10)
			envVal := rand.String(10)
			config.PassthroughEnvVars = map[string]string{envKey: envVal}
			job, err := kvTestData.controller.generateInstallStrategyJob(&v1.ComponentConfig{}, config)

			Expect(err).ToNot(HaveOccurred())
			Expect(job.Spec.Template.Spec.Containers[0].Env).To(ContainElement(k8sv1.EnvVar{Name: envKey, Value: envVal}))
		})

		It("should use custom virt-operator image if provided", func() {
			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			const expectedImage = "myimage123:mytag456"
			err := kvTestData.mockEnvVarManager.Setenv(util.VirtOperatorImageEnvName, expectedImage)
			Expect(err).ToNot(HaveOccurred())
			config := kvTestData.getConfig("", "")

			job, err := kvTestData.controller.generateInstallStrategyJob(&v1.ComponentConfig{}, config)
			Expect(err).ToNot(HaveOccurred())

			Expect(job.Spec.Template.Spec.Containers).ToNot(BeEmpty())
			container := job.Spec.Template.Spec.Containers[0]

			actualImage := ""
			for _, envVar := range container.Env {
				if envVar.Name == util.VirtOperatorImageEnvName {
					actualImage = envVar.Value
				}
			}

			Expect(actualImage).To(Equal(expectedImage))
		})

		It("should create an api server deployment with passthrough env vars, if provided in config", func() {
			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			config := kvTestData.getConfig("registry", "v1.1.1")
			envKey := rand.String(10)
			envVal := rand.String(10)
			config.PassthroughEnvVars = map[string]string{envKey: envVal}
			config.Namespace = NAMESPACE

			apiDeployment := components.NewApiServerDeployment(config, "", "", "")
			Expect(apiDeployment.Spec.Template.Spec.Containers[0].Env).To(ContainElement(k8sv1.EnvVar{Name: envKey, Value: envVal}))
		})

		It("should create a controller deployment with passthrough env vars, if provided in config", func() {
			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			config := kvTestData.getConfig("registry", "v1.1.1")
			envKey := rand.String(10)
			envVal := rand.String(10)
			config.PassthroughEnvVars = map[string]string{envKey: envVal}
			config.Namespace = NAMESPACE

			controllerDeployment := components.NewControllerDeployment(config, "", "", "")
			Expect(controllerDeployment.Spec.Template.Spec.Containers[0].Env).To(ContainElement(k8sv1.EnvVar{Name: envKey, Value: envVal}))
		})

		It("should create a handler daemonset with passthrough env vars, if provided in config", func() {
			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			config := kvTestData.getConfig("registry", "v1.1.1")
			envKey := rand.String(10)
			envVal := rand.String(10)
			config.PassthroughEnvVars = map[string]string{envKey: envVal}
			config.Namespace = NAMESPACE

			handlerDaemonset := components.NewHandlerDaemonSet(config, "", "", "")
			Expect(handlerDaemonset.Spec.Template.Spec.Containers[0].Env).To(ContainElement(k8sv1.EnvVar{Name: envKey, Value: envVal}))
		})

		It("should generate install strategy creation job if no install strategy exists", func() {
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
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.shouldExpectJobCreation()
			kvTestData.controller.Execute()

		})

		It("should apply the infra placement config on the install job", func() {
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

			affinity := &k8sv1.Affinity{
				NodeAffinity: &k8sv1.NodeAffinity{RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
					NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
						{MatchExpressions: []k8sv1.NodeSelectorRequirement{
							{
								Key:      "test",
								Operator: "In",
								Values:   []string{"something"},
							},
						}},
					},
				},
				},
			}
			job, err := kvTestData.controller.generateInstallStrategyJob(&v1.ComponentConfig{
				NodePlacement: &v1.NodePlacement{Affinity: affinity},
			}, util.GetTargetConfigFromKV(kv))
			Expect(err).ToNot(HaveOccurred())

			Expect(job.Spec.Template.Spec.Affinity).To(Equal(affinity))

		})

		It("should label install strategy creation job", func() {
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

			job, err := kvTestData.controller.generateInstallStrategyJob(nil, util.GetTargetConfigFromKV(kv))
			Expect(err).ToNot(HaveOccurred())

			Expect(job.Spec.Template.ObjectMeta.Labels).Should(HaveKeyWithValue(v1.AppLabel, virtOperatorJobAppLabel))
		})

		It("should delete install strategy creation job if job has failed", func() {
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
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)

			job, err := kvTestData.controller.generateInstallStrategyJob(&v1.ComponentConfig{}, util.GetTargetConfigFromKV(kv))
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

		})

		It("should not delete completed install strategy creation job if job has failed less that 10 seconds ago", func() {
			defer GinkgoRecover()

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
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)

			job, err := kvTestData.controller.generateInstallStrategyJob(kv.Spec.Infra, util.GetTargetConfigFromKV(kv))
			Expect(err).ToNot(HaveOccurred())

			job.Status.CompletionTime = now()

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategyJob(job)

			kvTestData.shouldExpectKubeVirtUpdateStatus(1)

			kvTestData.controller.Execute()

		})

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
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)

			job, err := kvTestData.controller.generateInstallStrategyJob(kv.Spec.Infra, util.GetTargetConfigFromKV(kv))
			Expect(err).ToNot(HaveOccurred())

			job.Status.CompletionTime = now()
			kvTestData.addInstallStrategyJob(job)

			// ensure completed jobs are garbage collected once install strategy
			// is loaded
			kvTestData.deleteFromCache = false
			kvTestData.shouldExpectJobDeletion()
			kvTestData.shouldExpectKubeVirtFinalizersPatch(1)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.shouldExpectCreations()

			kvTestData.controller.Execute()

			kv = kvTestData.getLatestKubeVirt(kv)
			Expect(kv.Status.Phase).To(Equal(v1.KubeVirtPhaseDeploying))
			Expect(kv.Status.Conditions).To(HaveLen(3))
			Expect(kv.ObjectMeta.Finalizers).To(HaveLen(1))
			shouldExpectHCOConditions(kv, k8sv1.ConditionFalse, k8sv1.ConditionTrue, k8sv1.ConditionFalse)

			// 8 in total are yet missing at this point
			// because waiting on controller, controller's PDB and virt-handler daemonset until API server deploys successfully
			// also exportProxy + PDB + route
			// also virt-template-apiserver and virt-template-controller
			expectedUncreatedResources := 8

			Expect(kvTestData.totalAdds).To(Equal(resourceCount - expectedUncreatedResources + expectedTemporaryResources + externalCAConfigMapCount))

			Expect(kvTestData.controller.stores.ServiceAccountCache.List()).To(HaveLen(7))
			Expect(kvTestData.controller.stores.ClusterRoleCache.List()).To(HaveLen(21))
			Expect(kvTestData.controller.stores.ClusterRoleBindingCache.List()).To(HaveLen(12))
			Expect(kvTestData.controller.stores.RoleCache.List()).To(HaveLen(7))
			Expect(kvTestData.controller.stores.RoleBindingCache.List()).To(HaveLen(8))
			Expect(kvTestData.controller.stores.OperatorCrdCache.List()).To(HaveLen(numCRDs))
			Expect(kvTestData.controller.stores.ServiceCache.List()).To(HaveLen(7))
			Expect(kvTestData.controller.stores.DeploymentCache.List()).To(HaveLen(1))
			Expect(kvTestData.controller.stores.DaemonSetCache.List()).To(BeEmpty())
			Expect(kvTestData.controller.stores.ValidationWebhookCache.List()).To(HaveLen(4))
			Expect(kvTestData.controller.stores.PodDisruptionBudgetCache.List()).To(HaveLen(1))
			Expect(kvTestData.controller.stores.SCCCache.List()).To(HaveLen(3))
			Expect(kvTestData.controller.stores.ServiceMonitorCache.List()).To(HaveLen(1))
			Expect(kvTestData.controller.stores.PrometheusRuleCache.List()).To(HaveLen(1))
			Expect(kvTestData.controller.stores.MutatingWebhookCache.List()).To(HaveLen(2))
			Expect(kvTestData.controller.stores.APIServiceCache.List()).To(HaveLen(3))

			Expect(kvTestData.resourceChanges["poddisruptionbudgets"][Added]).To(Equal(1))

		})

		It("should pause rollback until api server is rolled over.", func() {
			defer GinkgoRecover()

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			rollbackConfig := kvTestData.getConfig("otherregistry", "9.9.7")

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
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
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

			kvTestData.makeDeploymentsReady(kv)
			kvTestData.makeHandlerReady()

			kvTestData.addToCache = false
			kvTestData.shouldExpectRbacBackupCreations()
			kvTestData.shouldExpectPatchesAndUpdates(kv)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.shouldExpectInstallStrategyDeletion()

			kvTestData.controller.Execute()

			kv = kvTestData.getLatestKubeVirt(kv)
			// conditions should reflect an ongoing update
			shouldExpectHCOConditions(kv, k8sv1.ConditionTrue, k8sv1.ConditionTrue, k8sv1.ConditionTrue)

			// on rollback or create, api server must be online first before controllers and daemonset.
			// On rollback this prevents someone from posting invalid specs to
			// the cluster from newer versions when an older version is being deployed.
			// On create this prevents invalid specs from entering the cluster
			// while controllers are available to process them.

			// 9 because 2 for virt-controller service and deployment,
			// 1 because of the pdb of virt-controller
			// and another 1 because of the namespace was not patched yet.
			// also virt-exportproxy and pdb and route
			// also virt-template-apiserver and virt-template-controller
			Expect(kvTestData.totalPatches).To(Equal(patchCount - 9))
			Expect(kvTestData.totalUpdates).To(Equal(updateCount))

			Expect(kvTestData.resourceChanges["poddisruptionbudgets"][Patched]).To(Equal(1))
		})

		It("should pause update after daemonsets are rolled over", func() {
			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			err := kvTestData.mockEnvVarManager.Unsetenv(util.OldOperatorImageEnvName)
			Expect(err).NotTo(HaveOccurred())
			updatedConfig := kvTestData.getConfig("otherregistry", "9.9.10")

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
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
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

			kvTestData.makeDeploymentsReady(kv)
			kvTestData.makeHandlerReady()

			kvTestData.addToCache = false
			kvTestData.shouldExpectRbacBackupCreations()
			kvTestData.shouldExpectPatchesAndUpdates(kv)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.shouldExpectInstallStrategyDeletion()

			kvTestData.controller.Execute()

			kv = kvTestData.getLatestKubeVirt(kv)
			// conditions should reflect an ongoing update
			shouldExpectHCOConditions(kv, k8sv1.ConditionTrue, k8sv1.ConditionTrue, k8sv1.ConditionTrue)

			Expect(kvTestData.totalUpdates).To(Equal(updateCount))

			// daemonset, controller and apiserver pods are updated in this order.
			// this prevents the new API from coming online until the controllers can manage it.
			// The PDBs will prevent updated pods from getting "ready", so update should pause after
			//   daemonsets and before controller and namespace

			// 10 because virt-controller, virt-api, PDBs and the namespace are not patched
			// also virt-exportproxy and pdb and route
			// also virt-template-apiserver and virt-template-controller
			Expect(kvTestData.totalPatches).To(Equal(patchCount - 10))

			// Make sure the 5 unpatched are as expected
			Expect(kvTestData.resourceChanges["deployments"][Patched]).To(Equal(0))          // virt-controller and virt-api unpatched
			Expect(kvTestData.resourceChanges["poddisruptionbudgets"][Patched]).To(Equal(0)) // PDBs unpatched
			Expect(kvTestData.resourceChanges["namespace"][Patched]).To(Equal(0))            // namespace unpatched
		})

		It("should pause update after controllers are rolled over", func() {
			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			updatedConfig := kvTestData.getConfig("otherregistry", "9.9.10")

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
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
			kvTestData.defaultConfig.SetTargetDeploymentConfig(kv)
			kvTestData.defaultConfig.SetObservedDeploymentConfig(kv)
			util.UpdateConditionsCreated(kv)
			util.UpdateConditionsAvailable(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.addInstallStrategy(updatedConfig)

			kvTestData.addAllButHandler(kvTestData.defaultConfig, kv)
			// Create virt-api and virt-controller under kvTestData.defaultConfig,
			// but use updatedConfig for virt-handler (hack) to avoid pausing after daemonsets

			// add already updated virt-handler
			kvTestData.addVirtHandler(updatedConfig, kv)

			kvTestData.addPodsWithIndividualConfigs(kvTestData.defaultConfig, kvTestData.defaultConfig, kvTestData.defaultConfig, updatedConfig, true, kv)

			kvTestData.makeDeploymentsReady(kv)
			kvTestData.makeHandlerReady()

			kvTestData.addToCache = false
			kvTestData.shouldExpectRbacBackupCreations()
			kvTestData.shouldExpectPatchesAndUpdates(kv)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.shouldExpectInstallStrategyDeletion()

			kvTestData.controller.Execute()

			kv = kvTestData.getLatestKubeVirt(kv)
			// conditions should reflect an ongoing update
			shouldExpectHCOConditions(kv, k8sv1.ConditionTrue, k8sv1.ConditionTrue, k8sv1.ConditionTrue)

			Expect(kvTestData.totalUpdates).To(Equal(updateCount))

			// The update was hacked to avoid pausing after rolling out the daemonsets (virt-handler)
			// That will allow both daemonset and controller pods to get patched before the pause.

			// 9 because virt-handler, virt-api, PDB and the namespace should not be patched
			// also virt-exportproxy and pdb and route
			// also virt-template-apiserver and virt-template-controller
			Expect(kvTestData.totalPatches).To(Equal(patchCount - 9))

			// Make sure the 4 unpatched are as expected
			Expect(kvTestData.resourceChanges["deployments"][Patched]).To(Equal(1))          // virt-operator patched, virt-api unpatched
			Expect(kvTestData.resourceChanges["poddisruptionbudgets"][Patched]).To(Equal(1)) // 1 of 2 PDBs patched
			Expect(kvTestData.resourceChanges["namespace"][Patched]).To(Equal(0))            // namespace unpatched
			Expect(kvTestData.resourceChanges["daemonsets"][Patched]).To(Equal(0))           // namespace unpatched
		})

		Context("virt-api replica count", func() {
			var kvTestData KubeVirtTestData
			const (
				CustomizedReplicas              int32 = 4
				numOfUnschedulableNodes               = 1000
				numOfSchedulableNodes                 = 1000
				expectedReplicasForLargeCluster int32 = 100
			)

			BeforeEach(func() {
				kvTestData = KubeVirtTestData{}
				kvTestData.BeforeTest()
				DeferCleanup(kvTestData.AfterTest)

				var testNodes []k8sv1.Node

				totalNodes := numOfSchedulableNodes + numOfUnschedulableNodes
				for i := range totalNodes {
					node := k8sv1.Node{
						ObjectMeta: metav1.ObjectMeta{
							Name: fmt.Sprintf("testnode-%d", i),
						},
					}
					if i < numOfSchedulableNodes {
						node.Labels = map[string]string{
							v1.NodeSchedulable: "true",
						}
					}
					testNodes = append(testNodes, node)
				}

				kvTestData.kubeClient.Fake.PrependReactor("list", "nodes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					return true, &k8sv1.NodeList{Items: testNodes}, nil
				})
			})

			It("should apply the replica count from CustomizeComponents patch", func() {
				patchSet, err := patch.New(patch.WithReplace("/spec/replicas", CustomizedReplicas)).GeneratePayload()
				Expect(err).ToNot(HaveOccurred())

				kv := &v1.KubeVirt{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-install",
						Namespace: NAMESPACE,
					},
					Spec: v1.KubeVirtSpec{
						CustomizeComponents: v1.CustomizeComponents{
							Patches: []v1.CustomizeComponentsPatch{
								{
									ResourceName: components.VirtAPIName,
									ResourceType: "Deployment",
									Type:         v1.JSONPatchType,
									Patch:        string(patchSet),
								},
							},
						},
					},
				}
				enableTemplateFeatureGate(kv)
				enableContainerPathVolumesFeatureGate(kv)

				kubecontroller.SetLatestApiVersionAnnotation(kv)
				kvTestData.addKubeVirt(kv)
				kvTestData.addInstallStrategy(kvTestData.defaultConfig)

				kvTestData.defaultConfig.Namespace = NAMESPACE
				apiDeployment := components.NewApiServerDeployment(kvTestData.defaultConfig, "", "", "")
				kvTestData.addDeployment(apiDeployment, kv)

				kvTestData.kvInterface.EXPECT().
					Patch(
						context.Background(),
						kv.Name,
						types.JSONPatchType,
						gomock.Any(),
						metav1.PatchOptions{},
						gomock.Any(),
					)

				kvTestData.kubeClient.Fake.PrependReactor("patch", "deployments", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					patchAction, ok := action.(testing.PatchAction)
					Expect(ok).To(BeTrue())
					Expect(patchAction.GetName()).To(Equal(components.VirtAPIName))
					Expect(patchAction.GetPatchType()).To(Equal(types.JSONPatchType))

					var patches []map[string]interface{}
					err = json.Unmarshal(patchAction.GetPatch(), &patches)
					Expect(err).ToNot(HaveOccurred())

					var replicas int32
					for _, patch := range patches {
						if patch["op"] == "replace" && patch["path"] == "/spec" {
							specData, ok := patch["value"].(map[string]interface{})
							Expect(ok).To(BeTrue(), "Expected /spec patch value to be a map")

							replicasValue, exists := specData["replicas"]
							Expect(exists).To(BeTrue(), "Expected 'replicas' field in /spec patch")

							replicas = int32(replicasValue.(float64))
							break
						}
					}

					patchedDeployment := apiDeployment.DeepCopy()
					patchedDeployment.Spec.Replicas = pointer.P(replicas)

					Expect(*patchedDeployment.Spec.Replicas).To(Equal(CustomizedReplicas))
					return true, patchedDeployment, nil
				})

				kvTestData.shouldExpectKubeVirtUpdateStatus(1)
				kvTestData.shouldExpectCreations()

				kvTestData.controller.Execute()

				Expect(kvtesting.FilterActions(&kvTestData.kubeClient.Fake, "patch", "deployments")).To(HaveLen(1))
			})

			It("should determine replicas based on the number of nodes when CustomizeComponents is not used", func() {
				kv := &v1.KubeVirt{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-install",
						Namespace: NAMESPACE,
					},
				}
				enableTemplateFeatureGate(kv)
				enableContainerPathVolumesFeatureGate(kv)

				kubecontroller.SetLatestApiVersionAnnotation(kv)
				kvTestData.addKubeVirt(kv)
				kvTestData.addInstallStrategy(kvTestData.defaultConfig)

				kvTestData.defaultConfig.Namespace = NAMESPACE
				apiDeployment := components.NewApiServerDeployment(kvTestData.defaultConfig, "", "", "")
				kvTestData.addDeployment(apiDeployment, kv)

				kvTestData.kvInterface.EXPECT().
					Patch(
						context.Background(),
						kv.Name,
						types.JSONPatchType,
						gomock.Any(),
						metav1.PatchOptions{},
						gomock.Any(),
					)

				kvTestData.kubeClient.Fake.PrependReactor("patch", "deployments", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					patchAction, ok := action.(testing.PatchAction)
					Expect(ok).To(BeTrue())
					Expect(patchAction.GetName()).To(Equal(components.VirtAPIName))
					Expect(patchAction.GetPatchType()).To(Equal(types.JSONPatchType))

					var patches []map[string]interface{}
					err = json.Unmarshal(patchAction.GetPatch(), &patches)
					Expect(err).ToNot(HaveOccurred())

					var replicas int32
					for _, patch := range patches {
						if patch["op"] == "replace" && patch["path"] == "/spec" {
							specData, ok := patch["value"].(map[string]interface{})
							Expect(ok).To(BeTrue(), "Expected /spec patch value to be a map")

							replicasValue, exists := specData["replicas"]
							Expect(exists).To(BeTrue(), "Expected 'replicas' field in /spec patch")

							replicas = int32(replicasValue.(float64))
							break
						}
					}
					Expect(replicas).To(BeEquivalentTo(expectedReplicasForLargeCluster))
					return true, nil, nil
				})

				kvTestData.shouldExpectKubeVirtUpdateStatus(1)
				kvTestData.shouldExpectCreations()

				kvTestData.controller.Execute()

				Expect(kvtesting.FilterActions(&kvTestData.kubeClient.Fake, "patch", "deployments")).To(HaveLen(1))
			})
		})

		Context("product related labels of kubevirt install", func() {
			var kvTestData KubeVirtTestData

			BeforeEach(func() {
				kvTestData = KubeVirtTestData{}
				kvTestData.BeforeTest()
				DeferCleanup(kvTestData.AfterTest)
			})

			It("should be applied to kubevirt resources", func() {
				const (
					productName      = "kubevirt-test"
					productVersion   = "0.0.0"
					productComponent = "kubevirt-component"
				)

				kv := &v1.KubeVirt{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-install",
						Namespace:  NAMESPACE,
						Finalizers: []string{util.KubeVirtFinalizer},
					},
					Spec: v1.KubeVirtSpec{
						ImageRegistry:    kvTestData.defaultConfig.GetImageRegistry(),
						ProductName:      productName,
						ProductVersion:   productVersion,
						ProductComponent: productComponent,
					},
					Status: v1.KubeVirtStatus{
						Phase:           v1.KubeVirtPhaseDeployed,
						OperatorVersion: version.Get().String(),
					},
				}
				enableTemplateFeatureGate(kv)
				enableContainerPathVolumesFeatureGate(kv)
				customConfig := util.GetTargetConfigFromKVWithEnvVarManager(kv, kvTestData.mockEnvVarManager)

				kubecontroller.SetLatestApiVersionAnnotation(kv)
				kvTestData.addKubeVirt(kv)
				kvTestData.addInstallStrategy(customConfig)

				customConfig.Namespace = NAMESPACE
				apiDeployment := components.NewApiServerDeployment(customConfig, "", "", "")
				controllerDeployment := components.NewControllerDeployment(customConfig, "", "", "")
				handlerDaemonset := components.NewHandlerDaemonSet(customConfig, "", "", "")
				// omitempty ignores the field's zero value resulting in the json patch test op breaking
				apiDeployment.ObjectMeta.Generation = 123
				controllerDeployment.ObjectMeta.Generation = 123
				handlerDaemonset.ObjectMeta.Generation = 123

				virtTemplateResources, err := components.NewVirtTemplateResources(customConfig)
				Expect(err).ToNot(HaveOccurred())
				for i := range virtTemplateResources.Deployments {
					virtTemplateResources.Deployments[i].ObjectMeta.Generation = 123
					kvTestData.addDeployment(virtTemplateResources.Deployments[i], kv)
				}

				kvTestData.addDeployment(apiDeployment, kv)
				kvTestData.addDeployment(controllerDeployment, kv)
				kvTestData.addDaemonset(handlerDaemonset, kv)
				kvTestData.addPodsAndPodDisruptionBudgets(customConfig, kv)
				kvTestData.makeDeploymentsReady(kv)
				kvTestData.makeHandlerReady()

				var apiDeploy, ctrlDeploy, tplApiDeploy, tplCtrlDeploy *appsv1.Deployment
				kvTestData.deploymentPatchReactionFunc = func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					deploy := &appsv1.Deployment{}
					a := action.(testing.PatchActionImpl)
					patch, err := jsonpatch.DecodePatch(a.Patch)
					Expect(err).ToNot(HaveOccurred())

					o, exists, err := kvTestData.controller.stores.DeploymentCache.GetByKey(fmt.Sprintf("%s/%s", NAMESPACE, a.Name))
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeTrue())
					existing := o.(*appsv1.Deployment)
					existingDeploymentBytes, err := json.Marshal(existing)
					Expect(err).ToNot(HaveOccurred())

					targetDeploymentBytes, err := patch.Apply(existingDeploymentBytes)
					Expect(err).ToNot(HaveOccurred())

					Expect(json.Unmarshal(targetDeploymentBytes, deploy)).To(Succeed())

					switch a.Name {
					case "virt-api":
						apiDeploy = deploy.DeepCopy()
					case "virt-controller":
						ctrlDeploy = deploy.DeepCopy()
					case components.VirtTemplateApiserverDeploymentName:
						tplApiDeploy = deploy.DeepCopy()
					case components.VirtTemplateControllerDeploymentName:
						tplCtrlDeploy = deploy.DeepCopy()
					}

					return true, deploy, nil
				}

				ds := &appsv1.DaemonSet{}
				kvTestData.daemonSetPatchReactionFunc = func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					a := action.(testing.PatchActionImpl)
					patch, err := jsonpatch.DecodePatch(a.Patch)
					Expect(err).ToNot(HaveOccurred())

					o, exists, err := kvTestData.controller.stores.DaemonSetCache.GetByKey(fmt.Sprintf("%s/%s", NAMESPACE, a.Name))
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeTrue())
					existing := o.(*appsv1.DaemonSet)
					existingDeploymentBytes, err := json.Marshal(existing)
					Expect(err).ToNot(HaveOccurred())

					targetDeploymentBytes, err := patch.Apply(existingDeploymentBytes)
					Expect(err).ToNot(HaveOccurred())

					Expect(json.Unmarshal(targetDeploymentBytes, ds)).To(Succeed())

					return true, ds, nil
				}

				kvTestData.shouldExpectPatchesAndUpdates(kv)
				kvTestData.shouldExpectKubeVirtUpdateStatus(1)
				kvTestData.shouldExpectCreations()

				kvTestData.controller.Execute()

				Expect(kvtesting.FilterActions(&kvTestData.kubeClient.Fake, "patch", "deployments")).To(HaveLen(4))
				Expect(kvtesting.FilterActions(&kvTestData.kubeClient.Fake, "patch", "daemonsets")).To(HaveLen(1))

				for _, meta := range []metav1.Object{
					apiDeploy, &apiDeploy.Spec.Template,
					ctrlDeploy, &ctrlDeploy.Spec.Template,
					tplApiDeploy, &tplApiDeploy.Spec.Template,
					tplCtrlDeploy, &tplCtrlDeploy.Spec.Template,
					ds, &ds.Spec.Template,
				} {
					// Labels should be on both the pod/workload controller resource
					Expect(meta.GetLabels()[v1.AppPartOfLabel]).To(Equal(kv.Spec.ProductName))
					Expect(meta.GetLabels()[v1.AppVersionLabel]).To(Equal(kv.Spec.ProductVersion))
					Expect(meta.GetLabels()[v1.AppComponentLabel]).To(Equal(kv.Spec.ProductComponent))
				}
			})
		})

		DescribeTable("should update kubevirt resources when Operator version changes if no imageTag and imageRegistry is explicitly set.", func(withExport bool, patchCount, resourceCount, numPDBs int) {
			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kvTestData.mockEnvVarManager.Setenv(util.OldOperatorImageEnvName, fmt.Sprintf("%s/virt-operator:%s", "otherregistry", "1.1.1"))
			updatedConfig := kvTestData.getConfig("", "")

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
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
			if withExport {
				enableExportFeatureGate(kv)
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
			// also skip virt-handler as it takes more than 1 sync-loop execution
			// to perform a canary-upgrade
			kvTestData.addVirtHandler(updatedConfig, kv)
			kvTestData.addPodsWithOptionalPodDisruptionBudgets(updatedConfig, false, kv)

			kvTestData.makeDeploymentsReady(kv)
			kvTestData.makeHandlerReady()

			kvTestData.shouldExpectPatchesAndUpdates(kv)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.fakeNamespaceModificationEvent()
			kvTestData.shouldExpectNamespacePatch()
			kvTestData.shouldExpectInstallStrategyDeletion()

			kvTestData.controller.Execute()

			kv = kvTestData.getLatestKubeVirt(kv)
			// conditions should reflect a successful update
			shouldExpectHCOConditions(kv, k8sv1.ConditionTrue, k8sv1.ConditionFalse, k8sv1.ConditionFalse)

			// 1 because of virt-handler
			Expect(kvTestData.totalPatches).To(Equal(patchCount))
			Expect(kvTestData.totalUpdates).To(Equal(updateCount))

			// ensure every resource is either patched or updated
			// + 1 is for the namespace patch which we don't consider as a resource we own.
			// - 1 is for virt-handler which we did not patch.
			Expect(kvTestData.totalUpdates + kvTestData.totalPatches).To(Equal(resourceCount))

			Expect(kvTestData.resourceChanges["poddisruptionbudgets"][Patched]).To(Equal(numPDBs))

		},
			// -1 for virt-handler which is already updated
			// -3 for virt-exportproxy
			Entry("without export", false, patchCount-1-3, resourceCount-3, 2),
			Entry("with export", true, patchCount-1, resourceCount, 3),
		)

		DescribeTable("should update resources when changing KubeVirt version.", func(withExport bool, patchCount, resourceCount int) {
			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			updatedConfig := kvTestData.getConfig("otherregistry", "1.1.1")

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
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
			if withExport {
				enableExportFeatureGate(kv)
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

			kvTestData.addAllButHandler(kvTestData.defaultConfig, kv)
			kvTestData.addVirtHandler(updatedConfig, kv)
			kvTestData.addPodsAndPodDisruptionBudgets(kvTestData.defaultConfig, kv)

			// pods for the new version are added so this test won't
			// wait for daemonsets to rollover before updating/patching
			// all resources.
			kvTestData.addPodsWithOptionalPodDisruptionBudgets(updatedConfig, false, kv)

			kvTestData.makeDeploymentsReady(kv)
			kvTestData.makeHandlerReady()

			kvTestData.shouldExpectPatchesAndUpdates(kv)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.fakeNamespaceModificationEvent()
			kvTestData.shouldExpectNamespacePatch()
			kvTestData.shouldExpectInstallStrategyDeletion()

			kvTestData.controller.Execute()

			kv = kvTestData.getLatestKubeVirt(kv)
			// conditions should reflect a successful update
			shouldExpectHCOConditions(kv, k8sv1.ConditionTrue, k8sv1.ConditionFalse, k8sv1.ConditionFalse)

			if withExport {
				o, exists, err := kvTestData.controller.stores.DeploymentCache.GetByKey(fmt.Sprintf("%s/%s", NAMESPACE, "virt-exportproxy"))
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeTrue())
				proxy := o.(*appsv1.Deployment)
				Expect(proxy.Spec.Template.Annotations["openshift.io/required-scc"]).To(Equal("restricted-v2"))
			}

			Expect(kvTestData.totalPatches).To(Equal(patchCount))
			Expect(kvTestData.totalUpdates).To(Equal(updateCount))

			// ensure every resource is either patched or updated
			// + 1 is for the namespace patch which we don't consider as a resource we own.
			// - 1 is for virt-handler.
			Expect(kvTestData.totalUpdates + kvTestData.totalPatches).To(Equal(resourceCount))

		},
			// -1 for virt-handler which is already updated
			// -3 for virt-exportproxy
			Entry("without export", false, patchCount-1-3, resourceCount-3),
			Entry("with export", true, patchCount-1, resourceCount),
		)

		DescribeTable("should patch poddisruptionbudgets when changing KubeVirt version.", func(withExport bool, numPDBs int) {
			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			updatedConfig := kvTestData.getConfig("otherregistry", "1.1.1")

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
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
			if withExport {
				enableExportFeatureGate(kv)
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

			kvTestData.makeDeploymentsReady(kv)
			kvTestData.makeHandlerReady()

			kvTestData.shouldExpectPatchesAndUpdates(kv)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.shouldExpectInstallStrategyDeletion()

			kvTestData.controller.Execute()

			Expect(kvTestData.resourceChanges["poddisruptionbudgets"][Patched]).To(Equal(numPDBs))

		},
			Entry("without export", false, 2),
			Entry("with export", true, 3),
		)

		DescribeTable("should remove resources on deletion", func(withExport bool, resourceCount int) {

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-install",
					Namespace: NAMESPACE,
				},
			}
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
			kv.DeletionTimestamp = now()
			if withExport {
				enableExportFeatureGate(kv)
			}
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)

			// create all resources which should be deleted
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.addAll(kvTestData.defaultConfig, kv)

			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.shouldExpectDeletions()

			// Due to finalizers being added to CRDs during deletion
			kvTestData.extClient.Fake.PrependReactor("patch", "customresourcedefinitions", kvTestData.crdPatchFunc())

			kvTestData.shouldExpectInstallStrategyDeletion()

			kvTestData.controller.Execute()

			// Note: in real life during the first execution loop very probably only CRDs are deleted,
			// because that takes some time (see the check that the crd store is empty before going on with deletions)
			// But in this test the deletion succeeds immediately, so everything is deleted on first try
			Expect(kvTestData.totalDeletions).To(Equal(resourceCount))

			kv = kvTestData.getLatestKubeVirt(kv)
			Expect(kv.Status.Phase).To(Equal(v1.KubeVirtPhaseDeleted))
			Expect(kv.Status.Conditions).To(HaveLen(3))
			shouldExpectHCOConditions(kv, k8sv1.ConditionFalse, k8sv1.ConditionFalse, k8sv1.ConditionTrue)
		},
			Entry("without export", false, resourceCount-3),
			Entry("with export", true, resourceCount),
		)

		DescribeTable("should remove poddisruptionbudgets on deletion", func(withExport bool, numPDBs int) {

			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-install",
					Namespace: NAMESPACE,
				},
			}
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)

			kv.DeletionTimestamp = now()
			if withExport {
				enableExportFeatureGate(kv)
			}

			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)

			// create all resources which should be deleted
			kvTestData.addInstallStrategy(kvTestData.defaultConfig)
			kvTestData.addAll(kvTestData.defaultConfig, kv)

			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.shouldExpectDeletions()

			// Due to finalizers being added to CRDs during deletion
			kvTestData.extClient.Fake.PrependReactor("patch", "customresourcedefinitions", kvTestData.crdPatchFunc())
			kvTestData.shouldExpectInstallStrategyDeletion()

			kvTestData.controller.Execute()

			Expect(kvTestData.resourceChanges["poddisruptionbudgets"][Deleted]).To(Equal(numPDBs))
		},
			Entry("without export", false, 2),
			Entry("with export", true, 3),
		)
	})

	Context("VirtTemplateDeployment configuration", func() {
		var kvTestData *KubeVirtTestData

		virtTemplateResourcesFound := func() bool {
			for _, obj := range kvTestData.controller.stores.DeploymentCache.List() {
				deployment := obj.(*appsv1.Deployment)
				if strings.HasPrefix(deployment.Name, "virt-template") {
					return true
				}
			}
			for _, obj := range kvTestData.controller.stores.OperatorCrdCache.List() {
				crd := obj.(*extv1.CustomResourceDefinition)
				if strings.Contains(crd.Name, "template.kubevirt.io") {
					return true
				}
			}
			return false
		}

		BeforeEach(func() {
			kvTestData = &KubeVirtTestData{}
			kvTestData.BeforeTest()
			DeferCleanup(kvTestData.AfterTest)
			err := kvTestData.controller.stores.NamespaceCache.Add(&k8sv1.Namespace{
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
			Expect(err).ToNot(HaveOccurred())
		})

		DescribeTable("should not create virt-template resources", func(kv *v1.KubeVirt) {
			config := util.GetTargetConfigFromKVWithEnvVarManager(kv, kvTestData.mockEnvVarManager)
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategy(config)

			job, err := kvTestData.controller.generateInstallStrategyJob(kv.Spec.Infra, config)
			Expect(err).ToNot(HaveOccurred())

			job.Status.CompletionTime = now()
			kvTestData.addInstallStrategyJob(job)

			kvTestData.deleteFromCache = false
			kvTestData.shouldExpectJobDeletion()
			kvTestData.shouldExpectKubeVirtFinalizersPatch(1)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.shouldExpectCreations()

			kvTestData.controller.Execute()

			// 39 in total should be missing at this point
			// because waiting on controller, controller's PDB and virt-handler daemonset until API server deploys successfully
			// also exportProxy + PDB + route
			// and all virt-template resources
			expectedUncreatedResources := 39

			// KV should be progressing, resources have been added, but they are not all ready yet
			shouldExpectHCOConditions(kvTestData.getLatestKubeVirt(kv), k8sv1.ConditionFalse, k8sv1.ConditionTrue, k8sv1.ConditionFalse)
			Expect(kvTestData.totalAdds).To(Equal(resourceCount - expectedUncreatedResources + expectedTemporaryResources + externalCAConfigMapCount))
			Expect(virtTemplateResourcesFound()).To(BeFalse())
		},
			Entry("when Template feature gate is not enabled", &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-install",
					Namespace: NAMESPACE,
				},
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						DeveloperConfiguration: &v1.DeveloperConfiguration{
							FeatureGates: []string{featuregate.ContainerPathVolumesGate},
						},
						VirtTemplateDeployment: &v1.VirtTemplateDeployment{
							Enabled: pointer.P(true),
						},
					},
				},
			}),
			Entry("when VirtTemplateDeployment.Enabled is false", &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-install",
					Namespace: NAMESPACE,
				},
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						DeveloperConfiguration: &v1.DeveloperConfiguration{
							FeatureGates: []string{featuregate.Template, featuregate.ContainerPathVolumesGate},
						},
						VirtTemplateDeployment: &v1.VirtTemplateDeployment{
							Enabled: pointer.P(false),
						},
					},
				},
			}),
		)

		It("should delete virt-template resources when VirtTemplateDeployment is disabled after being enabled", func() {
			// Create existing KV with virt-template enabled
			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						VirtTemplateDeployment: &v1.VirtTemplateDeployment{
							Enabled: pointer.P(true),
						},
					},
				},
			}
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
			config := util.GetTargetConfigFromKVWithEnvVarManager(kv, kvTestData.mockEnvVarManager)

			// Create new KV with virt-template disabled
			newKv := kv.DeepCopy()
			newKv.ObjectMeta.Generation = 2
			newKv.Spec.Configuration.VirtTemplateDeployment = &v1.VirtTemplateDeployment{
				Enabled: pointer.P(false),
			}
			newConfig := util.GetTargetConfigFromKVWithEnvVarManager(newKv, kvTestData.mockEnvVarManager)

			kvTestData.deleteFromCache = true
			kubecontroller.SetLatestApiVersionAnnotation(newKv)
			kvTestData.addKubeVirt(newKv)
			kvTestData.addInstallStrategy(newConfig)

			// Add all resources including virt-template (simulating previous enabled state)
			kvTestData.addAll(config, kv)
			kvTestData.addPodsAndPodDisruptionBudgets(config, kv)

			// Add updated Pods for new config
			kvTestData.addPodsAndPodDisruptionBudgets(newConfig, newKv)
			kvTestData.addVirtHandler(newConfig, newKv)
			kvTestData.makeDeploymentsReady(kv)
			kvTestData.makeHandlerReady()

			kvTestData.shouldExpectDeletions()
			kvTestData.fakeNamespaceModificationEvent()
			kvTestData.shouldExpectNamespacePatch()
			kvTestData.shouldExpectPatchesAndUpdates(newKv)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)

			kvTestData.controller.Execute()

			// KV should be available, resources have been deleted and all others are already ready
			shouldExpectHCOConditions(kvTestData.getLatestKubeVirt(newKv), k8sv1.ConditionTrue, k8sv1.ConditionFalse, k8sv1.ConditionFalse)
			Expect(kvTestData.totalDeletions).To(Equal(virtTemplateResourceCount))
			Expect(virtTemplateResourcesFound()).To(BeFalse())
		})

		It("should create virt-template resources when VirtTemplateDeployment is enabled after being disabled", func() {
			// Create existing KV with virt-template disabled
			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						VirtTemplateDeployment: &v1.VirtTemplateDeployment{
							Enabled: pointer.P(false),
						},
					},
				},
			}
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
			config := util.GetTargetConfigFromKVWithEnvVarManager(kv, kvTestData.mockEnvVarManager)

			// Create new KV with virt-template enabled
			newKv := kv.DeepCopy()
			newKv.ObjectMeta.Generation = 2
			newKv.Spec.Configuration.VirtTemplateDeployment = &v1.VirtTemplateDeployment{
				Enabled: pointer.P(true),
			}
			newConfig := util.GetTargetConfigFromKVWithEnvVarManager(newKv, kvTestData.mockEnvVarManager)

			kubecontroller.SetLatestApiVersionAnnotation(newKv)
			kvTestData.addKubeVirt(newKv)
			kvTestData.addInstallStrategy(newConfig)

			job, err := kvTestData.controller.generateInstallStrategyJob(kv.Spec.Infra, newConfig)
			Expect(err).ToNot(HaveOccurred())

			job.Status.CompletionTime = now()
			kvTestData.addInstallStrategyJob(job)

			// Add all resources excluding virt-template (simulating previous disabled state)
			kvTestData.addAll(config, kv)
			kvTestData.addPodsAndPodDisruptionBudgets(config, kv)

			// Add updated Pods for new config
			kvTestData.addPodsAndPodDisruptionBudgets(newConfig, newKv)
			kvTestData.addVirtHandler(newConfig, newKv)
			kvTestData.makeDeploymentsReady(kv)
			kvTestData.makeHandlerReady()

			kvTestData.deleteFromCache = false
			kvTestData.shouldExpectJobDeletion()
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.shouldExpectCreations()
			kvTestData.shouldExpectPatchesAndUpdates(newKv)

			kvTestData.controller.Execute()

			// KV should be progressing, resources have been added, but they are not all ready yet
			shouldExpectHCOConditions(kvTestData.getLatestKubeVirt(newKv), k8sv1.ConditionFalse, k8sv1.ConditionTrue, k8sv1.ConditionFalse)
			Expect(kvTestData.totalAdds).To(Equal(virtTemplateResourceCount + externalCAConfigMapCount))
		})
	})

	Context("ContainerPathVolumes feature gate", func() {
		var kvTestData *KubeVirtTestData

		BeforeEach(func() {
			kvTestData = &KubeVirtTestData{}
			kvTestData.BeforeTest()
			DeferCleanup(kvTestData.AfterTest)
		})

		It("should not create virt-launcher-pod-mutator webhook when disabled", func() {
			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-install",
					Namespace: NAMESPACE,
				},
			}
			enableTemplateFeatureGate(kv)
			// ContainerPathVolumes NOT enabled
			config := util.GetTargetConfigFromKVWithEnvVarManager(kv, kvTestData.mockEnvVarManager)

			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)
			kvTestData.addInstallStrategy(config)

			job, err := kvTestData.controller.generateInstallStrategyJob(kv.Spec.Infra, config)
			Expect(err).ToNot(HaveOccurred())

			job.Status.CompletionTime = now()
			kvTestData.addInstallStrategyJob(job)

			kvTestData.deleteFromCache = false
			kvTestData.shouldExpectJobDeletion()
			kvTestData.shouldExpectKubeVirtFinalizersPatch(1)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.shouldExpectCreations()

			kvTestData.controller.Execute()

			_, exists, err := kvTestData.controller.stores.MutatingWebhookCache.GetByKey(components.VirtLauncherPodMutatingWebhookName)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse(), "virt-launcher-pod-mutator webhook should not exist when ContainerPathVolumes is disabled")
		})

		It("should delete virt-launcher-pod-mutator webhook when ContainerPathVolumes is disabled after being enabled", func() {
			// Create existing KV with ContainerPathVolumes enabled
			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
			}
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
			config := util.GetTargetConfigFromKVWithEnvVarManager(kv, kvTestData.mockEnvVarManager)

			// Create new KV with ContainerPathVolumes disabled
			newKv := kv.DeepCopy()
			newKv.ObjectMeta.Generation = 2
			newKv.Spec.Configuration.DeveloperConfiguration = &v1.DeveloperConfiguration{
				FeatureGates: []string{featuregate.Template},
			}
			newConfig := util.GetTargetConfigFromKVWithEnvVarManager(newKv, kvTestData.mockEnvVarManager)

			kvTestData.deleteFromCache = true
			kubecontroller.SetLatestApiVersionAnnotation(newKv)
			kvTestData.addKubeVirt(newKv)
			kvTestData.addInstallStrategy(newConfig)

			// Add all resources including containerpath webhook (simulating previous enabled state)
			kvTestData.addAll(config, kv)
			kvTestData.addPodsAndPodDisruptionBudgets(config, kv)

			// Add updated Pods for new config
			kvTestData.addPodsAndPodDisruptionBudgets(newConfig, newKv)
			kvTestData.addVirtHandler(newConfig, newKv)
			kvTestData.makeDeploymentsReady(kv)
			kvTestData.makeHandlerReady()

			kvTestData.shouldExpectDeletions()
			kvTestData.fakeNamespaceModificationEvent()
			kvTestData.shouldExpectNamespacePatch()
			kvTestData.shouldExpectPatchesAndUpdates(newKv)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)

			kvTestData.controller.Execute()

			_, exists, err := kvTestData.controller.stores.MutatingWebhookCache.GetByKey(components.VirtLauncherPodMutatingWebhookName)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse(), "virt-launcher-pod-mutator webhook should be deleted when ContainerPathVolumes is disabled")
		})

		It("should create virt-launcher-pod-mutator webhook when ContainerPathVolumes is enabled after being disabled", func() {
			// Create existing KV with ContainerPathVolumes disabled
			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
			}
			enableTemplateFeatureGate(kv)
			// ContainerPathVolumes NOT enabled
			config := util.GetTargetConfigFromKVWithEnvVarManager(kv, kvTestData.mockEnvVarManager)

			// Create new KV with ContainerPathVolumes enabled
			newKv := kv.DeepCopy()
			newKv.ObjectMeta.Generation = 2
			enableContainerPathVolumesFeatureGate(newKv)
			newConfig := util.GetTargetConfigFromKVWithEnvVarManager(newKv, kvTestData.mockEnvVarManager)

			kubecontroller.SetLatestApiVersionAnnotation(newKv)
			kvTestData.addKubeVirt(newKv)
			kvTestData.addInstallStrategy(newConfig)

			// Add all resources excluding containerpath webhook (simulating previous disabled state)
			kvTestData.addAll(config, kv)
			kvTestData.addPodsAndPodDisruptionBudgets(config, kv)

			// Add updated Pods for new config
			kvTestData.addPodsAndPodDisruptionBudgets(newConfig, newKv)
			kvTestData.addVirtHandler(newConfig, newKv)
			kvTestData.makeDeploymentsReady(kv)
			kvTestData.makeHandlerReady()

			kvTestData.shouldExpectCreations()
			kvTestData.shouldExpectPatchesAndUpdates(newKv)
			kvTestData.shouldExpectKubeVirtUpdateStatus(1)
			kvTestData.fakeNamespaceModificationEvent()
			kvTestData.shouldExpectNamespacePatch()

			kvTestData.controller.Execute()

			_, exists, err := kvTestData.controller.stores.MutatingWebhookCache.GetByKey(components.VirtLauncherPodMutatingWebhookName)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue(), "virt-launcher-pod-mutator webhook should exist when ContainerPathVolumes is enabled")
		})
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
			enableTemplateFeatureGate(kv)
			enableContainerPathVolumesFeatureGate(kv)
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			kvTestData.addKubeVirt(kv)

			// install strategy config
			resource, _ := install.NewInstallStrategyConfigMap(kvTestData.defaultConfig, "", NAMESPACE)
			resource.Name = fmt.Sprintf("%s-%s", resource.Name, rand.String(10))
			kvTestData.addResource(resource, kvTestData.defaultConfig, nil)

			job, err := kvTestData.controller.generateInstallStrategyJob(kv.Spec.Infra, util.GetTargetConfigFromKV(kv))
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

			Expect(kvTestData.controller.stores.RoleCache.List()).To(HaveLen(6))
			Expect(kvTestData.controller.stores.RoleBindingCache.List()).To(HaveLen(7))
			Expect(kvTestData.controller.stores.ServiceMonitorCache.List()).To(BeEmpty())
		})
	})

	Context("On install strategy dump", func() {
		It("should generate latest install strategy and post as config map", func() {
			kvTestData := KubeVirtTestData{}
			kvTestData.BeforeTest()
			defer kvTestData.AfterTest()

			config := kvTestData.defaultConfig
			// Mimic generateInstallStrategyJob
			// TODO: Refactor
			envVars := util.NewEnvVarMap(config.GetExtraEnv())
			envVarManager := util.DefaultEnvVarManager
			for _, envVar := range envVars {
				envVarManager.Setenv(envVar.Name, envVar.Value)
			}
			deploymentConfigJson, err := config.GetJson()
			Expect(err).ToNot(HaveOccurred())
			envVarManager.Setenv(util.TargetDeploymentConfig, deploymentConfigJson)

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
		})
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
	if config.GetProductComponent() != "" {
		objectMeta.Labels[v1.AppComponentLabel] = config.GetProductComponent()
	}
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
