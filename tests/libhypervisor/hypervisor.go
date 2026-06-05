package libhypervisor

import (
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/hypervisor"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/libkubevirt"
)

func GetHypervisorDeviceName(virtClient kubecli.KubevirtClient) string {
	kv := libkubevirt.GetCurrentKv(virtClient)
	hypervisorName := virtconfig.GetHypervisorFromKvConfig(&kv.Spec.Configuration, checks.HasFeature(featuregate.ConfigurableHypervisor)).Name
	hypervisorLauncherResources := hypervisor.NewLauncherHypervisorResources(hypervisorName)
	return hypervisorLauncherResources.GetHypervisorDevice()
}
