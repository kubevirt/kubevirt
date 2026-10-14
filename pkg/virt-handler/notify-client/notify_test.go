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

package notifyclient

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"google.golang.org/grpc"

	v2 "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/v2"
)

var _ = Describe("Notify client", func() {

	var (
		underlying *fakeV2Client
		client     *notifyClient
		ctx        context.Context
		cancel     context.CancelFunc
	)

	BeforeEach(func() {
		underlying = &fakeV2Client{}
		client = &notifyClient{v2Client: underlying}

		ctx, cancel = context.WithCancel(context.Background())
		DeferCleanup(cancel)
	})

	Context("DomainEvents", func() {

		It("should hand back the stream opened by the underlying client", func() {
			event := &v2.DomainEvent{EventType: "ADDED", Domain: []byte(`{"name":"testvmi"}`)}
			underlying.domainStream = &fakeDomainStream{events: []*v2.DomainEvent{event}}

			stream, err := client.DomainEvents(ctx)
			Expect(err).To(Succeed())

			received, err := stream.Recv()
			Expect(err).To(Succeed())
			Expect(received).To(BeIdenticalTo(event))
		})

		It("should pass the caller context to the underlying client", func() {
			underlying.domainStream = &fakeDomainStream{}

			_, err := client.DomainEvents(ctx)
			Expect(err).To(Succeed())
			Expect(underlying.domainCtx).To(BeIdenticalTo(ctx))
		})

		It("should open the stream with an empty request", func() {
			underlying.domainStream = &fakeDomainStream{}

			_, err := client.DomainEvents(ctx)
			Expect(err).To(Succeed())
			Expect(underlying.domainReq).To(Equal(&v2.EmptyRequest{}))
		})

		It("should return the error reported by the underlying client", func() {
			expectedErr := errors.New("domain stream could not be opened")
			underlying.domainErr = expectedErr

			stream, err := client.DomainEvents(ctx)
			Expect(err).To(MatchError(expectedErr))
			Expect(stream).To(BeNil())
		})
	})

	Context("KubernetesEvents", func() {

		It("should hand back the stream opened by the underlying client", func() {
			event := &v2.KubernetesEvent{Type: "Normal", Reason: "Started"}
			underlying.k8sStream = &fakeK8SStream{events: []*v2.KubernetesEvent{event}}

			stream, err := client.KubernetesEvents(ctx)
			Expect(err).To(Succeed())

			received, err := stream.Recv()
			Expect(err).To(Succeed())
			Expect(received).To(BeIdenticalTo(event))
		})

		It("should pass the caller context to the underlying client", func() {
			underlying.k8sStream = &fakeK8SStream{}

			_, err := client.KubernetesEvents(ctx)
			Expect(err).To(Succeed())
			Expect(underlying.k8sCtx).To(BeIdenticalTo(ctx))
		})

		It("should open the stream with an empty request", func() {
			underlying.k8sStream = &fakeK8SStream{}

			_, err := client.KubernetesEvents(ctx)
			Expect(err).To(Succeed())
			Expect(underlying.k8sReq).To(Equal(&v2.EmptyRequest{}))
		})

		It("should return the error reported by the underlying client", func() {
			expectedErr := errors.New("kubernetes stream could not be opened")
			underlying.k8sErr = expectedErr

			stream, err := client.KubernetesEvents(ctx)
			Expect(err).To(MatchError(expectedErr))
			Expect(stream).To(BeNil())
		})
	})

	Context("NewClient", func() {

		It("should stream domain events from a server on the socket", func() {
			event := &v2.DomainEvent{EventType: "ADDED", Domain: []byte(`{"name":"testvmi"}`)}
			socketPath := serveStub(&stubNotifyServer{domainEvents: []*v2.DomainEvent{event}})

			connected, err := NewClient(socketPath)
			Expect(err).To(Succeed())

			stream, err := connected.DomainEvents(ctx)
			Expect(err).To(Succeed())

			received, err := stream.Recv()
			Expect(err).To(Succeed())
			Expect(received.EventType).To(Equal("ADDED"))
			Expect(received.Domain).To(Equal([]byte(`{"name":"testvmi"}`)))

			_, err = stream.Recv()
			Expect(err).To(MatchError(io.EOF))
		})

		It("should stream kubernetes events from a server on the socket", func() {
			event := &v2.KubernetesEvent{Type: "Normal", Reason: "Started", Message: "VirtualMachineInstance started"}
			socketPath := serveStub(&stubNotifyServer{k8sEvents: []*v2.KubernetesEvent{event}})

			connected, err := NewClient(socketPath)
			Expect(err).To(Succeed())

			stream, err := connected.KubernetesEvents(ctx)
			Expect(err).To(Succeed())

			received, err := stream.Recv()
			Expect(err).To(Succeed())
			Expect(received.Type).To(Equal("Normal"))
			Expect(received.Reason).To(Equal("Started"))
			Expect(received.Message).To(Equal("VirtualMachineInstance started"))

			_, err = stream.Recv()
			Expect(err).To(MatchError(io.EOF))
		})

		It("should give up when there is nothing listening on the socket", func() {
			tmpDir, err := os.MkdirTemp("", "kubevirt-notify-client-test")
			Expect(err).To(Succeed())
			DeferCleanup(func() {
				Expect(os.RemoveAll(tmpDir)).To(Succeed())
			})

			connected, err := NewClient(filepath.Join(tmpDir, "missing.sock"))
			Expect(err).To(MatchError(context.DeadlineExceeded))
			Expect(connected).To(BeNil())
		})
	})
})

