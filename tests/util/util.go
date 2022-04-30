package util

import (
	"time"

	"github.com/onsi/gomega"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k6sv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests/flags"
)

// tests.NamespaceTestDefault is the default namespace, to test non-infrastructure related KubeVirt objects.
var NamespaceTestDefault = "kubevirt-test-default"

func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func GetCurrentKv(virtClient kubecli.KubevirtClient) *k6sv1.KubeVirt {
	kvs := GetKvList(virtClient)
	gomega.Expect(kvs).To(gomega.HaveLen(1))
	return &kvs[0]
}

func GetKvList(virtClient kubecli.KubevirtClient) []k6sv1.KubeVirt {
	var kvListInstallNS *k6sv1.KubeVirtList
	var kvListDefaultNS *k6sv1.KubeVirtList
	var items []k6sv1.KubeVirt

	var err error

	gomega.Eventually(func() error {

		kvListInstallNS, err = virtClient.KubeVirt(flags.KubeVirtInstallNamespace).List(&k8smetav1.ListOptions{})

		return err
	}, 10*time.Second, 1*time.Second).ShouldNot(gomega.HaveOccurred())

	gomega.Eventually(func() error {

		kvListDefaultNS, err = virtClient.KubeVirt(NamespaceTestDefault).List(&k8smetav1.ListOptions{})

		return err
	}, 10*time.Second, 1*time.Second).ShouldNot(gomega.HaveOccurred())

	items = append(items, kvListInstallNS.Items...)
	items = append(items, kvListDefaultNS.Items...)

	return items
}
