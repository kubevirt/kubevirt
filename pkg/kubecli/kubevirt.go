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

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

type KubevirtClient interface {
	VirtualMachineInstance(namespace string) VirtualMachineInstanceInterface
	ReplicaSet(namespace string) ReplicaSetInterface
	VirtualMachine(namespace string) VirtualMachineInterface
	ServerVersion() *ServerVersion
	RestClient() *rest.RESTClient
	kubernetes.Interface
}

type kubevirt struct {
	master     string
	kubeconfig string
	restClient *rest.RESTClient
	config     *rest.Config
	*kubernetes.Clientset
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
	SerialConsole(name string) (StreamInterface, error)
	VNC(name string) (StreamInterface, error)
}

type ReplicaSetInterface interface {
	Get(name string, options k8smetav1.GetOptions) (*v1.VirtualMachineInstanceReplicaSet, error)
	List(opts k8smetav1.ListOptions) (*v1.VirtualMachineInstanceReplicaSetList, error)
	Create(*v1.VirtualMachineInstanceReplicaSet) (*v1.VirtualMachineInstanceReplicaSet, error)
	Update(*v1.VirtualMachineInstanceReplicaSet) (*v1.VirtualMachineInstanceReplicaSet, error)
	Delete(name string, options *k8smetav1.DeleteOptions) error
}

type VMIPresetInterface interface {
	Get(name string, options k8smetav1.GetOptions) (*v1.VirtualMachineInstancePreset, error)
	List(opts k8smetav1.ListOptions) (*v1.VirtualMachineInstancePresetList, error)
	Create(*v1.VirtualMachineInstancePreset) (*v1.VirtualMachineInstancePreset, error)
	Update(*v1.VirtualMachineInstancePreset) (*v1.VirtualMachineInstancePreset, error)
	Delete(name string, options *k8smetav1.DeleteOptions) error
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachineInstancePreset, err error)
}

// VirtualMachineInterface provides convenience methods to work with
// offline virtual machines inside the cluster
type VirtualMachineInterface interface {
	Get(name string, options *k8smetav1.GetOptions) (*v1.VirtualMachine, error)
	List(opts *k8smetav1.ListOptions) (*v1.VirtualMachineList, error)
	Create(*v1.VirtualMachine) (*v1.VirtualMachine, error)
	Update(*v1.VirtualMachine) (*v1.VirtualMachine, error)
	Delete(name string, options *k8smetav1.DeleteOptions) error
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachine, err error)
}
