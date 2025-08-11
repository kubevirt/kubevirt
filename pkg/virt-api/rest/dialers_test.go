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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
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
		app := NewSubresourceAPIApp(virtClient, int(port), nil, config)
		dialer := app.virtHandlerDialer(func(_ *v1.VirtualMachineInstance, _ kubecli.VirtHandlerConn) (string, error) {
			return fullURL, nil
		})

		conn, statusErr := dialer.DialUnderlying(vmi)
		Expect(error(statusErr)).To(MatchError(ContainSubstring(errMsg)))
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
})
