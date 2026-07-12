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
	"os"
	"os/signal"
	"syscall"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/dynamic-resource-allocation/kubeletplugin"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/cmd/test-helpers/dra-network-test-driver/pkg/driver"
)

func main() {
	log.InitializeLogging("dra-network-test-driver")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Log.Reason(err).Error("Failed to build in-cluster config")
		os.Exit(1)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Log.Reason(err).Error("Failed to create Kubernetes client")
		os.Exit(1)
	}
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Log.Error("NODE_NAME environment variable is required")
		os.Exit(1)
	}

	d := driver.New(cancel)
	helper, err := kubeletplugin.Start(ctx, d,
		kubeletplugin.DriverName(driver.DriverName),
		kubeletplugin.KubeClient(clientset),
		kubeletplugin.NodeName(nodeName),
	)
	if err != nil {
		log.Log.Reason(err).Error("Failed to start the kubelet plugin")
		os.Exit(1)
	}

	log.Log.Info("DRA plugin started")
	<-ctx.Done()
	helper.Stop()
}
