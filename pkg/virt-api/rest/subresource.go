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

package rest

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"k8s.io/client-go/kubernetes"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	defaultProfilerComponentPort = 8443
)

type SubresourceAPIApp struct {
	virtClient              kubecli.KubevirtClient
	k8sClient               kubernetes.Interface
	consoleServerPort       int
	profilerComponentPort   int
	handlerTLSConfiguration *tls.Config
	clusterConfig           *virtconfig.ClusterConfig
	handlerHttpClient       *http.Client
}

func NewSubresourceAPIApp(virtClient kubecli.KubevirtClient, k8sClient kubernetes.Interface, consoleServerPort int, tlsConfiguration *tls.Config, clusterConfig *virtconfig.ClusterConfig) *SubresourceAPIApp {
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfiguration,
		},
		Timeout: 10 * time.Second,
	}

	return &SubresourceAPIApp{
		virtClient:              virtClient,
		k8sClient:               k8sClient,
		consoleServerPort:       consoleServerPort,
		profilerComponentPort:   defaultProfilerComponentPort,
		handlerTLSConfiguration: tlsConfiguration,
		clusterConfig:           clusterConfig,
		handlerHttpClient:       httpClient,
	}
}

func (app *SubresourceAPIApp) getVirtHandlerConnForVMI(vmi *v1.VirtualMachineInstance) (kubecli.VirtHandlerConn, error) {
	if !vmi.IsRunning() && !vmi.IsScheduled() {
		return nil, fmt.Errorf("Unable to connect to VirtualMachineInstance because phase is %s instead of %s or %s", vmi.Status.Phase, v1.Running, v1.Scheduled)
	}
	return kubecli.NewVirtHandlerClient(app.virtClient, app.handlerHttpClient).Port(app.consoleServerPort).ForNode(vmi.Status.NodeName), nil
}
