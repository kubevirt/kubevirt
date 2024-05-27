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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package apply

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	routev1 "github.com/openshift/api/route/v1"
	secv1 "github.com/openshift/api/security/v1"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

const (
	castFailedFmt   = "Cast failed! obj: %+v"
	deleteFailedFmt = "Failed to delete %s: %v"
	finalizerPath   = "/metadata/finalizers"
)

func deleteDummyWebhookValidators(kv *v1.KubeVirt,
	clientset kubecli.KubevirtClient,
	stores util.Stores,
	expectations *util.Expectations) error {

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	gracePeriod := int64(0)
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	objects := stores.ValidationWebhookCache.List()
	for _, obj := range objects {
		if webhook, ok := obj.(*admissionregistrationv1.ValidatingWebhookConfiguration); ok {
			if !strings.HasPrefix(webhook.Name, "virt-operator-tmp-webhook") {
				continue
			}
			if webhook.DeletionTimestamp != nil {
				continue
			}
			if key, err := controller.KeyFunc(webhook); err == nil {
				expectations.ValidationWebhook.AddExpectedDeletion(kvkey, key)
				err = clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.Background(), webhook.Name, deleteOptions)
				if err != nil {
					expectations.ValidationWebhook.DeletionObserved(kvkey, key)
					return fmt.Errorf("unable to delete validation webhook: %v", err)
				}
				log.Log.V(2).Infof("Temporary blocking validation webhook %s deleted", webhook.Name)
			}
		}
	}

	return nil
}

