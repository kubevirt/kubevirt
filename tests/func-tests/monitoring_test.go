package tests_test

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	openshiftroutev1 "github.com/openshift/api/route/v1"
	deschedulerv1 "github.com/openshift/cluster-kube-descheduler-operator/pkg/apis/descheduler/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promApi "github.com/prometheus/client_golang/api"
	promApiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	promConfig "github.com/prometheus/common/config"
	promModel "github.com/prometheus/common/model"
	authenticationv1 "k8s.io/api/authentication/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubevirtcorev1 "kubevirt.io/api/core/v1"

	hcoalerts "github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/hyperconverged/rules/alerts"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

var runbookClient = http.DefaultClient

const (
	prometheousTimeout = 5 * time.Minute
	prometheousPolling = 10 * time.Second
)

var _ = Describe("[crit:high][vendor:cnv-qe@redhat.com][level:system]Monitoring", Serial, Ordered, Label(tests.OpenshiftLabel, "monitoring"), func() {
	flag.Parse()

	var (
		cli            client.Client
		cliSet         *kubernetes.Clientset
		restClient     rest.Interface
		promClient     promApiv1.API
		prometheusRule monitoringv1.PrometheusRule
		hcoClient      *tests.HCOPrometheusClient
	)

	runbookClient.Timeout = time.Second * 3

	BeforeEach(func(ctx context.Context) {
		cli = tests.GetControllerRuntimeClient()
		cliSet = tests.GetK8sClientSet()
		restClient = cliSet.RESTClient()

		tests.FailIfNotOpenShift(ctx, cli, "Prometheus")
		promClient = initializePromClient(getPrometheusURL(ctx, restClient), getAuthorizationTokenForPrometheus(ctx, cliSet))
		prometheusRule = getPrometheusRule(ctx, restClient)

		var err error
		hcoClient, err = tests.GetHCOPrometheusClient(ctx, cli)
		Expect(err).NotTo(HaveOccurred())

		Expect(err).NotTo(HaveOccurred())
	})

	It("Alert rules should have all the required annotations", func() {
		for _, group := range prometheusRule.Spec.Groups {
			for _, rule := range group.Rules {
				if rule.Alert != "" {
					Expect(rule.Annotations).To(HaveKeyWithValue("summary", Not(BeEmpty())),
						"%s summary is missing or empty", rule.Alert)
					Expect(rule.Annotations).To(HaveKey("runbook_url"),
						"%s runbook_url is missing", rule.Alert)
					Expect(rule.Annotations).To(HaveKeyWithValue("runbook_url", ContainSubstring(rule.Alert)),
						"%s runbook_url doesn't include alert name", rule.Alert)
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

	It("KubeVirtCRModified alert should fired when there is a modification on a CR", Serial, func(ctx context.Context) {

		const (
			query     = `kubevirt_hco_out_of_band_modifications_total{component_name="kubevirt/kubevirt-kubevirt-hyperconverged"}`
			jsonPatch = `[{"op": "add", "path": "/spec/configuration/developerConfiguration/featureGates/-", "value": "fake-fg-for-testing"}]`
		)

		By(fmt.Sprintf("Reading the `%s` metric from HCO prometheus endpoint", query))
		var valueBefore float64
		Eventually(func(g Gomega, ctx context.Context) {
			var err error
			valueBefore, err = hcoClient.GetHCOMetric(ctx, query)
			g.Expect(err).NotTo(HaveOccurred())
		}).WithTimeout(10 * time.Second).WithPolling(500 * time.Millisecond).WithContext(ctx).Should(Succeed())
		GinkgoWriter.Printf("The metric value before the test is: %0.2f\n", valueBefore)

		By("Patching kubevirt object")
		patch := client.RawPatch(types.JSONPatchType, []byte(jsonPatch))

		kv := &kubevirtcorev1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt-kubevirt-hyperconverged",
				Namespace: tests.InstallNamespace,
			},
		}

		Expect(cli.Patch(ctx, kv, patch)).To(Succeed())

		By("checking that the HCO metric was increased by 1")
		Eventually(func(g Gomega, ctx context.Context) float64 {
			valueAfter, err := hcoClient.GetHCOMetric(ctx, query)
			g.Expect(err).NotTo(HaveOccurred())
			return valueAfter
		}).
			WithTimeout(60*time.Second).
			WithPolling(time.Second).
			WithContext(ctx).
			Should(
				Equal(valueBefore+float64(1)),
				"expected different counter value; value before: %0.2f; expected value: %0.2f", valueBefore, valueBefore+float64(1),
			)

		By("Checking the alert")
		Eventually(func(ctx context.Context) *promApiv1.Alert {
			alerts, err := promClient.Alerts(ctx)
			Expect(err).ToNot(HaveOccurred())
			alert := getAlertByName(alerts, "KubeVirtCRModified")
			return alert
		}).WithTimeout(prometheousTimeout).WithPolling(prometheousPolling).WithContext(ctx).ShouldNot(BeNil())
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
		}).WithTimeout(prometheousTimeout).WithPolling(prometheousPolling).WithContext(ctx).ShouldNot(BeNil())
	})

	Describe("KubeDescheduler", Serial, Ordered, Label(tests.OpenshiftLabel, "monitoring"), func() {

		var (
			initialDescheduler = &deschedulerv1.KubeDescheduler{}
		)

		BeforeAll(func(ctx context.Context) {
			crd := &apiextensionsv1.CustomResourceDefinition{}
			crdKey := client.ObjectKey{Name: hcoutil.DeschedulerCRDName}
			key := client.ObjectKey{Namespace: hcoutil.DeschedulerNamespace, Name: hcoutil.DeschedulerCRName}

			Eventually(func(g Gomega, ctx context.Context) {
				err := cli.Get(ctx, crdKey, crd)
				if apierrors.IsNotFound(err) {
					Skip("Skip test when KubeDescheduler CRD is not present")
				}
				g.Expect(err).NotTo(HaveOccurred())
				err = cli.Get(ctx, key, initialDescheduler)
				if apierrors.IsNotFound(err) {
					Skip("Skip test when KubeDescheduler CR is not present")
				}
				g.Expect(err).NotTo(HaveOccurred())
			}).WithTimeout(10 * time.Second).WithPolling(500 * time.Millisecond).WithContext(ctx).Should(Succeed())
		})

		AfterAll(func(ctx context.Context) {
			key := client.ObjectKey{Namespace: hcoutil.DeschedulerNamespace, Name: hcoutil.DeschedulerCRName}

			Eventually(func(g Gomega, ctx context.Context) {
				descheduler := &deschedulerv1.KubeDescheduler{}
				err := cli.Get(ctx, key, descheduler)
				g.Expect(err).NotTo(HaveOccurred())
				initialDescheduler.Spec.DeepCopyInto(&descheduler.Spec)
				err = cli.Update(ctx, descheduler)
				g.Expect(err).NotTo(HaveOccurred())
			}).WithTimeout(10 * time.Second).WithPolling(500 * time.Millisecond).WithContext(ctx).Should(Succeed())
		})

		It("KubeVirtCRModified alert should fired when KubeDescheduler is installed and not properly configured for KubeVirt", Serial, func(ctx context.Context) {

			const (
				query                 = `kubevirt_hco_misconfigured_descheduler`
				jsonPatchMisconfigure = `[{"op": "replace", "path": "/spec", "value": {"managementState": "Managed"}}]`
				jsonPatchConfigure    = `[{"op": "replace", "path": "/spec", "value": {"managementState": "Managed", "profileCustomizations": {"devEnableEvictionsInBackground": true }}}]`
			)

			By(fmt.Sprintf("Reading the `%s` metric from HCO prometheus endpoint", query))
			var valueBefore float64
			Eventually(func(g Gomega, ctx context.Context) {
				var err error
				valueBefore, err = hcoClient.GetHCOMetric(ctx, query)
				g.Expect(err).NotTo(HaveOccurred())
			}).WithTimeout(10 * time.Second).WithPolling(500 * time.Millisecond).WithContext(ctx).Should(Succeed())
			GinkgoWriter.Printf("The metric value before the test is: %0.2f\n", valueBefore)

			patchMisconfigure := client.RawPatch(types.JSONPatchType, []byte(jsonPatchMisconfigure))
			patchConfigure := client.RawPatch(types.JSONPatchType, []byte(jsonPatchConfigure))

			descheduler := &deschedulerv1.KubeDescheduler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      hcoutil.DeschedulerCRName,
					Namespace: hcoutil.DeschedulerNamespace,
				},
			}

			By("Misconfiguring the descheduler")
			Expect(cli.Patch(ctx, descheduler, patchMisconfigure)).To(Succeed())
			By("checking that the metric reports it as misconfigured (1.0)")
			Eventually(func(g Gomega, ctx context.Context) float64 {
				valueAfter, err := hcoClient.GetHCOMetric(ctx, query)
				g.Expect(err).NotTo(HaveOccurred())
				return valueAfter
			}).
				WithTimeout(60*time.Second).
				WithPolling(time.Second).
				WithContext(ctx).
				Should(
					Equal(float64(1)),
					"expected descheduler to be misconfigured; expected value: %0.2f", float64(1),
				)

			By("checking that the prometheus metric reports it as misconfigured (0.0)")
			Eventually(func(ctx context.Context) float64 {
				return getMetricValue(ctx, promClient, query)
			}).
				WithTimeout(5*time.Minute).
				WithPolling(10*time.Second).
				WithContext(ctx).
				Should(
					Equal(float64(1)),
					"expected descheduler to be misconfigured; expected value: %0.2f", float64(1),
				)

			By("Checking the alert")
			Eventually(func(ctx context.Context) *promApiv1.Alert {
				alerts, err := promClient.Alerts(ctx)
				Expect(err).ToNot(HaveOccurred())
				alert := getAlertByName(alerts, hcoalerts.MisconfiguredDeschedulerAlert)
				return alert
			}).WithTimeout(prometheousTimeout).WithPolling(prometheousPolling).WithContext(ctx).ShouldNot(BeNil())

			By("Correctly configuring the descheduler for KubeVirt")
			Expect(cli.Patch(ctx, descheduler, patchConfigure)).To(Succeed())
			By("checking that the metric doesn't report it as misconfigured (0.0)")
			Eventually(func(g Gomega, ctx context.Context) float64 {
				valueAfter, err := hcoClient.GetHCOMetric(ctx, query)
				g.Expect(err).NotTo(HaveOccurred())
				return valueAfter
			}).
				WithTimeout(60*time.Second).
				WithPolling(time.Second).
				WithContext(ctx).
				Should(
					Equal(float64(0)),
					"expected descheduler to NOT be misconfigured; expected value: %0.2f", float64(0),
				)

			By("checking that the prometheus metric doesn't report it as misconfigured (0.0)")
			Eventually(func(ctx context.Context) float64 {
				return getMetricValue(ctx, promClient, query)
			}).
				WithTimeout(5*time.Minute).
				WithPolling(10*time.Second).
				WithContext(ctx).
				Should(
					Equal(float64(0)),
					"expected descheduler to NOT be misconfigured; expected value: %0.2f", float64(0),
				)

			By("Checking the alert is not firing")
			Eventually(func(ctx context.Context) *promApiv1.Alert {
				alerts, err := promClient.Alerts(ctx)
				Expect(err).ToNot(HaveOccurred())
				alert := getAlertByName(alerts, hcoalerts.MisconfiguredDeschedulerAlert)
				return alert
			}).WithTimeout(prometheousTimeout).WithPolling(prometheousPolling).WithContext(ctx).Should(BeNil())

			By("Misconfiguring a second time the descheduler")
			Expect(cli.Patch(ctx, descheduler, patchMisconfigure)).To(Succeed())
			By("checking that the metric reports it as misconfigured (1.0)")
			Eventually(func(g Gomega, ctx context.Context) float64 {
				valueAfter, err := hcoClient.GetHCOMetric(ctx, query)
				g.Expect(err).NotTo(HaveOccurred())
				return valueAfter
			}).
				WithTimeout(60*time.Second).
				WithPolling(time.Second).
				WithContext(ctx).
				Should(
					Equal(float64(1)),
					"expected descheduler to be misconfigured; expected value: %0.2f", float64(1),
				)

			By("checking that the prometheus metric reports it as misconfigured (0.0)")
			Eventually(func(ctx context.Context) float64 {
				return getMetricValue(ctx, promClient, query)
			}).
				WithTimeout(5*time.Minute).
				WithPolling(10*time.Second).
				WithContext(ctx).
				Should(
					Equal(float64(1)),
					"expected descheduler to be misconfigured; expected value: %0.2f", float64(1),
				)

			By("Checking the alert")
			Eventually(func(ctx context.Context) *promApiv1.Alert {
				alerts, err := promClient.Alerts(ctx)
				Expect(err).ToNot(HaveOccurred())
				alert := getAlertByName(alerts, hcoalerts.MisconfiguredDeschedulerAlert)
				return alert
			}).WithTimeout(prometheousTimeout).WithPolling(prometheousPolling).WithContext(ctx).ShouldNot(BeNil())
		})
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

func getMetricValue(ctx context.Context, promClient promApiv1.API, metricName string) float64 {
	queryResult, _, err := promClient.Query(ctx, metricName, time.Now())
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	resultVector := queryResult.(promModel.Vector)
	if len(resultVector) == 0 {
		return 0
	}

	ExpectWithOffset(1, resultVector).To(HaveLen(1))

	metricValue, err := strconv.ParseFloat(resultVector[0].Value.String(), 64)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

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
		Do(ctx).Into(&prometheusRule)).To(Succeed())
	return prometheusRule
}

func checkRunbookURLAvailability(rule monitoringv1.Rule) {
	resp, err := runbookClient.Head(rule.Annotations["runbook_url"])
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), fmt.Sprintf("%s runbook is not available", rule.Alert))
	ExpectWithOffset(1, resp.StatusCode).To(Equal(http.StatusOK), fmt.Sprintf("%s runbook is not available", rule.Alert))
}

func initializePromClient(prometheusURL string, token string) promApiv1.API {
	defaultRoundTripper := promApi.DefaultRoundTripper
	tripper := defaultRoundTripper.(*http.Transport)
	tripper.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	c, err := promApi.NewClient(promApi.Config{
		Address:      prometheusURL,
		RoundTripper: promConfig.NewAuthorizationCredentialsRoundTripper("Bearer", promConfig.NewInlineSecret(token), defaultRoundTripper),
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
