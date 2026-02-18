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

package components

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"strings"
	"sync"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	apiregscheme "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/scheme"

	v1 "kubevirt.io/api/core/v1"

	operatorutil "kubevirt.io/kubevirt/pkg/virt-operator/util"
)

const (
	VirtTemplateApiserverDeploymentName  = "virt-template-apiserver"
	VirtTemplateControllerDeploymentName = "virt-template-controller"
)

//go:embed data/virt-template/install-virt-operator.yaml
var virtTemplateBundle []byte

var (
	parseOnce       sync.Once
	parsedBundle    *VirtTemplateResources
	parsedBundleErr error
)

// VirtTemplateResources holds all resources parsed from the virt-template bundle
type VirtTemplateResources struct {
	CRDs                              []*extv1.CustomResourceDefinition
	ServiceAccounts                   []*corev1.ServiceAccount
	Roles                             []*rbacv1.Role
	ClusterRoles                      []*rbacv1.ClusterRole
	RoleBindings                      []*rbacv1.RoleBinding
	ClusterRoleBindings               []*rbacv1.ClusterRoleBinding
	Services                          []*corev1.Service
	Deployments                       []*appsv1.Deployment
	ValidatingAdmissionPolicies       []*admissionregistrationv1.ValidatingAdmissionPolicy
	ValidatingAdmissionPolicyBindings []*admissionregistrationv1.ValidatingAdmissionPolicyBinding
	ValidatingWebhookConfigurations   []*admissionregistrationv1.ValidatingWebhookConfiguration
	APIServices                       []*apiregv1.APIService
	NetworkPolicies                   []*networkingv1.NetworkPolicy
}

// NewVirtTemplateResources parses the embedded virt-template YAML bundle once and returns
// a deep copy of all resources with the specified config applied.
func NewVirtTemplateResources(config *operatorutil.KubeVirtDeploymentConfig) (*VirtTemplateResources, error) {
	parseOnce.Do(func() {
		parsedBundle, parsedBundleErr = parseBundle()
	})
	if parsedBundleErr != nil {
		return nil, parsedBundleErr
	}

	return parsedBundle.deepCopyWithConfig(config)
}

func parseBundle() (*VirtTemplateResources, error) {
	s := runtime.NewScheme()
	if err := scheme.AddToScheme(s); err != nil {
		return nil, err
	}
	if err := extscheme.AddToScheme(s); err != nil {
		return nil, err
	}
	if err := apiregscheme.AddToScheme(s); err != nil {
		return nil, err
	}

	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(virtTemplateBundle), 4096)
	deserializer := serializer.NewCodecFactory(s).UniversalDeserializer()

	resources := &VirtTemplateResources{}
	for {
		var rawExt runtime.RawExtension
		if err := decoder.Decode(&rawExt); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if rawExt.Raw == nil {
			continue
		}

		obj, _, err := deserializer.Decode(rawExt.Raw, nil, nil)
		if err != nil {
			return nil, err
		}

		if err := resources.addObject(obj); err != nil {
			return nil, err
		}
	}

	return resources, nil
}

func (r *VirtTemplateResources) addObject(obj runtime.Object) error {
	switch typedObj := obj.(type) {
	case *extv1.CustomResourceDefinition:
		r.CRDs = append(r.CRDs, typedObj)
	case *corev1.ServiceAccount:
		r.ServiceAccounts = append(r.ServiceAccounts, typedObj)
	case *rbacv1.Role:
		r.Roles = append(r.Roles, typedObj)
	case *rbacv1.ClusterRole:
		r.ClusterRoles = append(r.ClusterRoles, typedObj)
	case *rbacv1.RoleBinding:
		r.RoleBindings = append(r.RoleBindings, typedObj)
	case *rbacv1.ClusterRoleBinding:
		r.ClusterRoleBindings = append(r.ClusterRoleBindings, typedObj)
	case *corev1.Service:
		r.Services = append(r.Services, typedObj)
	case *appsv1.Deployment:
		r.Deployments = append(r.Deployments, typedObj)
	case *admissionregistrationv1.ValidatingAdmissionPolicy:
		r.ValidatingAdmissionPolicies = append(r.ValidatingAdmissionPolicies, typedObj)
	case *admissionregistrationv1.ValidatingAdmissionPolicyBinding:
		r.ValidatingAdmissionPolicyBindings = append(r.ValidatingAdmissionPolicyBindings, typedObj)
	case *admissionregistrationv1.ValidatingWebhookConfiguration:
		r.ValidatingWebhookConfigurations = append(r.ValidatingWebhookConfigurations, typedObj)
	case *apiregv1.APIService:
		r.APIServices = append(r.APIServices, typedObj)
	case *networkingv1.NetworkPolicy:
		r.NetworkPolicies = append(r.NetworkPolicies, typedObj)
	default:
		return fmt.Errorf("unknown object type: %T", obj)
	}

	return nil
}

