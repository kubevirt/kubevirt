package install

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

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	secv1 "github.com/openshift/api/security/v1"
	k8coresv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	ext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/rbac"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
	operatorutil "kubevirt.io/kubevirt/pkg/virt-operator/util"
	marshalutil "kubevirt.io/kubevirt/tools/util"
)

const customSCCPrivilegedAccountsType = "KubevirtCustomSCCRule"
const ManifestsEncodingGzipBase64 = "gzip+base64"

//go:generate mockgen -source $GOFILE -imports "libvirt=libvirt.org/libvirt-go" -package=$GOPACKAGE -destination=generated_mock_$GOFILE

type APIServiceInterface interface {
	Get(ctx context.Context, name string, options metav1.GetOptions) (*apiregv1.APIService, error)
	Create(ctx context.Context, apiService *apiregv1.APIService, opts metav1.CreateOptions) (*apiregv1.APIService, error)
	Delete(ctx context.Context, name string, options metav1.DeleteOptions) error
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *apiregv1.APIService, err error)
}

type Strategy struct {
	serviceAccounts []*corev1.ServiceAccount

	clusterRoles        []*rbacv1.ClusterRole
	clusterRoleBindings []*rbacv1.ClusterRoleBinding

	roles        []*rbacv1.Role
	roleBindings []*rbacv1.RoleBinding

	crds []*extv1.CustomResourceDefinition

	services                        []*corev1.Service
	deployments                     []*appsv1.Deployment
	daemonSets                      []*appsv1.DaemonSet
	validatingWebhookConfigurations []*admissionregistrationv1.ValidatingWebhookConfiguration
	mutatingWebhookConfigurations   []*admissionregistrationv1.MutatingWebhookConfiguration
	apiServices                     []*apiregv1.APIService
	certificateSecrets              []*corev1.Secret
	sccs                            []*secv1.SecurityContextConstraints
	serviceMonitors                 []*promv1.ServiceMonitor
	prometheusRules                 []*promv1.PrometheusRule
	configMaps                      []*corev1.ConfigMap
}

func (ins *Strategy) ServiceAccounts() []*corev1.ServiceAccount {
	return ins.serviceAccounts
}

func (ins *Strategy) ClusterRoles() []*rbacv1.ClusterRole {
	return ins.clusterRoles
}

func (ins *Strategy) ClusterRoleBindings() []*rbacv1.ClusterRoleBinding {
	return ins.clusterRoleBindings
}

func (ins *Strategy) Roles() []*rbacv1.Role {
	return ins.roles
}

func (ins *Strategy) RoleBindings() []*rbacv1.RoleBinding {
	return ins.roleBindings
}

func (ins *Strategy) Services() []*corev1.Service {
	return ins.services
}

func (ins *Strategy) Deployments() []*appsv1.Deployment {
	return ins.deployments
}

func (ins *Strategy) ApiDeployments() []*appsv1.Deployment {
	var deployments []*appsv1.Deployment

	for _, deployment := range ins.deployments {
		if !strings.Contains(deployment.Name, "virt-api") {
			continue
		}
		deployments = append(deployments, deployment)
	}

	return deployments
}

func (ins *Strategy) ControllerDeployments() []*appsv1.Deployment {
	var deployments []*appsv1.Deployment

	for _, deployment := range ins.deployments {
		if strings.Contains(deployment.Name, "virt-api") {
			continue
		}
		deployments = append(deployments, deployment)

	}

	return deployments
}

func (ins *Strategy) DaemonSets() []*appsv1.DaemonSet {
	return ins.daemonSets
}

func (ins *Strategy) ValidatingWebhookConfigurations() []*admissionregistrationv1.ValidatingWebhookConfiguration {
	return ins.validatingWebhookConfigurations
}

func (ins *Strategy) MutatingWebhookConfigurations() []*admissionregistrationv1.MutatingWebhookConfiguration {
	return ins.mutatingWebhookConfigurations
}

func (ins *Strategy) APIServices() []*apiregv1.APIService {
	return ins.apiServices
}

func (ins *Strategy) CertificateSecrets() []*corev1.Secret {
	return ins.certificateSecrets
}

func (ins *Strategy) SCCs() []*secv1.SecurityContextConstraints {
	return ins.sccs
}

