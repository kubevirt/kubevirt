package util

import (
	"fmt"
	"time"

	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/flags"

	v13 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
)

func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func GetAllSchedulableNodes(virtClient kubecli.KubevirtClient) *v1.NodeList {
	nodes, err := virtClient.CoreV1().Nodes().List(v12.ListOptions{LabelSelector: v13.NodeSchedulable + "=" + "true"})
	gomega.Expect(err).ToNot(gomega.HaveOccurred(), "Should list compute nodes")
	return nodes
}

func DetectInstallNamespace() {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	kvs, err := virtCli.KubeVirt("").List(&v12.ListOptions{})
	PanicOnError(err)
	if len(kvs.Items) == 0 {
		PanicOnError(fmt.Errorf("Could not detect a kubevirt installation"))
	}
	if len(kvs.Items) > 1 {
		PanicOnError(fmt.Errorf("Invalid kubevirt installation, more than one KubeVirt resource found"))
	}
	flags.KubeVirtInstallNamespace = kvs.Items[0].Namespace
}

func GetCurrentKv(virtClient kubecli.KubevirtClient) *v13.KubeVirt {
	kvs := GetKvList(virtClient)
	gomega.Expect(len(kvs)).To(gomega.Equal(1))
	return &kvs[0]
}

func GetKvList(virtClient kubecli.KubevirtClient) []v13.KubeVirt {
	var kvListInstallNS *v13.KubeVirtList
	var items []v13.KubeVirt

	var err error

	gomega.Eventually(func() error {

		kvListInstallNS, err = virtClient.KubeVirt(flags.KubeVirtInstallNamespace).List(&v12.ListOptions{})

		return err
	}, 10*time.Second, 1*time.Second).ShouldNot(gomega.HaveOccurred())

	items = append(items, kvListInstallNS.Items...)

	return items
}
