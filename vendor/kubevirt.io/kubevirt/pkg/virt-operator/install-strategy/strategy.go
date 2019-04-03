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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghodss/yaml"

	secv1 "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	kvutil "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/rbac"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
	marshalutil "kubevirt.io/kubevirt/tools/util"
)

const customSCCPrivilegedAccountsType = "KubevirtCustomSCCRule"

type customSCCPrivilegedAccounts struct {
	// this isn't a real k8s object. We use the meta type
	// because it gives a consistent way to separate k8s
	// objects from our custom actions
	metav1.TypeMeta `json:",inline"`

	// this is the target scc we're adding service accounts to
	TargetSCC string `json:"TargetSCC"`

	// these are the service accounts being added to the scc
	ServiceAccounts []string `json:"serviceAccounts"`
}

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

	customSCCPrivileges []*customSCCPrivilegedAccounts
}

func NewInstallStrategyConfigMap(namespace string, imageTag string, imageRegistry string) (*corev1.ConfigMap, error) {

	strategy, err := GenerateCurrentInstallStrategy(
		namespace,
		imageTag,
		imageRegistry,
		corev1.PullIfNotPresent,
		"2")
	if err != nil {
		return nil, err
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kubevirt-install-strategy-",
			Namespace:    namespace,
			Labels: map[string]string{
				v1.ManagedByLabel:       v1.ManagedByLabelOperatorValue,
				v1.InstallStrategyLabel: "",
			},
			Annotations: map[string]string{
				v1.InstallStrategyVersionAnnotation:  imageTag,
				v1.InstallStrategyRegistryAnnotation: imageRegistry,
			},
		},
		Data: map[string]string{
			"manifests": string(dumpInstallStrategyToBytes(strategy)),
		},
	}
	return configMap, nil
}

