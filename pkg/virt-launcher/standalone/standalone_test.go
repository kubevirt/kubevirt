package standalone_test

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"kubevirt.io/kubevirt/pkg/virt-launcher/standalone"
	virtwrap "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	// Import for DomainSpec
)

var _ = Describe("HandleStandaloneMode", func() {
	var (
		mockCtrl *gomock.Controller
		mockDM   *virtwrap.MockDomainManager
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockDM = virtwrap.NewMockDomainManager(mockCtrl)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("should do nothing if STANDALONE_VMI env var is not set", func() {
		os.Unsetenv("STANDALONE_VMI")
		standalone.HandleStandaloneMode(mockDM)
	})

	It("should panic on invalid JSON in STANDALONE_VMI", func() {
		os.Setenv("STANDALONE_VMI", "invalid json")
		defer os.Unsetenv("STANDALONE_VMI")

		Expect(func() {
			standalone.HandleStandaloneMode(mockDM)
		}).To(Panic())
	})

	It("should panic if SyncVMI fails", func() {
		vmiJSON := `{"apiVersion":"kubevirt.io/v1","kind":"VirtualMachineInstance","metadata":{"name":"testvmi"}}`
		os.Setenv("STANDALONE_VMI", vmiJSON)
		defer os.Unsetenv("STANDALONE_VMI")

		mockDM.EXPECT().SyncVMI(gomock.Any(), true, nil).Return(nil, fmt.Errorf("sync error"))

		Expect(func() {
			standalone.HandleStandaloneMode(mockDM)
		}).To(PanicWith(MatchError(ContainSubstring("sync error"))))
	})

	It("should succeed with valid JSON and successful SyncVMI", func() {
		vmiJSON := `{"apiVersion":"kubevirt.io/v1","kind":"VirtualMachineInstance","metadata":{"name":"testvmi"}}`
		os.Setenv("STANDALONE_VMI", vmiJSON)
		defer os.Unsetenv("STANDALONE_VMI")

		mockDM.EXPECT().SyncVMI(gomock.Any(), true, nil).Return(nil, nil)

		Expect(func() {
			standalone.HandleStandaloneMode(mockDM)
		}).NotTo(Panic())
	})

	It("should succeed with valid YAML and successful SyncVMI", func() {
		vmiYAML := `apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  name: testvmi-yaml`
		os.Setenv("STANDALONE_VMI", vmiYAML)
		defer os.Unsetenv("STANDALONE_VMI")

		mockDM.EXPECT().SyncVMI(gomock.Any(), true, nil).Return(nil, nil)

		Expect(func() {
			standalone.HandleStandaloneMode(mockDM)
		}).NotTo(Panic())
	})

	It("should panic on invalid YAML in STANDALONE_VMI", func() {
		os.Setenv("STANDALONE_VMI", "invalid: yaml: here")
		defer os.Unsetenv("STANDALONE_VMI")

		Expect(func() {
			standalone.HandleStandaloneMode(mockDM)
		}).To(Panic())
	})
})
