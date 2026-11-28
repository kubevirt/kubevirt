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

package eventsclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"

	api2 "kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Buffer client", func() {
	Context("domain events", func() {
		It("should not deliver the event on the k8s event channel", func() {
			bufferClient := NewBufferClient()
			result := callAsync(func() error { return bufferClient.HandleDomainEvent(newDomainEvent()) })
			Consistently(bufferClient.K8SEventChan(), 100*time.Millisecond).ShouldNot(Receive())

			receiveInvoke(bufferClient.DomainEventChan()).ErrorCh <- nil
			Eventually(result).Should(Receive(BeNil()))
		})
	})

	Context("k8s events", func() {
		It("should not deliver the event on the domain event channel", func() {
			bufferClient := NewBufferClient()
			result := callAsync(func() error { return bufferClient.HandleK8SEvent(newK8sEvent()) })
			Consistently(bufferClient.DomainEventChan(), 100*time.Millisecond).ShouldNot(Receive())

			receiveInvoke(bufferClient.K8SEventChan()).ErrorCh <- nil
			Eventually(result).Should(Receive(BeNil()))
		})
	})

	Context("conforms with grpc client", func() {
		DescribeTable("should hand the event to a consumer and wait for its reply", func(send sendFunc, receive receiveFunc) {
			bufferClient := NewBufferClient()
			event, result := send(bufferClient)
			invoke := receive(bufferClient)
			Expect(invoke.Event).To(BeIdenticalTo(event))
			Consistently(result, 100*time.Millisecond).ShouldNot(Receive(), "should wait for the consumer's reply")

			invoke.ErrorCh <- nil
			Eventually(result).Should(Receive(BeNil()))
		},
			Entry("domain event", sendDomainEvent, receiveDomainEvent),
			Entry("k8s event", sendK8sEvent, receiveK8sEvent),
		)

		DescribeTable("should wait for a consumer to take the event", func(send sendFunc, receive receiveFunc) {
			bufferClient := NewBufferClient()
			_, result := send(bufferClient)
			Consistently(result, 300*time.Millisecond).ShouldNot(Receive())

			receive(bufferClient).ErrorCh <- nil
			Eventually(result).Should(Receive(BeNil()))
		},
			Entry("domain event", sendDomainEvent, receiveDomainEvent),
			Entry("k8s event", sendK8sEvent, receiveK8sEvent),
		)

		DescribeTable("should give up when no consumer takes the event within the total timeout", func(send sendFunc) {
			const totalTimeout = 300 * time.Millisecond
			bufferClient := newBufferClientWithCustomTimeouts(10*time.Millisecond, 100*time.Millisecond, totalTimeout)

			start := time.Now()
			_, result := send(bufferClient)

			var err error
			Eventually(result, 5*totalTimeout).Should(Receive(&err))
			Expect(err).To(MatchError(context.DeadlineExceeded))
			Expect(time.Since(start)).To(BeNumerically(">=", totalTimeout))
			expectNoEventOffered(bufferClient, "should stop offering the event")
		},
			Entry("domain event", sendDomainEvent),
			Entry("k8s event", sendK8sEvent),
		)

		DescribeTable("should offer the event again when the consumer does not reply within the send timeout", func(send sendFunc, receive receiveFunc) {
			bufferClient := newBufferClientWithCustomTimeouts(10*time.Millisecond, 300*time.Millisecond, defaultTotalTimeout)
			event, result := send(bufferClient)

			first := receive(bufferClient)
			second := receive(bufferClient)
			Expect(second.Event).To(BeIdenticalTo(event))

			Expect(first.ErrorCh).To(BeSent(errors.New("late reply")), "a late reply should not block the consumer")
			Consistently(result, 50*time.Millisecond).ShouldNot(Receive(), "a late reply should be ignored")

			second.ErrorCh <- nil
			Eventually(result).Should(Receive(BeNil()))
		},
			Entry("domain event", sendDomainEvent, receiveDomainEvent),
			Entry("k8s event", sendK8sEvent, receiveK8sEvent),
		)

		DescribeTable("should offer the event again after a transient error", func(send sendFunc, receive receiveFunc, transientErr error) {
			// Keep the retry interval well below the default Eventually timeout used by receive.
			bufferClient := newBufferClientWithCustomTimeouts(10*time.Millisecond, defaultSendTimeout, defaultTotalTimeout)
			event, result := send(bufferClient)

			receive(bufferClient).ErrorCh <- transientErr

			retried := receive(bufferClient)
			Expect(retried.Event).To(BeIdenticalTo(event))
			retried.ErrorCh <- nil
			Eventually(result).Should(Receive(BeNil()))
		},
			Entry("domain event, non-gRPC error such as a closed stream", sendDomainEvent, receiveDomainEvent, io.EOF),
			Entry("domain event, Unavailable", sendDomainEvent, receiveDomainEvent, status.Error(codes.Unavailable, "transport is closing")),
			Entry("domain event, DeadlineExceeded", sendDomainEvent, receiveDomainEvent, status.Error(codes.DeadlineExceeded, "deadline exceeded")),
			Entry("domain event, Aborted", sendDomainEvent, receiveDomainEvent, status.Error(codes.Aborted, "aborted")),
			Entry("k8s event, non-gRPC error such as a closed stream", sendK8sEvent, receiveK8sEvent, io.EOF),
			Entry("k8s event, Unavailable", sendK8sEvent, receiveK8sEvent, status.Error(codes.Unavailable, "transport is closing")),
			Entry("k8s event, DeadlineExceeded", sendK8sEvent, receiveK8sEvent, status.Error(codes.DeadlineExceeded, "deadline exceeded")),
			Entry("k8s event, Aborted", sendK8sEvent, receiveK8sEvent, status.Error(codes.Aborted, "aborted")),
		)

		DescribeTable("should return other gRPC errors without retrying", func(send sendFunc, receive receiveFunc, errorPrefix string, code codes.Code) {
			bufferClient := NewBufferClient()
			replyErr := status.Error(code, "handler failed")
			_, result := send(bufferClient)

			receive(bufferClient).ErrorCh <- replyErr

			var err error
			Eventually(result).Should(Receive(&err))
			Expect(err).To(MatchError(replyErr))
			Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("%s with %s", errorPrefix, code))))
			expectNoEventOffered(bufferClient)
		},
			Entry("domain event, Internal", sendDomainEvent, receiveDomainEvent, "failed to send domain notify event", codes.Internal),
			Entry("domain event, InvalidArgument", sendDomainEvent, receiveDomainEvent, "failed to send domain notify event", codes.InvalidArgument),
			Entry("k8s event, Internal", sendK8sEvent, receiveK8sEvent, "failed to notify k8s event", codes.Internal),
			Entry("k8s event, InvalidArgument", sendK8sEvent, receiveK8sEvent, "failed to notify k8s event", codes.InvalidArgument),
		)
	})

	Context("used by the Notifier", func() {
		var (
			bufferClient *client
			notifier     *Notifier
		)

		BeforeEach(func() {
			bufferClient = NewBufferClient()
			notifier = NewNotifier(bufferClient)
			DeferCleanup(notifier.Close)
		})

		It("should deliver domain events", func() {
			domain := api.NewMinimalDomain("testvmi")
			result := callAsync(func() error {
				return notifier.SendDomainEvent(watch.Event{Type: watch.Added, Object: domain})
			})

			invoke := receiveInvoke(bufferClient.DomainEventChan())
			Expect(invoke.Event).To(Equal(&watch.Event{Type: watch.Added, Object: domain}))

			invoke.ErrorCh <- nil
			Eventually(result).Should(Receive(BeNil()))
		})

		It("should deliver k8s events", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = "4321"
			result := callAsync(func() error {
				return notifier.SendK8sEvent(vmi, k8sv1.EventTypeWarning, "fooReason", "barMessage")
			})

			invoke := receiveInvoke(bufferClient.K8SEventChan())
			event := invoke.Event
			Expect(event).ToNot(BeNil())
			Expect(event.Type).To(Equal(k8sv1.EventTypeWarning))
			Expect(event.Reason).To(Equal("fooReason"))
			Expect(event.Message).To(Equal("barMessage"))
			Expect(event.InvolvedObject.Name).To(Equal(vmi.Name))
			Expect(event.InvolvedObject.UID).To(Equal(vmi.UID))

			invoke.ErrorCh <- nil
			Eventually(result).Should(Receive(BeNil()))
		})
	})
})

