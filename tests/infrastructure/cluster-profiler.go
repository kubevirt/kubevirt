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

package infrastructure

import (
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe(SIGSerial("cluster profiler for pprof data aggregation", func() {
	var virtClient kubecli.KubevirtClient
	var kvConfig v1.KubeVirtConfiguration

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		kv := libkubevirt.GetCurrentKv(virtClient)
		kvConfig = kv.Spec.Configuration

		if kvConfig.DeveloperConfiguration == nil {
			kvConfig.DeveloperConfiguration = &v1.DeveloperConfiguration{}
		}
	})

	Context("when ClusterProfiler configuration", func() {
		It("is disabled it should prevent subresource access", func() {
			kvConfig.DeveloperConfiguration.ClusterProfiler = false
			config.UpdateKubeVirtConfigValueAndWait(kvConfig)

			err := virtClient.ClusterProfiler().Start()
			Expect(err).To(HaveOccurred())

			err = virtClient.ClusterProfiler().Stop()
			Expect(err).To(HaveOccurred())

			_, err = virtClient.ClusterProfiler().Dump(&v1.ClusterProfilerRequest{})
			Expect(err).To(HaveOccurred())
		})
		It("[QUARANTINE]is enabled it should allow subresource access", decorators.Quarantine, func() {
			kvConfig.DeveloperConfiguration.ClusterProfiler = true
			config.UpdateKubeVirtConfigValueAndWait(kvConfig)

			err := virtClient.ClusterProfiler().Start()
			Expect(err).ToNot(HaveOccurred())

			err = virtClient.ClusterProfiler().Stop()
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.ClusterProfiler().Dump(&v1.ClusterProfilerRequest{})
			Expect(err).ToNot(HaveOccurred())
		})
	})
}))
