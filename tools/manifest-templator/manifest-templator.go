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

package main

import (
	"flag"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	"log"
	"os"
	"path"
	"sort"
	"strconv"

	"github.com/ghodss/yaml"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/components"
	"github.com/kubevirt/hyperconverged-cluster-operator/tools/util"

	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	rbacAPI string
)

// flags for the command line arguments we accept
var (
	cwd, _             = os.Getwd()
	deployDir          = flag.String("deploy-dir", "deploy", "Directory where manifests should be written")
	cnaCsv             = flag.String("cna-csv", "", "Cluster Network Addons CSV string")
	virtCsv            = flag.String("virt-csv", "", "KubeVirt CSV string")
	sspCsv             = flag.String("ssp-csv", "", "Scheduling Scale Performance CSV string")
	cdiCsv             = flag.String("cdi-csv", "", "Containerized Data Importer CSV String")
	nmoCsv             = flag.String("nmo-csv", "", "Node Maintenance Operator CSV String")
	hppCsv             = flag.String("hpp-csv", "", "HostPath Provisioner Operator CSV String")
	vmImportCsv        = flag.String("vmimport-csv", "", "Virtual Machine Import Operator CSV String")
	operatorNamespace  = flag.String("operator-namespace", "kubevirt-hyperconverged", "Name of the Operator")
	operatorImage      = flag.String("operator-image", "", "HyperConverged Cluster Operator image")
	webhookImage       = flag.String("webhook-image", "", "HyperConverged Cluster Webhook image")
	imsConversionImage = flag.String("ims-conversion-image-name", "", "IMS conversion image")
	imsVMWareImage     = flag.String("ims-vmware-image-name", "", "IMS VMWare image")
	kvVirtIOWinImage   = flag.String("kv-virtiowin-image-name", "", "KubeVirt VirtIO Win image")
	smbios             = flag.String("smbios", "", "Custom SMBIOS string for KubeVirt ConfigMap")
	machinetype        = flag.String("machinetype", "", "Custom MACHINETYPE string for KubeVirt ConfigMap")
	hcoKvIoVersion     = flag.String("hco-kv-io-version", "", "KubeVirt version")
	kubevirtVersion    = flag.String("kubevirt-version", "", "Kubevirt operator version")
	cdiVersion         = flag.String("cdi-version", "", "CDI operator version")
	cnaoVersion        = flag.String("cnao-version", "", "CNA operator version")
	sspVersion         = flag.String("ssp-version", "", "SSP operator version")
	nmoVersion         = flag.String("nmo-version", "", "NM operator version")
	hppoVersion        = flag.String("hppo-version", "", "HPP operator version")
	vmImportVersion    = flag.String("vm-import-version", "", "VM-Import operator version")
	apiSources         = flag.String("api-sources", cwd+"/...", "Project sources")
)

// check handles errors
func check(err error) {
	if err != nil {
		log.Println("error: ", err)
		panic(err)
	}
}
func init() {
	rbacAPI = rbacv1.SchemeGroupVersion.String()
	processCommandlineParams()
}

func processCommandlineParams() {
	flag.Parse()

	if webhookImage == nil || *webhookImage == "" {
		*webhookImage = *operatorImage
	}
}

