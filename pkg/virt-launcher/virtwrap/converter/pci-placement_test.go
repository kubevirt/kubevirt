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
		originalPciBasePath     string
		originalNodeBasePath    string
		originalDevicesBasePath string
		fakePciBasePath         string
		fakeNodeBasePath        string
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

	// setupFakeSysfsWithPCIeRoots creates a sysfs structure with both NUMA node
	// info and PCIe root symlinks. Each device gets a directory under a fake
	// devices tree, and a symlink in fakePciBasePath pointing to it.
	setupFakeSysfsWithPCIeRoots := func(
		devices map[string]struct{ numaNode, pcieRoot string },
		numaNodes map[string]string,
	) {
		var err error
		fakePciBasePath, err = os.MkdirTemp("", "pci_devices")
		Expect(err).ToNot(HaveOccurred())

		fakeNodeBasePath, err = os.MkdirTemp("", "numa_nodes")
		Expect(err).ToNot(HaveOccurred())

		fakeDevicesBasePath, err := os.MkdirTemp("", "sys_devices")
		Expect(err).ToNot(HaveOccurred())
		hardware.DevicesBasePath = fakeDevicesBasePath

		for pciAddr, info := range devices {
			// Create the real device directory under the fake device tree
			deviceDir := filepath.Join(fakeDevicesBasePath, info.pcieRoot, "0000:00:00.0", pciAddr)
			err = os.MkdirAll(deviceDir, 0o755)
			Expect(err).ToNot(HaveOccurred())

			// Write numa_node file in the real device directory
			err = os.WriteFile(filepath.Join(deviceDir, "numa_node"), []byte(info.numaNode+"\n"), 0o644)
			Expect(err).ToNot(HaveOccurred())

			// Create symlink: fakePciBasePath/<address> -> fakeDevicesBasePath/<pcieRoot>/.../<address>
			symlinkTarget := filepath.Join(fakeDevicesBasePath, info.pcieRoot, "0000:00:00.0", pciAddr)
			err = os.Symlink(symlinkTarget, filepath.Join(fakePciBasePath, pciAddr))
			Expect(err).ToNot(HaveOccurred())
		}

		for numaID, cpuList := range numaNodes {
			numaNodePath := filepath.Join(fakeNodeBasePath, "node"+numaID)
			err = os.MkdirAll(numaNodePath, 0o755)
			Expect(err).ToNot(HaveOccurred())

			err = os.WriteFile(filepath.Join(numaNodePath, "cpulist"), []byte(cpuList+"\n"), 0o644)
			Expect(err).ToNot(HaveOccurred())
		}

		hardware.PciBasePath = fakePciBasePath
		hardware.NodeBasePath = fakeNodeBasePath
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
		originalDevicesBasePath = hardware.DevicesBasePath
		setupFakeSysfs()
		hardware.PciBasePath = fakePciBasePath
		hardware.NodeBasePath = fakeNodeBasePath
	})

	AfterEach(func() {
		hardware.PciBasePath = originalPciBasePath
		hardware.NodeBasePath = originalNodeBasePath
		hardware.DevicesBasePath = originalDevicesBasePath
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

	Describe("PCIe root alignment", func() {
		countControllersByModel := func(controllers []api.Controller, model string) int {
			count := 0
			for _, c := range controllers {
				if c.Model == model {
					count++
				}
			}
			return count
		}

		It("should create switch hierarchy when multiple devices share a PCIe root", func() {
			setupFakeSysfsWithPCIeRoots(
				map[string]struct{ numaNode, pcieRoot string }{
					"0000:81:01.0": {"0", "pci0000:80"},
					"0000:81:02.0": {"0", "pci0000:80"},
				},
				map[string]string{"0": "0-3"},
			)

			domainSpec := createDomainSpecWithNUMA(
				[]api.NUMACell{{ID: "0", CPUs: "0-1"}},
				[]api.CPUTuneVCPUPin{{VCPU: 0, CPUSet: "0"}},
			)
			domainSpec.Devices.HostDevices = []api.HostDevice{
				createPCIDevice("device1", "0x81"),
				createPCIDevice("device2", "0x81"),
			}
			// Fix source addresses to match fake sysfs
			domainSpec.Devices.HostDevices[0].Source.Address.Slot = "0x01"
			domainSpec.Devices.HostDevices[1].Source.Address.Slot = "0x02"

			err := PlacePCIDevicesWithNUMAAlignment(domainSpec)
			Expect(err).ToNot(HaveOccurred())

			controllers := domainSpec.Devices.Controllers
			// 1 expander bus + 1 root port + 1 upstream switch + 2 downstream switches = 5
			Expect(countControllersByModel(controllers, api.ControllerModelPCIeExpanderBus)).To(Equal(1))
			Expect(countControllersByModel(controllers, api.ControllerModelPCIeRootPort)).To(Equal(1))
			Expect(countControllersByModel(controllers, api.ControllerModelPCIeSwitchUpstream)).To(Equal(1))
			Expect(countControllersByModel(controllers, api.ControllerModelPCIeSwitchDownstream)).To(Equal(2))
		})

		It("should not create switches for a single device on a PCIe root", func() {
			setupFakeSysfsWithPCIeRoots(
				map[string]struct{ numaNode, pcieRoot string }{
					"0000:81:01.0": {"0", "pci0000:80"},
				},
				map[string]string{"0": "0-3"},
			)

			domainSpec := createDomainSpecWithNUMA(
				[]api.NUMACell{{ID: "0", CPUs: "0-1"}},
				[]api.CPUTuneVCPUPin{{VCPU: 0, CPUSet: "0"}},
			)
			domainSpec.Devices.HostDevices = []api.HostDevice{
				createPCIDevice("device1", "0x81"),
			}
			domainSpec.Devices.HostDevices[0].Source.Address.Slot = "0x01"

			err := PlacePCIDevicesWithNUMAAlignment(domainSpec)
			Expect(err).ToNot(HaveOccurred())

			controllers := domainSpec.Devices.Controllers
			// 1 expander bus + 1 root port = 2 (no switches)
			Expect(countControllersByModel(controllers, api.ControllerModelPCIeExpanderBus)).To(Equal(1))
			Expect(countControllersByModel(controllers, api.ControllerModelPCIeRootPort)).To(Equal(1))
			Expect(countControllersByModel(controllers, api.ControllerModelPCIeSwitchUpstream)).To(Equal(0))
			Expect(countControllersByModel(controllers, api.ControllerModelPCIeSwitchDownstream)).To(Equal(0))
		})

		It("should create separate root ports for devices on different PCIe roots", func() {
			setupFakeSysfsWithPCIeRoots(
				map[string]struct{ numaNode, pcieRoot string }{
					"0000:81:01.0": {"0", "pci0000:80"},
					"0000:91:01.0": {"0", "pci0000:90"},
				},
				map[string]string{"0": "0-3"},
			)

			domainSpec := createDomainSpecWithNUMA(
				[]api.NUMACell{{ID: "0", CPUs: "0-1"}},
				[]api.CPUTuneVCPUPin{{VCPU: 0, CPUSet: "0"}},
			)
			domainSpec.Devices.HostDevices = []api.HostDevice{
				createPCIDevice("device1", "0x81"),
				createPCIDevice("device2", "0x91"),
			}
			domainSpec.Devices.HostDevices[0].Source.Address.Slot = "0x01"
			domainSpec.Devices.HostDevices[1].Source.Address.Slot = "0x01"

			err := PlacePCIDevicesWithNUMAAlignment(domainSpec)
			Expect(err).ToNot(HaveOccurred())

			controllers := domainSpec.Devices.Controllers
			// 1 expander bus + 2 root ports = 3 (no switches, different PCIe roots)
			Expect(countControllersByModel(controllers, api.ControllerModelPCIeExpanderBus)).To(Equal(1))
			Expect(countControllersByModel(controllers, api.ControllerModelPCIeRootPort)).To(Equal(2))
			Expect(countControllersByModel(controllers, api.ControllerModelPCIeSwitchUpstream)).To(Equal(0))
			Expect(countControllersByModel(controllers, api.ControllerModelPCIeSwitchDownstream)).To(Equal(0))
		})

		It("should handle mixed PCIe roots on the same NUMA node", func() {
			setupFakeSysfsWithPCIeRoots(
				map[string]struct{ numaNode, pcieRoot string }{
					"0000:81:01.0": {"0", "pci0000:80"},
					"0000:81:02.0": {"0", "pci0000:80"},
					"0000:91:01.0": {"0", "pci0000:90"},
				},
				map[string]string{"0": "0-3"},
			)

			domainSpec := createDomainSpecWithNUMA(
				[]api.NUMACell{{ID: "0", CPUs: "0-1"}},
				[]api.CPUTuneVCPUPin{{VCPU: 0, CPUSet: "0"}},
			)
			domainSpec.Devices.HostDevices = []api.HostDevice{
				createPCIDevice("device1", "0x81"),
				createPCIDevice("device2", "0x81"),
				createPCIDevice("device3", "0x91"),
			}
			domainSpec.Devices.HostDevices[0].Source.Address.Slot = "0x01"
			domainSpec.Devices.HostDevices[1].Source.Address.Slot = "0x02"
			domainSpec.Devices.HostDevices[2].Source.Address.Slot = "0x01"

			err := PlacePCIDevicesWithNUMAAlignment(domainSpec)
			Expect(err).ToNot(HaveOccurred())

			controllers := domainSpec.Devices.Controllers
			// pci0000:80 has 2 devices -> root port + upstream + 2 downstream
			// pci0000:90 has 1 device -> root port only
			// + 1 expander bus
			// Total: 1 expander + 2 root ports + 1 upstream + 2 downstream = 6
			Expect(countControllersByModel(controllers, api.ControllerModelPCIeExpanderBus)).To(Equal(1))
			Expect(countControllersByModel(controllers, api.ControllerModelPCIeRootPort)).To(Equal(2))
			Expect(countControllersByModel(controllers, api.ControllerModelPCIeSwitchUpstream)).To(Equal(1))
			Expect(countControllersByModel(controllers, api.ControllerModelPCIeSwitchDownstream)).To(Equal(2))
		})
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

			err := PlacePCIDevicesWithNUMAAlignment(domainSpec)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("insufficient bus numbers for NUMA-aligned PCIe topology"))
			Expect(err.Error()).To(ContainSubstring("current controller index 256"))
			Expect(err.Error()).To(ContainSubstring("last assigned expander bus number 255"))
		})

		It("should assign bus numbers for expander buses calculated as maxBusNr - controllerCount + 1", func() {
			domainSpec.Devices.HostDevices = []api.HostDevice{
				createPCIDevice("device1", "0x01"),
				createPCIDevice("device2", "0x02"),
			}

			err := PlacePCIDevicesWithNUMAAlignment(domainSpec)
			Expect(err).ToNot(HaveOccurred())

			// Bus numbers calculated as 255 - controllerCount + 1:
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
	})
})
