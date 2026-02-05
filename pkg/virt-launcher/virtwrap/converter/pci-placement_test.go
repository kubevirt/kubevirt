package converter

import (
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/utils/ptr"

	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("PCI placement", func() {
	Context("PCI topology with NUMA alignment", func() {
		var assigner *expanderBusAssigner
		var domainSpec *api.DomainSpec
		var mockDevices []api.HostDevice
		var (
			originalPciBasePath  string
			originalNodeBasePath string
			fakePciBasePath      string
			fakeNodeBasePath     string
		)

		createTempSysfsStructure := func() {
			var err error
			// Create fake PCI devices structure
			fakePciBasePath, err = os.MkdirTemp("", "pci_devices")
			Expect(err).ToNot(HaveOccurred())

			// Create fake NUMA node structure
			fakeNodeBasePath, err = os.MkdirTemp("", "numa_nodes")
			Expect(err).ToNot(HaveOccurred())

			// Create multiple test PCI devices with NUMA nodes
			testDevices := map[string]string{
				"0000:01:00.0": "0",
				"0000:02:00.0": "1",
				"0000:03:00.0": "0",
				"0000:04:00.0": "1",
				"0000:05:00.0": "0",
			}

			for pciAddr, numaNode := range testDevices {
				pciDevicePath := filepath.Join(fakePciBasePath, pciAddr)
				err = os.MkdirAll(pciDevicePath, 0755)
				Expect(err).ToNot(HaveOccurred())

				// Write NUMA node file for the PCI device
				numaNodeFile := filepath.Join(pciDevicePath, "numa_node")
				err = os.WriteFile(numaNodeFile, []byte(numaNode+"\n"), 0644)
				Expect(err).ToNot(HaveOccurred())
			}

			// Create NUMA node directories
			for numaID, cpuList := range map[string]string{"0": "0-3", "1": "4-7"} {
				numaNodePath := filepath.Join(fakeNodeBasePath, "node"+numaID)
				err = os.MkdirAll(numaNodePath, 0755)
				Expect(err).ToNot(HaveOccurred())

				// Write cpulist file for NUMA node
				cpuListFile := filepath.Join(numaNodePath, "cpulist")
				err = os.WriteFile(cpuListFile, []byte(cpuList+"\n"), 0644)
				Expect(err).ToNot(HaveOccurred())
			}
		}

		BeforeEach(func() {
			// Save original paths
			originalPciBasePath = hardware.PciBasePath
			originalNodeBasePath = hardware.NodeBasePath

			// Create fake sysfs structure
			createTempSysfsStructure()

			// Redirect to fake paths
			hardware.PciBasePath = fakePciBasePath
			hardware.NodeBasePath = fakeNodeBasePath
			domainSpec = &api.DomainSpec{
				Devices: api.Devices{
					Controllers: []api.Controller{},
				},
			}
			assigner = NewExpanderBusAssigner(domainSpec)
			mockDevices = []api.HostDevice{
				{
					Type: api.HostDevicePCI,
					Source: api.HostDeviceSource{
						Address: &api.Address{Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0"},
					},
				},
				{
					Type: api.HostDevicePCI,
					Source: api.HostDeviceSource{
						Address: &api.Address{Domain: "0x0000", Bus: "0x02", Slot: "0x00", Function: "0x0"},
					},
				},
			}
		})

		AfterEach(func() {
			// Restore original paths
			hardware.PciBasePath = originalPciBasePath
			hardware.NodeBasePath = originalNodeBasePath

			// Clean up temporary directories
			if fakePciBasePath != "" {
				os.RemoveAll(fakePciBasePath)
			}
			if fakeNodeBasePath != "" {
				os.RemoveAll(fakeNodeBasePath)
			}
		})

		Context("device grouping and topology creation", func() {
			It("should initialize PCIe expander bus assigner correctly", func() {
				Expect(assigner).ToNot(BeNil())
				Expect(assigner.domainSpec).To(Equal(domainSpec))
				Expect(assigner.topologyMap).ToNot(BeNil())
				Expect(assigner.controllerIndex).To(Equal(uint32(1)))
				Expect(assigner.devices).To(BeEmpty())
			})

			It("should correctly increment controller indices", func() {
				initialIndex := assigner.controllerIndex

				controller1 := assigner.createController(api.ControllerModelPCIeRootPort, "1", 0, nil)
				Expect(controller1.Index).To(Equal(fmt.Sprint(initialIndex + 1)))

				controller2 := assigner.createController(api.ControllerModelPCIeExpanderBus, "", 0, ptr.To(uint32(0)))
				Expect(controller2.Index).To(Equal(fmt.Sprint(initialIndex + 2)))

				controller3 := assigner.createController(api.ControllerModelPCIeRootPort, "2", 1, nil)
				Expect(controller3.Index).To(Equal(fmt.Sprint(initialIndex + 3)))

				Expect(assigner.controllerIndex).To(Equal(initialIndex + 3))
			})

			It("should filter PCI devices correctly", func() {
				mixedDevices := []api.HostDevice{
					{Type: api.HostDevicePCI, Source: api.HostDeviceSource{Address: &api.Address{Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0"}}},
					{Type: "usb"},
					{Type: api.HostDevicePCI, Source: api.HostDeviceSource{Address: &api.Address{Domain: "0x0000", Bus: "0x02", Slot: "0x00", Function: "0x0"}}},
					{Type: "scsi"},
				}

				Expect(assigner.devices).To(BeEmpty())

				assigner.AddDevices(mixedDevices)

				// Only PCI devices should be considered, but since there's no NUMA info in the domain spec,
				// devices without NUMA affinity are filtered out during AddDevices
				Expect(len(assigner.devices)).To(Equal(0))
			})

			It("should filter non-PCI devices and only include PCI devices with NUMA affinity", func() {
				// Set up domain spec with NUMA topology for devices to have affinity
				domainSpec.CPU = api.CPU{
					NUMA: &api.NUMA{
						Cells: []api.NUMACell{
							{ID: "0", CPUs: "0-1"},
							{ID: "1", CPUs: "2-3"},
						},
					},
				}
				domainSpec.Devices.HostDevices = []api.HostDevice{
					{
						Type: api.HostDevicePCI,
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0"},
						},
					},
				}
				domainSpec.CPUTune = &api.CPUTune{
					VCPUPin: []api.CPUTuneVCPUPin{
						{VCPU: 0, CPUSet: "0"},
					},
				}

				// Create new assigner with updated domain spec
				assignerWithNUMA := NewExpanderBusAssigner(domainSpec)

				mixedDevices := []api.HostDevice{
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("pci_device_1"),
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0"},
						},
					},
					{Type: "usb"},
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("pci_device_2"),
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x02", Slot: "0x00", Function: "0x0"},
						},
					},
					{Type: "scsi"},
				}

				Expect(assignerWithNUMA.devices).To(BeEmpty())

				assignerWithNUMA.AddDevices(mixedDevices)

				// Should filter out non-PCI devices (usb, scsi) and only consider PCI devices
				// However, only devices with NUMA affinity will be added to the devices slice
				// Since only the first PCI device matches the CPU pinning in the domain spec,
				// we expect exactly 1 device to be added
				Expect(len(assignerWithNUMA.devices)).To(Equal(1))
				Expect(assignerWithNUMA.devices[0].Type).To(Equal(api.HostDevicePCI))
				Expect(assignerWithNUMA.devices[0].Alias.GetName()).To(Equal("pci_device_1"))
			})
		})

		Context("PCI device placement with NUMA alignment", func() {
			It("should handle empty device list", func() {
				err := assigner.PlaceDevices()
				Expect(err).ToNot(HaveOccurred())

				Expect(domainSpec.Devices.Controllers).To(BeEmpty())
			})

			It("should not handle devices without NUMA affinity", func() {
				assigner.AddDevices(mockDevices)

				err := assigner.PlaceDevices()
				Expect(err).ToNot(HaveOccurred())

				Expect(domainSpec.Devices.Controllers).To(BeEmpty())
			})
		})

		Context("PCIe NUMA aware controller creation", func() {
			It("should create PCIe expander bus correctly", func() {
				numaNode := uint32(1)
				controller := assigner.createController(api.ControllerModelPCIeExpanderBus, "", 0, &numaNode)

				Expect(controller).ToNot(BeNil())
				Expect(controller.Type).To(Equal(api.ControllerTypePCI))
				Expect(controller.Model).To(Equal(api.ControllerModelPCIeExpanderBus))
				Expect(controller.Index).To(Equal("2"))
				Expect(controller.Target).ToNot(BeNil())
				Expect(controller.Target.NUMANode).To(Equal(&numaNode))
				Expect(controller.Address).To(BeNil())
			})

			It("should create PCIe root port correctly", func() {
				slot := uint32(5)
				parentBus := "1"
				controller := assigner.createController(api.ControllerModelPCIeRootPort, parentBus, slot, nil)

				Expect(controller).ToNot(BeNil())
				Expect(controller.Type).To(Equal(api.ControllerTypePCI))
				Expect(controller.Model).To(Equal(api.ControllerModelPCIeRootPort))
				Expect(controller.Index).To(Equal("2"))
				Expect(controller.Target).To(BeNil())
				Expect(controller.Address).ToNot(BeNil())
				Expect(controller.Address.Type).To(Equal(api.AddressPCI))
				Expect(controller.Address.Domain).To(Equal("0x0000"))
				Expect(controller.Address.Bus).To(Equal(parentBus))
				Expect(controller.Address.Slot).To(Equal("0x05"))
				Expect(controller.Address.Function).To(Equal("0x0"))
			})

			It("should increment index correctly for multiple controllers", func() {
				initialIndex := assigner.controllerIndex
				Expect(initialIndex).To(Equal(uint32(1)))

				controller1 := assigner.createController(api.ControllerModelPCIeRootPort, "1", 1, nil)
				Expect(controller1.Index).To(Equal("2"))
				Expect(assigner.controllerIndex).To(Equal(uint32(2)))

				controller2 := assigner.createController(api.ControllerModelPCIeRootPort, "1", 2, nil)
				Expect(controller2.Index).To(Equal("3"))
				Expect(assigner.controllerIndex).To(Equal(uint32(3)))
			})
		})

		Context("NUMA aware topology management", func() {
			It("should ensure NUMA aware topology creation", func() {
				key := numaKey{numaNode: 0}

				Expect(assigner.topologyMap).To(BeEmpty())

				topology := assigner.numaAwareTopology(key)
				Expect(topology).ToNot(BeNil())
				Expect(topology.expanderBus).ToNot(BeNil())
				Expect(topology.expanderBus.Model).To(Equal(api.ControllerModelPCIeExpanderBus))
				Expect(topology.expanderBus.Target.NUMANode).To(Equal(&key.numaNode))
				Expect(topology.addressPerDeviceAlias).ToNot(BeNil())
				Expect(topology.addressPerDeviceAlias).To(BeEmpty())

				topology2 := assigner.numaAwareTopology(key)
				Expect(topology2).To(Equal(topology))
				Expect(assigner.topologyMap).To(HaveLen(1))
			})

			It("should create separate topologies for different NUMA nodes", func() {
				key1 := numaKey{numaNode: 0}
				key2 := numaKey{numaNode: 1}

				topology1 := assigner.numaAwareTopology(key1)
				topology2 := assigner.numaAwareTopology(key2)

				Expect(topology1).ToNot(Equal(topology2))
				Expect(assigner.topologyMap).To(HaveLen(2))
				Expect(topology1.expanderBus.Target.NUMANode).To(Equal(&key1.numaNode))
				Expect(topology2.expanderBus.Target.NUMANode).To(Equal(&key2.numaNode))
			})
		})

		Context("topology building", func() {
			var mockTopology *numaAwareTopology
			var mockDevice *api.HostDevice

			BeforeEach(func() {
				mockTopology = &numaAwareTopology{
					expanderBus:           &api.Controller{Index: "1"},
					rootPorts:             []*api.Controller{},
					addressPerDeviceAlias: make(map[string]*api.Address),
				}
				mockDevice = &api.HostDevice{
					Type:  api.HostDevicePCI,
					Alias: api.NewUserDefinedAlias("device1"),
					Source: api.HostDeviceSource{
						Address: &api.Address{Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0"},
					},
				}
			})

			It("should place a host device correctly", func() {
				assigner.placeDevice(mockTopology, mockDevice)

				Expect(mockTopology.rootPorts).To(HaveLen(1))

				address, exists := mockTopology.addressPerDeviceAlias[mockDevice.Alias.GetName()]
				Expect(exists).To(BeTrue())
				Expect(address).ToNot(BeNil())
				Expect(address.Type).To(Equal(api.AddressPCI))
				Expect(address.Bus).To(Equal(mockTopology.rootPorts[0].Index))
			})
		})

		It("should handle mixed NUMA and non-NUMA devices", func() {
			mixedDevices := []api.HostDevice{
				{
					Type:  api.HostDevicePCI,
					Alias: api.NewUserDefinedAlias("numa_device"),
					Source: api.HostDeviceSource{
						Address: &api.Address{Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0"},
					},
				},
				{
					Type:  api.HostDevicePCI,
					Alias: api.NewUserDefinedAlias("non_numa_device"),
					Source: api.HostDeviceSource{
						Address: &api.Address{Domain: "0x0000", Bus: "0x02", Slot: "0x00", Function: "0x0"},
					},
				},
				{
					Type: "usb", // Non-PCI device should be filtered out
					Source: api.HostDeviceSource{
						Address: &api.Address{Domain: "0x0000", Bus: "0x03", Slot: "0x00", Function: "0x0"},
					},
				},
			}

			assigner.AddDevices(mixedDevices)
			err := assigner.PlaceDevices()
			Expect(err).ToNot(HaveOccurred())
		})

		Context("NUMA edge cases", func() {
			It("should handle invalid NUMA node IDs", func() {
				domainSpec.CPU = api.CPU{
					NUMA: &api.NUMA{
						Cells: []api.NUMACell{
							{ID: "invalid", CPUs: "0-1"},
							{ID: "1", CPUs: "2-3"},
						},
					},
				}
				domainSpec.CPUTune = &api.CPUTune{
					VCPUPin: []api.CPUTuneVCPUPin{
						{VCPU: 0, CPUSet: "0"},
					},
				}

				assignerWithInvalidNUMA := NewExpanderBusAssigner(domainSpec)

				devices := []api.HostDevice{
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("test_device"),
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0"},
						},
					},
				}

				assignerWithInvalidNUMA.AddDevices(devices)

				// Device should not be added due to invalid NUMA node ID
				Expect(len(assignerWithInvalidNUMA.devices)).To(Equal(0))

				err := assignerWithInvalidNUMA.PlaceDevices()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should handle asymmetric NUMA topology", func() {
				// Create asymmetric NUMA topology with different CPU counts per node
				domainSpec.CPU = api.CPU{
					NUMA: &api.NUMA{
						Cells: []api.NUMACell{
							{ID: "0", CPUs: "0"},   // Single CPU in node 0
							{ID: "1", CPUs: "1-3"}, // Three CPUs in node 1
							{ID: "2", CPUs: "4-7"}, // Four CPUs in node 2
						},
					},
				}
				domainSpec.CPUTune = &api.CPUTune{
					VCPUPin: []api.CPUTuneVCPUPin{
						{VCPU: 0, CPUSet: "0"}, // Device on node 0
						{VCPU: 1, CPUSet: "1"}, // Device on node 1
						{VCPU: 4, CPUSet: "4"}, // Device on node 2
					},
				}

				assignerWithAsymmetricNUMA := NewExpanderBusAssigner(domainSpec)

				devices := []api.HostDevice{
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("device_node0"),
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0"},
						},
					},
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("device_node1"),
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x02", Slot: "0x00", Function: "0x0"},
						},
					},
				}

				assignerWithAsymmetricNUMA.AddDevices(devices)

				// Both devices should be processed despite asymmetric topology
				Expect(len(assignerWithAsymmetricNUMA.devices)).To(Equal(2))

				err := assignerWithAsymmetricNUMA.PlaceDevices()
				Expect(err).ToNot(HaveOccurred())

				// Should create separate topologies for different NUMA nodes
				Expect(len(assignerWithAsymmetricNUMA.topologyMap)).To(Equal(2))
			})

			It("should handle missing NUMA topology in domain spec", func() {
				// Domain spec without NUMA configuration
				emptyDomainSpec := &api.DomainSpec{
					Devices: api.Devices{
						Controllers: []api.Controller{},
					},
				}
				assignerNoNUMA := NewExpanderBusAssigner(emptyDomainSpec)

				devices := []api.HostDevice{
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("test_device"),
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0"},
						},
					},
				}

				assignerNoNUMA.AddDevices(devices)

				// No devices should be added since there's no NUMA information
				Expect(len(assignerNoNUMA.devices)).To(Equal(0))

				err := assignerNoNUMA.PlaceDevices()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(assignerNoNUMA.topologyMap)).To(Equal(0))
			})

			It("should handle devices without CPU affinity information", func() {
				domainSpec.CPU = api.CPU{
					NUMA: &api.NUMA{
						Cells: []api.NUMACell{
							{ID: "0", CPUs: "0-1"},
							{ID: "1", CPUs: "2-3"},
						},
					},
				}
				// No CPUTune configuration - devices won't have vCPU affinity

				assignerNoAffinity := NewExpanderBusAssigner(domainSpec)

				devices := []api.HostDevice{
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("no_affinity_device"),
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0"},
						},
					},
				}

				assignerNoAffinity.AddDevices(devices)

				// Device should not be added since it has no CPU affinity
				Expect(len(assignerNoAffinity.devices)).To(Equal(0))

				err := assignerNoAffinity.PlaceDevices()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("Input validation and error handling", func() {
			It("should handle malformed PCI addresses gracefully", func() {
				domainSpec.CPU = api.CPU{
					NUMA: &api.NUMA{
						Cells: []api.NUMACell{
							{ID: "0", CPUs: "0-1"},
						},
					},
				}
				domainSpec.CPUTune = &api.CPUTune{
					VCPUPin: []api.CPUTuneVCPUPin{
						{VCPU: 0, CPUSet: "0"},
					},
				}

				assignerMalformed := NewExpanderBusAssigner(domainSpec)

				malformedDevices := []api.HostDevice{
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("malformed_device"),
						Source: api.HostDeviceSource{
							Address: &api.Address{
								Domain:   "invalid", // Invalid domain format
								Bus:      "xyz",     // Invalid bus format
								Slot:     "invalid", // Invalid slot format
								Function: "z",       // Invalid function format
							},
						},
					},
				}

				assignerMalformed.AddDevices(malformedDevices)

				// Malformed devices should be filtered out during NUMA lookup
				Expect(len(assignerMalformed.devices)).To(Equal(0))

				err := assignerMalformed.PlaceDevices()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should handle invalid CPU set ranges", func() {
				domainSpec.CPU = api.CPU{
					NUMA: &api.NUMA{
						Cells: []api.NUMACell{
							{ID: "0", CPUs: "invalid-range"},
							{ID: "1", CPUs: "0-"},  // Invalid range format
							{ID: "2", CPUs: "abc"}, // Non-numeric CPUs
						},
					},
				}
				domainSpec.CPUTune = &api.CPUTune{
					VCPUPin: []api.CPUTuneVCPUPin{
						{VCPU: 0, CPUSet: "0"},
					},
				}

				assignerInvalidCPU := NewExpanderBusAssigner(domainSpec)

				devices := []api.HostDevice{
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("invalid_cpu_device"),
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0"},
						},
					},
				}

				assignerInvalidCPU.AddDevices(devices)

				// Device should not be added due to invalid CPU ranges in NUMA cells
				Expect(len(assignerInvalidCPU.devices)).To(Equal(0))

				err := assignerInvalidCPU.PlaceDevices()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should handle bus number overflow scenarios", func() {
				domainSpec.CPU = api.CPU{
					NUMA: &api.NUMA{
						Cells: []api.NUMACell{
							{ID: "0", CPUs: "0-1"},
							{ID: "1", CPUs: "2-3"},
						},
					},
				}
				domainSpec.CPUTune = &api.CPUTune{
					VCPUPin: []api.CPUTuneVCPUPin{
						{VCPU: 0, CPUSet: "0"},
						{VCPU: 1, CPUSet: "2"},
					},
				}

				assignerOverflow := NewExpanderBusAssigner(domainSpec)

				// Create a high index to simulate bus number overflow
				assignerOverflow.controllerIndex = 255 // This should trigger bus number underflow

				devices := []api.HostDevice{
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("overflow_device"),
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0"},
						},
					},
				}

				assignerOverflow.AddDevices(devices)

				// Device should be added but placement should still work (may assign invalid bus numbers)
				Expect(len(assignerOverflow.devices)).To(Equal(1))

				err := assignerOverflow.PlaceDevices()
				// Should not error, but bus number calculation might be problematic
				Expect(err).ToNot(HaveOccurred())
			})

			It("should handle nil device addresses", func() {
				domainSpec.CPU = api.CPU{
					NUMA: &api.NUMA{
						Cells: []api.NUMACell{
							{ID: "0", CPUs: "0-1"},
						},
					},
				}
				domainSpec.CPUTune = &api.CPUTune{
					VCPUPin: []api.CPUTuneVCPUPin{
						{VCPU: 0, CPUSet: "0"},
					},
				}

				assignerNilAddress := NewExpanderBusAssigner(domainSpec)

				devicesWithNilAddress := []api.HostDevice{
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("nil_address_device"),
						Source: api.HostDeviceSource{
							Address: nil, // Nil address
						},
					},
				}

				assignerNilAddress.AddDevices(devicesWithNilAddress)

				// Device with nil address should be filtered out
				Expect(len(assignerNilAddress.devices)).To(Equal(0))

				err := assignerNilAddress.PlaceDevices()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should handle empty device alias", func() {
				domainSpec.CPU = api.CPU{
					NUMA: &api.NUMA{
						Cells: []api.NUMACell{
							{ID: "0", CPUs: "0-1"},
						},
					},
				}
				domainSpec.CPUTune = &api.CPUTune{
					VCPUPin: []api.CPUTuneVCPUPin{
						{VCPU: 0, CPUSet: "0"},
					},
				}

				assignerEmptyAlias := NewExpanderBusAssigner(domainSpec)

				deviceWithoutAlias := []api.HostDevice{
					{
						Type: api.HostDevicePCI,
						// No Alias field set
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0"},
						},
					},
				}

				assignerEmptyAlias.AddDevices(deviceWithoutAlias)

				// Device should still be processed, but placement might have issues with empty alias
				err := assignerEmptyAlias.PlaceDevices()
				// Should not panic or error, even with missing alias
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("NUMA device grouping", func() {
			It("should group devices by NUMA node correctly", func() {
				domainSpec.CPU = api.CPU{
					NUMA: &api.NUMA{
						Cells: []api.NUMACell{
							{ID: "0", CPUs: "0-1"},
							{ID: "1", CPUs: "2-3"},
						},
					},
				}
				domainSpec.CPUTune = &api.CPUTune{
					VCPUPin: []api.CPUTuneVCPUPin{
						{VCPU: 0, CPUSet: "0"}, // Maps to NUMA node 0
						{VCPU: 2, CPUSet: "4"}, // Maps to NUMA node 1
					},
				}

				groupingAssigner := NewExpanderBusAssigner(domainSpec)

				devices := []api.HostDevice{
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("device_numa0"),
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0"},
						},
					},
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("device_numa1"),
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x02", Slot: "0x00", Function: "0x0"},
						},
					},
				}

				groupingAssigner.AddDevices(devices)

				groups, err := groupingAssigner.groupDevicesByNUMA()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(groups)).To(Equal(2))

				// Verify devices are grouped by their NUMA nodes
				for key, deviceList := range groups {
					Expect(len(deviceList)).To(Equal(1))
					if key.numaNode == 0 {
						Expect(deviceList[0].Alias.GetName()).To(Equal("device_numa0"))
					} else if key.numaNode == 1 {
						Expect(deviceList[0].Alias.GetName()).To(Equal("device_numa1"))
					}
				}
			})

			It("should handle multiple devices on the same NUMA node", func() {
				domainSpec.CPU = api.CPU{
					NUMA: &api.NUMA{
						Cells: []api.NUMACell{
							{ID: "0", CPUs: "0-3"},
						},
					},
				}
				domainSpec.CPUTune = &api.CPUTune{
					VCPUPin: []api.CPUTuneVCPUPin{
						{VCPU: 0, CPUSet: "0"},
						{VCPU: 1, CPUSet: "1"},
					},
				}

				multiDeviceAssigner := NewExpanderBusAssigner(domainSpec)

				devices := []api.HostDevice{
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("device1_numa0"),
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0"},
						},
					},
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("device2_numa0"),
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x03", Slot: "0x00", Function: "0x0"},
						},
					},
				}

				multiDeviceAssigner.AddDevices(devices)

				groups, err := multiDeviceAssigner.groupDevicesByNUMA()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(groups)).To(Equal(1))

				// Both devices should be in the same NUMA group
				for _, deviceList := range groups {
					Expect(len(deviceList)).To(Equal(2))
				}
			})
		})

		Context("Bus number assignment", func() {
			It("should assign unique bus numbers for multiple expander buses", func() {
				domainSpec.CPU = api.CPU{
					NUMA: &api.NUMA{
						Cells: []api.NUMACell{
							{ID: "0", CPUs: "0-1"},
							{ID: "1", CPUs: "2-3"},
							{ID: "2", CPUs: "4-5"},
						},
					},
				}
				domainSpec.CPUTune = &api.CPUTune{
					VCPUPin: []api.CPUTuneVCPUPin{
						{VCPU: 0, CPUSet: "0"}, // NUMA node 0 (CPUs 0-3)
						{VCPU: 2, CPUSet: "4"}, // NUMA node 1 (CPUs 4-7)
						{VCPU: 4, CPUSet: "5"}, // NUMA node 1 (CPUs 4-7)
					},
				}

				busAssigner := NewExpanderBusAssigner(domainSpec)

				devices := []api.HostDevice{
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("device_numa0"),
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0"},
						},
					},
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("device_numa1"),
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x02", Slot: "0x00", Function: "0x0"},
						},
					},
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("device_numa2"),
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x03", Slot: "0x00", Function: "0x0"},
						},
					},
				}

				busAssigner.AddDevices(devices)
				err := busAssigner.PlaceDevices()
				Expect(err).ToNot(HaveOccurred())

				busNumbers := make(map[uint32]bool)
				expanderBusCount := 0

				// Check that each expander bus gets a unique bus number
				for _, controller := range domainSpec.Devices.Controllers {
					if controller.Model == api.ControllerModelPCIeExpanderBus {
						expanderBusCount++
						Expect(controller.Target).ToNot(BeNil())
						Expect(controller.Target.BusNr).ToNot(BeNil())

						busNr := *controller.Target.BusNr
						Expect(busNumbers[busNr]).To(BeFalse(), "Bus number %d should be unique", busNr)
						busNumbers[busNr] = true
					}
				}

				Expect(expanderBusCount).To(Equal(2))
			})
		})

		Context("Root port slot assignment", func() {
			It("should assign sequential slots for root ports on the same expander bus", func() {
				domainSpec.CPU = api.CPU{
					NUMA: &api.NUMA{
						Cells: []api.NUMACell{
							{ID: "0", CPUs: "0-3"},
						},
					},
				}
				domainSpec.CPUTune = &api.CPUTune{
					VCPUPin: []api.CPUTuneVCPUPin{
						{VCPU: 0, CPUSet: "0"},
					},
				}

				slotAssigner := NewExpanderBusAssigner(domainSpec)

				// Multiple devices on same NUMA node should get sequential slots
				devices := []api.HostDevice{
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("device1"),
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0"},
						},
					},
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("device2"),
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x03", Slot: "0x00", Function: "0x0"},
						},
					},
					{
						Type:  api.HostDevicePCI,
						Alias: api.NewUserDefinedAlias("device3"),
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0000", Bus: "0x05", Slot: "0x00", Function: "0x0"},
						},
					},
				}

				slotAssigner.AddDevices(devices)
				err := slotAssigner.PlaceDevices()
				Expect(err).ToNot(HaveOccurred())

				rootPortSlots := []string{}
				expanderBusIndex := ""

				// Find the expander bus and collect root port slots
				for _, controller := range domainSpec.Devices.Controllers {
					if controller.Model == api.ControllerModelPCIeExpanderBus {
						expanderBusIndex = controller.Index
					} else if controller.Model == api.ControllerModelPCIeRootPort {
						rootPortSlots = append(rootPortSlots, controller.Address.Slot)
					}
				}

				// Should have 3 root ports with sequential slots
				Expect(len(rootPortSlots)).To(Equal(3))
				Expect(rootPortSlots).To(ContainElement("0x00"))
				Expect(rootPortSlots).To(ContainElement("0x01"))
				Expect(rootPortSlots).To(ContainElement("0x02"))

				// All root ports should be on the same expander bus
				for _, controller := range domainSpec.Devices.Controllers {
					if controller.Model == api.ControllerModelPCIeRootPort {
						Expect(controller.Address.Bus).To(Equal(expanderBusIndex))
					}
				}
			})
		})
	})
})
