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

package libvirtxml

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("QEMU commandline", func() {
	Context("TransferQEMUCommandline", func() {
		It("should extract args and envs from domain XML", func() {
			domainXML := `<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
				<name>test-vm</name>
				<qemu:commandline>
					<qemu:arg value="-fw_cfg"/>
					<qemu:arg value="name=opt/com.coreos/config,file=/data.ign"/>
					<qemu:env name="QEMU_AUDIO_DRV" value="none"/>
				</qemu:commandline>
			</domain>`

			spec := &api.DomainSpec{}
			TransferQEMUCommandline(domainXML, spec)

			Expect(spec.XmlNS).To(Equal("http://libvirt.org/schemas/domain/qemu/1.0"))
			Expect(spec.QEMUCmd).NotTo(BeNil())
			Expect(spec.QEMUCmd.QEMUArg).To(HaveLen(2))
			Expect(spec.QEMUCmd.QEMUArg[0].Value).To(Equal("-fw_cfg"))
			Expect(spec.QEMUCmd.QEMUArg[1].Value).To(Equal("name=opt/com.coreos/config,file=/data.ign"))
			Expect(spec.QEMUCmd.QEMUEnv).To(HaveLen(1))
			Expect(spec.QEMUCmd.QEMUEnv[0].Name).To(Equal("QEMU_AUDIO_DRV"))
			Expect(spec.QEMUCmd.QEMUEnv[0].Value).To(Equal("none"))
		})

		It("should set XmlNS but not QEMUCmd for empty commandline", func() {
			domainXML := `<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
				<name>test-vm</name>
				<qemu:commandline></qemu:commandline>
			</domain>`

			spec := &api.DomainSpec{}
			TransferQEMUCommandline(domainXML, spec)

			Expect(spec.XmlNS).To(Equal("http://libvirt.org/schemas/domain/qemu/1.0"))
			Expect(spec.QEMUCmd).To(BeNil())
		})

		It("should not modify spec when no qemu namespace is present", func() {
			domainXML := `<domain type="kvm">
				<name>test-vm</name>
				<devices><disk type="file"/></devices>
			</domain>`

			spec := &api.DomainSpec{}
			TransferQEMUCommandline(domainXML, spec)

			Expect(spec.XmlNS).To(BeEmpty())
			Expect(spec.QEMUCmd).To(BeNil())
		})

		It("should handle envs without value attribute", func() {
			domainXML := `<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
				<name>test-vm</name>
				<qemu:commandline>
					<qemu:env name="DISPLAY"/>
				</qemu:commandline>
			</domain>`

			spec := &api.DomainSpec{}
			TransferQEMUCommandline(domainXML, spec)

			Expect(spec.QEMUCmd).NotTo(BeNil())
			Expect(spec.QEMUCmd.QEMUEnv).To(HaveLen(1))
			Expect(spec.QEMUCmd.QEMUEnv[0].Name).To(Equal("DISPLAY"))
			Expect(spec.QEMUCmd.QEMUEnv[0].Value).To(BeEmpty())
		})

		It("should skip env entries without a name attribute", func() {
			domainXML := `<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
				<name>test-vm</name>
				<qemu:commandline>
					<qemu:env value="orphan"/>
					<qemu:env name="VALID" value="yes"/>
				</qemu:commandline>
			</domain>`

			spec := &api.DomainSpec{}
			TransferQEMUCommandline(domainXML, spec)

			Expect(spec.QEMUCmd).NotTo(BeNil())
			Expect(spec.QEMUCmd.QEMUEnv).To(HaveLen(1))
			Expect(spec.QEMUCmd.QEMUEnv[0].Name).To(Equal("VALID"))
		})

		It("should include arg with empty value attribute", func() {
			domainXML := `<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
				<name>test-vm</name>
				<qemu:commandline>
					<qemu:arg value=""/>
					<qemu:arg value="-nographic"/>
				</qemu:commandline>
			</domain>`

			spec := &api.DomainSpec{}
			TransferQEMUCommandline(domainXML, spec)

			Expect(spec.QEMUCmd).NotTo(BeNil())
			Expect(spec.QEMUCmd.QEMUArg).To(HaveLen(2))
			Expect(spec.QEMUCmd.QEMUArg[0].Value).To(BeEmpty())
			Expect(spec.QEMUCmd.QEMUArg[1].Value).To(Equal("-nographic"))
		})

		It("should aggregate args across multiple commandline blocks", func() {
			domainXML := `<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
				<name>test-vm</name>
				<qemu:commandline>
					<qemu:arg value="-fw_cfg"/>
				</qemu:commandline>
				<qemu:commandline>
					<qemu:arg value="-device"/>
				</qemu:commandline>
			</domain>`

			spec := &api.DomainSpec{}
			TransferQEMUCommandline(domainXML, spec)

			Expect(spec.QEMUCmd).NotTo(BeNil())
			Expect(spec.QEMUCmd.QEMUArg).To(HaveLen(2))
			Expect(spec.QEMUCmd.QEMUArg[0].Value).To(Equal("-fw_cfg"))
			Expect(spec.QEMUCmd.QEMUArg[1].Value).To(Equal("-device"))
		})

		It("should skip qemu elements outside commandline block", func() {
			domainXML := `<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
				<name>test-vm</name>
				<qemu:arg value="stray"/>
				<qemu:commandline>
					<qemu:arg value="real"/>
				</qemu:commandline>
			</domain>`

			spec := &api.DomainSpec{}
			TransferQEMUCommandline(domainXML, spec)

			Expect(spec.QEMUCmd).NotTo(BeNil())
			Expect(spec.QEMUCmd.QEMUArg).To(HaveLen(1))
			Expect(spec.QEMUCmd.QEMUArg[0].Value).To(Equal("real"))
		})
	})

})
