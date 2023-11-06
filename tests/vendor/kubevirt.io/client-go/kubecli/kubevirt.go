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
	"context"
	"io"
	"net"
	"time"

	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"

	clonev1alpha1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/clone/v1alpha1"

	secv1 "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	autov1 "k8s.io/api/autoscaling/v1"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
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
	vmexportv1alpha1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/export/v1alpha1"
	instancetypev1beta1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/instancetype/v1beta1"
	migrationsv1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/migrations/v1alpha1"
	poolv1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/pool/v1alpha1"
	vmsnapshotv1alpha1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/snapshot/v1alpha1"
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
	VirtualMachineSnapshot(namespace string) vmsnapshotv1alpha1.VirtualMachineSnapshotInterface
	VirtualMachineSnapshotContent(namespace string) vmsnapshotv1alpha1.VirtualMachineSnapshotContentInterface
	VirtualMachineRestore(namespace string) vmsnapshotv1alpha1.VirtualMachineRestoreInterface
	VirtualMachineExport(namespace string) vmexportv1alpha1.VirtualMachineExportInterface
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

func (k kubevirt) VirtualMachineSnapshot(namespace string) vmsnapshotv1alpha1.VirtualMachineSnapshotInterface {
	return k.generatedKubeVirtClient.SnapshotV1alpha1().VirtualMachineSnapshots(namespace)
}

func (k kubevirt) VirtualMachineSnapshotContent(namespace string) vmsnapshotv1alpha1.VirtualMachineSnapshotContentInterface {
	return k.generatedKubeVirtClient.SnapshotV1alpha1().VirtualMachineSnapshotContents(namespace)
}

func (k kubevirt) VirtualMachineRestore(namespace string) vmsnapshotv1alpha1.VirtualMachineRestoreInterface {
	return k.generatedKubeVirtClient.SnapshotV1alpha1().VirtualMachineRestores(namespace)
}

func (k kubevirt) VirtualMachineExport(namespace string) vmexportv1alpha1.VirtualMachineExportInterface {
	return k.generatedKubeVirtClient.ExportV1alpha1().VirtualMachineExports(namespace)
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

type StreamOptions struct {
	In  io.Reader
	Out io.Writer
}

type StreamInterface interface {
	Stream(options StreamOptions) error
	AsConn() net.Conn
}

type VirtualMachineInstanceInterface interface {
	Get(ctx context.Context, name string, options *metav1.GetOptions) (*v1.VirtualMachineInstance, error)
	List(ctx context.Context, opts *metav1.ListOptions) (*v1.VirtualMachineInstanceList, error)
	Create(ctx context.Context, instance *v1.VirtualMachineInstance) (*v1.VirtualMachineInstance, error)
	Update(ctx context.Context, instance *v1.VirtualMachineInstance) (*v1.VirtualMachineInstance, error)
	Delete(ctx context.Context, name string, options *metav1.DeleteOptions) error
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, patchOptions *metav1.PatchOptions, subresources ...string) (result *v1.VirtualMachineInstance, err error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	SerialConsole(name string, options *SerialConsoleOptions) (StreamInterface, error)
	USBRedir(vmiName string) (StreamInterface, error)
	VNC(name string) (StreamInterface, error)
	Screenshot(ctx context.Context, name string, options *v1.ScreenshotOptions) ([]byte, error)
	PortForward(name string, port int, protocol string) (StreamInterface, error)
	Pause(ctx context.Context, name string, pauseOptions *v1.PauseOptions) error
	Unpause(ctx context.Context, name string, unpauseOptions *v1.UnpauseOptions) error
	Freeze(ctx context.Context, name string, unfreezeTimeout time.Duration) error
	Unfreeze(ctx context.Context, name string) error
	SoftReboot(ctx context.Context, name string) error
	GuestOsInfo(ctx context.Context, name string) (v1.VirtualMachineInstanceGuestAgentInfo, error)
	UserList(ctx context.Context, name string) (v1.VirtualMachineInstanceGuestOSUserList, error)
	FilesystemList(ctx context.Context, name string) (v1.VirtualMachineInstanceFileSystemList, error)
	AddVolume(ctx context.Context, name string, addVolumeOptions *v1.AddVolumeOptions) error
	RemoveVolume(ctx context.Context, name string, removeVolumeOptions *v1.RemoveVolumeOptions) error
	VSOCK(name string, options *v1.VSOCKOptions) (StreamInterface, error)
	SEVFetchCertChain(name string) (v1.SEVPlatformInfo, error)
	SEVQueryLaunchMeasurement(name string) (v1.SEVMeasurementInfo, error)
	SEVSetupSession(name string, sevSessionOptions *v1.SEVSessionOptions) error
	SEVInjectLaunchSecret(name string, sevSecretOptions *v1.SEVSecretOptions) error
}

type ReplicaSetInterface interface {
	Get(name string, options metav1.GetOptions) (*v1.VirtualMachineInstanceReplicaSet, error)
	List(opts metav1.ListOptions) (*v1.VirtualMachineInstanceReplicaSetList, error)
	Create(*v1.VirtualMachineInstanceReplicaSet) (*v1.VirtualMachineInstanceReplicaSet, error)
	Update(*v1.VirtualMachineInstanceReplicaSet) (*v1.VirtualMachineInstanceReplicaSet, error)
	Delete(name string, options *metav1.DeleteOptions) error
	GetScale(replicaSetName string, options metav1.GetOptions) (*autov1.Scale, error)
	UpdateScale(replicaSetName string, scale *autov1.Scale) (*autov1.Scale, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachineInstanceReplicaSet, err error)
	UpdateStatus(*v1.VirtualMachineInstanceReplicaSet) (*v1.VirtualMachineInstanceReplicaSet, error)
	PatchStatus(name string, pt types.PatchType, data []byte) (result *v1.VirtualMachineInstanceReplicaSet, err error)
}

type VirtualMachineInstancePresetInterface interface {
	Get(name string, options metav1.GetOptions) (*v1.VirtualMachineInstancePreset, error)
	List(opts metav1.ListOptions) (*v1.VirtualMachineInstancePresetList, error)
	Create(*v1.VirtualMachineInstancePreset) (*v1.VirtualMachineInstancePreset, error)
	Update(*v1.VirtualMachineInstancePreset) (*v1.VirtualMachineInstancePreset, error)
	Delete(name string, options *metav1.DeleteOptions) error
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachineInstancePreset, err error)
}

