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

package libnet

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/libnet/cluster"
)

func SkipWhenClusterNotSupportIpv4() {
	clusterSupportsIpv4, err := cluster.SupportsIpv4()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "should have been able to infer if the cluster supports ipv4")
	if !clusterSupportsIpv4 {
		Skip("This test requires an ipv4 network config.")
	}
}

func SkipWhenClusterNotSupportIpv6() {
	clusterSupportsIpv6, err := cluster.SupportsIpv6()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "should have been able to infer if the cluster supports ipv6")
	if !clusterSupportsIpv6 {
		Skip("This test requires an ipv6 network config.")
	}
}

func SkipWhenClusterNotSupportIPFamily(ipFamily k8sv1.IPFamily) {
	if ipFamily == k8sv1.IPv4Protocol {
		SkipWhenClusterNotSupportIpv4()
	} else {
		SkipWhenClusterNotSupportIpv6()
	}
}