func newDomainEvent() *watch.Event {
	return &watch.Event{Type: watch.Modified, Object: api.NewMinimalDomain("testvmi")}
}

func newK8sEvent() *k8sv1.Event {
	return &k8sv1.Event{Type: k8sv1.EventTypeNormal, Reason: "fooReason", Message: "barMessage"}
}

// sendFunc hands a new event to the client asynchronously and returns the event together
// with the channel the handler's result is delivered on.
type sendFunc func(*client) (event any, result <-chan error)

// receiveFunc takes the next offered event off the client's channel.
type receiveFunc func(*client) Invoke[any]

func sendDomainEvent(c *client) (any, <-chan error) {
	event := newDomainEvent()
	return event, callAsync(func() error { return c.HandleDomainEvent(event) })
}

func receiveDomainEvent(c *client) Invoke[any] {
	GinkgoHelper()
	return toAnyInvoke(receiveInvoke(c.DomainEventChan()))
}

func sendK8sEvent(c *client) (any, <-chan error) {
	event := newK8sEvent()
	return event, callAsync(func() error { return c.HandleK8SEvent(event) })
}

func receiveK8sEvent(c *client) Invoke[any] {
	GinkgoHelper()
	return toAnyInvoke(receiveInvoke(c.K8SEventChan()))
}

// toAnyInvoke erases the event type so domain and k8s events can share table bodies.
func toAnyInvoke[T any](invoke Invoke[T]) Invoke[any] {
	return Invoke[any]{Event: invoke.Event, ErrorCh: invoke.ErrorCh}
}

// expectNoEventOffered asserts that the client offers no event on any of its channels.
func expectNoEventOffered(c *client, description ...any) {
	GinkgoHelper()
	Consistently(c.DomainEventChan(), 50*time.Millisecond).ShouldNot(Receive(), description...)
	Consistently(c.K8SEventChan(), 50*time.Millisecond).ShouldNot(Receive(), description...)
}

// callAsync runs f in a goroutine and delivers its result on the returned channel.
func callAsync(f func() error) <-chan error {
	result := make(chan error, 1)
	go func() {
		result <- f()
	}()
	return result
}

func receiveInvoke[T any](ch <-chan Invoke[T]) Invoke[T] {
	GinkgoHelper()
	var invoke Invoke[T]
	EventuallyWithOffset(1, ch).Should(Receive(&invoke))
	return invoke
}
