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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package network

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Common Methods", func() {
	tempFile := "/tmp/data"

	Context("Function ParseNameservers()", func() {
		It("should return a byte array of nameservers", func() {
			ns1, ns2 := []uint8{8, 8, 8, 8}, []uint8{8, 8, 4, 4}
			resolvConf := "nameserver 8.8.8.8\nnameserver 8.8.4.4\n"
			nameservers, err := ParseNameservers(resolvConf)
			Expect(nameservers).To(Equal([][]uint8{ns1, ns2}))
			Expect(err).To(BeNil())
		})

		It("should ignore non-nameserver lines and malformed nameserver lines", func() {
			ns1, ns2 := []uint8{8, 8, 8, 8}, []uint8{8, 8, 4, 4}
			resolvConf := "search example.com\nnameserver 8.8.8.8\nnameserver 8.8.4.4\nnameserver mynameserver\n"
			nameservers, err := ParseNameservers(resolvConf)
			Expect(nameservers).To(Equal([][]uint8{ns1, ns2}))
			Expect(err).To(BeNil())
		})

		It("should return a default nameserver if none is parsed", func() {
			nameservers, err := ParseNameservers("")
			expectedDNS := net.ParseIP(defaultDNS).To4()
			Expect(nameservers).To(Equal([][]uint8{expectedDNS}))
			Expect(err).To(BeNil())
		})
	})

	Context("Function ParseSearchDomains()", func() {
		It("should return a string of search domains", func() {
			resolvConf := "search cluster.local svc.cluster.local example.com\nnameserver 8.8.8.8\n"
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{"cluster.local", "svc.cluster.local", "example.com"}))
			Expect(err).To(BeNil())
		})

		It("should handle multi-line search domains", func() {
			resolvConf := "search cluster.local\nsearch svc.cluster.local example.com\nnameserver 8.8.8.8\n"
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{"cluster.local", "svc.cluster.local", "example.com"}))
			Expect(err).To(BeNil())
		})

		It("should clean up extra whitespace between search domains", func() {
			resolvConf := "search cluster.local\tsvc.cluster.local    example.com\nnameserver 8.8.8.8\n"
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{"cluster.local", "svc.cluster.local", "example.com"}))
			Expect(err).To(BeNil())
		})

		It("should handle non-presence of search domains by returning default search domain", func() {
			resolvConf := fmt.Sprintf("nameserver %s\n", defaultDNS)
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{defaultSearchDomain}))
			Expect(err).To(BeNil())
		})

		It("should allow partial search domains", func() {
			resolvConf := "search local\nnameserver 8.8.8.8\n"
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{"local"}))
			Expect(err).To(BeNil())
		})
	})
	Context("Function setCachedInterface()", func() {
		It("should persist interface payload", func() {
			tmpDir, _ := ioutil.TempDir("", "commontest")
			setInterfaceCacheFile(tmpDir + "/cache-%s.json")

			ifaceName := "iface_name"
			iface := api.Interface{Type: "fake_type", Source: api.InterfaceSource{Bridge: "fake_br"}}
			err := setCachedInterface(ifaceName, &iface)
			Expect(err).ToNot(HaveOccurred())

			cached_iface, err := getCachedInterface(ifaceName)
			Expect(err).ToNot(HaveOccurred())

			Expect(iface).To(Equal(*cached_iface))
		})
	})

	Context("Function Check bridge interface", func() {

	})
	Context("Function Check slirp interface", func() {

	})
	Context("Functions read and write", func() {
		data := `{
"QEMUEnv": null,
"QEMUArg": [
	{
	"Value": "-device"
	},
	{
	"Value": "virtio,netdev=testnet"
	}
]
}`

		AfterEach(func() {
			os.Remove(tempFile)
		})

		It("Should create a file with command line configuration", func() {
			commandLine := new(api.Commandline)
			commandLine.QEMUArg = []api.Arg{{Value: "-device"}, {Value: fmt.Sprintf("%s,netdev=%s", "virtio", "testnet")}}

			err := WriteToCachedFile(*commandLine, tempFile)
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should read from a file and convert the data", func() {
			err := ioutil.WriteFile(tempFile, []byte(data), 0644)
			Expect(err).ToNot(HaveOccurred())

			var commandLine = api.Commandline{}
			err = ReadFromCachedFile(tempFile, &commandLine)
			Expect(err).ToNot(HaveOccurred())
			Expect(commandLine.QEMUArg[0].Value).To(Equal("-device"))
		})
	})
})
