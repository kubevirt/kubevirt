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

package installstrategy

import (
	"encoding/json"
	"fmt"

	secv1 "github.com/openshift/api/security/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	pspv1b1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/controller"
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
	deleteOptions := &metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	objects := stores.ValidationWebhookCache.List()
	for _, obj := range objects {
		if webhook, ok := obj.(*admissionregistrationv1beta1.ValidatingWebhookConfiguration); ok {

			if webhook.DeletionTimestamp != nil {
				continue
			}
			if key, err := controller.KeyFunc(webhook); err == nil {
				expectations.ValidationWebhook.AddExpectedDeletion(kvkey, key)
				err = clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Delete(webhook.Name, deleteOptions)
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
	strategy *InstallStrategy,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) error {

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	gracePeriod := int64(0)
	deleteOptions := &metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	// first delete CRDs only
	ext := clientset.ExtensionsClient()
	objects := stores.CrdCache.List()
	for _, obj := range objects {
		if crd, ok := obj.(*extv1beta1.CustomResourceDefinition); ok && crd.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(crd); err == nil {
				expectations.Crd.AddExpectedDeletion(kvkey, key)
				err := ext.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crd.Name, deleteOptions)
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
				err := clientset.AppsV1().DaemonSets(ds.Namespace).Delete(ds.Name, deleteOptions)
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

	// delete deployments
	objects = stores.DeploymentCache.List()
	for _, obj := range objects {
		if depl, ok := obj.(*appsv1.Deployment); ok && depl.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(depl); err == nil {
				expectations.Deployment.AddExpectedDeletion(kvkey, key)
				err := clientset.AppsV1().Deployments(depl.Namespace).Delete(depl.Name, deleteOptions)
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

	// delete services
	objects = stores.ServiceCache.List()
	for _, obj := range objects {
		if svc, ok := obj.(*corev1.Service); ok && svc.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(svc); err == nil {
				expectations.Service.AddExpectedDeletion(kvkey, key)
				err := clientset.CoreV1().Services(kv.Namespace).Delete(svc.Name, deleteOptions)
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

	// delete RBAC
	objects = stores.ClusterRoleBindingCache.List()
	for _, obj := range objects {
		if crb, ok := obj.(*rbacv1.ClusterRoleBinding); ok && crb.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(crb); err == nil {
				expectations.ClusterRoleBinding.AddExpectedDeletion(kvkey, key)
				err := clientset.RbacV1().ClusterRoleBindings().Delete(crb.Name, deleteOptions)
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
				err := clientset.RbacV1().ClusterRoles().Delete(cr.Name, deleteOptions)
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
				err := clientset.RbacV1().RoleBindings(kv.Namespace).Delete(rb.Name, deleteOptions)
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
				err := clientset.RbacV1().Roles(kv.Namespace).Delete(role.Name, deleteOptions)
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
				err := clientset.CoreV1().ServiceAccounts(kv.Namespace).Delete(sa.Name, deleteOptions)
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

	objects = stores.PodSecurityPolicyCache.List()
	for _, obj := range objects {
		if psp, ok := obj.(*pspv1b1.PodSecurityPolicy); ok && psp.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(psp); err == nil {
				expectations.PodSecurityPolicy.AddExpectedDeletion(kvkey, key)
				err := clientset.PolicyV1beta1().PodSecurityPolicies().Delete(psp.Name, deleteOptions)
				if err != nil {
					expectations.PodSecurityPolicy.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete pod security policy %+v: %v", psp, err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf("Cast failed! obj: %+v", obj)
			return nil
		}
	}

	scc := clientset.SecClient()
	for _, sccPriv := range strategy.customSCCPrivileges {
		privSCCObj, exists, err := stores.SCCCache.GetByKey(sccPriv.TargetSCC)
		if !exists {
			return nil
		} else if err != nil {
			return err
		}

		privSCC, ok := privSCCObj.(*secv1.SecurityContextConstraints)
		if !ok {
			return fmt.Errorf("couldn't cast object to SecurityContextConstraints: %+v", privSCCObj)
		}

		oldUsers := privSCC.Users
		privSCCCopy := privSCC.DeepCopy()
		users := privSCCCopy.Users

		modified := false

		for _, acc := range sccPriv.ServiceAccounts {
			removed := false
			users, removed = remove(users, acc)
			modified = modified || removed
		}

		if modified {
			oldUserBytes, err := json.Marshal(oldUsers)
			if err != nil {
				return err
			}
			userBytes, err := json.Marshal(users)
			if err != nil {
				return err
			}

			test := fmt.Sprintf(`{ "op": "test", "path": "/users", "value": %s }`, string(oldUserBytes))
			patch := fmt.Sprintf(`{ "op": "replace", "path": "/users", "value": %s }`, string(userBytes))

			_, err = scc.SecurityContextConstraints().Patch(sccPriv.TargetSCC, types.JSONPatchType, []byte(fmt.Sprintf("[ %s, %s ]", test, patch)))
			if err != nil {
				return fmt.Errorf("unable to patch scc: %v", err)
			}
		}
	}

	deleteDummyWebhookValidators(kv, clientset, stores, expectations)

	return nil
}
