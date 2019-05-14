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

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	k8sv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extclientfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/version"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/rbac"
	installstrategy "kubevirt.io/kubevirt/pkg/virt-operator/install-strategy"
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
	var validatingWebhookSource *framework.FakeControllerSource
	var sccSource *framework.FakeControllerSource
	var installStrategyConfigMapSource *framework.FakeControllerSource
	var installStrategyJobSource *framework.FakeControllerSource
	var infrastructurePodSource *framework.FakeControllerSource

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

	defaultImageTag := "v9.9.9"
	defaultRegistry := "someregistry"

	var totalAdds int
	var totalUpdates int
	var totalPatches int
	var totalDeletions int

	NAMESPACE := "kubevirt-test"
	resourceCount := 29
	patchCount := 13
	updateCount := 16

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
		go informers.SCC.Run(stop)
		go informers.InstallStrategyJob.Run(stop)
		go informers.InstallStrategyConfigMap.Run(stop)
		go informers.InfrastructurePod.Run(stop)

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
		cache.WaitForCacheSync(stop, informers.SCC.HasSynced)
		cache.WaitForCacheSync(stop, informers.InstallStrategyJob.HasSynced)
		cache.WaitForCacheSync(stop, informers.InstallStrategyConfigMap.HasSynced)
		cache.WaitForCacheSync(stop, informers.InfrastructurePod.HasSynced)
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

		os.Setenv(util.OperatorImageEnvName, fmt.Sprintf("%s/virt-operator:%s", defaultRegistry, defaultImageTag))

		totalAdds = 0
		totalUpdates = 0
		totalPatches = 0
		totalDeletions = 0
		deleteFromCache = true
		addToCache = true

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

		informers.ValidationWebhook, validatingWebhookSource = testutils.NewFakeInformerFor(&admissionregistrationv1beta1.ValidatingWebhookConfiguration{})
		stores.ValidationWebhookCache = informers.ValidationWebhook.GetStore()

		informers.SCC, sccSource = testutils.NewFakeInformerFor(&secv1.SecurityContextConstraints{})
		stores.SCCCache = informers.SCC.GetStore()

		informers.InstallStrategyConfigMap, installStrategyConfigMapSource = testutils.NewFakeInformerFor(&k8sv1.ConfigMap{})
		stores.InstallStrategyConfigMapCache = informers.InstallStrategyConfigMap.GetStore()

		informers.InstallStrategyJob, installStrategyJobSource = testutils.NewFakeInformerFor(&batchv1.Job{})
		stores.InstallStrategyJobCache = informers.InstallStrategyJob.GetStore()

		informers.InfrastructurePod, infrastructurePodSource = testutils.NewFakeInformerFor(&k8sv1.Pod{})
		stores.InfrastructurePodCache = informers.InfrastructurePod.GetStore()

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

		virtClient.EXPECT().AdmissionregistrationV1beta1().Return(kubeClient.AdmissionregistrationV1beta1()).AnyTimes()
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().BatchV1().Return(kubeClient.BatchV1()).AnyTimes()
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

	injectMetadata := func(objectMeta *metav1.ObjectMeta, version string, registry string) {

		if version == "" && registry == "" {
			return
		}
		if objectMeta.Labels == nil {
			objectMeta.Labels = make(map[string]string)
		}
		objectMeta.Labels[v1.ManagedByLabel] = v1.ManagedByLabelOperatorValue

		if objectMeta.Annotations == nil {
			objectMeta.Annotations = make(map[string]string)
		}
		objectMeta.Annotations[v1.InstallStrategyVersionAnnotation] = version
		objectMeta.Annotations[v1.InstallStrategyRegistryAnnotation] = registry
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

	addValidatingWebhook := func(wh *admissionregistrationv1beta1.ValidatingWebhookConfiguration) {
		mockQueue.ExpectAdds(1)
		validatingWebhookSource.Add(wh)
		mockQueue.Wait()
	}

	addInstallStrategyConfigMap := func(c *k8sv1.ConfigMap) {
		mockQueue.ExpectAdds(1)
		installStrategyConfigMapSource.Add(c)
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

	addResource := func(obj runtime.Object, version string, registry string) {
		switch resource := obj.(type) {
		case *k8sv1.ServiceAccount:
			injectMetadata(&obj.(*k8sv1.ServiceAccount).ObjectMeta, version, registry)
			addServiceAccount(resource)
		case *rbacv1.ClusterRole:
			injectMetadata(&obj.(*rbacv1.ClusterRole).ObjectMeta, version, registry)
			addClusterRole(resource)
		case *rbacv1.ClusterRoleBinding:
			injectMetadata(&obj.(*rbacv1.ClusterRoleBinding).ObjectMeta, version, registry)
			addClusterRoleBinding(resource)
		case *rbacv1.Role:
			injectMetadata(&obj.(*rbacv1.Role).ObjectMeta, version, registry)
			addRole(resource)
		case *rbacv1.RoleBinding:
			injectMetadata(&obj.(*rbacv1.RoleBinding).ObjectMeta, version, registry)
			addRoleBinding(resource)
		case *extv1beta1.CustomResourceDefinition:
			injectMetadata(&obj.(*extv1beta1.CustomResourceDefinition).ObjectMeta, version, registry)
			addCrd(resource)
		case *k8sv1.Service:
			injectMetadata(&obj.(*k8sv1.Service).ObjectMeta, version, registry)
			addService(resource)
		case *appsv1.Deployment:
			injectMetadata(&obj.(*appsv1.Deployment).ObjectMeta, version, registry)
			addDeployment(resource)
		case *appsv1.DaemonSet:
			injectMetadata(&obj.(*appsv1.DaemonSet).ObjectMeta, version, registry)
			addDaemonset(resource)
		case *admissionregistrationv1beta1.ValidatingWebhookConfiguration:
			injectMetadata(&obj.(*admissionregistrationv1beta1.ValidatingWebhookConfiguration).ObjectMeta, version, registry)
			addValidatingWebhook(resource)
		case *batchv1.Job:
			injectMetadata(&obj.(*batchv1.Job).ObjectMeta, version, registry)
			addInstallStrategyJob(resource)
		case *k8sv1.ConfigMap:
			injectMetadata(&obj.(*k8sv1.ConfigMap).ObjectMeta, version, registry)
			addInstallStrategyConfigMap(resource)
		case *k8sv1.Pod:
			injectMetadata(&obj.(*k8sv1.Pod).ObjectMeta, version, registry)
			addPod(resource)
		default:
			Fail("unknown resource type")
		}
	}

	addInstallStrategy := func(imageTag string, imageRegistry string) {
		// install strategy config
		resource, _ := installstrategy.NewInstallStrategyConfigMap(NAMESPACE, imageTag, imageRegistry)

		resource.Name = fmt.Sprintf("%s-%s", resource.Name, rand.String(10))
		addResource(resource, imageTag, imageRegistry)
	}

	addPods := func(version string, registry string) {
		pullPolicy := "IfNotPresent"
		imagePullPolicy := k8sv1.PullPolicy(pullPolicy)
		verbosity := "2"

		// we need at least one active pod for
		// virt-api
		// virt-controller
		// virt-handler
		apiDeployment, _ := components.NewApiServerDeployment(NAMESPACE, registry, version, imagePullPolicy, verbosity)

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
		injectMetadata(&pod.ObjectMeta, version, registry)
		pod.Name = "virt-api-xxxx"
		addPod(pod)

		controller, _ := components.NewControllerDeployment(NAMESPACE, registry, version, imagePullPolicy, verbosity)
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
		injectMetadata(&pod.ObjectMeta, version, registry)
		addPod(pod)

		handler, _ := components.NewHandlerDaemonSet(NAMESPACE, registry, version, imagePullPolicy, verbosity)
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
		injectMetadata(&pod.ObjectMeta, version, registry)
		pod.Name = "virt-handler-xxxx"
		addPod(pod)
	}

	generateRandomResources := func() int {
		version := fmt.Sprintf("rand-%s", rand.String(10))
		registry := fmt.Sprintf("rand-%s", rand.String(10))

		all := make([]interface{}, 0)
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
		all = append(all, &extv1beta1.CustomResourceDefinition{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apiextensions.k8s.io/v1beta1",
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
		for _, obj := range all {

			if resource, ok := obj.(runtime.Object); ok {
				addResource(resource, version, registry)
			} else {
				Fail("could not cast to runtime.Object")
			}
		}
		return len(all)
	}

	addDummyValidationWebhook := func() {
		version := fmt.Sprintf("rand-%s", rand.String(10))
		registry := fmt.Sprintf("rand-%s", rand.String(10))

		validationWebhook := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "virt-operator-tmp-webhook",
			},
		}

		injectMetadata(&validationWebhook.ObjectMeta, version, registry)
		addValidatingWebhook(validationWebhook)
	}

	addAll := func(version string, registry string) {
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
		apiDeployment, _ := components.NewApiServerDeployment(NAMESPACE, registry, version, imagePullPolicy, verbosity)
		controller, _ := components.NewControllerDeployment(NAMESPACE, registry, version, imagePullPolicy, verbosity)
		handler, _ := components.NewHandlerDaemonSet(NAMESPACE, registry, version, imagePullPolicy, verbosity)
		all = append(all, apiDeployment, controller, handler)

		for _, obj := range all {

			if resource, ok := obj.(runtime.Object); ok {
				addResource(resource, version, registry)
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

	deleteValidationWebhook := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.ValidationWebhook.GetStore().GetByKey(key); exists {
			validatingWebhookSource.Delete(obj.(runtime.Object))
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

	deleteInstallStrategyConfigMap := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.InstallStrategyConfigMap.GetStore().GetByKey(key); exists {
			installStrategyConfigMapSource.Delete(obj.(runtime.Object))
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
		case "jobs":
			deleteInstallStrategyJob(key)
		case "configmaps":
			deleteInstallStrategyConfigMap(key)
		default:
			Fail("unknown resource type")
		}
	}

	genericUpdateFunc := func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		update, ok := action.(testing.UpdateAction)
		Expect(ok).To(BeTrue())
		totalUpdates++
		return true, update.GetObject(), nil
	}

	genericPatchFunc := func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		_, ok := action.(testing.PatchAction)
		Expect(ok).To(BeTrue())
		totalPatches++

		return true, nil, nil
	}

	genericCreateFunc := func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())
		totalAdds++
		if addToCache {
			addResource(create.GetObject(), "", "")
		}
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
		if deleteFromCache {
			deleteResource(delete.GetResource().Resource, key)
		}
		return true, nil, nil
	}
	expectUsersDeleted := func(userBytes []byte) {
		deletePatch := `[ { "op": "test", "path": "/users", "value": ["someUser","system:serviceaccount:kubevirt-test:kubevirt-handler","system:serviceaccount:kubevirt-test:kubevirt-apiserver","system:serviceaccount:kubevirt-test:kubevirt-controller"] }, { "op": "replace", "path": "/users", "value": ["someUser"] } ]`
		Expect(userBytes).To(Equal([]byte(deletePatch)))
	}
	expectUsersAdded := func(userBytes []byte) {
		addPatch := `[ { "op": "test", "path": "/users", "value": ["someUser"] }, { "op": "replace", "path": "/users", "value": ["someUser","system:serviceaccount:kubevirt-test:kubevirt-handler","system:serviceaccount:kubevirt-test:kubevirt-apiserver","system:serviceaccount:kubevirt-test:kubevirt-controller"] } ]`
		Expect(userBytes).To(Equal([]byte(addPatch)))
	}

	shouldExpectInstallStrategyDeletion := func() {
		kubeClient.Fake.PrependReactor("delete", "configmaps", func(action testing.Action) (handled bool, obj runtime.Object, err error) {

			delete, ok := action.(testing.DeleteAction)
			Expect(ok).To(BeTrue())
			var key string
			if len(delete.GetNamespace()) > 0 {
				key = delete.GetNamespace() + "/"
			}
			key += delete.GetName()
			deleteResource(delete.GetResource().Resource, key)
			return true, nil, nil
		})
	}

	shouldExpectDeletions := func() {
		kubeClient.Fake.PrependReactor("delete", "serviceaccounts", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "clusterroles", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "clusterrolebindings", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "roles", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "rolebindings", genericDeleteFunc)

		secClient.Fake.PrependReactor("patch", "securitycontextconstraints", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			patch, _ := action.(testing.PatchAction)
			expectUsersDeleted(patch.GetPatch())
			return true, nil, nil
		})
		extClient.Fake.PrependReactor("delete", "customresourcedefinitions", genericDeleteFunc)

		kubeClient.Fake.PrependReactor("delete", "services", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "deployments", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "daemonsets", genericDeleteFunc)
		kubeClient.Fake.PrependReactor("delete", "validatingwebhookconfigurations", genericDeleteFunc)
	}

	shouldExpectJobDeletion := func() {
		kubeClient.Fake.PrependReactor("delete", "jobs", genericDeleteFunc)
	}

	shouldExpectJobCreation := func() {
		kubeClient.Fake.PrependReactor("create", "jobs", genericCreateFunc)
	}

	shouldExpectPatchesAndUpdates := func() {
		extClient.Fake.PrependReactor("patch", "customresourcedefinitions", genericPatchFunc)
		kubeClient.Fake.PrependReactor("patch", "serviceaccounts", genericPatchFunc)
		kubeClient.Fake.PrependReactor("update", "clusterroles", genericUpdateFunc)
		kubeClient.Fake.PrependReactor("update", "clusterrolebindings", genericUpdateFunc)
		kubeClient.Fake.PrependReactor("update", "roles", genericUpdateFunc)
		kubeClient.Fake.PrependReactor("update", "rolebindings", genericUpdateFunc)

		kubeClient.Fake.PrependReactor("patch", "services", genericPatchFunc)
		kubeClient.Fake.PrependReactor("patch", "daemonsets", genericPatchFunc)
		kubeClient.Fake.PrependReactor("patch", "deployments", genericPatchFunc)
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

		secClient.Fake.PrependReactor("patch", "securitycontextconstraints", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			patch, _ := action.(testing.PatchAction)
			expectUsersAdded(patch.GetPatch())
			return true, nil, nil
		})
		extClient.Fake.PrependReactor("create", "customresourcedefinitions", genericCreateFunc)

		kubeClient.Fake.PrependReactor("create", "services", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "deployments", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "daemonsets", genericCreateFunc)
		kubeClient.Fake.PrependReactor("create", "validatingwebhookconfigurations", genericCreateFunc)
	}

	shouldExpectKubeVirtUpdate := func(times int) {
		update := kvInterface.EXPECT().Update(gomock.Any())
		update.Do(func(kv *v1.KubeVirt) {
			kvInformer.GetStore().Update(kv)
			update.Return(kv, nil)
		}).Times(times)
	}

	shouldExpectKubeVirtUpdateVersion := func(times int, imageTag string) {
		update := kvInterface.EXPECT().Update(gomock.Any())
		update.Do(func(kv *v1.KubeVirt) {

			Expect(kv.Status.TargetKubeVirtVersion).To(Equal(imageTag))
			Expect(kv.Status.ObservedKubeVirtVersion).To(Equal(imageTag))
			kvInformer.GetStore().Update(kv)
			update.Return(kv, nil)
		}).Times(times)
	}

	shouldExpectKubeVirtUpdateFailureCondition := func(reason string) {
		update := kvInterface.EXPECT().Update(gomock.Any())
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

	Context("On valid KubeVirt object", func() {
		It("should delete install strategy configmap once kubevirt install is deleted", func(done Done) {
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

			shouldExpectInstallStrategyDeletion()

			addKubeVirt(kv)
			addInstallStrategy(defaultImageTag, defaultRegistry)
			controller.Execute()
		}, 15)

		It("should observe custom image tag in status during deploy", func(done Done) {
			defer close(done)

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Spec: v1.KubeVirtSpec{
					ImageTag: "custom.tag",
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
					OperatorVersion: version.Get().String(),
				},
			}

			// create all resources which should already exist
			addKubeVirt(kv)
			addAll("custom.tag", defaultRegistry)
			// install strategy config
			addInstallStrategy("custom.tag", defaultRegistry)
			addPods("custom.tag", defaultRegistry)

			makeApiAndControllerReady()
			makeHandlerReady()

			shouldExpectKubeVirtUpdateVersion(1, "custom.tag")
			controller.Execute()

		}, 15)

		It("delete temporary validation webhook once virt-api is deployed", func(done Done) {
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
					OperatorVersion:          version.Get().String(),
					TargetKubeVirtVersion:    defaultImageTag,
					TargetKubeVirtRegistry:   defaultRegistry,
					ObservedKubeVirtVersion:  defaultImageTag,
					ObservedKubeVirtRegistry: defaultRegistry,
				},
			}

			deleteFromCache = false

			// create all resources which should already exist
			addKubeVirt(kv)
			addDummyValidationWebhook()
			addInstallStrategy(defaultImageTag, defaultRegistry)
			addAll(defaultImageTag, defaultRegistry)
			addPods(defaultImageTag, defaultRegistry)
			makeApiAndControllerReady()
			makeHandlerReady()

			shouldExpectDeletions()

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
					OperatorVersion:          version.Get().String(),
					TargetKubeVirtVersion:    defaultImageTag,
					TargetKubeVirtRegistry:   defaultRegistry,
					ObservedKubeVirtVersion:  defaultImageTag,
					ObservedKubeVirtRegistry: defaultRegistry,
				},
			}

			// create all resources which should already exist
			addKubeVirt(kv)
			addInstallStrategy(defaultImageTag, defaultRegistry)
			addAll(defaultImageTag, defaultRegistry)
			addPods(defaultImageTag, defaultRegistry)
			makeApiAndControllerReady()
			makeHandlerReady()

			controller.Execute()

		}, 15)

		It("should delete operator managed resources not in the deployed installstrategy", func(done Done) {
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
					},
					OperatorVersion:          version.Get().String(),
					TargetKubeVirtVersion:    defaultImageTag,
					TargetKubeVirtRegistry:   defaultRegistry,
					ObservedKubeVirtVersion:  defaultImageTag,
					ObservedKubeVirtRegistry: defaultRegistry,
				},
			}

			deleteFromCache = false

			// create all resources which should already exist
			addKubeVirt(kv)
			addInstallStrategy(defaultImageTag, defaultRegistry)
			addAll(defaultImageTag, defaultRegistry)
			numResources := generateRandomResources()
			addPods(defaultImageTag, defaultRegistry)

			makeApiAndControllerReady()
			makeHandlerReady()

			shouldExpectDeletions()

			controller.Execute()
			Expect(totalDeletions).To(Equal(numResources))
		}, 15)

		It("should fail if KubeVirt object already exists", func(done Done) {
			defer close(done)

			kv1 := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install-1",
					Namespace:  NAMESPACE,
					UID:        "11111111111",
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

			kv2 := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-install-2",
					Namespace: NAMESPACE,
					UID:       "123123123",
				},
				Status: v1.KubeVirtStatus{},
			}

			addKubeVirt(kv1)
			addKubeVirt(kv2)

			shouldExpectKubeVirtUpdateFailureCondition(ConditionReasonDeploymentFailedExisting)

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
					OperatorVersion:          version.Get().String(),
					TargetKubeVirtVersion:    defaultImageTag,
					TargetKubeVirtRegistry:   defaultRegistry,
					ObservedKubeVirtVersion:  defaultImageTag,
					ObservedKubeVirtRegistry: defaultRegistry,
				},
			}

			// create all resources which should already exist
			addKubeVirt(kv)
			addInstallStrategy(defaultImageTag, defaultRegistry)

			shouldExpectKubeVirtUpdate(1)
			shouldExpectJobCreation()
			controller.Execute()

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
			addKubeVirt(kv)
			shouldExpectKubeVirtUpdate(1)
			shouldExpectJobCreation()
			controller.Execute()

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

			job := controller.generateInstallStrategyJob(kv)

			// will only create a new job after 10 seconds has passed.
			// this is just a simple mechanism to prevent spin loops
			// in the event that jobs are fast failing for some unknown reason.
			completionTime := time.Now().Add(time.Duration(-10) * time.Second)
			job.Status.CompletionTime = &metav1.Time{Time: completionTime}

			// create all resources which should already exist
			addKubeVirt(kv)
			addInstallStrategyJob(job)

			shouldExpectJobDeletion()
			shouldExpectKubeVirtUpdate(1)

			controller.Execute()

		}, 15)

		It("should not delete completed install strategy creation job if job has failed less that 10 seconds ago", func(done Done) {
			defer close(done)

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Status: v1.KubeVirtStatus{},
			}

			job := controller.generateInstallStrategyJob(kv)

			job.Status.CompletionTime = now()

			// create all resources which should already exist
			addKubeVirt(kv)
			addInstallStrategyJob(job)

			shouldExpectKubeVirtUpdate(1)

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
			addInstallStrategy(defaultImageTag, defaultRegistry)

			job := controller.generateInstallStrategyJob(kv)

			job.Status.CompletionTime = now()
			addInstallStrategyJob(job)

			// ensure completed jobs are garbage collected once install strategy
			// is loaded
			deleteFromCache = false
			shouldExpectJobDeletion()
			shouldExpectKubeVirtUpdate(1)
			shouldExpectCreations()

			controller.Execute()

			kv = getLatestKubeVirt(kv)
			Expect(kv.Status.Phase).To(Equal(v1.KubeVirtPhaseDeploying))
			Expect(len(kv.Status.Conditions)).To(Equal(0))

			// 2 because waiting on controller and virt-handler daemonset until API server deploys successfully
			expectedUncreatedResources := 2

			// 1 because a temporary validation webhook is created to block new CRDs until api server is deployed
			expectedTemporaryResources := 1

			Expect(totalAdds).To(Equal(resourceCount - expectedUncreatedResources + expectedTemporaryResources))
			Expect(len(controller.stores.ServiceAccountCache.List())).To(Equal(3))
			Expect(len(controller.stores.ClusterRoleCache.List())).To(Equal(7))
			Expect(len(controller.stores.ClusterRoleBindingCache.List())).To(Equal(5))
			Expect(len(controller.stores.RoleCache.List())).To(Equal(2))
			Expect(len(controller.stores.RoleBindingCache.List())).To(Equal(2))
			Expect(len(controller.stores.CrdCache.List())).To(Equal(5))
			Expect(len(controller.stores.ServiceCache.List())).To(Equal(2))
			Expect(len(controller.stores.DeploymentCache.List())).To(Equal(1))
			Expect(len(controller.stores.DaemonSetCache.List())).To(Equal(0))
			Expect(len(controller.stores.ValidationWebhookCache.List())).To(Equal(1))
		}, 15)

		It("should pause rollback until api server is rolled over.", func(done Done) {
			defer close(done)

			rollbackVersion := "9.9.7"
			rollbackRegistry := "otherregistry"

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Spec: v1.KubeVirtSpec{
					ImageTag:      rollbackVersion,
					ImageRegistry: rollbackRegistry,
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
					OperatorVersion:          version.Get().String(),
					TargetKubeVirtVersion:    defaultImageTag,
					TargetKubeVirtRegistry:   defaultRegistry,
					ObservedKubeVirtVersion:  defaultImageTag,
					ObservedKubeVirtRegistry: defaultRegistry,
				},
			}

			// create all resources which should already exist
			addKubeVirt(kv)
			addInstallStrategy(defaultImageTag, defaultRegistry)
			addInstallStrategy(rollbackVersion, rollbackRegistry)

			addAll(defaultImageTag, defaultRegistry)
			addPods(defaultImageTag, defaultRegistry)

			makeApiAndControllerReady()
			makeHandlerReady()

			addToCache = false
			shouldExpectRbacBackupCreations()
			shouldExpectPatchesAndUpdates()
			shouldExpectKubeVirtUpdate(1)

			controller.Execute()

			// on rollback or create, api server must be online first before controllers and daemonset.
			// On rollback this prevents someone from posting invalid specs to
			// the cluster from newer versions when an older version is being deployed.
			// On create this prevents invalid specs from entering the cluster
			// while controllers are available to process them.
			Expect(totalPatches).To(Equal(patchCount - 2))
			Expect(totalUpdates).To(Equal(updateCount))
		}, 15)

		It("should pause update until daemonsets and controllers are rolled over.", func(done Done) {
			defer close(done)

			updatedVersion := "9.9.10"
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
					OperatorVersion:          version.Get().String(),
					TargetKubeVirtVersion:    defaultImageTag,
					TargetKubeVirtRegistry:   defaultRegistry,
					ObservedKubeVirtVersion:  defaultImageTag,
					ObservedKubeVirtRegistry: defaultRegistry,
				},
			}

			// create all resources which should already exist
			addKubeVirt(kv)
			addInstallStrategy(defaultImageTag, defaultRegistry)
			addInstallStrategy(updatedVersion, updatedRegistry)

			addAll(defaultImageTag, defaultRegistry)
			addPods(defaultImageTag, defaultRegistry)

			makeApiAndControllerReady()
			makeHandlerReady()

			addToCache = false
			shouldExpectRbacBackupCreations()
			shouldExpectPatchesAndUpdates()
			shouldExpectKubeVirtUpdate(1)

			controller.Execute()

			// on update, apiserver won't get patched until daemonset and controller pods are online.
			// this prevents the new API from coming online until the controllers can manage it.
			Expect(totalPatches).To(Equal(patchCount - 1))
			Expect(totalUpdates).To(Equal(updateCount))
		}, 15)

		It("should update kubevirt resources when Operator version changes if no imageTag and imageRegistry is explicilty set.", func(done Done) {
			defer close(done)

			updatedVersion := "1.1.1"
			updatedRegistry := "otherregistry"

			os.Setenv(util.OperatorImageEnvName, fmt.Sprintf("%s/virt-operator:%s", updatedRegistry, updatedVersion))

			controller.config = util.GetConfig()

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-install",
					Namespace:  NAMESPACE,
					Finalizers: []string{util.KubeVirtFinalizer},
				},
				Spec: v1.KubeVirtSpec{},
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
					OperatorVersion:          version.Get().String(),
					TargetKubeVirtVersion:    defaultImageTag,
					TargetKubeVirtRegistry:   defaultRegistry,
					ObservedKubeVirtVersion:  defaultImageTag,
					ObservedKubeVirtRegistry: defaultRegistry,
				},
			}

			// create all resources which should already exist
			addKubeVirt(kv)
			addInstallStrategy(defaultImageTag, defaultRegistry)
			addInstallStrategy(updatedVersion, updatedRegistry)

			addAll(defaultImageTag, defaultRegistry)
			addPods(defaultImageTag, defaultRegistry)

			// pods for the new version are added so this test won't
			// wait for daemonsets to rollover before updating/patching
			// all resources.
			addPods(updatedVersion, updatedRegistry)

			makeApiAndControllerReady()
			makeHandlerReady()

			shouldExpectPatchesAndUpdates()
			shouldExpectKubeVirtUpdate(1)

			controller.Execute()

			Expect(totalPatches).To(Equal(patchCount))
			Expect(totalUpdates).To(Equal(updateCount))

			// ensure every resource is either patched or updated
			Expect(totalUpdates + totalPatches).To(Equal(resourceCount))

		}, 15)

		It("should update resources when changing KubeVirt version.", func(done Done) {
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
					OperatorVersion:          version.Get().String(),
					TargetKubeVirtVersion:    defaultImageTag,
					TargetKubeVirtRegistry:   defaultRegistry,
					ObservedKubeVirtVersion:  defaultImageTag,
					ObservedKubeVirtRegistry: defaultRegistry,
				},
			}

			// create all resources which should already exist
			addKubeVirt(kv)
			addInstallStrategy(defaultImageTag, defaultRegistry)
			addInstallStrategy(updatedVersion, updatedRegistry)

			addAll(defaultImageTag, defaultRegistry)
			addPods(defaultImageTag, defaultRegistry)

			// pods for the new version are added so this test won't
			// wait for daemonsets to rollover before updating/patching
			// all resources.
			addPods(updatedVersion, updatedRegistry)

			makeApiAndControllerReady()
			makeHandlerReady()

			shouldExpectPatchesAndUpdates()
			shouldExpectKubeVirtUpdate(1)

			controller.Execute()

			Expect(totalPatches).To(Equal(patchCount))
			Expect(totalUpdates).To(Equal(updateCount))

			// ensure every resource is either patched or updated
			Expect(totalUpdates + totalPatches).To(Equal(resourceCount))

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
			addInstallStrategy(defaultImageTag, defaultRegistry)
			addAll(defaultImageTag, defaultRegistry)

			shouldExpectKubeVirtUpdate(1)
			shouldExpectDeletions()
			shouldExpectInstallStrategyDeletion()

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
	Context("On install strategy dump", func() {
		It("should generate latest install strategy and post as config map", func(done Done) {
			defer close(done)

			kubeClient.Fake.PrependReactor("create", "configmaps", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				create, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())

				configMap := create.GetObject().(*k8sv1.ConfigMap)
				Expect(configMap.GenerateName).To(Equal("kubevirt-install-strategy-"))

				version, ok := configMap.ObjectMeta.Annotations[v1.InstallStrategyVersionAnnotation]
				Expect(ok).To(BeTrue())

				Expect(version).To(Equal(defaultImageTag))

				registry, ok := configMap.ObjectMeta.Annotations[v1.InstallStrategyRegistryAnnotation]
				Expect(registry).To(Equal(defaultRegistry))
				Expect(ok).To(BeTrue())

				_, ok = configMap.Data["manifests"]
				Expect(ok).To(BeTrue())

				return true, create.GetObject(), nil
			})

			// This generates and posts the install strategy config map
			installstrategy.DumpInstallStrategyToConfigMap(virtClient)
		}, 15)
	})
})

func now() *metav1.Time {
	now := metav1.Now()
	return &now
}
