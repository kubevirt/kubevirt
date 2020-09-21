package libnet

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/kubecli"
)

func SkipWhenNotDualStackCluster(virtClient kubecli.KubevirtClient) {
	isClusterDualStack, err := IsClusterDualStack(virtClient)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "should have been able to infer if the cluster is dual stack")
	if !isClusterDualStack {
		Skip("This test requires a dual stack network config.")
	}
}
