package libnet

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/libnet/cluster"

	"kubevirt.io/client-go/kubecli"
)

func SkipWhenNotDualStackCluster(virtClient kubecli.KubevirtClient) {
	isClusterDualStack, err := cluster.DualStack(virtClient)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "should have been able to infer if the cluster is dual stack")
	if !isClusterDualStack {
		Skip("This test requires a dual stack network config.")
	}
}

func SkipWhenClusterNotSupportIpv4(virtClient kubecli.KubevirtClient) {
	clusterSupportsIpv4, err := cluster.SupportsIpv4(virtClient)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "should have been able to infer if the cluster supports ipv4")
	if !clusterSupportsIpv4 {
		Skip("This test requires an ipv4 network config.")
	}
}

func SkipWhenClusterNotSupportIpv6(virtClient kubecli.KubevirtClient) {
	clusterSupportsIpv6, err := cluster.SupportsIpv6(virtClient)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "should have been able to infer if the cluster supports ipv6")
	if !clusterSupportsIpv6 {
		Skip("This test requires an ipv6 network config.")
	}
}

func SkipWhenClusterNotSupportIpFamily(virtClient kubecli.KubevirtClient, ipFamily k8sv1.IPFamily) {
	if ipFamily == k8sv1.IPv4Protocol {
		SkipWhenClusterNotSupportIpv4(virtClient)
	} else {
		SkipWhenClusterNotSupportIpv6(virtClient)
	}
}
