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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package plugin_test

import (
	"bytes"
	"fmt"
	"kubevirt.io/kubevirt/cmd/cniplugins/bridge-binding/pkg/plugin/netlink"
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	vishnetlink "github.com/vishvananda/netlink"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/plugins/pkg/ns"

	"kubevirt.io/kubevirt/cmd/cniplugins/bridge-binding/pkg/plugin"
)

const (
	testNSPath = "/test/ns"
)

var (
	testMACAddress = net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
)

var _ = Describe("bridge-binding-plugin", func() {

	Context("Add", func() {
		It("succeeds", func() {
			var err error

			cmd := plugin.NewCmd(stubNetNS{}, stubSysCtl{}, &stubNetLink{NetLink: netlink.NetLink{PodNs: stubNetNS{}}})
			args := &skel.CmdArgs{
				ContainerID: "123456789",
				Netns:       testNSPath,
				IfName:      "",
				StdinData:   []byte(`{"cniVersion":"1.0.0"}`),
			}
			result, err := cmd.CmdAddResult(args)
			Expect(err).NotTo(HaveOccurred())

			versionedResult, err := result.GetAsVersion(result.Version())
			Expect(err).NotTo(HaveOccurred())

			var buf bytes.Buffer
			Expect(versionedResult.PrintTo(&buf)).To(Succeed())
			Expect(buf.String()).To(MatchJSON(fmt.Sprintf(`
			{
				"cniVersion": "1.0.0",
				"interfaces": [
					{
						"name": "br0",
						"mac": "%s",
						"sandbox": "%s"
					},
					{
						"name": "tap0",
						"mac": "%s",
						"sandbox": "%s"
					},
					{
						"name": "eth0",
						"mac": "%s",
						"sandbox": "%s"
					}
				],
				"dns": {}
			}
		`, testMACAddress.String(), testNSPath,
				testMACAddress.String(), testNSPath,
				testMACAddress.String(), testNSPath,
			)))
		})
	})

	It("Del", func() {
		Expect(plugin.CmdDel(&skel.CmdArgs{})).To(Succeed())
	})

	It("Check", func() {
		Expect(plugin.CmdCheck(&skel.CmdArgs{})).To(Succeed())
	})
})

type stubNetNS struct{}

func (s stubNetNS) Do(toRun func(ns.NetNS) error) error {
	return toRun(s)
}

func (s stubNetNS) Set() error {
	return nil
}

func (s stubNetNS) Path() string {
	return testNSPath
}

func (s stubNetNS) Fd() uintptr {
	return 0
}

func (s stubNetNS) Close() error {
	return nil
}

type stubSysCtl struct {
	pingGroupErr  error
	unprivPortErr error
}

func (s stubSysCtl) IPv4SetPingGroupRange(from, to int) error {
	return s.pingGroupErr
}

func (s stubSysCtl) IPv4SetUnprivilegedPortStart(port int) error {
	return s.unprivPortErr
}

type stubNetLink struct {
	netlink.NetLink
}

func (s *stubNetLink) ReadLink(name string) (vishnetlink.Link, error) {
	return &vishnetlink.Dummy{LinkAttrs: vishnetlink.LinkAttrs{Name: name}}, nil
}

func (s *stubNetLink) EnsureBridgeLink() error {
	s.BrLink = &vishnetlink.Bridge{LinkAttrs: vishnetlink.LinkAttrs{Name: "br0", HardwareAddr: testMACAddress}}
	return nil
}

func (s *stubNetLink) EnsureDummyLink() error {
	s.DummyLink = &vishnetlink.Dummy{LinkAttrs: vishnetlink.LinkAttrs{Name: "eth0", HardwareAddr: testMACAddress}}
	return nil
}

func (s *stubNetLink) EnsureTapLink() error {
	s.TapLink = &vishnetlink.Tuntap{LinkAttrs: vishnetlink.LinkAttrs{Name: "tap0", HardwareAddr: testMACAddress}}
	return nil
}

func (s *stubNetLink) ConfigurePodNetworks() error {
	return nil
}