// VirtualMachineInterface provides convenience methods to work with
// virtual machines inside the cluster
type VirtualMachineInterface interface {
	Get(ctx context.Context, name string, options *metav1.GetOptions) (*v1.VirtualMachine, error)
	GetWithExpandedSpec(ctx context.Context, name string) (*v1.VirtualMachine, error)
	List(ctx context.Context, opts *metav1.ListOptions) (*v1.VirtualMachineList, error)
	Create(ctx context.Context, vm *v1.VirtualMachine) (*v1.VirtualMachine, error)
	Update(ctx context.Context, vm *v1.VirtualMachine) (*v1.VirtualMachine, error)
	Delete(ctx context.Context, name string, options *metav1.DeleteOptions) error
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, patchOptions *metav1.PatchOptions, subresources ...string) (result *v1.VirtualMachine, err error)
	UpdateStatus(ctx context.Context, vm *v1.VirtualMachine) (*v1.VirtualMachine, error)
	PatchStatus(ctx context.Context, name string, pt types.PatchType, data []byte, patchOptions *metav1.PatchOptions) (result *v1.VirtualMachine, err error)
	Restart(ctx context.Context, name string, restartOptions *v1.RestartOptions) error
	ForceRestart(ctx context.Context, name string, restartOptions *v1.RestartOptions) error
	Start(ctx context.Context, name string, startOptions *v1.StartOptions) error
	Stop(ctx context.Context, name string, stopOptions *v1.StopOptions) error
	ForceStop(ctx context.Context, name string, stopOptions *v1.StopOptions) error
	Migrate(ctx context.Context, name string, migrateOptions *v1.MigrateOptions) error
	AddVolume(ctx context.Context, name string, addVolumeOptions *v1.AddVolumeOptions) error
	RemoveVolume(ctx context.Context, name string, removeVolumeOptions *v1.RemoveVolumeOptions) error
	PortForward(name string, port int, protocol string) (StreamInterface, error)
	MemoryDump(ctx context.Context, name string, memoryDumpRequest *v1.VirtualMachineMemoryDumpRequest) error
	RemoveMemoryDump(ctx context.Context, name string) error
}

type VirtualMachineInstanceMigrationInterface interface {
	Get(name string, options *metav1.GetOptions) (*v1.VirtualMachineInstanceMigration, error)
	List(opts *metav1.ListOptions) (*v1.VirtualMachineInstanceMigrationList, error)
	Create(migration *v1.VirtualMachineInstanceMigration, options *metav1.CreateOptions) (*v1.VirtualMachineInstanceMigration, error)
	Update(*v1.VirtualMachineInstanceMigration) (*v1.VirtualMachineInstanceMigration, error)
	Delete(name string, options *metav1.DeleteOptions) error
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachineInstanceMigration, err error)
	UpdateStatus(*v1.VirtualMachineInstanceMigration) (*v1.VirtualMachineInstanceMigration, error)
	PatchStatus(name string, pt types.PatchType, data []byte) (result *v1.VirtualMachineInstanceMigration, err error)
}

type KubeVirtInterface interface {
	Get(name string, options *metav1.GetOptions) (*v1.KubeVirt, error)
	List(opts *metav1.ListOptions) (*v1.KubeVirtList, error)
	Create(instance *v1.KubeVirt) (*v1.KubeVirt, error)
	Update(*v1.KubeVirt) (*v1.KubeVirt, error)
	Delete(name string, options *metav1.DeleteOptions) error
	Patch(name string, pt types.PatchType, data []byte, patchOptions *metav1.PatchOptions, subresources ...string) (result *v1.KubeVirt, err error)
	UpdateStatus(*v1.KubeVirt) (*v1.KubeVirt, error)
	PatchStatus(name string, pt types.PatchType, data []byte, patchOptions *metav1.PatchOptions) (result *v1.KubeVirt, err error)
}

type ServerVersionInterface interface {
	Get() (*version.Info, error)
}

type ExpandSpecInterface interface {
	ForVirtualMachine(vm *v1.VirtualMachine) (*v1.VirtualMachine, error)
}
