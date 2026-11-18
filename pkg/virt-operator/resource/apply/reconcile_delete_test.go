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

package apply

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	secv1 "github.com/openshift/api/security/v1"
	secfake "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1/fake"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"go.uber.org/mock/gomock"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	virtfake "kubevirt.io/client-go/kubevirt/fake"
	promfake "kubevirt.io/client-go/prometheusoperator/fake"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

type cleanupResource struct {
	object       runtime.Object
	store        cache.Store
	expectations *controller.UIDTrackingControllerExpectations
}

type cleanupAPIServiceClient struct {
	install.APIServiceInterface
	testing.Fake
}

func (client *cleanupAPIServiceClient) Delete(_ context.Context, name string, options metav1.DeleteOptions) error {
	_, err := client.Invokes(testing.NewRootDeleteActionWithOptions(apiregv1.SchemeGroupVersion.WithResource("apiservices"), name, options), nil)
	return err
}

var _ = Describe("deleteObjectsNotInInstallStrategy", func() {
	var reconciler *Reconciler
	var virtClient *kubecli.MockKubevirtClient
	var extClient *extfake.Clientset
	var secClient *secfake.FakeSecurityV1
	var promClient *promfake.Clientset
	var clusterTypeClient *virtfake.Clientset
	var resources map[string]cleanupResource
	var deletions []testing.DeleteAction
	var deleteErr error

	BeforeEach(func() {
		virtClient = kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))
		k8sClient := k8sfake.NewSimpleClientset()
		extClient = extfake.NewSimpleClientset()
		secClient = &secfake.FakeSecurityV1{Fake: &testing.Fake{}}
		promClient = promfake.NewSimpleClientset()
		clusterTypeClient = virtfake.NewSimpleClientset()
		aggregatorClient := &cleanupAPIServiceClient{}
		deletions = nil
		deleteErr = nil

		recordDeletion := func(action testing.Action) (bool, runtime.Object, error) {
			Expect(action.GetVerb()).To(Equal("delete"))
			deletions = append(deletions, action.(testing.DeleteAction))
			return true, nil, deleteErr
		}
		for _, client := range []*testing.Fake{&k8sClient.Fake, &extClient.Fake, secClient.Fake, &promClient.Fake, &aggregatorClient.Fake} {
			client.PrependReactor("*", "*", recordDeletion)
		}
		clusterTypeClient.PrependReactor("delete", "*", recordDeletion)

		reconciler = &Reconciler{
			kv:               &v1.KubeVirt{ObjectMeta: metav1.ObjectMeta{Name: "kubevirt", Namespace: Namespace}},
			kvKey:            Namespace + "/kubevirt",
			virtClient:       virtClient,
			k8sClient:        k8sClient,
			aggregatorclient: aggregatorClient,
			targetStrategy:   &install.Strategy{},
			stores: util.Stores{
				ValidationWebhookCache:                cache.NewStore(cache.MetaNamespaceKeyFunc),
				MutatingWebhookCache:                  cache.NewStore(cache.MetaNamespaceKeyFunc),
				APIServiceCache:                       cache.NewStore(cache.MetaNamespaceKeyFunc),
				SecretCache:                           cache.NewStore(cache.MetaNamespaceKeyFunc),
				ConfigMapCache:                        cache.NewStore(cache.MetaNamespaceKeyFunc),
				ValidatingAdmissionPolicyBindingCache: cache.NewStore(cache.MetaNamespaceKeyFunc),
				ValidatingAdmissionPolicyCache:        cache.NewStore(cache.MetaNamespaceKeyFunc),
				OperatorCrdCache:                      cache.NewStore(cache.MetaNamespaceKeyFunc),
				DaemonSetCache:                        cache.NewStore(cache.MetaNamespaceKeyFunc),
				DeploymentCache:                       cache.NewStore(cache.MetaNamespaceKeyFunc),
				ServiceCache:                          cache.NewStore(cache.MetaNamespaceKeyFunc),
				ClusterRoleBindingCache:               cache.NewStore(cache.MetaNamespaceKeyFunc),
				ClusterRoleCache:                      cache.NewStore(cache.MetaNamespaceKeyFunc),
				RoleBindingCache:                      cache.NewStore(cache.MetaNamespaceKeyFunc),
				RoleCache:                             cache.NewStore(cache.MetaNamespaceKeyFunc),
				ServiceAccountCache:                   cache.NewStore(cache.MetaNamespaceKeyFunc),
				SCCCache:                              cache.NewStore(cache.MetaNamespaceKeyFunc),
				PrometheusRuleCache:                   cache.NewStore(cache.MetaNamespaceKeyFunc),
				ServiceMonitorCache:                   cache.NewStore(cache.MetaNamespaceKeyFunc),
				InstallStrategyConfigMapCache:         cache.NewStore(cache.MetaNamespaceKeyFunc),
			},
			expectations: &util.Expectations{
				ValidationWebhook:                controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ValidationWebhook")),
				MutatingWebhook:                  controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("MutatingWebhook")),
				APIService:                       controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("APIService")),
				Secrets:                          controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("Secrets")),
				ConfigMap:                        controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ConfigMap")),
				ValidatingAdmissionPolicyBinding: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ValidatingAdmissionPolicyBinding")),
				ValidatingAdmissionPolicy:        controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ValidatingAdmissionPolicy")),
				OperatorCrd:                      controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("OperatorCrd")),
				DaemonSet:                        controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("DaemonSet")),
				Deployment:                       controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("Deployment")),
				Service:                          controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("Service")),
				ClusterRoleBinding:               controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ClusterRoleBinding")),
				ClusterRole:                      controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ClusterRole")),
				RoleBinding:                      controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("RoleBinding")),
				Role:                             controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("Role")),
				ServiceAccount:                   controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ServiceAccount")),
				SCC:                              controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("SCC")),
				PrometheusRule:                   controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("PrometheusRule")),
				ServiceMonitor:                   controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ServiceMonitor")),
			},
		}
		clusterMetadata := metav1.ObjectMeta{Name: "obsolete-resource", Labels: map[string]string{v1.ManagedByLabel: v1.ManagedByLabelOperatorValue}}
		namespacedMetadata := *clusterMetadata.DeepCopy()
		namespacedMetadata.Namespace = Namespace
		resources = map[string]cleanupResource{
			"secrets": {
				object:       &corev1.Secret{ObjectMeta: namespacedMetadata},
				store:        reconciler.stores.SecretCache,
				expectations: reconciler.expectations.Secrets,
			},
			"configmaps": {
				object:       &corev1.ConfigMap{ObjectMeta: namespacedMetadata},
				store:        reconciler.stores.ConfigMapCache,
				expectations: reconciler.expectations.ConfigMap,
			},
			"daemonsets": {
				object:       &appsv1.DaemonSet{ObjectMeta: namespacedMetadata},
				store:        reconciler.stores.DaemonSetCache,
				expectations: reconciler.expectations.DaemonSet,
			},
			"deployments": {
				object:       &appsv1.Deployment{ObjectMeta: namespacedMetadata},
				store:        reconciler.stores.DeploymentCache,
				expectations: reconciler.expectations.Deployment,
			},
			"services": {
				object:       &corev1.Service{ObjectMeta: namespacedMetadata},
				store:        reconciler.stores.ServiceCache,
				expectations: reconciler.expectations.Service,
			},
			"clusterrolebindings": {
				object:       &rbacv1.ClusterRoleBinding{ObjectMeta: clusterMetadata},
				store:        reconciler.stores.ClusterRoleBindingCache,
				expectations: reconciler.expectations.ClusterRoleBinding,
			},
			"clusterroles": {
				object:       &rbacv1.ClusterRole{ObjectMeta: clusterMetadata},
				store:        reconciler.stores.ClusterRoleCache,
				expectations: reconciler.expectations.ClusterRole,
			},
			"rolebindings": {
				object:       &rbacv1.RoleBinding{ObjectMeta: namespacedMetadata},
				store:        reconciler.stores.RoleBindingCache,
				expectations: reconciler.expectations.RoleBinding,
			},
			"roles": {
				object:       &rbacv1.Role{ObjectMeta: namespacedMetadata},
				store:        reconciler.stores.RoleCache,
				expectations: reconciler.expectations.Role,
			},
			"serviceaccounts": {
				object:       &corev1.ServiceAccount{ObjectMeta: namespacedMetadata},
				store:        reconciler.stores.ServiceAccountCache,
				expectations: reconciler.expectations.ServiceAccount,
			},
		}
		scheme := runtime.NewScheme()
		for _, addToScheme := range []func(*runtime.Scheme) error{
			admissionregistrationv1.AddToScheme, appsv1.AddToScheme, corev1.AddToScheme,
			rbacv1.AddToScheme, extv1.AddToScheme, apiregv1.AddToScheme,
			secv1.AddToScheme, promv1.AddToScheme,
		} {
			Expect(addToScheme(scheme)).To(Succeed())
		}
		for _, resource := range resources {
			kinds, _, err := scheme.ObjectKinds(resource.object)
			Expect(err).NotTo(HaveOccurred())
			Expect(kinds).To(HaveLen(1))
			resource.object.GetObjectKind().SetGroupVersionKind(kinds[0])
		}
	})

	DescribeTableSubtree("cached resources", func(resourceName string) {
		var resource cleanupResource
		var metadata metav1.Object

		BeforeEach(func() {
			resource = resources[resourceName]
			var err error
			metadata, err = meta.Accessor(resource.object)
			Expect(err).NotTo(HaveOccurred())
			targetResource := resource.object.DeepCopyObject()
			targetMetadata, err := meta.Accessor(targetResource)
			Expect(err).NotTo(HaveOccurred())
			targetMetadata.SetName("required-resource")
			reconciler.targetStrategy = loadTargetStrategy(targetResource, getConfig(Registry, Version), reconciler.stores)

			Expect(resource.store.Add(resource.object)).To(Succeed())
		})

		It("deletes obsolete resources and waits for the deletion event", func() {
			virtClient.EXPECT().ExtensionsClient().Return(extClient)
			virtClient.EXPECT().SecClient().Return(secClient).AnyTimes()
			virtClient.EXPECT().PrometheusClient().Return(promClient).AnyTimes()
			virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(clusterTypeClient.InstancetypeV1beta1().VirtualMachineClusterInstancetypes()).AnyTimes()
			virtClient.EXPECT().VirtualMachineClusterPreference().Return(clusterTypeClient.InstancetypeV1beta1().VirtualMachineClusterPreferences()).AnyTimes()

			Expect(reconciler.deleteObjectsNotInInstallStrategy()).To(Succeed())

			Expect(deletions).To(ConsistOf(And(
				HaveField("Name", metadata.GetName()),
				HaveField("Namespace", metadata.GetNamespace()),
				HaveField("Resource.Resource", resourceName),
			)))
			resourceKey, err := controller.KeyFunc(resource.object)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.expectations.GetUIDs(reconciler.kvKey)).To(HaveKey(resourceKey))
			Expect(resource.expectations.SatisfiedExpectations(reconciler.kvKey)).To(BeFalse())
			resource.expectations.DeletionObserved(reconciler.kvKey, resourceKey)
			Expect(resource.expectations.SatisfiedExpectations(reconciler.kvKey)).To(BeTrue())
		})

		It("preserves resources in the target strategy", func() {
			virtClient.EXPECT().ExtensionsClient().Return(extClient)
			virtClient.EXPECT().SecClient().Return(secClient).AnyTimes()
			virtClient.EXPECT().PrometheusClient().Return(promClient).AnyTimes()
			virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(clusterTypeClient.InstancetypeV1beta1().VirtualMachineClusterInstancetypes()).AnyTimes()
			virtClient.EXPECT().VirtualMachineClusterPreference().Return(clusterTypeClient.InstancetypeV1beta1().VirtualMachineClusterPreferences()).AnyTimes()

			reconciler.targetStrategy = loadTargetStrategy(resource.object, getConfig(Registry, Version), reconciler.stores)

			Expect(reconciler.deleteObjectsNotInInstallStrategy()).To(Succeed())
			Expect(deletions).To(BeEmpty())
			Expect(resource.expectations.GetUIDs(reconciler.kvKey)).To(BeEmpty())
		})

		It("does not delete resources already being deleted", func() {
			virtClient.EXPECT().ExtensionsClient().Return(extClient)
			virtClient.EXPECT().SecClient().Return(secClient).AnyTimes()
			virtClient.EXPECT().PrometheusClient().Return(promClient).AnyTimes()
			virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(clusterTypeClient.InstancetypeV1beta1().VirtualMachineClusterInstancetypes()).AnyTimes()
			virtClient.EXPECT().VirtualMachineClusterPreference().Return(clusterTypeClient.InstancetypeV1beta1().VirtualMachineClusterPreferences()).AnyTimes()

			metadata.SetDeletionTimestamp(now())

			Expect(reconciler.deleteObjectsNotInInstallStrategy()).To(Succeed())
			Expect(deletions).To(BeEmpty())
			Expect(resource.expectations.GetUIDs(reconciler.kvKey)).To(BeEmpty())
		})

		It("returns deletion errors and clears the failed deletion expectation", func() {
			virtClient.EXPECT().ExtensionsClient().Return(extClient)
			virtClient.EXPECT().SecClient().Return(secClient).AnyTimes()
			virtClient.EXPECT().PrometheusClient().Return(promClient).AnyTimes()
			virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(clusterTypeClient.InstancetypeV1beta1().VirtualMachineClusterInstancetypes()).AnyTimes()
			virtClient.EXPECT().VirtualMachineClusterPreference().Return(clusterTypeClient.InstancetypeV1beta1().VirtualMachineClusterPreferences()).AnyTimes()

			deleteErr = errors.New("deletion failed")
			resource.expectations.AddExpectedDeletion(reconciler.kvKey, "pending-resource")

			Expect(reconciler.deleteObjectsNotInInstallStrategy()).To(MatchError(deleteErr))
			Expect(deletions).To(HaveLen(1))
			Expect(resource.expectations.GetUIDs(reconciler.kvKey)).To(HaveLen(1))
			Expect(resource.expectations.GetUIDs(reconciler.kvKey)).To(HaveKey("pending-resource"))
			Expect(resource.expectations.SatisfiedExpectations(reconciler.kvKey)).To(BeFalse())
			resource.expectations.DeletionObserved(reconciler.kvKey, "pending-resource")
			Expect(resource.expectations.SatisfiedExpectations(reconciler.kvKey)).To(BeTrue())
		})
	},
		Entry("secrets", "secrets"),
		Entry("configmaps", "configmaps"),
		Entry("daemonsets", "daemonsets"),
		Entry("deployments", "deployments"),
		Entry("services", "services"),
		Entry("clusterrolebindings", "clusterrolebindings"),
		Entry("clusterroles", "clusterroles"),
		Entry("rolebindings", "rolebindings"),
		Entry("roles", "roles"),
		Entry("serviceaccounts", "serviceaccounts"),
	)

	DescribeTable("matches namespaced resources by both name and namespace", func(resourceName string) {
		virtClient.EXPECT().ExtensionsClient().Return(extClient)
		virtClient.EXPECT().PrometheusClient().Return(promClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(clusterTypeClient.InstancetypeV1beta1().VirtualMachineClusterInstancetypes())
		virtClient.EXPECT().VirtualMachineClusterPreference().Return(clusterTypeClient.InstancetypeV1beta1().VirtualMachineClusterPreferences())

		resource := resources[resourceName]
		reconciler.targetStrategy = loadTargetStrategy(resource.object, getConfig(Registry, Version), reconciler.stores)
		metadata, err := meta.Accessor(resource.object)
		Expect(err).NotTo(HaveOccurred())
		metadata.SetNamespace("previous-installation")
		Expect(resource.store.Add(resource.object)).To(Succeed())

		Expect(reconciler.deleteObjectsNotInInstallStrategy()).To(Succeed())
		Expect(deletions).To(ConsistOf(And(
			HaveField("Name", metadata.GetName()),
			HaveField("Namespace", "previous-installation"),
			HaveField("Resource.Resource", resourceName),
		)))
	},
		Entry("daemonsets", "daemonsets"),
		Entry("deployments", "deployments"),
		Entry("services", "services"),
		Entry("rolebindings", "rolebindings"),
		Entry("roles", "roles"),
		Entry("serviceaccounts", "serviceaccounts"),
	)

})