func main() {
	// the CSVs we expect to handle
	componentsWithCSVs := getCsvWithComponent()

	operatorParams := getOperatorParameters()

	// these represent the bare necessities for the HCO manifests, that is,
	// enough to deploy the HCO itself.
	// 1 deployment
	// 1 service account
	// 1 cluster role
	// 1 cluster role binding
	// as we handle each CSV we will add to our slices of deployments,
	// permissions, role bindings, cluster permissions, and cluster role bindings.
	// service accounts are represented as a map to prevent us from generating the
	// same service account multiple times.
	deployments := []appsv1.Deployment{
		components.GetDeploymentOperator(operatorParams),
		components.GetDeploymentWebhook(
			*operatorNamespace,
			*webhookImage,
			"IfNotPresent",
			*hcoKvIoVersion,
			[]corev1.EnvVar{},
		),
	}
	// hco-operator and hco-webhook
	for i := range deployments {
		overwriteDeploymentLabels(&deployments[i], hcoutil.AppComponentDeployment)
	}

	services := []v1.Service{
		components.GetServiceWebhook(*operatorNamespace),
	}

	serviceAccounts := map[string]v1.ServiceAccount{
		"hyperconverged-cluster-operator": components.GetServiceAccount(*operatorNamespace),
	}
	permissions := make([]rbacv1.Role, 0)
	roleBindings := make([]rbacv1.RoleBinding, 0)
	clusterPermissions := []rbacv1.ClusterRole{
		components.GetClusterRole(),
	}
	clusterRoleBindings := []rbacv1.ClusterRoleBinding{
		components.GetClusterRoleBinding(*operatorNamespace),
	}

	for _, csvStr := range componentsWithCSVs {
		if csvStr.Csv == "" {
			continue
		}
		csvBytes := []byte(csvStr.Csv)

		csvStruct := &csvv1alpha1.ClusterServiceVersion{}

		check(yaml.Unmarshal(csvBytes, csvStruct))

		services = getServices(csvStruct, services)

		strategySpec := csvStruct.Spec.InstallStrategy.StrategySpec

		// CSVs only contain the deployment spec, we must wrap
		// the spec with the Type and Object Meta to make it a valid
		// manifest
		for _, deploymentSpec := range strategySpec.DeploymentSpecs {
			deploy := getBasicDeployment(deploymentSpec)
			injectWebhookMounts(csvStruct.Spec.WebhookDefinitions, &deploy)
			overwriteDeploymentLabels(&deploy, csvStr.Component)
			deployments = append(deployments, deploy)
		}

		// Every permission we encounter in a CSV means we need to (potentially)
		// add a service account to our map, add a Role to our slice of permissions
		// (much like we did with deployments we need to wrap the rules with the
		// Type and Object Meta to make a valid Role), and add a RoleBinding to our
		// slice.
		for _, permission := range strategySpec.Permissions {
			serviceAccounts[permission.ServiceAccountName] = getBasicServiceAccount(permission)
			permissions = append(permissions, getRole(permission))
			roleBindings = append(roleBindings, getRoleBinding(permission))
		}

		// Same as permissions except ClusterRole instead of Role and
		// ClusterRoleBinding instead of RoleBinding.
		for _, clusterPermission := range strategySpec.ClusterPermissions {
			serviceAccounts[clusterPermission.ServiceAccountName] = createServiceAccount(clusterPermission)
			clusterPermissions = append(clusterPermissions, createClusterRole(clusterPermission))
			clusterRoleBindings = append(clusterRoleBindings, createClusterRoleBinding(clusterPermission))
		}
	}

	// Write out CRDs and CR
	writeOperatorCR()
	writeOperatorCRD()
	writeV2VCR()
	writeV2VCRD()

	// Write out deployments and services
	writeOperatorDeploymentsAndServices(deployments, services)

	// Write out rbac
	writeServiceAccounts(serviceAccounts)
	writeClusterRoleBindings(roleBindings, clusterRoleBindings)
	writeClusterRoles(permissions, clusterPermissions)
}

func getServices(csvStruct *csvv1alpha1.ClusterServiceVersion, services []v1.Service) []v1.Service {
	for _, webhook := range csvStruct.Spec.WebhookDefinitions {
		services = append(services, createService(webhook, csvStruct))
	}
	return services
}

func createClusterRoleBinding(clusterPermission csvv1alpha1.StrategyDeploymentPermissions) rbacv1.ClusterRoleBinding {
	return rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacAPI,
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterPermission.ServiceAccountName,
			Labels: map[string]string{
				"name": clusterPermission.ServiceAccountName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterPermission.ServiceAccountName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      clusterPermission.ServiceAccountName,
				Namespace: *operatorNamespace,
			},
		},
	}
}

func createClusterRole(clusterPermission csvv1alpha1.StrategyDeploymentPermissions) rbacv1.ClusterRole {
	return rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacAPI,
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterPermission.ServiceAccountName,
			Labels: map[string]string{
				"name": clusterPermission.ServiceAccountName,
			},
		},
		Rules: clusterPermission.Rules,
	}
}

func createServiceAccount(clusterPermission csvv1alpha1.StrategyDeploymentPermissions) v1.ServiceAccount {
	return v1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterPermission.ServiceAccountName,
			Namespace: *operatorNamespace,
			Labels: map[string]string{
				"name": clusterPermission.ServiceAccountName,
			},
		},
	}
}

