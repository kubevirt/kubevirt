/*
 * This file is part of the kubevirt project
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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests_test

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/ini.v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/rest"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Vmlifecycle", func() {

	flag.Parse()

	restClient, err := kubecli.GetRESTClient()
	tests.PanicOnError(err)
	var vm *v1.VM

	BeforeEach(func() {
		vm = tests.NewRandomVMWithSpice()
		tests.BeforeTestCleanup()
	})

	Context("New VM with a spice connection given", func() {

		It("should return no connection details if VM does not exist", func(done Done) {
			result := restClient.Get().Resource("vms").SubResource("spice").Namespace(tests.NamespaceTestDefault).Name("something-random").Do()
			Expect(result.Error()).NotTo(BeNil())
			close(done)
		}, 3)

		It("should return connection details for running VMs in ini format", func(done Done) {
			// Create the VM
			result := restClient.Post().Resource("vms").Namespace(tests.NamespaceTestDefault).Body(vm).Do()
			obj, err := result.Get()
			Expect(err).To(BeNil())

			// Block until the VM is running
			tests.WaitForSuccessfulVMStart(obj)

			raw, err := restClient.Get().Resource("vms").SetHeader("Accept", rest.MIME_INI).SubResource("spice").Namespace(tests.NamespaceTestDefault).Name(vm.GetObjectMeta().GetName()).Do().Raw()
			spice, err := ini.Load(raw)
			Expect(err).To(Not(HaveOccurred()))

			Expect(spice.Section("virt-viewer")).NotTo(BeNil())
			section := spice.Section("virt-viewer")
			Expect(section.HasKey("type")).To(BeTrue())
			Expect(section.HasKey("host")).To(BeTrue())
			Expect(section.HasKey("port")).To(BeTrue())
			Expect(section.Key("port").Value()).To(Equal("4000"))
			Expect(section.HasKey("proxy")).To(BeTrue())
			close(done)
		}, 30)

		It("should return connection details for running VMs in json format", func(done Done) {
			// Create the VM
			result := restClient.Post().Resource("vms").Namespace(tests.NamespaceTestDefault).Body(vm).Do()
			obj, err := result.Get()
			Expect(err).To(BeNil())

			// Block until the VM is running
			tests.WaitForSuccessfulVMStart(obj)

			obj, err = restClient.Get().Resource("vms").SetHeader("Accept", rest.MIME_JSON).SubResource("spice").Namespace(tests.NamespaceTestDefault).Name(vm.GetObjectMeta().GetName()).Do().Get()
			Expect(err).To(BeNil())
			spice := obj.(*v1.Spice).Info
			Expect(spice.Type).To(Equal("spice"))
			Expect(spice.Port).To(Equal(int32(4000)))
			close(done)
		}, 30)

		It("should allow accessing the spice device on the VM", func(done Done) {
			// Create the VM
			result := restClient.Post().Resource("vms").Namespace(tests.NamespaceTestDefault).Body(vm).Do()
			obj, err := result.Get()
			Expect(err).To(BeNil())

			// Block until the VM is running
			tests.WaitForSuccessfulVMStart(obj)

			raw, err := restClient.Get().Resource("vms").SetHeader("Accept", rest.MIME_INI).SubResource("spice").Namespace(tests.NamespaceTestDefault).Name(vm.GetObjectMeta().GetName()).Do().Raw()
			Expect(err).To(BeNil())
			spiceINI, err := ini.Load(raw)
			Expect(err).NotTo(HaveOccurred())
			spice := v1.SpiceInfo{}
			Expect(spiceINI.Section("virt-viewer").MapTo(&spice)).To(Succeed())

			proxy := strings.TrimPrefix(spice.Proxy, "http://")
			host := fmt.Sprintf("%s:%d", spice.Host, spice.Port)

			// Let's see if we can connect to the spice port through the proxy
			conn, err := net.Dial("tcp", proxy)
			Expect(err).To(BeNil())
			conn.Write([]byte("CONNECT " + host + " HTTP/1.1\r\n"))
			conn.Write([]byte("Host: " + host + "\r\n"))
			conn.Write([]byte("\r\n"))
			line, err := bufio.NewReader(conn).ReadString('\n')
			Expect(err).To(BeNil())
			Expect(strings.TrimSpace(line)).To(Equal("HTTP/1.1 200 Connection established"))

			// Let's send a spice handshake
			conn.Write(newSpiceHandshake())

			// Let's parse the response
			var i int32
			x := make([]byte, 4, 4)
			io.ReadFull(conn, x)
			Expect(string(x)).To(Equal("REDQ")) // spice magic
			binary.Read(conn, binary.LittleEndian, &i)
			Expect(i).To(Equal(int32(2)), "Major version does not match.")
			binary.Read(conn, binary.LittleEndian, &i)
			Expect(i).To(Equal(int32(2)), "Minor version does not match.")
			binary.Read(conn, binary.LittleEndian, &i)
			Expect(i).To(BeNumerically(">", 4), "Message not long enough.")
			binary.Read(conn, binary.LittleEndian, &i)
			Expect(i).To(Equal(int32(0)), "Message status is not OK.") // 0 is equal to OK

			close(done)
		}, 30)
	})

})

func newSpiceHandshake() []byte {
	var b []byte
	bb := bytes.NewBuffer(b)
	bb.Write([]byte("REDQ"))                                  // spice magic
	binary.Write(bb, binary.LittleEndian, uint32(2))          // protocol major version
	binary.Write(bb, binary.LittleEndian, uint32(2))          // protocol minor version
	binary.Write(bb, binary.LittleEndian, uint32(22))         // message size
	binary.Write(bb, binary.LittleEndian, uint32(rand.Int())) // session id
	binary.Write(bb, binary.LittleEndian, uint8(3))           // channel type
	binary.Write(bb, binary.LittleEndian, uint8(0))           // channel id
	binary.Write(bb, binary.LittleEndian, uint32(1))          // number of common capabilities
	binary.Write(bb, binary.LittleEndian, uint32(0))          // number of channel capabilities
	binary.Write(bb, binary.LittleEndian, uint32(18))         // capabilities offset
	binary.Write(bb, binary.LittleEndian, uint32(13))         // client common capabilities
	return bb.Bytes()
}
