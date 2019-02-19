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
	"fmt"
	"os"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	secv1 "github.com/openshift/api/security/v1"
	secv1fake "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1/fake"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extclientfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/rbac"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

var _ = Describe("KubeVirt Operator", func() {
	log.Log.SetIOWriter(GinkgoWriter)

	var ctrl *gomock.Controller
	var kvInterface *kubecli.MockKubeVirtInterface
	var kvSource *framework.FakeControllerSource
	var kvInformer cache.SharedIndexInformer

	var serviceAccountSource *framework.FakeControllerSource
	var clusterRoleSource *framework.FakeControllerSource
	var clusterRoleBindingSource *framework.FakeControllerSource
	var roleSource *framework.FakeControllerSource
	var roleBindingSource *framework.FakeControllerSource
	var crdSource *framework.FakeControllerSource
	var serviceSource *framework.FakeControllerSource
	var deploymentSource *framework.FakeControllerSource
	var daemonSetSource *framework.FakeControllerSource
	var sccSource *framework.FakeControllerSource

	var stop chan struct{}
	var controller *KubeVirtController

	var recorder *record.FakeRecorder

	var mockQueue *testutils.MockWorkQueue
	var virtClient *kubecli.MockKubevirtClient
	var kubeClient *fake.Clientset
	var secClient *secv1fake.FakeSecurityV1
	var extClient *extclientfake.Clientset

	var informers util.Informers
	var stores util.Stores

	os.Setenv(util.OperatorImageEnvName, "whatever/virt-operator:thisversion")

	var totalAdds int
	var totalDeletions int

	NAMESPACE := "kubevirt-test"
	resourceCount := 29

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
		go informers.SCC.Run(stop)

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
		cache.WaitForCacheSync(stop, informers.SCC.HasSynced)

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

	BeforeEach(func() {

		totalAdds = 0
		totalDeletions = 0

		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		kvInterface = kubecli.NewMockKubeVirtInterface(ctrl)

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

		informers.Crd, crdSource = testutils.NewFakeInformerFor(&extv1beta1.CustomResourceDefinition{})
		stores.CrdCache = informers.Crd.GetStore()

		informers.Service, serviceSource = testutils.NewFakeInformerFor(&k8sv1.Service{})
		stores.ServiceCache = informers.Service.GetStore()

		informers.Deployment, deploymentSource = testutils.NewFakeInformerFor(&appsv1.Deployment{})
		stores.DeploymentCache = informers.Deployment.GetStore()

		informers.DaemonSet, daemonSetSource = testutils.NewFakeInformerFor(&appsv1.DaemonSet{})
		stores.DaemonSetCache = informers.DaemonSet.GetStore()

		informers.SCC, sccSource = testutils.NewFakeInformerFor(&secv1.SecurityContextConstraints{})
		stores.SCCCache = informers.SCC.GetStore()

		controller = NewKubeVirtController(virtClient, kvInformer, recorder, stores, informers)

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

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().RbacV1().Return(kubeClient.RbacV1()).AnyTimes()
		virtClient.EXPECT().AppsV1().Return(kubeClient.AppsV1()).AnyTimes()
		virtClient.EXPECT().SecClient().Return(secClient).AnyTimes()
		virtClient.EXPECT().ExtensionsClient().Return(extClient).AnyTimes()

		// Make sure that all unexpected calls to kubeClient will fail
		kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
		secClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
		extClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
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

	addCrd := func(crd *extv1beta1.CustomResourceDefinition) {
		mockQueue.ExpectAdds(1)
		crdSource.Add(crd)
		mockQueue.Wait()
	}

	addService := func(svc *k8sv1.Service) {
		mockQueue.ExpectAdds(1)
		serviceSource.Add(svc)
		mockQueue.Wait()
	}

	addDeployment := func(depl *appsv1.Deployment) {
		mockQueue.ExpectAdds(1)
		deploymentSource.Add(depl)
		mockQueue.Wait()
	}

	addDaemonset := func(ds *appsv1.DaemonSet) {
		mockQueue.ExpectAdds(1)
		daemonSetSource.Add(ds)
		mockQueue.Wait()
	}

	addResource := func(obj runtime.Object) {
		switch resource := obj.(type) {
		case *k8sv1.ServiceAccount:
			addServiceAccount(resource)
		case *rbacv1.ClusterRole:
			addClusterRole(resource)
		case *rbacv1.ClusterRoleBinding:
			addClusterRoleBinding(resource)
		case *rbacv1.Role:
			addRole(resource)
		case *rbacv1.RoleBinding:
			addRoleBinding(resource)
		case *extv1beta1.CustomResourceDefinition:
			addCrd(resource)
		case *k8sv1.Service:
			addService(resource)
		case *appsv1.Deployment:
			addDeployment(resource)
		case *appsv1.DaemonSet:
			addDaemonset(resource)
		default:
			Fail("unknown resource type")
		}
	}

	addAll := func() {
		repository := "kubevirt"
		version := "latest"
		pullPolicy := "IfNotPresent"
		imagePullPolicy := k8sv1.PullPolicy(pullPolicy)
		verbosity := "2"

		all := make([]interface{}, 0)
		// rbac
		all = append(all, rbac.GetAllCluster(NAMESPACE)...)
		all = append(all, rbac.GetAllApiServer(NAMESPACE)...)
		all = append(all, rbac.GetAllHandler(NAMESPACE)...)
		all = append(all, rbac.GetAllController(NAMESPACE)...)
		// crds
		all = append(all, components.NewVirtualMachineInstanceCrd())
		all = append(all, components.NewPresetCrd())
		all = append(all, components.NewReplicaSetCrd())
		all = append(all, components.NewVirtualMachineCrd())
		all = append(all, components.NewVirtualMachineInstanceMigrationCrd())
		// services and deployments
		all = append(all, components.NewPrometheusService(NAMESPACE))
		all = append(all, components.NewApiServerService(NAMESPACE))
		apiDeployment, _ := components.NewApiServerDeployment(NAMESPACE, repository, version, imagePullPolicy, verbosity)
		controller, _ := components.NewControllerDeployment(NAMESPACE, repository, version, imagePullPolicy, verbosity)
		handler, _ := components.NewHandlerDaemonSet(NAMESPACE, repository, version, imagePullPolicy, verbosity)
		all = append(all, apiDeployment, controller, handler)

		for _, obj := range all {
			if resource, ok := obj.(runtime.Object); ok {
				addResource(resource)
			} else {
				Fail("could not cast to runtime.Object")
			}
		}

		// update SCC
		scc := getSCC()
		prefix := "system:serviceaccount"
		scc.Users = append(scc.Users,
			fmt.Sprintf("%s:%s:%s", prefix, NAMESPACE, "kubevirt-handler"),
			fmt.Sprintf("%s:%s:%s", prefix, NAMESPACE, "kubevirt-apiserver"),
			fmt.Sprintf("%s:%s:%s", prefix, NAMESPACE, "kubevirt-controller"))
		sccSource.Modify(&scc)

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
		default:
			Fail("unknown resource type")
		}
	}

	genericCreateFunc := func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())
		totalAdds++
		addResource(create.GetObject())
		return true, create.GetObject(), nil
	}

	genericDeleteFunc := func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		delete, ok := action.(testing.DeleteAction)
		Expect(ok).To(BeTrue())
		totalDeletions++
		var key string
		if len(delete.GetNamespace()) > 0 {
			key = delete.GetNamespace() + "/"
		}
		key += delete.GetName()
		deleteResource(delete.GetResource().Resource, key)
		return true, nil, nil
	}

	expectUsers := func(sccObj runtime.Object, count int) {
		scc, ok := sccObj.(*secv1.SecurityContextConstraints)
		ExpectWithOffset(2, ok).To(BeTrue())
		ExpectWithOffset(2, len(scc.Users)).To(Equal(count))
	}

	shouldExpectDeletions := func() {
		kubeClient.Fake.PrependReactor("delete", "serviceaccounts", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "clusterroles", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "clusterrolebindings", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "roles", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "rolebindings", genericDeleteFunc)

		secClient.Fake.PrependReactor("update", "securitycontextconstraints", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, _ := action.(testing.UpdateAction)
			updatedObj := update.GetObject()
			expectUsers(updatedObj, 1)
			return true, updatedObj, nil
		})
		extClient.Fake.PrependReactor("delete", "customresourcedefinitions", genericDeleteFunc)

		kubeClient.Fake.PrependReactor("delete", "services", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "deployments", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "daemonsets", genericDeleteFunc)
	}

	shouldExpectCreations := func() {
		kubeClient.Fake.PrependReactor("create", "serviceaccounts", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "clusterroles", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "clusterrolebindings", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "roles", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "rolebindings", genericCreateFunc)

		secClient.Fake.PrependReactor("update", "securitycontextconstraints", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, _ := action.(testing.UpdateAction)
			updatedObj := update.GetObject()
			expectUsers(updatedObj, 4)
			return true, updatedObj, nil
		})
		extClient.Fake.PrependReactor("create", "customresourcedefinitions", genericCreateFunc)

		kubeClient.Fake.PrependReactor("create", "services", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "deployments", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "daemonsets", genericCreateFunc)
	}

	shouldExpectKubeVirtUpdate := func(times int) {
		update := kvInterface.EXPECT().Update(gomock.Any())
		update.Do(func(kv *v1.KubeVirt) {
			kvInformer.GetStore().Update(kv)
			update.Return(kv, nil)
		}).Times(times)
	}

	getLatestKubeVirt := func(kv *v1.KubeVirt) *v1.KubeVirt {
		if obj, exists, _ := kvInformer.GetStore().GetByKey(kv.GetNamespace() + "/" + kv.GetName()); exists {
			if kvLatest, ok := obj.(*v1.KubeVirt); ok {
				return kvLatest
			}
		}
		return nil
	}

	Context("On valid KubeVirt object", func() {
		It("should do nothing if KubeVirt object is deleted", func(done Done) {
			defer close(done)

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-install",
					Namespace: NAMESPACE,
				},
				Status: v1.KubeVirtStatus{
					Phase: v1.KubeVirtPhaseDeleted,
				},
			}
			kv.DeletionTimestamp = now()

			addKubeVirt(kv)
			controller.Execute()
		}, 15)

		It("should do nothing if KubeVirt object is deployed", func(done Done) {
			defer close(done)

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Status: v1.KubeVirtStatus{
					Phase: v1.KubeVirtPhaseDeployed,
					Conditions: []v1.KubeVirtCondition{
						{
							Type:    v1.KubeVirtConditionCreated,
							Status:  k8sv1.ConditionTrue,
							Reason:  ConditionReasonDeploymentCreated,
							Message: "All resources were created.",
						},
						{
							Type:    v1.KubeVirtConditionReady,
							Status:  k8sv1.ConditionTrue,
							Reason:  ConditionReasonDeploymentReady,
							Message: "All components are ready.",
						},
					},
					OperatorVersion: "v0.0.0-master+$Format:%h$",
				},
			}

			// create all resources which should already exist
			addKubeVirt(kv)
			addAll()
			makeApiAndControllerReady()
			makeHandlerReady()

			controller.Execute()

		}, 15)

		It("should add resources on create", func(done Done) {
			defer close(done)

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-install",
					Namespace: NAMESPACE,
				},
			}
			addKubeVirt(kv)

			shouldExpectKubeVirtUpdate(1)
			shouldExpectCreations()

			controller.Execute()

			Expect(totalAdds).To(Equal(resourceCount))
			Expect(len(controller.stores.ServiceAccountCache.List())).To(Equal(3))
			Expect(len(controller.stores.ClusterRoleCache.List())).To(Equal(7))
			Expect(len(controller.stores.ClusterRoleBindingCache.List())).To(Equal(5))
			Expect(len(controller.stores.RoleCache.List())).To(Equal(2))
			Expect(len(controller.stores.RoleBindingCache.List())).To(Equal(2))
			Expect(len(controller.stores.CrdCache.List())).To(Equal(5))
			Expect(len(controller.stores.ServiceCache.List())).To(Equal(2))
			Expect(len(controller.stores.DeploymentCache.List())).To(Equal(2))
			Expect(len(controller.stores.DaemonSetCache.List())).To(Equal(1))

			kv = getLatestKubeVirt(kv)
			Expect(kv.Status.Phase).To(Equal(v1.KubeVirtPhaseDeploying))
			Expect(len(kv.Status.Conditions)).To(Equal(0))

			// in 2nd run everything should already be created, and the Created condition should be set
			totalAdds = 0
			shouldExpectKubeVirtUpdate(1)
			controller.Execute()

			Expect(totalAdds).To(Equal(0))

			kv = getLatestKubeVirt(kv)
			Expect(kv.Status.Phase).To(Equal(v1.KubeVirtPhaseDeploying))
			Expect(len(kv.Status.Conditions)).To(Equal(1))
			condition := kv.Status.Conditions[0]
			Expect(condition.Type).To(Equal(v1.KubeVirtConditionCreated))
			Expect(condition.Status).To(Equal(k8sv1.ConditionTrue))
			Expect(condition.Reason).To(Equal(ConditionReasonDeploymentCreated))

			// make some(!) components ready
			makeApiAndControllerReady()

			// nothing should change as long as not every component is ready
			totalAdds = 0
			controller.Execute()
			Expect(totalAdds).To(Equal(0))

			// make last component ready
			makeHandlerReady()

			// when everything is ready, the Deployed status and Created + Ready condition should be set
			totalAdds = 0
			shouldExpectKubeVirtUpdate(1)
			controller.Execute()
			Expect(totalAdds).To(Equal(0))

			kv = getLatestKubeVirt(kv)
			Expect(kv.Status.Phase).To(Equal(v1.KubeVirtPhaseDeployed))
			Expect(len(kv.Status.Conditions)).To(Equal(2))

			condition1 := kv.Status.Conditions[0]
			Expect(condition1.Type).To(Equal(v1.KubeVirtConditionCreated))
			Expect(condition1.Status).To(Equal(k8sv1.ConditionTrue))
			Expect(condition1.Reason).To(Equal(ConditionReasonDeploymentCreated))

			condition2 := kv.Status.Conditions[1]
			Expect(condition2.Type).To(Equal(v1.KubeVirtConditionReady))
			Expect(condition2.Status).To(Equal(k8sv1.ConditionTrue))
			Expect(condition2.Reason).To(Equal(ConditionReasonDeploymentReady))

		}, 15)

		It("should remove resources on deletion", func(done Done) {
			defer close(done)

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-install",
					Namespace: NAMESPACE,
				},
			}
			kv.DeletionTimestamp = now()
			addKubeVirt(kv)

			// create all resources which should be deleted
			addAll()

			shouldExpectKubeVirtUpdate(1)
			shouldExpectDeletions()

			controller.Execute()

			// Note: in real life during the first execution loop very probably only CRDs are deleted,
			// because that takes some time (see the check that the crd store is empty before going on with deletions)
			// But in this test the deletion succeeds immediately, so everything is deleted on first try
			Expect(totalDeletions).To(Equal(resourceCount))

			kv = getLatestKubeVirt(kv)
			Expect(kv.Status.Phase).To(Equal(v1.KubeVirtPhaseDeleted))
			Expect(len(kv.Status.Conditions)).To(Equal(0))

		}, 15)
	})
})

func now() *metav1.Time {
	now := metav1.Now()
	return &now
}
