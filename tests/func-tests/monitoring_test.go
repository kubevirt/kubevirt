package tests_test

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	openshiftroutev1 "github.com/openshift/api/route/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kvtutil "kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/kubevirt/tests/flags"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promApi "github.com/prometheus/client_golang/api"
	promApiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	promConfig "github.com/prometheus/common/config"
	promModel "github.com/prometheus/common/model"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
	"kubevirt.io/client-go/kubecli"
)

var runbookClient = http.DefaultClient

const (
	noneImpact float64 = iota
	warningImpact
	criticalImpact
)

var _ = Describe("[crit:high][vendor:cnv-qe@redhat.com][level:system]Monitoring", func() {
	flag.Parse()

	var err error
	var virtCli kubecli.KubevirtClient
	var promClient promApiv1.API
	var prometheusRule monitoringv1.PrometheusRule
	var initialOperatorHealthMetricValue float64

	runbookClient.Timeout = time.Second * 3

	BeforeEach(func() {
		virtCli, err = kubecli.GetKubevirtClient()
		kvtutil.PanicOnError(err)

		tests.SkipIfNotOpenShift(virtCli, "Prometheus")
		promClient = initializePromClient(getPrometheusUrl(virtCli), getAuthorizationTokenForPrometheus(virtCli))
		prometheusRule = getPrometheusRule(virtCli)

		initialOperatorHealthMetricValue = getMetricValue(promClient, "kubevirt_hyperconverged_operator_health_status")
	})

	It("Alert rules should have all the requried annotations", func() {
		for _, group := range prometheusRule.Spec.Groups {
			for _, rule := range group.Rules {
				if rule.Alert != "" {
					Expect(rule.Annotations).To(HaveKeyWithValue("summary", Not(BeEmpty())),
						fmt.Sprintf("%s summary is missing or empty", rule.Alert))
					Expect(rule.Annotations).To(HaveKeyWithValue("runbook_url", Not(BeEmpty())),
						fmt.Sprintf("%s runbook_url is missing or empty", rule.Alert))
					checkRunbookUrlAvailability(rule)
				}
			}
		}
	})

	It("Alert rules should have all the requried labels", func() {
		for _, group := range prometheusRule.Spec.Groups {
			for _, rule := range group.Rules {
				if rule.Alert != "" {
					Expect(rule.Labels).To(HaveKeyWithValue("severity", BeElementOf("info", "warning", "critical")),
						fmt.Sprintf("%s severity label is missing or not valid", rule.Alert))
					Expect(rule.Labels).To(HaveKeyWithValue("kubernetes_operator_part_of", "kubevirt"),
						fmt.Sprintf("%s kubernetes_operator_part_of label is missing or not valid", rule.Alert))
					Expect(rule.Labels).To(HaveKeyWithValue("kubernetes_operator_component", "hyperconverged-cluster-operator"),
						fmt.Sprintf("%s kubernetes_operator_component label is missing or not valid", rule.Alert))
					Expect(rule.Labels).To(HaveKeyWithValue("operator_health_impact", BeElementOf("none", "warning", "critical")),
						fmt.Sprintf("%s operator_health_impact label is missing or not valid", rule.Alert))
				}
			}
		}
	})

	It("KubevirtHyperconvergedClusterOperatorCRModification alert should fired when there is a modification on a CR", func() {
		By("Fetching kubevirt object")
		kubevirt, err := virtCli.KubeVirt(flags.KubeVirtInstallNamespace).Get("kubevirt-kubevirt-hyperconverged", &metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		By("Updating kubevirt object with a new label")
		kubevirt.Labels["test-label"] = "test-label-value"
		_, err = virtCli.KubeVirt(flags.KubeVirtInstallNamespace).Update(kubevirt)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() *promApiv1.Alert {
			alerts, err := promClient.Alerts(context.TODO())
			Expect(err).ShouldNot(HaveOccurred())
			alert := getAlertByName(alerts, "KubevirtHyperconvergedClusterOperatorCRModification")
			return alert
		}, 60*time.Second, time.Second).ShouldNot(BeNil())

		verifyOperatorHealthMetricValue(promClient, initialOperatorHealthMetricValue, warningImpact)
	})

	It("KubevirtHyperconvergedClusterOperatorUSModification alert should fired when there is an jsonpatch annotation to modify an operand CRs", func() {
		By("Updating HCO object with a new label")
		hco := getHCO(virtCli)

		hco.Annotations = map[string]string{
			"kubevirt.kubevirt.io/jsonpatch": `[{"op": "add", "path": "/spec/configuration/migrations", "value": {"allowPostCopy": true}}]`,
		}
		updateHCO(virtCli, hco)

		Eventually(func() *promApiv1.Alert {
			alerts, err := promClient.Alerts(context.TODO())
			Expect(err).ShouldNot(HaveOccurred())
			alert := getAlertByName(alerts, "KubevirtHyperconvergedClusterOperatorUSModification")
			return alert
		}, 60*time.Second, time.Second).ShouldNot(BeNil())
		verifyOperatorHealthMetricValue(promClient, initialOperatorHealthMetricValue, warningImpact)
	})

})

