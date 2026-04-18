/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package validating_webhook

import (
	"net/http"

	"kubevirt.io/client-go/kubecli"

	preferencewebhooks "kubevirt.io/kubevirt/pkg/instancetype/preference/webhooks"
	instancetypewebhooks "kubevirt.io/kubevirt/pkg/instancetype/webhooks"
	storageadmitters "kubevirt.io/kubevirt/pkg/storage/admitters"
	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/validating-webhook/admitters"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

func ServeVMICreate(
	resp http.ResponseWriter,
	req *http.Request,
	clusterConfig *virtconfig.ClusterConfig,
	kubeVirtServiceAccounts map[string]struct{},
	specValidators ...admitters.SpecValidator,
) {
	validating_webhooks.Serve(resp, req, &admitters.VMICreateAdmitter{
		ClusterConfig:           clusterConfig,
		KubeVirtServiceAccounts: kubeVirtServiceAccounts,
		SpecValidators:          specValidators,
	})
}

func ServeVMIUpdate(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig, kubeVirtServiceAccounts map[string]struct{}) {
	validating_webhooks.Serve(resp, req, admitters.NewVMIUpdateAdmitter(clusterConfig, kubeVirtServiceAccounts))
}

func ServeVMs(
	resp http.ResponseWriter,
	req *http.Request,
	clusterConfig *virtconfig.ClusterConfig,
	virtCli kubecli.KubevirtClient,
	informers *webhooks.Informers,
	kubeVirtServiceAccounts map[string]struct{},
) {
	validating_webhooks.Serve(resp, req, admitters.NewVMsAdmitter(clusterConfig, virtCli, informers, kubeVirtServiceAccounts))
}

func ServeVMIRS(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig) {
	validating_webhooks.Serve(resp, req, &admitters.VMIRSAdmitter{ClusterConfig: clusterConfig})
}

func ServeVMPool(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig, kubeVirtServiceAccounts map[string]struct{}) {
	validating_webhooks.Serve(resp, req, &admitters.VMPoolAdmitter{ClusterConfig: clusterConfig, KubeVirtServiceAccounts: kubeVirtServiceAccounts})
}

func ServeVMIPreset(resp http.ResponseWriter, req *http.Request) {
	validating_webhooks.Serve(resp, req, &admitters.VMIPresetAdmitter{})
}

func ServeMigrationCreate(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig, virtCli kubecli.KubevirtClient, kubeVirtServiceAccounts map[string]struct{}) {
	validating_webhooks.Serve(resp, req, admitters.NewMigrationCreateAdmitter(virtCli.GeneratedKubeVirtClient(), clusterConfig, kubeVirtServiceAccounts))
}

func ServeMigrationUpdate(resp http.ResponseWriter, req *http.Request) {
	validating_webhooks.Serve(resp, req, &admitters.MigrationUpdateAdmitter{})
}

func ServeVMSnapshots(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig, virtCli kubecli.KubevirtClient) {
	validating_webhooks.Serve(resp, req, storageadmitters.NewVMSnapshotAdmitter(clusterConfig, virtCli))
}

func ServeVMRestores(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig, virtCli kubecli.KubevirtClient, informers *webhooks.Informers) {
	validating_webhooks.Serve(resp, req, storageadmitters.NewVMRestoreAdmitter(clusterConfig, virtCli, informers.VMRestoreInformer))
}

func ServeVMBackups(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig, virtCli kubecli.KubevirtClient, informers *webhooks.Informers) {
	validating_webhooks.Serve(resp, req, storageadmitters.NewVMBackupAdmitter(clusterConfig, virtCli, informers.VMBackupInformer))
}

func ServeVMBackupTrackers(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig) {
	validating_webhooks.Serve(resp, req, storageadmitters.NewVMBackupTrackerAdmitter(clusterConfig))
}

func ServeVMExports(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig) {
	validating_webhooks.Serve(resp, req, storageadmitters.NewVMExportAdmitter(clusterConfig))
}

func ServeVmInstancetypes(resp http.ResponseWriter, req *http.Request) {
	validating_webhooks.Serve(resp, req, &instancetypewebhooks.InstancetypeAdmitter{})
}

func ServeVmClusterInstancetypes(resp http.ResponseWriter, req *http.Request) {
	validating_webhooks.Serve(resp, req, &instancetypewebhooks.ClusterInstancetypeAdmitter{})
}

func ServeVmPreferences(resp http.ResponseWriter, req *http.Request) {
	validating_webhooks.Serve(resp, req, &preferencewebhooks.PreferenceAdmitter{})
}

func ServeVmClusterPreferences(resp http.ResponseWriter, req *http.Request) {
	validating_webhooks.Serve(resp, req, &preferencewebhooks.ClusterPreferenceAdmitter{})
}

func ServeStatusValidation(resp http.ResponseWriter,
	req *http.Request,
	clusterConfig *virtconfig.ClusterConfig,
	virtCli kubecli.KubevirtClient,
	informers *webhooks.Informers,
	kubeVirtServiceAccounts map[string]struct{},
) {
	validating_webhooks.Serve(resp, req, &admitters.StatusAdmitter{
		VmsAdmitter: admitters.NewVMsAdmitter(clusterConfig, virtCli, informers, kubeVirtServiceAccounts),
	})
}

func ServePodEvictionInterceptor(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig, virtCli kubecli.KubevirtClient) {
	validating_webhooks.Serve(resp, req, admitters.NewPodEvictionAdmitter(clusterConfig, virtCli, virtCli.GeneratedKubeVirtClient()))
}

func ServeMigrationPolicies(resp http.ResponseWriter, req *http.Request) {
	validating_webhooks.Serve(resp, req, admitters.NewMigrationPolicyAdmitter())
}

func ServeVirtualMachineClones(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig, virtCli kubecli.KubevirtClient) {
	validating_webhooks.Serve(resp, req, admitters.NewVMCloneAdmitter(clusterConfig, virtCli))
}
