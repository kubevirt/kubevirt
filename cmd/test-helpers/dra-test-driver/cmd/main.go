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
 * Copyright The KubeVirt Authors.
 *
 */

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/dynamic-resource-allocation/kubeletplugin"
	"k8s.io/dynamic-resource-allocation/resourceslice"

	"kubevirt.io/kubevirt/cmd/test-helpers/dra-test-driver/pkg/driver"
)

const (
	driverName = "hostpath.dra.kubevirt.io"
	maxDevices = 5
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Fatal("NODE_NAME environment variable is required")
	}

	d := &driver.Driver{}
	helper, err := kubeletplugin.Start(ctx, d,
		kubeletplugin.DriverName(driverName),
		kubeletplugin.KubeClient(clientset),
		kubeletplugin.NodeName(nodeName),
	)
	if err != nil {
		log.Fatal(err)
	}

	var devices []resourceapi.Device
	for index := 0; index < maxDevices; index++ {
		devices = append(devices, resourceapi.Device{
			Name: fmt.Sprintf("hostpath-%d", index),
		})
	}

	if err := helper.PublishResources(ctx, resourceslice.DriverResources{
		Pools: map[string]resourceslice.Pool{
			"hostpath": {
				Slices: []resourceslice.Slice{{
					Devices: devices,
				}},
			},
		},
	}); err != nil {
		log.Fatal(err)
	}

	log.Println("DRA plugin started")
	<-ctx.Done()
	helper.Stop()
}
