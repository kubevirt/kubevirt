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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	openshiftroutev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promApi "github.com/prometheus/client_golang/api"
	promApiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	promConfig "github.com/prometheus/common/config"
	promModel "github.com/prometheus/common/model"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests/flags"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

var runbookClient = http.DefaultClient

const (
	noneImpact float64 = iota
	warningImpact
	criticalImpact
)

var _ = Describe("[crit:high][vendor:cnv-qe@redhat.com][level:system]Monitoring", Serial, Ordered, Label(tests.OpenshiftLabel), func() {
	flag.Parse()

	var err error
	var virtCli kubecli.KubevirtClient
	var promClient promApiv1.API
	var prometheusRule monitoringv1.PrometheusRule
	var initialOperatorHealthMetricValue float64
	ctx := context.TODO()

	runbookClient.Timeout = time.Second * 3

	BeforeEach(func() {
		virtCli, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())

		tests.FailIfNotOpenShift(virtCli, "Prometheus")
		promClient = initializePromClient(getPrometheusURL(virtCli), getAuthorizationTokenForPrometheus(virtCli))
		prometheusRule = getPrometheusRule(virtCli)

		initialOperatorHealthMetricValue = getMetricValue(promClient, "kubevirt_hyperconverged_operator_health_status")
	})

	It("Alert rules should have all the requried annotations", func() {
		for _, group := range prometheusRule.Spec.Groups {
			for _, rule := range group.Rules {
				if rule.Alert != "" {
					Expect(rule.Annotations).To(HaveKeyWithValue("summary", Not(BeEmpty())),
						"%s summary is missing or empty", rule.Alert)
					Expect(rule.Annotations).To(HaveKey("runbook_url"),
						"%s runbook_url is missing", rule.Alert)
					Expect(rule.Annotations).To(HaveKeyWithValue("runbook_url", HaveSuffix(rule.Alert)),
						"%s runbook_url is not equal to alert name", rule.Alert)
					checkRunbookURLAvailability(rule)
				}
			}
		}
	})

	It("Alert rules should have all the requried labels", func() {
		for _, group := range prometheusRule.Spec.Groups {
			for _, rule := range group.Rules {
				if rule.Alert != "" {
					Expect(rule.Labels).To(HaveKeyWithValue("severity", BeElementOf("info", "warning", "critical")),
						"%s severity label is missing or not valid", rule.Alert)
					Expect(rule.Labels).To(HaveKeyWithValue("kubernetes_operator_part_of", "kubevirt"),
						"%s kubernetes_operator_part_of label is missing or not valid", rule.Alert)
					Expect(rule.Labels).To(HaveKeyWithValue("kubernetes_operator_component", "hyperconverged-cluster-operator"),
						"%s kubernetes_operator_component label is missing or not valid", rule.Alert)
					Expect(rule.Labels).To(HaveKeyWithValue("operator_health_impact", BeElementOf("none", "warning", "critical")),
						"%s operator_health_impact label is missing or not valid", rule.Alert)
				}
			}
		}
	})

	It("KubeVirtCRModified alert should fired when there is a modification on a CR", func() {
		By("Patching kubevirt object")
		const (
			fakeFG = "fake-fg-for-testing"
			query  = `kubevirt_hco_out_of_band_modifications_total{component_name="kubevirt/kubevirt-kubevirt-hyperconverged"}`
		)

		var valueBefore float64
		Eventually(func(g Gomega) {
			valueBefore = getMetricValue(promClient, query)
		}).WithTimeout(10 * time.Second).WithPolling(500 * time.Millisecond).Should(Succeed())

		Eventually(func(g Gomega) []string {
			patch := []byte(fmt.Sprintf(`[{"op": "add", "path": "/spec/configuration/developerConfiguration/featureGates/-", "value": %q}]`, fakeFG))
			kv, err := virtCli.KubeVirt(flags.KubeVirtInstallNamespace).Patch(ctx, "kubevirt-kubevirt-hyperconverged", types.JSONPatchType, patch, metav1.PatchOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			return kv.Spec.Configuration.DeveloperConfiguration.FeatureGates
		}).WithTimeout(10 * time.Second).
			WithPolling(100 * time.Millisecond).
			Should(ContainElement(fakeFG))

		Eventually(func(g Gomega) float64 {
			return getMetricValue(promClient, query)
		}).WithTimeout(60 * time.Second).WithPolling(time.Second).Should(Equal(valueBefore + 1))

		Eventually(func() *promApiv1.Alert {
			alerts, err := promClient.Alerts(context.TODO())
			Expect(err).ToNot(HaveOccurred())
			alert := getAlertByName(alerts, "KubeVirtCRModified")
			return alert
		}).WithTimeout(60 * time.Second).WithPolling(time.Second).ShouldNot(BeNil())

		verifyOperatorHealthMetricValue(promClient, initialOperatorHealthMetricValue, warningImpact)
	})

	It("UnsupportedHCOModification alert should fired when there is an jsonpatch annotation to modify an operand CRs", func() {
		By("Updating HCO object with a new label")
		hco := tests.GetHCO(ctx, virtCli)

		hco.Annotations = map[string]string{
			"kubevirt.kubevirt.io/jsonpatch": `[{"op": "add", "path": "/spec/configuration/migrations", "value": {"allowPostCopy": true}}]`,
		}
		tests.UpdateHCORetry(ctx, virtCli, hco)

		Eventually(func() *promApiv1.Alert {
			alerts, err := promClient.Alerts(context.TODO())
			Expect(err).ToNot(HaveOccurred())
			alert := getAlertByName(alerts, "UnsupportedHCOModification")
			return alert
		}, 60*time.Second, time.Second).ShouldNot(BeNil())
		verifyOperatorHealthMetricValue(promClient, initialOperatorHealthMetricValue, warningImpact)
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

func verifyOperatorHealthMetricValue(promClient promApiv1.API, initialOperatorHealthMetricValue, alertImpact float64) {
	Eventually(func(g Gomega) {
		if alertImpact >= initialOperatorHealthMetricValue {
			systemHealthMetricValue := getMetricValue(promClient, "kubevirt_hco_system_health_status")
			operatorHealthMetricValue := getMetricValue(promClient, "kubevirt_hyperconverged_operator_health_status")
			expectedOperatorHealthMetricValue := math.Max(alertImpact, systemHealthMetricValue)

			g.Expect(operatorHealthMetricValue).To(Equal(expectedOperatorHealthMetricValue),
				"kubevirt_hyperconverged_operator_health_status value is %f, but its expected value is %f, "+
					"while kubevirt_hco_system_health_status value is %f.",
				operatorHealthMetricValue, expectedOperatorHealthMetricValue, systemHealthMetricValue)
		}

	}, 60*time.Second, 5*time.Second).Should(Succeed())
}

func getMetricValue(promClient promApiv1.API, metricName string) float64 {
	queryResult, _, err := promClient.Query(context.TODO(), metricName, time.Now())
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())

	resultVector := queryResult.(promModel.Vector)
	if len(resultVector) == 0 {
		return 0
	}

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

func checkRunbookURLAvailability(rule monitoringv1.Rule) {
	resp, err := runbookClient.Head(rule.Annotations["runbook_url"])
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), fmt.Sprintf("%s runbook is not available", rule.Alert))
	ExpectWithOffset(1, resp.StatusCode).Should(Equal(http.StatusOK), fmt.Sprintf("%s runbook is not available", rule.Alert))
}

func initializePromClient(prometheusURL string, token string) promApiv1.API {
	defaultRoundTripper := promApi.DefaultRoundTripper
	tripper := defaultRoundTripper.(*http.Transport)
	tripper.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	c, err := promApi.NewClient(promApi.Config{
		Address:      prometheusURL,
		RoundTripper: promConfig.NewAuthorizationCredentialsRoundTripper("Bearer", promConfig.Secret(token), defaultRoundTripper),
	})

	Expect(err).ToNot(HaveOccurred())

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

func getPrometheusURL(cli kubecli.KubevirtClient) string {
	s := scheme.Scheme
	_ = openshiftroutev1.Install(s)
	s.AddKnownTypes(openshiftroutev1.GroupVersion)

	var route openshiftroutev1.Route

	Eventually(func() error {
		return cli.RestClient().Get().
			Resource("routes").
			Name("prometheus-k8s").
			Namespace("openshift-monitoring").
			AbsPath("/apis", openshiftroutev1.GroupVersion.Group, openshiftroutev1.GroupVersion.Version).
			Timeout(10 * time.Second).
			Do(context.TODO()).Into(&route)
	}).WithTimeout(2 * time.Minute).
		WithPolling(15 * time.Second). // longer than the request timeout
		Should(Succeed())

	return fmt.Sprintf("https://%s", route.Spec.Host)
}
