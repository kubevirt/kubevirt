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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubevirtcorev1 "kubevirt.io/api/core/v1"

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

	var (
		cli                              client.Client
		cliSet                           *kubernetes.Clientset
		restClient                       rest.Interface
		promClient                       promApiv1.API
		prometheusRule                   monitoringv1.PrometheusRule
		initialOperatorHealthMetricValue float64
	)

	runbookClient.Timeout = time.Second * 3

	BeforeEach(func(ctx context.Context) {
		cli = tests.GetControllerRuntimeClient()
		cliSet = tests.GetK8sClientSet()
		restClient = cliSet.RESTClient()

		tests.FailIfNotOpenShift(ctx, cli, "Prometheus")
		promClient = initializePromClient(getPrometheusURL(ctx, restClient), getAuthorizationTokenForPrometheus(ctx, cliSet))
		prometheusRule = getPrometheusRule(ctx, restClient)

		initialOperatorHealthMetricValue = getMetricValue(ctx, promClient, "kubevirt_hyperconverged_operator_health_status")
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

	It("KubeVirtCRModified alert should fired when there is a modification on a CR", func(ctx context.Context) {
		By("Patching kubevirt object")
		const (
			fakeFG = "fake-fg-for-testing"
			query  = `kubevirt_hco_out_of_band_modifications_total{component_name="kubevirt/kubevirt-kubevirt-hyperconverged"}`
		)

		var valueBefore float64
		Eventually(func(g Gomega, ctx context.Context) {
			valueBefore = getMetricValue(ctx, promClient, query)
		}).WithTimeout(10 * time.Second).WithPolling(500 * time.Millisecond).WithContext(ctx).Should(Succeed())

		patchBytes := []byte(fmt.Sprintf(`[{"op": "add", "path": "/spec/configuration/developerConfiguration/featureGates/-", "value": %q}]`, fakeFG))
		patch := client.RawPatch(types.JSONPatchType, patchBytes)

		retries := float64(0)
		Eventually(func(g Gomega, ctx context.Context) []string {
			kv := &kubevirtcorev1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubevirt-kubevirt-hyperconverged",
					Namespace: tests.InstallNamespace,
				},
			}

			g.Expect(cli.Patch(ctx, kv, patch)).To(Succeed())
			retries++
			g.Expect(cli.Get(ctx, client.ObjectKeyFromObject(kv), kv)).To(Succeed())

			return kv.Spec.Configuration.DeveloperConfiguration.FeatureGates
		}).WithTimeout(10 * time.Second).
			WithPolling(100 * time.Millisecond).
			WithContext(ctx).
			Should(ContainElement(fakeFG))

		Expect(retries).To(BeNumerically(">", 0))

		Eventually(func(g Gomega, ctx context.Context) float64 {
			return getMetricValue(ctx, promClient, query)
		}).WithTimeout(60*time.Second).
			WithPolling(time.Second).
			WithContext(ctx).
			Should(
				Equal(valueBefore+retries),
				"expected different counter value; valueBefore: %0.2f; retries: %0.2f", valueBefore, retries,
			)

		Eventually(func(ctx context.Context) *promApiv1.Alert {
			alerts, err := promClient.Alerts(ctx)
			Expect(err).ToNot(HaveOccurred())
			alert := getAlertByName(alerts, "KubeVirtCRModified")
			return alert
		}).WithTimeout(60 * time.Second).WithPolling(time.Second).WithContext(ctx).ShouldNot(BeNil())

		verifyOperatorHealthMetricValue(ctx, promClient, initialOperatorHealthMetricValue, warningImpact)
	})

	It("UnsupportedHCOModification alert should fired when there is an jsonpatch annotation to modify an operand CRs", func(ctx context.Context) {
		By("Updating HCO object with a new label")
		hco := tests.GetHCO(ctx, cli)

		hco.Annotations = map[string]string{
			"kubevirt.kubevirt.io/jsonpatch": `[{"op": "add", "path": "/spec/configuration/migrations", "value": {"allowPostCopy": true}}]`,
		}
		tests.UpdateHCORetry(ctx, cli, hco)

		Eventually(func(ctx context.Context) *promApiv1.Alert {
			alerts, err := promClient.Alerts(ctx)
			Expect(err).ToNot(HaveOccurred())
			alert := getAlertByName(alerts, "UnsupportedHCOModification")
			return alert
		}).WithTimeout(60 * time.Second).WithPolling(time.Second).WithContext(ctx).ShouldNot(BeNil())
		verifyOperatorHealthMetricValue(ctx, promClient, initialOperatorHealthMetricValue, warningImpact)
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

func verifyOperatorHealthMetricValue(ctx context.Context, promClient promApiv1.API, initialOperatorHealthMetricValue, alertImpact float64) {
	Eventually(func(g Gomega, ctx context.Context) {
		if alertImpact >= initialOperatorHealthMetricValue {
			systemHealthMetricValue := getMetricValue(ctx, promClient, "kubevirt_hco_system_health_status")
			operatorHealthMetricValue := getMetricValue(ctx, promClient, "kubevirt_hyperconverged_operator_health_status")
			expectedOperatorHealthMetricValue := math.Max(alertImpact, systemHealthMetricValue)

			g.Expect(operatorHealthMetricValue).To(Equal(expectedOperatorHealthMetricValue),
				"kubevirt_hyperconverged_operator_health_status value is %f, but its expected value is %f, "+
					"while kubevirt_hco_system_health_status value is %f.",
				operatorHealthMetricValue, expectedOperatorHealthMetricValue, systemHealthMetricValue)
		}

	}).WithTimeout(60 * time.Second).WithPolling(5 * time.Second).WithContext(ctx).Should(Succeed())
}

func getMetricValue(ctx context.Context, promClient promApiv1.API, metricName string) float64 {
	queryResult, _, err := promClient.Query(ctx, metricName, time.Now())
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

func getPrometheusRule(ctx context.Context, cli rest.Interface) monitoringv1.PrometheusRule {
	var prometheusRule monitoringv1.PrometheusRule

	ExpectWithOffset(1, cli.Get().
		Resource("prometheusrules").
		Name("kubevirt-hyperconverged-prometheus-rule").
		Namespace(tests.InstallNamespace).
		AbsPath("/apis", monitoringv1.SchemeGroupVersion.Group, monitoringv1.SchemeGroupVersion.Version).
		Timeout(10*time.Second).
		Do(ctx).Into(&prometheusRule)).Should(Succeed())
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

func getAuthorizationTokenForPrometheus(ctx context.Context, cli *kubernetes.Clientset) string {
	var token string
	Eventually(func(ctx context.Context) bool {
		treq, err := cli.CoreV1().ServiceAccounts("openshift-monitoring").CreateToken(
			ctx,
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
	}).WithTimeout(10 * time.Second).WithPolling(time.Second).WithContext(ctx).Should(BeTrue())
	return token
}

func getPrometheusURL(ctx context.Context, cli rest.Interface) string {
	s := scheme.Scheme
	_ = openshiftroutev1.Install(s)
	s.AddKnownTypes(openshiftroutev1.GroupVersion)

	var route openshiftroutev1.Route

	Eventually(func(ctx context.Context) error {
		return cli.Get().
			Resource("routes").
			Name("prometheus-k8s").
			Namespace("openshift-monitoring").
			AbsPath("/apis", openshiftroutev1.GroupVersion.Group, openshiftroutev1.GroupVersion.Version).
			Timeout(10 * time.Second).
			Do(ctx).Into(&route)
	}).WithTimeout(2 * time.Minute).
		WithPolling(15 * time.Second). // longer than the request timeout
		WithContext(ctx).
		Should(Succeed())

	return fmt.Sprintf("https://%s", route.Spec.Host)
}
