package rest

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
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
		errMsg := "foo bar from the App handler!"
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			response := restful.NewResponse(rw)
			response.WriteHeader(http.StatusBadRequest)
			_, err := response.Write([]byte(errMsg))
			response.Flush()
			Expect(err).ToNot(HaveOccurred())
		}))
		defer server.Close()

		kv := getKubeVirt()
		config, _, _ := testutils.NewFakeClusterConfigUsingKV(kv)

		addr := server.URL[strings.LastIndex(server.URL, "/")+1:]
		fullURL := fmt.Sprintf("ws://%s%s", addr, request.Request.URL.RequestURI())
		port, err := strconv.ParseInt(server.URL[strings.LastIndex(server.URL, ":")+1:], 10, 32)
		Expect(err).NotTo(HaveOccurred())

		ctrl := gomock.NewController(GinkgoT())
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		k8sfakeClient := fake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(k8sfakeClient.CoreV1()).AnyTimes()

		vmi := libvmi.New()
		vmi.Status.Phase = v1.Running
		app := NewSubresourceAPIApp(virtClient, int(port), nil, config)
		dialer := app.virtHandlerDialer(func(_ *v1.VirtualMachineInstance, _ kubecli.VirtHandlerConn) (string, error) {
			return fullURL, nil
		})

		Eventually(func(g Gomega) {
			conn, statusErr := dialer.DialUnderlying(vmi)
			g.Expect(statusErr).To(MatchError(ContainSubstring(errMsg)))
			g.Expect(conn).To(BeNil())
		}).Should(Succeed())
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

func getKubeVirt() *v1.KubeVirt {
	return &v1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt",
			Namespace: "kubevirt",
		},
		Spec: v1.KubeVirtSpec{
			Configuration: v1.KubeVirtConfiguration{
				DeveloperConfiguration: &v1.DeveloperConfiguration{},
			},
		},
		Status: v1.KubeVirtStatus{
			Phase: v1.KubeVirtPhaseDeployed,
		},
	}
}
