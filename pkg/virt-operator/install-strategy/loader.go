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
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/ghodss/yaml"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

type InstallStrategy struct {
	serviceAccounts []*corev1.ServiceAccount

	clusterRoles        []*rbacv1.ClusterRole
	clusterRoleBindings []*rbacv1.ClusterRoleBinding

	roles        []*rbacv1.Role
	roleBindings []*rbacv1.RoleBinding

	crds []*extv1beta1.CustomResourceDefinition

	services    []*corev1.Service
	deployments []*appsv1.Deployment
	daemonSets  []*appsv1.DaemonSet
}

func LoadInstallStrategyFromFile(filePath string) (*InstallStrategy, error) {

	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return ParseInstallStrategy(string(b))

}

func CreateAll(kv *v1.KubeVirt,
	strategy *InstallStrategy,
	config util.KubeVirtDeploymentConfig,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) (int, error) {

	kvkey, err := controller.KeyFunc(kv)

	objectsAdded := 0
	ext := clientset.ExtensionsClient()
	core := clientset.CoreV1()
	rbac := clientset.RbacV1()
	apps := clientset.AppsV1()

	// CRDs
	for _, crd := range strategy.crds {
		if _, exists, _ := stores.CrdCache.Get(crd); !exists {
			expectations.Crd.RaiseExpectations(kvkey, 1, 0)
			_, err := ext.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
			if err != nil {
				expectations.Crd.LowerExpectations(kvkey, 1, 0)
				return objectsAdded, fmt.Errorf("unable to create crd %+v: %v", crd, err)
			} else if err == nil {
				objectsAdded++
			}
		} else {
			log.Log.V(4).Infof("crd %v already exists", crd.GetName())
		}
	}

	// ServiceAccounts
	for _, sa := range strategy.serviceAccounts {
		if _, exists, _ := stores.ServiceAccountCache.Get(sa); !exists {
			expectations.ServiceAccount.RaiseExpectations(kvkey, 1, 0)
			_, err := core.ServiceAccounts(kv.Namespace).Create(sa)
			if err != nil {
				expectations.ServiceAccount.LowerExpectations(kvkey, 1, 0)
				return objectsAdded, fmt.Errorf("unable to create serviceaccount %+v: %v", sa, err)
			} else if err == nil {
				objectsAdded++
			}
		} else {
			log.Log.V(4).Infof("serviceaccount %v already exists", sa.GetName())
		}
	}

	// ClusterRoles
	for _, cr := range strategy.clusterRoles {
		if _, exists, _ := stores.ClusterRoleCache.Get(cr); !exists {
			expectations.ClusterRole.RaiseExpectations(kvkey, 1, 0)
			_, err := rbac.ClusterRoles().Create(cr)
			if err != nil {
				expectations.ClusterRole.LowerExpectations(kvkey, 1, 0)
				return objectsAdded, fmt.Errorf("unable to create clusterrole %+v: %v", cr, err)
			} else if err == nil {
				objectsAdded++
			}
		} else {
			log.Log.V(4).Infof("clusterrole %v already exists", cr.GetName())
		}
	}

	// ClusterRoleBindings
	for _, crb := range strategy.clusterRoleBindings {
		if _, exists, _ := stores.ClusterRoleBindingCache.Get(crb); !exists {
			expectations.ClusterRoleBinding.RaiseExpectations(kvkey, 1, 0)
			_, err := rbac.ClusterRoleBindings().Create(crb)
			if err != nil {
				expectations.ClusterRoleBinding.LowerExpectations(kvkey, 1, 0)
				return objectsAdded, fmt.Errorf("unable to create clusterrolebinding %+v: %v", crb, err)
			} else if err == nil {
				objectsAdded++
			}
		} else {
			log.Log.V(4).Infof("clusterrolebinding %v already exists", crb.GetName())
		}
	}

	// Roles
	for _, r := range strategy.roles {
		if _, exists, _ := stores.RoleCache.Get(r); !exists {
			expectations.Role.RaiseExpectations(kvkey, 1, 0)
			_, err := rbac.Roles(kv.Namespace).Create(r)
			if err != nil {
				expectations.Role.LowerExpectations(kvkey, 1, 0)
				return objectsAdded, fmt.Errorf("unable to create role %+v: %v", r, err)
			} else if err == nil {
				objectsAdded++
			}
		} else {
			log.Log.V(4).Infof("role %v already exists", r.GetName())
		}
	}

	// RoleBindings
	for _, rb := range strategy.roleBindings {
		if _, exists, _ := stores.RoleBindingCache.Get(rb); !exists {
			expectations.RoleBinding.RaiseExpectations(kvkey, 1, 0)
			_, err := rbac.RoleBindings(kv.Namespace).Create(rb)
			if err != nil {
				expectations.RoleBinding.LowerExpectations(kvkey, 1, 0)
				return objectsAdded, fmt.Errorf("unable to create rolebinding %+v: %v", rb, err)
			} else if err == nil {
				objectsAdded++
			}
		} else {
			log.Log.V(4).Infof("rolebinding %v already exists", rb.GetName())
		}
	}

	// Services
	for _, service := range strategy.services {
		if _, exists, _ := stores.ServiceCache.Get(service); !exists {
			expectations.Service.RaiseExpectations(kvkey, 1, 0)
			_, err := core.Services(kv.Namespace).Create(service)
			if err != nil {
				expectations.Service.LowerExpectations(kvkey, 1, 0)
				return objectsAdded, fmt.Errorf("unable to create service %+v: %v", service, err)
			} else if err == nil {
				objectsAdded++
			}
		} else {
			log.Log.V(4).Infof("service %v already exists", service.GetName())
		}
	}

	// Deployments
	for _, deployment := range strategy.deployments {
		if _, exists, _ := stores.DeploymentCache.Get(deployment); !exists {
			expectations.Deployment.RaiseExpectations(kvkey, 1, 0)
			_, err := apps.Deployments(kv.Namespace).Create(deployment)
			if err != nil {
				expectations.Deployment.LowerExpectations(kvkey, 1, 0)
				return objectsAdded, fmt.Errorf("unable to create deployment %+v: %v", deployment, err)
			} else if err == nil {
				objectsAdded++
			}
		} else {
			log.Log.V(4).Infof("deployment %v already exists", deployment.GetName())
		}
	}

	// Daemonsets
	for _, daemonSet := range strategy.daemonSets {
		if _, exists, _ := stores.DaemonSetCache.Get(daemonSet); !exists {
			expectations.DaemonSet.RaiseExpectations(kvkey, 1, 0)
			_, err = apps.DaemonSets(kv.Namespace).Create(daemonSet)
			if err != nil {
				expectations.DaemonSet.LowerExpectations(kvkey, 1, 0)
				return objectsAdded, fmt.Errorf("unable to create daemonset %+v: %v", daemonSet, err)
			} else if err == nil {
				objectsAdded++
			}
		} else {
			log.Log.V(4).Infof("daemonset %v already exists", daemonSet.GetName())
		}
	}

	return objectsAdded, nil
}