func getRoleBinding(permission csvv1alpha1.StrategyDeploymentPermissions) rbacv1.RoleBinding {
	return rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacAPI,
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      permission.ServiceAccountName,
			Namespace: *operatorNamespace,
			Labels: map[string]string{
				"name": permission.ServiceAccountName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     permission.ServiceAccountName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      permission.ServiceAccountName,
				Namespace: *operatorNamespace,
			},
		},
	}
}

func getRole(permission csvv1alpha1.StrategyDeploymentPermissions) rbacv1.Role {
	return rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacAPI,
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: permission.ServiceAccountName,
			Labels: map[string]string{
				"name": permission.ServiceAccountName,
			},
		},
		Rules: permission.Rules,
	}
}

func getBasicServiceAccount(permission csvv1alpha1.StrategyDeploymentPermissions) v1.ServiceAccount {
	return v1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      permission.ServiceAccountName,
			Namespace: *operatorNamespace,
			Labels: map[string]string{
				"name": permission.ServiceAccountName,
			},
		},
	}
}

func getBasicDeployment(deploymentSpec csvv1alpha1.StrategyDeploymentSpec) appsv1.Deployment {
	return appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: deploymentSpec.Name,
			Labels: map[string]string{
				"name": deploymentSpec.Name,
			},
		},
		Spec: deploymentSpec.Spec,
	}
}

