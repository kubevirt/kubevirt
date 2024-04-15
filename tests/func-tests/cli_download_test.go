package tests_test

import (
	"context"
	"crypto/tls"
	"flag"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	consolev1 "github.com/openshift/api/console/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"kubevirt.io/client-go/kubecli"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

var _ = Describe("[rfe_id:5100][crit:medium][vendor:cnv-qe@redhat.com][level:system]HyperConverged Cluster Operator should create ConsoleCliDownload objects", Label(tests.OpenshiftLabel), func() {
	flag.Parse()

	var cli kubecli.KubevirtClient
	BeforeEach(func() {
		tests.BeforeEach()
		virtCli, err := kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())

		cli, err = kubecli.GetKubevirtClientFromRESTConfig(virtCli.Config())
		Expect(err).ToNot(HaveOccurred())
		tests.FailIfNotOpenShift(cli, "ConsoleCliDownload")
	})

	It("[test_id:6956]should create ConsoleCliDownload objects with expected spec", Label("test_id:6956"), func() {
		By("Checking existence of ConsoleCliDownload")
		s := scheme.Scheme
		_ = consolev1.Install(s)
		s.AddKnownTypes(consolev1.GroupVersion)

		var ccd consolev1.ConsoleCLIDownload
		ExpectWithOffset(1, cli.RestClient().Get().
			Resource("consoleclidownloads").
			Name("virtctl-clidownloads-kubevirt-hyperconverged").
			AbsPath("/apis", consolev1.GroupVersion.Group, consolev1.GroupVersion.Version).
			Timeout(10*time.Second).
			Do(context.TODO()).Into(&ccd)).To(Succeed())

		ExpectWithOffset(1, ccd.Spec.Links).Should(HaveLen(6))

		for _, link := range ccd.Spec.Links {
			// virtctl for Windows for ARM 64 is still not shipped, avoid checking it
			// TODO: remove this once ready
			if !(strings.Contains(link.Href, "windows") && strings.Contains(link.Href, "arm64")) {
				By("Checking links. Link:" + link.Href)
				client := &http.Client{Transport: &http.Transport{
					// ssl of the route is irrelevant
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				}}
				resp, err := client.Get(link.Href)
				_ = resp.Body.Close()

				ExpectWithOffset(1, err).ToNot(HaveOccurred())
				ExpectWithOffset(1, resp).Should(HaveHTTPStatus(http.StatusOK))
			}
		}
	})
})
