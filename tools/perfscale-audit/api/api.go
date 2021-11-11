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

package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"time"
)

type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return fmt.Errorf("invalid duration")
	}
}

type InputThreshold struct {
	Value float64 `json:"value"`
}

type InputConfig struct {

	// StartTime when set, represents the beginning of the metric time range
	// This defaults to EndTime - Duration when duration is set.
	StartTime *time.Time `json:"startTime,omitempty"`
	// EndTime when set, represents end of the metric time range
	// This defaults to the current time
	EndTime *time.Time `json:"endTime,omitempty"`
	// Duration represents how long to go back from EndTime when creating the metric time range
	// This is mutually exclusive with the StartTime value. Only one of these
	// two values can be set.
	Duration *Duration `json:"duration,omitempty"`

	PrometheusURL         string `json:"prometheusURL"`
	PrometheusUserName    string `json:"prometheusUserName"`
	PrometheusPassword    string `json:"prometheusPassword"`
	PrometheusBearerToken string `json:"prometheusBearerToken"`
	PrometheusVerifyTLS   bool   `json:"prometheusVerifyTLS"`

	ThresholdExpectations map[ResultType]InputThreshold `json:"thresholdExpectations,omitempty"`
}

func (i *InputConfig) GetDuration() time.Duration {
	return time.Duration(*i.Duration)
}

type ResultType string

const (
	ResultTypeVMICreationToRunningP99 ResultType = "vmiCreationToRunningSecondsP99"
	ResultTypeVMICreationToRunningP95 ResultType = "vmiCreationToRunningSecondsP95"
	ResultTypeVMICreationToRunningP50 ResultType = "vmiCreationToRunningSecondsP50"
)

const (
	ResultTypeResourceOperationCountFormat = "%s-%s-count"
)

const (
	ResultTypePhaseCountFormat = "%s-phase-count"
)

type ThresholdResult struct {
	ThresholdValue    float64 `json:"thresholdValue"`
	ThresholdExceeded bool    `json:"thresholdExceeded"`
}

type ResultValue struct {
	Value           float64          `json:"value"`
	ThresholdResult *ThresholdResult `json:"thresholdResult,omitempty"`
}

type Result struct {
	Values map[ResultType]ResultValue
}

func (r *Result) toString() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *Result) DumpToFile(filePath string) error {
	str, err := r.toString()
	if err != nil {
		return err
	}

	log.Printf("Writing results to file at path %s", filePath)

	return ioutil.WriteFile(filePath, []byte(str), 0644)
}

func (r *Result) DumpToStdout() error {

	str, err := r.toString()
	if err != nil {
		return err
	}
	fmt.Println(str)
	return nil
}

func ReadInputFile(filePath string) (*InputConfig, error) {
	cfg := &InputConfig{}

	log.Printf("Reading config at path %s", filePath)

	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("Unable to read file [%s]: %v", filePath, err)
	}

	if err := json.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("Failed to json unmarshal input config: %v", err)
	}

	if cfg.EndTime == nil {
		now := time.Now()
		cfg.EndTime = &now
	}

	if cfg.StartTime == nil && cfg.Duration == nil {
		defaultDuration := 10 * time.Minute
		startTime := cfg.EndTime.Add(-1 * defaultDuration)

		duration := Duration(defaultDuration)
		cfg.StartTime = &startTime
		cfg.Duration = &duration
	} else if cfg.StartTime == nil {
		startTime := cfg.EndTime.Add(-1 * time.Duration(*cfg.Duration))
		cfg.StartTime = &startTime
	} else if cfg.Duration == nil {
		duration := Duration(cfg.EndTime.Sub(*cfg.StartTime))
		cfg.Duration = &duration
	}

	log.Printf("Using the following cfg values\n%v\n", cfg)

	return cfg, nil
}
