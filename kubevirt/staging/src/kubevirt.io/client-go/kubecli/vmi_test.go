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

package kubecli

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "kubevirt.io/client-go/api/v1"
)

var _ = Describe("Kubevirt VirtualMachineInstance Client", func() {

	var upgrader websocket.Upgrader
	var server *ghttp.Server
	var client KubevirtClient
	basePath := "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances"
	vmiPath := basePath + "/testvm"
	subVMPath := "/apis/subresources.kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvm"

	BeforeEach(func() {
		var err error
		server = ghttp.NewServer()
		client, err = GetKubevirtClientFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should fetch a VirtualMachineInstance", func() {
		vmi := v1.NewMinimalVMI("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", vmiPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
		))
		fetchedVMI, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).Get("testvm", &k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVMI).To(Equal(vmi))
	})

	It("should detect non existent VMIs", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", vmiPath),
			ghttp.RespondWithJSONEncoded(http.StatusNotFound, errors.NewNotFound(schema.GroupResource{}, "testvm")),
		))
		_, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).Get("testvm", &k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).To(HaveOccurred())
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})

	It("should fetch a VirtualMachineInstance list", func() {
		vmi := v1.NewMinimalVMI("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, NewVMIList(*vmi)),
		))
		fetchedVMIList, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).List(&k8smetav1.ListOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVMIList.Items).To(HaveLen(1))
		Expect(fetchedVMIList.Items[0]).To(Equal(*vmi))
	})

	It("should create a VirtualMachineInstance", func() {
		vmi := v1.NewMinimalVMI("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusCreated, vmi),
		))
		createdVMI, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).Create(vmi)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(createdVMI).To(Equal(vmi))
	})

	It("should update a VirtualMachineInstance", func() {
		vmi := v1.NewMinimalVMI("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", vmiPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
		))
		updatedVMI, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).Update(vmi)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMI).To(Equal(vmi))
	})

	It("should delete a VirtualMachineInstance", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("DELETE", vmiPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).Delete("testvm", &k8smetav1.DeleteOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	})

	It("should allow to connect a stream to a VM", func() {
		vncPath := "/apis/subresources.kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvm/vnc"

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", vncPath),
			func(w http.ResponseWriter, r *http.Request) {
				_, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
			},
		))
		_, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).VNC("testvm")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should handle a failure connecting to the VM", func() {
		vncPath := "/apis/subresources.kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvm/vnc"

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", vncPath),
			func(w http.ResponseWriter, r *http.Request) {
				return
			},
		))
		_, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).VNC("testvm")
		Expect(err).To(HaveOccurred())
	})

	It("should exchange data with the VM", func() {
		vncPath := subVMPath + "/vnc"

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", vncPath),
			func(w http.ResponseWriter, r *http.Request) {
				c, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					panic("server upgrader failed")
				}
				defer c.Close()

				for {
					mt, message, err := c.ReadMessage()
					if err != nil {
						io.WriteString(GinkgoWriter, fmt.Sprintf("server read failed: %v\n", err))
						break
					}

					err = c.WriteMessage(mt, message)
					if err != nil {
						io.WriteString(GinkgoWriter, fmt.Sprintf("server write failed: %v\n", err))
						break
					}
				}
			},
		))

		By("establishing connection")

		vnc, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).VNC("testvm")
		Expect(err).ToNot(HaveOccurred())

		By("wiring the pipes")

		pipeInReader, pipeInWriter := io.Pipe()
		pipeOutReader, pipeOutWriter := io.Pipe()

		go func() {
			vnc.Stream(StreamOptions{
				In:  pipeInReader,
				Out: pipeOutWriter,
			})
		}()

		By("sending data around")
		msg := "hello, vnc!"
		bufIn := make([]byte, 64)
		copy(bufIn[:], msg)

		_, err = pipeInWriter.Write(bufIn)
		Expect(err).ToNot(HaveOccurred())

		By("reading back data")
		bufOut := make([]byte, 64)
		_, err = pipeOutReader.Read(bufOut)
		Expect(err).ToNot(HaveOccurred())

		By("checking the result")
		Expect(bufOut).To(Equal(bufIn))
	})

	It("should pause a VirtualMachineInstance", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", subVMPath+"/pause"),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).Pause("testvm")

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	})

	It("should unpause a VirtualMachineInstance", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", subVMPath+"/unpause"),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).Unpause("testvm")

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	})

	It("should fetch GuestOSInfo from VirtualMachineInstance via subresource", func() {
		osInfo := v1.VirtualMachineInstanceGuestAgentInfo{
			GAVersion: "4.1.1",
		}

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", subVMPath+"/guestosinfo"),
			ghttp.RespondWithJSONEncoded(http.StatusOK, osInfo),
		))
		fetchedInfo, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).GuestOsInfo("testvm")

		Expect(err).ToNot(HaveOccurred(), "should fetch info normally")
		Expect(fetchedInfo).To(Equal(osInfo), "fetched info should be the same as passed in")
	})

	It("should fetch UserList from VirtualMachineInstance via subresource", func() {
		userList := v1.VirtualMachineInstanceGuestOSUserList{
			Items: []v1.VirtualMachineInstanceGuestOSUser{
				{UserName: "testUser"},
			},
		}
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", subVMPath+"/userlist"),
			ghttp.RespondWithJSONEncoded(http.StatusOK, userList),
		))
		fetchedInfo, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).UserList("testvm")

		Expect(err).ToNot(HaveOccurred(), "should fetch info normally")
		Expect(fetchedInfo).To(Equal(userList), "fetched info should be the same as passed in")
	})

	It("should fetch FilesystemList from VirtualMachineInstance via subresource", func() {
		fileSystemList := v1.VirtualMachineInstanceFileSystemList{
			Items: []v1.VirtualMachineInstanceFileSystem{
				{
					DiskName:   "main",
					MountPoint: "/",
				},
			},
		}

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", subVMPath+"/filesystemlist"),
			ghttp.RespondWithJSONEncoded(http.StatusOK, fileSystemList),
		))
		fetchedInfo, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).FilesystemList("testvm")

		Expect(err).ToNot(HaveOccurred(), "should fetch info normally")
		Expect(fetchedInfo).To(Equal(fileSystemList), "fetched info should be the same as passed in")
	})

	AfterEach(func() {
		server.Close()
	})
})

func NewVMIList(vms ...v1.VirtualMachineInstance) *v1.VirtualMachineInstanceList {
	return &v1.VirtualMachineInstanceList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachineInstanceList"}, Items: vms}
}