// serveStub runs service on a unix socket and returns its path.
func serveStub(service v2.NotifyServer) string {
	tmpDir, err := os.MkdirTemp("", "kubevirt-notify-client-test")
	Expect(err).To(Succeed())
	DeferCleanup(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	socketPath := filepath.Join(tmpDir, "notify.sock")
	listener, err := net.Listen("unix", socketPath)
	Expect(err).To(Succeed())

	server := grpc.NewServer()
	v2.RegisterNotifyServer(server, service)
	go func() {
		defer GinkgoRecover()
		_ = server.Serve(listener)
	}()
	DeferCleanup(server.Stop)

	return socketPath
}

// fakeV2Client stands in for the generated notify client and records how it
// was called.
type fakeV2Client struct {
	domainStream v2.Notify_DomainEventsClient
	domainErr    error
	domainCtx    context.Context
	domainReq    *v2.EmptyRequest

	k8sStream v2.Notify_KubernetesEventsClient
	k8sErr    error
	k8sCtx    context.Context
	k8sReq    *v2.EmptyRequest
}

var _ v2.NotifyClient = &fakeV2Client{}

func (f *fakeV2Client) DomainEvents(ctx context.Context, in *v2.EmptyRequest, _ ...grpc.CallOption) (v2.Notify_DomainEventsClient, error) {
	f.domainCtx = ctx
	f.domainReq = in
	if f.domainErr != nil {
		return nil, f.domainErr
	}
	return f.domainStream, nil
}

func (f *fakeV2Client) KubernetesEvents(ctx context.Context, in *v2.EmptyRequest, _ ...grpc.CallOption) (v2.Notify_KubernetesEventsClient, error) {
	f.k8sCtx = ctx
	f.k8sReq = in
	if f.k8sErr != nil {
		return nil, f.k8sErr
	}
	return f.k8sStream, nil
}

// fakeDomainStream replays a fixed list of domain events. grpc.ClientStream is
// embedded to satisfy the generated interface; only Recv is ever called.
type fakeDomainStream struct {
	grpc.ClientStream
	events []*v2.DomainEvent
}

func (s *fakeDomainStream) Recv() (*v2.DomainEvent, error) {
	if len(s.events) == 0 {
		return nil, io.EOF
	}
	event := s.events[0]
	s.events = s.events[1:]
	return event, nil
}

// fakeK8SStream is the kubernetes event counterpart of fakeDomainStream.
type fakeK8SStream struct {
	grpc.ClientStream
	events []*v2.KubernetesEvent
}

func (s *fakeK8SStream) Recv() (*v2.KubernetesEvent, error) {
	if len(s.events) == 0 {
		return nil, io.EOF
	}
	event := s.events[0]
	s.events = s.events[1:]
	return event, nil
}

// stubNotifyServer serves a fixed list of events and then ends the stream.
type stubNotifyServer struct {
	domainEvents []*v2.DomainEvent
	k8sEvents    []*v2.KubernetesEvent
}

var _ v2.NotifyServer = &stubNotifyServer{}

func (s *stubNotifyServer) DomainEvents(_ *v2.EmptyRequest, stream v2.Notify_DomainEventsServer) error {
	for _, event := range s.domainEvents {
		if err := stream.Send(event); err != nil {
			return err
		}
	}
	return nil
}

func (s *stubNotifyServer) KubernetesEvents(_ *v2.EmptyRequest, stream v2.Notify_KubernetesEventsServer) error {
	for _, event := range s.k8sEvents {
		if err := stream.Send(event); err != nil {
			return err
		}
	}
	return nil
}