func ParseInstallStrategy(data string) (*InstallStrategy, error) {
	strategy := &InstallStrategy{}
	entries := strings.Split(data, "---")

	for _, entry := range entries {
		entry := strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		var obj metav1.TypeMeta
		if err := yaml.Unmarshal([]byte(entry), &obj); err != nil {
			return nil, err
		}

		switch obj.Kind {
		case "ServiceAccount":
			sa := &corev1.ServiceAccount{}
			if err := yaml.Unmarshal([]byte(entry), &sa); err != nil {
				return nil, err
			}
			strategy.serviceAccounts = append(strategy.serviceAccounts, sa)
		case "ClusterRole":
			cr := &rbacv1.ClusterRole{}
			if err := yaml.Unmarshal([]byte(entry), &cr); err != nil {
				return nil, err
			}
			strategy.clusterRoles = append(strategy.clusterRoles, cr)
		case "ClusterRoleBinding":
			crb := &rbacv1.ClusterRoleBinding{}
			if err := yaml.Unmarshal([]byte(entry), &crb); err != nil {
				return nil, err
			}
			strategy.clusterRoleBindings = append(strategy.clusterRoleBindings, crb)
		case "Role":
			r := &rbacv1.Role{}
			if err := yaml.Unmarshal([]byte(entry), &r); err != nil {
				return nil, err
			}
			strategy.roles = append(strategy.roles, r)
		case "RoleBinding":
			rb := &rbacv1.RoleBinding{}
			if err := yaml.Unmarshal([]byte(entry), &rb); err != nil {
				return nil, err
			}
			strategy.roleBindings = append(strategy.roleBindings, rb)
		case "Service":
			s := &corev1.Service{}
			if err := yaml.Unmarshal([]byte(entry), &s); err != nil {
				return nil, err
			}
			strategy.services = append(strategy.services, s)
		case "Deployment":
			d := &appsv1.Deployment{}
			if err := yaml.Unmarshal([]byte(entry), &d); err != nil {
				return nil, err
			}
			strategy.deployments = append(strategy.deployments, d)
		case "DaemonSet":
			d := &appsv1.DaemonSet{}
			if err := yaml.Unmarshal([]byte(entry), &d); err != nil {
				return nil, err
			}
			strategy.daemonSets = append(strategy.daemonSets, d)
		case "CustomResourceDefinition":
			crd := &extv1beta1.CustomResourceDefinition{}
			if err := yaml.Unmarshal([]byte(entry), &crd); err != nil {
				return nil, err
			}
			strategy.crds = append(strategy.crds, crd)
		case "Namespace":
			// skipped. We don't do anything with namespaces
		default:
			return nil, fmt.Errorf("UNKNOWN TYPE %s detected", obj.Kind)

		}
		log.Log.Infof("%s loaded", obj.Kind)
	}
	return strategy, nil
}
