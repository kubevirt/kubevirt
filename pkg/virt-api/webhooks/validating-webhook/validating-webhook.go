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

package validating_webhook

import (
	"net/http"

	"kubevirt.io/client-go/kubecli"
	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/validating-webhook/admitters"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

func ServeVMICreate(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig) {
	validating_webhooks.Serve(resp, req, &admitters.VMICreateAdmitter{ClusterConfig: clusterConfig})
}

func ServeVMIUpdate(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig) {
	validating_webhooks.Serve(resp, req, &admitters.VMIUpdateAdmitter{ClusterConfig: clusterConfig})
}

func ServeVMs(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig, virtCli kubecli.KubevirtClient, informers *webhooks.Informers) {

	validating_webhooks.Serve(resp, req, admitters.NewVMsAdmitter(clusterConfig, virtCli, informers))
}

func ServeVMIRS(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig) {
	validating_webhooks.Serve(resp, req, &admitters.VMIRSAdmitter{ClusterConfig: clusterConfig})
}

func ServeVMIPreset(resp http.ResponseWriter, req *http.Request) {
	validating_webhooks.Serve(resp, req, &admitters.VMIPresetAdmitter{})
}

func ServeMigrationCreate(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig, virtCli kubecli.KubevirtClient, informers *webhooks.Informers) {
	validating_webhooks.Serve(resp, req, &admitters.MigrationCreateAdmitter{ClusterConfig: clusterConfig, VirtClient: virtCli, VMIInformer: informers.VMIInformer})
}

func ServeMigrationUpdate(resp http.ResponseWriter, req *http.Request) {
	validating_webhooks.Serve(resp, req, &admitters.MigrationUpdateAdmitter{})
}

func ServeVMSnapshots(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig, virtCli kubecli.KubevirtClient) {
	validating_webhooks.Serve(resp, req, admitters.NewVMSnapshotAdmitter(clusterConfig, virtCli))
}

func ServeVMRestores(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig, virtCli kubecli.KubevirtClient, informers *webhooks.Informers) {
	validating_webhooks.Serve(resp, req, admitters.NewVMRestoreAdmitter(clusterConfig, virtCli, informers.VMRestoreInformer))
}

func ServeVmFlavors(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig, virtCli kubecli.KubevirtClient) {
	validating_webhooks.Serve(resp, req, &admitters.FlavorAdmitter{})
}

func ServeVmClusterFlavors(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig, virtCli kubecli.KubevirtClient) {
	validating_webhooks.Serve(resp, req, &admitters.ClusterFlavorAdmitter{})
}

func ServeStatusValidation(resp http.ResponseWriter,
	req *http.Request,
	clusterConfig *virtconfig.ClusterConfig,
	virtCli kubecli.KubevirtClient,
	informers *webhooks.Informers) {
	validating_webhooks.Serve(resp, req, &admitters.StatusAdmitter{
		VmsAdmitter: admitters.NewVMsAdmitter(clusterConfig, virtCli, informers),
	})
}

func ServePodEvictionInterceptor(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig, virtCli kubecli.KubevirtClient) {
	validating_webhooks.Serve(resp, req, &admitters.PodEvictionAdmitter{
		ClusterConfig: clusterConfig, VirtClient: virtCli,
	})
}
