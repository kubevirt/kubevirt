package rest

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
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
			},
			Spec: v1.VirtualMachineInstanceSpec{},
			Status: v1.VirtualMachineInstanceStatus{
				Interfaces: interfaces,
			},
		}
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
})
