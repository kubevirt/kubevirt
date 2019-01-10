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

	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	apiclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

func Delete(kv *v1.KubeVirt, clientset kubecli.KubevirtClient) (int, error) {

	objectsDeleted := 0

	gracePeriod := int64(0)
	deleteOptions := &metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	// delete vmimigrations, vmirs, vm, vmi
	vmimList, err := clientset.VirtualMachineInstanceMigration(metav1.NamespaceAll).List(&metav1.ListOptions{})
	if err != nil {
		log.Log.Errorf("Failed to delete vmims: %v", err)
		return objectsDeleted, err
	}
	for _, vmim := range vmimList.Items {
		if vmim.DeletionTimestamp != nil {
			continue
		}
		clientset.VirtualMachineInstanceMigration(vmim.Namespace).Delete(vmim.Namespace, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete vmim %+v: %v", vmim, err)
			return objectsDeleted, err
		}
		objectsDeleted++
	}

	rslist, err := clientset.ReplicaSet(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		log.Log.Errorf("Failed to delete vmrss: %v", err)
		return objectsDeleted, err
	}
	for _, vmrs := range rslist.Items {
		if vmrs.DeletionTimestamp != nil {
			continue
		}
		clientset.VirtualMachineInstanceMigration(vmrs.Namespace).Delete(vmrs.Namespace, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete vmrs %+v: %v", vmrs, err)
			return objectsDeleted, err
		}
		objectsDeleted++
	}

	vmlist, err := clientset.VirtualMachine(metav1.NamespaceAll).List(&metav1.ListOptions{})
	if err != nil {
		log.Log.Errorf("Failed to delete vm: %v", err)
		return objectsDeleted, err
	}
	for _, vm := range vmlist.Items {
		if vm.DeletionTimestamp != nil {
			continue
		}
		clientset.VirtualMachine(vm.Namespace).Delete(vm.Namespace, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete vm %+v: %v", vm, err)
			return objectsDeleted, err
		}
		objectsDeleted++
	}

	vmilist, err := clientset.VirtualMachineInstance(metav1.NamespaceAll).List(&metav1.ListOptions{})
	if err != nil {
		log.Log.Errorf("Failed to delete vmis: %v", err)
		return objectsDeleted, err
	}
	for _, vmi := range vmilist.Items {
		if vmi.DeletionTimestamp != nil {
			continue
		}
		// remove finalizer first
		patchStr := fmt.Sprintf(`{"metadata":{"finalizers":"[]"}}`)
		vmi, _ := clientset.VirtualMachine(vmi.Namespace).Patch(vmi.Name, types.MergePatchType, []byte(patchStr))
		clientset.VirtualMachine(vmi.Namespace).Delete(vmi.Namespace, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete vmi %+v: %v", vmi, err)
			return objectsDeleted, err
		}
		objectsDeleted++
	}

	// delete launcher pods
	podList, err := clientset.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=virt-launcher", v1.AppLabel)})
	if err != nil {
		log.Log.Errorf("Failed to list launcher pods: %v", err)
		return objectsDeleted, err
	}
	for _, pod := range podList.Items {
		if pod.DeletionTimestamp != nil {
			continue
		}
		err := clientset.CoreV1().Pods(pod.Namespace).Delete(pod.Name, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete launcher pod %+v: %v", pod)
			return objectsDeleted, err
		}
		objectsDeleted++
	}

	// delete handler daemonset
	handler, err := clientset.AppsV1().DaemonSets(kv.Namespace).Get("virt-handler", metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Log.Errorf("Failed to get virt-handler: %v", err)
		return objectsDeleted, err
	} else if !errors.IsNotFound(err) && handler.DeletionTimestamp == nil {
		err := clientset.AppsV1().DaemonSets(kv.Namespace).Delete("virt-handler", deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete virt-handler: %v", err)
			return objectsDeleted, err
		}
		objectsDeleted++
	}

	// delete controller and apiserver deployment
	for _, name := range []string{"virt-controller", "virt-api"} {
		deployment, err := clientset.AppsV1().Deployments(kv.Namespace).Get(name, metav1.GetOptions{})
		if err != nil && !errors.IsNotFound(err) {
			log.Log.Errorf("Failed to get %v: %v", name, err)
			return objectsDeleted, err
		} else if !errors.IsNotFound(err) && deployment.DeletionTimestamp == nil {
			err = clientset.AppsV1().Deployments(kv.Namespace).Delete(name, deleteOptions)
			if err != nil {
				log.Log.Errorf("Failed to delete %v: %v", name, err)
				return objectsDeleted, err
			}
			objectsDeleted++
		}
	}

	// delete apiservices
	api, err := apiclient.NewForConfig(clientset.Config())
	for _, name := range []string{v1.GroupVersion.Version + "." + v1.GroupName, v1.SubresourceGroupVersion.Version + "." + v1.SubresourceGroupName} {
		apiSvc, err := api.ApiregistrationV1().APIServices().Get(name, metav1.GetOptions{})
		if err != nil && !errors.IsNotFound(err) {
			log.Log.Errorf("Failed to get apiservice %v: %v", name, err)
			return objectsDeleted, err
		} else if !errors.IsNotFound(err) && apiSvc.DeletionTimestamp == nil {
			err = api.ApiregistrationV1().APIServices().Delete(name, deleteOptions)
			if err != nil {
				log.Log.Errorf("Failed to delete apiservice %v: %v", name, err)
				return objectsDeleted, err
			}
			objectsDeleted++
		}
	}

	// delete services
	svcList, err := clientset.CoreV1().Services(kv.Namespace).List(metav1.ListOptions{LabelSelector: v1.AppLabel})
	if err != nil {
		log.Log.Errorf("Failed to list services: %v", err)
		return objectsDeleted, err
	}
	for _, svc := range svcList.Items {
		if svc.DeletionTimestamp != nil {
			continue
		}
		err := clientset.CoreV1().Services(kv.Namespace).Delete(svc.Name, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete service %+v: %v", svc)
			return objectsDeleted, err
		}
		objectsDeleted++
	}

	// delete RBAC
	crbList, err := clientset.RbacV1().ClusterRoleBindings().List(metav1.ListOptions{LabelSelector: v1.AppLabel})
	if err != nil {
		log.Log.Errorf("Failed to list crds: %v", err)
		return objectsDeleted, err
	}
	for _, crb := range crbList.Items {
		if crb.DeletionTimestamp != nil {
			continue
		}
		err := clientset.RbacV1().ClusterRoleBindings().Delete(crb.Name, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete crd %+v: %v", crb, err)
			return objectsDeleted, err
		}
		objectsDeleted++
	}

	crList, err := clientset.RbacV1().ClusterRoles().List(metav1.ListOptions{LabelSelector: v1.AppLabel})
	if err != nil {
		log.Log.Errorf("Failed to list crs: %v", err)
		return objectsDeleted, err
	}
	for _, cr := range crList.Items {
		if cr.DeletionTimestamp != nil {
			continue
		}
		err := clientset.RbacV1().ClusterRoles().Delete(cr.Name, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete cr %+v: %v", cr, err)
			return objectsDeleted, err
		}
		objectsDeleted++
	}

	rbList, err := clientset.RbacV1().RoleBindings(kv.Namespace).List(metav1.ListOptions{LabelSelector: v1.AppLabel})
	if err != nil {
		log.Log.Errorf("Failed to list rbs: %v", err)
		return objectsDeleted, err
	}
	for _, rb := range rbList.Items {
		if rb.DeletionTimestamp != nil {
			continue
		}
		err := clientset.RbacV1().RoleBindings(kv.Namespace).Delete(rb.Name, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete rb %+v: %v", rb, err)
			return objectsDeleted, err
		}
		objectsDeleted++
	}

	rList, err := clientset.RbacV1().Roles(kv.Namespace).List(metav1.ListOptions{LabelSelector: v1.AppLabel})
	if err != nil {
		log.Log.Errorf("Failed to list roles: %v", err)
		return objectsDeleted, err
	}
	for _, role := range rList.Items {
		if role.DeletionTimestamp != nil {
			continue
		}
		err := clientset.RbacV1().Roles(kv.Namespace).Delete(role.Name, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete crd %+v: %v", role, err)
			return objectsDeleted, err
		}
		objectsDeleted++
	}

	saList, err := clientset.CoreV1().ServiceAccounts(kv.Namespace).List(metav1.ListOptions{LabelSelector: v1.AppLabel})
	if err != nil {
		log.Log.Errorf("Failed to list serviceaccounts: %v", err)
		return objectsDeleted, err
	}
	for _, sa := range saList.Items {
		if sa.DeletionTimestamp != nil {
			continue
		}
		err := clientset.CoreV1().ServiceAccounts(kv.Namespace).Delete(sa.Name, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete serviceaccount %+v: %v", sa, err)
			return objectsDeleted, err
		}
		objectsDeleted++
	}

	// delete CRDs
	ext, err := extclient.NewForConfig(clientset.Config())
	crdList, err := ext.ApiextensionsV1beta1().CustomResourceDefinitions().List(metav1.ListOptions{LabelSelector: v1.AppLabel})
	if err != nil {
		log.Log.Errorf("Failed to list crds: %v", err)
		return objectsDeleted, err
	}
	for _, crd := range crdList.Items {
		if crd.DeletionTimestamp != nil {
			continue
		}
		err := ext.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crd.Name, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete crd %+v: %v", crd, err)
			return objectsDeleted, err
		}
		objectsDeleted++
	}

	return objectsDeleted, nil

}
