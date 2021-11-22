package checks

import (
	"fmt"

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

func SkipTestIfNoFeatureGate(featureGate string) {
	if !HasFeature(featureGate) {
		ginkgo.Skip(fmt.Sprintf("the %v feature gate is not enabled.", featureGate))
	}
}

func SkipTestIfNotEnoughNodesWithCPUManagerWith2MiHugepages(nodeCount int) {
	if !HasFeature(virtconfig.CPUManager) {
		ginkgo.Skip("the CPUManager feature gate is not enabled.")
	}

	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
	nodes := util.GetAllSchedulableNodes(virtClient)

	found := 0
	for _, node := range nodes.Items {
		if IsCPUManagerPresent(&node) && Has2MiHugepages(&node) {
			found++
		}
	}

	if found < nodeCount {
		msg := fmt.Sprintf(
			"not enough node with CPUManager and 2Mi hugepages detected: expected %v nodes, but got %v",
			nodeCount,
			found,
		)
		ginkgo.Skip(msg, 1)
	}
}

func SkipTestIfNotRealtimeCapable() {

	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
	nodes := util.GetAllSchedulableNodes(virtClient)

	for _, node := range nodes.Items {
		if IsRealtimeCapable(&node) && IsCPUManagerPresent(&node) && Has2MiHugepages(&node) {
			return
		}
	}
	ginkgo.Skip("no node capable of running realtime workloads detected", 1)

}
