package health

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/tools/cache"

	v13 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
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

	It("should mark the nodes as schedulable", func() {
		now := v12.Now()
		t, err := json.Marshal(now)
		Expect(err).ToNot(HaveOccurred())
		patch := fmt.Sprintf(`{"metadata": { "labels": {"kubevirt.io/schedulable": "true"}, "annotations": {"kubevirt.io/heartbeat": %s}}}`, string(t))
		atomic.StoreUint64(&errorCount, 99)

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodPatch, "/api/v1/nodes/testhost"),
			ghttp.VerifyBody([]byte(patch)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, &v1.Node{}),
		))
		stop := make(chan struct{})
		defer close(stop)
		go func() {
			checker := NewReadinessChecker(client, "testhost")
			checker.Clock = clock.NewFakeClock(now.Time)
			checker.HeartBeat(1*time.Second, 100, stop)
		}()
		time.Sleep(500 * time.Millisecond)
		Expect(server.ReceivedRequests()).To(HaveLen(1))
	})

	It("should mark the nodes as unschedulable if error rate is exceeded", func() {
		now := v12.Now()
		t, err := json.Marshal(now)
		Expect(err).ToNot(HaveOccurred())
		atomic.StoreUint64(&errorCount, 101)
		patch := fmt.Sprintf(`{"metadata": { "labels": {"kubevirt.io/schedulable": "false"}, "annotations": {"kubevirt.io/heartbeat": %s}}}`, string(t))

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodPatch, "/api/v1/nodes/testhost"),
			ghttp.VerifyBody([]byte(patch)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, &v1.Node{}),
		))
		stop := make(chan struct{})
		defer close(stop)
		go func() {
			checker := NewReadinessChecker(client, "testhost")
			checker.Clock = clock.NewFakeClock(now.Time)
			checker.HeartBeat(1*time.Second, 100, stop)
		}()
		time.Sleep(500 * time.Millisecond)
		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(errorCount).To(Equal(uint64(0)))
	})

	It("should count non-user-facing errors", func() {
		lw := cache.NewListWatchFromClient(client.RestClient(), "virtualmachineinstance", v1.NamespaceAll, fields.Everything())
		informer := cache.NewSharedIndexInformer(lw, &v13.VirtualMachineInstance{}, 1*time.Second, cache.Indexers{})
		server.AllowUnhandledRequests = true
		stop := make(chan struct{})
		defer close(stop)
		go informer.Run(stop)
		Eventually(func() uint64 { return atomic.LoadUint64(&errorCount) }, 1*time.Second).Should(BeNumerically(">", 0))
	})

	AfterEach(func() {
		server.Close()
	})
})
