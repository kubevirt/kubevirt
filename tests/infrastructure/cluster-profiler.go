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
 * Copyright 2017-2023 Red Hat, Inc.
 *
 */

package infrastructure

import (
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
)

var _ = DescribeInfra("cluster profiler for pprof data aggregation", func() {
	var (
		virtClient kubecli.KubevirtClient
	)
	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("when ClusterProfiler feature gate", func() {
		It("is disabled it should prevent subresource access", func() {
			tests.DisableFeatureGate("ClusterProfiler")

			err := virtClient.ClusterProfiler().Start()
			Expect(err).To(HaveOccurred())

			err = virtClient.ClusterProfiler().Stop()
			Expect(err).To(HaveOccurred())

			_, err = virtClient.ClusterProfiler().Dump(&v1.ClusterProfilerRequest{})
			Expect(err).To(HaveOccurred())
		})
		It("is enabled it should allow subresource access", func() {
			tests.EnableFeatureGate("ClusterProfiler")

			err := virtClient.ClusterProfiler().Start()
			Expect(err).ToNot(HaveOccurred())

			err = virtClient.ClusterProfiler().Stop()
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.ClusterProfiler().Dump(&v1.ClusterProfilerRequest{})
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