func (r *VirtTemplateResources) deepCopyWithConfig(config *operatorutil.KubeVirtDeploymentConfig) (*VirtTemplateResources, error) {
	resources := &VirtTemplateResources{}

	for _, obj := range r.CRDs {
		resources.CRDs = append(resources.CRDs, obj.DeepCopy())
	}
	for _, obj := range r.ServiceAccounts {
		copied := obj.DeepCopy()
		copied.SetNamespace(config.GetNamespace())
		resources.ServiceAccounts = append(resources.ServiceAccounts, copied)
	}
	for _, obj := range r.Roles {
		copied := obj.DeepCopy()
		copied.SetNamespace(config.GetNamespace())
		resources.Roles = append(resources.Roles, copied)
	}
	for _, obj := range r.ClusterRoles {
		resources.ClusterRoles = append(resources.ClusterRoles, obj.DeepCopy())
	}
	for _, obj := range r.RoleBindings {
		copied := obj.DeepCopy()
		UpdateTemplateAuthReaderNamespace(copied, config.GetNamespace())
		updateSubjectsNamespace(copied.Subjects, config.GetNamespace())
		resources.RoleBindings = append(resources.RoleBindings, copied)
	}
	for _, obj := range r.ClusterRoleBindings {
		copied := obj.DeepCopy()
		updateSubjectsNamespace(copied.Subjects, config.GetNamespace())
		resources.ClusterRoleBindings = append(resources.ClusterRoleBindings, copied)
	}
	for _, obj := range r.Services {
		copied := obj.DeepCopy()
		copied.SetNamespace(config.GetNamespace())
		resources.Services = append(resources.Services, copied)
	}
	for _, obj := range r.Deployments {
		copied := obj.DeepCopy()
		copied.SetNamespace(config.GetNamespace())
		if err := updateDeployment(copied, config); err != nil {
			return nil, err
		}
		resources.Deployments = append(resources.Deployments, copied)
	}
	for _, obj := range r.ValidatingAdmissionPolicies {
		resources.ValidatingAdmissionPolicies = append(resources.ValidatingAdmissionPolicies, obj.DeepCopy())
	}
	for _, obj := range r.ValidatingAdmissionPolicyBindings {
		resources.ValidatingAdmissionPolicyBindings = append(resources.ValidatingAdmissionPolicyBindings, obj.DeepCopy())
	}
	for _, obj := range r.ValidatingWebhookConfigurations {
		copied := obj.DeepCopy()
		updateWebhooksServiceNamespace(copied.Webhooks, config.GetNamespace())
		resources.ValidatingWebhookConfigurations = append(resources.ValidatingWebhookConfigurations, copied)
	}
	for _, obj := range r.APIServices {
		copied := obj.DeepCopy()
		updateAPIServiceNamespace(copied, config.GetNamespace())
		resources.APIServices = append(resources.APIServices, copied)
	}
	for _, obj := range r.NetworkPolicies {
		copied := obj.DeepCopy()
		copied.SetNamespace(config.GetNamespace())
		resources.NetworkPolicies = append(resources.NetworkPolicies, copied)
	}

	return resources, nil
}

