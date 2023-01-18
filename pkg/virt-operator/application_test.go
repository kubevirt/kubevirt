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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package virt_operator

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
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
		app.stores.ServiceMonitorEnabled = hasServiceMonitor
		app.stores.PrometheusRulesEnabled = hasPrometheusRules

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
