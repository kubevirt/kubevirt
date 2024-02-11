package libshadownode

import (
	"context"

	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

func GetAllSchedulableShadowNodes(virtClient kubecli.KubevirtClient) *v1.ShadowNodeList {
	const true = "true"
	nodes, err := virtClient.ShadowNodeClient().List(context.Background(), k8smetav1.ListOptions{LabelSelector: v1.NodeSchedulable + "=" + true})
	Expect(err).ToNot(HaveOccurred(), "Should list compute shadowNodes")
	return nodes
}