func DumpInstallStrategyToConfigMap(clientset kubecli.KubevirtClient) error {

	conf := util.GetConfig()
	imageTag := conf.ImageTag
	imageRegistry := conf.ImageRegistry

	namespace, err := kvutil.GetNamespace()
	if err != nil {
		return err
	}

	configMap, err := NewInstallStrategyConfigMap(namespace, imageTag, imageRegistry)
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().ConfigMaps(namespace).Create(configMap)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// force update if already exists
			_, err = clientset.CoreV1().ConfigMaps(namespace).Update(configMap)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func dumpInstallStrategyToBytes(strategy *InstallStrategy) []byte {

	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	for _, entry := range strategy.serviceAccounts {
		marshalutil.MarshallObject(entry, writer)
	}
	for _, entry := range strategy.clusterRoles {
		marshalutil.MarshallObject(entry, writer)
	}
	for _, entry := range strategy.clusterRoleBindings {
		marshalutil.MarshallObject(entry, writer)
	}
	for _, entry := range strategy.roles {
		marshalutil.MarshallObject(entry, writer)
	}
	for _, entry := range strategy.roleBindings {
		marshalutil.MarshallObject(entry, writer)
	}
	for _, entry := range strategy.crds {
		marshalutil.MarshallObject(entry, writer)
	}
	for _, entry := range strategy.services {
		marshalutil.MarshallObject(entry, writer)
	}
	for _, entry := range strategy.deployments {
		marshalutil.MarshallObject(entry, writer)
	}
	for _, entry := range strategy.daemonSets {
		marshalutil.MarshallObject(entry, writer)
	}
	for _, entry := range strategy.customSCCPrivileges {
		marshalutil.MarshallObject(entry, writer)
	}
	writer.Flush()

	return b.Bytes()
}

func GenerateCurrentInstallStrategy(namespace string,
	version string,
	repository string,
	imagePullPolicy corev1.PullPolicy,
	verbosity string) (*InstallStrategy, error) {

	strategy := &InstallStrategy{}

	strategy.crds = append(strategy.crds, components.NewVirtualMachineInstanceCrd())
	strategy.crds = append(strategy.crds, components.NewPresetCrd())
	strategy.crds = append(strategy.crds, components.NewReplicaSetCrd())
	strategy.crds = append(strategy.crds, components.NewVirtualMachineCrd())
	strategy.crds = append(strategy.crds, components.NewVirtualMachineInstanceMigrationCrd())

	rbaclist := make([]interface{}, 0)
	rbaclist = append(rbaclist, rbac.GetAllCluster(namespace)...)
	rbaclist = append(rbaclist, rbac.GetAllApiServer(namespace)...)
	rbaclist = append(rbaclist, rbac.GetAllController(namespace)...)
	rbaclist = append(rbaclist, rbac.GetAllHandler(namespace)...)

	for _, entry := range rbaclist {
		cr, ok := entry.(*rbacv1.ClusterRole)
		if ok {
			strategy.clusterRoles = append(strategy.clusterRoles, cr)
		}
		crb, ok := entry.(*rbacv1.ClusterRoleBinding)
		if ok {
			strategy.clusterRoleBindings = append(strategy.clusterRoleBindings, crb)
		}

		r, ok := entry.(*rbacv1.Role)
		if ok {
			strategy.roles = append(strategy.roles, r)
		}

		rb, ok := entry.(*rbacv1.RoleBinding)
		if ok {
			strategy.roleBindings = append(strategy.roleBindings, rb)
		}

		sa, ok := entry.(*corev1.ServiceAccount)
		if ok {
			strategy.serviceAccounts = append(strategy.serviceAccounts, sa)
		}
	}

	strategy.services = append(strategy.services, components.NewPrometheusService(namespace))

	strategy.services = append(strategy.services, components.NewApiServerService(namespace))
	apiDeployment, err := components.NewApiServerDeployment(namespace, repository, version, imagePullPolicy, verbosity)
	if err != nil {
		return nil, fmt.Errorf("error generating virt-apiserver deployment %v", err)
	}
	strategy.deployments = append(strategy.deployments, apiDeployment)

	controller, err := components.NewControllerDeployment(namespace, repository, version, imagePullPolicy, verbosity)
	if err != nil {
		return nil, fmt.Errorf("error generating virt-controller deployment %v", err)
	}
	strategy.deployments = append(strategy.deployments, controller)

	handler, err := components.NewHandlerDaemonSet(namespace, repository, version, imagePullPolicy, verbosity)
	if err != nil {
		return nil, fmt.Errorf("error generating virt-handler deployment %v", err)
	}
	strategy.daemonSets = append(strategy.daemonSets, handler)

	prefix := "system:serviceaccount"
	typeMeta := metav1.TypeMeta{
		Kind: customSCCPrivilegedAccountsType,
	}
	strategy.customSCCPrivileges = append(strategy.customSCCPrivileges, &customSCCPrivilegedAccounts{
		TypeMeta:  typeMeta,
		TargetSCC: "privileged",
		ServiceAccounts: []string{
			fmt.Sprintf("%s:%s:%s", prefix, namespace, "kubevirt-handler"),
			fmt.Sprintf("%s:%s:%s", prefix, namespace, "kubevirt-apiserver"),
			fmt.Sprintf("%s:%s:%s", prefix, namespace, "kubevirt-controller"),
		},
	})

	return strategy, nil
}

func mostRecentConfigMap(configMaps []*corev1.ConfigMap) *corev1.ConfigMap {
	var configMap *corev1.ConfigMap
	// choose the most recent configmap if multiple match.
	mostRecentTime := metav1.Time{}
	for _, config := range configMaps {
		if configMap == nil {
			configMap = config
			mostRecentTime = config.ObjectMeta.CreationTimestamp
		} else if mostRecentTime.Before(&config.ObjectMeta.CreationTimestamp) {
			configMap = config
			mostRecentTime = config.ObjectMeta.CreationTimestamp
		}
	}
	return configMap
}

func LoadInstallStrategyFromCache(stores util.Stores, namespace string, imageTag string, imageRegistry string) (*InstallStrategy, error) {
	var configMap *corev1.ConfigMap
	var matchingConfigMaps []*corev1.ConfigMap

	for _, obj := range stores.InstallStrategyConfigMapCache.List() {
		config, ok := obj.(*corev1.ConfigMap)
		if !ok {
			continue
		}
		if config.ObjectMeta.Annotations == nil {
			continue
		}

		version, _ := config.ObjectMeta.Annotations[v1.InstallStrategyVersionAnnotation]
		registry, _ := config.ObjectMeta.Annotations[v1.InstallStrategyRegistryAnnotation]
		if version == imageTag && registry == imageRegistry {
			matchingConfigMaps = append(matchingConfigMaps, config)
		}
	}

	if len(matchingConfigMaps) == 0 {
		return nil, fmt.Errorf("no install strategy configmap found for version %s with registry %s", imageTag, imageRegistry)
	}

	// choose the most recent configmap if multiple match.
	configMap = mostRecentConfigMap(matchingConfigMaps)

	data, ok := configMap.Data["manifests"]
	if !ok {
		return nil, fmt.Errorf("install strategy configmap %s does not contain 'manifests' key", configMap.Name)
	}

	strategy, err := loadInstallStrategyFromBytes(data)
	if err != nil {
		return nil, err
	}

	return strategy, nil
}

func loadInstallStrategyFromBytes(data string) (*InstallStrategy, error) {
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
		case customSCCPrivilegedAccountsType:
			priv := &customSCCPrivilegedAccounts{}
			if err := yaml.Unmarshal([]byte(entry), &priv); err != nil {
				return nil, err
			}
			strategy.customSCCPrivileges = append(strategy.customSCCPrivileges, priv)
		default:
			return nil, fmt.Errorf("UNKNOWN TYPE %s detected", obj.Kind)

		}
		log.Log.Infof("%s loaded", obj.Kind)
	}
	return strategy, nil
}

func addOperatorLabel(objectMeta *metav1.ObjectMeta) {
	if objectMeta.Labels == nil {
		objectMeta.Labels = make(map[string]string)
	}
	objectMeta.Labels[v1.ManagedByLabel] = v1.ManagedByLabelOperatorValue
}

func remove(users []string, user string) ([]string, bool) {
	var newUsers []string
	modified := false
	for _, u := range users {
		if u != user {
			newUsers = append(newUsers, u)
		} else {
			modified = true
		}
	}
	return newUsers, modified
}