func UpdateTemplateAuthReaderNamespace(rb *rbacv1.RoleBinding, namespace string) {
	// RoleBinding virt-template-apiserver-auth-reader must be in namespace kube-system
	if rb.GetName() == "virt-template-apiserver-auth-reader" {
		rb.SetNamespace("kube-system")
	} else {
		rb.SetNamespace(namespace)
	}
}

func updateSubjectsNamespace(subjects []rbacv1.Subject, namespace string) {
	for i := range subjects {
		subjects[i].Namespace = namespace
	}
}

func updateWebhooksServiceNamespace(webhooks []admissionregistrationv1.ValidatingWebhook, namespace string) {
	for i := range webhooks {
		if webhooks[i].ClientConfig.Service != nil {
			webhooks[i].ClientConfig.Service.Namespace = namespace
		}
	}
}

func updateAPIServiceNamespace(apiSvc *apiregv1.APIService, namespace string) {
	if apiSvc.Spec.Service != nil {
		apiSvc.Spec.Service.Namespace = namespace
	}
}

func updateDeployment(deployment *appsv1.Deployment, config *operatorutil.KubeVirtDeploymentConfig) error {
	if deployment.Spec.Replicas != nil && *deployment.Spec.Replicas != 1 {
		return fmt.Errorf("expected exactly 1 replica in deployment %s, got %d", deployment.Name, *deployment.Spec.Replicas)
	}
	if containers := len(deployment.Spec.Template.Spec.Containers); containers != 1 {
		return fmt.Errorf("expected exactly 1 container in deployment %s, got %d", deployment.Name, containers)
	}

	if deployment.ObjectMeta.Labels == nil {
		deployment.ObjectMeta.Labels = map[string]string{}
	}
	if deployment.Spec.Template.ObjectMeta.Labels == nil {
		deployment.Spec.Template.ObjectMeta.Labels = map[string]string{}
	}
	if config.GetProductVersion() != "" && operatorutil.IsValidLabel(config.GetProductVersion()) {
		deployment.ObjectMeta.Labels[v1.AppVersionLabel] = config.GetProductVersion()
		deployment.Spec.Template.ObjectMeta.Labels[v1.AppVersionLabel] = config.GetProductVersion()
	}
	if config.GetProductName() != "" && operatorutil.IsValidLabel(config.GetProductName()) {
		deployment.ObjectMeta.Labels[v1.AppPartOfLabel] = config.GetProductName()
		deployment.Spec.Template.ObjectMeta.Labels[v1.AppPartOfLabel] = config.GetProductName()
	}
	if config.GetProductComponent() != "" && operatorutil.IsValidLabel(config.GetProductComponent()) {
		deployment.ObjectMeta.Labels[v1.AppComponentLabel] = config.GetProductComponent()
		deployment.Spec.Template.ObjectMeta.Labels[v1.AppComponentLabel] = config.GetProductComponent()
	}

	if len(config.GetImagePullSecrets()) > 0 {
		deployment.Spec.Template.Spec.ImagePullSecrets = config.GetImagePullSecrets()
	}

	container := &deployment.Spec.Template.Spec.Containers[0]

	if config.GetImageRegistry() != "" || config.GetImagePrefix() != "" {
		container.Image = replaceImageRegistryAndPrefix(container.Image, config.GetImageRegistry(), config.GetImagePrefix())
	}

	if config.GetImagePullPolicy() != "" {
		container.ImagePullPolicy = config.GetImagePullPolicy()
	}

	for key, value := range config.GetExtraEnv() {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	if config.GetVerbosity() != "" {
		container.Args = append(container.Args, "-v", config.GetVerbosity())
	}

	return nil
}

func replaceImageRegistryAndPrefix(image, newRegistry, newPrefix string) string {
	registry := ""
	imageNameAndTagOrDigest := ""

	if lastSlash := strings.LastIndex(image, "/"); lastSlash > -1 {
		registry = image[:lastSlash]
		imageNameAndTagOrDigest = image[lastSlash+1:]
	} else {
		imageNameAndTagOrDigest = image
	}

	if newRegistry == "" {
		newRegistry = registry
	}

	return fmt.Sprintf("%s/%s%s", newRegistry, newPrefix, imageNameAndTagOrDigest)
}
