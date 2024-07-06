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

package kubecli

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"time"

	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"

	clonev1alpha1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/clone/v1alpha1"

	secv1 "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
	cdiclient "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned"
	k8ssnapshotclient "kubevirt.io/client-go/generated/external-snapshotter/clientset/versioned"
	generatedclient "kubevirt.io/client-go/generated/kubevirt/clientset/versioned"
	kvcorev1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/core/v1"
	exportv1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/export/v1beta1"
	instancetypev1beta1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/instancetype/v1beta1"
	migrationsv1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/migrations/v1alpha1"
	poolv1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/pool/v1alpha1"
	snapshotv1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/snapshot/v1beta1"
	networkclient "kubevirt.io/client-go/generated/network-attachment-definition-client/clientset/versioned"
	promclient "kubevirt.io/client-go/generated/prometheus-operator/clientset/versioned"
	"kubevirt.io/client-go/version"
)

type KubevirtClient interface {
	VirtualMachineInstance(namespace string) VirtualMachineInstanceInterface
	VirtualMachineInstanceMigration(namespace string) VirtualMachineInstanceMigrationInterface
	ReplicaSet(namespace string) ReplicaSetInterface
	VirtualMachinePool(namespace string) poolv1.VirtualMachinePoolInterface
	VirtualMachine(namespace string) VirtualMachineInterface
	KubeVirt(namespace string) KubeVirtInterface
	VirtualMachineInstancePreset(namespace string) VirtualMachineInstancePresetInterface
	VirtualMachineSnapshot(namespace string) snapshotv1.VirtualMachineSnapshotInterface
	VirtualMachineSnapshotContent(namespace string) snapshotv1.VirtualMachineSnapshotContentInterface
	VirtualMachineRestore(namespace string) snapshotv1.VirtualMachineRestoreInterface
	VirtualMachineExport(namespace string) exportv1.VirtualMachineExportInterface
	VirtualMachineInstancetype(namespace string) instancetypev1beta1.VirtualMachineInstancetypeInterface
	VirtualMachineClusterInstancetype() instancetypev1beta1.VirtualMachineClusterInstancetypeInterface
	VirtualMachinePreference(namespace string) instancetypev1beta1.VirtualMachinePreferenceInterface
	VirtualMachineClusterPreference() instancetypev1beta1.VirtualMachineClusterPreferenceInterface
	MigrationPolicy() migrationsv1.MigrationPolicyInterface
	ExpandSpec(namespace string) ExpandSpecInterface
	ServerVersion() ServerVersionInterface
	VirtualMachineClone(namespace string) clonev1alpha1.VirtualMachineCloneInterface
	ClusterProfiler() *ClusterProfiler
	GuestfsVersion() *GuestfsVersion
	RestClient() *rest.RESTClient
	GeneratedKubeVirtClient() generatedclient.Interface
	CdiClient() cdiclient.Interface
	NetworkClient() networkclient.Interface
	ExtensionsClient() extclient.Interface
	SecClient() secv1.SecurityV1Interface
	RouteClient() routev1.RouteV1Interface
	DiscoveryClient() discovery.DiscoveryInterface
	PrometheusClient() promclient.Interface
	KubernetesSnapshotClient() k8ssnapshotclient.Interface
	DynamicClient() dynamic.Interface
	MigrationPolicyClient() *migrationsv1.MigrationsV1alpha1Client
	kubernetes.Interface
	Config() *rest.Config
	SetRestTimeout(timeout time.Duration) (KubevirtClient, error)
}

type kubevirt struct {
	master                  string
	kubeconfig              string
	restClient              *rest.RESTClient
	config                  *rest.Config
	generatedKubeVirtClient *generatedclient.Clientset
	cdiClient               *cdiclient.Clientset
	networkClient           *networkclient.Clientset
	extensionsClient        *extclient.Clientset
	secClient               *secv1.SecurityV1Client
	routeClient             *routev1.RouteV1Client
	discoveryClient         *discovery.DiscoveryClient
	prometheusClient        *promclient.Clientset
	snapshotClient          *k8ssnapshotclient.Clientset
	dynamicClient           dynamic.Interface
	migrationsClient        *migrationsv1.MigrationsV1alpha1Client
	cloneClient             *clonev1alpha1.CloneV1alpha1Client
	*kubernetes.Clientset
}

func (k kubevirt) SetRestTimeout(timeout time.Duration) (KubevirtClient, error) {
	config := rest.CopyConfig(k.config)
	config.Timeout = timeout
	k.config = config

	restClient, err := rest.RESTClientFor(k.config)
	if err != nil {
		return &k, err
	}
	k.restClient = restClient

	return &k, nil
}

func (k kubevirt) Config() *rest.Config {
	return k.config
}

func (k kubevirt) CdiClient() cdiclient.Interface {
	return k.cdiClient
}

func (k kubevirt) NetworkClient() networkclient.Interface {
	return k.networkClient
}

