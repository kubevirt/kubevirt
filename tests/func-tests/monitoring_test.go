package tests_test

import (
	"context"
	"flag"
	"net/http"
	"time"

	"kubevirt.io/kubevirt/tests/flags"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("[crit:high][vendor:cnv-qe@redhat.com][level:system]Prometheus Alerts", func() {
	flag.Parse()

	BeforeEach(func() {
		tests.BeforeEach()
	})

	It("should have available runbook URLs", func() {
		virtCli, err := kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())

		client, err := kubecli.GetKubevirtClientFromRESTConfig(virtCli.Config())
		Expect(err).ToNot(HaveOccurred())

		skipIfPrometheusRuleDoesNotExist(virtCli)

		checkRunbookUrls(client)
	})

})

func skipIfPrometheusRuleDoesNotExist(cli kubecli.KubevirtClient) {
	By("Checking PrometheusRule CRD exists or not")

	_, err := cli.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "prometheusrules.monitoring.coreos.com", metav1.GetOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		Skip("PrometheusRule CRD does not exist")
	}
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())
}

func checkRunbookUrls(client kubecli.KubevirtClient) {
	By("Checking expected prometheusrule objects")
	s := scheme.Scheme
	_ = monitoringv1.AddToScheme(s)
	s.AddKnownTypes(monitoringv1.SchemeGroupVersion)

	var prometheusRule monitoringv1.PrometheusRule

	err := client.RestClient().Get().
		Resource("prometheusrules").
		Name("kubevirt-hyperconverged-prometheus-rule").
		Namespace(flags.KubeVirtInstallNamespace).
		AbsPath("/apis", monitoringv1.SchemeGroupVersion.Group, monitoringv1.SchemeGroupVersion.Version).
		Timeout(10 * time.Second).
		Do(context.TODO()).Into(&prometheusRule)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())

	for _, group := range prometheusRule.Spec.Groups {
		for _, rule := range group.Rules {
			if len(rule.Alert) > 0 {
				ExpectWithOffset(1, rule.Annotations).ToNot(BeNil())
				url, ok := rule.Annotations["runbook_url"]
				ExpectWithOffset(1, ok).To(BeTrue())
				checkAvailabilityOfUrl(url)
			}
		}
	}

}

func checkAvailabilityOfUrl(url string) {
	resp, err := http.Get(url)
	ExpectWithOffset(2, err).ToNot(HaveOccurred())
	ExpectWithOffset(2, resp.StatusCode).Should(Equal(200))
}
