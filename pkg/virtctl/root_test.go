package virtctl_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

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

	DescribeTable("should pass client flags to the SSH proxy command", func(command string, vsock bool, credential string) {
		if runtime.GOOS == "windows" {
			Skip("requires a POSIX shell")
		}
		dir := GinkgoT().TempDir()
		capture := filepath.Join(dir, "proxy-args")
		envCapture := filepath.Join(dir, "proxy-env")
		recorder := filepath.Join(dir, "virtctl-recorder")
		Expect(os.WriteFile(recorder, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$PROXY_ARGS\"\nprintf '%s' \"$VIRTCTL_PROXY_FLAGS\" > \"$PROXY_ENV\"\n"), 0o700)).To(Succeed())
		// Execute the ProxyCommand just as the local SSH client would, without
		// requiring a running VM or making any API requests.
		client := []byte("#!/bin/sh\neval \"${2#ProxyCommand=}\"\n")
		Expect(os.WriteFile(filepath.Join(dir, command), client, 0o700)).To(Succeed())
		GinkgoT().Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
		GinkgoT().Setenv("PROXY_ARGS", capture)
		GinkgoT().Setenv("PROXY_ENV", envCapture)
		GinkgoT().Setenv("VIRTCTL_PROXY_FLAGS", `["--token=stale"]`)
		originalArg := os.Args[0]
		os.Args[0] = recorder
		DeferCleanup(func() { os.Args[0] = originalArg })

		config := filepath.Join(dir, "kube config's file")
		Expect(os.WriteFile(config, []byte(`apiVersion: v1
kind: Config
current-context: default
clusters:
- name: test
  cluster:
    server: https://selected.example.invalid
contexts:
- name: default
  context:
    cluster: test
    namespace: default
    user: test
- name: selected
  context:
    cluster: test
    namespace: selected-namespace
    user: test
users:
- name: test
  user:
    token: test-token
`), 0o600)).To(Succeed())
		if credential == "password" {
			data, err := os.ReadFile(config)
			Expect(err).ToNot(HaveOccurred())
			Expect(os.WriteFile(config, bytes.ReplaceAll(data, []byte("token: test-token"), []byte("username: test-user")), 0o600)).To(Succeed())
		}
		credentialFlag := "--" + credential + "=example's value"
		args := []string{"--insecure-skip-tls-verify=false", command, "--kubeconfig=" + config, "--context=selected", credentialFlag,
			"--as=test-user", "--as-group=one,two", "--as-group=three"}
		if vsock {
			args = append(args, "--vsock")
		}
		if command == "scp" {
			args = append(args, "local-file", "vmi/test-vmi:remote-file")
		} else {
			args = append(args, "vmi/test-vmi")
		}
		cmd := virtctl.NewVirtctlCommand()
		cmd.SetArgs(args)
		Expect(cmd.Execute()).To(Succeed())
		output, err := os.ReadFile(capture)
		Expect(err).ToNot(HaveOccurred())
		payload, err := os.ReadFile(envCapture)
		Expect(err).ToNot(HaveOccurred())
		var flags []string
		Expect(json.Unmarshal(payload, &flags)).To(Succeed())
		Expect(flags).To(ContainElements(credentialFlag, "--insecure-skip-tls-verify=false", "--as-group=one,two", "--as-group=three", "--context=selected", "--kubeconfig="+config))
		Expect(flags).NotTo(ContainElement("--token=stale"))
		Expect(string(output)).NotTo(ContainSubstring(credentialFlag))
		Expect(strings.Join(flags, "\n")).To(ContainSubstring("--as-group=one,two\n--as-group=three"))
		Expect(string(output)).To(ContainSubstring("vmi/test-vmi/selected-namespace\n22\n"))
		if vsock {
			Expect(string(output)).To(ContainSubstring("--vsock\n--vsock-tls=false\n"))
		}
	},
		Entry("for SSH over TCP with token", "ssh", false, "token"),
		Entry("for SSH over VSOCK with token", "ssh", true, "token"),
		Entry("for SCP over TCP with token", "scp", false, "token"),
		Entry("for SCP over VSOCK with token", "scp", true, "token"),
		Entry("for SSH over TCP with password", "ssh", false, "password"),
		Entry("for SSH over VSOCK with password", "ssh", true, "password"),
		Entry("for SCP over TCP with password", "scp", false, "password"),
		Entry("for SCP over VSOCK with password", "scp", true, "password"),
	)

	It("Execute should print a message if and error occurred and server and client virtctl versions are different", func() {
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