func getHCO(client kubecli.KubevirtClient) v1beta1.HyperConverged {
	s := scheme.Scheme
	_ = v1beta1.AddToScheme(s)
	s.AddKnownTypes(v1beta1.SchemeGroupVersion)

	var hco v1beta1.HyperConverged
	ExpectWithOffset(
		1,
		client.RestClient().Get().
			Resource("hyperconvergeds").
			Name("kubevirt-hyperconverged").
			Namespace(flags.KubeVirtInstallNamespace).
			AbsPath("/apis", v1beta1.SchemeGroupVersion.Group, v1beta1.SchemeGroupVersion.Version).
			Timeout(10*time.Second).
			Do(context.TODO()).Into(&hco),
	).To(Succeed())

	return hco
}

func updateHCO(client kubecli.KubevirtClient, hco v1beta1.HyperConverged) v1beta1.HyperConverged {
	hco.Kind = "HyperConverged"
	hco.APIVersion = v1beta1.SchemeGroupVersion.String()

	ExpectWithOffset(1, client.RestClient().Put().
		Resource("hyperconvergeds").
		Name("kubevirt-hyperconverged").
		Namespace(flags.KubeVirtInstallNamespace).
		AbsPath("/apis", v1beta1.SchemeGroupVersion.Group, v1beta1.SchemeGroupVersion.Version).
		Body(&hco).
		Timeout(10*time.Second).
		Do(context.TODO()).Into(&hco)).Should(Succeed())

	return hco
}

func getAlertByName(alerts promApiv1.AlertsResult, alertName string) *promApiv1.Alert {
	for _, alert := range alerts.Alerts {
		if string(alert.Labels["alertname"]) == alertName {
			return &alert
		}
	}
	return nil
}

func verifyOperatorHealthMetricValue(promClient promApiv1.API, initialOperatorHealthMetricValue, alertImpact float64) {
	systemHealthMetricValue := getMetricValue(promClient, "kubevirt_hco_system_health_status")
	operatorHealthMetricValue := getMetricValue(promClient, "kubevirt_hyperconverged_operator_health_status")

	expectedOperatorHealthMetricValue := math.Max(alertImpact, math.Max(systemHealthMetricValue, initialOperatorHealthMetricValue))
	ExpectWithOffset(1, operatorHealthMetricValue).To(Equal(expectedOperatorHealthMetricValue))
}

func getMetricValue(promClient promApiv1.API, metricName string) float64 {
	queryResult, _, err := promClient.Query(context.TODO(), metricName, time.Now())
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())

	resultVector := queryResult.(promModel.Vector)
	ExpectWithOffset(1, resultVector).To(HaveLen(1))

	metricValue, err := strconv.ParseFloat(resultVector[0].Value.String(), 64)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())

	return metricValue
}

func getPrometheusRule(client kubecli.KubevirtClient) monitoringv1.PrometheusRule {
	s := scheme.Scheme
	_ = monitoringv1.AddToScheme(s)
	s.AddKnownTypes(monitoringv1.SchemeGroupVersion)

	var prometheusRule monitoringv1.PrometheusRule

	ExpectWithOffset(1, client.RestClient().Get().
		Resource("prometheusrules").
		Name("kubevirt-hyperconverged-prometheus-rule").
		Namespace(flags.KubeVirtInstallNamespace).
		AbsPath("/apis", monitoringv1.SchemeGroupVersion.Group, monitoringv1.SchemeGroupVersion.Version).
		Timeout(10*time.Second).
		Do(context.TODO()).Into(&prometheusRule)).Should(Succeed())
	return prometheusRule
}

func checkRunbookUrlAvailability(rule monitoringv1.Rule) {
	resp, err := runbookClient.Head(rule.Annotations["runbook_url"])
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), fmt.Sprintf("%s runbook is not available", rule.Alert))
	ExpectWithOffset(1, resp.StatusCode).Should(Equal(http.StatusOK), fmt.Sprintf("%s runbook is not available", rule.Alert))
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
		treq, err := cli.CoreV1().ServiceAccounts("openshift-monitoring").CreateToken(
			context.TODO(),
			"prometheus-k8s",
			&authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					// Avoid specifying any audiences so that the token will be
					// issued for the default audience of the issuer.
				},
			},
			metav1.CreateOptions{},
		)
		if err != nil {
			return false
		}
		token = treq.Status.Token
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
