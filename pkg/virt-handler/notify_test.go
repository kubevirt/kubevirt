package virthandler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	notifyserver "kubevirt.io/kubevirt/pkg/virt-handler/notify-server"
	"kubevirt.io/kubevirt/pkg/virt-handler/notify-server/pipe"
	notifyclient "kubevirt.io/kubevirt/pkg/virt-launcher/notify-client"
)

var _ = Describe("DomainNotifyServerRestarts", func() {
	Context("should establish a notify server pipe", func() {
		var shareDir string
		var serverStopChan chan struct{}
		var serverIsStoppedChan chan struct{}
		var recorder *record.FakeRecorder
		var vmiStore cache.Store

		BeforeEach(func() {
			var err error
			serverStopChan = make(chan struct{})
			serverIsStoppedChan = make(chan struct{})
			shareDir, err = os.MkdirTemp("", "kubevirt-share")
			Expect(err).ToNot(HaveOccurred())

			recorder = record.NewFakeRecorder(10)
			recorder.IncludeObject = true
			vmiInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
			vmiStore = vmiInformer.GetStore()

			go func(serverIsStoppedChan chan struct{}) {
				notifyserver.RunServer(shareDir, serverStopChan, make(chan watch.Event, 100), recorder, vmiStore)
				close(serverIsStoppedChan)
			}(serverIsStoppedChan)

			time.Sleep(3)
		})

		AfterEach(func() {
			close(serverStopChan)
			<-serverIsStoppedChan
			os.RemoveAll(shareDir)
		})

		It("should get notify events", func() {
			vmi := api.NewMinimalVMI("fake-vmi")
			vmi.UID = "4321"
			vmiStore.Add(vmi)

			eventType := "Normal"
			eventReason := "fooReason"
			eventMessage := "barMessage"

			pipeDir := filepath.Join(shareDir, "client_path")
			err := os.MkdirAll(pipeDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fdChan, err := pipe.InjectNotify(ctx, log.Log, fakeIsolationResult(shareDir), "client_path", false)
			Expect(err).ToNot(HaveOccurred())

			go pipe.Proxy(ctx, log.Log, fdChan, shareDir, pipe.ConnectToNotify(shareDir))
			time.Sleep(1)

			client := notifyclient.NewNotifier(pipeDir)
			defer client.Close()

			err = client.SendK8sEvent(vmi, eventType, eventReason, eventMessage)
			Expect(err).ToNot(HaveOccurred())

			timedOut := false
			timeout := time.After(4 * time.Second)
			select {
			case <-timeout:
				timedOut = true
			case event := <-recorder.Events:
				Expect(event).To(Equal(fmt.Sprintf("%s %s %s involvedObject{kind=VirtualMachineInstance,apiVersion=kubevirt.io/v1}", eventType, eventReason, eventMessage)))
			}

			Expect(timedOut).To(BeFalse(), "should not time out")
		})

		It("should eventually get notify events once pipe is online", func() {
			vmi := api.NewMinimalVMI("fake-vmi")
			vmi.UID = "4321"
			vmiStore.Add(vmi)

			eventType := "Normal"
			eventReason := "fooReason"
			eventMessage := "barMessage"

			pipeDir := filepath.Join(shareDir, "client_path")
			err := os.MkdirAll(pipeDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			// Client should fail when pipe is offline
			client := notifyclient.NewNotifier(pipeDir)
			defer client.Close()

			client.SetCustomTimeouts(1*time.Second, 1*time.Second, 3*time.Second)

			err = client.SendK8sEvent(vmi, eventType, eventReason, eventMessage)
			Expect(err).To(HaveOccurred())

			// Client should automatically come online when pipe is established
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fdChan, err := pipe.InjectNotify(ctx, log.Log, fakeIsolationResult(shareDir), "client_path", false)
			Expect(err).ToNot(HaveOccurred())

			go pipe.Proxy(ctx, log.Log, fdChan, shareDir, pipe.ConnectToNotify(shareDir))
			time.Sleep(1)

			// Expect the client to reconnect and succeed despite initial failure
			err = client.SendK8sEvent(vmi, eventType, eventReason, eventMessage)
			Expect(err).ToNot(HaveOccurred())

		})

		It("should be resilient to notify server restarts", func() {
			vmi := api.NewMinimalVMI("fake-vmi")
			vmi.UID = "4321"
			vmiStore.Add(vmi)

			eventType := "Normal"
			eventReason := "fooReason"
			eventMessage := "barMessage"

			pipeDir := filepath.Join(shareDir, "client_path")
			err := os.MkdirAll(pipeDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fdChan, err := pipe.InjectNotify(ctx, log.Log, fakeIsolationResult(shareDir), "client_path", false)
			Expect(err).ToNot(HaveOccurred())

			go pipe.Proxy(ctx, log.Log, fdChan, shareDir, pipe.ConnectToNotify(shareDir))
			time.Sleep(1)

			client := notifyclient.NewNotifier(pipeDir)
			defer client.Close()

			for i := 1; i < 5; i++ {
				// close and wait for server to stop
				close(serverStopChan)
				<-serverIsStoppedChan

				client.SetCustomTimeouts(1*time.Second, 1*time.Second, 1*time.Second)
				// Expect a client error to occur here because the server is down
				err = client.SendK8sEvent(vmi, eventType, eventReason, eventMessage)
				Expect(err).To(HaveOccurred())

				// Restart the server now that it is down.
				serverStopChan = make(chan struct{})
				serverIsStoppedChan = make(chan struct{})
				go func() {
					notifyserver.RunServer(shareDir, serverStopChan, make(chan watch.Event), recorder, vmiStore)
					close(serverIsStoppedChan)
				}()

				// Expect the client to reconnect and succeed despite server restarts
				client.SetCustomTimeouts(1*time.Second, 1*time.Second, 3*time.Second)
				err = client.SendK8sEvent(vmi, eventType, eventReason, eventMessage)
				Expect(err).ToNot(HaveOccurred())

				timedOut := false
				timeout := time.After(4 * time.Second)
				select {
				case <-timeout:
					timedOut = true
				case event := <-recorder.Events:
					Expect(event).To(Equal(fmt.Sprintf("%s %s %s involvedObject{kind=VirtualMachineInstance,apiVersion=kubevirt.io/v1}", eventType, eventReason, eventMessage)))
				}
				Expect(timedOut).To(BeFalse(), "should not time out")
			}
		})
	})
})

func fakeIsolationResult(shareDir string) isolation.IsolationResult {
	isoRes := isolation.NewMockIsolationResult(gomock.NewController(GinkgoT()))
	isoRes.EXPECT().MountRoot().DoAndReturn(func() (*safepath.Path, error) { return safepath.NewPathNoFollow(shareDir) })
	return isoRes
}
