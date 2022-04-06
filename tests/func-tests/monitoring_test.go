package tests_test

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	openshiftroutev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kvtutil "kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/kubevirt/tests/flags"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promApi "github.com/prometheus/client_golang/api"
	promApiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	promConfig "github.com/prometheus/common/config"
	"k8s.io/client-go/kubernetes/scheme"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
	"kubevirt.io/client-go/kubecli"
)

var runbookClient = http.DefaultClient

var _ = Describe("[crit:high][vendor:cnv-qe@redhat.com][level:system]Monitoring", func() {
	flag.Parse()

	var err error
	var virtCli kubecli.KubevirtClient
	var promClient promApiv1.API

	runbookClient.Timeout = time.Second * 3

	BeforeEach(func() {
		virtCli, err = kubecli.GetKubevirtClient()
		kvtutil.PanicOnError(err)

		tests.SkipIfNotOpenShift(virtCli)
		promClient = initializePromClient(getPrometheusUrl(virtCli), getAuthorizationTokenForPrometheus(virtCli))
	})

	It("PrometheusRule Objects should have available runbook URLs", func() {
		prometheusRule := getPrometheusRule(virtCli)
		checkRunbookUrls(prometheusRule)
	})

	It("KubevirtHyperconvergedClusterOperatorCRModification alert should fired when there is a modification on a CR", func() {
		By("Fetching kubevirt object")
		kubevirt, err := virtCli.KubeVirt(flags.KubeVirtInstallNamespace).Get("kubevirt-kubevirt-hyperconverged", &metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		By("Updating kubevirt object with a new label")
		kubevirt.Labels["test-label"] = "test-label-value"
		kubevirt, err = virtCli.KubeVirt(flags.KubeVirtInstallNamespace).Update(kubevirt)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() *promApiv1.Alert {
			alerts, err := promClient.Alerts(context.TODO())
			Expect(err).ShouldNot(HaveOccurred())
			alert := getAlertByName(alerts, "KubevirtHyperconvergedClusterOperatorCRModification")
			return alert
		}, 60*time.Second, time.Second).ShouldNot(BeNil())

	})

})

func getAlertByName(alerts promApiv1.AlertsResult, alertName string) *promApiv1.Alert {
	for _, alert := range alerts.Alerts {
		if string(alert.Labels["alertname"]) == alertName {
			return &alert
		}
	}
	return nil
}

func checkRunbookUrls(prometheusRule monitoringv1.PrometheusRule) {
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

func getPrometheusRule(client kubecli.KubevirtClient) monitoringv1.PrometheusRule {
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
	return prometheusRule
}

func checkAvailabilityOfUrl(url string) {
	resp, err := runbookClient.Head(url)
	ExpectWithOffset(2, err).ToNot(HaveOccurred())
	ExpectWithOffset(2, resp.StatusCode).Should(Equal(200))
}

func initializePromClient(prometheusUrl string, token string) promApiv1.API {
	defaultRoundTripper := promApi.DefaultRoundTripper
	tripper := defaultRoundTripper.(*http.Transport)
	tripper.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	c, err := promApi.NewClient(promApi.Config{
		Address:      prometheusUrl,
		RoundTripper: promConfig.NewAuthorizationCredentialsRoundTripper("Bearer", promConfig.Secret(token), defaultRoundTripper),
	})

	kvtutil.PanicOnError(err)

	promClient := promApiv1.NewAPI(c)
	return promClient
}

func getAuthorizationTokenForPrometheus(cli kubecli.KubevirtClient) string {
	var token string
	Eventually(func() bool {
		var secretName string
		sa, err := cli.CoreV1().ServiceAccounts("openshift-monitoring").Get(context.TODO(), "prometheus-k8s", metav1.GetOptions{})
		if err != nil {
			return false
		}
		for _, secret := range sa.Secrets {
			if strings.HasPrefix(secret.Name, "prometheus-k8s-token") {
				secretName = secret.Name
			}
		}
		secret, err := cli.CoreV1().Secrets("openshift-monitoring").Get(context.TODO(), secretName, metav1.GetOptions{})
		if err != nil {
			return false
		}
		if _, ok := secret.Data["token"]; !ok {
			return false
		}
		token = string(secret.Data["token"])
		return true
	}, 10*time.Second, time.Second).Should(BeTrue())
	return token
}

func getPrometheusUrl(cli kubecli.KubevirtClient) string {
	s := scheme.Scheme
	_ = openshiftroutev1.Install(s)
	s.AddKnownTypes(openshiftroutev1.GroupVersion)

	var route openshiftroutev1.Route

	err := cli.RestClient().Get().
		Resource("routes").
		Name("prometheus-k8s").
		Namespace("openshift-monitoring").
		AbsPath("/apis", openshiftroutev1.GroupVersion.Group, openshiftroutev1.GroupVersion.Version).
		Timeout(10 * time.Second).
		Do(context.TODO()).Into(&route)

	kvtutil.PanicOnError(err)

	return fmt.Sprintf("https://%s", route.Spec.Host)
}