func (k kubevirt) ExtensionsClient() extclient.Interface {
	return k.extensionsClient
}

func (k kubevirt) SecClient() secv1.SecurityV1Interface {
	return k.secClient
}

func (k kubevirt) RouteClient() routev1.RouteV1Interface {
	return k.routeClient
}

func (k kubevirt) DiscoveryClient() discovery.DiscoveryInterface {
	return k.discoveryClient
}

func (k kubevirt) PrometheusClient() promclient.Interface {
	return k.prometheusClient
}

func (k kubevirt) RestClient() *rest.RESTClient {
	return k.restClient
}

func (k kubevirt) GeneratedKubeVirtClient() generatedclient.Interface {
	return k.generatedKubeVirtClient
}

func (k kubevirt) VirtualMachinePool(namespace string) poolv1.VirtualMachinePoolInterface {
	return k.generatedKubeVirtClient.PoolV1alpha1().VirtualMachinePools(namespace)
}

func (k kubevirt) VirtualMachineSnapshot(namespace string) snapshotv1.VirtualMachineSnapshotInterface {
	return k.generatedKubeVirtClient.SnapshotV1beta1().VirtualMachineSnapshots(namespace)
}

func (k kubevirt) VirtualMachineSnapshotContent(namespace string) snapshotv1.VirtualMachineSnapshotContentInterface {
	return k.generatedKubeVirtClient.SnapshotV1beta1().VirtualMachineSnapshotContents(namespace)
}

func (k kubevirt) VirtualMachineRestore(namespace string) snapshotv1.VirtualMachineRestoreInterface {
	return k.generatedKubeVirtClient.SnapshotV1beta1().VirtualMachineRestores(namespace)
}

func (k kubevirt) VirtualMachineExport(namespace string) exportv1.VirtualMachineExportInterface {
	return k.generatedKubeVirtClient.ExportV1beta1().VirtualMachineExports(namespace)
}

func (k kubevirt) VirtualMachineInstancetype(namespace string) instancetypev1beta1.VirtualMachineInstancetypeInterface {
	return k.generatedKubeVirtClient.InstancetypeV1beta1().VirtualMachineInstancetypes(namespace)
}

func (k kubevirt) VirtualMachineClusterInstancetype() instancetypev1beta1.VirtualMachineClusterInstancetypeInterface {
	return k.generatedKubeVirtClient.InstancetypeV1beta1().VirtualMachineClusterInstancetypes()
}

func (k kubevirt) VirtualMachinePreference(namespace string) instancetypev1beta1.VirtualMachinePreferenceInterface {
	return k.generatedKubeVirtClient.InstancetypeV1beta1().VirtualMachinePreferences(namespace)
}

func (k kubevirt) VirtualMachineClusterPreference() instancetypev1beta1.VirtualMachineClusterPreferenceInterface {
	return k.generatedKubeVirtClient.InstancetypeV1beta1().VirtualMachineClusterPreferences()
}

func (k kubevirt) KubernetesSnapshotClient() k8ssnapshotclient.Interface {
	return k.snapshotClient
}

func (k kubevirt) DynamicClient() dynamic.Interface {
	return k.dynamicClient
}

func (k kubevirt) MigrationPolicy() migrationsv1.MigrationPolicyInterface {
	return k.generatedKubeVirtClient.MigrationsV1alpha1().MigrationPolicies()
}

func (k kubevirt) MigrationPolicyClient() *migrationsv1.MigrationsV1alpha1Client {
	return k.migrationsClient
}

func (k kubevirt) VirtualMachineClone(namespace string) clonev1alpha1.VirtualMachineCloneInterface {
	return k.generatedKubeVirtClient.CloneV1alpha1().VirtualMachineClones(namespace)
}

func (k kubevirt) VirtualMachineCloneClient() *clonev1alpha1.CloneV1alpha1Client {
	return k.cloneClient // TODO ihol3 delete function? who's using it?
}

type VirtualMachineInstanceInterface interface {
	kvcorev1.VirtualMachineInstanceInterface
}

type ReplicaSetInterface interface {
	kvcorev1.VirtualMachineInstanceReplicaSetInterface
}

type VirtualMachineInstancePresetInterface interface {
	kvcorev1.VirtualMachineInstancePresetInterface
}

// VirtualMachineInterface provides convenience methods to work with
// virtual machines inside the cluster
type VirtualMachineInterface interface {
	kvcorev1.VirtualMachineInterface
}

type VirtualMachineInstanceMigrationInterface interface {
	kvcorev1.VirtualMachineInstanceMigrationInterface
}

type KubeVirtInterface interface {
	kvcorev1.KubeVirtInterface
}

type ServerVersionInterface interface {
	Get() (*version.Info, error)
}

type ExpandSpecInterface interface {
	ForVirtualMachine(vm *v1.VirtualMachine) (*v1.VirtualMachine, error)
}
