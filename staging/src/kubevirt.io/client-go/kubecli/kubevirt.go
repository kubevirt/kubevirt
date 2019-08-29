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
	"time"

	secv1 "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	autov1 "k8s.io/api/autoscaling/v1"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	networkclient "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned"

	promclient "github.com/coreos/prometheus-operator/pkg/client/versioned"

	v1 "kubevirt.io/client-go/api/v1"
	cdiclient "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned"
)

type KubevirtClient interface {
	VirtualMachineInstance(namespace string) VirtualMachineInstanceInterface
	VirtualMachineInstanceMigration(namespace string) VirtualMachineInstanceMigrationInterface
	ReplicaSet(namespace string) ReplicaSetInterface
	VirtualMachine(namespace string) VirtualMachineInterface
	KubeVirt(namespace string) KubeVirtInterface
	VirtualMachineInstancePreset(namespace string) VirtualMachineInstancePresetInterface
	ServerVersion() *ServerVersion
	RestClient() *rest.RESTClient
	CdiClient() cdiclient.Interface
	NetworkClient() networkclient.Interface
	ExtensionsClient() extclient.Interface
	SecClient() secv1.SecurityV1Interface
	DiscoveryClient() discovery.DiscoveryInterface
	PrometheusClient() promclient.Interface
	kubernetes.Interface
	Config() *rest.Config
}

type kubevirt struct {
	master           string
	kubeconfig       string
	restClient       *rest.RESTClient
	config           *rest.Config
	cdiClient        *cdiclient.Clientset
	networkClient    *networkclient.Clientset
	extensionsClient *extclient.Clientset
	secClient        *secv1.SecurityV1Client
	discoveryClient  *discovery.DiscoveryClient
	prometheusClient *promclient.Clientset
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

type StreamOptions struct {
	In  io.Reader
	Out io.Writer
}

type StreamInterface interface {
	Stream(options StreamOptions) error
}

type VirtualMachineInstanceInterface interface {
	Get(name string, options *k8smetav1.GetOptions) (*v1.VirtualMachineInstance, error)
	List(opts *k8smetav1.ListOptions) (*v1.VirtualMachineInstanceList, error)
	Create(instance *v1.VirtualMachineInstance) (*v1.VirtualMachineInstance, error)
	Update(*v1.VirtualMachineInstance) (*v1.VirtualMachineInstance, error)
	Delete(name string, options *k8smetav1.DeleteOptions) error
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachineInstance, err error)
	SerialConsole(name string, timeout time.Duration) (StreamInterface, error)
	VNC(name string) (StreamInterface, error)
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
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachine, err error)
	Restart(name string) error
	Start(name string) error
	Stop(name string) error
	Migrate(name string) error
}

type VirtualMachineInstanceMigrationInterface interface {
	Get(name string, options *k8smetav1.GetOptions) (*v1.VirtualMachineInstanceMigration, error)
	List(opts *k8smetav1.ListOptions) (*v1.VirtualMachineInstanceMigrationList, error)
	Create(*v1.VirtualMachineInstanceMigration) (*v1.VirtualMachineInstanceMigration, error)
	Update(*v1.VirtualMachineInstanceMigration) (*v1.VirtualMachineInstanceMigration, error)
	Delete(name string, options *k8smetav1.DeleteOptions) error
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachineInstanceMigration, err error)
}

type KubeVirtInterface interface {
	Get(name string, options *k8smetav1.GetOptions) (*v1.KubeVirt, error)
	List(opts *k8smetav1.ListOptions) (*v1.KubeVirtList, error)
	Create(instance *v1.KubeVirt) (*v1.KubeVirt, error)
	Update(*v1.KubeVirt) (*v1.KubeVirt, error)
	Delete(name string, options *k8smetav1.DeleteOptions) error
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.KubeVirt, err error)
}
