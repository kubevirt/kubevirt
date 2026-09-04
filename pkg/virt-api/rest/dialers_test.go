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

package rest

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("NetDialer", func() {
	const (
		vmName      = "test-vm"
		vmNamespace = "test-namespace"
	)

	var (
		request *restful.Request
	)

	BeforeEach(func() {
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/apis/subresources.kubevirt.io/v1alpha3/namespaces/%s/virtualmachineinstances/%s/ssh/22", vmNamespace, vmName), nil)
		request = restful.NewRequest(httpReq)
	})

	makeVMIWithInterfaceStatus := func(interfaces []v1.VirtualMachineInstanceNetworkInterface) *v1.VirtualMachineInstance {
		return &v1.VirtualMachineInstance{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      vmName,
				Namespace: vmNamespace,
				UID:       "1234",
			},
			Spec: v1.VirtualMachineInstanceSpec{},
			Status: v1.VirtualMachineInstanceStatus{
				Interfaces: interfaces,
			},
		}
	}

	launcherPodForVMI := func(vmi *v1.VirtualMachineInstance, node, podIP string) *k8sv1.Pod {
		return &k8sv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:            vmi.Name + "-launcher",
				Namespace:       vmi.Namespace,
				Labels:          map[string]string{v1.AppLabel: "virt-launcher"},
				OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(vmi, v1.VirtualMachineInstanceGroupVersionKind)},
			},
			Spec:   k8sv1.PodSpec{NodeName: node},
			Status: k8sv1.PodStatus{PodIP: podIP},
		}
	}

	newLauncherPodIndexer := func(pods ...*k8sv1.Pod) cache.Indexer {
		indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		for _, pod := range pods {
			Expect(indexer.Add(pod)).To(Succeed())
		}
		return indexer
	}

	It("Should fail if vmi has no network interfaces", func() {
		dialer := netDial{
			request: request,
		}
		_, statusErr := dialer.DialUnderlying(makeVMIWithInterfaceStatus(nil))
		Expect(statusErr.Status().Message).To(Equal("no network interfaces are present"))
	})

	It("Should fail if request has no port", func() {
		request.PathParameters()["port"] = ""
		dialer := netDial{
			request: request,
		}
		_, statusErr := dialer.DialUnderlying(makeVMIWithInterfaceStatus([]v1.VirtualMachineInstanceNetworkInterface{
			{
				IP: "192.168.0.1",
			},
		}))
		Expect(statusErr.Status().Message).To(Equal("port must not be empty"))
	})

	It("Should forward error from Request's Body", func() {
		const errMsg = "foo bar from the App handler!"
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			response := restful.NewResponse(rw)
			response.WriteHeader(http.StatusBadRequest)
			nbytes, err := response.Write([]byte(errMsg))
			Expect(nbytes).To(Equal(len(errMsg)))
			Expect(err).ToNot(HaveOccurred())
			response.Flush()
		}))
		defer server.Close()

		config, _, _ := testutils.NewFakeClusterConfigUsingKV(&v1.KubeVirt{})

		u, err := url.Parse(server.URL)
		Expect(err).NotTo(HaveOccurred())

		fullURL := "ws://" + u.Host + request.Request.URL.RequestURI()
		port, err := strconv.ParseInt(u.Port(), 10, 32)
		Expect(err).NotTo(HaveOccurred())

		ctrl := gomock.NewController(GinkgoT())
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		k8sfakeClient := fake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(k8sfakeClient.CoreV1()).AnyTimes()

		runningStatus := libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithPhase(v1.Running)))
		vmi := libvmi.New(runningStatus)
		app := NewSubresourceAPIApp(virtClient, k8sfakeClient, nil, int(port), nil, config)
		dialer := app.virtHandlerDialer(func(_ *v1.VirtualMachineInstance, _ kubecli.VirtHandlerConn) (string, error) {
			return fullURL, nil
		})

		conn, statusErr := dialer.DialUnderlying(vmi)
		Expect(statusErr).To(MatchError(ContainSubstring(errMsg)))
		Expect(conn).To(BeNil())
	})

	DescribeTable("Should dial vmi", func(ipAddr string) {
		ln, err := net.Listen("tcp", fmt.Sprintf("%s:0", ipAddr))
		Expect(err).NotTo(HaveOccurred())
		defer ln.Close()
		tcpAddr := ln.Addr().(*net.TCPAddr)

		request.PathParameters()["port"] = strconv.FormatInt(int64(tcpAddr.Port), 10)
		dialer := netDial{
			request: request,
		}
		conn, statusErr := dialer.DialUnderlying(makeVMIWithInterfaceStatus([]v1.VirtualMachineInstanceNetworkInterface{
			{
				IP: tcpAddr.IP.String(),
			},
		}))
		Expect(statusErr).NotTo(HaveOccurred())
		Expect(conn).NotTo(BeNil())
	},
		Entry("with ipv4 ip address", "127.0.0.1"),
		Entry("with ipv6 ip address", "[::1]"),
	)

	Context("with launcher pod IP resolution", func() {
		const (
			nodeName    = "node1"
			podIP       = "10.244.0.10"
			interfaceIP = "192.0.2.1"
		)

		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = makeVMIWithInterfaceStatus([]v1.VirtualMachineInstanceNetworkInterface{{IP: interfaceIP}})
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Status.NodeName = nodeName
		})

		DescribeTable("Should resolve the target IP", func(pods func(vmi *v1.VirtualMachineInstance) []*k8sv1.Pod, expectedIP string) {
			dialer := netDial{request: request, launcherPodIndexer: newLauncherPodIndexer(pods(vmi)...)}
			Expect(dialer.resolveTargetIP(vmi)).To(Equal(expectedIP))
		},
			Entry("preferring the launcher pod IP over the interface IP",
				func(vmi *v1.VirtualMachineInstance) []*k8sv1.Pod {
					return []*k8sv1.Pod{launcherPodForVMI(vmi, nodeName, podIP)}
				}, podIP),
			Entry("falling back to the interface IP when there is no launcher pod",
				func(_ *v1.VirtualMachineInstance) []*k8sv1.Pod {
					return nil
				}, interfaceIP),
			Entry("falling back to the interface IP when the launcher pod has no IP yet",
				func(vmi *v1.VirtualMachineInstance) []*k8sv1.Pod {
					return []*k8sv1.Pod{launcherPodForVMI(vmi, nodeName, "")}
				}, interfaceIP),
			Entry("ignoring a launcher pod scheduled to a different node",
				func(vmi *v1.VirtualMachineInstance) []*k8sv1.Pod {
					return []*k8sv1.Pod{launcherPodForVMI(vmi, "node2", podIP)}
				}, interfaceIP),
			Entry("ignoring a pod not controlled by the VMI",
				func(vmi *v1.VirtualMachineInstance) []*k8sv1.Pod {
					pod := launcherPodForVMI(vmi, nodeName, podIP)
					pod.OwnerReferences[0].UID = "other-vmi"
					return []*k8sv1.Pod{pod}
				}, interfaceIP),
		)

		It("Should not use the launcher pod IP when the VMI is not connected to the pod network", func() {
			vmi.Spec.Networks = []v1.Network{{Name: "secondary", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "secondary"}}}}
			dialer := netDial{request: request, launcherPodIndexer: newLauncherPodIndexer(launcherPodForVMI(vmi, nodeName, podIP))}
			Expect(dialer.resolveTargetIP(vmi)).To(Equal(interfaceIP))
		})

		It("Should use the launcher pod IP when no interface IP has been reported yet", func() {
			vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{{IP: ""}}
			dialer := netDial{request: request, launcherPodIndexer: newLauncherPodIndexer(launcherPodForVMI(vmi, nodeName, podIP))}
			Expect(dialer.resolveTargetIP(vmi)).To(Equal(podIP))
		})

		It("Should use the launcher pod IP when the VMI reports no interfaces", func() {
			vmi.Status.Interfaces = nil
			dialer := netDial{request: request, launcherPodIndexer: newLauncherPodIndexer(launcherPodForVMI(vmi, nodeName, podIP))}
			Expect(dialer.resolveTargetIP(vmi)).To(Equal(podIP))
		})

		It("Should fail when neither a launcher pod nor an interface is present", func() {
			vmi.Status.Interfaces = nil
			dialer := netDial{request: request, launcherPodIndexer: newLauncherPodIndexer()}
			_, err := dialer.resolveTargetIP(vmi)
			Expect(err).To(MatchError("no network interfaces are present"))
		})

		It("Should fall back to the interface IP when the pod lookup fails", func() {
			// An indexer without the namespace index makes the lookup fail.
			indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			Expect(indexer.Add(launcherPodForVMI(vmi, nodeName, podIP))).To(Succeed())
			dialer := netDial{request: request, launcherPodIndexer: indexer}
			Expect(dialer.resolveTargetIP(vmi)).To(Equal(interfaceIP))
		})

		It("Should fall back to the interface IP when no launcher pod indexer is available", func() {
			dialer := netDial{request: request}
			Expect(dialer.resolveTargetIP(vmi)).To(Equal(interfaceIP))
		})

		DescribeTable("Should dial the launcher pod IP", func(ipAddr, unreachableIP string) {
			ln, err := net.Listen("tcp", fmt.Sprintf("%s:0", ipAddr))
			Expect(err).NotTo(HaveOccurred())
			defer ln.Close()
			tcpAddr := ln.Addr().(*net.TCPAddr)
			request.PathParameters()["port"] = strconv.FormatInt(int64(tcpAddr.Port), 10)

			// Nothing listens on the interface IP; the listener is only reachable via the pod IP.
			vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{{IP: unreachableIP}}
			pod := launcherPodForVMI(vmi, nodeName, tcpAddr.IP.String())
			dialer := netDial{request: request, launcherPodIndexer: newLauncherPodIndexer(pod)}

			conn, statusErr := dialer.DialUnderlying(vmi)
			Expect(statusErr).NotTo(HaveOccurred())
			Expect(conn).NotTo(BeNil())
			defer conn.Close()
			Expect(conn.RemoteAddr().String()).To(Equal(ln.Addr().String()))
		},
			Entry("with ipv4 ip address", "127.0.0.1", "::1"),
			Entry("with ipv6 ip address", "[::1]", "127.0.0.1"),
		)
	})
})
