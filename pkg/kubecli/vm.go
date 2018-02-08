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

import (
	goerror "errors"
	"fmt"
	"io"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"

	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

func (k *kubevirt) VM(namespace string) VMInterface {
	return &vms{
		restClient: k.restClient,
		clientSet:  k.Clientset,
		namespace:  namespace,
		resource:   "virtualmachines",
		master:     k.master,
		kubeconfig: k.kubeconfig,
	}
}

type vms struct {
	restClient *rest.RESTClient
	clientSet  *kubernetes.Clientset
	namespace  string
	resource   string
	master     string
	kubeconfig string
}

func findPod(clientSet *kubernetes.Clientset, namespace string, name string) (string, error) {
	fieldSelector := fields.ParseSelectorOrDie("status.phase==" + string(k8sv1.PodRunning))
	labelSelector, err := labels.Parse(fmt.Sprintf(v1.AppLabel+"=virt-launcher,"+v1.DomainLabel+" in (%s)", name))
	if err != nil {
		return "", err
	}
	selector := k8smetav1.ListOptions{FieldSelector: fieldSelector.String(), LabelSelector: labelSelector.String()}

	podList, err := clientSet.CoreV1().Pods(namespace).List(selector)
	if err != nil {
		return "", err
	}

	if len(podList.Items) == 0 {
		return "", goerror.New("console connection failed. No VM pod is running")
	}
	return podList.Items[0].ObjectMeta.Name, nil
}

func (v *vms) remoteExecHelper(name string, cmd []string, in io.Reader, out io.Writer) error {

	// ensure VM is in running phase before attempting to connect.
	vm, err := v.Get(name, k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return goerror.New(fmt.Sprintf("Unable to connect to VM %s in namespace %s does not exist.", name, v.namespace))
		}
		return err
	}

	if vm.IsRunning() == false {
		return goerror.New(fmt.Sprintf("Unable to connect to VM because phase is %s instead of %s", vm.Status.Phase, v1.Running))
	}

	podName, err := findPod(v.clientSet, v.namespace, name)
	if err != nil {
		return fmt.Errorf("unable to find matching pod for remote execution: %v", err)
	}

	config, err := clientcmd.BuildConfigFromFlags(v.master, v.kubeconfig)
	if err != nil {
		return fmt.Errorf("unable to build api config for remote execution: %v", err)
	}

	gv := k8sv1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/api"
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	restClient, err := restclient.RESTClientFor(config)
	if err != nil {
		return fmt.Errorf("unable to create restClient for remote execution: %v", err)
	}
	containerName := "compute"
	req := restClient.Post().
		Resource("pods").
		Name(podName).
		Namespace(v.namespace).
		SubResource("exec").
		Param("container", containerName)

	req = req.VersionedParams(&k8sv1.PodExecOptions{
		Container: containerName,
		Command:   cmd,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)

	// execute request
	method := "POST"
	url := req.URL()
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return fmt.Errorf("remote execution failed: %v", err)
	}

	return exec.Stream(remotecommand.StreamOptions{
		Stdin:             in,
		Stdout:            out,
		Stderr:            out,
		Tty:               false,
		TerminalSizeQueue: nil,
	})
}

func (v *vms) VNC(name string, in io.Reader, out io.Writer) error {
	cmd := []string{"/sock-connector", fmt.Sprintf("/var/run/kubevirt-private/%s/%s/virt-vnc", v.namespace, name)}
	return v.remoteExecHelper(name, cmd, in, out)
}

func (v *vms) SerialConsole(name string, device string, in io.Reader, out io.Writer) error {
	cmd := []string{"/sock-connector", fmt.Sprintf("/var/run/kubevirt-private/%s/%s/virt-%s", v.namespace, name, device)}
	return v.remoteExecHelper(name, cmd, in, out)
}

func (v *vms) Get(name string, options k8smetav1.GetOptions) (vm *v1.VirtualMachine, err error) {
	vm = &v1.VirtualMachine{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(vm)
	vm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)
	return
}

func (v *vms) List(options k8smetav1.ListOptions) (vmList *v1.VirtualMachineList, err error) {
	vmList = &v1.VirtualMachineList{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(vmList)
	for _, vm := range vmList.Items {
		vm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)
	}

	return
}

func (v *vms) Create(vm *v1.VirtualMachine) (result *v1.VirtualMachine, err error) {
	result = &v1.VirtualMachine{}
	err = v.restClient.Post().
		Namespace(v.namespace).
		Resource(v.resource).
		Body(vm).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)
	return
}

func (v *vms) Update(vm *v1.VirtualMachine) (result *v1.VirtualMachine, err error) {
	result = &v1.VirtualMachine{}
	err = v.restClient.Put().
		Name(vm.ObjectMeta.Name).
		Namespace(v.namespace).
		Resource(v.resource).
		Body(vm).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)
	return
}

func (v *vms) Delete(name string, options *k8smetav1.DeleteOptions) error {
	return v.restClient.Delete().
		Namespace(v.namespace).
		Resource(v.resource).
		Name(name).
		Body(options).
		Do().
		Error()
}

func (v *vms) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachine, err error) {
	result = &v1.VirtualMachine{}
	err = v.restClient.Patch(pt).
		Namespace(v.namespace).
		Resource(v.resource).
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
