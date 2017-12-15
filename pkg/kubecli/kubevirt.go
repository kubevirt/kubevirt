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
 * Copyright 2017 Red Hat, Inc.
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
	VM(namespace string) VMInterface
	Migration(namespace string) MigrationInterface
	ReplicaSet(namespace string) ReplicaSetInterface
	RestClient() *rest.RESTClient
	kubernetes.Interface
}

type kubevirt struct {
	master     string
	kubeconfig string
	restClient *rest.RESTClient
	*kubernetes.Clientset
}

func (k kubevirt) RestClient() *rest.RESTClient {
	return k.restClient
}

type VMInterface interface {
	Get(name string, options k8smetav1.GetOptions) (*v1.VirtualMachine, error)
	List(opts k8smetav1.ListOptions) (*v1.VirtualMachineList, error)
	Create(*v1.VirtualMachine) (*v1.VirtualMachine, error)
	Update(*v1.VirtualMachine) (*v1.VirtualMachine, error)
	Delete(name string, options *k8smetav1.DeleteOptions) error
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachine, err error)
	SerialConsole(name string, device string, in io.Reader, out io.Writer) error
}

type ReplicaSetInterface interface {
	Get(name string, options k8smetav1.GetOptions) (*v1.VirtualMachineReplicaSet, error)
	List(opts k8smetav1.ListOptions) (*v1.VirtualMachineReplicaSetList, error)
	Create(*v1.VirtualMachineReplicaSet) (*v1.VirtualMachineReplicaSet, error)
	Update(*v1.VirtualMachineReplicaSet) (*v1.VirtualMachineReplicaSet, error)
	Delete(name string, options *k8smetav1.DeleteOptions) error
}

type MigrationInterface interface {
	Get(name string, options k8smetav1.GetOptions) (*v1.Migration, error)
	List(opts k8smetav1.ListOptions) (*v1.MigrationList, error)
	Create(*v1.Migration) (*v1.Migration, error)
	Update(*v1.Migration) (*v1.Migration, error)
	Delete(name string, options *k8smetav1.DeleteOptions) error
}
