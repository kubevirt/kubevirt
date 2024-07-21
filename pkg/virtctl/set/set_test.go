package set_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"

	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("Set command", func() {
	var vmInterface *kubecli.MockVirtualMachineInterface
	var ctrl *gomock.Controller
	const vmName = "testvm"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).AnyTimes()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	DescribeTable("Input parameters validation without VM interaction", func(flags []string, errMsg string) {
		cmd := clientcmd.NewRepeatableVirtctlCommand(append([]string{"set"}, flags...)...)
		Expect(cmd()).To(MatchError(errMsg))
	},
		Entry("with missing input parameters", []string{}, "argument validation failed"),
		Entry("with invalid CPU count", []string{vmName, "--cpu=invalid"}, "invalid argument \"invalid\" for \"--cpu\" flag: strconv.ParseUint: parsing \"invalid\": invalid syntax"),
		Entry("with negative CPU count", []string{vmName, "--cpu=-1"}, "invalid argument \"-1\" for \"--cpu\" flag: strconv.ParseUint: parsing \"-1\": invalid syntax"),
	)

	DescribeTable("Input parameters validation with VM interaction", func(flags []string, errMsg string) {
		vm := kubecli.NewMinimalVM(vmName)
		vm.ObjectMeta.Namespace = k8smetav1.NamespaceDefault
		vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{}
		vmInterface.EXPECT().Get(gomock.Any(), vmName, k8smetav1.GetOptions{}).Return(vm, nil).Times(1)

		cmd := clientcmd.NewRepeatableVirtctlCommand(append([]string{"set"}, flags...)...)
		Expect(cmd()).To(MatchError(errMsg))
	},
		Entry("with no CPU or memory specified", []string{vmName}, "at least one of --cpu or --memory must be set"),
		Entry("with invalid memory size", []string{vmName, "--memory=invalidSize"}, "invalid memory size: invalidSize"),
	)

	Context("Patch CPU and Memory", func() {
		It("should succeed with valid CPU", func() {
			vm := kubecli.NewMinimalVM(vmName)
			vm.ObjectMeta.Namespace = k8smetav1.NamespaceDefault
			vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{}
			vmInterface.EXPECT().Get(gomock.Any(), vmName, k8smetav1.GetOptions{}).Return(vm, nil).Times(1)

			expectedPatch := `[{"op":"add","path":"/spec/template/spec/domain/cpu","value":{"sockets":2}}]`
			vmInterface.EXPECT().Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}, gomock.Any()).DoAndReturn(
				func(ctx context.Context, name string, pt types.PatchType, data []byte, options k8smetav1.PatchOptions, subresources ...string) (*v1.VirtualMachine, error) {
					Expect(data).To(MatchJSON(expectedPatch))
					return vm, nil
				}).Times(1)

			cmd := clientcmd.NewRepeatableVirtctlCommand("set", vmName, "--cpu=2")
			Expect(cmd()).To(Succeed())
		})

		It("should succeed with valid Memory", func() {
			vm := kubecli.NewMinimalVM(vmName)
			vm.ObjectMeta.Namespace = k8smetav1.NamespaceDefault
			vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{}
			vmInterface.EXPECT().Get(gomock.Any(), vmName, k8smetav1.GetOptions{}).Return(vm, nil).Times(1)

			expectedPatch := `[{"op":"add","path":"/spec/template/spec/domain/memory","value":{"guest":"2Gi"}}]`
			vmInterface.EXPECT().Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}, gomock.Any()).DoAndReturn(
				func(ctx context.Context, name string, pt types.PatchType, data []byte, options k8smetav1.PatchOptions, subresources ...string) (*v1.VirtualMachine, error) {
					Expect(data).To(MatchJSON(expectedPatch))
					return vm, nil
				}).Times(1)

			cmd := clientcmd.NewRepeatableVirtctlCommand("set", vmName, "--memory=2048Mi")
			Expect(cmd()).To(Succeed())
		})

		It("should succeed with existing CPU, replacing sockets", func() {
			vm := kubecli.NewMinimalVM(vmName)
			vm.ObjectMeta.Namespace = k8smetav1.NamespaceDefault
			vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{Sockets: 1},
					},
				},
			}
			vmInterface.EXPECT().Get(gomock.Any(), vmName, k8smetav1.GetOptions{}).Return(vm, nil).Times(1)

			expectedPatch := `[{"op":"replace","path":"/spec/template/spec/domain/cpu/sockets","value":2}]`
			vmInterface.EXPECT().Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}, gomock.Any()).DoAndReturn(
				func(ctx context.Context, name string, pt types.PatchType, data []byte, options k8smetav1.PatchOptions, subresources ...string) (*v1.VirtualMachine, error) {
					Expect(data).To(MatchJSON(expectedPatch))
					return vm, nil
				}).Times(1)

			cmd := clientcmd.NewRepeatableVirtctlCommand("set", vmName, "--cpu=2")
			Expect(cmd()).To(Succeed())
		})

		It("should succeed with existing Memory, replacing guest size", func() {
			vm := kubecli.NewMinimalVM(vmName)
			vm.ObjectMeta.Namespace = k8smetav1.NamespaceDefault
			vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Memory: &v1.Memory{Guest: resource.NewQuantity(1024, resource.BinarySI)},
					},
				},
			}
			vmInterface.EXPECT().Get(gomock.Any(), vmName, k8smetav1.GetOptions{}).Return(vm, nil).Times(1)

			expectedPatch := `[{"op":"replace","path":"/spec/template/spec/domain/memory/guest","value":"2Gi"}]`
			vmInterface.EXPECT().Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}, gomock.Any()).DoAndReturn(
				func(ctx context.Context, name string, pt types.PatchType, data []byte, options k8smetav1.PatchOptions, subresources ...string) (*v1.VirtualMachine, error) {
					Expect(data).To(MatchJSON(expectedPatch))
					return vm, nil
				}).Times(1)

			cmd := clientcmd.NewRepeatableVirtctlCommand("set", vmName, "--memory=2048Mi")
			Expect(cmd()).To(Succeed())
		})
	})
})