func (ins *Strategy) ServiceMonitors() []*promv1.ServiceMonitor {
	return ins.serviceMonitors
}

func (ins *Strategy) PrometheusRules() []*promv1.PrometheusRule {
	return ins.prometheusRules
}

func (ins *Strategy) ConfigMaps() []*corev1.ConfigMap {
	return ins.configMaps
}

func (ins *Strategy) CRDs() []*extv1.CustomResourceDefinition {
	return ins.crds
}

func encodeManifests(manifests []byte) (string, error) {
	var buf bytes.Buffer

	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(manifests)
	if err != nil {
		return "", err
	}
	if err = zw.Close(); err != nil {
		return "", err
	}
	base64Strategy := base64.StdEncoding.EncodeToString(buf.Bytes())
	return base64Strategy, nil
}

func decodeManifests(strategy []byte) (string, error) {
	var decodedStrategy strings.Builder

	gzippedStrategy, err := base64.StdEncoding.DecodeString(string(strategy))
	if err != nil {
		return "", err
	}
	buf := bytes.NewBuffer(gzippedStrategy)
	zr, err := gzip.NewReader(buf)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(&decodedStrategy, zr); err != nil {
		return "", err
	}
	return decodedStrategy.String(), nil
}

func NewInstallStrategyConfigMap(config *operatorutil.KubeVirtDeploymentConfig, monitorNamespace string, operatorNamespace string) (*corev1.ConfigMap, error) {
	strategy, err := GenerateCurrentInstallStrategy(config, monitorNamespace, operatorNamespace)
	if err != nil {
		return nil, err
	}

	manifests, err := encodeManifests(dumpInstallStrategyToBytes(strategy))
	if err != nil {
		return nil, err
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kubevirt-install-strategy-",
			Namespace:    config.GetNamespace(),
			Labels: map[string]string{
				v1.ManagedByLabel:       v1.ManagedByLabelOperatorValue,
				v1.InstallStrategyLabel: "",
			},
			Annotations: map[string]string{
				v1.InstallStrategyVersionAnnotation:    config.GetKubeVirtVersion(),
				v1.InstallStrategyRegistryAnnotation:   config.GetImageRegistry(),
				v1.InstallStrategyIdentifierAnnotation: config.GetDeploymentID(),
				v1.InstallStrategyConfigMapEncoding:    ManifestsEncodingGzipBase64,
			},
		},
		Data: map[string]string{
			"manifests": manifests,
		},
	}
	return configMap, nil
}

func getMonitorNamespace(clientset k8coresv1.CoreV1Interface, config *operatorutil.KubeVirtDeploymentConfig) (namespace string, err error) {
	for _, ns := range config.GetMonitorNamespaces() {
		if nsExists, err := isNamespaceExist(clientset, ns); nsExists {
			if saExists, err := isServiceAccountExist(clientset, ns, config.GetMonitorServiceAccount()); saExists {
				return ns, nil
			} else if err != nil {
				return "", err
			}
		} else if err != nil {
			return "", err
		}
	}
	return "", nil
}

