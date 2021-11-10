package tests_test

import (
	"context"
	"flag"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	consolev1 "github.com/openshift/api/console/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("[rfe_id:5108][crit:medium][vendor:cnv-qe@redhat.com][level:system]Dashboard configmaps", func() {
	flag.Parse()

	BeforeEach(func() {
		tests.BeforeEach()
	})

	It("[test_id:5919]should create configmaps for OCP Dashboard", func() {
		virtCli, err := kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())

		client, err := kubecli.GetKubevirtClientFromRESTConfig(virtCli.Config())
		Expect(err).ToNot(HaveOccurred())

		checkExpectedConfigMaps(client)
	})

})

func checkExpectedConfigMaps(client kubecli.KubevirtClient) {
	By("Checking expected configmaps")
	s := scheme.Scheme
	_ = consolev1.Install(s)
	s.AddKnownTypes(consolev1.GroupVersion)

	items := tests.GetConfig().Dashboard.TestItems

	if len(items) == 0 {
		Skip("There is no test item for dashboard tests.")
	}

	for _, item := range items {
		cm, err := client.CoreV1().ConfigMaps(item.Namespace).Get(context.TODO(), item.Name, v1.GetOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		for _, key := range item.Keys {
			_, ok := cm.Data[key]
			ExpectWithOffset(1, ok).Should(BeTrue())
		}
	}

}
