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

package notifymanager

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	kvapi "kubevirt.io/client-go/api"

	v2 "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/v2"
	"kubevirt.io/kubevirt/pkg/safepath"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("processDomainEvents", func() {
	var directChan chan watch.Event

	const domainJSON = `{"ObjectMeta":{"name":"test-domain"}}`

	BeforeEach(func() {
		directChan = make(chan watch.Event, 10)
	})

	DescribeTable("should forward recognized event types to the direct channel",
		func(eventType watch.EventType) {
			stream := &fakeDomainStream{responses: []*v2.DomainEvent{
				{EventType: string(eventType), Domain: []byte(domainJSON)},
			}}

			processDomainEvents(stream, directChan)

			Expect(directChan).To(Receive(HaveField("Type", eventType)))
		},
		Entry("added", watch.Added),
		Entry("modified", watch.Modified),
		Entry("deleted", watch.Deleted),
	)

	It("should carry the unmarshalled domain object on the event", func() {
		stream := &fakeDomainStream{responses: []*v2.DomainEvent{
			{EventType: string(watch.Added), Domain: []byte(domainJSON)},
		}}

		processDomainEvents(stream, directChan)

		var event watch.Event
		Expect(directChan).To(Receive(&event))
		domain, ok := event.Object.(*api.Domain)
		Expect(ok).To(BeTrue(), "expected event object to be *api.Domain")
		Expect(domain.ObjectMeta.Name).To(Equal("test-domain"))
	})

	It("should drop error event types without forwarding them", func() {
		stream := &fakeDomainStream{responses: []*v2.DomainEvent{
			{EventType: string(watch.Error), Domain: []byte(domainJSON)},
		}}

		processDomainEvents(stream, directChan)

		Expect(directChan).ToNot(Receive())
	})

	It("should forward multiple events in order", func() {
		stream := &fakeDomainStream{responses: []*v2.DomainEvent{
			{EventType: string(watch.Added), Domain: []byte(domainJSON)},
			{EventType: string(watch.Modified), Domain: []byte(domainJSON)},
			{EventType: string(watch.Deleted), Domain: []byte(domainJSON)},
		}}

		processDomainEvents(stream, directChan)

		Expect(directChan).To(Receive(HaveField("Type", watch.Added)))
		Expect(directChan).To(Receive(HaveField("Type", watch.Modified)))
		Expect(directChan).To(Receive(HaveField("Type", watch.Deleted)))
	})

	// TODO
	It("should still forward the event when the domain payload cannot be unmarshalled", func() {
		stream := &fakeDomainStream{responses: []*v2.DomainEvent{
			{EventType: string(watch.Added), Domain: []byte("not-json")},
		}}

		processDomainEvents(stream, directChan)

		Expect(directChan).To(Receive(HaveField("Type", watch.Added)))
	})

	It("should not forward anything when Recv fails immediately", func() {
		stream := &fakeDomainStream{err: errors.New("stream closed")}

		processDomainEvents(stream, directChan)

		Expect(directChan).ToNot(Receive())
	})
})

var _ = Describe("processKubernetesEvents", func() {
	var (
		recorder *record.FakeRecorder
		vmi      *v1.VirtualMachineInstance
	)

	BeforeEach(func() {
		recorder = record.NewFakeRecorder(10)
		vmi = kvapi.NewMinimalVMI("testvmi")
		vmi.UID = "1234"
	})

	It("should record events emitted by the stream, in order", func() {
		stream := &fakeKubernetesStream{responses: []*v2.KubernetesEvent{
			{Type: "Normal", Reason: "Created", Message: "VMI created"},
			{Type: "Warning", Reason: "Failed", Message: "something went wrong"},
		}}

		processKubernetesEvents(stream, recorder, vmi)

		Expect(recorder.Events).To(Receive(Equal("Normal Created VMI created")))
		Expect(recorder.Events).To(Receive(Equal("Warning Failed something went wrong")))
	})

	It("should not record anything when Recv fails immediately", func() {
		stream := &fakeKubernetesStream{err: errors.New("stream closed")}

		processKubernetesEvents(stream, recorder, vmi)

		Expect(recorder.Events).ToNot(Receive())
	})
})

