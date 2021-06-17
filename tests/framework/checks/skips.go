package checks

import (
	"github.com/onsi/ginkgo"

	"kubevirt.io/client-go/kubecli"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests/util"
)

func SkipTestIfNoCPUManager() {
	if !HasFeature(virtconfig.CPUManager) {
		ginkgo.Skip("the CPUManager feature gate is not enabled.")
	}

	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
	nodes := util.GetAllSchedulableNodes(virtClient)

	for _, node := range nodes.Items {
		if IsCPUManagerPresent(&node) {
			return
		}
	}
	ginkgo.Skip("no node with CPUManager detected", 1)
}
