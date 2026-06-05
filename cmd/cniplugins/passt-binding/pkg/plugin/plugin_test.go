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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package plugin_test

import (
	"bytes"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/plugins/pkg/ns"

	"kubevirt.io/kubevirt/cmd/cniplugins/passt-binding/pkg/plugin"
)

const (
	testNSPath = "/test/ns"
	cniVersion = `{"cniVersion":"1.0.0"}`
)

var _ = Describe("passt-binding-plugin", func() {
	Context("Add", func() {
		It("succeeds", func() {
			cmd := plugin.NewCmd(stubNetNS{}, stubSysCtl{})
			args := &skel.CmdArgs{
				ContainerID: "123456789",
				Netns:       testNSPath,
				IfName:      "",
				StdinData:   []byte(cniVersion),
			}
			result, err := cmd.CmdAddResult(args)
			Expect(err).NotTo(HaveOccurred())

			versionedResult, err := result.GetAsVersion(result.Version())
			Expect(err).NotTo(HaveOccurred())

			var buf bytes.Buffer
			Expect(versionedResult.PrintTo(&buf)).To(Succeed())
			Expect(buf.String()).To(MatchJSON(cniVersion))
		})

		unprivPortErr := errors.New("unpriv port")
		pingGroupErr := errors.New("ping group")

		DescribeTable("fails to", func(sysCtl stubSysCtl, expectedErr error) {
			cmd := plugin.NewCmd(stubNetNS{}, sysCtl)
			args := &skel.CmdArgs{
				ContainerID: "123456789",
				Netns:       testNSPath,
				IfName:      "",
				StdinData:   []byte(cniVersion),
			}
			_, err := cmd.CmdAddResult(args)
			Expect(err).To(MatchError(expectedErr))
		},
			Entry("set unprivileged port", stubSysCtl{unprivPortErr: unprivPortErr}, unprivPortErr),
			Entry("set ping port", stubSysCtl{pingGroupErr: pingGroupErr}, pingGroupErr),
		)
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
