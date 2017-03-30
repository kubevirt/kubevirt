package rest

import (
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	fake2 "k8s.io/client-go/kubernetes/fake"
	k8scorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	k8sv1 "k8s.io/client-go/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
)

var _ = Describe("Console", func() {

	var ctrl *gomock.Controller
	var virtClient *kubecli.MockKubevirtClient
	var vmInterface *kubecli.MockVMInterface
	var k8sClient k8scorev1.CoreV1Interface
	var vm *v1.VM
	var node *k8sv1.Node
	var server *httptest.Server
	var dial func(vm string, console string) *websocket.Conn
	var get func(vm string) (*http.Response, error)

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVMInterface(ctrl)
		virtClient.EXPECT().VM(k8sv1.NamespaceDefault).Return(vmInterface)

		vm = v1.NewMinimalVM("testvm")
		vm.Status.Phase = v1.Running
		vm.Status.NodeName = "testnode"

		node = &k8sv1.Node{
			ObjectMeta: k8sv1.ObjectMeta{
				Name: "testnode",
			},
		}
		k8sClient = fake2.NewSimpleClientset(node).CoreV1()

		ws := new(restful.WebService)
		handler := http.Handler(restful.NewContainer().Add(ws))

		// Endpoint to test
		consoleResource := NewConsoleResource(virtClient, k8sClient)
		ws.Route(ws.GET("/virt-api/{name}").To(consoleResource.Console))

		// Mock out virt-handler. Mirror the first message and exit.
		ws.Route(ws.GET("/api/v1/console/{name}").To(func(request *restful.Request, response *restful.Response) {
			defer GinkgoRecover()
			ws, err := upgrader.Upgrade(response.ResponseWriter, request.Request, nil)
			Expect(err).ToNot(HaveOccurred())
			defer ws.Close()
			t, data, err := ws.ReadMessage()
			Expect(err).ToNot(HaveOccurred())
			err = ws.WriteMessage(t, data)
			Expect(err).ToNot(HaveOccurred())
			response.WriteHeader(http.StatusOK)
		}))

		server = httptest.NewServer(handler)

		wsUrl, err := url.Parse(server.URL)
		serverUrl, err := url.ParseRequestURI(server.URL)
		Expect(err).ToNot(HaveOccurred())
		// Use the test server url as virt-handler destination
		node.Status = k8sv1.NodeStatus{
			Addresses: []k8sv1.NodeAddress{
				{
					Type:    k8sv1.NodeInternalIP,
					Address: strings.Split(serverUrl.Host, ":")[0],
				},
			},
		}
		k8sClient.Nodes().Update(node)
		consoleResource.VirtHandlerPort = strings.Split(serverUrl.Host, ":")[1]

		dial = func(vm string, console string) *websocket.Conn {
			wsUrl.Scheme = "ws"
			wsUrl.Path = "/virt-api/" + vm
			wsUrl.RawQuery = "console=" + console
			c, _, err := websocket.DefaultDialer.Dial(wsUrl.String(), nil)
			Expect(err).ToNot(HaveOccurred())
			return c
		}

		get = func(vm string) (*http.Response, error) {
			wsUrl.Scheme = "http"
			wsUrl.Path = "/virt-api/" + vm
			return http.DefaultClient.Get(wsUrl.String())
		}
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should proxy message through virt-api", func() {

		vmInterface.EXPECT().Get("testvm", gomock.Any()).Return(vm, true, nil)

		ws := dial("testvm", "console0")
		defer ws.Close()
		ws.WriteMessage(websocket.TextMessage, []byte("hello echo!"))
		t, data, err := ws.ReadMessage()
		Expect(t).To(Equal(websocket.TextMessage))
		Expect(err).ToNot(HaveOccurred())
		Expect(string(data)).To(Equal("hello echo!"))
	})

	It("Should return 404 if the VM does not exist", func() {
		vmInterface.EXPECT().Get("testvm", gomock.Any()).Return(vm, false, nil)
		response, err := get("testvm")
		Expect(err).ToNot(HaveOccurred())
		Expect(response.StatusCode).To(Equal(http.StatusNotFound))
	})

	It("Should return 500 if looking up the VM failed", func() {
		vmInterface.EXPECT().Get("testvm", gomock.Any()).Return(vm, false, fmt.Errorf("something is weird"))
		response, err := get("testvm")
		Expect(err).ToNot(HaveOccurred())
		Expect(response.StatusCode).To(Equal(http.StatusInternalServerError))
		Expect(body(response)).To(ContainSubstring("something is weird"))
	})

	It("Should return 400 if the VM is not running", func() {
		vmInterface.EXPECT().Get("testvm", gomock.Any()).Return(vm, true, nil)
		vm.Status.Phase = v1.Succeeded
		response, err := get("testvm")
		Expect(err).ToNot(HaveOccurred())
		Expect(response.StatusCode).To(Equal(http.StatusBadRequest))
	})

	It("Should return 500 if we can't look up the node", func() {
		vmInterface.EXPECT().Get("testvm", gomock.Any()).Return(vm, true, nil)
		vm.Status.NodeName = "nonexistentnode"
		response, err := get("testvm")
		Expect(err).ToNot(HaveOccurred())
		Expect(response.StatusCode).To(Equal(http.StatusInternalServerError))
		Expect(body(response)).To(ContainSubstring("Node \"nonexistentnode\" not found"))
	})

	It("Should return 500 if we can't find an internal ip to connect to", func() {
		vmInterface.EXPECT().Get("testvm", gomock.Any()).Return(vm, true, nil)
		node.Status.Addresses = []k8sv1.NodeAddress{}
		k8sClient.Nodes().Update(node)
		response, err := get("testvm")
		Expect(err).ToNot(HaveOccurred())
		Expect(response.StatusCode).To(Equal(http.StatusInternalServerError))
		Expect(body(response)).To(ContainSubstring("Could not find a connection IP"))
	})

	AfterEach(func() {
		ctrl.Finish()
	})
})

func body(request *http.Response) string {
	b, err := ioutil.ReadAll(request.Body)
	Expect(err).ToNot(HaveOccurred())
	return string(b)
}
