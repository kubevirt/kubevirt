package components

import (
	"errors"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsinternal "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiservervalidation "k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	validationerrors "k8s.io/kube-openapi/pkg/validation/errors"
	"k8s.io/kube-openapi/pkg/validation/validate"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Test CRD validation", func() {
	var validator *validate.SchemaValidator

	BeforeEach(func() {
		vmiCRD := CRDsValidation["virtualmachineinstance"]
		validationSchema := &apiextensionsv1.CustomResourceValidation{}
		err := yaml.Unmarshal([]byte(vmiCRD), validationSchema)
		Expect(err).ToNot(HaveOccurred())

		internalValidationSchema := &apiextensionsinternal.CustomResourceValidation{}
		err = apiextensionsv1.Convert_v1_CustomResourceValidation_To_apiextensions_CustomResourceValidation(validationSchema, internalValidationSchema, nil)
		Expect(err).ToNot(HaveOccurred())

		validator, _, err = apiservervalidation.NewSchemaValidator(internalValidationSchema)
		Expect(err).ToNot(HaveOccurred())
		Expect(validator).ToNot(BeNil())
	})

	DescribeTable("check vmi CRD validation", func(spec v1.VirtualMachineInstanceSpec, numErrors int, expectedErr ...string) {
		vmi := &v1.VirtualMachineInstance{
			Spec: spec,
		}
		res := validator.Validate(vmi)
		if numErrors > 0 {
			Expect(res.IsValid()).To(BeFalse())
			Expect(res.Errors).To(HaveLen(numErrors))
			for _, err := range res.Errors {
				expected := &validationerrors.Validation{}
				isValidationError := errors.As(err, &expected)
				Expect(isValidationError).To(BeTrue())
				for _, e := range expectedErr {
					Expect(err).To(MatchError(ContainSubstring(e)))
				}
			}

		} else {
			Expect(res.IsValid()).To(BeTrue(), "%#v", res.Errors)
		}

	},
		Entry("check firmware.serial - max length",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Firmware: &v1.Firmware{
						Serial: strings.Repeat("a", 256),
					},
				},
			}, 0),
		Entry("check firmware.serial - to long",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Firmware: &v1.Firmware{
						Serial: strings.Repeat("a", 257),
					},
				},
			}, 1, "spec.domain.firmware.serial", "256"),
		Entry("check firmware.serial - pattern (spaces)",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Firmware: &v1.Firmware{
						Serial: "a serial number with spaces",
					},
				},
			}, 1, "spec.domain.firmware.serial", "should match '^[A-Za-z0-9_.+-]+$'"),
		Entry("check firmware.serial - pattern (special chars)",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Firmware: &v1.Firmware{
						Serial: "a1b2@3",
					},
				},
			}, 1, "spec.domain.firmware.serial", "should match '^[A-Za-z0-9_.+-]+$'"),
		Entry("max num of volumes",
			v1.VirtualMachineInstanceSpec{
				Volumes: make([]v1.Volume, 256),
			}, 0),
		Entry("too many volumes",
			v1.VirtualMachineInstanceSpec{
				Volumes: make([]v1.Volume, 257),
			}, 1, "256", "spec.volume"),
		Entry("too many disks",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Disks: make([]v1.Disk, 257),
					},
				},
			}, 1, "256", "spec.domain.devices.disks"),
		Entry("disks serial max length",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Disks: []v1.Disk{
							{
								Serial: strings.Repeat("a", 256),
							},
						},
					},
				},
			}, 0),
		Entry("disks serial too long",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Disks: []v1.Disk{
							{
								Serial: strings.Repeat("a", 257),
							},
						},
					},
				},
			}, 1, "256", "spec.domain.devices.disks[0].serial"),
		Entry("check Input: no input type",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Inputs: []v1.Input{
							{
								Name: "aName",
							},
						},
					},
				},
			}, 1, "spec.domain.devices.inputs[0].type"),
		Entry("check Input: input type is not 'tablet'",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Inputs: []v1.Input{
							{
								Name: "aName",
								Type: "iPod",
							},
						},
					},
				},
			}, 1, "spec.domain.devices.inputs[0].type"),
		Entry("check Input: input type is 'tablet'",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Inputs: []v1.Input{
							{
								Name: "aName",
								Type: "tablet",
							},
						},
					},
				},
			}, 0),
		Entry("check Input: bus is virtio",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Inputs: []v1.Input{
							{
								Name: "aName",
								Type: "tablet",
								Bus:  "virtio",
							},
						},
					},
				},
			}, 0),
		Entry("check Input: bus is usb",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Inputs: []v1.Input{
							{
								Name: "aName",
								Type: "tablet",
								Bus:  "usb",
							},
						},
					},
				},
			}, 0),
		Entry("check Input: bus is something else",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Inputs: []v1.Input{
							{
								Name: "aName",
								Type: "tablet",
								Bus:  "other",
							},
						},
					},
				},
			}, 1, "spec.domain.devices.inputs[0].bus", "usb", "virtio"),
		Entry("check IOThreadsPolicy: 'shared'",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					IOThreadsPolicy: ptr.To[v1.IOThreadsPolicy](v1.IOThreadsPolicyShared),
				},
			}, 0),
		Entry("check IOThreadsPolicy: 'auto'",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					IOThreadsPolicy: ptr.To[v1.IOThreadsPolicy](v1.IOThreadsPolicyAuto),
				},
			}, 0),
		Entry("check IOThreadsPolicy: 'host-passthrough'",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					IOThreadsPolicy: ptr.To[v1.IOThreadsPolicy](v1.CPUModeHostPassthrough),
				},
			}, 1, "spec.domain.ioThreadsPolicy", "shared", "auto"),
		Entry("check sound device: no name",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Sound: &v1.SoundDevice{},
					},
				},
			}, 1, "spec.domain.devices.sound.name"),
		Entry("check sound device: empty name",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Sound: &v1.SoundDevice{
							Name: "",
						},
					},
				},
			}, 1, "spec.domain.devices.sound.name"),
		Entry("check sound device: 1 char name",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Sound: &v1.SoundDevice{
							Name: "a",
						},
					},
				},
			}, 0),
		Entry("check sound device: model = ich9",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Sound: &v1.SoundDevice{
							Name:  "name",
							Model: "ich9",
						},
					},
				},
			}, 0),
		Entry("check sound device: model = ac97",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Sound: &v1.SoundDevice{
							Name:  "name",
							Model: "ac97",
						},
					},
				},
			}, 0),
		Entry("check sound device: model = other",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Sound: &v1.SoundDevice{
							Name:  "name",
							Model: "other",
						},
					},
				},
			}, 1, "spec.domain.devices.sound.model", "ac97", "ich9"),
		Entry("check interface name",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Interfaces: []v1.Interface{
							{
								Name: "aB3_e6G",
							},
						},
					},
				},
			}, 0),
		Entry("check interface name (with spaces)",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Interfaces: []v1.Interface{
							{
								Name: "interface name",
							},
						},
					},
				},
			}, 1, "spec.domain.devices.interfaces[0].name", "should match '^[A-Za-z0-9-_]+$'"),
		Entry("check interface name (with special chars)",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Interfaces: []v1.Interface{
							{
								Name: "ab@d",
							},
						},
					},
				},
			}, 1, "spec.domain.devices.interfaces[0].name", "should match '^[A-Za-z0-9-_]+$'"),
		Entry("check interface valid models",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Interfaces: []v1.Interface{
							{
								Name:  "interface1",
								Model: "e1000",
							},
							{
								Name:  "interface2",
								Model: "e1000e",
							},
							{
								Name:  "interface3",
								Model: "ne2k_pci",
							},
							{
								Name:  "interface4",
								Model: "pcnet",
							},
							{
								Name:  "interface5",
								Model: "rtl8139",
							},
							{
								Name:  "interface6",
								Model: "virtio",
							},
						},
					},
				},
			}, 0),
		Entry("check interface invalid models",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Interfaces: []v1.Interface{
							{
								Name:  "interface1",
								Model: "e1000",
							},
							{
								Name:  "interface2",
								Model: "not-vaalid-1",
							},
							{
								Name:  "interface3",
								Model: "not-vaalid-2",
							},
						},
					},
				},
			}, 2, "spec.domain.devices.interfaces[", "].model", "e1000", "e1000e", "ne2k_pci", "pcnet", "rtl8139", "virtio"),
		Entry("check interface valid state",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Interfaces: []v1.Interface{
							{
								Name:  "interface1",
								State: v1.InterfaceStateAbsent,
							},
						},
					},
				},
			}, 0),
		Entry("check interface invalid state",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Interfaces: []v1.Interface{
							{
								Name:  "interface1",
								State: "other",
							},
						},
					},
				},
			}, 1, "spec.domain.devices.interfaces[0].state", string(v1.InterfaceStateAbsent)),
		Entry("check interface port - no data",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Interfaces: []v1.Interface{
							{
								Name: "interface1",
								Ports: []v1.Port{
									{},
								},
							},
						},
					},
				},
			}, 1, "spec.domain.devices.interfaces[0].ports[0].port", "should be greater than 0"),
		Entry("check interface port - port = 0",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Interfaces: []v1.Interface{
							{
								Name: "interface1",
								Ports: []v1.Port{
									{
										Port: 0,
									},
								},
							},
						},
					},
				},
			}, 1, "spec.domain.devices.interfaces[0].ports[0].port", "should be greater than 0"),
		Entry("check interface port - port = 65536",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Interfaces: []v1.Interface{
							{
								Name: "interface1",
								Ports: []v1.Port{
									{
										Port: 65536,
									},
								},
							},
						},
					},
				},
			}, 1, "spec.domain.devices.interfaces[0].ports[0].port", "should be less than 65536"),
		Entry("check interface port - valid ports",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Interfaces: []v1.Interface{
							{
								Name: "interface1",
								Ports: []v1.Port{
									{
										Port: 1,
									},
									{
										Port: 32000,
									},
									{
										Port: 8080,
									},
									{
										Port: 65535,
									},
								},
							},
						},
					},
				},
			}, 0),
		Entry("check interface port - valid protocol",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Interfaces: []v1.Interface{
							{
								Name: "interface1",
								Ports: []v1.Port{
									{
										Port:     8080,
										Protocol: "TCP",
									},
									{
										Port:     8080,
										Protocol: "UDP",
									},
									{
										Port:     8080,
										Protocol: "",
									},
								},
							},
						},
					},
				},
			}, 0),
		Entry("check interface port - unsupported protocol",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Interfaces: []v1.Interface{
							{
								Name: "interface1",
								Ports: []v1.Port{
									{
										Port:     8080,
										Protocol: "HTTP",
									},
								},
							},
						},
					},
				},
			}, 1, "spec.domain.devices.interfaces[0].ports[0].protocol", "UDP", "TCP"),
		Entry("check CPU - valid features",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Features: []v1.CPUFeature{
							{
								Policy: "force",
							},
							{
								Policy: "require",
							},
							{
								Policy: "optional",
							},
							{
								Policy: "disable",
							},
							{
								Policy: "forbid",
							},
						},
					},
				},
			}, 0),
		Entry("check CPU - unknown features",
			v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Features: []v1.CPUFeature{
							{
								Policy: "force",
							},
							{
								Policy: "f1",
							},
							{
								Policy: "f2",
							},
						},
					},
				},
			}, 2, "spec.domain.cpu.features[", "].policy", "force", "require", "optional", "disable", "forbid"),
	)
})
