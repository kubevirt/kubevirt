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

package notifyserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"google.golang.org/grpc"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"

	v2 "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/v2"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	timeout      = 5 * time.Second
	settling     = 200 * time.Millisecond
	streamBuffer = 10
)

var _ = Describe("Notify server", func() {

	var (
		watchChan chan *watch.Event
		eventChan chan *k8sv1.Event
		server    *notifyServer
		ctx       context.Context
		cancel    context.CancelFunc
	)

	BeforeEach(func() {
		watchChan = make(chan *watch.Event, streamBuffer)
		eventChan = make(chan *k8sv1.Event, streamBuffer)
		server = NewNotifyServer(watchChan, eventChan)

		ctx, cancel = context.WithCancel(context.Background())
		// unblocks any server goroutine the spec left running
		DeferCleanup(cancel)
	})

	Context("domain events", func() {

		It("should stream a domain event received on the watch channel", func() {
			domain := api.NewMinimalDomain("testvmi")
			expectedPayload, err := json.Marshal(domain)
			Expect(err).ToNot(HaveOccurred())

			stream := newFakeDomainStream(ctx)
			watchChan <- &watch.Event{Type: watch.Added, Object: domain}

			startDomainEvents(server, stream)

			var received *v2.DomainEvent
			Eventually(stream.sent, timeout).Should(Receive(&received))
			Expect(received.EventType).To(Equal(string(watch.Added)))
			Expect(received.Domain).To(Equal(expectedPayload))

			decoded := &api.Domain{}
			Expect(json.Unmarshal(received.Domain, decoded)).To(Succeed())
			Expect(decoded.ObjectMeta.Name).To(Equal("testvmi"))
		})

		It("should stream domain events in the order they are received", func() {
			eventTypes := []watch.EventType{watch.Added, watch.Modified, watch.Deleted}

			stream := newFakeDomainStream(ctx)
			for _, eventType := range eventTypes {
				watchChan <- &watch.Event{
					Type:   eventType,
					Object: api.NewMinimalDomain("testvmi"),
				}
			}

			startDomainEvents(server, stream)

			for _, expectedType := range eventTypes {
				var received *v2.DomainEvent
				Eventually(stream.sent, timeout).Should(Receive(&received))
				Expect(received.EventType).To(Equal(string(expectedType)))
			}
		})

		It("should keep serving until the stream context is done", func() {
			stream := newFakeDomainStream(ctx)
			errChan := startDomainEvents(server, stream)

			Consistently(errChan, settling).ShouldNot(Receive())

			cancel()

			Eventually(errChan, timeout).Should(Receive(BeNil()))
		})

		It("should return nil when the client closed the stream", func() {
			stream := newFakeDomainStream(ctx).failSendsWith(io.EOF)
			watchChan <- &watch.Event{Type: watch.Added, Object: api.NewMinimalDomain("testvmi")}

			errChan := startDomainEvents(server, stream)

			Eventually(errChan, timeout).Should(Receive(BeNil()))
		})

		It("should return the error when sending on the stream fails", func() {
			sendErr := errors.New("stream is broken")
			stream := newFakeDomainStream(ctx).failSendsWith(sendErr)
			watchChan <- &watch.Event{Type: watch.Added, Object: api.NewMinimalDomain("testvmi")}

			errChan := startDomainEvents(server, stream)

			var err error
			Eventually(errChan, timeout).Should(Receive(&err))
			Expect(err).To(MatchError(sendErr))
		})
	})

	Context("kubernetes events", func() {

		It("should stream a kubernetes event received on the event channel", func() {
			event := &k8sv1.Event{
				Type:    k8sv1.EventTypeNormal,
				Reason:  "Started",
				Message: "VirtualMachineInstance started",
			}

			stream := newFakeK8SStream(ctx)
			eventChan <- event

			startKubernetesEvents(server, stream)

			var received *v2.KubernetesEvent
			Eventually(stream.sent, timeout).Should(Receive(&received))
			Expect(received.Type).To(Equal(k8sv1.EventTypeNormal))
			Expect(received.Reason).To(Equal(event.Reason))
			Expect(received.Message).To(Equal(event.Message))
		})

		It("should stream kubernetes events in the order they are received", func() {
			reasons := []string{"Created", "Started", "Stopped"}

			stream := newFakeK8SStream(ctx)
			for _, reason := range reasons {
				eventChan <- &k8sv1.Event{Type: k8sv1.EventTypeNormal, Reason: reason}
			}

			startKubernetesEvents(server, stream)

			for _, expectedReason := range reasons {
				var received *v2.KubernetesEvent
				Eventually(stream.sent, timeout).Should(Receive(&received))
				Expect(received.Reason).To(Equal(expectedReason))
			}
		})

		It("should keep serving until the stream context is done", func() {
			stream := newFakeK8SStream(ctx)
			errChan := startKubernetesEvents(server, stream)

			Consistently(errChan, settling).ShouldNot(Receive())

			cancel()

			Eventually(errChan, timeout).Should(Receive(BeNil()))
		})

		It("should return nil when the client closed the stream", func() {
			stream := newFakeK8SStream(ctx).failSendsWith(io.EOF)
			eventChan <- &k8sv1.Event{Reason: "Started"}

			errChan := startKubernetesEvents(server, stream)

			Eventually(errChan, timeout).Should(Receive(BeNil()))
		})

		It("should return the error when sending on the stream fails", func() {
			sendErr := errors.New("stream is broken")
			stream := newFakeK8SStream(ctx).failSendsWith(sendErr)
			eventChan <- &k8sv1.Event{Reason: "Started"}

			errChan := startKubernetesEvents(server, stream)

			var err error
			Eventually(errChan, timeout).Should(Receive(&err))
			Expect(err).To(MatchError(sendErr))
		})
	})

	It("should only serve domain events on the domain stream", func() {
		domainStream := newFakeDomainStream(ctx)
		k8sStream := newFakeK8SStream(ctx)

		startDomainEvents(server, domainStream)
		startKubernetesEvents(server, k8sStream)

		watchChan <- &watch.Event{Type: watch.Added, Object: api.NewMinimalDomain("testvmi")}

		Eventually(domainStream.sent, timeout).Should(Receive())
		Consistently(k8sStream.sent, settling).ShouldNot(Receive())
	})
})

