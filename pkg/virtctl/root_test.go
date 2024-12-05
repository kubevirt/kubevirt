package virtctl_test

import (
	"fmt"
	"runtime"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/spf13/pflag"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/version"

	"kubevirt.io/kubevirt/pkg/virtctl"
)

var _ = Describe("Root", func() {

	It("returns virtctl", func() {
		Expect(virtctl.GetProgramName("virtctl")).To(BeEquivalentTo("virtctl"))
	})

	It("returns virtctl as default", func() {
		Expect(virtctl.GetProgramName("42")).To(BeEquivalentTo("virtctl"))
	})

	It("returns kubectl", func() {
		Expect(virtctl.GetProgramName("kubectl-virt")).To(BeEquivalentTo("kubectl virt"))
	})

	It("returns oc", func() {
		Expect(virtctl.GetProgramName("oc-virt")).To(BeEquivalentTo("oc virt"))
	})

	It("CheckClientServerVersion should print a message if server and client virtctl versions are different", func() {
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
		clientConfig := kubecli.DefaultClientConfig(&pflag.FlagSet{})
		Expect(virtctl.CheckClientServerVersion(clientConfig)).To(MatchError(ContainSubstring("You are using a client virtctl version that is different from the KubeVirt version running in the cluster")))
	})
})
