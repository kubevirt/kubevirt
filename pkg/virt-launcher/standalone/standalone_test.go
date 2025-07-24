package standalone_test

import (
	"bytes"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/standalone"
)

var _ = Describe("HandleStandaloneMode", func() {
	var (
		mockCtrl *gomock.Controller
		mockDM   *mockDomainManager
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockDM = &mockDomainManager{mockCtrl: mockCtrl}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("should do nothing if STANDALONE_VMI env var is not set", func() {
		os.Unsetenv("STANDALONE_VMI")
		standalone.HandleStandaloneMode(mockDM)
	})

	It("should panic on invalid JSON in STANDALONE_VMI", func() {
		os.Setenv("VMI_OBJ", "invalid json")
		defer os.Unsetenv("VMI_OBJ")

		Expect(func() {
			standalone.HandleStandaloneMode(mockDM)
		}).To(Panic())
	})

	It("should panic if SyncVMI fails", func() {
		vmiJSON := `{"apiVersion":"kubevirt.io/v1","kind":"VirtualMachineInstance","metadata":{"name":"testvmi"}}`
		os.Setenv("STANDALONE_VMI", vmiJSON)
		defer os.Unsetenv("STANDALONE_VMI")

		mockDM.EXPECT().SyncVMI(gomock.Any(), true, nil).Return(true, fmt.Errorf("sync error"))

		Expect(func() {
			standalone.HandleStandaloneMode(mockDM)
		}).To(PanicWith("sync error"))
	})

	It("should succeed with valid JSON and successful SyncVMI", func() {
		vmiJSON := `{"apiVersion":"kubevirt.io/v1","kind":"VirtualMachineInstance","metadata":{"name":"testvmi"}}`
		os.Setenv("STANDALONE_VMI", vmiJSON)
		defer os.Unsetenv("STANDALONE_VMI")

		mockDM.EXPECT().SyncVMI(gomock.Any(), true, nil).Return(true, nil)

		Expect(func() {
			standalone.HandleStandaloneMode(mockDM)
		}).NotTo(Panic())
	})
})

type mockDomainManager struct {
	mockCtrl *gomock.Controller
}

func (m *mockDomainManager) SyncVMI(vmi *v1.VirtualMachineInstance, allowEmulation bool, secretUUID *string) (bool, error) {
	return true, nil
}
