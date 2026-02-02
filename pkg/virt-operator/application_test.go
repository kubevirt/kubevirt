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
 * Copyright The KubeVirt Authors.
 *
 */

package virt_operator

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Reinitialization conditions", func() {
	DescribeTable("Re-trigger initialization", func(
		hasServiceMonitor bool, hasPrometheusRules bool,
		addServiceMonitorCrd bool, removeServiceMonitorCrd bool,
		addPrometheusRuleCrd bool, removePrometheusRuleCrd bool,
		expectReInit bool) {
		var reInitTriggered bool

		app := VirtOperatorApp{}

		clusterConfig, crdInformer, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
		app.clusterConfig = clusterConfig
		app.reInitChan = make(chan string, 10)
		app.config.ServiceMonitorEnabled = hasServiceMonitor
		app.config.PrometheusRulesEnabled = hasPrometheusRules

		if addServiceMonitorCrd {
			testutils.AddServiceMonitorAPI(crdInformer)
		} else if removeServiceMonitorCrd {
			testutils.RemoveServiceMonitorAPI(crdInformer)
		}

		if addPrometheusRuleCrd {
			testutils.AddPrometheusRuleAPI(crdInformer)
		} else if removePrometheusRuleCrd {
			testutils.RemovePrometheusRuleAPI(crdInformer)
		}

		app.clusterConfig.SetConfigModifiedCallback(app.configModificationCallback)

		select {
		case <-app.reInitChan:
			reInitTriggered = true
		case <-time.After(1 * time.Second):
			reInitTriggered = false
		}

		Expect(reInitTriggered).To(Equal(expectReInit))
	},
		Entry("when ServiceMonitor is introduced", false, false, true, false, false, false, true),
		Entry("when ServiceMonitor is removed", true, false, false, true, false, false, true),
		Entry("when PrometheusRule is introduced", false, false, false, false, true, false, true),
		Entry("when PrometheusRule is removed", false, true, false, false, false, true, true),

		Entry("when ServiceMonitor and PrometheusRule are introduced", false, false, true, false, true, false, true),
		Entry("when ServiceMonitor and PrometheusRule are removed", true, true, false, true, false, true, true),

		Entry("not when nothing changed and ServiceMonitor and PrometheusRule exists", true, true, true, false, true, false, false),
		Entry("not when nothing changed and ServiceMonitor and PrometheusRule does not exists", false, false, false, true, false, true, false),

		Entry("when ServiceMonitor is introduced and PrometheusRule is removed", false, true, true, false, false, true, true),
		Entry("when ServiceMonitor is removed and PrometheusRule is introduced", true, false, false, true, true, false, true),
	)
})

