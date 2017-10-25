package network

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
	"kubevirt.io/kubevirt/pkg/api/v1"
)

type MockNetworkInterface struct {
	/*SetInterfaceAttributes(mac string, ip string, device string) error
	Plug(domainManager virtwrap.DomainManager) error
	Unplug(domainManager virtwrap.DomainManager) error
	GetConfig() (*v1.Interface, error)
	DecorateInterfaceMetadata() *v1.MetadataDevice*/

	failSetInterfaceAttributes bool
	failPlug bool
	failUnplug bool
	failGetConfig bool
}

func (vif MockNetworkInterface) SetInterfaceAttributes(mac string, ip string, device string) error {
	if vif.failSetInterfaceAttributes {
		return fmt.Errorf("simulated SetInterfaceAttributes failure")
	}
	return nil
}

func (vif MockNetworkInterface) Plug(domainManager virtwrap.DomainManager) error {
	if vif.failPlug {
		return fmt.Errorf("simulated Plug failure")
	}
	return nil
}

func (vif MockNetworkInterface) Unplug(domainManager virtwrap.DomainManager) error {
	if vif.failUnplug {
		return fmt.Errorf("simulated Unplug failure")
	}
	return nil
}

func (vif MockNetworkInterface) GetConfig() (*v1.Interface, error) {
	if vif.failGetConfig {
		return nil, fmt.Errorf("simulated GetConfig failure")
	}
	return &v1.Interface{}, nil
}

func (vif MockNetworkInterface) DecorateInterfaceMetadata() *v1.MetadataDevice {
	return &v1.MetadataDevice{}
}

func makeMockInterface(failSetInterfaceAttributes bool, failPlug bool, failUnplug bool, failGetConfig bool) interfaceFunc {
	getMockInterface := func(objName string) (VirtualInterface, error) {
		res := MockNetworkInterface{failSetInterfaceAttributes: failSetInterfaceAttributes,
			failPlug:failPlug,
			failUnplug: failUnplug,
			failGetConfig: failGetConfig,
		}
		return res, nil
	}
	return getMockInterface
}

var _ = Describe("Virt Handler Network", func() {
	Context("Meta tests", func() {
		It("should create an interface that doesn't fail", func() {
			getInterface := makeMockInterface(false, false, false,false)
			iface, _ := getInterface("")
			err := iface.SetInterfaceAttributes("", "", "")
			Expect(err).ToNot(HaveOccurred())

			err = iface.Plug(&virtwrap.LibvirtDomainManager{})
			Expect(err).ToNot(HaveOccurred())

			err = iface.Unplug(&virtwrap.LibvirtDomainManager{})
			Expect(err).ToNot(HaveOccurred())

			_, err = iface.GetConfig()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create an interface that fails", func() {
			getInterface := makeMockInterface(true, true, true, true)
			iface, _ := getInterface("")
			err := iface.SetInterfaceAttributes("", "", "")
			Expect(err).To(HaveOccurred())

			err = iface.Plug(&virtwrap.LibvirtDomainManager{})
			Expect(err).To(HaveOccurred())

			err = iface.Unplug(&virtwrap.LibvirtDomainManager{})
			Expect(err).To(HaveOccurred())

			_, err = iface.GetConfig()
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Unplug tests", func() {
		var vm v1.VirtualMachine
		var domainManager virtwrap.LibvirtDomainManager

		BeforeEach(func() {
			domainManager = virtwrap.LibvirtDomainManager{}

			vm = v1.VirtualMachine{}
			vm.Spec.Domain = &v1.DomainSpec{}
			vm.Spec.Domain.Metadata = &v1.Metadata{}
			vm.Spec.Domain.Metadata.Interfaces.Devices = []v1.MetadataDevice{v1.MetadataDevice{}}
		})

		It("Should not panic when called with incomplete metadata", func() {
			vm := v1.VirtualMachine{}
			getInterface := makeMockInterface(false, false, false, false)
			err := _unPlugNetworkDevices(getInterface, &vm, &domainManager)
			Expect(err).ToNot(HaveOccurred())

			// try again with a domain (but don't stock metadata)
			vm.Spec.Domain = &v1.DomainSpec{}
			err = _unPlugNetworkDevices(getInterface, &vm, &domainManager)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should work if no underlying errors occured", func() {
			getInterface := makeMockInterface(false, false, false, false)

			err := _unPlugNetworkDevices(getInterface, &vm, &domainManager)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should return an error if SetInterfaceAttributes fails", func() {
			getInterface := makeMockInterface(true, false, false, false)

			err := _unPlugNetworkDevices(getInterface, &vm, &domainManager)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should not return an error if Unplug fails", func() {
			getInterface := makeMockInterface(false, false, true, false)

			err := _unPlugNetworkDevices(getInterface, &vm, &domainManager)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Plug Tests", func() {
		var vm v1.VirtualMachine
		var domainManager virtwrap.LibvirtDomainManager

		BeforeEach(func() {
			domainManager = virtwrap.LibvirtDomainManager{}

			vm = v1.VirtualMachine{}
			vm.Spec.Domain = &v1.DomainSpec{}
			vm.Spec.Domain.Metadata = &v1.Metadata{}
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{}}
		})

		It("Should not error", func(){
			getInterface := makeMockInterface(false, false, false, false)

			vmCopy, err := _plugNetworkDevices(getInterface, &vm, &domainManager)
			Expect(err).ToNot(HaveOccurred())
			Expect(vmCopy.Spec.Domain.Metadata).ToNot(BeNil())
		})

		It("Should fail if Plug() fails", func(){
			getInterface := makeMockInterface(false, true, false, false)

			vmCopy, err := _plugNetworkDevices(getInterface, &vm, &domainManager)
			Expect(err).To(HaveOccurred())
			Expect(vmCopy).To(BeNil())
			Expect(err.Error()).To(Equal("simulated Plug failure"))
		})

		It("Should fail if GetConfig() fails", func(){
			getInterface := makeMockInterface(false, false, false, true)

			vmCopy, err := _plugNetworkDevices(getInterface, &vm, &domainManager)
			Expect(err).To(HaveOccurred())
			Expect(vmCopy).To(BeNil())
			Expect(err.Error()).To(Equal("simulated GetConfig failure"))
		})
	})
})


func TestNetwork(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Virt Handler Network")
}