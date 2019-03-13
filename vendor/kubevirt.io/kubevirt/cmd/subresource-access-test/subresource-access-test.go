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
	"fmt"
	"os"

	"k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/kubecli"
)

func main() {
	var statusCode int
	flag.Parse()

	// creates the connection
	client, err := kubecli.GetKubevirtSubresourceClient()
	if err != nil {
		panic(err)
	}

	restClient := client.RestClient()
	var result rest.Result

	if os.Args[1] == "version" {
		result = restClient.Get().Resource("version").Do()
	} else {
		result = restClient.Get().Resource("virtualmachineinstances").Namespace("default").Name("fake").SubResource("test").Do()
	}

	err = result.Error()
	if err != nil {
		panic(err)
	}

	result.StatusCode(&statusCode)
	if statusCode != 200 {
		panic(fmt.Errorf("http status code is %d", statusCode))
	} else {
		fmt.Println("Subresource Test Endpoint returned 200 OK")
	}
}
