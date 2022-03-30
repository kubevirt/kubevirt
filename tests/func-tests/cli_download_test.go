package tests_test

import (
	"context"
	"crypto/tls"
	"flag"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	consolev1 "github.com/openshift/api/console/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("[rfe_id:5100][crit:medium][vendor:cnv-qe@redhat.com][level:system]HyperConverged Cluster Operator should create ConsoleCliDownload objects", func() {
	flag.Parse()

	BeforeEach(func() {
		tests.BeforeEach()
	})

	It("[test_id:6956]should create ConsoleCliDownload objects with expected spec", func() {
		virtCli, err := kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())

		client, err := kubecli.GetKubevirtClientFromRESTConfig(virtCli.Config())
		Expect(err).ToNot(HaveOccurred())

		skipIfConsoleCliDownloadsCrdDoesNotExist(virtCli)

		checkConsoleCliDownloadSpec(client)
	})

})

func skipIfConsoleCliDownloadsCrdDoesNotExist(cli kubecli.KubevirtClient) {
	By("Checking ConsoleCLIDownload CRD exists or not")

	_, err := cli.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "consoleclidownloads.console.openshift.io", metav1.GetOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		Skip("ConsoleCLIDownload CRD does not exist")
	}
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

func checkConsoleCliDownloadSpec(client kubecli.KubevirtClient) {
	By("Checking existence of ConsoleCliDownload")
	s := scheme.Scheme
	_ = consolev1.Install(s)
	s.AddKnownTypes(consolev1.GroupVersion)

	var ccd consolev1.ConsoleCLIDownload
	err := client.RestClient().Get().
		Resource("consoleclidownloads").
		Name("virtctl-clidownloads-kubevirt-hyperconverged").
		AbsPath("/apis", consolev1.GroupVersion.Group, consolev1.GroupVersion.Version).
		Timeout(10 * time.Second).
		Do(context.TODO()).Into(&ccd)

	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, len(ccd.Spec.Links)).Should(Equal(3))

	for _, link := range ccd.Spec.Links {
		By("Checking links. Link:" + link.Href)
		client := &http.Client{Transport: &http.Transport{
			// ssl of the route is irrelevant
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}}
		resp, err := client.Get(link.Href)
		_ = resp.Body.Close()

		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		ExpectWithOffset(1, resp.StatusCode).Should(Equal(200))

	}
}
