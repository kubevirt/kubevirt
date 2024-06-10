package version_test

import (
	"bytes"
	"fmt"
	goruntime "runtime"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/kubecli"
	virt_version "kubevirt.io/client-go/version"

	"kubevirt.io/kubevirt/pkg/virtctl"
	"kubevirt.io/kubevirt/pkg/virtctl/version"
)

var _ = Describe("Version", func() {

	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		serverVersionInterface := kubecli.NewMockServerVersionInterface(ctrl)
		kubecli.MockKubevirtClientInstance.EXPECT().ServerVersion().Return(serverVersionInterface).AnyTimes()
		serverVersionInterface.EXPECT().Get().Return(&virt_version.Info{
			GitVersion:   "v0.46.1",
			GitCommit:    "fda30004223b51f9e604276419a2b376652cb5ad",
			GitTreeState: "clear",
			BuildDate:    time.Now().Format("%Y-%m-%dT%H:%M:%SZ"),
			GoVersion:    goruntime.Version(),
			Compiler:     goruntime.Compiler,
			Platform:     fmt.Sprintf("%s/%s", goruntime.GOOS, goruntime.GOARCH),
		}, nil,
		).AnyTimes()
	})

	Context("should print a message if server and client virtctl versions are different", func() {
		It("in version command", func() {
			// Skip on s390x, since it uses go-test, which does not change the version variables during the compile step,
			// causing unintended behaviour of the function
			// TODO: Remove when switching to bazel-test
			if goruntime.GOARCH == "s390x" {
				Skip("Skip version when invoking via go-test")
			}

			var buf bytes.Buffer
			cmd, clientConfig := virtctl.NewVirtctlCommand()
			cmd.SetOut(&buf)
			version.CheckClientServerVersion(&clientConfig)
			//Print out the captured output to show the test output also in the console
			fmt.Printf(buf.String())
			Expect(buf.String()).To(ContainSubstring("You are using a client virtctl version that is different from the KubeVirt version running in the cluster"),
				"Warning message was not shown or has been changed")
		})

	})
})
