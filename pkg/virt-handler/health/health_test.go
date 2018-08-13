package health

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/onsi/ginkgo/extensions/table"
	"k8s.io/apimachinery/pkg/util/clock"

	v13 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-handler/devices"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

var _ = Describe("Health", func() {
	var server *ghttp.Server
	var client kubecli.KubevirtClient
	log.Log.SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		var err error
		server = ghttp.NewServer()
		client, err = kubecli.GetKubevirtClientFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())
	})

	table.DescribeTable("should mark the nodes", func(schedulable bool, device devices.Device) {
		now := v12.Now()
		t, err := json.Marshal(now)
		Expect(err).ToNot(HaveOccurred())
		patch := fmt.Sprintf(`{"metadata": { "labels": {"kubevirt.io/schedulable": "%t"}, "annotations": {"kubevirt.io/heartbeat": %s}}}`, schedulable, string(t))

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodPatch, "/api/v1/nodes/testhost"),
			ghttp.VerifyBody([]byte(patch)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, &v1.Node{}),
		))
		stop := make(chan struct{})
		defer close(stop)
		go func() {
			GinkgoRecover()
			checker := &ReadinessChecker{
				clientset: client,
				host:      "testhost",
				plugins:   map[string]devices.Device{"test": device},
				clock:     clock.NewFakeClock(now.Time),
			}
			checker.HeartBeat(1*time.Second, 100, stop)
		}()
		time.Sleep(500 * time.Millisecond)
		Expect(server.ReceivedRequests()).To(HaveLen(1))
	},
		table.Entry("should mark the node  as non-schedulable because of a failing device check", false, &testDevice{fail: true}),
		table.Entry("should mark the node  as schedulable", true, &testDevice{fail: false}),
	)

	AfterEach(func() {
		server.Close()
	})

})

type testDevice struct {
	fail bool
}

func (*testDevice) Setup(vmi *v13.VirtualMachineInstance, hostNamespaces *isolation.IsolationResult, podNamespaces *isolation.IsolationResult) error {
	return nil
}

func (t *testDevice) Available() error {
	if t.fail {
		return fmt.Errorf("failing")
	}
	return nil
}
