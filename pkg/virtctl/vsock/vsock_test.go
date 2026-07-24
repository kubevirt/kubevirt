package vsock_test

import (
	"fmt"
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"

	"kubevirt.io/kubevirt/pkg/virtctl/testing"
	"kubevirt.io/kubevirt/pkg/virtctl/vsock"
)

type fakeStreamer struct {
	streamErr error
}

func (f *fakeStreamer) Stream(_ kvcorev1.StreamOptions) error { return f.streamErr }
func (f *fakeStreamer) AsConn() net.Conn                      { return nil }

var _ = Describe("Vsock", func() {
	var ctrl *gomock.Controller
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
	})

	It("NewCommand creates vsock command", func() {
		cmd := vsock.NewCommand()
		Expect(cmd).ToNot(BeNil())
		Expect(cmd.Use).To(ContainSubstring("vsock"))
	})

	It("NewCommand has tls flag", func() {
		cmd := vsock.NewCommand()
		Expect(cmd.Flags().Lookup("tls")).ToNot(BeNil())
	})

	It("NewCommand requires exactly 2 args", func() {
		cmd := vsock.NewCommand()
		Expect(cmd.Args(cmd, []string{})).To(MatchError(ContainSubstring("accepts 2 arg(s)")))
		Expect(cmd.Args(cmd, []string{"vmi/testvm"})).To(MatchError(ContainSubstring("accepts 2 arg(s)")))
		Expect(cmd.Args(cmd, []string{"vmi/testvm", "22", "extra"})).To(MatchError(ContainSubstring("accepts 2 arg(s)")))
	})

	DescribeTable("calls the VSOCK subresource API once the VMI is confirmed running",
		func(target, resolvedName string) {
			useTLS := true

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface)
			vmiInterface.EXPECT().Get(gomock.Any(), resolvedName, k8smetav1.GetOptions{}).Return(&v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{Phase: v1.Running},
			}, nil)
			vmiInterface.EXPECT().VSOCK(resolvedName, &v1.VSOCKOptions{
				TargetPort: 22,
				UseTLS:     &useTLS,
			}).Return(&fakeStreamer{}, nil)

			Expect(testing.NewRepeatableVirtctlCommand("vsock", target, "22")()).To(Succeed())
		},
		Entry("vmi kind", "vmi/testvmi", "testvmi"),
		Entry("vm kind", "vm/testvm", "testvm"),
	)

	DescribeTable("resolves the namespace from the target when calling the VMI client",
		func(target, resolvedName, resolvedNamespace string) {
			useTLS := true

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(resolvedNamespace).Return(vmiInterface)
			vmiInterface.EXPECT().Get(gomock.Any(), resolvedName, k8smetav1.GetOptions{}).Return(&v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{Phase: v1.Running},
			}, nil)
			vmiInterface.EXPECT().VSOCK(resolvedName, &v1.VSOCKOptions{
				TargetPort: 22,
				UseTLS:     &useTLS,
			}).Return(&fakeStreamer{}, nil)

			Expect(testing.NewRepeatableVirtctlCommand("vsock", target, "22")()).To(Succeed())
		},
		Entry("vmi kind with explicit namespace", "vmi/testvmi/mynamespace", "testvmi", "mynamespace"),
		Entry("vm kind with explicit namespace", "vm/testvm/mynamespace", "testvm", "mynamespace"),
	)

	DescribeTable("returns error when the VMI cannot be found",
		func(target string) {
			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface)
			vmiInterface.EXPECT().Get(gomock.Any(), gomock.Any(), k8smetav1.GetOptions{}).Return(nil, fmt.Errorf("not found"))

			Expect(testing.NewRepeatableVirtctlCommand("vsock", target, "22")()).
				To(MatchError(ContainSubstring("failed to find VirtualMachineInstance")))
		},
		Entry("vmi kind", "vmi/testvmi"),
		Entry("vm kind", "vm/testvm"),
	)

	DescribeTable("returns error when the VMI is not running",
		func(target string) {
			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface)
			vmiInterface.EXPECT().Get(gomock.Any(), gomock.Any(), k8smetav1.GetOptions{}).Return(&v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{Phase: v1.Scheduling},
			}, nil)

			Expect(testing.NewRepeatableVirtctlCommand("vsock", target, "22")()).
				To(MatchError(ContainSubstring("is not running (phase: Scheduling)")))
		},
		Entry("vmi kind", "vmi/testvmi"),
		Entry("vm kind", "vm/testvm"),
	)

	DescribeTable("returns error when the target port is invalid",
		func(port string) {
			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface)
			vmiInterface.EXPECT().Get(gomock.Any(), "testvmi", k8smetav1.GetOptions{}).Return(&v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{Phase: v1.Running},
			}, nil)

			Expect(testing.NewRepeatableVirtctlCommand("vsock", "vmi/testvmi", port)()).
				To(MatchError(ContainSubstring(fmt.Sprintf("invalid port %q", port))))
		},
		Entry("non-numeric port", "abc"),
		Entry("empty port", ""),
		Entry("port overflowing uint16", "65536"),
	)

	It("returns error when the VSOCK subresource call fails", func() {
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface)
		vmiInterface.EXPECT().Get(gomock.Any(), "testvmi", k8smetav1.GetOptions{}).Return(&v1.VirtualMachineInstance{
			Status: v1.VirtualMachineInstanceStatus{Phase: v1.Running},
		}, nil)
		vmiInterface.EXPECT().VSOCK("testvmi", gomock.Any()).Return(nil, fmt.Errorf("vsock subresource failed"))

		Expect(testing.NewRepeatableVirtctlCommand("vsock", "vmi/testvmi", "22")()).
			To(MatchError(ContainSubstring("vsock subresource failed")))
	})

	It("returns error when the VSOCK stream fails", func() {
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface)
		vmiInterface.EXPECT().Get(gomock.Any(), "testvmi", k8smetav1.GetOptions{}).Return(&v1.VirtualMachineInstance{
			Status: v1.VirtualMachineInstanceStatus{Phase: v1.Running},
		}, nil)
		vmiInterface.EXPECT().VSOCK("testvmi", gomock.Any()).
			Return(&fakeStreamer{streamErr: fmt.Errorf("streaming failed")}, nil)

		Expect(testing.NewRepeatableVirtctlCommand("vsock", "vmi/testvmi", "22")()).
			To(MatchError(ContainSubstring("streaming failed")))
	})
})
