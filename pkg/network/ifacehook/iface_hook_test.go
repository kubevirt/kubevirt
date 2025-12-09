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
 */

package ifacehook

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"libvirt.org/go/libvirtxml"

	"kubevirt.io/kubevirt/pkg/libvmi"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Premigration Hook Server", func() {
	Context("Oridinal Interface  Hook", func() {
		It("should change interface targets to hashed", func() {
			////
			//mac
			//target
			//model
			//mtu
			//rom
			//alias
			////
			domXML := `
<domain xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0" type="kvm">
  <name>kubevirt</name>
  <devices>
    <interface type="ethernet">
	  <mac address="02:66:7b:d1:ed:3f"/>
      <target dev="tap0" managed="no"/>
      <model type="virtio-non-transitional"/>
      <mtu size="1430"/>
  	  <rom enabled="no"/>
      <alias name="ua-default"/>
    </interface>
    <interface type="ethernet">
 <mac address="02:66:7b:d1:ed:40"/>
      <target dev="tap1" managed="no"/>
      <model type="virtio-non-transitional"/>
      <mtu size="1400"/>
<rom enabled="no"/>
      <alias name="ua-sec"/>
      
    </interface>
</devices>
</domain>
`
			expectedXML := `
<domain type="kvm">
  <name>kubevirt</name>
  <devices>
    <interface type="ethernet">
      <mac address="02:66:7b:d1:ed:3f"/>
      <target dev="tap0" managed="no"/>
      <model type="virtio-non-transitional"/>
      <mtu size="1430"/>
  	  <rom enabled="no"/>
      <alias name="ua-default"/>
    </interface>
    <interface type="ethernet">
  <mac address="02:66:7b:d1:ed:40"/>
      <target dev="tapadd93534eeb" managed="no"/>
      <model type="virtio-non-transitional"/>
      <mtu size="1400"/>
  <rom enabled="no"/>
      <alias name="ua-sec"/>
    
    </interface>
</devices>
</domain>
`

			By("creating a VMI with dedicated CPU cores")
			vmi := libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("sec")),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork("sec", "secnad")),
			)

			By("parsing the input domain XML")
			var domain libvirtxml.Domain
			err := domain.Unmarshal(domXML)
			Expect(err).NotTo(HaveOccurred(), "failed to parse input domain XML")

			By("running the CPU dedicated hook")
			err = HasheIfaceNameHook(vmi, &domain)
			Expect(err).NotTo(HaveOccurred(), "failed to modify domain")

			By("marshaling the modified domain back to XML")
			newXML, err := domain.Marshal()
			Expect(err).NotTo(HaveOccurred(), "failed to marshal modified domain")

			By("ensuring the generated XML is accurate")
			Expect(newXML).To(MatchXML(expectedXML), "the target XML is not as expected")
		})
	})
})
