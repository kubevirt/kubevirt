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
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	ini "gopkg.in/ini.v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/rest"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Vmlifecycle", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)
	var vm *v1.VirtualMachine

	getVmNode := func() string {
		obj, err := virtClient.RestClient().Get().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Name(vm.GetObjectMeta().GetName()).Do().Get()
		Expect(err).ToNot(HaveOccurred())
		return obj.(*v1.VirtualMachine).Status.NodeName
	}

	checkSpiceConnection := func() {
		raw, err := virtClient.RestClient().Get().Resource("virtualmachines").SetHeader("Accept", rest.MIME_INI).SubResource("spice").Namespace(tests.NamespaceTestDefault).Name(vm.GetObjectMeta().GetName()).Do().Raw()
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
	}

	BeforeEach(func() {
		vm = tests.NewRandomVM()
		tests.BeforeTestCleanup()
	})

	Context("New VM with a spice connection given", func() {

		It("should return no connection details if VM does not exist", func(done Done) {
			result := virtClient.RestClient().Get().Resource("virtualmachines").SubResource("spice").Namespace(tests.NamespaceTestDefault).Name("something-random").Do()
			Expect(result.Error()).NotTo(BeNil())
			close(done)
		}, 3)

		It("should return connection details for running VMs in ini format", func(done Done) {
			// Create the VM
			result := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do()
			obj, err := result.Get()
			Expect(err).To(BeNil())

			// Block until the VM is running
			tests.WaitForSuccessfulVMStart(obj)

			raw, err := virtClient.RestClient().Get().Resource("virtualmachines").SetHeader("Accept", rest.MIME_INI).SubResource("spice").Namespace(tests.NamespaceTestDefault).Name(vm.GetObjectMeta().GetName()).Do().Raw()
			spice, err := ini.Load(raw)
			Expect(err).To(Not(HaveOccurred()))

			Expect(spice.Section("virt-viewer")).NotTo(BeNil())
			section := spice.Section("virt-viewer")
			Expect(section.HasKey("type")).To(BeTrue())
			Expect(section.HasKey("host")).To(BeTrue())
			Expect(section.HasKey("port")).To(BeTrue())
			Expect(strconv.Atoi(section.Key("port").Value())).To(BeNumerically(">=", int32(5900)))
			Expect(section.HasKey("proxy")).To(BeTrue())
			close(done)
		}, 30)

		It("should return connection details for running VMs in json format", func(done Done) {
			// Create the VM
			result := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do()
			obj, err := result.Get()
			Expect(err).To(BeNil())

			// Block until the VM is running
			tests.WaitForSuccessfulVMStart(obj)

			obj, err = virtClient.RestClient().Get().Resource("virtualmachines").SetHeader("Accept", rest.MIME_JSON).SubResource("spice").Namespace(tests.NamespaceTestDefault).Name(vm.GetObjectMeta().GetName()).Do().Get()
			Expect(err).To(BeNil())
			spice := obj.(*v1.Spice).Info
			Expect(spice.Type).To(Equal("spice"))
			Expect(spice.Port).To(BeNumerically(">=", int32(5900)))
			close(done)
		}, 30)

		It("should allow accessing the spice device on the VM", func(done Done) {
			// Create the VM
			result := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do()
			obj, err := result.Get()
			Expect(err).To(BeNil())

			// Block until the VM is running
			tests.WaitForSuccessfulVMStart(obj)

			checkSpiceConnection()

			close(done)
		}, 30)
	})

	Context("Two new VMs scheduled on the same node with a spice graphics device", func() {

		It("should start without port clashes", func(done Done) {
			// Create the VM
			result := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do()
			obj, err := result.Get()
			Expect(err).To(BeNil())

			// Block until the VM is running
			nodeName := tests.WaitForSuccessfulVMStart(obj)

			vm1 := tests.NewRandomVM()
			vm1.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": nodeName}
			result = virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm1).Do()
			obj1, err := result.Get()
			Expect(err).To(BeNil())

			// Block until the VM is running
			tests.WaitForSuccessfulVMStart(obj1)

			obj, err = virtClient.RestClient().Get().Resource("virtualmachines").SetHeader("Accept", rest.MIME_JSON).SubResource("spice").Namespace(tests.NamespaceTestDefault).Name(vm.GetObjectMeta().GetName()).Do().Get()
			Expect(err).To(BeNil())
			obj1, err = virtClient.RestClient().Get().Resource("virtualmachines").SetHeader("Accept", rest.MIME_JSON).SubResource("spice").Namespace(tests.NamespaceTestDefault).Name(vm1.GetObjectMeta().GetName()).Do().Get()
			Expect(err).To(BeNil())
			Expect(obj.(*v1.Spice).Info.Port).ToNot(BeNumerically("==", obj1.(*v1.Spice).Info.Port))
			close(done)
		}, 30)
	})

	Context("Migrate VM with spice connection", func() {

		BeforeEach(func() {
			if len(tests.GetReadyNodes()) < 2 {
				Skip("To test migrations, at least two nodes need to be active")
			}
			// Create the VM
			result := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do()
			obj, err := result.Get()
			Expect(err).To(BeNil())

			// Block until the VM is running
			tests.WaitForSuccessfulVMStart(obj)
		})

		It("should allow accessing the spice device on the VM", func() {
			sourceNode := getVmNode()

			migration := tests.NewRandomMigrationForVm(vm)
			err = virtClient.RestClient().Post().Resource("migrations").Namespace(tests.NamespaceTestDefault).Body(migration).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			Eventually(getVmNode, 30*time.Second, time.Second).ShouldNot(Equal(sourceNode))

			Eventually(func() v1.VMPhase {
				obj, err := virtClient.RestClient().Get().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Name(vm.GetObjectMeta().GetName()).Do().Get()
				Expect(err).ToNot(HaveOccurred())
				fetchedVM := obj.(*v1.VirtualMachine)
				return fetchedVM.Status.Phase
			}, 10*time.Second, time.Second).Should(Equal(v1.Running))

			checkSpiceConnection()
		})
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
