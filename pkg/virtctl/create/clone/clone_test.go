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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package clone_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/yaml"
	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"

	"kubevirt.io/kubevirt/pkg/virtctl/create/clone"
	"kubevirt.io/kubevirt/tests/clientcmd"
)

const (
	create = "create"

	vmKind, vmApiGroup             = "VirtualMachine", "kubevirt.io"
	snapshotKind, snapshotApiGroup = "VirtualMachineSnapshot", "snapshot.kubevirt.io"
)
const (
	LabelFilters = iota
	AnnotationsFilters
	LabelAndAnnotationsFilters
	TemplateLabelFilters
	TemplateAnnotationsFilters
	TemplateLabelAndAnnotationsFilters
)

var _ = Describe("create clone", func() {
	Context("required arguments", func() {

		It("source name must be specified", func() {
			_, err := newCommand()
			Expect(err).To(HaveOccurred())
		})

		It("source name is the only required argument", func() {
			flags := addFlag(nil, clone.SourceNameFlag, "fake-name")
			_, err := newCommand(flags...)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("source and target", func() {

		DescribeTable("supported types", func(sourceType, expectedSourceKind, expectedSourceApiGroup, targetType, expectedTargetKind, expectedTargetApiGroup string) {
			const sourceName, targetName = "source-name", "target-name"

			flags := addFlag(nil, clone.SourceNameFlag, sourceName)
			flags = addFlag(flags, clone.TargetNameFlag, targetName)
			flags = addFlag(flags, clone.SourceTypeFlag, sourceType)
			flags = addFlag(flags, clone.TargetTypeFlag, targetType)

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
			flags := addFlag(nil, clone.SourceNameFlag, "source-name")
			flags = addFlag(flags, clone.TargetNameFlag, "target-name")
			flags = addFlag(flags, clone.TargetTypeFlag, "snapshot")

			_, err := newCommand(flags...)
			Expect(err).To(HaveOccurred())
		})

		It("unknown source type", func() {
			flags := getSourceNameFlags()
			flags = addFlag(flags, clone.SourceTypeFlag, "unknown type")

			_, err := newCommand(flags...)
			Expect(err).To(HaveOccurred())
		})

		It("unknown target type", func() {
			flags := getSourceNameFlags()
			flags = addFlag(flags, clone.TargetTypeFlag, "unknown type")

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
			case LabelFilters:
				flags = addFlag(flags, clone.LabelFilterFlag, "*")
				flags = addFlag(flags, clone.LabelFilterFlag, `"!some/key"`)
			case AnnotationsFilters:
				flags = addFlag(flags, clone.AnnotationFilterFlag, "*")
				flags = addFlag(flags, clone.AnnotationFilterFlag, `"!some/key"`)
			case LabelAndAnnotationsFilters:
				flags = addFlag(flags, clone.LabelFilterFlag, "*")
				flags = addFlag(flags, clone.LabelFilterFlag, `"!some/key"`)
				flags = addFlag(flags, clone.AnnotationFilterFlag, "*")
				flags = addFlag(flags, clone.AnnotationFilterFlag, `"!some/key"`)
			case TemplateLabelFilters:
				flags = addFlag(flags, clone.TemplateLabelFilterFlag, "*")
				flags = addFlag(flags, clone.TemplateLabelFilterFlag, `"!some/key"`)
			case TemplateAnnotationsFilters:
				flags = addFlag(flags, clone.TemplateAnnotationFilterFlag, "*")
				flags = addFlag(flags, clone.TemplateAnnotationFilterFlag, `"!some/key"`)
			case TemplateLabelAndAnnotationsFilters:
				flags = addFlag(flags, clone.TemplateLabelFilterFlag, "*")
				flags = addFlag(flags, clone.TemplateLabelFilterFlag, `"!some/key"`)
				flags = addFlag(flags, clone.TemplateAnnotationFilterFlag, "*")
				flags = addFlag(flags, clone.TemplateAnnotationFilterFlag, `"!some/key"`)
			}

			cloneObj, err := newCommand(flags...)
			Expect(err).ToNot(HaveOccurred())
			const expectedLen = 2

			switch filterType {
			case LabelFilters:
				Expect(cloneObj.Spec.LabelFilters).To(HaveLen(expectedLen))
			case AnnotationsFilters:
				Expect(cloneObj.Spec.AnnotationFilters).To(HaveLen(expectedLen))
			case LabelAndAnnotationsFilters:
				Expect(cloneObj.Spec.LabelFilters).To(HaveLen(expectedLen))
				Expect(cloneObj.Spec.AnnotationFilters).To(HaveLen(expectedLen))
			case TemplateLabelFilters:
				Expect(cloneObj.Spec.Template.LabelFilters).To(HaveLen(expectedLen))
			case TemplateAnnotationsFilters:
				Expect(cloneObj.Spec.Template.AnnotationFilters).To(HaveLen(expectedLen))
			case TemplateLabelAndAnnotationsFilters:
				Expect(cloneObj.Spec.Template.LabelFilters).To(HaveLen(expectedLen))
				Expect(cloneObj.Spec.Template.AnnotationFilters).To(HaveLen(expectedLen))
			}

		},
			Entry("label filters", LabelFilters),
			Entry("annotation filters", AnnotationsFilters),
			Entry("label and annotation filters", LabelAndAnnotationsFilters),
			Entry("template-label filters", TemplateLabelFilters),
			Entry("template-annotation filters", TemplateAnnotationsFilters),
			Entry("template-label and template-annotation filters", TemplateLabelAndAnnotationsFilters),
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
				flags = addFlag(flags, clone.NewMacAddressesFlag, fmt.Sprintf(newMacAddressValueFmt, interfaceName, newMacAddress))
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
			flags = addFlag(flags, clone.NewMacAddressesFlag, fmt.Sprintf(newMacAddressValueFmt, interfaceName, newMacAddress))

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
		flags = addFlag(flags, clone.NewSMBiosSerialFlag, newSerial)

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

func newCommand(createCloneFlags ...string) (*clonev1alpha1.VirtualMachineClone, error) {
	baseArgs := []string{create, clone.Clone}
	args := append(baseArgs, createCloneFlags...)

	bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(args...)()
	if err != nil {
		return nil, err
	}

	cloneObj := clonev1alpha1.VirtualMachineClone{}
	err = yaml.Unmarshal(bytes, &cloneObj)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return &cloneObj, nil
}

func getSourceNameFlags() []string {
	return addFlag(nil, clone.SourceNameFlag, "source-vm")
}
