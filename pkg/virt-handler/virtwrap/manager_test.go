package virtwrap

import (
	"encoding/xml"
	"github.com/golang/mock/gomock"
	"github.com/libvirt/libvirt-go"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/record"
	"kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("Manager", func() {
	var mockConn *MockConnection
	var mockDomain *MockVirDomain
	var ctrl *gomock.Controller
	var recorder *record.FakeRecorder

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockConn = NewMockConnection(ctrl)
		mockDomain = NewMockVirDomain(ctrl)
		recorder = record.NewFakeRecorder(10)
	})

	Context("on successful VM sync", func() {
		It("should define and start a new VM", func() {
			vm := newVM("testvm")
			mockConn.EXPECT().LookupDomainByName("testvm").Return(mockDomain, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
			xml, err := xml.Marshal(vm.Spec.Domain)
			Expect(err).To(BeNil())
			mockConn.EXPECT().DomainDefineXML(string(xml)).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().Create().Return(nil)
			manager, _ := NewLibvirtDomainManager(mockConn, recorder)
			err = manager.SyncVM(vm)
			Expect(err).To(BeNil())
			Expect(<-recorder.Events).To(ContainSubstring(v1.Created.String()))
			Expect(<-recorder.Events).To(ContainSubstring(v1.Started.String()))
			Expect(recorder.Events).To(BeEmpty())
		})
		It("should leave a defined and started VM alone", func() {
			mockConn.EXPECT().LookupDomainByName("testvm").Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			manager, _ := NewLibvirtDomainManager(mockConn, recorder)
			err := manager.SyncVM(newVM("testvm"))
			Expect(err).To(BeNil())
			Expect(recorder.Events).To(BeEmpty())
		})
		table.DescribeTable("should try to start a VM in state",
			func(state libvirt.DomainState) {
				mockConn.EXPECT().LookupDomainByName("testvm").Return(mockDomain, nil)
				mockDomain.EXPECT().GetState().Return(state, 1, nil)
				mockDomain.EXPECT().Create().Return(nil)
				manager, _ := NewLibvirtDomainManager(mockConn, recorder)
				err := manager.SyncVM(newVM("testvm"))
				Expect(err).To(BeNil())
				Expect(<-recorder.Events).To(ContainSubstring(v1.Started.String()))
				Expect(recorder.Events).To(BeEmpty())
			},
			table.Entry("crashed", libvirt.DOMAIN_CRASHED),
			table.Entry("shutdown", libvirt.DOMAIN_SHUTDOWN),
			table.Entry("shutoff", libvirt.DOMAIN_SHUTOFF),
			table.Entry("unknown", libvirt.DOMAIN_NOSTATE),
		)
		It("should resume a paused VM", func() {
			mockConn.EXPECT().LookupDomainByName("testvm").Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, 1, nil)
			mockDomain.EXPECT().Resume().Return(nil)
			manager, _ := NewLibvirtDomainManager(mockConn, recorder)
			err := manager.SyncVM(newVM("testvm"))
			Expect(err).To(BeNil())
			Expect(<-recorder.Events).To(ContainSubstring(v1.Resumed.String()))
			Expect(recorder.Events).To(BeEmpty())
		})
	})

	Context("on successful VM kill", func() {
		table.DescribeTable("should try to undefine a VM in state",
			func(state libvirt.DomainState) {
				mockConn.EXPECT().LookupDomainByName("testvm").Return(mockDomain, nil)
				mockDomain.EXPECT().GetState().Return(state, 1, nil)
				mockDomain.EXPECT().Undefine().Return(nil)
				manager, _ := NewLibvirtDomainManager(mockConn, recorder)
				err := manager.KillVM(newVM("testvm"))
				Expect(err).To(BeNil())
			},
			table.Entry("crashed", libvirt.DOMAIN_CRASHED),
			table.Entry("shutdown", libvirt.DOMAIN_SHUTDOWN),
			table.Entry("shutoff", libvirt.DOMAIN_SHUTOFF),
			table.Entry("unknown", libvirt.DOMAIN_NOSTATE),
		)
		table.DescribeTable("should try to destroy and undefine a VM in state",
			func(state libvirt.DomainState) {
				mockConn.EXPECT().LookupDomainByName("testvm").Return(mockDomain, nil)
				mockDomain.EXPECT().GetState().Return(state, 1, nil)
				mockDomain.EXPECT().Destroy().Return(nil)
				mockDomain.EXPECT().Undefine().Return(nil)
				manager, _ := NewLibvirtDomainManager(mockConn, recorder)
				err := manager.KillVM(newVM("testvm"))
				Expect(err).To(BeNil())
				Expect(<-recorder.Events).To(ContainSubstring(v1.Stopped.String()))
				Expect(<-recorder.Events).To(ContainSubstring(v1.Deleted.String()))
				Expect(recorder.Events).To(BeEmpty())
			},
			table.Entry("running", libvirt.DOMAIN_RUNNING),
			table.Entry("paused", libvirt.DOMAIN_PAUSED),
		)
	})

	// TODO: test error reporting on non successful VM syncs and kill attempts

	AfterEach(func() {
		ctrl.Finish()
	})
})

func newVM(name string) *v1.VM {
	return &v1.VM{
		ObjectMeta: kubev1.ObjectMeta{Name: name},
		Spec:       v1.VMSpec{Domain: v1.NewMinimalDomainSpec(name)},
	}
}
