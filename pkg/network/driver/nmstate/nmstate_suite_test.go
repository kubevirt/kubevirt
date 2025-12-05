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

// nmstate_test includes integration tests which create a network namespace
// per test.
package nmstate_test

import (
	"net"
	"testing"

	"github.com/onsi/ginkgo/v2"

	vishnetlink "github.com/vishvananda/netlink"

	"kubevirt.io/client-go/testutils"

	nlfake "kubevirt.io/kubevirt/pkg/network/driver/netlink/fake"
	psfake "kubevirt.io/kubevirt/pkg/network/driver/procsys/fake"
)

const (
	macAddress0 = "31:32:33:34:35:36" // []byte("123456")
)

var integrationLabel = ginkgo.Label("integration")

func TestNMState(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}

type testAdapter struct {
	nlfake.NetLink
	psfake.ProcSys
}

func (t *testAdapter) AddTapDeviceWithSELinuxLabel(name string, mtu int, queues int, ownerID int, pid int) error {
	return nil
}

func (t *testAdapter) setIPConfigOnLink(linkIndex int, ipAddresses ...string) error {
	links, err := t.LinkList()
	if err != nil {
		return err
	}

	for _, ipAddr := range ipAddresses {
		ip, ipNet, err := net.ParseCIDR(ipAddr)
		if err != nil {
			return err
		}
		addr := vishnetlink.Addr{IPNet: &net.IPNet{IP: ip, Mask: ipNet.Mask}}
		if err := t.AddrAdd(links[linkIndex], &addr); err != nil {
			return err
		}
	}
	return nil
}
