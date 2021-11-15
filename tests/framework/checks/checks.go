package checks

import (
	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"

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
