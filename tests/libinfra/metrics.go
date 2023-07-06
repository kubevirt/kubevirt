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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package libinfra

import (
	"encoding/xml"
	"strconv"

	expect "github.com/google/goexpect"
	"github.com/onsi/ginkgo/v2"
	v1 "kubevirt.io/api/core/v1"

	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd/api"
	"kubevirt.io/kubevirt/tests/console"
)

func GetDownwardMetrics(vmi *v1.VirtualMachineInstance) (*api.Metrics, error) {
	res, err := console.SafeExpectBatchWithResponse(vmi, []expect.Batcher{
		&expect.BSnd{S: `sudo vm-dump-metrics 2> /dev/null` + "\n"},
		&expect.BExp{R: `(?s)(<metrics>.+</metrics>)`},
	}, 5)
	if err != nil {
		return nil, err
	}
	metricsStr := res[0].Match[2]
	metrics := &api.Metrics{}
	Expect(xml.Unmarshal([]byte(metricsStr), metrics)).To(Succeed())
	return metrics, nil
}

func GetTimeFromMetrics(metrics *api.Metrics) int {

	for _, m := range metrics.Metrics {
		if m.Name == "Time" {
			val, err := strconv.Atoi(m.Value)
			Expect(err).ToNot(HaveOccurred())
			return val
		}
	}
	ginkgo.Fail("no Time in metrics XML")
	return -1
}

func GetHostnameFromMetrics(metrics *api.Metrics) string {
	for _, m := range metrics.Metrics {
		if m.Name == "HostName" {
			return m.Value
		}
	}
	ginkgo.Fail("no hostname in metrics XML")
	return ""
}
