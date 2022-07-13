package checks

import (
	"fmt"
	"time"

	"kubevirt.io/kubevirt/tests/libnode"

	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/util/cluster"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/util"

	v12 "kubevirt.io/api/core/v1"
)

func IsCPUManagerPresent(node *v1.Node) bool {
	gomega.Expect(node).ToNot(gomega.BeNil())
	nodeHaveCpuManagerLabel := false

	for label, val := range node.Labels {
		if label == v12.CPUManager && val == "true" {
			nodeHaveCpuManagerLabel = true
			break
		}
	}
	return nodeHaveCpuManagerLabel
}

func IsRealtimeCapable(node *v1.Node) bool {
	gomega.Expect(node).ToNot(gomega.BeNil())
	for label, _ := range node.Labels {
		if label == v12.RealtimeLabel {
			return true
		}
	}
	return false
}

func Has2MiHugepages(node *v1.Node) bool {
	gomega.Expect(node).ToNot(gomega.BeNil())
	_, exists := node.Status.Capacity[v1.ResourceHugePagesPrefix+"2Mi"]
	return exists
}

func HasFeature(feature string) bool {
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	featureGates := []string{}
	kv := util.GetCurrentKv(virtClient)
	if kv.Spec.Configuration.DeveloperConfiguration != nil {
		featureGates = kv.Spec.Configuration.DeveloperConfiguration.FeatureGates
	}

	for _, fg := range featureGates {
		if fg == feature {
			return true
		}
	}

	return false
}

func IsSEVCapable(node *v1.Node) bool {
	gomega.Expect(node).ToNot(gomega.BeNil())
	for label, _ := range node.Labels {
		if label == v12.SEVLabel {
			return true
		}
	}
	return false
}

func IsARM64(arch string) bool {
	return arch == "arm64"
}

func HasLiveMigration() bool {
	return HasFeature("LiveMigration")
}

func HasAtLeastTwoNodes() bool {
	var nodes *v1.NodeList
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	gomega.Eventually(func() []v1.Node {
		nodes = libnode.GetAllSchedulableNodes(virtClient)
		return nodes.Items
	}, 60*time.Second, time.Second).ShouldNot(gomega.BeEmpty(), "There should be some compute node")

	return len(nodes.Items) >= 2
}

func IsOpenShift() bool {
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	isOpenShift, err := cluster.IsOnOpenShift(virtClient)
	if err != nil {
		fmt.Printf("ERROR: Can not determine cluster type %v\n", err)
		panic(err)
	}

	return isOpenShift
}
