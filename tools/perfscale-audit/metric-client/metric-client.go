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

const (
	vmiCreationTimePercentileQuery   = `histogram_quantile(0.%d, rate(kubevirt_vmi_phase_transition_time_from_creation_seconds_bucket{phase="Running"}[%ds]))`
	resourceRequestCountsByOperation = `increase(rest_client_requests_total{pod=~"virt-controller.*|virt-handler.*|virt-operator.*|virt-api.*"}[%ds])`
)

// Gauge - Using a Gauge doesn't require using an offset because it holds the accurate count
//         at all times.
const (
	vmiPhaseCount = `sum by (phase) (kubevirt_vmi_phase_count{})`
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
	log.Printf("Making query [%s]", query)
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

func (m *MetricClient) getCreationToRunningTimePercentiles(r *audit_api.Result) error {

	type percentile struct {
		p int
		t audit_api.ResultType
	}
	percentiles := []percentile{
		{
			p: 99,
			t: audit_api.ResultTypeVMICreationToRunningP99,
		},
		{
			p: 95,
			t: audit_api.ResultTypeVMICreationToRunningP95,
		},
		{
			p: 50,
			t: audit_api.ResultTypeVMICreationToRunningP50,
		},
	}

	for _, percentile := range percentiles {
		query := fmt.Sprintf(vmiCreationTimePercentileQuery, percentile.p, int(m.cfg.GetDuration().Seconds()))

		val, err := m.query(query)
		if err != nil {
			return err
		}

		results, err := parseVector(val)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			r.Values[percentile.t] = audit_api.ResultValue{
				Value: 0.0,
			}
		} else {
			r.Values[percentile.t] = audit_api.ResultValue{
				Value: results[0].value,
			}
		}

	}
	return nil
}

func (m *MetricClient) getPhaseBreakdown(r *audit_api.Result) error {
	query := vmiPhaseCount

	val, err := m.query(query)
	if err != nil {
		return err
	}

	results, err := parseVector(val)
	if err != nil {
		return err
	}

	for _, result := range results {
		if result.value < 1 {
			continue
		}
		phase, ok := result.labels["phase"]
		if !ok {
			continue
		}

		key := audit_api.ResultType(fmt.Sprintf(audit_api.ResultTypePhaseCountFormat, phase))

		val, ok := r.Values[key]
		if ok {
			val.Value = val.Value + result.value
			r.Values[key] = val
		} else {
			r.Values[key] = audit_api.ResultValue{
				Value: result.value,
			}
		}
	}
	return nil
}

func (m *MetricClient) getResourceRequestCountsByOperation(r *audit_api.Result) error {
	query := fmt.Sprintf(resourceRequestCountsByOperation, int(m.cfg.GetDuration().Seconds()))

	val, err := m.query(query)
	if err != nil {
		return err
	}

	results, err := parseVector(val)
	if err != nil {
		return err
	}

	for _, result := range results {
		if result.value < 1 {
			continue
		}
		resource, ok := result.labels["resource"]
		if !ok {
			continue
		}
		verb, ok := result.labels["verb"]
		if !ok {
			continue
		}

		key := audit_api.ResultType(fmt.Sprintf(audit_api.ResultTypeResourceOperationCountFormat, verb, resource))

		val, ok := r.Values[key]
		if ok {
			val.Value = val.Value + result.value
			r.Values[key] = val
		} else {
			r.Values[key] = audit_api.ResultValue{
				Value: result.value,
			}
		}
	}

	return nil
}

func (m *MetricClient) gatherMetrics() (*audit_api.Result, error) {
	r := &audit_api.Result{
		Values: make(map[audit_api.ResultType]audit_api.ResultValue),
	}

	err := m.getCreationToRunningTimePercentiles(r)
	if err != nil {
		return nil, err
	}

	err = m.getResourceRequestCountsByOperation(r)
	if err != nil {
		return nil, err
	}

	err = m.getPhaseBreakdown(r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (m *MetricClient) calculateThresholds(r *audit_api.Result) error {

	inputCfg := m.cfg

	if len(inputCfg.ThresholdExpectations) == 0 {
		return nil
	}

	for key, v := range inputCfg.ThresholdExpectations {
		result, ok := r.Values[key]
		if !ok {
			log.Printf("Encountered input threshold [%s] with no matching results. Double check threshold key is accurate. If accurate, then results are likely 0.", key)
			continue
		}

		thresholdResult := audit_api.ThresholdResult{
			ThresholdValue:    v.Value,
			ThresholdExceeded: false,
		}
		if result.Value > v.Value {
			thresholdResult.ThresholdExceeded = true
		}
		result.ThresholdResult = &thresholdResult
		r.Values[key] = result
	}

	return nil
}

func (m *MetricClient) GenerateResults() (*audit_api.Result, error) {
	r, err := m.gatherMetrics()
	if err != nil {
		return nil, err
	}

	err = m.calculateThresholds(r)
	if err != nil {
		return nil, err
	}

	return r, nil
}
