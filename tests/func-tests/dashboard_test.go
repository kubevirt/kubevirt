package tests_test

import (
	"context"
	"flag"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	consolev1 "github.com/openshift/api/console/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"kubevirt.io/client-go/kubecli"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

var _ = Describe("[rfe_id:5108][crit:medium][vendor:cnv-qe@redhat.com][level:system]Dashboard configmaps", Label(tests.OpenshiftLabel), func() {
	flag.Parse()

	var cli kubecli.KubevirtClient

	BeforeEach(func() {
		tests.BeforeEach()

		virtCli, err := kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())

		tests.FailIfNotOpenShift(virtCli, "Dashboard configmaps")

		cli, err = kubecli.GetKubevirtClientFromRESTConfig(virtCli.Config())
		Expect(err).ToNot(HaveOccurred())
	})

	It("[test_id:5919]should create configmaps for OCP Dashboard", Label("test_id:5919"), func() {
		By("Checking expected configmaps")
		s := scheme.Scheme
		_ = consolev1.Install(s)
		s.AddKnownTypes(consolev1.GroupVersion)

		items := tests.GetConfig().Dashboard.TestItems

		if len(items) == 0 {
			GinkgoLogr.Info("There is no test item for dashboard tests.")
		}

		for _, item := range items {
			cm, err := cli.CoreV1().ConfigMaps(item.Namespace).Get(context.TODO(), item.Name, v1.GetOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			for _, key := range item.Keys {
				_, ok := cm.Data[key]
				ExpectWithOffset(1, ok).Should(BeTrue())
			}
		}
	})
})