func contains(users []string, user string) bool {
	for _, u := range users {
		if u == user {
			return true
		}
	}
	return false
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

	// delete handler daemonset
	obj, exists, err := stores.DaemonSetCache.GetByKey(fmt.Sprintf("%s/%s", kv.Namespace, "virt-handler"))
	if err != nil {
		log.Log.Errorf("Failed to get virt-handler: %v", err)
		return err
	} else if exists {
		if ds, ok := obj.(*appsv1.DaemonSet); ok && ds.DeletionTimestamp == nil {
			if key, err := controller.KeyFunc(ds); err == nil {
				expectations.DaemonSet.AddExpectedDeletion(kvkey, key)
				err := clientset.AppsV1().DaemonSets(kv.Namespace).Delete("virt-handler", deleteOptions)
				if err != nil {
					expectations.DaemonSet.DeletionObserved(kvkey, key)
					log.Log.Errorf("Failed to delete virt-handler: %v", err)
					return err
				}
			}
		} else if !ok {
			log.Log.Errorf("Cast failed! obj: %+v", obj)
			return nil
		}
	}

	// delete controller and apiserver deployment
	for _, name := range []string{"virt-controller", "virt-api"} {
		obj, exists, err := stores.DeploymentCache.GetByKey(fmt.Sprintf("%s/%s", kv.Namespace, name))
		if err != nil {
			log.Log.Errorf("Failed to get %v: %v", name, err)
			return err
		} else if exists {
			if depl, ok := obj.(*appsv1.Deployment); ok && depl.DeletionTimestamp == nil {
				if key, err := controller.KeyFunc(depl); err == nil {
					expectations.Deployment.AddExpectedDeletion(kvkey, key)
					err := clientset.AppsV1().Deployments(kv.Namespace).Delete(name, deleteOptions)
					if err != nil {
						expectations.Deployment.DeletionObserved(kvkey, key)
						log.Log.Errorf("Failed to delete virt-handler: %v", err)
						return err
					}
				}
			} else if !ok {
				log.Log.Errorf("Cast failed! obj: %+v", obj)
				return nil
			}
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
		privSCCCopy := privSCC.DeepCopy()

		modified := false
		users := privSCCCopy.Users
		for _, acc := range sccPriv.ServiceAccounts {
			removed := false
			users, removed = remove(users, acc)
			modified = modified || removed
		}

		if modified {
			userBytes, err := json.Marshal(users)
			if err != nil {
				return err
			}

			data := []byte(fmt.Sprintf(`{"users": %s}`, userBytes))
			_, err = scc.SecurityContextConstraints().Patch(sccPriv.TargetSCC, types.StrategicMergePatchType, data)
			if err != nil {
				return fmt.Errorf("unable to patch scc: %v", err)
			}
		}
	}

	return nil

}

func CreateAll(kv *v1.KubeVirt,
	strategy *InstallStrategy,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) (int, error) {

	kvkey, err := controller.KeyFunc(kv)

	objectsAdded := 0
	ext := clientset.ExtensionsClient()
	core := clientset.CoreV1()
	rbac := clientset.RbacV1()
	apps := clientset.AppsV1()
	scc := clientset.SecClient()

	// CRDs
	for _, crd := range strategy.crds {
		addOperatorLabel(&crd.ObjectMeta)
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
		addOperatorLabel(&sa.ObjectMeta)
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
		addOperatorLabel(&cr.ObjectMeta)
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
		addOperatorLabel(&crb.ObjectMeta)
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
		addOperatorLabel(&r.ObjectMeta)
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
		addOperatorLabel(&rb.ObjectMeta)
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
		addOperatorLabel(&service.ObjectMeta)
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
		addOperatorLabel(&deployment.ObjectMeta)
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
		addOperatorLabel(&daemonSet.ObjectMeta)
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

	// Add service accounts to SCC
	for _, sccPriv := range strategy.customSCCPrivileges {
		privSCCObj, exists, err := stores.SCCCache.GetByKey(sccPriv.TargetSCC)
		if !exists {
			return objectsAdded, nil
		} else if err != nil {
			return objectsAdded, err
		}

		privSCC, ok := privSCCObj.(*secv1.SecurityContextConstraints)
		if !ok {
			return objectsAdded, fmt.Errorf("couldn't cast object to SecurityContextConstraints: %+v", privSCCObj)
		}
		privSCCCopy := privSCC.DeepCopy()

		modified := false
		users := privSCCCopy.Users
		for _, acc := range sccPriv.ServiceAccounts {
			if !contains(users, acc) {
				users = append(users, acc)
				modified = true
			}
		}

		if modified {
			userBytes, err := json.Marshal(users)
			if err != nil {
				return objectsAdded, err
			}

			data := []byte(fmt.Sprintf(`{"users": %s}`, userBytes))
			_, err = scc.SecurityContextConstraints().Patch(sccPriv.TargetSCC, types.StrategicMergePatchType, data)
			if err != nil {
				return objectsAdded, fmt.Errorf("unable to patch scc: %v", err)
			}
		}
	}

	return objectsAdded, nil
}
