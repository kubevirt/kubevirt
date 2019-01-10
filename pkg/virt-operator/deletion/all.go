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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package deletion

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/util"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

func Delete(kv *v1.KubeVirt, clientset kubecli.KubevirtClient, stores util.Stores) (int, error) {

	objectsDeleted := 0

	gracePeriod := int64(0)
	deleteOptions := &metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	// first delete CRDs only
	ext, err := extclient.NewForConfig(clientset.Config())
	objects := stores.CrdCache.List()
	for _, obj := range objects {
		if crd, ok := obj.(apiextensions.CustomResourceDefinition); ok && crd.DeletionTimestamp == nil {
			err := ext.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crd.Name, deleteOptions)
			if err != nil {
				log.Log.Errorf("Failed to delete crd %+v: %v", crd, err)
				return objectsDeleted, err
			}
			objectsDeleted++
		}
	}
	if objectsDeleted > 0 {
		// wait until CRDs are gone
		return objectsDeleted, nil
	}

	// delete handler daemonset
	obj, exists, err := stores.DaemonSetCache.GetByKey(fmt.Sprintf("%s/%s", kv.Namespace, "virt-handler"))
	if err != nil {
		log.Log.Errorf("Failed to get virt-handler: %v", err)
		return objectsDeleted, err
	} else if exists {
		if ds, ok := obj.(*appsv1.DaemonSet); ok && ds.DeletionTimestamp == nil {
			err := clientset.AppsV1().DaemonSets(kv.Namespace).Delete("virt-handler", deleteOptions)
			if err != nil {
				log.Log.Errorf("Failed to delete virt-handler: %v", err)
				return objectsDeleted, err
			}
			objectsDeleted++
		}
	}

	// delete controller and apiserver deployment
	for _, name := range []string{"virt-controller", "virt-api"} {
		obj, exists, err := stores.DeploymentCache.GetByKey(fmt.Sprintf("%s/%s", kv.Namespace, name))
		if err != nil {
			log.Log.Errorf("Failed to get %v: %v", name, err)
			return objectsDeleted, err
		} else if exists {
			if depl, ok := obj.(*appsv1.Deployment); ok && depl.DeletionTimestamp == nil {
				err := clientset.AppsV1().Deployments(kv.Namespace).Delete(name, deleteOptions)
				if err != nil {
					log.Log.Errorf("Failed to delete virt-handler: %v", err)
					return objectsDeleted, err
				}
				objectsDeleted++
			}
		}
	}

	// delete services
	objects = stores.ServiceCache.List()
	for _, obj := range objects {
		if svc, ok := obj.(corev1.Service); ok && svc.DeletionTimestamp == nil {
			err := clientset.CoreV1().Services(kv.Namespace).Delete(svc.Name, deleteOptions)
			if err != nil {
				log.Log.Errorf("Failed to delete service %+v: %v", svc, err)
				return objectsDeleted, err
			}
			objectsDeleted++
		}
	}

	// delete RBAC
	objects = stores.ClusterRoleBindingCache.List()
	for _, obj := range objects {
		if crb, ok := obj.(rbacv1.ClusterRoleBinding); ok && crb.DeletionTimestamp == nil {
			err := clientset.RbacV1().ClusterRoleBindings().Delete(crb.Name, deleteOptions)
			if err != nil {
				log.Log.Errorf("Failed to delete crb %+v: %v", crb, err)
				return objectsDeleted, err
			}
			objectsDeleted++
		}
	}

	objects = stores.ClusterRoleCache.List()
	for _, obj := range objects {
		if cr, ok := obj.(rbacv1.ClusterRole); ok && cr.DeletionTimestamp == nil {
			err := clientset.RbacV1().ClusterRoles().Delete(cr.Name, deleteOptions)
			if err != nil {
				log.Log.Errorf("Failed to delete cr %+v: %v", cr, err)
				return objectsDeleted, err
			}
			objectsDeleted++
		}
	}

	objects = stores.RoleBindingCache.List()
	for _, obj := range objects {
		if rb, ok := obj.(rbacv1.RoleBinding); ok && rb.DeletionTimestamp == nil {
			err := clientset.RbacV1().RoleBindings(kv.Namespace).Delete(rb.Name, deleteOptions)
			if err != nil {
				log.Log.Errorf("Failed to delete rb %+v: %v", rb, err)
				return objectsDeleted, err
			}
			objectsDeleted++
		}
	}

	objects = stores.RoleCache.List()
	for _, obj := range objects {
		if role, ok := obj.(rbacv1.Role); ok && role.DeletionTimestamp == nil {
			err := clientset.RbacV1().Roles(kv.Namespace).Delete(role.Name, deleteOptions)
			if err != nil {
				log.Log.Errorf("Failed to delete role %+v: %v", role, err)
				return objectsDeleted, err
			}
			objectsDeleted++
		}
	}

	objects = stores.ServiceAccountCache.List()
	for _, obj := range objects {
		if sa, ok := obj.(corev1.ServiceAccount); ok && sa.DeletionTimestamp == nil {
			err := clientset.CoreV1().ServiceAccounts(kv.Namespace).Delete(sa.Name, deleteOptions)
			if err != nil {
				log.Log.Errorf("Failed to delete serviceaccount %+v: %v", sa, err)
				return objectsDeleted, err
			}
			objectsDeleted++
		}
	}

	err = util.UpdateScc(clientset, kv, false)
	if err != nil {
		log.Log.Errorf("Failed to update SCC: %v", err)
		return objectsDeleted, err
	}

	return objectsDeleted, nil

}
