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
 * Copyright 2022 Red Hat, Inc.
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

	nlfake "kubevirt.io/kubevirt/pkg/network/driver/netlink/fake"
	"kubevirt.io/kubevirt/pkg/network/driver/nmstate"
	psfake "kubevirt.io/kubevirt/pkg/network/driver/procsys/fake"
	"kubevirt.io/kubevirt/pkg/pointer"

	"kubevirt.io/client-go/testutils"
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
	txChecksum map[string]*bool
}

func (t *testAdapter) ReadTXChecksum(name string) (bool, error) {
	if txChecksum, exists := t.txChecksum[name]; exists && txChecksum != nil {
		return *txChecksum, nil
	}
	// By default, TX checksum is usually enabled by default.
	return true, nil
}

func (t *testAdapter) TXChecksumOff(name string) error {
	t.txChecksum[name] = pointer.P(false)
	return nil
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

func defaultEthtool() nmstate.Ethtool {
	return nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(true)}}
}