var _ = Describe("StartDomainNotify", func() {
	var (
		detector  *isolation.MockPodIsolationDetector
		isoResult *isolation.MockIsolationResult
		safeRoot  *safepath.Path
		podRoot   string
		vmi       *v1.VirtualMachineInstance
		stopChan  chan struct{}
		mgr       *notifyV2Manager
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		detector = isolation.NewMockPodIsolationDetector(ctrl)
		isoResult = isolation.NewMockIsolationResult(ctrl)

		// podRoot stands in for the launcher pod's mount root.
		podRoot = GinkgoT().TempDir()
		var err error
		safeRoot, err = safepath.JoinAndResolveWithRelativeRoot(podRoot)
		Expect(err).To(Succeed())

		vmi = kvapi.NewMinimalVMI("testvmi")
		vmi.UID = "1234"

		stopChan = make(chan struct{})
		DeferCleanup(func() { close(stopChan) })

		mgr = NewNotifyV2Manager(nil, nil, detector)
	})

	// expectIsolation programs the detector to report an isolation result
	// rooted at the stand-in pod mount root.
	expectIsolation := func() {
		detector.EXPECT().Detect(vmi).Return(isoResult, nil)
		isoResult.EXPECT().MountRoot().Return(safeRoot, nil)
	}

	// serveNotifyStub starts a gRPC server on the socket StartDomainNotify
	// expects to find under podRoot, mimicking the launcher pod's command
	// socket. register is handed the bare server so a test can decide
	// whether (and how) to register the notify service.
	serveNotifyStub := func(register func(*grpc.Server)) {
		socketDir := filepath.Join(podRoot, cmdclient.SocketsDirectory())
		Expect(os.MkdirAll(socketDir, 0755)).To(Succeed())
		socketPath := filepath.Join(socketDir, cmdclient.StandardLauncherSocketFileName)

		listener, err := net.Listen("unix", socketPath)
		Expect(err).To(Succeed())

		server := grpc.NewServer()
		register(server)
		go func() {
			defer GinkgoRecover()
			_ = server.Serve(listener)
		}()
		DeferCleanup(server.Stop)
	}

	Context("failures", func() {
		It("should return the error reported by isolation detection", func() {
			detectErr := errors.New("no such pod")
			detector.EXPECT().Detect(vmi).Return(nil, detectErr)

			err := mgr.StartDomainNotify(stopChan, vmi)
			Expect(err).To(MatchError(detectErr))
		})

		It("should return the error reported when the mount root is unavailable", func() {
			mountErr := errors.New("mount root unavailable")
			detector.EXPECT().Detect(vmi).Return(isoResult, nil)
			isoResult.EXPECT().MountRoot().Return(nil, mountErr)

			err := mgr.StartDomainNotify(stopChan, vmi)
			Expect(err).To(MatchError(mountErr))
		})

		It("should return an error when nothing is listening on the launcher socket", func() {
			expectIsolation()

			err := mgr.StartDomainNotify(stopChan, vmi)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("the initial ping", func() {
		DescribeTable("should return notImplemented when the launcher",
			func(register func(*grpc.Server)) {
				serveNotifyStub(register)
				expectIsolation()

				err := mgr.StartDomainNotify(stopChan, vmi)
				Expect(err).To(MatchError(notImplemented))
			},
			Entry("doesn't serve the notify service at all", func(*grpc.Server) {}),
			Entry("reports Ping as unimplemented", func(server *grpc.Server) {
				v2.RegisterNotifyServer(server, &scriptedNotifyServer{
					ping: func(context.Context) error {
						return status.Error(codes.Unimplemented, "no ping here")
					},
				})
			}),
		)

		It("should return any other ping error as-is and not open the event streams", func() {
			streamOpened := make(chan struct{}, 2)
			serveNotifyStub(func(server *grpc.Server) {
				v2.RegisterNotifyServer(server, &scriptedNotifyServer{
					ping: func(context.Context) error {
						return status.Error(codes.Internal, "boom")
					},
					domainEvents: func(*v2.EmptyRequest, v2.Notify_DomainEventsServer) error {
						streamOpened <- struct{}{}
						return nil
					},
					k8sEvents: func(*v2.EmptyRequest, v2.Notify_KubernetesEventsServer) error {
						streamOpened <- struct{}{}
						return nil
					},
				})
			})
			expectIsolation()

			err := mgr.StartDomainNotify(stopChan, vmi)
			Expect(err).ToNot(MatchError(notImplemented))
			Expect(status.Code(err)).To(Equal(codes.Internal))
			Consistently(streamOpened, 100*time.Millisecond).ShouldNot(Receive())
		})

		It("should give up with an error when the launcher doesn't answer the ping in time", func() {
			serveNotifyStub(func(server *grpc.Server) {
				v2.RegisterNotifyServer(server, &scriptedNotifyServer{
					ping: func(ctx context.Context) error {
						<-ctx.Done()
						return ctx.Err()
					},
				})
			})
			expectIsolation()

			start := time.Now()
			err := mgr.StartDomainNotify(stopChan, vmi)
			Expect(status.Code(err)).To(Equal(codes.DeadlineExceeded))
			Expect(time.Since(start)).To(BeNumerically(">=", 3*time.Second))
		})

		It("should start successfully once the ping succeeds, regardless of what DomainEvents returns", func() {
			serveNotifyStub(func(server *grpc.Server) {
				v2.RegisterNotifyServer(server, &scriptedNotifyServer{
					domainEvents: func(*v2.EmptyRequest, v2.Notify_DomainEventsServer) error {
						return status.Error(codes.Unimplemented, "no domain events here")
					},
				})
			})
			expectIsolation()

			directChan := make(chan watch.Event, 10)
			mgr.directChan = directChan

			Expect(mgr.StartDomainNotify(stopChan, vmi)).To(Succeed())
			Consistently(directChan, 100*time.Millisecond).ShouldNot(Receive())
		})
	})

	Context("success", func() {
		It("should deliver domain and kubernetes events to the manager's channel and recorder", func() {
			domainJSON := []byte(`{"ObjectMeta":{"name":"test-domain"}}`)
			serveNotifyStub(func(server *grpc.Server) {
				v2.RegisterNotifyServer(server, &scriptedNotifyServer{
					domainEvents: func(_ *v2.EmptyRequest, stream v2.Notify_DomainEventsServer) error {
						return stream.Send(&v2.DomainEvent{EventType: string(watch.Added), Domain: domainJSON})
					},
					k8sEvents: func(_ *v2.EmptyRequest, stream v2.Notify_KubernetesEventsServer) error {
						return stream.Send(&v2.KubernetesEvent{Type: "Normal", Reason: "Created", Message: "VMI created"})
					},
				})
			})
			expectIsolation()

			directChan := make(chan watch.Event, 100)
			recorder := record.NewFakeRecorder(100)
			mgr.directChan = directChan
			mgr.recorder = recorder

			Expect(mgr.StartDomainNotify(stopChan, vmi)).To(Succeed())

			var event watch.Event
			Eventually(directChan, 5*time.Second).Should(Receive(&event))
			Expect(event.Type).To(Equal(watch.Added))
			domain, ok := event.Object.(*api.Domain)
			Expect(ok).To(BeTrue(), "expected event object to be *api.Domain")
			Expect(domain.ObjectMeta.Name).To(Equal("test-domain"))

			Eventually(recorder.Events, 5*time.Second).Should(Receive(Equal("Normal Created VMI created")))
		})
	})
})

// fakeDomainStream is a test double for notifyclient.DomainEventsStream. It
// hands back the configured responses in order and then returns err (io.EOF
// by default) to signal the end of the stream.
type fakeDomainStream struct {
	responses []*v2.DomainEvent
	err       error
}

func (f *fakeDomainStream) Recv() (*v2.DomainEvent, error) {
	if len(f.responses) == 0 {
		if f.err != nil {
			return nil, f.err
		}
		return nil, io.EOF
	}
	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}

// fakeKubernetesStream is a test double for notifyclient.KubernetesEventsStream.
type fakeKubernetesStream struct {
	responses []*v2.KubernetesEvent
	err       error
}

func (f *fakeKubernetesStream) Recv() (*v2.KubernetesEvent, error) {
	if len(f.responses) == 0 {
		if f.err != nil {
			return nil, f.err
		}
		return nil, io.EOF
	}
	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}

// scriptedNotifyServer is a v2.NotifyServer whose RPC handlers are supplied
// per test. A nil ping handler answers the ping successfully, a nil stream
// handler behaves like a server that closes the stream immediately.
type scriptedNotifyServer struct {
	ping         func(context.Context) error
	domainEvents func(*v2.EmptyRequest, v2.Notify_DomainEventsServer) error
	k8sEvents    func(*v2.EmptyRequest, v2.Notify_KubernetesEventsServer) error
}

var _ v2.NotifyServer = &scriptedNotifyServer{}

func (s *scriptedNotifyServer) Ping(ctx context.Context, _ *v2.EmptyRequest) (*v2.EmptyResponse, error) {
	if s.ping != nil {
		if err := s.ping(ctx); err != nil {
			return nil, err
		}
	}
	return &v2.EmptyResponse{}, nil
}

func (s *scriptedNotifyServer) DomainEvents(req *v2.EmptyRequest, stream v2.Notify_DomainEventsServer) error {
	if s.domainEvents == nil {
		return nil
	}
	return s.domainEvents(req, stream)
}

func (s *scriptedNotifyServer) KubernetesEvents(req *v2.EmptyRequest, stream v2.Notify_KubernetesEventsServer) error {
	if s.k8sEvents == nil {
		return nil
	}
	return s.k8sEvents(req, stream)
}
