package converter

import (
	"os"
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type devicePlacementTestCase struct {
	name                  string
	numaCells             []api.NUMACell
	vcpuPins              []api.CPUTuneVCPUPin
	devices               []api.HostDevice
	expectedControllers   int
	expectedExpanderBuses int
	expectedRootPorts     int
	expectedError         string
}

type addDevicesTestCase struct {
	name            string
	devices         []api.HostDevice
	numaCells       []api.NUMACell
	vcpuPins        []api.CPUTuneVCPUPin
	expectedDevices int
	description     string
}

var _ = Describe("PCIe Expander Bus Assigner", func() {
	var (
		originalPciBasePath  string
		originalNodeBasePath string
		fakePciBasePath      string
		fakeNodeBasePath     string
	)

	createDomainSpecWithNUMA := func(numaCells []api.NUMACell, vcpuPins []api.CPUTuneVCPUPin) *api.DomainSpec {
		spec := &api.DomainSpec{
			Devices: api.Devices{
				Controllers: []api.Controller{},
			},
		}
		if len(numaCells) > 0 {
			spec.CPU = api.CPU{
				NUMA: &api.NUMA{Cells: numaCells},
			}
		}
		if len(vcpuPins) > 0 {
			spec.CPUTune = &api.CPUTune{VCPUPin: vcpuPins}
		}
		return spec
	}

	createPCIDevice := func(alias, bus string) api.HostDevice {
		return api.HostDevice{
			Type:  api.HostDevicePCI,
			Alias: api.NewUserDefinedAlias(alias),
			Source: api.HostDeviceSource{
				Address: &api.Address{
					Domain: "0x0000", Bus: bus,
					Slot: "0x00", Function: "0x0",
				},
			},
		}
	}

	createPCIDeviceWithGuestAddress := func(alias, hostBus, guestBus, guestSlot string) api.HostDevice {
		device := createPCIDevice(alias, hostBus)
		device.Address = newPCIAddress(guestBus, guestSlot)
		return device
	}

	createPCIDeviceWithPartialGuestAddress := func(alias, hostBus string) api.HostDevice {
		device := createPCIDevice(alias, hostBus)
		device.Address = &api.Address{Type: api.AddressPCI, Bus: "0x00"}
		return device
	}

	createNonPCIDevice := func(deviceType string) api.HostDevice {
		return api.HostDevice{
			Type: deviceType,
		}
	}

	createPCIDeviceWithoutAddress := func(alias string) api.HostDevice {
		return api.HostDevice{
			Type:  api.HostDevicePCI,
			Alias: api.NewUserDefinedAlias(alias),
		}
	}

	setupFakeSysfs := func() {
		var err error
		fakePciBasePath, err = os.MkdirTemp("", "pci_devices")
		Expect(err).ToNot(HaveOccurred())

		fakeNodeBasePath, err = os.MkdirTemp("", "numa_nodes")
		Expect(err).ToNot(HaveOccurred())

		// Create test PCI devices with NUMA nodes
		testDevices := map[string]string{
			"0000:01:00.0": "0",
			"0000:02:00.0": "1",
			"0000:03:00.0": "0",
			"0000:04:00.0": "1",
			"0000:05:00.0": "0",
		}

		for pciAddr, numaNode := range testDevices {
			pciDevicePath := filepath.Join(fakePciBasePath, pciAddr)
			err = os.MkdirAll(pciDevicePath, 0o755)
			Expect(err).ToNot(HaveOccurred())

			numaNodeFile := filepath.Join(pciDevicePath, "numa_node")
			err = os.WriteFile(numaNodeFile, []byte(numaNode+"\n"), 0o644)
			Expect(err).ToNot(HaveOccurred())
		}

		// Create NUMA node directories
		for numaID, cpuList := range map[string]string{"0": "0-3", "1": "4-7"} {
			numaNodePath := filepath.Join(fakeNodeBasePath, "node"+numaID)
			err = os.MkdirAll(numaNodePath, 0o755)
			Expect(err).ToNot(HaveOccurred())

			cpuListFile := filepath.Join(numaNodePath, "cpulist")
			err = os.WriteFile(cpuListFile, []byte(cpuList+"\n"), 0o644)
			Expect(err).ToNot(HaveOccurred())
		}
	}

	BeforeEach(func() {
		originalPciBasePath = hardware.PciBasePath
		originalNodeBasePath = hardware.NodeBasePath
		setupFakeSysfs()
		hardware.PciBasePath = fakePciBasePath
		hardware.NodeBasePath = fakeNodeBasePath
	})

	AfterEach(func() {
		hardware.PciBasePath = originalPciBasePath
		hardware.NodeBasePath = originalNodeBasePath
		if fakePciBasePath != "" {
			os.RemoveAll(fakePciBasePath)
		}
		if fakeNodeBasePath != "" {
			os.RemoveAll(fakeNodeBasePath)
		}
	})

	Describe("getCurrentControllerIndex", func() {
		It("should return the highest index of the existing controllers", func() {
			domainSpec := &api.DomainSpec{
				Devices: api.Devices{
					Controllers: []api.Controller{
						{Model: api.ControllerModelPCIeRoot, Index: "0"},
						{Model: api.ControllerModelPCIeRootPort, Index: "4"},
					},
				},
			}

			Expect(getCurrentControllerIndex(domainSpec)).To(Equal(uint32(4)))
		})
	})

	Describe("expanderBusAssigner", func() {
		var (
			assigner   *expanderBusAssigner
			domainSpec *api.DomainSpec
		)

		BeforeEach(func() {
			domainSpec = createDomainSpecWithNUMA(
				[]api.NUMACell{{ID: "0", CPUs: "0-1"}, {ID: "1", CPUs: "2-3"}},
				[]api.CPUTuneVCPUPin{{VCPU: 0, CPUSet: "0"}, {VCPU: 2, CPUSet: "4"}},
			)
			assigner = newExpanderBusAssigner(domainSpec)
		})

		DescribeTable("addDevices",
			func(testCase addDevicesTestCase) {
				if testCase.numaCells != nil || testCase.vcpuPins != nil {
					domainSpec = createDomainSpecWithNUMA(testCase.numaCells, testCase.vcpuPins)
					assigner = newExpanderBusAssigner(domainSpec)
				}

				assigner.addDevices(testCase.devices)
				Expect(assigner.devices).To(HaveLen(testCase.expectedDevices), testCase.description)
			},
			Entry("filters non-PCI devices", addDevicesTestCase{
				name: "mixed device types",
				devices: []api.HostDevice{
					createPCIDevice("pci1", "0x01"),
					createNonPCIDevice("usb"),
					createPCIDevice("pci2", "0x02"),
					createNonPCIDevice("scsi"),
				},
				expectedDevices: 2,
				description:     "should only accept PCI devices",
			}),
			Entry("filters devices without source address", addDevicesTestCase{
				name: "devices without address",
				devices: []api.HostDevice{
					createPCIDevice("pci1", "0x01"),
					createPCIDeviceWithoutAddress("pci2"),
				},
				expectedDevices: 1,
				description:     "should skip devices without source address",
			}),
			Entry("filters devices with existing guest PCI addresses", addDevicesTestCase{
				name: "devices with guest PCI addresses",
				devices: []api.HostDevice{
					createPCIDevice("pci1", "0x01"),
					createPCIDeviceWithGuestAddress("pci2", "0x02", "0x00", "0x07"),
				},
				expectedDevices: 1,
				description:     "should skip devices that already have a guest PCI address",
			}),
			Entry("filters devices with partial guest PCI addresses", addDevicesTestCase{
				name: "devices with partial guest PCI addresses",
				devices: []api.HostDevice{
					createPCIDevice("pci1", "0x01"),
					createPCIDeviceWithPartialGuestAddress("pci2", "0x02"),
				},
				expectedDevices: 1,
				description:     "should skip devices that already have any guest PCI coordinate",
			}),
			Entry("filters devices without NUMA affinity", addDevicesTestCase{
				name:            "devices without NUMA topology",
				devices:         []api.HostDevice{createPCIDevice("pci1", "0x01")},
				numaCells:       []api.NUMACell{},
				vcpuPins:        []api.CPUTuneVCPUPin{},
				expectedDevices: 0,
				description:     "should skip devices when no NUMA topology is configured",
			}),
			Entry("accepts PCI devices with NUMA affinity", addDevicesTestCase{
				name: "valid PCI devices",
				devices: []api.HostDevice{
					createPCIDevice("device1", "0x01"),
					createPCIDevice("device2", "0x02"),
				},
				expectedDevices: 2,
				description:     "should accept all valid PCI devices with NUMA affinity",
			}),
		)

		DescribeTable("PlaceNumaAlignedDevices",
			func(testCase devicePlacementTestCase) {
				if testCase.numaCells != nil || testCase.vcpuPins != nil {
					domainSpec = createDomainSpecWithNUMA(testCase.numaCells, testCase.vcpuPins)
					assigner = newExpanderBusAssigner(domainSpec)
				}

				domainSpec.Devices.HostDevices = testCase.devices

				err := assigner.PlaceNumaAlignedDevices()

				if testCase.expectedError != "" {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(testCase.expectedError))
				} else {
					Expect(err).ToNot(HaveOccurred())
				}

				if testCase.expectedControllers >= 0 {
					Expect(domainSpec.Devices.Controllers).To(HaveLen(testCase.expectedControllers))
				}

				if testCase.expectedExpanderBuses > 0 {
					expanderBusCount := 0
					for _, controller := range domainSpec.Devices.Controllers {
						if controller.Model == api.ControllerModelPCIeExpanderBus {
							expanderBusCount++
						}
					}
					Expect(expanderBusCount).To(Equal(testCase.expectedExpanderBuses))
				}

				if testCase.expectedRootPorts > 0 {
					rootPortCount := 0
					for _, controller := range domainSpec.Devices.Controllers {
						if controller.Model == api.ControllerModelPCIeRootPort {
							rootPortCount++
						}
					}
					Expect(rootPortCount).To(Equal(testCase.expectedRootPorts))
				}
			},
			Entry("handles empty device list", devicePlacementTestCase{
				name:                "no devices",
				devices:             []api.HostDevice{},
				expectedControllers: 0,
			}),
			Entry("places single device on single NUMA node", devicePlacementTestCase{
				name:                  "single device",
				devices:               []api.HostDevice{createPCIDevice("device1", "0x01")},
				expectedControllers:   2,
				expectedExpanderBuses: 1,
				expectedRootPorts:     1,
			}),
			Entry("places multiple devices on same NUMA node", devicePlacementTestCase{
				name: "multiple devices same NUMA",
				devices: []api.HostDevice{
					createPCIDevice("device1", "0x01"),
					createPCIDevice("device2", "0x03"),
				},
				expectedControllers:   3,
				expectedExpanderBuses: 1,
				expectedRootPorts:     2,
			}),
			Entry("places devices on different NUMA nodes", devicePlacementTestCase{
				name: "devices on different NUMA nodes",
				devices: []api.HostDevice{
					createPCIDevice("device_numa0", "0x01"),
					createPCIDevice("device_numa1", "0x02"),
				},
				expectedControllers:   4,
				expectedExpanderBuses: 2,
				expectedRootPorts:     2,
			}),
			Entry("handles domain spec without NUMA topology", devicePlacementTestCase{
				name:                "no NUMA topology",
				numaCells:           []api.NUMACell{},
				vcpuPins:            []api.CPUTuneVCPUPin{},
				devices:             []api.HostDevice{createPCIDevice("device1", "0x01")},
				expectedControllers: 0,
			}),
			Entry("handles domain spec without CPU affinity", devicePlacementTestCase{
				name:                "no CPU affinity",
				numaCells:           []api.NUMACell{{ID: "0", CPUs: "0-1"}},
				vcpuPins:            []api.CPUTuneVCPUPin{},
				devices:             []api.HostDevice{createPCIDevice("device1", "0x01")},
				expectedControllers: 0,
			}),
		)
	})

	Describe("PlacePCIDevicesWithNUMAAlignment", func() {
		var domainSpec *api.DomainSpec

		BeforeEach(func() {
			domainSpec = createDomainSpecWithNUMA(
				[]api.NUMACell{{ID: "0", CPUs: "0-1"}, {ID: "1", CPUs: "2-3"}},
				[]api.CPUTuneVCPUPin{{VCPU: 0, CPUSet: "0"}, {VCPU: 2, CPUSet: "4"}},
			)
		})

		It("should return error when controller index exceeds the last expander bus number", func() {
			// Set current controller index to the maximum to trigger the validation
			domainSpec.Devices.Controllers = []api.Controller{
				{Model: api.ControllerModelPCIeRootPort, Index: strconv.Itoa(maxExpanderBusNr)},
			}

			// Add a device, this would require creating new controllers
			domainSpec.Devices.HostDevices = []api.HostDevice{createPCIDevice("device1", "0x01")}
			originalDomainSpec := domainSpec.DeepCopy()

			err := PlacePCIDevicesWithNUMAAlignment(domainSpec)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("insufficient bus numbers for NUMA-aligned PCIe topology"))
			Expect(err.Error()).To(ContainSubstring("current controller index 256"))
			Expect(err.Error()).To(ContainSubstring("last assigned expander bus number 255"))
			Expect(domainSpec).To(Equal(originalDomainSpec))
		})

		It("should assign bus numbers for expander buses calculated as maxBusNr - controllerCount + 1", func() {
			domainSpec.Devices.HostDevices = []api.HostDevice{
				createPCIDevice("device1", "0x01"),
				createPCIDevice("device2", "0x02"),
			}

			err := PlacePCIDevicesWithNUMAAlignment(domainSpec)
			Expect(err).ToNot(HaveOccurred())

			// Bus numbers calculated as 254 - controllerCount + 1:
			// NUMA 0: 255 - 2 + 1 = 254 (after creating expander bus + root port)
			// NUMA 1: 255 - 4 + 1 = 252 (after creating 2nd expander bus + root port)
			expectedBusNumbers := map[uint32]bool{254: false, 252: false}
			for _, controller := range domainSpec.Devices.Controllers {
				if controller.Model == api.ControllerModelPCIeExpanderBus {
					Expect(controller.Target).ToNot(BeNil())
					Expect(controller.Target.BusNr).ToNot(BeNil())
					busNr := *controller.Target.BusNr
					_, expected := expectedBusNumbers[busNr]
					Expect(expected).To(BeTrue(), "Bus number %d should be one of the expected values (254, 252)", busNr)
					expectedBusNumbers[busNr] = true
				}
			}

			// Ensure both expected bus numbers were assigned
			for busNr, assigned := range expectedBusNumbers {
				Expect(assigned).To(BeTrue(), "Expected bus number %d was not assigned", busNr)
			}
		})

		It("should assign devices to correct root ports", func() {
			domainSpec.Devices.HostDevices = []api.HostDevice{
				createPCIDevice("device1", "0x01"),
				createPCIDevice("device2", "0x02"),
			}

			err := PlacePCIDevicesWithNUMAAlignment(domainSpec)
			Expect(err).ToNot(HaveOccurred())

			for _, device := range domainSpec.Devices.HostDevices {
				Expect(device.Address).ToNot(BeNil())
				Expect(device.Address.Type).To(Equal(api.AddressPCI))
				Expect(device.Address.Bus).ToNot(BeEmpty())
				Expect(device.Address.Slot).To(Equal("0x00"))
			}
		})

		It("should preserve existing hostdev guest PCI addresses", func() {
			domainSpec.Devices.HostDevices = []api.HostDevice{
				createPCIDeviceWithGuestAddress("addressed", "0x01", "0x00", "0x07"),
				createPCIDevice("unaddressed", "0x02"),
			}

			err := PlacePCIDevicesWithNUMAAlignment(domainSpec)
			Expect(err).ToNot(HaveOccurred())

			Expect(domainSpec.Devices.Controllers).To(HaveLen(2))
			Expect(domainSpec.Devices.Controllers[0].Model).To(Equal(api.ControllerModelPCIeExpanderBus))
			Expect(*domainSpec.Devices.Controllers[0].Target.NUMANode).To(Equal(uint32(1)))
			Expect(domainSpec.Devices.Controllers[1].Model).To(Equal(api.ControllerModelPCIeRootPort))

			Expect(domainSpec.Devices.HostDevices[0].Address).To(Equal(newPCIAddress("0x00", "0x07")))
			Expect(domainSpec.Devices.HostDevices[1].Address).To(Equal(newPCIAddress("2", "0x00")))
		})

		It("should leave devices without host NUMA affinity for default placement", func() {
			domainSpec.Devices.HostDevices = []api.HostDevice{
				createPCIDevice("missing_numa", "0x06"),
			}

			err := PlacePCIDevicesWithNUMAAlignment(domainSpec)
			Expect(err).ToNot(HaveOccurred())

			Expect(domainSpec.Devices.Controllers).To(BeEmpty())
			Expect(domainSpec.Devices.HostDevices[0].Address).To(BeNil())
		})

		It("should place only devices with host NUMA affinity", func() {
			domainSpec.Devices.HostDevices = []api.HostDevice{
				createPCIDevice("missing_numa", "0x06"),
				createPCIDevice("numa0", "0x01"),
			}

			err := PlacePCIDevicesWithNUMAAlignment(domainSpec)
			Expect(err).ToNot(HaveOccurred())

			Expect(domainSpec.Devices.Controllers).To(HaveLen(2))
			Expect(domainSpec.Devices.Controllers[0].Model).To(Equal(api.ControllerModelPCIeExpanderBus))
			Expect(*domainSpec.Devices.Controllers[0].Target.NUMANode).To(Equal(uint32(0)))
			Expect(domainSpec.Devices.Controllers[1].Model).To(Equal(api.ControllerModelPCIeRootPort))

			Expect(domainSpec.Devices.HostDevices[0].Address).To(BeNil())
			Expect(domainSpec.Devices.HostDevices[1].Address).To(Equal(newPCIAddress("2", "0x00")))
		})

		It("should prefer NUMATune memnode mapping for guest NUMA placement", func() {
			domainSpec.NUMATune = &api.NUMATune{
				MemNodes: []api.MemNode{
					{CellID: 1, Mode: "strict", NodeSet: "0"},
				},
			}
			domainSpec.Devices.HostDevices = []api.HostDevice{
				createPCIDevice("numa0", "0x01"),
			}

			err := PlacePCIDevicesWithNUMAAlignment(domainSpec)
			Expect(err).ToNot(HaveOccurred())

			Expect(domainSpec.Devices.Controllers).To(HaveLen(2))
			Expect(domainSpec.Devices.Controllers[0].Model).To(Equal(api.ControllerModelPCIeExpanderBus))
			Expect(*domainSpec.Devices.Controllers[0].Target.NUMANode).To(Equal(uint32(1)))
			Expect(domainSpec.Devices.HostDevices[0].Address).To(Equal(newPCIAddress("2", "0x00")))
		})

		It("should fall back to vCPU pinning for ambiguous NUMATune memnode mapping", func() {
			domainSpec.NUMATune = &api.NUMATune{
				MemNodes: []api.MemNode{
					{CellID: 0, Mode: "strict", NodeSet: "0"},
					{CellID: 1, Mode: "strict", NodeSet: "0"},
				},
			}
			domainSpec.Devices.HostDevices = []api.HostDevice{
				createPCIDevice("numa0", "0x01"),
			}

			err := PlacePCIDevicesWithNUMAAlignment(domainSpec)
			Expect(err).ToNot(HaveOccurred())

			Expect(domainSpec.Devices.Controllers).To(HaveLen(2))
			Expect(domainSpec.Devices.Controllers[0].Model).To(Equal(api.ControllerModelPCIeExpanderBus))
			Expect(*domainSpec.Devices.Controllers[0].Target.NUMANode).To(Equal(uint32(0)))
			Expect(domainSpec.Devices.HostDevices[0].Address).To(Equal(newPCIAddress("2", "0x00")))
		})

		It("should place NUMA groups and devices in deterministic order", func() {
			domainSpec.Devices.HostDevices = []api.HostDevice{
				createPCIDevice("numa1_b", "0x04"),
				createPCIDevice("numa0_b", "0x03"),
				createPCIDevice("numa0_a", "0x01"),
				createPCIDevice("numa1_a", "0x02"),
			}

			err := PlacePCIDevicesWithNUMAAlignment(domainSpec)
			Expect(err).ToNot(HaveOccurred())

			Expect(domainSpec.Devices.Controllers).To(HaveLen(6))

			Expect(domainSpec.Devices.Controllers[0].Model).To(Equal(api.ControllerModelPCIeExpanderBus))
			Expect(domainSpec.Devices.Controllers[0].Index).To(Equal("1"))
			Expect(*domainSpec.Devices.Controllers[0].Target.NUMANode).To(Equal(uint32(0)))
			Expect(*domainSpec.Devices.Controllers[0].Target.BusNr).To(Equal(uint32(253)))

			Expect(domainSpec.Devices.Controllers[1].Model).To(Equal(api.ControllerModelPCIeRootPort))
			Expect(domainSpec.Devices.Controllers[1].Index).To(Equal("2"))
			Expect(domainSpec.Devices.Controllers[1].Address.Bus).To(Equal("1"))
			Expect(domainSpec.Devices.Controllers[1].Address.Slot).To(Equal("0x00"))

			Expect(domainSpec.Devices.Controllers[2].Model).To(Equal(api.ControllerModelPCIeRootPort))
			Expect(domainSpec.Devices.Controllers[2].Index).To(Equal("3"))
			Expect(domainSpec.Devices.Controllers[2].Address.Bus).To(Equal("1"))
			Expect(domainSpec.Devices.Controllers[2].Address.Slot).To(Equal("0x01"))

			Expect(domainSpec.Devices.Controllers[3].Model).To(Equal(api.ControllerModelPCIeExpanderBus))
			Expect(domainSpec.Devices.Controllers[3].Index).To(Equal("4"))
			Expect(*domainSpec.Devices.Controllers[3].Target.NUMANode).To(Equal(uint32(1)))
			Expect(*domainSpec.Devices.Controllers[3].Target.BusNr).To(Equal(uint32(250)))

			Expect(domainSpec.Devices.Controllers[4].Model).To(Equal(api.ControllerModelPCIeRootPort))
			Expect(domainSpec.Devices.Controllers[4].Index).To(Equal("5"))
			Expect(domainSpec.Devices.Controllers[4].Address.Bus).To(Equal("4"))
			Expect(domainSpec.Devices.Controllers[4].Address.Slot).To(Equal("0x00"))

			Expect(domainSpec.Devices.Controllers[5].Model).To(Equal(api.ControllerModelPCIeRootPort))
			Expect(domainSpec.Devices.Controllers[5].Index).To(Equal("6"))
			Expect(domainSpec.Devices.Controllers[5].Address.Bus).To(Equal("4"))
			Expect(domainSpec.Devices.Controllers[5].Address.Slot).To(Equal("0x01"))

			deviceBusBySource := map[string]string{}
			for _, device := range domainSpec.Devices.HostDevices {
				deviceBusBySource[hardware.PCIAddressToString(device.Source.Address)] = device.Address.Bus
			}
			Expect(deviceBusBySource).To(Equal(map[string]string{
				"0000:01:00.0": "2",
				"0000:03:00.0": "3",
				"0000:02:00.0": "5",
				"0000:04:00.0": "6",
			}))
		})

		It("should allocate new controller indexes after existing controllers", func() {
			domainSpec.Devices.Controllers = []api.Controller{
				{Model: api.ControllerModelPCIeRoot, Index: "0"},
				{Model: api.ControllerModelPCIeRootPort, Index: "7"},
			}
			domainSpec.Devices.HostDevices = []api.HostDevice{createPCIDevice("device1", "0x01")}

			err := PlacePCIDevicesWithNUMAAlignment(domainSpec)
			Expect(err).ToNot(HaveOccurred())

			Expect(domainSpec.Devices.Controllers).To(HaveLen(4))
			Expect(domainSpec.Devices.Controllers[2].Model).To(Equal(api.ControllerModelPCIeExpanderBus))
			Expect(domainSpec.Devices.Controllers[2].Index).To(Equal("8"))
			Expect(domainSpec.Devices.Controllers[3].Model).To(Equal(api.ControllerModelPCIeRootPort))
			Expect(domainSpec.Devices.Controllers[3].Index).To(Equal("9"))
			Expect(domainSpec.Devices.HostDevices[0].Address.Bus).To(Equal("9"))
		})
	})
})
