package tests_test

import (
	"flag"
	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	consolev1 "github.com/openshift/api/console/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"kubevirt.io/client-go/kubecli"
	"time"
)

var defaultItems = []tests.QuickStartTestItem{{Name: "test-quick-start", DisplayName: "Test Quickstart Tour"}}

var _ = Describe("[rfe_id:5882][crit:high][vendor:cnv-qe@redhat.com][level:system]ConsoleQuickStart objects", func() {
	flag.Parse()

	BeforeEach(func() {
		tests.BeforeEach()
	})

	It("[test_id:5883]should create ConsoleQuickStart objects", func() {
		virtCli, err := kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())

		client, err := kubecli.GetKubevirtClientFromRESTConfig(virtCli.Config())
		Expect(err).ToNot(HaveOccurred())

		skipIfQuickStartCrdDoesNotExist(virtCli)

		checkExpectedQuickStarts(client)
	})

})

func skipIfQuickStartCrdDoesNotExist(cli kubecli.KubevirtClient) {
	By("Checking ConsoleQuickStarts CRD exists or not")

	_, err := cli.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Get("consolequickstarts.console.openshift.io", metav1.GetOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		Skip("ConsoleQuickStarts CRD does not exist")
	}
	ExpectWithOffset(1, err).Should(BeNil())
}

func checkExpectedQuickStarts(client kubecli.KubevirtClient) {
	By("Checking expected quickstart objects")
	s := scheme.Scheme
	_ = consolev1.Install(s)
	s.AddKnownTypes(consolev1.GroupVersion)

	items := defaultItems
	if tests.GetConfig().QuickStart.TestItems != nil {
		items = tests.GetConfig().QuickStart.TestItems
	}
	for _, qs := range items {
		// use a fresh object for each loop. get requests only override non-empty fields
		var cqs consolev1.ConsoleQuickStart
		err := client.RestClient().Get().
			Resource("consolequickstarts").
			Name(qs.Name).
			AbsPath("/apis", consolev1.GroupVersion.Group, consolev1.GroupVersion.Version).
			Timeout(10 * time.Second).
			Do().Into(&cqs)

		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		ExpectWithOffset(1, cqs.Spec.DisplayName).Should(Equal(qs.DisplayName))
	}

}