func startDomainEvents(server *notifyServer, stream v2.Notify_DomainEventsServer) <-chan error {
	errChan := make(chan error, 1)
	go func() {
		defer GinkgoRecover()
		errChan <- server.DomainEvents(&v2.EmptyRequest{}, stream)
	}()
	return errChan
}

func startKubernetesEvents(server *notifyServer, stream v2.Notify_KubernetesEventsServer) <-chan error {
	errChan := make(chan error, 1)
	go func() {
		defer GinkgoRecover()
		errChan <- server.KubernetesEvents(&v2.EmptyRequest{}, stream)
	}()
	return errChan
}

// fakeDomainStream records what the server sends on a domain event stream.
// grpc.ServerStream is embedded to satisfy the interface; the server only ever
// calls Send and Context, so the remaining methods are never reached.
type fakeDomainStream struct {
	grpc.ServerStream
	ctx     context.Context
	sent    chan *v2.DomainEvent
	sendErr error
}

func newFakeDomainStream(ctx context.Context) *fakeDomainStream {
	return &fakeDomainStream{
		ctx:  ctx,
		sent: make(chan *v2.DomainEvent, streamBuffer),
	}
}

// failSendsWith must be called before the stream is handed to the server, so
// that the write to sendErr happens before the serving goroutine starts.
func (s *fakeDomainStream) failSendsWith(err error) *fakeDomainStream {
	s.sendErr = err
	return s
}

func (s *fakeDomainStream) Context() context.Context {
	return s.ctx
}

func (s *fakeDomainStream) Send(event *v2.DomainEvent) error {
	if s.sendErr != nil {
		return s.sendErr
	}
	s.sent <- event
	return nil
}

// fakeK8SStream is the kubernetes event counterpart of fakeDomainStream.
type fakeK8SStream struct {
	grpc.ServerStream
	ctx     context.Context
	sent    chan *v2.KubernetesEvent
	sendErr error
}

func newFakeK8SStream(ctx context.Context) *fakeK8SStream {
	return &fakeK8SStream{
		ctx:  ctx,
		sent: make(chan *v2.KubernetesEvent, streamBuffer),
	}
}

// failSendsWith must be called before the stream is handed to the server, so
// that the write to sendErr happens before the serving goroutine starts.
func (s *fakeK8SStream) failSendsWith(err error) *fakeK8SStream {
	s.sendErr = err
	return s
}

func (s *fakeK8SStream) Context() context.Context {
	return s.ctx
}

func (s *fakeK8SStream) Send(event *v2.KubernetesEvent) error {
	if s.sendErr != nil {
		return s.sendErr
	}
	s.sent <- event
	return nil
}
