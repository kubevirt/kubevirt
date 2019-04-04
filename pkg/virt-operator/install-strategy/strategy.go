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
	"fmt"
	"strings"

	"github.com/ghodss/yaml"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
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
