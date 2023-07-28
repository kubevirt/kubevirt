package monitoring

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/gomega"

	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
)

type AlertRequestResult struct {
	Alerts prometheusv1.AlertsResult `json:"data"`
	Status string                    `json:"status"`
}

type QueryRequestResult struct {
	Data   promData `json:"data"`
	Status string   `json:"status"`
}

type promData struct {
	ResultType string       `json:"resultType"`
	Result     []promResult `json:"result"`
}

type promResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

func getAlerts(cli kubecli.KubevirtClient) ([]prometheusv1.Alert, error) {
	bodyBytes := DoPrometheusHTTPRequest(cli, "/alerts")

	var result AlertRequestResult
	err := json.Unmarshal(bodyBytes, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("api request failed. result: %v", result)
	}

	return result.Alerts.Alerts, nil
}

func waitForMetricValue(client kubecli.KubevirtClient, metric string, expectedValue float64) {
	waitForMetricValueWithLabels(client, metric, expectedValue, nil)
}

func waitForMetricValueWithLabels(client kubecli.KubevirtClient, metric string, expectedValue float64, labels map[string]string) {
	EventuallyWithOffset(1, func() float64 {
		i, err := getMetricValueWithLabels(client, metric, labels)
		if err != nil {
			return -1
		}
		return i
	}, 3*time.Minute, 1*time.Second).Should(BeNumerically("==", expectedValue))
}

func getMetricValueWithLabels(cli kubecli.KubevirtClient, query string, labels map[string]string) (float64, error) {
	result, err := fetchMetric(cli, query)
	if err != nil {
		return -1, err
	}

	returnObj := findMetricWithLabels(result, labels)
	var output string

	if returnObj == nil {
		return -1, fmt.Errorf("metric value not populated yet")
	}

	if s, ok := returnObj.(string); ok {
		output = s
	} else {
		return -1, fmt.Errorf("metric value is not string")
	}

	returnVal, err := strconv.ParseFloat(output, 64)
	if err != nil {
		return -1, err
	}

	return returnVal, nil
}

func findMetricWithLabels(result *QueryRequestResult, labels map[string]string) interface{} {
	for _, r := range result.Data.Result {
		if labelsMatch(r, labels) {
			return r.Value[1]
		}
	}

	return nil
}

func labelsMatch(pr promResult, labels map[string]string) bool {
	for k, v := range labels {
		if pr.Metric[k] != v {
			return false
		}
	}

	return true
}

func fetchMetric(cli kubecli.KubevirtClient, query string) (*QueryRequestResult, error) {
	bodyBytes := DoPrometheusHTTPRequest(cli, fmt.Sprintf("/query?query=%s", query))

	var result QueryRequestResult
	err := json.Unmarshal(bodyBytes, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("api request failed. result: %v", result)
	}

	return &result, nil
}

func DoPrometheusHTTPRequest(cli kubecli.KubevirtClient, endpoint string) []byte {

	monitoringNs := getMonitoringNs(cli)
	token := getAuthorizationToken(cli, monitoringNs)

	var result []byte
	var err error
	if checks.IsOpenShift() {
		url := getPrometheusURLForOpenShift()
		resp := doHttpRequest(url, endpoint, token)
		defer resp.Body.Close()
		result, err = io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
	} else {
		sourcePort := 4321 + rand.Intn(6000)
		targetPort := 9090
		Eventually(func() error {
			_, cmd, err := clientcmd.CreateCommandWithNS(monitoringNs, clientcmd.GetK8sCmdClient(),
				"port-forward", "service/prometheus-k8s", fmt.Sprintf("%d:%d", sourcePort, targetPort))
			if err != nil {
				return err
			}
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				return err
			}
			if err := cmd.Start(); err != nil {
				return err
			}
			WaitForPortForwardCmd(stdout, sourcePort, targetPort)
			defer KillPortForwardCommand(cmd)

			url := fmt.Sprintf("http://localhost:%d", sourcePort)
			resp := doHttpRequest(url, endpoint, token)
			defer resp.Body.Close()
			result, err = io.ReadAll(resp.Body)
			return err
		}, 10*time.Second, time.Second).ShouldNot(HaveOccurred())
	}
	return result
}

func getPrometheusURLForOpenShift() string {
	var host string

	Eventually(func() error {
		var stderr string
		var err error
		host, stderr, err = clientcmd.RunCommand(clientcmd.GetK8sCmdClient(), "-n", "openshift-monitoring", "get", "route", "prometheus-k8s", "--template", "{{.spec.host}}")
		if err != nil {
			return fmt.Errorf("error while getting route. err:'%v', stderr:'%v'", err, stderr)
		}
		return nil
	}, 10*time.Second, time.Second).Should(BeTrue())

	return fmt.Sprintf("https://%s", host)
}

func doHttpRequest(url string, endpoint string, token string) *http.Response {
	var resp *http.Response
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	Eventually(func() bool {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/%s", url, endpoint), nil)
		if err != nil {
			return false
		}
		req.Header.Add("Authorization", "Bearer "+token)
		resp, err = client.Do(req)
		if err != nil {
			return false
		}
		if resp.StatusCode != http.StatusOK {
			return false
		}
		return true
	}, 10*time.Second, 1*time.Second).Should(BeTrue())

	return resp
}

