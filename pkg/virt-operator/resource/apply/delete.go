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
	"fmt"
	"strings"

	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	secv1 "github.com/openshift/api/security/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
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
	ext := clientset.ExtensionsClient()
	objects := stores.CrdCache.List()
	for _, obj := range objects {
		if crd, ok := obj.(*extv1.CustomResourceDefinition); ok && crd.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(crd); err == nil {
				expectations.Crd.AddExpectedDeletion(kvkey, key)
				err := ext.ApiextensionsV1().CustomResourceDefinitions().Delete(context.Background(), crd.Name, deleteOptions)
				if err != nil {
					expectations.Crd.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete crd %+v: %v", crd, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf("Cast failed! obj: %+v", obj)
			return nil
		}

	}
	if !util.IsStoreEmpty(stores.CrdCache) {
		// wait until CRDs are gone
		return nil
	}

	// delete daemonsets
	objects = stores.DaemonSetCache.List()
	for _, obj := range objects {
		if ds, ok := obj.(*appsv1.DaemonSet); ok && ds.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(ds); err == nil {
				expectations.DaemonSet.AddExpectedDeletion(kvkey, key)
				err := clientset.AppsV1().DaemonSets(ds.Namespace).Delete(context.Background(), ds.Name, deleteOptions)
				if err != nil {
					expectations.DaemonSet.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete %s: %v", ds.Name, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf("Cast failed! obj: %+v", obj)
			return nil
		}
	}

	// delete podDisruptionBudgets
	objects = stores.PodDisruptionBudgetCache.List()
	for _, obj := range objects {
		if pdb, ok := obj.(*policyv1beta1.PodDisruptionBudget); ok && pdb.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(pdb); err == nil {
				pdbClient := clientset.PolicyV1beta1().PodDisruptionBudgets(pdb.Namespace)
				expectations.PodDisruptionBudget.AddExpectedDeletion(kvkey, key)
				err = pdbClient.Delete(context.Background(), pdb.Name, metav1.DeleteOptions{})
				if err != nil {
					expectations.PodDisruptionBudget.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete %s: %v", pdb.Name, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf("Cast failed! obj: %+v", obj)
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
					log.Log.Errorf("Failed to delete %s: %v", depl.Name, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf("Cast failed! obj: %+v", obj)
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
			log.Log.Errorf("Cast failed! obj: %+v", obj)
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
			log.Log.Errorf("Cast failed! obj: %+v", obj)
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
			log.Log.Errorf("Cast failed! obj: %+v", obj)
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
			log.Log.Errorf("Cast failed! obj: %+v", obj)
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
			log.Log.Errorf("Cast failed! obj: %+v", obj)
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
			log.Log.Errorf("Cast failed! obj: %+v", obj)
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
			log.Log.Errorf("Cast failed! obj: %+v", obj)
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
			log.Log.Errorf("Cast failed! obj: %+v", obj)
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
			log.Log.Errorf("Cast failed! obj: %+v", obj)
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
			log.Log.Errorf("Cast failed! obj: %+v", obj)
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
			log.Log.Errorf("Cast failed! obj: %+v", obj)
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
			log.Log.Errorf("Cast failed! obj: %+v", obj)
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
			log.Log.Errorf("Cast failed! obj: %+v", obj)
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
			log.Log.Errorf("Cast failed! obj: %+v", obj)
			return nil
		}
	}

	err = deleteDummyWebhookValidators(kv, clientset, stores, expectations)
	if err != nil {
		return err
	}
	return nil
}
