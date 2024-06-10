package tests_test

import (
	"context"
	"crypto/tls"
	"flag"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	consolev1 "github.com/openshift/api/console/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

var _ = Describe("[rfe_id:5100][crit:medium][vendor:cnv-qe@redhat.com][level:system]HyperConverged Cluster Operator should create ConsoleCliDownload objects", Label(tests.OpenshiftLabel, "ConsoleCliDownload"), func() {
	flag.Parse()

	var (
		cli client.Client
		ctx context.Context
	)

	BeforeEach(func() {
		tests.BeforeEach()
		cfg, err := config.GetConfig()
		Expect(err).ToNot(HaveOccurred())

		s := scheme.Scheme
		Expect(consolev1.AddToScheme(s)).To(Succeed())
		cli, err = client.New(cfg, client.Options{Scheme: s})
		Expect(err).ToNot(HaveOccurred())

		ctx = context.Background()
		tests.FailIfNotOpenShift(ctx, cli, "ConsoleCliDownload")
	})

	It("[test_id:6956]should create ConsoleCliDownload objects with expected spec", Label("test_id:6956"), func() {
		By("Checking existence of ConsoleCliDownload")

		ccd := &consolev1.ConsoleCLIDownload{
			ObjectMeta: metav1.ObjectMeta{
				Name: "virtctl-clidownloads-kubevirt-hyperconverged",
			},
		}

		Expect(cli.Get(ctx, client.ObjectKeyFromObject(ccd), ccd)).To(Succeed())

		Expect(ccd.Spec.Links).To(HaveLen(6))

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
