/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2021 Red Hat, Inc.
 *
 */

package metricclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	audit_api "kubevirt.io/kubevirt/tools/perfscale-audit/api"

	api "github.com/prometheus/client_golang/api"
	apiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type transport struct {
	transport http.RoundTripper
	userName  string
	password  string
	token     string
}

func (a transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if a.userName != "" {
		req.SetBasicAuth(a.userName, a.password)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.token))
	return a.transport.RoundTrip(req)
}

type MetricClient struct {
	client apiv1.API
	cfg    *audit_api.InputConfig
}

func NewMetricClient(cfg *audit_api.InputConfig) (*MetricClient, error) {

	url := cfg.PrometheusURL
	token := cfg.PrometheusBearerToken
	userName := cfg.PrometheusUserName
	password := cfg.PrometheusPassword
	tlsVerify := cfg.PrometheusVerifyTLS

	apiCfg := api.Config{
		Address: url,
		RoundTripper: transport{
			transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: tlsVerify}},
			token:     token,
			userName:  userName,
			password:  password,
		},
	}
	c, err := api.NewClient(apiCfg)
	if err != nil {
		return nil, err
	}
	api := apiv1.NewAPI(c)

	_, err = api.Config(context.TODO())
	if err != nil {
		return nil, err
	}
	log.Print("Established connection with prometheus endpoint.")

	return &MetricClient{client: api, cfg: cfg}, nil
}

func (m *MetricClient) query(query string) (model.Value, error) {
	log.Printf("Makeing query [%s]", query)
	val, _, err := m.client.Query(context.TODO(), query, *m.cfg.EndTime)
	if err != nil {
		return val, err
	}
	return val, nil

}

type metric struct {
	labels    map[string]string
	value     float64
	timestamp time.Time
}

func parseVector(value model.Value) ([]metric, error) {
	var metrics []metric

	data, ok := value.(model.Vector)
	if !ok {
		return metrics, fmt.Errorf("unexpected format %s, expected vector", value.Type().String())
	}
	for _, v := range data {
		m := metric{
			labels: make(map[string]string),
		}
		for k, v := range v.Metric {
			m.labels[string(k)] = string(v)
		}
		if math.IsNaN(float64(v.Value)) {
			m.value = 0
		} else {
			m.value = float64(v.Value)
		}
		m.timestamp = v.Timestamp.Time()
		metrics = append(metrics, m)
	}
	return metrics, nil
}

func (m *MetricClient) getCreationToRunningTimePercentile(percentile int) (float64, error) {
	query := fmt.Sprintf(`histogram_quantile(0.%d, rate(kubevirt_vmi_phase_transition_time_from_creation_seconds_bucket{phase="Running"}[%ds]))`, percentile, int(m.cfg.Duration.Seconds()))

	val, err := m.query(query)
	if err != nil {
		return 0, err
	}

	results, err := parseVector(val)
	if err != nil {
		return 0, err
	}

	if len(results) == 0 {
		return 0.0, nil
	}

	return results[0].value, nil
}

func (m *MetricClient) GenerateResults() (*audit_api.Result, error) {
	var err error
	r := &audit_api.Result{}

	r.PhaseTransitionPercentiles.P99CreationToRunning, err = m.getCreationToRunningTimePercentile(99)
	if err != nil {
		return nil, err
	}
	r.PhaseTransitionPercentiles.P95CreationToRunning, err = m.getCreationToRunningTimePercentile(95)
	if err != nil {
		return nil, err
	}
	r.PhaseTransitionPercentiles.P50CreationToRunning, err = m.getCreationToRunningTimePercentile(50)
	if err != nil {
		return nil, err
	}

	return r, nil
}
