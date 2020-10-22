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
	"os"
	"path"
	"sort"

	"github.com/ghodss/yaml"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/components"
	"github.com/kubevirt/hyperconverged-cluster-operator/tools/util"

	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	imsConversionImage = flag.String("ims-conversion-image-name", "", "IMS conversion image")
	imsVMWareImage     = flag.String("ims-vmware-image-name", "", "IMS VMWare image")
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
		panic(err)
	}
}

func main() {
	flag.Parse()

	// open files for writing
	operatorYaml, err := os.Create(path.Join(*deployDir, "operator.yaml"))
	check(err)
	saYaml, err := os.Create(path.Join(*deployDir, "service_account.yaml"))
	check(err)
	crbYaml, err := os.Create(path.Join(*deployDir, "cluster_role_binding.yaml"))
	check(err)
	crYaml, err := os.Create(path.Join(*deployDir, "cluster_role.yaml"))
	check(err)
	operatorCrd, err := os.Create(path.Join(*deployDir, "crds/hco.crd.yaml"))
	check(err)
	v2vCrd, err := os.Create(path.Join(*deployDir, "crds/v2vvmware.crd.yaml"))
	check(err)
	v2voVirtCrd, err := os.Create(path.Join(*deployDir, "crds/v2vovirt.crd.yaml"))
	check(err)
	operatorCr, err := os.Create(path.Join(*deployDir, "hco.cr.yaml"))
	check(err)
	defer operatorYaml.Close()
	defer saYaml.Close()
	defer crbYaml.Close()
	defer crYaml.Close()
	defer operatorCrd.Close()
	defer operatorCr.Close()
	defer v2vCrd.Close()
	defer v2voVirtCrd.Close()

	// the CSVs we expect to handle
	csvs := []string{
		*cnaCsv,
		*virtCsv,
		*sspCsv,
		*cdiCsv,
		*nmoCsv,
		*hppCsv,
		*vmImportCsv,
	}

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
		components.GetDeploymentOperator(
			*operatorNamespace,
			*operatorImage,
			"IfNotPresent",
			*imsConversionImage,
			*imsVMWareImage,
			*smbios,
			*machinetype,
			*hcoKvIoVersion,
			*kubevirtVersion,
			*cdiVersion,
			*cnaoVersion,
			*sspVersion,
			*nmoVersion,
			*hppoVersion,
			*vmImportVersion,
			[]corev1.EnvVar{},
		),
		components.GetDeploymentWebhook(
			*operatorNamespace,
			*operatorImage,
			"IfNotPresent",
			[]corev1.EnvVar{},
		),
	}
	serviceAccounts := map[string]v1.ServiceAccount{
		"hyperconverged-cluster-operator": components.GetServiceAccount(*operatorNamespace),
	}
	permissions := []rbacv1.Role{}
	roleBindings := []rbacv1.RoleBinding{}
	clusterPermissions := []rbacv1.ClusterRole{
		components.GetClusterRole(),
	}
	clusterRoleBindings := []rbacv1.ClusterRoleBinding{
		components.GetClusterRoleBinding(*operatorNamespace),
	}

	for _, csvStr := range csvs {
		if csvStr != "" {
			csvBytes := []byte(csvStr)

			csvStruct := &csvv1alpha1.ClusterServiceVersion{}

			err := yaml.Unmarshal(csvBytes, csvStruct)
			if err != nil {
				panic(err)
			}

			strategySpec := csvStruct.Spec.InstallStrategy.StrategySpec

			// CSVs only contain the deployment spec, we must wrap
			// the spec with the Type and Object Meta to make it a valid
			// manifest
			for _, deploymentSpec := range strategySpec.DeploymentSpecs {
				deployments = append(deployments, appsv1.Deployment{
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
				})
			}

			// Every permission we encounter in a CSV means we need to (potentially)
			// add a service account to our map, add a Role to our slice of permissions
			// (much like we did with deployments we need to wrap the rules with the
			// Type and Object Meta to make a valid Role), and add a RoleBinding to our
			// slice.
			for _, permission := range strategySpec.Permissions {
				serviceAccounts[permission.ServiceAccountName] = v1.ServiceAccount{
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
				permissions = append(permissions, rbacv1.Role{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "rbac.authorization.k8s.io/v1",
						Kind:       "Role",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: permission.ServiceAccountName,
						Labels: map[string]string{
							"name": permission.ServiceAccountName,
						},
					},
					Rules: permission.Rules,
				})
				roleBindings = append(roleBindings, rbacv1.RoleBinding{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "rbac.authorization.k8s.io/v1",
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
						rbacv1.Subject{
							Kind:      "ServiceAccount",
							Name:      permission.ServiceAccountName,
							Namespace: *operatorNamespace,
						},
					},
				})
			}

			// Same as permissions except ClusterRole instead of Role and
			// ClusterRoleBinding instead of RoleBinding.
			for _, clusterPermission := range strategySpec.ClusterPermissions {
				serviceAccounts[clusterPermission.ServiceAccountName] = v1.ServiceAccount{
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
				clusterPermissions = append(clusterPermissions, rbacv1.ClusterRole{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "rbac.authorization.k8s.io/v1",
						Kind:       "ClusterRole",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: clusterPermission.ServiceAccountName,
						Labels: map[string]string{
							"name": clusterPermission.ServiceAccountName,
						},
					},
					Rules: clusterPermission.Rules,
				})
				clusterRoleBindings = append(clusterRoleBindings, rbacv1.ClusterRoleBinding{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "rbac.authorization.k8s.io/v1",
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
						rbacv1.Subject{
							Kind:      "ServiceAccount",
							Name:      clusterPermission.ServiceAccountName,
							Namespace: *operatorNamespace,
						},
					},
				})
			}
		}
	}

	// Write out CRDs and CR
	util.MarshallObject(components.GetOperatorCR(), operatorCr)
	util.MarshallObject(components.GetOperatorCRD(*apiSources), operatorCrd)
	util.MarshallObject(components.GetV2VCRD(), v2vCrd)
	util.MarshallObject(components.GetV2VOvirtProviderCRD(), v2voVirtCrd)

	// Write out deployments
	for _, deployment := range deployments {
		util.MarshallObject(deployment, operatorYaml)
	}

	// Write out rbac
	// since maps are not ordered we must enforce one before writing
	var keys []string
	for saName := range serviceAccounts {
		keys = append(keys, saName)
	}
	sort.Strings(keys)
	for _, k := range keys {
		util.MarshallObject(serviceAccounts[k], saYaml)
	}
	for _, permission := range permissions {
		util.MarshallObject(permission, crYaml)
	}
	for _, roleBinding := range roleBindings {
		util.MarshallObject(roleBinding, crbYaml)
	}
	for _, clusterPermission := range clusterPermissions {
		util.MarshallObject(clusterPermission, crYaml)
	}
	for _, clusterRoleBinding := range clusterRoleBindings {
		util.MarshallObject(clusterRoleBinding, crbYaml)
	}
}
