package offlinevm

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"github.com/golang/mock/gomock"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("OfflineVirtualMachine", func() {

	var ovmInterface *kubecli.MockOfflineVirtualMachineInterface
	var ctrl = gomock.NewController(GinkgoT())

	Context("OfflineVirtualMachine command invocation", func() {
		var commandName string
		var cmd *Command

		BeforeEach(func() {
			commandName = "test"
			cmd = NewCommand(commandName)
		})

		It("should create commands based on given verb", func() {
			Expect(cmd.command).To(Equal(commandName))
		})

		It("should patch OVM with spec:running:true", func() {
			ovm := NewMinimalOVM("testvm")
			ovm.Spec.Running = true
			ovmInterface = kubecli.NewMockOfflineVirtualMachineInterface(ctrl)
			ovmInterface.EXPECT().Patch(ovm.Name, types.MergePatchType,	[]byte("{\"spec\":{\"running\":true}}")).Return(ovm, nil)
			Expect(ovm.Spec.Running).To(Equal(true))
		})
	})
})

func TestOVM(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OfflineVM")
}

func NewMinimalOVM(name string) *v1.OfflineVirtualMachine {
	return &v1.OfflineVirtualMachine{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "OfflineVirtualMachine"}, ObjectMeta: k8smetav1.ObjectMeta{Name: name}}
}
