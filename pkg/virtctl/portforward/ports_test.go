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

package portforward

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ports", func() {

	DescribeTable("parsePort", func(arg string, port forwardedPort, success bool) {
		result, err := parsePort(arg)
		if success {
			Expect(err).NotTo(HaveOccurred())
		} else {
			Expect(err).To(HaveOccurred())
		}
		Expect(result).To(Equal(port))
	},
		Entry("only one port", "8080", forwardedPort{local: 8080, remote: 8080, protocol: protocolTCP}, true),
		Entry("local and remote port", "8080:8090", forwardedPort{local: 8080, remote: 8090, protocol: protocolTCP}, true),
		Entry("protocol and one port", "udp/8080", forwardedPort{local: 8080, remote: 8080, protocol: protocolUDP}, true),

		Entry("protocol and both ports", "udp/8080:8090", forwardedPort{local: 8080, remote: 8090, protocol: protocolUDP}, true),

		Entry("only protocol no slash", "udp", forwardedPort{local: 0, remote: 0, protocol: protocolTCP}, false),
		Entry("only protocol with slash", "udp/", forwardedPort{local: 0, remote: 0, protocol: protocolUDP}, false),
		Entry("invalid symbol in port", "80C0:8X90", forwardedPort{local: 0, remote: 0, protocol: protocolTCP}, false),
	)
})
