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
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	secv1 "github.com/openshift/api/security/v1"
	secv1fake "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1/fake"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	k8sv1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
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

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/version"
	kubecontroller "kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/rbac"
	installstrategy "kubevirt.io/kubevirt/pkg/virt-operator/install-strategy"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

const (
	Added   = "added"
	Updated = "updated"
	Patched = "patched"
	Deleted = "deleted"
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
	var podDisruptionBudgetSource *framework.FakeControllerSource

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

	resourceCount := 31
	patchCount := 15
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
		go informers.PodDisruptionBudget.Run(stop)

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
		cache.WaitForCacheSync(stop, informers.PodDisruptionBudget.HasSynced)
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

		informers.PodDisruptionBudget, podDisruptionBudgetSource = testutils.NewFakeInformerFor(&policyv1beta1.PodDisruptionBudget{})
		stores.PodDisruptionBudgetCache = informers.PodDisruptionBudget.GetStore()

		controller = NewKubeVirtController(virtClient, kvInformer, recorder, stores, informers, NAMESPACE)

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
		virtClient.EXPECT().PolicyV1beta1().Return(kubeClient.PolicyV1beta1()).AnyTimes()

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

	injectMetadata := func(objectMeta *metav1.ObjectMeta, config *util.KubeVirtDeploymentConfig) {

		if config == nil {
			return
		}
		if objectMeta.Labels == nil {
			objectMeta.Labels = make(map[string]string)
		}
		objectMeta.Labels[v1.ManagedByLabel] = v1.ManagedByLabelOperatorValue

		if objectMeta.Annotations == nil {
			objectMeta.Annotations = make(map[string]string)
		}
		objectMeta.Annotations[v1.InstallStrategyVersionAnnotation] = config.GetKubeVirtVersion()
		objectMeta.Annotations[v1.InstallStrategyRegistryAnnotation] = config.GetImageRegistry()
		objectMeta.Annotations[v1.InstallStrategyIdentifierAnnotation] = config.GetDeploymentID()
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

	addPodDisruptionBudget := func(podDisruptionBudget *policyv1beta1.PodDisruptionBudget) {
		mockQueue.ExpectAdds(1)
		podDisruptionBudgetSource.Add(podDisruptionBudget)
		mockQueue.Wait()
	}

	addResource := func(obj runtime.Object, config *util.KubeVirtDeploymentConfig) {
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
		case *extv1beta1.CustomResourceDefinition:
			injectMetadata(&obj.(*extv1beta1.CustomResourceDefinition).ObjectMeta, config)
			addCrd(resource)
		case *k8sv1.Service:
			injectMetadata(&obj.(*k8sv1.Service).ObjectMeta, config)
			addService(resource)
		case *appsv1.Deployment:
			injectMetadata(&obj.(*appsv1.Deployment).ObjectMeta, config)
			addDeployment(resource)
		case *appsv1.DaemonSet:
			injectMetadata(&obj.(*appsv1.DaemonSet).ObjectMeta, config)
			addDaemonset(resource)
		case *admissionregistrationv1beta1.ValidatingWebhookConfiguration:
			injectMetadata(&obj.(*admissionregistrationv1beta1.ValidatingWebhookConfiguration).ObjectMeta, config)
			addValidatingWebhook(resource)
		case *batchv1.Job:
			injectMetadata(&obj.(*batchv1.Job).ObjectMeta, config)
			addInstallStrategyJob(resource)
		case *k8sv1.ConfigMap:
			injectMetadata(&obj.(*k8sv1.ConfigMap).ObjectMeta, config)
			addInstallStrategyConfigMap(resource)
		case *k8sv1.Pod:
			injectMetadata(&obj.(*k8sv1.Pod).ObjectMeta, config)
			addPod(resource)
		case *policyv1beta1.PodDisruptionBudget:
			injectMetadata(&obj.(*policyv1beta1.PodDisruptionBudget).ObjectMeta, config)
			addPodDisruptionBudget(resource)
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
		resource, _ := installstrategy.NewInstallStrategyConfigMap(config)

		resource.Name = fmt.Sprintf("%s-%s", resource.Name, rand.String(10))
		addResource(resource, config)
	}

	addPodDisruptionBudgets := func(config *util.KubeVirtDeploymentConfig, apiDeployment *appsv1.Deployment, controller *appsv1.Deployment) {
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
		addPodDisruptionBudget(apiPodDisruptionBudget)
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
		addPodDisruptionBudget(controllerPodDisruptionBudget)
	}

	addPodsWithOptionalPodDisruptionBudgets := func(config *util.KubeVirtDeploymentConfig, shouldAddPodDisruptionBudgets bool) {
		// we need at least one active pod for
		// virt-api
		// virt-controller
		// virt-handler
		apiDeployment, _ := components.NewApiServerDeployment(NAMESPACE, config.GetImageRegistry(), config.GetApiVersion(), config.GetImagePullPolicy(), config.GetVerbosity())

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

		controller, _ := components.NewControllerDeployment(NAMESPACE, config.GetImageRegistry(), config.GetControllerVersion(), config.GetLauncherVersion(), config.GetImagePullPolicy(), config.GetVerbosity())
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
		injectMetadata(&pod.ObjectMeta, config)
		addPod(pod)

		handler, _ := components.NewHandlerDaemonSet(NAMESPACE, config.GetImageRegistry(), config.GetHandlerVersion(), config.GetImagePullPolicy(), config.GetVerbosity())
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
		injectMetadata(&pod.ObjectMeta, config)
		pod.Name = "virt-handler-xxxx"
		addPod(pod)

		if shouldAddPodDisruptionBudgets {
			addPodDisruptionBudgets(config, apiDeployment, controller)
		}
	}

	addPodsAndPodDisruptionBudgets := func(config *util.KubeVirtDeploymentConfig) {
		addPodsWithOptionalPodDisruptionBudgets(config, true)
	}

	generateRandomResources := func() int {
		version := fmt.Sprintf("rand-%s", rand.String(10))
		registry := fmt.Sprintf("rand-%s", rand.String(10))
		config := getConfig(registry, version)

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
				addResource(resource, config)
			} else {
				Fail("could not cast to runtime.Object")
			}
		}
		return len(all)
	}

	addDummyValidationWebhook := func() {
		version := fmt.Sprintf("rand-%s", rand.String(10))
		registry := fmt.Sprintf("rand-%s", rand.String(10))
		config := getConfig(registry, version)

		validationWebhook := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "virt-operator-tmp-webhook",
			},
		}

		injectMetadata(&validationWebhook.ObjectMeta, config)
		addValidatingWebhook(validationWebhook)
	}

	addAll := func(config *util.KubeVirtDeploymentConfig) {
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
		apiDeployment, _ := components.NewApiServerDeployment(NAMESPACE, config.GetImageRegistry(), config.GetApiVersion(), config.GetImagePullPolicy(), config.GetVerbosity())
		apiDeploymentPdb := components.NewPodDisruptionBudgetForDeployment(apiDeployment)
		injectVersionAnnotation(apiDeploymentPdb, config.GetApiVersion(), config.GetImageRegistry())
		controller, _ := components.NewControllerDeployment(NAMESPACE, config.GetImageRegistry(), config.GetControllerVersion(), config.GetLauncherVersion(), config.GetImagePullPolicy(), config.GetVerbosity())
		controllerPdb := components.NewPodDisruptionBudgetForDeployment(controller)
		injectVersionAnnotation(controllerPdb, config.GetApiVersion(), config.GetImageRegistry())
		handler, _ := components.NewHandlerDaemonSet(NAMESPACE, config.GetImageRegistry(), config.GetApiVersion(), config.GetImagePullPolicy(), config.GetVerbosity())
		all = append(all, apiDeployment, apiDeploymentPdb, controller, controllerPdb, handler)

		for _, obj := range all {

			if resource, ok := obj.(runtime.Object); ok {
				addResource(resource, config)
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

	deletePodDisruptionBudget := func(key string) {
		mockQueue.ExpectAdds(1)
		if obj, exists, _ := informers.PodDisruptionBudget.GetStore().GetByKey(key); exists {
			podDisruptionBudgetSource.Delete(obj.(runtime.Object))
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
		case "poddisruptionbudgets":
			deletePodDisruptionBudget(key)
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
		Expect(ok).To(BeTrue())
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

	genericCreateFunc := func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())
		totalAdds++
		if addToCache {
			addResource(create.GetObject(), nil)
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

			deleted, ok := action.(testing.DeleteAction)
			Expect(ok).To(BeTrue())
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
		kubeClient.Fake.PrependReactor("delete", "poddisruptionbudgets", genericDeleteFunc)
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
		kubeClient.Fake.PrependReactor("patch", "poddisruptionbudgets", genericPatchFunc)
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
		kubeClient.Fake.PrependReactor("create", "poddisruptionbudgets", genericCreateFunc)
	}

	shouldExpectKubeVirtUpdate := func(times int) {
		update := kvInterface.EXPECT().Update(gomock.Any())
		update.Do(func(kv *v1.KubeVirt) {
			kvInformer.GetStore().Update(kv)
			update.Return(kv, nil)
		}).Times(times)
	}

	shouldExpectKubeVirtUpdateVersion := func(times int, config *util.KubeVirtDeploymentConfig) {
		update := kvInterface.EXPECT().Update(gomock.Any())
		update.Do(func(kv *v1.KubeVirt) {

			Expect(kv.Status.TargetKubeVirtVersion).To(Equal(config.GetKubeVirtVersion()))
			Expect(kv.Status.ObservedKubeVirtVersion).To(Equal(config.GetKubeVirtVersion()))
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

			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)
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
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			customConfig := getConfig(defaultConfig.GetImageRegistry(), "custom.tag")

			addAll(customConfig)
			// install strategy config
			addInstallStrategy(customConfig)
			addPodsAndPodDisruptionBudgets(customConfig)

			makeApiAndControllerReady()
			makeHandlerReady()

			shouldExpectKubeVirtUpdateVersion(1, customConfig)
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
					OperatorVersion: version.Get().String(),
				},
			}
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)

			deleteFromCache = false

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addDummyValidationWebhook()
			addInstallStrategy(defaultConfig)
			addAll(defaultConfig)
			addPodsAndPodDisruptionBudgets(defaultConfig)
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
					OperatorVersion: version.Get().String(),
				},
			}
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)
			addAll(defaultConfig)
			addPodsAndPodDisruptionBudgets(defaultConfig)
			makeApiAndControllerReady()
			makeHandlerReady()

			controller.Execute()

		}, 15)

		It("should delete operator managed resources not in the deployed installstrategy", func() {
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
					OperatorVersion: version.Get().String(),
				},
			}
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)

			deleteFromCache = false

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)
			addAll(defaultConfig)
			numResources := generateRandomResources()
			addPodsAndPodDisruptionBudgets(defaultConfig)

			makeApiAndControllerReady()
			makeHandlerReady()

			shouldExpectDeletions()

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

			kubecontroller.SetLatestApiVersionAnnotation(kv1)
			addKubeVirt(kv1)
			kubecontroller.SetLatestApiVersionAnnotation(kv2)
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
					OperatorVersion: version.Get().String(),
				},
			}
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)

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
			kubecontroller.SetLatestApiVersionAnnotation(kv)
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
			shouldExpectKubeVirtUpdate(1)

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

			shouldExpectKubeVirtUpdate(1)

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
			shouldExpectCreations()

			controller.Execute()

			kv = getLatestKubeVirt(kv)
			Expect(kv.Status.Phase).To(Equal(v1.KubeVirtPhaseDeploying))
			Expect(len(kv.Status.Conditions)).To(Equal(0))

			// 3, 2 because waiting on controller and virt-handler daemonset until API server deploys successfully
			// 	and
			// 1 because uncreated PDB for virt-controller
			expectedUncreatedResources := 3

			// 1 because a temporary validation webhook is created to block new CRDs until api server is deployed
			expectedTemporaryResources := 1

			Expect(totalAdds).To(Equal(resourceCount - expectedUncreatedResources + expectedTemporaryResources))
			//+ expectedPDBsCreated))

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
			Expect(len(controller.stores.PodDisruptionBudgetCache.List())).To(Equal(1))

			Expect(resourceChanges["poddisruptionbudgets"][Added]).To(Equal(1))

		}, 15)

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
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)
			addInstallStrategy(rollbackConfig)

			addAll(defaultConfig)
			addPodsAndPodDisruptionBudgets(defaultConfig)

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

			// 3 because 2 for virt-controller service and deployment
			// and
			// 1 because of the pdb of virt-controller
			Expect(totalPatches).To(Equal(patchCount - 3))
			Expect(totalUpdates).To(Equal(updateCount))

			Expect(resourceChanges["poddisruptionbudgets"][Patched]).To(Equal(1))
		}, 15)

		It("should pause update until daemonsets and controllers are rolled over.", func(done Done) {
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
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)
			addInstallStrategy(updatedConfig)

			addAll(defaultConfig)
			addPodsAndPodDisruptionBudgets(defaultConfig)

			makeApiAndControllerReady()
			makeHandlerReady()

			addToCache = false
			shouldExpectRbacBackupCreations()
			shouldExpectPatchesAndUpdates()
			shouldExpectKubeVirtUpdate(1)

			controller.Execute()

			// on update, apiserver won't get patched until daemonset and controller pods are online.
			// this prevents the new API from coming online until the controllers can manage it.

			// 2 because virt-api and PDB are not updated
			Expect(totalPatches).To(Equal(patchCount - 2))
			Expect(totalUpdates).To(Equal(updateCount))

			Expect(resourceChanges["poddisruptionbudgets"][Patched]).To(Equal(1))
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
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)
			addInstallStrategy(updatedConfig)

			addAll(defaultConfig)
			addPodsAndPodDisruptionBudgets(defaultConfig)

			// pods for the new version are added so this test won't
			// wait for daemonsets to rollover before updating/patching
			// all resources.
			addPodsWithOptionalPodDisruptionBudgets(updatedConfig, false)

			makeApiAndControllerReady()
			makeHandlerReady()

			shouldExpectPatchesAndUpdates()
			shouldExpectKubeVirtUpdate(1)

			controller.Execute()

			Expect(totalPatches).To(Equal(patchCount))
			Expect(totalUpdates).To(Equal(updateCount))

			// ensure every resource is either patched or updated
			Expect(totalUpdates + totalPatches).To(Equal(resourceCount))

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
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)
			addInstallStrategy(updatedConfig)

			addAll(defaultConfig)
			addPodsAndPodDisruptionBudgets(defaultConfig)

			// pods for the new version are added so this test won't
			// wait for daemonsets to rollover before updating/patching
			// all resources.
			addPodsWithOptionalPodDisruptionBudgets(updatedConfig, false)

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
			defaultConfig.SetTargetDeploymentConfig(kv)
			defaultConfig.SetObservedDeploymentConfig(kv)

			// create all resources which should already exist
			kubecontroller.SetLatestApiVersionAnnotation(kv)
			addKubeVirt(kv)
			addInstallStrategy(defaultConfig)
			addInstallStrategy(updatedConfig)

			addAll(defaultConfig)
			addPodsAndPodDisruptionBudgets(defaultConfig)

			// pods for the new version are added so this test won't
			// wait for daemonsets to rollover before updating/patching
			// all resources.
			addPodsWithOptionalPodDisruptionBudgets(updatedConfig, false)

			makeApiAndControllerReady()
			makeHandlerReady()

			shouldExpectPatchesAndUpdates()
			shouldExpectKubeVirtUpdate(1)

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
			addAll(defaultConfig)

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
			addAll(defaultConfig)

			shouldExpectKubeVirtUpdate(1)
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
			installstrategy.DumpInstallStrategyToConfigMap(virtClient)
		}, 15)
	})
})

func injectVersionAnnotation(budget *policyv1beta1.PodDisruptionBudget, version string, registry string) {
	objectMeta := &budget.ObjectMeta
	if objectMeta.Annotations == nil {
		objectMeta.Annotations = make(map[string]string)
	}
	objectMeta.Annotations[v1.InstallStrategyVersionAnnotation] = version
	objectMeta.Annotations[v1.InstallStrategyRegistryAnnotation] = registry
}

func now() *metav1.Time {
	now := metav1.Now()
	return &now
}
