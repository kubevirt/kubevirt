package observability

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/alertmanager"
)

const (
	alertmanagerSvcHost = "https://alertmanager-main.openshift-monitoring.svc.cluster.local:9094"
	tlsCertPath         = "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
)

func (r *Reconciler) ensurePodDisruptionBudgetAtLimitIsSilenced() error {
	if r.amApi == nil {
		var err error
		r.amApi, err = r.NewAlertmanagerApi()
		if err != nil {
			return fmt.Errorf("failed to initialize alertmanager api: %w", err)
		}
	}

	amSilences, err := r.amApi.ListSilences()
	if err != nil {
		return fmt.Errorf("failed to list alertmanager silences: %w", err)
	}

	if FindPodDisruptionBudgetAtLimitSilence(amSilences) != nil {
		log.Info("KubeVirt PodDisruptionBudgetAtLimit alerts are already silenced")
		return nil
	}

	silence := alertmanager.Silence{
		Comment:   "Silence KubeVirt PodDisruptionBudgetAtLimit alerts",
		CreatedBy: "hyperconverged-cluster-operator",
		EndsAt:    "3000-01-01T00:00:00Z",
		Matchers: []alertmanager.Matcher{
			{
				IsEqual: true,
				Name:    "alertname",
				Value:   "PodDisruptionBudgetAtLimit",
			},
			{
				IsEqual: true,
				IsRegex: true,
				Name:    "poddisruptionbudget",
				Value:   "kubevirt-disruption-budget-.*",
			},
		},
		StartsAt: time.Now().Format(time.RFC3339),
	}

	if err := r.amApi.CreateSilence(silence); err != nil {
		return fmt.Errorf("failed to create alertmanager silence: %w", err)
	}
	log.Info("Silenced PodDisruptionBudgetAtLimit alerts")

	return nil
}

func (r *Reconciler) NewAlertmanagerApi() (*alertmanager.Api, error) {
	caCert, err := os.ReadFile(tlsCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ca cert: %w", err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	httpClient := http.Client{}
	httpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: caCertPool},
	}

	return alertmanager.NewAPI(httpClient, alertmanagerSvcHost, r.config.BearerToken), nil
}

func FindPodDisruptionBudgetAtLimitSilence(amSilences []alertmanager.Silence) *alertmanager.Silence {
	for _, silence := range amSilences {
		if silence.Status.State != "active" {
			continue
		}

		var isPDBSilence bool
		var isKubeVirtPDBSilence bool

		for _, matcher := range silence.Matchers {
			if matcher.Name == "alertname" && matcher.Value == "PodDisruptionBudgetAtLimit" && matcher.IsEqual {
				isPDBSilence = true
			}

			if matcher.Name == "poddisruptionbudget" && matcher.IsRegex && matcher.Value == "kubevirt-disruption-budget-.*" {
				isKubeVirtPDBSilence = true
			}
		}

		if isPDBSilence && isKubeVirtPDBSilence {
			return &silence
		}
	}

	return nil
}
