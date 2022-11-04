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
	"context"
	"flag"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

func genericRequest(resource string) {
	client, err := kubecli.GetKubevirtSubresourceClient()
	if err != nil {
		panic(err)
	}

	result := client.RestClient().Get().Resource(resource).Do(context.Background())
	err = result.Error()
	if err != nil {
		panic(err)
	}

	var statusCode int
	result.StatusCode(&statusCode)
	if statusCode != 200 {
		panic(fmt.Errorf("http status code is %d", statusCode))
	}
	fmt.Println("Subresource Test Endpoint returned 200 OK")
}

func expandSpecRequest(namespace string) {
	vm := &v1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vm-test",
			Namespace: namespace,
		},
		Spec: v1.VirtualMachineSpec{
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{},
					},
				},
			},
		},
	}
	vm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)

	client, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}
	_, err = client.ExpandSpec(namespace).ForVirtualMachine(vm)
	if err != nil {
		panic(err)
	}
}

func startVM(namespace string, resource string) {
	client, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}
	err = client.VirtualMachine(namespace).Start(resource, &v1.StartOptions{})
	if err != nil {
		panic(err)
	}
}

func main() {
	var namespace string
	var resource string
	flag.StringVar(&namespace, "n", "", "namespace to use")
	flag.Parse()

	resource = flag.Arg(0)
	if resource == "version" || resource == "guestfs" {
		genericRequest(resource)
	} else if resource == "expand-vm-spec" {
		expandSpecRequest(namespace)
	} else {
		startVM(namespace, resource)
	}
}