func createService(webhook csvv1alpha1.WebhookDescription, csvStruct *csvv1alpha1.ClusterServiceVersion) v1.Service {
	return v1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: webhook.DeploymentName + "-service",
			Labels: map[string]string{
				"name": webhook.DeploymentName,
			},
		},
		Spec: v1.ServiceSpec{
			Selector: getSelectorOfWebhookDeployment(webhook.DeploymentName, csvStruct.Spec.InstallStrategy.StrategySpec.DeploymentSpecs),
			Ports: []v1.ServicePort{
				{
					Name:       strconv.Itoa(int(webhook.ContainerPort)),
					Port:       webhook.ContainerPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(int(webhook.ContainerPort)),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}

func getCsvWithComponent() []util.CsvWithComponent {
	componentsWithCsvs := []util.CsvWithComponent{
		{
			Csv:       *cnaCsv,
			Component: hcoutil.AppComponentNetwork,
		},
		{
			Csv:       *virtCsv,
			Component: hcoutil.AppComponentCompute,
		},
		{
			Csv:       *sspCsv,
			Component: hcoutil.AppComponentSchedule,
		},
		{
			Csv:       *cdiCsv,
			Component: hcoutil.AppComponentStorage,
		},
		{
			Csv:       *nmoCsv,
			Component: hcoutil.AppComponentNetwork,
		},
		{
			Csv:       *hppCsv,
			Component: hcoutil.AppComponentStorage,
		},
		{
			Csv:       *vmImportCsv,
			Component: hcoutil.AppComponentImport,
		},
	}
	return componentsWithCsvs
}

func writeV2VCRD() {
	v2voVirtCrd, err := os.Create(path.Join(*deployDir, "crds/v2vovirt.crd.yaml"))
	check(err)
	defer v2voVirtCrd.Close()
	check(util.MarshallObject(components.GetV2VOvirtProviderCRD(), v2voVirtCrd))
}

func writeV2VCR() {
	v2vCrd, err := os.Create(path.Join(*deployDir, "crds/v2vvmware.crd.yaml"))
	check(err)
	defer v2vCrd.Close()

	check(util.MarshallObject(components.GetV2VCRD(), v2vCrd))
}

func getOperatorParameters() *components.DeploymentOperatorParams {
	params := &components.DeploymentOperatorParams{
		Namespace:           *operatorNamespace,
		Image:               *operatorImage,
		ImagePullPolicy:     "IfNotPresent",
		ConversionContainer: *imsConversionImage,
		VmwareContainer:     *imsVMWareImage,
		VirtIOWinContainer:  *kvVirtIOWinImage,
		Smbios:              *smbios,
		Machinetype:         *machinetype,
		HcoKvIoVersion:      *hcoKvIoVersion,
		KubevirtVersion:     *kubevirtVersion,
		CdiVersion:          *cdiVersion,
		CnaoVersion:         *cnaoVersion,
		SspVersion:          *sspVersion,
		NmoVersion:          *nmoVersion,
		HppoVersion:         *hppoVersion,
		VMImportVersion:     *vmImportVersion,
		Env:                 []corev1.EnvVar{},
	}
	return params
}

func writeOperatorCR() {
	operatorCr, err := os.Create(path.Join(*deployDir, "hco.cr.yaml"))
	check(err)
	defer operatorCr.Close()
	check(util.MarshallObject(components.GetOperatorCR(), operatorCr))
}

func writeOperatorCRD() {
	operatorCrd, err := os.Create(path.Join(*deployDir, "crds/hco.crd.yaml"))
	check(err)
	defer operatorCrd.Close()
	check(util.MarshallObject(components.GetOperatorCRD(*apiSources), operatorCrd))
}

func writeOperatorDeploymentsAndServices(deployments []appsv1.Deployment, services []v1.Service) {
	operatorYaml, err := os.Create(path.Join(*deployDir, "operator.yaml"))
	check(err)
	defer operatorYaml.Close()
	for _, deployment := range deployments {
		check(util.MarshallObject(deployment, operatorYaml))
	}

	// Write out services
	for _, service := range services {
		check(util.MarshallObject(service, operatorYaml))
	}
}

func writeServiceAccounts(serviceAccounts map[string]v1.ServiceAccount) {
	var keys []string
	for saName := range serviceAccounts {
		keys = append(keys, saName)
	}
	// since maps are not ordered we must enforce one before writing
	sort.Strings(keys)

	saYaml, err := os.Create(path.Join(*deployDir, "service_account.yaml"))
	check(err)
	defer saYaml.Close()

	for _, k := range keys {
		check(util.MarshallObject(serviceAccounts[k], saYaml))
	}
}

func writeClusterRoleBindings(roleBindings []rbacv1.RoleBinding, clusterRoleBindings []rbacv1.ClusterRoleBinding) {
	crbYaml, err := os.Create(path.Join(*deployDir, "cluster_role_binding.yaml"))
	check(err)
	defer crbYaml.Close()

	for _, roleBinding := range roleBindings {
		check(util.MarshallObject(roleBinding, crbYaml))
	}

	for _, clusterRoleBinding := range clusterRoleBindings {
		check(util.MarshallObject(clusterRoleBinding, crbYaml))
	}
}

func writeClusterRoles(permissions []rbacv1.Role, clusterPermissions []rbacv1.ClusterRole) {
	crYaml, err := os.Create(path.Join(*deployDir, "cluster_role.yaml"))
	check(err)
	defer crYaml.Close()

	for _, permission := range permissions {
		check(util.MarshallObject(permission, crYaml))
	}
	for _, clusterPermission := range clusterPermissions {
		check(util.MarshallObject(clusterPermission, crYaml))
	}
}

func getSelectorOfWebhookDeployment(deployment string, specs []csvv1alpha1.StrategyDeploymentSpec) map[string]string {
	for _, ds := range specs {
		if ds.Name == deployment {
			return ds.Spec.Selector.MatchLabels
		}
	}

	panic("no deployment spec for webhook:" + deployment)
}

func injectWebhookMounts(webhookDefs []csvv1alpha1.WebhookDescription, deploy *appsv1.Deployment) {
	for _, webhook := range webhookDefs {
		if webhook.DeploymentName == deploy.Name {
			components.InjectVolumesForWebHookCerts(deploy)
		}
	}
}

func overwriteDeploymentLabels(deploy *appsv1.Deployment, component hcoutil.AppComponent) {
	if deploy.Labels == nil {
		deploy.Labels = make(map[string]string)
	}
	if deploy.Spec.Template.Labels == nil {
		deploy.Spec.Template.Labels = make(map[string]string)
	}
	overwriteWithStandardLabels(deploy.Spec.Template.Labels, *hcoKvIoVersion, component)
	overwriteWithStandardLabels(deploy.Labels, *hcoKvIoVersion, component)
}

func overwriteWithStandardLabels(labels map[string]string, version string, component hcoutil.AppComponent) {
	// managed-by label is not set here since we don't know who is going to deploy the generated yaml files
	labels[hcoutil.AppLabelVersion] = version
	labels[hcoutil.AppLabelPartOf] = hcoutil.HyperConvergedCluster
	labels[hcoutil.AppLabelComponent] = string(component)
}