var _ = Describe("Client rate limiter configuration", func() {
	AfterEach(func() {
		// Clean up environment variables after each test
		os.Unsetenv(EnvVirtOperatorClientQPS)
		os.Unsetenv(EnvVirtOperatorClientBurst)
	})

	Context("getClientRateLimiterConfig", func() {
		It("should return default values when no flags or env vars are set", func() {
			qps, burst := getClientRateLimiterConfig(0, 0)
			Expect(qps).To(Equal(virtconfig.DefaultVirtOperatorQPS))
			Expect(burst).To(Equal(virtconfig.DefaultVirtOperatorBurst))
		})

		It("should use flag values when provided", func() {
			qps, burst := getClientRateLimiterConfig(100.0, 200)
			Expect(qps).To(Equal(float32(100.0)))
			Expect(burst).To(Equal(200))
		})

		It("should use environment variables when flags are not set", func() {
			os.Setenv(EnvVirtOperatorClientQPS, "150")
			os.Setenv(EnvVirtOperatorClientBurst, "300")

			qps, burst := getClientRateLimiterConfig(0, 0)
			Expect(qps).To(Equal(float32(150.0)))
			Expect(burst).To(Equal(300))
		})

		It("should prefer flags over environment variables", func() {
			os.Setenv(EnvVirtOperatorClientQPS, "150")
			os.Setenv(EnvVirtOperatorClientBurst, "300")

			qps, burst := getClientRateLimiterConfig(100.0, 200)
			Expect(qps).To(Equal(float32(100.0)))
			Expect(burst).To(Equal(200))
		})

		It("should use default when QPS env var is invalid", func() {
			os.Setenv(EnvVirtOperatorClientQPS, "invalid")
			os.Setenv(EnvVirtOperatorClientBurst, "300")

			qps, burst := getClientRateLimiterConfig(0, 0)
			Expect(qps).To(Equal(virtconfig.DefaultVirtOperatorQPS))
			Expect(burst).To(Equal(300))
		})

		It("should use default when Burst env var is invalid", func() {
			os.Setenv(EnvVirtOperatorClientQPS, "150")
			os.Setenv(EnvVirtOperatorClientBurst, "invalid")

			qps, burst := getClientRateLimiterConfig(0, 0)
			Expect(qps).To(Equal(float32(150.0)))
			Expect(burst).To(Equal(virtconfig.DefaultVirtOperatorBurst))
		})

		It("should use default when QPS env var is negative", func() {
			os.Setenv(EnvVirtOperatorClientQPS, "-100")
			os.Setenv(EnvVirtOperatorClientBurst, "300")

			qps, burst := getClientRateLimiterConfig(0, 0)
			Expect(qps).To(Equal(virtconfig.DefaultVirtOperatorQPS))
			Expect(burst).To(Equal(300))
		})

		It("should use default when Burst env var is negative", func() {
			os.Setenv(EnvVirtOperatorClientQPS, "150")
			os.Setenv(EnvVirtOperatorClientBurst, "-200")

			qps, burst := getClientRateLimiterConfig(0, 0)
			Expect(qps).To(Equal(float32(150.0)))
			Expect(burst).To(Equal(virtconfig.DefaultVirtOperatorBurst))
		})

		It("should use default when QPS env var is zero", func() {
			os.Setenv(EnvVirtOperatorClientQPS, "0")
			os.Setenv(EnvVirtOperatorClientBurst, "300")

			qps, burst := getClientRateLimiterConfig(0, 0)
			Expect(qps).To(Equal(virtconfig.DefaultVirtOperatorQPS))
			Expect(burst).To(Equal(300))
		})

		It("should use default when Burst env var is zero", func() {
			os.Setenv(EnvVirtOperatorClientQPS, "150")
			os.Setenv(EnvVirtOperatorClientBurst, "0")

			qps, burst := getClientRateLimiterConfig(0, 0)
			Expect(qps).To(Equal(float32(150.0)))
			Expect(burst).To(Equal(virtconfig.DefaultVirtOperatorBurst))
		})

		It("should use default when QPS flag is negative", func() {
			qps, burst := getClientRateLimiterConfig(-100.0, 200)
			Expect(qps).To(Equal(virtconfig.DefaultVirtOperatorQPS))
			Expect(burst).To(Equal(200))
		})

		It("should use default when Burst flag is negative", func() {
			qps, burst := getClientRateLimiterConfig(100.0, -200)
			Expect(qps).To(Equal(float32(100.0)))
			Expect(burst).To(Equal(virtconfig.DefaultVirtOperatorBurst))
		})

		It("should handle QPS as float value", func() {
			os.Setenv(EnvVirtOperatorClientQPS, "123.45")

			qps, burst := getClientRateLimiterConfig(0, 0)
			Expect(qps).To(Equal(float32(123.45)))
			Expect(burst).To(Equal(virtconfig.DefaultVirtOperatorBurst))
		})

		It("should use only QPS from env when only QPS is set", func() {
			os.Setenv(EnvVirtOperatorClientQPS, "150")

			qps, burst := getClientRateLimiterConfig(0, 0)
			Expect(qps).To(Equal(float32(150.0)))
			Expect(burst).To(Equal(virtconfig.DefaultVirtOperatorBurst))
		})

		It("should use only Burst from env when only Burst is set", func() {
			os.Setenv(EnvVirtOperatorClientBurst, "300")

			qps, burst := getClientRateLimiterConfig(0, 0)
			Expect(qps).To(Equal(virtconfig.DefaultVirtOperatorQPS))
			Expect(burst).To(Equal(300))
		})

		It("should use flag for QPS and env for Burst", func() {
			os.Setenv(EnvVirtOperatorClientBurst, "300")

			qps, burst := getClientRateLimiterConfig(100.0, 0)
			Expect(qps).To(Equal(float32(100.0)))
			Expect(burst).To(Equal(300))
		})

		It("should use env for QPS and flag for Burst", func() {
			os.Setenv(EnvVirtOperatorClientQPS, "150")

			qps, burst := getClientRateLimiterConfig(0, 200)
			Expect(qps).To(Equal(float32(150.0)))
			Expect(burst).To(Equal(200))
		})
	})
})
