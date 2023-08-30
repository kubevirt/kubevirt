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
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"

	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
)

var _ = Describe("Kubevirt VirtualMachineInstance Client", func() {

	var upgrader websocket.Upgrader
	var server *ghttp.Server
	basePath := "/apis/kubevirt.io/v1/namespaces/default/virtualmachineinstances"
	vmiPath := path.Join(basePath, "testvm")
	subVMIPath := "/apis/subresources.kubevirt.io/v1/namespaces/default/virtualmachineinstances/testvm"
	proxyPath := "/proxy/path"

	BeforeEach(func() {
		server = ghttp.NewServer()
	})

	DescribeTable("should fetch a VirtualMachineInstance", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		vmi := api.NewMinimalVMI("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, vmiPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
		))
		fetchedVMI, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).Get(context.Background(), "testvm", &k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVMI).To(Equal(vmi))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should detect non existent VMIs", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, vmiPath)),
			ghttp.RespondWithJSONEncoded(http.StatusNotFound, errors.NewNotFound(schema.GroupResource{}, "testvm")),
		))
		_, err = client.VirtualMachineInstance(k8sv1.NamespaceDefault).Get(context.Background(), "testvm", &k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).To(HaveOccurred())
		Expect(errors.IsNotFound(err)).To(BeTrue())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should fetch a VirtualMachineInstance list", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		vmi := api.NewMinimalVMI("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, basePath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, NewVMIList(*vmi)),
		))
		fetchedVMIList, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).List(context.Background(), &k8smetav1.ListOptions{})
		apiVersion, kind := v1.VirtualMachineInstanceGroupVersionKind.ToAPIVersionAndKind()

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVMIList.Items).To(HaveLen(1))
		Expect(fetchedVMIList.Items[0].APIVersion).To(Equal(apiVersion))
		Expect(fetchedVMIList.Items[0].Kind).To(Equal(kind))
		Expect(fetchedVMIList.Items[0]).To(Equal(*vmi))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should create a VirtualMachineInstance", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		vmi := api.NewMinimalVMI("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", path.Join(proxyPath, basePath)),
			ghttp.RespondWithJSONEncoded(http.StatusCreated, vmi),
		))
		createdVMI, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).Create(context.Background(), vmi)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(createdVMI).To(Equal(vmi))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should update a VirtualMachineInstance", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		vmi := api.NewMinimalVMI("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", path.Join(proxyPath, vmiPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
		))
		updatedVMI, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).Update(context.Background(), vmi)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMI).To(Equal(vmi))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should delete a VirtualMachineInstance", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("DELETE", path.Join(proxyPath, vmiPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err = client.VirtualMachineInstance(k8sv1.NamespaceDefault).Delete(context.Background(), "testvm", &k8smetav1.DeleteOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should allow to connect a stream to a VM", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		vncPath := "/apis/subresources.kubevirt.io/v1/namespaces/default/virtualmachineinstances/testvm/vnc"

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, vncPath)),
			func(w http.ResponseWriter, r *http.Request) {
				_, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
			},
		))
		_, err = client.VirtualMachineInstance(k8sv1.NamespaceDefault).VNC("testvm")
		Expect(err).ToNot(HaveOccurred())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should handle a failure connecting to the VM", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		vncPath := "/apis/subresources.kubevirt.io/v1/namespaces/default/virtualmachineinstances/testvm/vnc"

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, vncPath)),
			func(w http.ResponseWriter, r *http.Request) {
				return
			},
		))
		_, err = client.VirtualMachineInstance(k8sv1.NamespaceDefault).VNC("testvm")
		Expect(err).To(HaveOccurred())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should exchange data with the VM", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		vncPath := path.Join(subVMIPath, "vnc")

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, vncPath)),
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
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should pause a VirtualMachineInstance", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", path.Join(proxyPath, subVMIPath, "pause")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err = client.VirtualMachineInstance(k8sv1.NamespaceDefault).Pause(context.Background(), "testvm", &v1.PauseOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should unpause a VirtualMachineInstance", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", path.Join(proxyPath, subVMIPath, "unpause")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err = client.VirtualMachineInstance(k8sv1.NamespaceDefault).Unpause(context.Background(), "testvm", &v1.UnpauseOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should freeze a VirtualMachineInstance", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", path.Join(proxyPath, subVMIPath, "freeze")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err = client.VirtualMachineInstance(k8sv1.NamespaceDefault).Freeze(context.Background(), "testvm", 0*time.Second)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should unfreeze a VirtualMachineInstance", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", path.Join(proxyPath, subVMIPath, "unfreeze")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err = client.VirtualMachineInstance(k8sv1.NamespaceDefault).Unfreeze(context.Background(), "testvm")

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should soft reboot a VirtualMachineInstance", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", path.Join(proxyPath, subVMIPath, "softreboot")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err = client.VirtualMachineInstance(k8sv1.NamespaceDefault).SoftReboot(context.Background(), "testvm")

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should fetch GuestOSInfo from VirtualMachineInstance via subresource", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		osInfo := v1.VirtualMachineInstanceGuestAgentInfo{
			GAVersion: "4.1.1",
		}

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, subVMIPath, "guestosinfo")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, osInfo),
		))
		fetchedInfo, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).GuestOsInfo(context.Background(), "testvm")

		Expect(err).ToNot(HaveOccurred(), "should fetch info normally")
		Expect(fetchedInfo).To(Equal(osInfo), "fetched info should be the same as passed in")
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should fetch UserList from VirtualMachineInstance via subresource", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		userList := v1.VirtualMachineInstanceGuestOSUserList{
			Items: []v1.VirtualMachineInstanceGuestOSUser{
				{UserName: "testUser"},
			},
		}
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, subVMIPath, "userlist")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, userList),
		))
		fetchedInfo, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).UserList(context.Background(), "testvm")

		Expect(err).ToNot(HaveOccurred(), "should fetch info normally")
		Expect(fetchedInfo).To(Equal(userList), "fetched info should be the same as passed in")
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should fetch FilesystemList from VirtualMachineInstance via subresource", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		fileSystemList := v1.VirtualMachineInstanceFileSystemList{
			Items: []v1.VirtualMachineInstanceFileSystem{
				{
					DiskName:   "main",
					MountPoint: "/",
				},
			},
		}

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, subVMIPath, "filesystemlist")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, fileSystemList),
		))
		fetchedInfo, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).FilesystemList(context.Background(), "testvm")

		Expect(err).ToNot(HaveOccurred(), "should fetch info normally")
		Expect(fetchedInfo).To(Equal(fileSystemList), "fetched info should be the same as passed in")
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	It("should fetch SEV platform info via subresource", func() {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		sevPlatformIfo := v1.SEVPlatformInfo{
			PDH:       "AAABBB",
			CertChain: "CCCDDD",
		}

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(subVMIPath, "sev/fetchcertchain")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, sevPlatformIfo),
		))
		fetchedInfo, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).SEVFetchCertChain("testvm")

		Expect(err).ToNot(HaveOccurred(), "should fetch info normally")
		Expect(fetchedInfo).To(Equal(sevPlatformIfo), "fetched info should be the same as passed in")
	})

	It("should query SEV launch measurement info via subresource", func() {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		sevMeasurementInfo := v1.SEVMeasurementInfo{
			Measurement: "AAABBB",
			APIMajor:    1,
			APIMinor:    2,
			BuildID:     0xFFEE,
			Policy:      0xFF,
		}

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(subVMIPath, "sev/querylaunchmeasurement")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, sevMeasurementInfo),
		))
		fetchedInfo, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).SEVQueryLaunchMeasurement("testvm")

		Expect(err).ToNot(HaveOccurred(), "should fetch info normally")
		Expect(fetchedInfo).To(Equal(sevMeasurementInfo), "fetched info should be the same as passed in")
	})

	It("should setup SEV session for a VirtualMachineInstance", func() {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", path.Join(subVMIPath, "sev/setupsession")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err = client.VirtualMachineInstance(k8sv1.NamespaceDefault).SEVSetupSession("testvm", &v1.SEVSessionOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	})

	It("should inject SEV launch secret into a VirtualMachineInstance", func() {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", path.Join(subVMIPath, "sev/injectlaunchsecret")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err = client.VirtualMachineInstance(k8sv1.NamespaceDefault).SEVInjectLaunchSecret("testvm", &v1.SEVSecretOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})
})

func NewVMIList(vms ...v1.VirtualMachineInstance) *v1.VirtualMachineInstanceList {
	return &v1.VirtualMachineInstanceList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachineInstanceList"}, Items: vms}
}
