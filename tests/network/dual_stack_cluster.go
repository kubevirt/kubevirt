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
 */

package network

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/libnet/cluster"

	"kubevirt.io/kubevirt/tests/flags"
)

var _ = Describe(SIG("Dual stack cluster network configuration", func() {
	Context("when dual stack cluster configuration is enabled", func() {
		Specify("the cluster must be dual stack", func() {
			if flags.SkipDualStackTests {
				Skip("user requested the dual stack check on the live cluster to be skipped")
			}

			isClusterDualStack, err := cluster.DualStack()
			Expect(err).NotTo(HaveOccurred(), "must be able to infer the dual stack configuration from the live cluster")
			Expect(isClusterDualStack).To(BeTrue(), "the live cluster should be in dual stack mode")
		})
	})
}))
