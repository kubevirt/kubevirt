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
	"io"
	"net"
	"time"

	migrationsv1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/migrations/v1alpha1"

	secv1 "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	autov1 "k8s.io/api/autoscaling/v1"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
	cdiclient "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned"
	k8ssnapshotclient "kubevirt.io/client-go/generated/external-snapshotter/clientset/versioned"
	generatedclient "kubevirt.io/client-go/generated/kubevirt/clientset/versioned"
	flavorv1alpha1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/flavor/v1alpha1"
	poolv1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/pool/v1alpha1"
	vmsnapshotv1alpha1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/snapshot/v1alpha1"
	networkclient "kubevirt.io/client-go/generated/network-attachment-definition-client/clientset/versioned"
	promclient "kubevirt.io/client-go/generated/prometheus-operator/clientset/versioned"
)

type KubevirtClient interface {
	VirtualMachineInstance(namespace string) VirtualMachineInstanceInterface
	VirtualMachineInstanceMigration(namespace string) VirtualMachineInstanceMigrationInterface
	ReplicaSet(namespace string) ReplicaSetInterface
	VirtualMachinePool(namespace string) poolv1.VirtualMachinePoolInterface
	VirtualMachine(namespace string) VirtualMachineInterface
	KubeVirt(namespace string) KubeVirtInterface
	VirtualMachineInstancePreset(namespace string) VirtualMachineInstancePresetInterface
	VirtualMachineSnapshot(namespace string) vmsnapshotv1alpha1.VirtualMachineSnapshotInterface
	VirtualMachineSnapshotContent(namespace string) vmsnapshotv1alpha1.VirtualMachineSnapshotContentInterface
	VirtualMachineRestore(namespace string) vmsnapshotv1alpha1.VirtualMachineRestoreInterface
	VirtualMachineFlavor(namespace string) flavorv1alpha1.VirtualMachineFlavorInterface
	VirtualMachineClusterFlavor() flavorv1alpha1.VirtualMachineClusterFlavorInterface
	MigrationPolicy() migrationsv1.MigrationPolicyInterface
	ServerVersion() *ServerVersion
	ClusterProfiler() *ClusterProfiler
	GuestfsVersion() *GuestfsVersion
	RestClient() *rest.RESTClient
	GeneratedKubeVirtClient() generatedclient.Interface
	CdiClient() cdiclient.Interface
	NetworkClient() networkclient.Interface
	ExtensionsClient() extclient.Interface
	SecClient() secv1.SecurityV1Interface
	DiscoveryClient() discovery.DiscoveryInterface
	PrometheusClient() promclient.Interface
	KubernetesSnapshotClient() k8ssnapshotclient.Interface
	DynamicClient() dynamic.Interface
	MigrationPolicyClient() *migrationsv1.MigrationsV1alpha1Client
	kubernetes.Interface
	Config() *rest.Config
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
	discoveryClient         *discovery.DiscoveryClient
	prometheusClient        *promclient.Clientset
	snapshotClient          *k8ssnapshotclient.Clientset
	dynamicClient           dynamic.Interface
	migrationsClient        *migrationsv1.MigrationsV1alpha1Client
	*kubernetes.Clientset
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

func (k kubevirt) VirtualMachineSnapshot(namespace string) vmsnapshotv1alpha1.VirtualMachineSnapshotInterface {
	return k.generatedKubeVirtClient.SnapshotV1alpha1().VirtualMachineSnapshots(namespace)
}

func (k kubevirt) VirtualMachineSnapshotContent(namespace string) vmsnapshotv1alpha1.VirtualMachineSnapshotContentInterface {
	return k.generatedKubeVirtClient.SnapshotV1alpha1().VirtualMachineSnapshotContents(namespace)
}

func (k kubevirt) VirtualMachineRestore(namespace string) vmsnapshotv1alpha1.VirtualMachineRestoreInterface {
	return k.generatedKubeVirtClient.SnapshotV1alpha1().VirtualMachineRestores(namespace)
}

func (k kubevirt) VirtualMachineFlavor(namespace string) flavorv1alpha1.VirtualMachineFlavorInterface {
	return k.generatedKubeVirtClient.FlavorV1alpha1().VirtualMachineFlavors(namespace)
}

func (k kubevirt) VirtualMachineClusterFlavor() flavorv1alpha1.VirtualMachineClusterFlavorInterface {
	return k.generatedKubeVirtClient.FlavorV1alpha1().VirtualMachineClusterFlavors()
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

type StreamOptions struct {
	In  io.Reader
	Out io.Writer
}

type StreamInterface interface {
	Stream(options StreamOptions) error
	AsConn() net.Conn
}

type VirtualMachineInstanceInterface interface {
	Get(name string, options *k8smetav1.GetOptions) (*v1.VirtualMachineInstance, error)
	List(opts *k8smetav1.ListOptions) (*v1.VirtualMachineInstanceList, error)
	Create(instance *v1.VirtualMachineInstance) (*v1.VirtualMachineInstance, error)
	Update(*v1.VirtualMachineInstance) (*v1.VirtualMachineInstance, error)
	Delete(name string, options *k8smetav1.DeleteOptions) error
	Patch(name string, pt types.PatchType, data []byte, patchOptions *k8smetav1.PatchOptions, subresources ...string) (result *v1.VirtualMachineInstance, err error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	SerialConsole(name string, options *SerialConsoleOptions) (StreamInterface, error)
	USBRedir(vmiName string) (StreamInterface, error)
	VNC(name string) (StreamInterface, error)
	PortForward(name string, port int, protocol string) (StreamInterface, error)
	Pause(name string, pauseOptions *v1.PauseOptions) error
	Unpause(name string, unpauseOptions *v1.UnpauseOptions) error
	Freeze(name string, unfreezeTimeout time.Duration) error
	Unfreeze(name string) error
	SoftReboot(name string) error
	GuestOsInfo(name string) (v1.VirtualMachineInstanceGuestAgentInfo, error)
	UserList(name string) (v1.VirtualMachineInstanceGuestOSUserList, error)
	FilesystemList(name string) (v1.VirtualMachineInstanceFileSystemList, error)
	AddVolume(name string, addVolumeOptions *v1.AddVolumeOptions) error
	RemoveVolume(name string, removeVolumeOptions *v1.RemoveVolumeOptions) error
}

type ReplicaSetInterface interface {
	Get(name string, options k8smetav1.GetOptions) (*v1.VirtualMachineInstanceReplicaSet, error)
	List(opts k8smetav1.ListOptions) (*v1.VirtualMachineInstanceReplicaSetList, error)
	Create(*v1.VirtualMachineInstanceReplicaSet) (*v1.VirtualMachineInstanceReplicaSet, error)
	Update(*v1.VirtualMachineInstanceReplicaSet) (*v1.VirtualMachineInstanceReplicaSet, error)
	Delete(name string, options *k8smetav1.DeleteOptions) error
	GetScale(replicaSetName string, options k8smetav1.GetOptions) (*autov1.Scale, error)
	UpdateScale(replicaSetName string, scale *autov1.Scale) (*autov1.Scale, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachineInstanceReplicaSet, err error)
	UpdateStatus(*v1.VirtualMachineInstanceReplicaSet) (*v1.VirtualMachineInstanceReplicaSet, error)
	PatchStatus(name string, pt types.PatchType, data []byte) (result *v1.VirtualMachineInstanceReplicaSet, err error)
}

type VirtualMachineInstancePresetInterface interface {
	Get(name string, options k8smetav1.GetOptions) (*v1.VirtualMachineInstancePreset, error)
	List(opts k8smetav1.ListOptions) (*v1.VirtualMachineInstancePresetList, error)
	Create(*v1.VirtualMachineInstancePreset) (*v1.VirtualMachineInstancePreset, error)
	Update(*v1.VirtualMachineInstancePreset) (*v1.VirtualMachineInstancePreset, error)
	Delete(name string, options *k8smetav1.DeleteOptions) error
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachineInstancePreset, err error)
}

// VirtualMachineInterface provides convenience methods to work with
// virtual machines inside the cluster
type VirtualMachineInterface interface {
	Get(name string, options *k8smetav1.GetOptions) (*v1.VirtualMachine, error)
	List(opts *k8smetav1.ListOptions) (*v1.VirtualMachineList, error)
	Create(*v1.VirtualMachine) (*v1.VirtualMachine, error)
	Update(*v1.VirtualMachine) (*v1.VirtualMachine, error)
	Delete(name string, options *k8smetav1.DeleteOptions) error
	Patch(name string, pt types.PatchType, data []byte, patchOptions *metav1.PatchOptions, subresources ...string) (result *v1.VirtualMachine, err error)
	UpdateStatus(*v1.VirtualMachine) (*v1.VirtualMachine, error)
	PatchStatus(name string, pt types.PatchType, data []byte, patchOptions *metav1.PatchOptions) (result *v1.VirtualMachine, err error)
	Restart(name string, restartOptions *v1.RestartOptions) error
	ForceRestart(name string, restartOptions *v1.RestartOptions) error
	Start(name string, startOptions *v1.StartOptions) error
	Stop(name string, stopOptions *v1.StopOptions) error
	ForceStop(name string, stopOptions *v1.StopOptions) error
	Migrate(name string, migrateOptions *v1.MigrateOptions) error
	AddVolume(name string, addVolumeOptions *v1.AddVolumeOptions) error
	RemoveVolume(name string, removeVolumeOptions *v1.RemoveVolumeOptions) error
	PortForward(name string, port int, protocol string) (StreamInterface, error)
}

type VirtualMachineInstanceMigrationInterface interface {
	Get(name string, options *k8smetav1.GetOptions) (*v1.VirtualMachineInstanceMigration, error)
	List(opts *k8smetav1.ListOptions) (*v1.VirtualMachineInstanceMigrationList, error)
	Create(migration *v1.VirtualMachineInstanceMigration, options *k8smetav1.CreateOptions) (*v1.VirtualMachineInstanceMigration, error)
	Update(*v1.VirtualMachineInstanceMigration) (*v1.VirtualMachineInstanceMigration, error)
	Delete(name string, options *k8smetav1.DeleteOptions) error
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachineInstanceMigration, err error)
	UpdateStatus(*v1.VirtualMachineInstanceMigration) (*v1.VirtualMachineInstanceMigration, error)
	PatchStatus(name string, pt types.PatchType, data []byte) (result *v1.VirtualMachineInstanceMigration, err error)
}

type KubeVirtInterface interface {
	Get(name string, options *k8smetav1.GetOptions) (*v1.KubeVirt, error)
	List(opts *k8smetav1.ListOptions) (*v1.KubeVirtList, error)
	Create(instance *v1.KubeVirt) (*v1.KubeVirt, error)
	Update(*v1.KubeVirt) (*v1.KubeVirt, error)
	Delete(name string, options *k8smetav1.DeleteOptions) error
	Patch(name string, pt types.PatchType, data []byte, patchOptions *metav1.PatchOptions, subresources ...string) (result *v1.KubeVirt, err error)
	UpdateStatus(*v1.KubeVirt) (*v1.KubeVirt, error)
	PatchStatus(name string, pt types.PatchType, data []byte, patchOptions *metav1.PatchOptions) (result *v1.KubeVirt, err error)
}
