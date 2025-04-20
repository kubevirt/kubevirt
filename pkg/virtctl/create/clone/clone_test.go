/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package clone_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/yaml"
	clone "kubevirt.io/api/clone/v1beta1"

	"kubevirt.io/kubevirt/pkg/virtctl/testing"

	virtctlclone "kubevirt.io/kubevirt/pkg/virtctl/create/clone"
)

const (
	create = "create"

	vmKind, vmApiGroup             = "VirtualMachine", "kubevirt.io"
	snapshotKind, snapshotApiGroup = "VirtualMachineSnapshot", "snapshot.kubevirt.io"
)
const (
	labelFilters = iota
	annotationsFilters
	labelAndAnnotationsFilters
	templateLabelFilters
	templateAnnotationsFilters
	templateLabelAndAnnotationsFilters
)

var _ = Describe("create clone", func() {
	Context("required arguments", func() {

		It("source name must be specified", func() {
			_, err := newCommand()
			Expect(err).To(HaveOccurred())
		})

		It("source name is the only required argument", func() {
			flags := addFlag(nil, virtctlclone.SourceNameFlag, "fake-name")
			_, err := newCommand(flags...)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("source and target", func() {

		DescribeTable("supported types", func(sourceType, expectedSourceKind, expectedSourceApiGroup, targetType, expectedTargetKind, expectedTargetApiGroup string) {
			const sourceName, targetName = "source-name", "target-name"

			flags := addFlag(nil, virtctlclone.SourceNameFlag, sourceName)
			flags = addFlag(flags, virtctlclone.TargetNameFlag, targetName)
			flags = addFlag(flags, virtctlclone.SourceTypeFlag, sourceType)
			flags = addFlag(flags, virtctlclone.TargetTypeFlag, targetType)

			cloneObj, err := newCommand(flags...)
			Expect(err).ToNot(HaveOccurred())

			Expect(cloneObj.Spec.Source.Name).To(Equal(sourceName))
			Expect(cloneObj.Spec.Source.Kind).To(Equal(expectedSourceKind))
			Expect(cloneObj.Spec.Source.APIGroup).ToNot(BeNil())
			Expect(*cloneObj.Spec.Source.APIGroup).To(Equal(expectedSourceApiGroup))

			Expect(cloneObj.Spec.Target.Name).To(Equal(targetName))
			Expect(cloneObj.Spec.Target.Kind).To(Equal(expectedTargetKind))
			Expect(cloneObj.Spec.Target.APIGroup).ToNot(BeNil())
			Expect(*cloneObj.Spec.Target.APIGroup).To(Equal(expectedTargetApiGroup))
		},
			Entry("vm source, vm target", "vm", vmKind, vmApiGroup, "vm", vmKind, vmApiGroup),
			Entry("VM source, vm target", "VM", vmKind, vmApiGroup, "vm", vmKind, vmApiGroup),
			Entry("VirtualMachine source, vm target", "VirtualMachine", vmKind, vmApiGroup, "vm", vmKind, vmApiGroup),
			Entry("virtualmachine, vm target", "virtualmachine", vmKind, vmApiGroup, "vm", vmKind, vmApiGroup),
			Entry("vm source, VirtualMachine target", "vm", vmKind, vmApiGroup, "VirtualMachine", vmKind, vmApiGroup),

			Entry("snapshot source, vm target", "snapshot", snapshotKind, snapshotApiGroup, "vm", vmKind, vmApiGroup),
			Entry("VirtualMachineSnapshot source, vm target", "VirtualMachineSnapshot", snapshotKind, snapshotApiGroup, "vm", vmKind, vmApiGroup),
			Entry("vmsnapshot source, vm target", "vmsnapshot", snapshotKind, snapshotApiGroup, "vm", vmKind, vmApiGroup),
			Entry("VMSnapshot source, vm target", "VMSnapshot", snapshotKind, snapshotApiGroup, "vm", vmKind, vmApiGroup),
		)

		It("snapshot is not supported as a target type", func() {
			flags := addFlag(nil, virtctlclone.SourceNameFlag, "source-name")
			flags = addFlag(flags, virtctlclone.TargetNameFlag, "target-name")
			flags = addFlag(flags, virtctlclone.TargetTypeFlag, "snapshot")

			_, err := newCommand(flags...)
			Expect(err).To(HaveOccurred())
		})

		It("unknown source type", func() {
			flags := getSourceNameFlags()
			flags = addFlag(flags, virtctlclone.SourceTypeFlag, "unknown type")

			_, err := newCommand(flags...)
			Expect(err).To(HaveOccurred())
		})

		It("unknown target type", func() {
			flags := getSourceNameFlags()
			flags = addFlag(flags, virtctlclone.TargetTypeFlag, "unknown type")

			_, err := newCommand(flags...)
			Expect(err).To(HaveOccurred())
		})

		It("no source name", func() {
			_, err := newCommand()
			Expect(err).To(HaveOccurred())
		})

	})

	Context("label and annotation filters", func() {

		DescribeTable("with", func(filterType int) {
			flags := getSourceNameFlags()

			switch filterType {
			case labelFilters:
				flags = addFlag(flags, virtctlclone.LabelFilterFlag, "*")
				flags = addFlag(flags, virtctlclone.LabelFilterFlag, `"!some/key"`)
			case annotationsFilters:
				flags = addFlag(flags, virtctlclone.AnnotationFilterFlag, "*")
				flags = addFlag(flags, virtctlclone.AnnotationFilterFlag, `"!some/key"`)
			case labelAndAnnotationsFilters:
				flags = addFlag(flags, virtctlclone.LabelFilterFlag, "*")
				flags = addFlag(flags, virtctlclone.LabelFilterFlag, `"!some/key"`)
				flags = addFlag(flags, virtctlclone.AnnotationFilterFlag, "*")
				flags = addFlag(flags, virtctlclone.AnnotationFilterFlag, `"!some/key"`)
			case templateLabelFilters:
				flags = addFlag(flags, virtctlclone.TemplateLabelFilterFlag, "*")
				flags = addFlag(flags, virtctlclone.TemplateLabelFilterFlag, `"!some/key"`)
			case templateAnnotationsFilters:
				flags = addFlag(flags, virtctlclone.TemplateAnnotationFilterFlag, "*")
				flags = addFlag(flags, virtctlclone.TemplateAnnotationFilterFlag, `"!some/key"`)
			case templateLabelAndAnnotationsFilters:
				flags = addFlag(flags, virtctlclone.TemplateLabelFilterFlag, "*")
				flags = addFlag(flags, virtctlclone.TemplateLabelFilterFlag, `"!some/key"`)
				flags = addFlag(flags, virtctlclone.TemplateAnnotationFilterFlag, "*")
				flags = addFlag(flags, virtctlclone.TemplateAnnotationFilterFlag, `"!some/key"`)
			}

			cloneObj, err := newCommand(flags...)
			Expect(err).ToNot(HaveOccurred())
			const expectedLen = 2

			switch filterType {
			case labelFilters:
				Expect(cloneObj.Spec.LabelFilters).To(HaveLen(expectedLen))
			case annotationsFilters:
				Expect(cloneObj.Spec.AnnotationFilters).To(HaveLen(expectedLen))
			case labelAndAnnotationsFilters:
				Expect(cloneObj.Spec.LabelFilters).To(HaveLen(expectedLen))
				Expect(cloneObj.Spec.AnnotationFilters).To(HaveLen(expectedLen))
			case templateLabelFilters:
				Expect(cloneObj.Spec.Template.LabelFilters).To(HaveLen(expectedLen))
			case templateAnnotationsFilters:
				Expect(cloneObj.Spec.Template.AnnotationFilters).To(HaveLen(expectedLen))
			case templateLabelAndAnnotationsFilters:
				Expect(cloneObj.Spec.Template.LabelFilters).To(HaveLen(expectedLen))
				Expect(cloneObj.Spec.Template.AnnotationFilters).To(HaveLen(expectedLen))
			}

		},
			Entry("label filters", labelFilters),
			Entry("annotation filters", annotationsFilters),
			Entry("label and annotation filters", labelAndAnnotationsFilters),
			Entry("template-label filters", templateLabelFilters),
			Entry("template-annotation filters", templateAnnotationsFilters),
			Entry("template-label and template-annotation filters", templateLabelAndAnnotationsFilters),
		)
	})

	Context("new mac addresses", func() {
		const newMacAddressValueFmt = `%s:%s`

		It("with valid arguments", func() {
			flags := getSourceNameFlags()

			newMacAddresses := map[string]string{
				"interface0": "custom-mac-address0",
				"interface1": "custom-mac-address1",
			}

			for interfaceName, newMacAddress := range newMacAddresses {
				flags = addFlag(flags, virtctlclone.NewMacAddressesFlag, fmt.Sprintf(newMacAddressValueFmt, interfaceName, newMacAddress))
			}

			cloneObj, err := newCommand(flags...)
			Expect(err).ToNot(HaveOccurred())

			for interfaceName, newMacAddress := range cloneObj.Spec.NewMacAddresses {
				expectedAddress, exists := cloneObj.Spec.NewMacAddresses[interfaceName]
				Expect(exists).To(BeTrue())
				Expect(newMacAddress).To(Equal(expectedAddress))
			}
		})

		DescribeTable("with invalid arguments", func(interfaceName, newMacAddress string) {
			flags := getSourceNameFlags()
			flags = addFlag(flags, virtctlclone.NewMacAddressesFlag, fmt.Sprintf(newMacAddressValueFmt, interfaceName, newMacAddress))

			_, err := newCommand(flags...)
			Expect(err).To(HaveOccurred())
		},
			Entry("empty interface name", "", "custom-mac-address"),
			Entry("empty mac address", "interface", ""),
			Entry("interface name with ':' inside its name", "interf:ace", "custom-mac-address"),
		)
	})

	It("new smbios serial", func() {
		flags := getSourceNameFlags()

		const newSerial = "newSerial"
		flags = addFlag(flags, virtctlclone.NewSMBiosSerialFlag, newSerial)

		cloneObj, err := newCommand(flags...)
		Expect(err).ToNot(HaveOccurred())

		Expect(cloneObj.Spec.NewSMBiosSerial).ToNot(BeNil())
		Expect(*cloneObj.Spec.NewSMBiosSerial).To(Equal(newSerial))
	})

	It("sets the provided namespace", func() {
		flags := getSourceNameFlags()

		const namespace = "my-namespace"
		flags = addFlag(flags, "namespace", namespace)

		cloneObj, err := newCommand(flags...)
		Expect(err).ToNot(HaveOccurred())

		Expect(cloneObj.Namespace).To(Equal(namespace))
	})

})

func addFlag(s []string, flag, value string) []string {
	return append(s, fmt.Sprintf("--%s", flag), value)
}

func newCommand(createCloneFlags ...string) (*clone.VirtualMachineClone, error) {
	baseArgs := []string{create, virtctlclone.Clone}
	args := append(baseArgs, createCloneFlags...)

	bytes, err := testing.NewRepeatableVirtctlCommandWithOut(args...)()
	if err != nil {
		return nil, err
	}

	cloneObj := clone.VirtualMachineClone{}
	err = yaml.Unmarshal(bytes, &cloneObj)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return &cloneObj, nil
}

func getSourceNameFlags() []string {
	return addFlag(nil, virtctlclone.SourceNameFlag, "source-vm")
}
