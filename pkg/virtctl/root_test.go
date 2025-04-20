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

package virtctl_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/version"

	"kubevirt.io/kubevirt/pkg/virtctl"
	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

var _ = Describe("virtctl", func() {
	DescribeTable("GetProgramName", func(binary, expected string) {
		Expect(virtctl.GetProgramName(binary)).To(Equal(expected))
	},
		Entry("returns virtctl", "virtctl", "virtctl"),
		Entry("returns virtctl as default", "42", "virtctl"),
		Entry("returns kubectl", "kubectl-virt", "kubectl virt"),
		Entry("returns oc", "oc-virt", "oc virt"),
	)

	DescribeTable("the log verbosity flag should be supported", func(arg string) {
		Expect(testing.NewRepeatableVirtctlCommand(arg)()).To(Succeed())
	},
		Entry("regular flag", "--v=2"),
		Entry("shorthand flag", "-v=2"),
	)

	It("Execute should print a message if and error occured and server and client virtctl versions are different", func() {
		ctrl := gomock.NewController(GinkgoT())
		serverVersionInterface := kubecli.NewMockServerVersionInterface(ctrl)
		serverVersionInterface.EXPECT().Get().Return(&version.Info{
			GitVersion:   "v0.46.1",
			GitCommit:    "fda30004223b51f9e604276419a2b376652cb5ad",
			GitTreeState: "clear",
			BuildDate:    time.Now().Format("%Y-%m-%dT%H:%M:%SZ"),
			GoVersion:    runtime.Version(),
			Compiler:     runtime.Compiler,
			Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		}, nil,
		).AnyTimes()
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		kubecli.MockKubevirtClientInstance.EXPECT().ServerVersion().Return(serverVersionInterface).AnyTimes()

		const testError = "testError"
		cmd := &cobra.Command{
			RunE: func(_ *cobra.Command, _ []string) error {
				return errors.New(testError)
			},
		}
		out := &bytes.Buffer{}
		cmd.SetErr(out)
		cmd.SetContext(clientconfig.NewContext(
			context.Background(), kubecli.DefaultClientConfig(&pflag.FlagSet{}),
		))

		virtctl.NewVirtctlCommand = func() *cobra.Command {
			return cmd
		}
		DeferCleanup(func() {
			virtctl.NewVirtctlCommand = virtctl.NewVirtctlCommandFn
		})

		Expect(virtctl.Execute()).To(Equal(1))
		Expect(out.String()).To(ContainSubstring(testError))
		Expect(out.String()).To(ContainSubstring("You are using a client virtctl version that is different from the KubeVirt version running in the cluster"))
	})
})