func DeleteAll(kv *v1.KubeVirt,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	aggregatorclient install.APIServiceInterface,
	expectations *util.Expectations) error {

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	gracePeriod := int64(0)
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	// first delete CRDs only
	err = crdHandleDeletion(kvkey, stores, clientset, expectations)
	if err != nil {
		return err
	}

	if !util.IsStoreEmpty(stores.CrdCache) {
		// wait until CRDs are gone
		return nil
	}

	// delete daemonsets
	objects := stores.DaemonSetCache.List()
	for _, obj := range objects {
		if ds, ok := obj.(*appsv1.DaemonSet); ok && ds.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(ds); err == nil {
				expectations.DaemonSet.AddExpectedDeletion(kvkey, key)
				err := clientset.AppsV1().DaemonSets(ds.Namespace).Delete(context.Background(), ds.Name, deleteOptions)
				if err != nil {
					expectations.DaemonSet.DeletionObserved(kvkey, key)
					log.Log.Errorf(deleteFailedFmt, ds.Name, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	// delete podDisruptionBudgets
	objects = stores.PodDisruptionBudgetCache.List()
	for _, obj := range objects {
		if pdb, ok := obj.(*policyv1.PodDisruptionBudget); ok && pdb.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(pdb); err == nil {
				pdbClient := clientset.PolicyV1().PodDisruptionBudgets(pdb.Namespace)
				expectations.PodDisruptionBudget.AddExpectedDeletion(kvkey, key)
				err = pdbClient.Delete(context.Background(), pdb.Name, metav1.DeleteOptions{})
				if err != nil {
					expectations.PodDisruptionBudget.DeletionObserved(kvkey, key)
					log.Log.Errorf(deleteFailedFmt, pdb.Name, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	// delete deployments
	objects = stores.DeploymentCache.List()
	for _, obj := range objects {
		if depl, ok := obj.(*appsv1.Deployment); ok && depl.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(depl); err == nil {
				expectations.Deployment.AddExpectedDeletion(kvkey, key)
				err = clientset.AppsV1().Deployments(depl.Namespace).Delete(context.Background(), depl.Name, deleteOptions)
				if err != nil {
					expectations.Deployment.DeletionObserved(kvkey, key)
					log.Log.Errorf(deleteFailedFmt, depl.Name, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	// delete validatingwebhooks
	objects = stores.ValidationWebhookCache.List()
	for _, obj := range objects {
		if webhookConfiguration, ok := obj.(*admissionregistrationv1.ValidatingWebhookConfiguration); ok && webhookConfiguration.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(webhookConfiguration); err == nil {
				expectations.ValidationWebhook.AddExpectedDeletion(kvkey, key)
				err := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.Background(), webhookConfiguration.Name, deleteOptions)
				if err != nil {
					expectations.ValidationWebhook.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete validatingwebhook %+v: %v", webhookConfiguration, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	// delete mutatingwebhooks
	objects = stores.MutatingWebhookCache.List()
	for _, obj := range objects {
		if webhookConfiguration, ok := obj.(*admissionregistrationv1.MutatingWebhookConfiguration); ok && webhookConfiguration.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(webhookConfiguration); err == nil {
				expectations.MutatingWebhook.AddExpectedDeletion(kvkey, key)
				err := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.Background(), webhookConfiguration.Name, deleteOptions)
				if err != nil {
					expectations.MutatingWebhook.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete mutatingwebhook %+v: %v", webhookConfiguration, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	// delete apiservices
	objects = stores.APIServiceCache.List()
	for _, obj := range objects {
		if apiservice, ok := obj.(*apiregv1.APIService); ok && apiservice.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(apiservice); err == nil {
				expectations.APIService.AddExpectedDeletion(kvkey, key)
				err := aggregatorclient.Delete(context.Background(), apiservice.Name, deleteOptions)
				if err != nil {
					expectations.APIService.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete apiservice %+v: %v", apiservice, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	// delete services
	objects = stores.ServiceCache.List()
	for _, obj := range objects {
		if svc, ok := obj.(*corev1.Service); ok && svc.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(svc); err == nil {
				expectations.Service.AddExpectedDeletion(kvkey, key)
				err := clientset.CoreV1().Services(svc.Namespace).Delete(context.Background(), svc.Name, deleteOptions)
				if err != nil {
					expectations.Service.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete service %+v: %v", svc, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	// delete serviceMonitor
	prometheusClient := clientset.PrometheusClient()

	objects = stores.ServiceMonitorCache.List()
	for _, obj := range objects {
		if serviceMonitor, ok := obj.(*promv1.ServiceMonitor); ok && serviceMonitor.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(serviceMonitor); err == nil {
				expectations.ServiceMonitor.AddExpectedDeletion(kvkey, key)
				err := prometheusClient.MonitoringV1().ServiceMonitors(serviceMonitor.Namespace).Delete(context.Background(), serviceMonitor.Name, deleteOptions)
				if err != nil {
					expectations.ServiceMonitor.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete serviceMonitor %+v: %v", serviceMonitor, err)
					return err
				}
				expectations.ServiceMonitor.DeletionObserved(kvkey, key)
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	// delete PrometheusRules
	objects = stores.PrometheusRuleCache.List()
	for _, obj := range objects {
		if prometheusRule, ok := obj.(*promv1.PrometheusRule); ok && prometheusRule.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(prometheusRule); err == nil {
				expectations.PrometheusRule.AddExpectedDeletion(kvkey, key)
				err := prometheusClient.MonitoringV1().PrometheusRules(prometheusRule.Namespace).Delete(context.Background(), prometheusRule.Name, deleteOptions)
				if err != nil {
					log.Log.Errorf("Failed to delete prometheusRule %+v: %v", prometheusRule, err)
					expectations.PrometheusRule.DeletionObserved(kvkey, key)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	// delete RBAC
	objects = stores.ClusterRoleBindingCache.List()
	for _, obj := range objects {
		if crb, ok := obj.(*rbacv1.ClusterRoleBinding); ok && crb.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(crb); err == nil {
				expectations.ClusterRoleBinding.AddExpectedDeletion(kvkey, key)
				err := clientset.RbacV1().ClusterRoleBindings().Delete(context.Background(), crb.Name, deleteOptions)
				if err != nil {
					expectations.ClusterRoleBinding.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete crb %+v: %v", crb, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	objects = stores.ClusterRoleCache.List()
	for _, obj := range objects {
		if cr, ok := obj.(*rbacv1.ClusterRole); ok && cr.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(cr); err == nil {
				expectations.ClusterRole.AddExpectedDeletion(kvkey, key)
				err := clientset.RbacV1().ClusterRoles().Delete(context.Background(), cr.Name, deleteOptions)
				if err != nil {
					expectations.ClusterRole.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete cr %+v: %v", cr, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	objects = stores.RoleBindingCache.List()
	for _, obj := range objects {
		if rb, ok := obj.(*rbacv1.RoleBinding); ok && rb.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(rb); err == nil {
				expectations.RoleBinding.AddExpectedDeletion(kvkey, key)
				err := clientset.RbacV1().RoleBindings(kv.Namespace).Delete(context.Background(), rb.Name, deleteOptions)
				if err != nil {
					expectations.RoleBinding.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete rb %+v: %v", rb, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	objects = stores.RoleCache.List()
	for _, obj := range objects {
		if role, ok := obj.(*rbacv1.Role); ok && role.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(role); err == nil {
				expectations.Role.AddExpectedDeletion(kvkey, key)
				err := clientset.RbacV1().Roles(kv.Namespace).Delete(context.Background(), role.Name, deleteOptions)
				if err != nil {
					expectations.Role.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete role %+v: %v", role, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	objects = stores.ServiceAccountCache.List()
	for _, obj := range objects {
		if sa, ok := obj.(*corev1.ServiceAccount); ok && sa.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(sa); err == nil {
				expectations.ServiceAccount.AddExpectedDeletion(kvkey, key)
				err := clientset.CoreV1().ServiceAccounts(kv.Namespace).Delete(context.Background(), sa.Name, deleteOptions)
				if err != nil {
					expectations.ServiceAccount.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete serviceaccount %+v: %v", sa, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	objects = stores.SecretCache.List()
	for _, obj := range objects {
		if secret, ok := obj.(*corev1.Secret); ok && secret.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(secret); err == nil {
				expectations.Secrets.AddExpectedDeletion(kvkey, key)
				err := clientset.CoreV1().Secrets(kv.Namespace).Delete(context.Background(), secret.Name, deleteOptions)
				if err != nil {
					expectations.Secrets.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete secret %+v: %v", secret, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	objects = stores.ConfigMapCache.List()
	for _, obj := range objects {
		if configMap, ok := obj.(*corev1.ConfigMap); ok && configMap.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(configMap); err == nil {
				expectations.ConfigMap.AddExpectedDeletion(kvkey, key)
				err := clientset.CoreV1().ConfigMaps(kv.Namespace).Delete(context.Background(), configMap.Name, deleteOptions)
				if err != nil {
					expectations.ConfigMap.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete configMap %+v: %v", configMap, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	scc := clientset.SecClient()
	objects = stores.SCCCache.List()
	for _, obj := range objects {
		if s, ok := obj.(*secv1.SecurityContextConstraints); ok && s.DeletionTimestamp == nil {

			// informer watches all SCC objects, it cannot be changed because of kubevirt updates
			if !util.IsManagedByOperator(s.GetLabels()) {
				continue
			}

			if key, err := controller.KeyFunc(s); err == nil {
				expectations.SCC.AddExpectedDeletion(kvkey, key)
				err := scc.SecurityContextConstraints().Delete(context.Background(), s.Name, deleteOptions)
				if err != nil {
					expectations.SCC.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete SecurityContextConstraints %+v: %v", s, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	objects = stores.RouteCache.List()
	for _, obj := range objects {
		if route, ok := obj.(*routev1.Route); ok && route.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(route); err == nil {
				expectations.Route.AddExpectedDeletion(kvkey, key)
				err := clientset.RouteClient().Routes(kv.Namespace).Delete(context.Background(), route.Name, deleteOptions)
				if err != nil {
					expectations.Route.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete route %+v: %v", route, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	objects = stores.ValidatingAdmissionPolicyBindingCache.List()
	for _, obj := range objects {
		if validatingAdmissionPolicyBinding, ok := obj.(*admissionregistrationv1.ValidatingAdmissionPolicyBinding); ok && validatingAdmissionPolicyBinding.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(validatingAdmissionPolicyBinding); err == nil {
				expectations.ConfigMap.AddExpectedDeletion(kvkey, key)
				err := clientset.AdmissionregistrationV1().ValidatingAdmissionPolicyBindings().Delete(context.Background(), validatingAdmissionPolicyBinding.Name, deleteOptions)
				if err != nil {
					expectations.ValidatingAdmissionPolicyBinding.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete validatingAdmissionPolicyBinding %+v: %v", validatingAdmissionPolicyBinding, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	objects = stores.ValidatingAdmissionPolicyCache.List()
	for _, obj := range objects {
		if validatingAdmissionPolicy, ok := obj.(*admissionregistrationv1.ValidatingAdmissionPolicy); ok && validatingAdmissionPolicy.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(validatingAdmissionPolicy); err == nil {
				expectations.ConfigMap.AddExpectedDeletion(kvkey, key)
				err := clientset.AdmissionregistrationV1().ValidatingAdmissionPolicies().Delete(context.Background(), validatingAdmissionPolicy.Name, deleteOptions)
				if err != nil {
					expectations.ValidatingAdmissionPolicy.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete validatingAdmissionPolicy %+v: %v", validatingAdmissionPolicy, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
	}

	err = deleteDummyWebhookValidators(kv, clientset, stores, expectations)
	if err != nil {
		return err
	}
	return nil
}

func crdInstanceDeletionCompleted(crd *extv1.CustomResourceDefinition) bool {
	// Below is an example of what is being looked for here.
	// The CRD will have this condition once a CRD which is being
	// deleted has all instances removed related to this CRD.
	//
	//    message: removed all instances
	//    reason: InstanceDeletionCompleted
	//    status: "False"
	//    type: Terminating

	if crd.DeletionTimestamp == nil {
		return false
	}

	for _, condition := range crd.Status.Conditions {
		if condition.Type == extv1.Terminating &&
			condition.Status == extv1.ConditionFalse &&
			condition.Reason == "InstanceDeletionCompleted" {
			return true
		}
	}
	return false
}

func crdFilterNeedFinalizerAdded(crds []*extv1.CustomResourceDefinition) []*extv1.CustomResourceDefinition {
	filtered := []*extv1.CustomResourceDefinition{}

	for _, crd := range crds {
		if crd.DeletionTimestamp == nil && !controller.HasFinalizer(crd, v1.VirtOperatorComponentFinalizer) {
			filtered = append(filtered, crd)
		}
	}

	return filtered
}

func crdFilterNeedDeletion(crds []*extv1.CustomResourceDefinition) []*extv1.CustomResourceDefinition {
	filtered := []*extv1.CustomResourceDefinition{}

	for _, crd := range crds {
		if crd.DeletionTimestamp == nil {
			filtered = append(filtered, crd)
		}
	}
	return filtered
}

func crdFilterNeedFinalizerRemoved(crds []*extv1.CustomResourceDefinition) []*extv1.CustomResourceDefinition {
	filtered := []*extv1.CustomResourceDefinition{}
	for _, crd := range crds {
		if !crdInstanceDeletionCompleted(crd) {
			// All crds must have all crs removed before any CRD finalizer can be removed
			return []*extv1.CustomResourceDefinition{}
		} else if controller.HasFinalizer(crd, v1.VirtOperatorComponentFinalizer) {
			filtered = append(filtered, crd)
		}
	}
	return filtered
}

func crdHandleDeletion(kvkey string,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) error {

	ext := clientset.ExtensionsClient()
	objects := stores.CrdCache.List()

	finalizerPath := "/metadata/finalizers"

	crds := []*extv1.CustomResourceDefinition{}
	for _, obj := range objects {
		crd, ok := obj.(*extv1.CustomResourceDefinition)
		if !ok {
			log.Log.Errorf(castFailedFmt, obj)
			return nil
		}
		crds = append(crds, crd)
	}

	needFinalizerAdded := crdFilterNeedFinalizerAdded(crds)
	needDeletion := crdFilterNeedDeletion(crds)
	needFinalizerRemoved := crdFilterNeedFinalizerRemoved(crds)

	for _, crd := range needFinalizerAdded {
		crdCopy := crd.DeepCopy()
		controller.AddFinalizer(crdCopy, v1.VirtOperatorComponentFinalizer)

		patchBytes, err := json.Marshal(crdCopy.Finalizers)
		if err != nil {
			return err
		}
		ops := fmt.Sprintf(`[{ "op": "add", "path": "%s", "value": %s }]`, finalizerPath, string(patchBytes))
		_, err = ext.ApiextensionsV1().CustomResourceDefinitions().Patch(context.Background(), crd.Name, types.JSONPatchType, []byte(ops), metav1.PatchOptions{})
		if err != nil {
			return err
		}
	}

	for _, crd := range needDeletion {
		key, err := controller.KeyFunc(crd)
		if err != nil {
			return err
		}

		expectations.Crd.AddExpectedDeletion(kvkey, key)
		err = ext.ApiextensionsV1().CustomResourceDefinitions().Delete(context.Background(), crd.Name, metav1.DeleteOptions{})
		if err != nil {
			expectations.Crd.DeletionObserved(kvkey, key)
			log.Log.Errorf("Failed to delete crd %+v: %v", crd, err)
			return err
		}
	}

	for _, crd := range needFinalizerRemoved {
		var ops string
		if len(crd.Finalizers) > 1 {
			crdCopy := crd.DeepCopy()
			controller.RemoveFinalizer(crdCopy, v1.VirtOperatorComponentFinalizer)

			newPatchBytes, err := json.Marshal(crdCopy.Finalizers)
			if err != nil {
				return err
			}

			oldPatchBytes, err := json.Marshal(crd.Finalizers)
			if err != nil {
				return err
			}

			ops = fmt.Sprintf(`[{ "op": "test", "path": "%s", "value": %s }, { "op": "replace", "path": "%s", "value": %s }]`,
				finalizerPath,
				string(oldPatchBytes),
				finalizerPath,
				string(newPatchBytes))
		} else {
			ops = fmt.Sprintf(`[{ "op": "remove", "path": "%s" }]`, finalizerPath)
		}

		_, err := ext.ApiextensionsV1().CustomResourceDefinitions().Patch(context.Background(), crd.Name, types.JSONPatchType, []byte(ops), metav1.PatchOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}