func getAuthorizationToken(cli kubecli.KubevirtClient, monitoringNs string) string {
	var token string
	Eventually(func() bool {
		secretName := fmt.Sprintf("prometheus-k8s-%s-token", monitoringNs)
		secret, err := cli.CoreV1().Secrets(monitoringNs).Get(context.TODO(), secretName, metav1.GetOptions{})
		if err != nil {
			secretToken := k8sv1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: secretName,
					Annotations: map[string]string{
						"kubernetes.io/service-account.name": "prometheus-k8s",
					},
				},
				Type: k8sv1.SecretTypeServiceAccountToken,
			}
			_, err := cli.CoreV1().Secrets(monitoringNs).Create(context.Background(), &secretToken, metav1.CreateOptions{})
			if err != nil {
				return false
			}
			secret, err = cli.CoreV1().Secrets(monitoringNs).Get(context.TODO(), secretName, metav1.GetOptions{})
			if err != nil {
				return false
			}
		}
		if _, ok := secret.Data["token"]; !ok {
			return false
		}
		token = string(secret.Data["token"])
		return true
	}, 10*time.Second, time.Second).Should(BeTrue())
	return token
}

func getMonitoringNs(cli kubecli.KubevirtClient) string {
	if checks.IsOpenShift() {
		return "openshift-monitoring"
	}

	return "monitoring"
}

func WaitForPortForwardCmd(stdout io.ReadCloser, src, dst int) {
	Eventually(func() string {
		tmp := make([]byte, 1024)
		_, err := stdout.Read(tmp)
		Expect(err).NotTo(HaveOccurred())

		return string(tmp)
	}, 30*time.Second, 1*time.Second).Should(ContainSubstring(fmt.Sprintf("Forwarding from 127.0.0.1:%d -> %d", src, dst)))
}

func KillPortForwardCommand(portForwardCmd *exec.Cmd) error {
	if portForwardCmd == nil {
		return nil
	}

	portForwardCmd.Process.Kill()
	_, err := portForwardCmd.Process.Wait()
	return err
}

func checkAlertExists(virtClient kubecli.KubevirtClient, alertName string) bool {
	currentAlerts, err := getAlerts(virtClient)
	if err != nil {
		return false
	}
	for _, alert := range currentAlerts {
		if string(alert.Labels["alertname"]) == alertName {
			return true
		}
	}
	return false
}

func verifyAlertExistWithCustomTime(virtClient kubecli.KubevirtClient, alertName string, timeout time.Duration) {
	EventuallyWithOffset(1, func() bool {
		return checkAlertExists(virtClient, alertName)
	}, timeout, 10*time.Second).Should(BeTrue(), "Alert %s should exist", alertName)
}

func verifyAlertExist(virtClient kubecli.KubevirtClient, alertName string) {
	verifyAlertExistWithCustomTime(virtClient, alertName, 120*time.Second)
}

func waitUntilAlertDoesNotExistWithCustomTime(virtClient kubecli.KubevirtClient, timeout time.Duration, alertNames ...string) {
	presentAlert := EventuallyWithOffset(1, func() string {
		for _, name := range alertNames {
			if checkAlertExists(virtClient, name) {
				return name
			}
		}
		return ""
	}, timeout, 1*time.Second)

	presentAlert.Should(BeEmpty(), "Alert %v should not exist", presentAlert)
}

func waitUntilAlertDoesNotExist(virtClient kubecli.KubevirtClient, alertNames ...string) {
	waitUntilAlertDoesNotExistWithCustomTime(virtClient, 5*time.Minute, alertNames...)
}

func reduceAlertPendingTime(virtClient kubecli.KubevirtClient) {
	newRules := getPrometheusAlerts(virtClient)
	var re = regexp.MustCompile("\\[\\d+m\\]")

	var gs []promv1.RuleGroup
	for _, group := range newRules.Spec.Groups {
		var rs []promv1.Rule
		for _, rule := range group.Rules {
			var r promv1.Rule
			rule.DeepCopyInto(&r)
			if r.Alert != "" {
				r.For = "0m"
				r.Expr = intstr.FromString(re.ReplaceAllString(r.Expr.String(), `[1m]`))
				r.Expr = intstr.FromString(strings.ReplaceAll(r.Expr.String(), ">= 300", ">= 0"))
			}
			rs = append(rs, r)
		}

		gs = append(gs, promv1.RuleGroup{
			Name:  group.Name,
			Rules: rs,
		})
	}
	newRules.Spec.Groups = gs

	updatePromRules(virtClient, &newRules)
}

func updatePromRules(virtClient kubecli.KubevirtClient, newRules *promv1.PrometheusRule) {
	err := virtClient.
		PrometheusClient().MonitoringV1().
		PrometheusRules(flags.KubeVirtInstallNamespace).
		Delete(context.Background(), "prometheus-kubevirt-rules", metav1.DeleteOptions{})

	Expect(err).ToNot(HaveOccurred())

	_, err = virtClient.
		PrometheusClient().MonitoringV1().
		PrometheusRules(flags.KubeVirtInstallNamespace).
		Create(context.Background(), newRules, metav1.CreateOptions{})

	Expect(err).ToNot(HaveOccurred())
}

func getPrometheusAlerts(virtClient kubecli.KubevirtClient) promv1.PrometheusRule {
	promRules, err := virtClient.
		PrometheusClient().MonitoringV1().
		PrometheusRules(flags.KubeVirtInstallNamespace).
		Get(context.Background(), "prometheus-kubevirt-rules", metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	var newRules promv1.PrometheusRule
	promRules.DeepCopyInto(&newRules)

	newRules.Annotations = nil
	newRules.ObjectMeta.ResourceVersion = ""
	newRules.ObjectMeta.UID = ""

	return newRules
}