func DumpInstallStrategyToConfigMap(clientset kubecli.KubevirtClient, operatorNamespace string) error {

	config, err := util.GetConfigFromEnv()
	if err != nil {
		return err
	}

	monitorNamespace, err := getMonitorNamespace(clientset.CoreV1(), config)
	if err != nil {
		return err
	}

	configMap, err := NewInstallStrategyConfigMap(config, monitorNamespace, operatorNamespace)
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().ConfigMaps(config.GetNamespace()).Create(context.Background(), configMap, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// force update if already exists
			_, err = clientset.CoreV1().ConfigMaps(config.GetNamespace()).Update(context.Background(), configMap, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func dumpInstallStrategyToBytes(strategy *Strategy) []byte {

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
		b, _ := yaml.Marshal(entry)
		writer.Write([]byte("---\n"))
		writer.Write(b)
	}
	for _, entry := range strategy.services {
		marshalutil.MarshallObject(entry, writer)
	}
	for _, entry := range strategy.certificateSecrets {
		marshalutil.MarshallObject(entry, writer)
	}
	for _, entry := range strategy.validatingWebhookConfigurations {
		marshalutil.MarshallObject(entry, writer)
	}
	for _, entry := range strategy.mutatingWebhookConfigurations {
		marshalutil.MarshallObject(entry, writer)
	}
	for _, entry := range strategy.apiServices {
		marshalutil.MarshallObject(entry, writer)
	}
	for _, entry := range strategy.deployments {
		marshalutil.MarshallObject(entry, writer)
	}
	for _, entry := range strategy.daemonSets {
		marshalutil.MarshallObject(entry, writer)
	}
	for _, entry := range strategy.sccs {
		marshalutil.MarshallObject(entry, writer)
	}
	for _, entry := range strategy.serviceMonitors {
		marshalutil.MarshallObject(entry, writer)
	}
	for _, entry := range strategy.prometheusRules {
		marshalutil.MarshallObject(entry, writer)
	}
	for _, entry := range strategy.configMaps {
		marshalutil.MarshallObject(entry, writer)
	}
	writer.Flush()

	return b.Bytes()
}

func GenerateCurrentInstallStrategy(config *operatorutil.KubeVirtDeploymentConfig, monitorNamespace string, operatorNamespace string) (*Strategy, error) {

	strategy := &Strategy{}

	functions := []func() (*extv1.CustomResourceDefinition, error){
		components.NewVirtualMachineInstanceCrd, components.NewPresetCrd, components.NewReplicaSetCrd,
		components.NewVirtualMachineCrd, components.NewVirtualMachineInstanceMigrationCrd,
		components.NewVirtualMachineSnapshotCrd, components.NewVirtualMachineSnapshotContentCrd,
		components.NewVirtualMachineRestoreCrd,
	}
	for _, f := range functions {
		crd, err := f()
		if err != nil {
			return nil, err
		}
		strategy.crds = append(strategy.crds, crd)
	}

	rbaclist := make([]runtime.Object, 0)
	rbaclist = append(rbaclist, rbac.GetAllCluster()...)
	rbaclist = append(rbaclist, rbac.GetAllApiServer(config.GetNamespace())...)
	rbaclist = append(rbaclist, rbac.GetAllController(config.GetNamespace())...)
	rbaclist = append(rbaclist, rbac.GetAllHandler(config.GetNamespace())...)

	if monitorNamespace != "" {

		workloadUpdatesEnabled := config.WorkloadUpdatesEnabled()

		monitorServiceAccount := config.GetMonitorServiceAccount()
		rbaclist = append(rbaclist, rbac.GetAllServiceMonitor(config.GetNamespace(), monitorNamespace, monitorServiceAccount)...)
		strategy.serviceMonitors = append(strategy.serviceMonitors, components.NewServiceMonitorCR(config.GetNamespace(), monitorNamespace, true))
		strategy.prometheusRules = append(strategy.prometheusRules, components.NewPrometheusRuleCR(config.GetNamespace(), workloadUpdatesEnabled))
	} else {
		glog.Warningf("failed to create service monitor resources because namespace %s does not exist", monitorNamespace)
	}

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

	var productName string
	var productVersion string

	if operatorutil.IsValidLabel(config.GetProductName()) {
		productName = config.GetProductName()
	} else {
		log.Log.Errorf("invalid kubevirt.spec.productName: labels must be 63 characters or less, begin and end with alphanumeric characters, and contain only dot, hyphen or dash")
	}
	if operatorutil.IsValidLabel(config.GetProductVersion()) {
		productVersion = config.GetProductVersion()
	} else {
		log.Log.Errorf("invalid kubevirt.spec.productVersion: labels must be 63 characters or less, begin and end with alphanumeric characters, and contain only dot, hyphen or dash")
	}

	strategy.validatingWebhookConfigurations = append(strategy.validatingWebhookConfigurations, components.NewOpertorValidatingWebhookConfiguration(operatorNamespace))
	strategy.validatingWebhookConfigurations = append(strategy.validatingWebhookConfigurations, components.NewVirtAPIValidatingWebhookConfiguration(config.GetNamespace()))
	strategy.mutatingWebhookConfigurations = append(strategy.mutatingWebhookConfigurations, components.NewVirtAPIMutatingWebhookConfiguration(config.GetNamespace()))

	strategy.services = append(strategy.services, components.NewPrometheusService(config.GetNamespace()))
	strategy.services = append(strategy.services, components.NewApiServerService(config.GetNamespace()))
	strategy.services = append(strategy.services, components.NewOperatorWebhookService(operatorNamespace))
	apiDeployment, err := components.NewApiServerDeployment(config.GetNamespace(), config.GetImageRegistry(), config.GetImagePrefix(), config.GetApiVersion(), productName, productVersion, config.GetImagePullPolicy(), config.GetVerbosity(), config.GetExtraEnv())
	if err != nil {
		return nil, fmt.Errorf("error generating virt-apiserver deployment %v", err)
	}
	strategy.deployments = append(strategy.deployments, apiDeployment)

	controller, err := components.NewControllerDeployment(config.GetNamespace(), config.GetImageRegistry(), config.GetImagePrefix(), config.GetControllerVersion(), config.GetLauncherVersion(), productName, productVersion, config.GetImagePullPolicy(), config.GetVerbosity(), config.GetExtraEnv())
	if err != nil {
		return nil, fmt.Errorf("error generating virt-controller deployment %v", err)
	}
	strategy.deployments = append(strategy.deployments, controller)

	strategy.configMaps = append(strategy.configMaps, components.NewKubeVirtCAConfigMap(operatorNamespace))

	handler, err := components.NewHandlerDaemonSet(config.GetNamespace(), config.GetImageRegistry(), config.GetImagePrefix(), config.GetHandlerVersion(), config.GetLauncherVersion(), productName, productVersion, config.GetImagePullPolicy(), config.GetVerbosity(), config.GetExtraEnv())
	if err != nil {
		return nil, fmt.Errorf("error generating virt-handler deployment %v", err)
	}

	strategy.daemonSets = append(strategy.daemonSets, handler)
	strategy.sccs = append(strategy.sccs, components.GetAllSCC(config.GetNamespace())...)
	strategy.apiServices = components.NewVirtAPIAPIServices(config.GetNamespace())
	strategy.certificateSecrets = components.NewCertSecrets(config.GetNamespace(), operatorNamespace)
	strategy.certificateSecrets = append(strategy.certificateSecrets, components.NewCACertSecret(operatorNamespace))
	strategy.configMaps = append(strategy.configMaps, components.NewKubeVirtCAConfigMap(operatorNamespace))

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

func isEncoded(configMap *corev1.ConfigMap) bool {
	_, ok := configMap.Annotations[v1.InstallStrategyConfigMapEncoding]
	return ok
}

func getManifests(configMap *corev1.ConfigMap) (string, error) {
	manifests, ok := configMap.Data["manifests"]
	if !ok {
		return "", fmt.Errorf("install strategy configmap %s does not contain 'manifests' key", configMap.Name)
	}

	if isEncoded(configMap) {
		var err error

		manifests, err = decodeManifests([]byte(manifests))
		if err != nil {
			return "", err
		}
	}
	return manifests, nil
}

func LoadInstallStrategyFromCache(stores util.Stores, config *operatorutil.KubeVirtDeploymentConfig) (*Strategy, error) {
	var matchingConfigMaps []*corev1.ConfigMap

	for _, obj := range stores.InstallStrategyConfigMapCache.List() {
		cm, ok := obj.(*corev1.ConfigMap)
		if !ok {
			continue
		} else if cm.ObjectMeta.Annotations == nil {
			continue
		} else if cm.ObjectMeta.Namespace != config.GetNamespace() {
			continue
		}

		// deprecated, keep it for backwards compatibility
		version, _ := cm.ObjectMeta.Annotations[v1.InstallStrategyVersionAnnotation]
		// deprecated, keep it for backwards compatibility
		registry, _ := cm.ObjectMeta.Annotations[v1.InstallStrategyRegistryAnnotation]
		id, _ := cm.ObjectMeta.Annotations[v1.InstallStrategyIdentifierAnnotation]

		if id == config.GetDeploymentID() ||
			(id == "" && version == config.GetKubeVirtVersion() && registry == config.GetImageRegistry()) {
			matchingConfigMaps = append(matchingConfigMaps, cm)
		}
	}

	if len(matchingConfigMaps) == 0 {
		return nil, fmt.Errorf("no install strategy configmap found for version %s with registry %s", config.GetKubeVirtVersion(), config.GetImageRegistry())
	}

	manifests, err := getManifests(mostRecentConfigMap(matchingConfigMaps))
	if err != nil {
		return nil, err
	}

	strategy, err := loadInstallStrategyFromBytes(manifests)
	if err != nil {
		return nil, err
	}

	return strategy, nil
}

func loadInstallStrategyFromBytes(data string) (*Strategy, error) {
	strategy := &Strategy{}
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
		case "ValidatingWebhookConfiguration":
			webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}
			if err := yaml.Unmarshal([]byte(entry), &webhook); err != nil {
				return nil, err
			}
			webhook.TypeMeta = obj
			strategy.validatingWebhookConfigurations = append(strategy.validatingWebhookConfigurations, webhook)
		case "MutatingWebhookConfiguration":
			webhook := &admissionregistrationv1.MutatingWebhookConfiguration{}
			if err := yaml.Unmarshal([]byte(entry), &webhook); err != nil {
				return nil, err
			}
			webhook.TypeMeta = obj
			strategy.mutatingWebhookConfigurations = append(strategy.mutatingWebhookConfigurations, webhook)
		case "APIService":
			apiService := &apiregv1.APIService{}
			if err := yaml.Unmarshal([]byte(entry), &apiService); err != nil {
				return nil, err
			}
			strategy.apiServices = append(strategy.apiServices, apiService)
		case "Secret":
			secret := &corev1.Secret{}
			if err := yaml.Unmarshal([]byte(entry), &secret); err != nil {
				return nil, err
			}
			strategy.certificateSecrets = append(strategy.certificateSecrets, secret)
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
			crdv1 := &extv1.CustomResourceDefinition{}
			switch obj.APIVersion {
			case extv1beta1.SchemeGroupVersion.String():
				crd := &ext.CustomResourceDefinition{}
				crdv1beta1 := &extv1beta1.CustomResourceDefinition{}

				if err := yaml.Unmarshal([]byte(entry), &crdv1beta1); err != nil {
					return nil, err
				}
				err := extv1beta1.Convert_v1beta1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(crdv1beta1, crd, nil)
				if err != nil {
					return nil, err
				}
				err = extv1.Convert_apiextensions_CustomResourceDefinition_To_v1_CustomResourceDefinition(crd, crdv1, nil)
				if err != nil {
					return nil, err
				}
			case extv1.SchemeGroupVersion.String():
				if err := yaml.Unmarshal([]byte(entry), &crdv1); err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("crd ApiVersion %s not supported", obj.APIVersion)
			}
			strategy.crds = append(strategy.crds, crdv1)
		case "SecurityContextConstraints":
			s := &secv1.SecurityContextConstraints{}
			if err := yaml.Unmarshal([]byte(entry), &s); err != nil {
				return nil, err
			}
			strategy.sccs = append(strategy.sccs, s)
		case "ServiceMonitor":
			sm := &promv1.ServiceMonitor{}
			if err := yaml.Unmarshal([]byte(entry), &sm); err != nil {
				return nil, err
			}
			strategy.serviceMonitors = append(strategy.serviceMonitors, sm)
		case "PrometheusRule":
			pr := &promv1.PrometheusRule{}
			if err := yaml.Unmarshal([]byte(entry), &pr); err != nil {
				return nil, err
			}
			strategy.prometheusRules = append(strategy.prometheusRules, pr)
		case "ConfigMap":
			configMap := &corev1.ConfigMap{}
			if err := yaml.Unmarshal([]byte(entry), &configMap); err != nil {
				return nil, err
			}
			strategy.configMaps = append(strategy.configMaps, configMap)
		default:
			return nil, fmt.Errorf("UNKNOWN TYPE %s detected", obj.Kind)

		}
		log.Log.Infof("%s loaded", obj.Kind)
	}
	return strategy, nil
}

func isNamespaceExist(clientset k8coresv1.CoreV1Interface, ns string) (bool, error) {
	_, err := clientset.Namespaces().Get(context.Background(), ns, metav1.GetOptions{})
	if err == nil {
		return true, nil
	}

	if errors.IsNotFound(err) {
		return false, nil
	}

	return false, err
}

func isServiceAccountExist(clientset k8coresv1.CoreV1Interface, ns string, serviceAccount string) (bool, error) {
	_, err := clientset.ServiceAccounts(ns).Get(context.Background(), serviceAccount, metav1.GetOptions{})
	if err == nil {
		return true, nil
	}

	if errors.IsNotFound(err) {
		return false, nil
	}

	return false, err
}
