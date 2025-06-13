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

package fuzz

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/api/errors"

	"go.uber.org/mock/gomock"

	secv1 "github.com/openshift/api/security/v1"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	promclientfake "kubevirt.io/client-go/prometheusoperator/fake"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/apply"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	installstrategy "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	marshalutil "kubevirt.io/kubevirt/tools/util"

	routev1 "github.com/openshift/api/route/v1"
	k8sv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/util"

	fuzz "github.com/google/gofuzz"
)

const (
	Namespace = "ns"
	resources = map[int]string{
		0:  "Route",
		1:  "ServiceAccount",
		2:  "ClusterRole",
		3:  "ClusterRoleBinding",
		4:  "Role",
		5:  "RoleBinding",
		6:  "Service",
		7:  "Deployment",
		8:  "DaemonSet",
		9:  "ValidationWebhook",
		10: "MutatingWebhook",
		11: "APIService",
		12: "SCC",
		13: "InstallStrategyJob",
		14: "InfrastructurePod",
		15: "PodDisruptionBudget",
		16: "ServiceMonitor",
		17: "Namespace",
		18: "PrometheusRule",
		19: "Secret",
		20: "ConfigMap",
		21: "ValidatingAdmissionPolicyBinding",
		22: "ValidatingAdmissionPolicy",
	}
)

