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
 * Copyright 2024 The Kubevirt Authors
 *
 */

package convertmachinetype

import (
	"fmt"
	"os"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
)

func Run() {
	// check env variables and set them accordingly
	var (
		restartRequired bool
		targetNs        = metav1.NamespaceAll
		labelSelector   = labels.Everything()
		err             error
	)

	machineTypeEnv, exists := os.LookupEnv("MACHINE_TYPE")
	if !exists {
		fmt.Println("No machine type was specified.")
		os.Exit(1)
	}

	restartEnv, exists := os.LookupEnv("RESTART_REQUIRED")
	if exists {
		restartRequired, err = strconv.ParseBool(restartEnv)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	namespaceEnv, exists := os.LookupEnv("TARGET_NS")
	if exists && namespaceEnv != "" {
		targetNs = namespaceEnv
	}

	fmt.Println("Setting label selector")
	selectorEnv, exists := os.LookupEnv("LABEL_SELECTOR")
	if exists {
		ls, err := labels.ConvertSelectorToLabelsMap(selectorEnv)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		labelSelector, err = ls.AsValidatedSelector()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	// set up JobController
	virtCli, err := getVirtCli()
	if err != nil {
		os.Exit(1)
	}

	var vmListWatcher *cache.ListWatch
	var vmiListWatcher *cache.ListWatch

	vmListWatcher = controller.NewListWatchFromClient(virtCli.RestClient(), "virtualmachines", targetNs, fields.Everything(), labelSelector)
	vmiListWatcher = controller.NewListWatchFromClient(virtCli.RestClient(), "virtualmachineinstances", targetNs, fields.Everything(), labelSelector)
	vmInformer := cache.NewSharedIndexInformer(vmListWatcher, &k6tv1.VirtualMachine{}, 1*time.Hour, cache.Indexers{})
	vmiInformer := cache.NewSharedIndexInformer(vmiListWatcher, &k6tv1.VirtualMachineInstance{}, 1*time.Hour, cache.Indexers{})

	jobController, err := NewJobController(vmInformer, vmiInformer, virtCli, machineTypeEnv, restartRequired)
	if err != nil {
		os.Exit(1)
	}

	go jobController.run(jobController.exitJobChan)
	<-jobController.exitJobChan
	os.Exit(0)
}

func getVirtCli() (kubecli.KubevirtClient, error) {
	clientConfig, err := kubecli.GetKubevirtClientConfig()
	if err != nil {
		return nil, err
	}

	virtCli, err := kubecli.GetKubevirtClientFromRESTConfig(clientConfig)
	if err != nil {
		return nil, err
	}

	return virtCli, err
}
