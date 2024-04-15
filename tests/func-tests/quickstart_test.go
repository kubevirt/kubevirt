package tests_test

import (
	"context"
	"flag"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	consolev1 "github.com/openshift/api/console/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"kubevirt.io/client-go/kubecli"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

var _ = Describe("[rfe_id:5882][crit:high][vendor:cnv-qe@redhat.com][level:system]ConsoleQuickStart objects", Label(tests.OpenshiftLabel), func() {
	flag.Parse()

	var cli kubecli.KubevirtClient
	BeforeEach(func() {
		tests.BeforeEach()
		virtCli, err := kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())

		tests.FailIfNotOpenShift(virtCli, "quickstart")

		cli, err = kubecli.GetKubevirtClientFromRESTConfig(virtCli.Config())
		Expect(err).ToNot(HaveOccurred())
	})

	It("[test_id:5883]should create ConsoleQuickStart objects", Label("test_id:5883"), func() {
		By("Checking expected quickstart objects")
		s := scheme.Scheme
		_ = consolev1.Install(s)
		s.AddKnownTypes(consolev1.GroupVersion)

		items := tests.GetConfig().QuickStart.TestItems

		if len(items) == 0 {
			GinkgoLogr.Info("There is no quickstart test item for quickstart tests.")
		}

		for _, qs := range items {
			// use a fresh object for each loop. get requests only override non-empty fields
			var cqs consolev1.ConsoleQuickStart
			ExpectWithOffset(1, cli.RestClient().Get().
				Resource("consolequickstarts").
				Name(qs.Name).
				AbsPath("/apis", consolev1.GroupVersion.Group, consolev1.GroupVersion.Version).
				Timeout(10*time.Second).
				Do(context.TODO()).Into(&cqs)).To(Succeed())

			ExpectWithOffset(1, cqs.Spec.DisplayName).Should(Equal(qs.DisplayName))
		}
	})

})