func createRandomizedObject(fdp *fuzz.Fuzzer, resourceType string) runtime.Object {
	switch resourceType {
	case "Route":
		obj := &routev1.Route{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: routev1.SchemeGroupVersion.String(),
			Kind:       "Route",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "ServiceAccount":
		obj := &k8sv1.ServiceAccount{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: k8sv1.SchemeGroupVersion.String(),
			Kind:       "ServiceAccount",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "ClusterRole":
		obj := &rbacv1.ClusterRole{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRole",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "ClusterRoleBinding":
		obj := &rbacv1.ClusterRoleBinding{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRoleBinding",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "Role":
		obj := &rbacv1.Role{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "Role",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "RoleBinding":
		obj := &rbacv1.RoleBinding{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "RoleBinding",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "Service":
		obj := &k8sv1.Service{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: k8sv1.SchemeGroupVersion.String(),
			Kind:       "Service",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "Deployment":
		obj := &appsv1.Deployment{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "DaemonSet":
		obj := &appsv1.DaemonSet{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: routev1.SchemeGroupVersion.String(),
			Kind:       "Route",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "ValidationWebhook":
		obj := &admissionregistrationv1.ValidatingWebhookConfiguration{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: admissionregistrationv1.SchemeGroupVersion.String(),
			Kind:       "ValidationWebhook",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "MutatingWebhook":
		obj := &admissionregistrationv1.MutatingWebhookConfiguration{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: admissionregistrationv1.SchemeGroupVersion.String(),
			Kind:       "MutatingWebhook",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "APIService":
		obj := &apiregv1.APIService{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: apiregv1.SchemeGroupVersion.String(),
			Kind:       "APIService",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "SCC":
		obj := &secv1.SecurityContextConstraints{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: secv1.SchemeGroupVersion.String(),
			Kind:       "SCC",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "InstallStrategyJob":
		obj := &batchv1.Job{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: batchv1.SchemeGroupVersion.String(),
			Kind:       "InstallStrategyJob",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "InfrastructurePod":
		obj := &k8sv1.Pod{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: k8sv1.SchemeGroupVersion.String(),
			Kind:       "InfrastructurePod",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "PodDisruptionBudget":
		obj := &policyv1.PodDisruptionBudget{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: policyv1.SchemeGroupVersion.String(),
			Kind:       "PodDisruptionBudget",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "ServiceMonitor":
		obj := &promv1.ServiceMonitor{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: routev1.SchemeGroupVersion.String(),
			Kind:       "Route",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "Namespace":
		obj := &k8sv1.Namespace{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: k8sv1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "PrometheusRule":
		obj := &promv1.PrometheusRule{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: promv1.SchemeGroupVersion.String(),
			Kind:       "PrometheusRule",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "Secret":
		obj := &k8sv1.Secret{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: k8sv1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "ConfigMap":
		obj := &k8sv1.ConfigMap{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: k8sv1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "ValidatingAdmissionPolicyBinding":
		obj := &admissionregistrationv1.ValidatingAdmissionPolicyBinding{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: admissionregistrationv1.SchemeGroupVersion.String(),
			Kind:       "ValidatingAdmissionPolicyBinding",
		}
		obj.TypeMeta = typeMeta
		return obj
	case "ValidatingAdmissionPolicy":
		obj := &admissionregistrationv1.ValidatingAdmissionPolicy{}
		fdp.Fuzz(obj)
		typeMeta := metav1.TypeMeta{
			APIVersion: admissionregistrationv1.SchemeGroupVersion.String(),
			Kind:       "ValidatingAdmissionPolicy",
		}
		obj.TypeMeta = typeMeta
		return obj
	default:
		// This should not happen. If it does, it is an indicator that
		// the fuzzer is not efficient, and we prefer to know about it
		// rather than letting the fuzzer run, hence the panic.
		panic(fmt.Sprintf("should not happen: '%s'", resourceType))
	}
	// This should not happen. If it does, it is an indicator that
	// the fuzzer is not efficient, and we prefer to know about it
	// rather than letting the fuzzer run, hence the panic.
	panic("should not happen")
}

func createManifests(fdp *fuzz.Fuzzer) ([]byte, error) {
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)
	createdResource := 0
	for range 10 {
		var add bool
		fdp.Fuzz(&add)
		if !add {
			continue
		}
		var resourceType uint8
		fdp.Fuzz(&resourceType)
		resourceTypeStr := resources[int(resourceType)%len(resources)]
		obj := createRandomizedObject(fdp, resourceTypeStr)
		err := marshalutil.MarshallObject(obj, writer)
		if err == nil {
			createdResource += 1
		}
	}
	if createdResource < 3 {
		return b.Bytes(), fmt.Errorf("Too few resources for efficient fuzzing")
	}
	writer.Flush()

	return b.Bytes(), nil
}

func loadTargetStrategyForFuzzing(resources []byte, config *util.KubeVirtDeploymentConfig, stores util.Stores) (*install.Strategy, error) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kubevirt-install-strategy-",
			Namespace:    config.GetNamespace(),
			Labels: map[string]string{
				v1.ManagedByLabel:       v1.ManagedByLabelOperatorValue,
				v1.InstallStrategyLabel: "",
			},
			Annotations: map[string]string{
				v1.InstallStrategyVersionAnnotation:    config.GetKubeVirtVersion(),
				v1.InstallStrategyRegistryAnnotation:   config.GetImageRegistry(),
				v1.InstallStrategyIdentifierAnnotation: config.GetDeploymentID(),
			},
		},
		Data: map[string]string{
			"manifests": string(resources),
		},
	}

	err := stores.InstallStrategyConfigMapCache.Add(configMap)
	if err != nil {
		return nil, err
	}
	targetStrategy, err := installstrategy.LoadInstallStrategyFromCache(stores, config)
	return targetStrategy, err
}

func FuzzReconciler(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte, callType uint8) {
		fdp := fuzz.NewFromGoFuzz(data)
		manifests, err := createManifests(fdp)
		if err != nil {
			return
		}

		config := getConfig("fake-registry", "v9.9.9")
		stores := util.Stores{}
		stores.InstallStrategyConfigMapCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		strat, err := loadTargetStrategyForFuzzing(manifests, config, stores)
		if err != nil {
			return
		}
		origQueue := workqueue.NewTypedRateLimitingQueue[string](workqueue.DefaultTypedControllerRateLimiter[string]())
		queue := testutils.NewMockWorkQueue(origQueue)

		// Set up the stores caches
		stores.RouteCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.ServiceAccountCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.ClusterRoleCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.ClusterRoleBindingCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.RoleCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.RoleBindingCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.ServiceCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.DeploymentCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.DaemonSetCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.ValidationWebhookCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.MutatingWebhookCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.APIServiceCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.SCCCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.InstallStrategyJobCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.InfrastructurePodCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.PodDisruptionBudgetCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.ServiceMonitorCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.NamespaceCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.PrometheusRuleCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.SecretCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.ConfigMapCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.ValidatingAdmissionPolicyBindingCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.ValidatingAdmissionPolicyCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.ClusterInstancetype = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.ClusterPreference = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)

		// Create at least 3 resources in the stores
		createdResource := 0
		for range 10 {
			var add bool
			fdp.Fuzz(&add)
			if !add {
				continue
			}
			var randUint8 uint8
			fdp.Fuzz(&randUint8)
			resourceType := resources[int(randUint8)%len(resources)]
			obj := createRandomizedObject(fdp, resourceType)
			switch resourceType {
			case "Route":
				stores.RouteCache.Add(obj)
			case "ServiceAccount":
				stores.ServiceAccountCache.Add(obj)
			case "ClusterRole":
				stores.ClusterRoleCache.Add(obj)
			case "ClusterRoleBinding":
				stores.ClusterRoleBindingCache.Add(obj)
			case "Role":
				stores.RoleCache.Add(obj)
			case "RoleBinding":
				stores.RoleBindingCache.Add(obj)
			case "Service":
				stores.ServiceCache.Add(obj)
			case "Deployment":
				stores.DeploymentCache.Add(obj)
			case "DaemonSet":
				stores.DaemonSetCache.Add(obj)
			case "ValidationWebhook":
				stores.ValidationWebhookCache.Add(obj)
			case "MutatingWebhook":
				stores.MutatingWebhookCache.Add(obj)
			case "APIService":
				stores.APIServiceCache.Add(obj)
			case "SCC":
				stores.SCCCache.Add(obj)
			case "InstallStrategyJob":
				stores.InstallStrategyJobCache.Add(obj)
			case "InfrastructurePod":
				stores.InfrastructurePodCache.Add(obj)
			case "PodDisruptionBudget":
				stores.PodDisruptionBudgetCache.Add(obj)
			case "ServiceMonitor":
				stores.ServiceMonitorCache.Add(obj)
			case "Namespace":
				stores.NamespaceCache.Add(obj)
			case "PrometheusRule":
				stores.PrometheusRuleCache.Add(obj)
			case "Secret":
				stores.SecretCache.Add(obj)
			case "ConfigMap":
				stores.ConfigMapCache.Add(obj)
			case "ValidatingAdmissionPolicyBinding":
				stores.ValidatingAdmissionPolicyBindingCache.Add(obj)
			case "ValidatingAdmissionPolicy":
				stores.ValidatingAdmissionPolicyCache.Add(obj)
			default:
				// This should not happen. If it does, it is an indicator that
				// the fuzzer is not efficient, and we prefer to know about it
				// rather than letting the fuzzer run, hence the panic.
				panic("should not happen")
			}
			key, err := controller.KeyFunc(obj)
			if err != nil {
				panic(err)
			}
			queue.Add(key)
			createdResource += 1
		}
		// Only proceed if we actually have resources.
		if createdResource == 0 {
			return
		}

		// Setting up the Kubevirt clients
		ctrl := gomock.NewController(t)
		clientset := kubecli.NewMockKubevirtClient(ctrl)
		kv := &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: Namespace,
			},
		}
		expectations := &util.Expectations{}
		expectations.DaemonSet = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("DaemonSet"))
		expectations.PodDisruptionBudget = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("PodDisruptionBudget"))
		expectations.ValidatingAdmissionPolicyBinding = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ValidatingAdmissionPolicyBinding"))
		expectations.ServiceAccount = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ServiceAccount"))
		expectations.ValidatingAdmissionPolicy = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ValidatingAdmissionPolicy"))
		expectations.Deployment = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("Deployment"))
		expectations.ValidationWebhook = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ValidationWebhook"))
		expectations.MutatingWebhook = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("MutatingWebhook"))
		expectations.APIService = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("APIService"))
		expectations.Secrets = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("Secrets"))
		expectations.OperatorCrd = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("OperatorCrd"))
		expectations.Service = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("Service"))
		expectations.ClusterRoleBinding = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ClusterRoleBinding"))
		expectations.ClusterRole = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ClusterRole"))
		expectations.RoleBinding = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("RoleBinding"))
		expectations.Role = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("Role"))
		expectations.SCC = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("SCC"))
		expectations.PrometheusRule = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("PrometheusRule"))
		expectations.ServiceMonitor = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ServiceMonitor"))

		pdbClient := fake.NewSimpleClientset()
		dsClient := fake.NewSimpleClientset()
		admissionClient := fake.NewSimpleClientset()
		coreclientset := fake.NewSimpleClientset()
		rbacClient := fake.NewSimpleClientset()
		promClient := promclientfake.NewSimpleClientset()

		kvInterface := kubecli.NewMockKubeVirtInterface(ctrl)
		clientset.EXPECT().KubeVirt(Namespace).Return(kvInterface).AnyTimes()
		clientset.EXPECT().PolicyV1().Return(pdbClient.PolicyV1()).AnyTimes()
		clientset.EXPECT().AppsV1().Return(dsClient.AppsV1()).AnyTimes()
		clientset.EXPECT().AdmissionregistrationV1().Return(admissionClient.AdmissionregistrationV1()).AnyTimes()
		clientset.EXPECT().CoreV1().Return(coreclientset.CoreV1()).AnyTimes()
		clientset.EXPECT().RbacV1().Return(rbacClient.RbacV1()).AnyTimes()
		clientset.EXPECT().PrometheusClient().Return(promClient).AnyTimes()

		clientset.EXPECT().SecClient().Return(fake.NewSimpleClientset()).AnyTimes()
		aggregatorclient := install.NewMockAPIServiceInterface(ctrl)
		aggregatorclient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, errors.NewNotFound(schema.GroupResource{Group: "", Resource: "apiservices"}, "whatever"))
		aggregatorclient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Do(func(ctx context.Context, obj runtime.Object, opts metav1.CreateOptions) {})
		aggregatorclient.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Do(func(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, _ ...string) {
		})

		reconcilerConfig := util.OperatorConfig{
			IsOnOpenshift:                           true,
			ServiceMonitorEnabled:                   true,
			PrometheusRulesEnabled:                  true,
			ValidatingAdmissionPolicyBindingEnabled: true,
			ValidatingAdmissionPolicyEnabled:        true,
		}
		r, err := apply.NewReconciler(kv, strat, stores, reconcilerConfig, clientset, aggregatorclient, expectations, record.NewFakeRecorder(100))
		if err != nil {
			return
		}

		// Call the target entrypoint
		r.Sync(queue)
	})
}

func getConfig(registry, version string) *util.KubeVirtDeploymentConfig {
	return util.GetTargetConfigFromKV(&v1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: Namespace,
		},
		Spec: v1.KubeVirtSpec{
			ImageRegistry: registry,
			ImageTag:      version,
		},
	})
}
