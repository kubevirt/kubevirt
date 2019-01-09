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
		clientset.VirtualMachineInstanceMigration(vmim.Namespace).Delete(vmim.Namespace, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete vmim %+v: %v", vmim, err)
			return objectsDeleted, err
		} else {
			objectsDeleted++
		}
	}

	rslist, err := clientset.ReplicaSet(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		log.Log.Errorf("Failed to delete vmrss: %v", err)
		return objectsDeleted, err
	}
	for _, vmrs := range rslist.Items {
		clientset.VirtualMachineInstanceMigration(vmrs.Namespace).Delete(vmrs.Namespace, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete vmrs %+v: %v", vmrs, err)
			return objectsDeleted, err
		} else {
			objectsDeleted++
		}
	}

	vmlist, err := clientset.VirtualMachine(metav1.NamespaceAll).List(&metav1.ListOptions{})
	if err != nil {
		log.Log.Errorf("Failed to delete vm: %v", err)
		return objectsDeleted, err
	}
	for _, vm := range vmlist.Items {
		clientset.VirtualMachine(vm.Namespace).Delete(vm.Namespace, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete vm %+v: %v", vm, err)
			return objectsDeleted, err
		} else {
			objectsDeleted++
		}
	}

	vmilist, err := clientset.VirtualMachineInstance(metav1.NamespaceAll).List(&metav1.ListOptions{})
	if err != nil {
		log.Log.Errorf("Failed to delete vmis: %v", err)
		return objectsDeleted, err
	}
	for _, vmi := range vmilist.Items {
		// remove finalizer first
		patchStr := fmt.Sprintf(`{"metadata":{"finalizers":"[]"}}`)
		vmi, _ := clientset.VirtualMachine(vmi.Namespace).Patch(vmi.Name, types.MergePatchType, []byte(patchStr))
		clientset.VirtualMachine(vmi.Namespace).Delete(vmi.Namespace, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete vmi %+v: %v", vmi, err)
			return objectsDeleted, err
		} else {
			objectsDeleted++
		}
	}

	// delete launcher pods
	podList, err := clientset.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=virt-launcher", v1.AppLabel)})
	if err != nil {
		log.Log.Errorf("Failed to list launcher pods: %v", err)
		return objectsDeleted, err
	}
	for _, pod := range podList.Items {
		err := clientset.CoreV1().Pods(pod.Namespace).Delete(pod.Name, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete launcher pod %+v: %v", pod)
			return objectsDeleted, err
		} else {
			objectsDeleted++
		}
	}

	// delete handler, controller, api
	err = clientset.AppsV1().DaemonSets(kv.Namespace).Delete("virt-handler", deleteOptions)
	if err != nil {
		log.Log.Errorf("Failed to delete virt-handler: %v", err)
		return objectsDeleted, err
	} else {
		objectsDeleted++
	}

	err = clientset.AppsV1().Deployments(kv.Namespace).Delete("virt-controller", deleteOptions)
	if err != nil {
		log.Log.Errorf("Failed to delete virt-controller: %v", err)
		return objectsDeleted, err
	} else {
		objectsDeleted++
	}

	err = clientset.AppsV1().Deployments(kv.Namespace).Delete("virt-api", deleteOptions)
	if err != nil {
		log.Log.Errorf("Failed to delete virt-api: %v", err)
		return objectsDeleted, err
	} else {
		objectsDeleted++
	}

	// delete apiservices
	api, err := apiclient.NewForConfig(clientset.Config())
	err = api.ApiregistrationV1().APIServices().Delete(v1.GroupVersion.Version+"."+v1.GroupName, deleteOptions)
	if err != nil {
		log.Log.Errorf("Failed to delete apiservices: %v", err)
		return objectsDeleted, err
	} else {
		objectsDeleted++
	}

	err = api.ApiregistrationV1().APIServices().Delete(v1.SubresourceGroupVersion.Version+"."+v1.SubresourceGroupName, deleteOptions)
	if err != nil {
		log.Log.Errorf("Failed to delete subresource apiservices: %v", err)
		return objectsDeleted, err
	} else {
		objectsDeleted++
	}

	// delete services
	svcList, err := clientset.CoreV1().Services(kv.Namespace).List(metav1.ListOptions{LabelSelector: v1.AppLabel})
	if err != nil {
		log.Log.Errorf("Failed to list services: %v", err)
		return objectsDeleted, err
	}
	for _, svc := range svcList.Items {
		err := clientset.CoreV1().Services(kv.Namespace).Delete(svc.Name, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete service %+v: %v", svc)
			return objectsDeleted, err
		} else {
			objectsDeleted++
		}
	}

	// delete RBAC
	crbList, err := clientset.RbacV1().ClusterRoleBindings().List(metav1.ListOptions{LabelSelector: v1.AppLabel})
	if err != nil {
		log.Log.Errorf("Failed to list crds: %v", err)
		return objectsDeleted, err
	}
	for _, crb := range crbList.Items {
		err := clientset.RbacV1().ClusterRoleBindings().Delete(crb.Name, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete crd %+v: %v", crb, err)
			return objectsDeleted, err
		} else {
			objectsDeleted++
		}
	}

	crList, err := clientset.RbacV1().ClusterRoles().List(metav1.ListOptions{LabelSelector: v1.AppLabel})
	if err != nil {
		log.Log.Errorf("Failed to list crs: %v", err)
		return objectsDeleted, err
	}
	for _, cr := range crList.Items {
		err := clientset.RbacV1().ClusterRoles().Delete(cr.Name, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete cr %+v: %v", cr, err)
			return objectsDeleted, err
		} else {
			objectsDeleted++
		}
	}

	rbList, err := clientset.RbacV1().RoleBindings(kv.Namespace).List(metav1.ListOptions{LabelSelector: v1.AppLabel})
	if err != nil {
		log.Log.Errorf("Failed to list rbs: %v", err)
		return objectsDeleted, err
	}
	for _, rb := range rbList.Items {
		err := clientset.RbacV1().RoleBindings(kv.Namespace).Delete(rb.Name, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete rb %+v: %v", rb, err)
			return objectsDeleted, err
		} else {
			objectsDeleted++
		}
	}

	rList, err := clientset.RbacV1().Roles(kv.Namespace).List(metav1.ListOptions{LabelSelector: v1.AppLabel})
	if err != nil {
		log.Log.Errorf("Failed to list roles: %v", err)
		return objectsDeleted, err
	}
	for _, role := range rList.Items {
		err := clientset.RbacV1().Roles(kv.Namespace).Delete(role.Name, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete crd %+v: %v", role, err)
			return objectsDeleted, err
		} else {
			objectsDeleted++
		}
	}

	saList, err := clientset.CoreV1().ServiceAccounts(kv.Namespace).List(metav1.ListOptions{LabelSelector: v1.AppLabel})
	if err != nil {
		log.Log.Errorf("Failed to list serviceaccounts: %v", err)
		return objectsDeleted, err
	}
	for _, sa := range saList.Items {
		err := clientset.CoreV1().ServiceAccounts(kv.Namespace).Delete(sa.Name, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete serviceaccount %+v: %v", sa, err)
			return objectsDeleted, err
		} else {
			objectsDeleted++
		}
	}

	// delete CRDs
	ext, err := extclient.NewForConfig(clientset.Config())
	crdList, err := ext.ApiextensionsV1beta1().CustomResourceDefinitions().List(metav1.ListOptions{LabelSelector: v1.AppLabel})
	if err != nil {
		log.Log.Errorf("Failed to list crds: %v", err)
		return objectsDeleted, err
	}
	for _, crd := range crdList.Items {
		err := ext.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crd.Name, deleteOptions)
		if err != nil {
			log.Log.Errorf("Failed to delete crd %+v: %v", crd, err)
			return objectsDeleted, err
		} else {
			objectsDeleted++
		}
	}

	return objectsDeleted, nil

}
