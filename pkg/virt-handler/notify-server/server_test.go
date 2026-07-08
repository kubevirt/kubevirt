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

package eventsserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"

	notifyv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type errorStore struct {
	cache.Store
}

func (s *errorStore) GetByKey(_ string) (interface{}, bool, error) {
	return nil, false, fmt.Errorf("store error")
}

var _ = Describe("RunServer", func() {
	const testWatchInterval = 100 * time.Millisecond

	var (
		virtShareDir string
		stopChan     chan struct{}
		eventChan    chan watch.Event
		recorder     record.EventRecorder
		vmiStore     cache.Store
	)

	BeforeEach(func() {
		virtShareDir = GinkgoT().TempDir()
		stopChan = make(chan struct{})
		eventChan = make(chan watch.Event, 10)
		recorder = record.NewFakeRecorder(10)
		vmiStore = cache.NewStore(cache.MetaNamespaceKeyFunc)
	})

	startServer := func() chan error {
		errChan := make(chan error, 1)
		go func() {
			defer GinkgoRecover()
			errChan <- RunServer(virtShareDir, stopChan, eventChan, recorder, vmiStore, testWatchInterval)
		}()
		sockFile := filepath.Join(virtShareDir, "domain-notify.sock")
		Eventually(func() bool {
			_, err := os.Lstat(sockFile)
			return err == nil
		}).WithTimeout(5 * time.Second).Should(BeTrue())
		return errChan
	}

	It("should return an error when socket file is deleted", func() {
		errChan := startServer()

		sockFile := filepath.Join(virtShareDir, "domain-notify.sock")
		Expect(os.Remove(sockFile)).To(Succeed())

		Eventually(errChan).WithTimeout(5 * time.Second).Should(Receive(MatchError(ContainSubstring("removed or replaced externally"))))
	})

	It("should shut down gracefully when stopChan is closed", func() {
		errChan := startServer()

		close(stopChan)

		Eventually(errChan).WithTimeout(15 * time.Second).Should(Receive(BeNil()))
	})

	It("should keep running when socket file is unchanged", func() {
		errChan := startServer()

		Consistently(errChan).WithTimeout(500 * time.Millisecond).ShouldNot(Receive())

		close(stopChan)
		Eventually(errChan).WithTimeout(15 * time.Second).Should(Receive(BeNil()))
	})
})

var _ = Describe("HandleDomainEvent", func() {
	var (
		notifyServer *Notify
		eventChan    chan watch.Event
	)

	BeforeEach(func() {
		eventChan = make(chan watch.Event, 10)
		notifyServer = &Notify{EventChan: eventChan}
	})

	DescribeTable("should push the correct event type to the channel",
		func(eventType string, expectedType watch.EventType) {
			domain := &api.Domain{}
			domain.Name = "test-domain"
			domainJSON, err := json.Marshal(domain)
			Expect(err).ToNot(HaveOccurred())

			resp, err := notifyServer.HandleDomainEvent(context.Background(), &notifyv1.DomainEventRequest{
				DomainJSON: domainJSON,
				EventType:  eventType,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Success).To(BeTrue())

			var ev watch.Event
			Eventually(eventChan).Should(Receive(&ev))
			Expect(ev.Type).To(Equal(expectedType))
			Expect(ev.Object.(*api.Domain).Name).To(Equal("test-domain"))
		},
		Entry("Added", string(watch.Added), watch.Added),
		Entry("Modified", string(watch.Modified), watch.Modified),
		Entry("Deleted", string(watch.Deleted), watch.Deleted),
	)

	It("should handle Error event type with status", func() {
		domain := &api.Domain{}
		domainJSON, err := json.Marshal(domain)
		Expect(err).ToNot(HaveOccurred())

		k8sStatus := &metav1.Status{Message: "domain crashed"}
		statusJSON, err := json.Marshal(k8sStatus)
		Expect(err).ToNot(HaveOccurred())

		resp, err := notifyServer.HandleDomainEvent(context.Background(), &notifyv1.DomainEventRequest{
			DomainJSON: domainJSON,
			StatusJSON: statusJSON,
			EventType:  string(watch.Error),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Success).To(BeTrue())

		var ev watch.Event
		Eventually(eventChan).Should(Receive(&ev))
		Expect(ev.Type).To(Equal(watch.Error))
		Expect(ev.Object.(*metav1.Status).Message).To(Equal("domain crashed"))
	})

	It("should succeed with empty DomainJSON and StatusJSON", func() {
		resp, err := notifyServer.HandleDomainEvent(context.Background(), &notifyv1.DomainEventRequest{
			EventType: string(watch.Added),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Success).To(BeTrue())

		var ev watch.Event
		Eventually(eventChan).Should(Receive(&ev))
		Expect(ev.Type).To(Equal(watch.Added))
	})

	It("should succeed without pushing when event type is unknown", func() {
		resp, err := notifyServer.HandleDomainEvent(context.Background(), &notifyv1.DomainEventRequest{
			EventType: "UnknownType",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Success).To(BeTrue())
		Consistently(eventChan).ShouldNot(Receive())
	})

	It("should return InvalidArgument for invalid DomainJSON", func() {
		resp, err := notifyServer.HandleDomainEvent(context.Background(), &notifyv1.DomainEventRequest{
			DomainJSON: []byte("not valid json"),
			EventType:  string(watch.Added),
		})
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		st, ok := status.FromError(err)
		Expect(ok).To(BeTrue())
		Expect(st.Code()).To(Equal(codes.InvalidArgument))
		Expect(st.Message()).To(ContainSubstring("unmarshal domain json"))
	})

	It("should return InvalidArgument for invalid StatusJSON", func() {
		domain := &api.Domain{}
		domainJSON, err := json.Marshal(domain)
		Expect(err).ToNot(HaveOccurred())

		resp, err := notifyServer.HandleDomainEvent(context.Background(), &notifyv1.DomainEventRequest{
			DomainJSON: domainJSON,
			StatusJSON: []byte("not valid json"),
			EventType:  string(watch.Error),
		})
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		st, ok := status.FromError(err)
		Expect(ok).To(BeTrue())
		Expect(st.Code()).To(Equal(codes.InvalidArgument))
		Expect(st.Message()).To(ContainSubstring("unmarshal status json"))
	})
})

var _ = Describe("HandleK8SEvent", func() {
	const (
		vmiName      = "test-vmi"
		vmiNamespace = "test-ns"
		vmiUID       = types.UID("test-uid-123")
	)

	var (
		notifyServer *Notify
		eventChan    chan watch.Event
		fakeRecorder *record.FakeRecorder
		vmiStore     cache.Store
	)

	BeforeEach(func() {
		eventChan = make(chan watch.Event, 10)
		fakeRecorder = record.NewFakeRecorder(10)
		vmiStore = cache.NewStore(cache.MetaNamespaceKeyFunc)
		notifyServer = &Notify{
			EventChan: eventChan,
			recorder:  fakeRecorder,
			vmiStore:  vmiStore,
		}

		testVMI := &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vmiName,
				Namespace: vmiNamespace,
				UID:       vmiUID,
			},
		}
		Expect(vmiStore.Add(testVMI)).To(Succeed())
	})

	makeEventJSON := func(namespace, name string, uid types.UID, reason, message string) []byte {
		event := k8sv1.Event{
			InvolvedObject: k8sv1.ObjectReference{
				Name:      name,
				Namespace: namespace,
				UID:       uid,
			},
			Type:    "Normal",
			Reason:  reason,
			Message: message,
		}
		data, err := json.Marshal(event)
		Expect(err).ToNot(HaveOccurred())
		return data
	}

	It("should record event when VMI exists and UID matches", func() {
		eventJSON := makeEventJSON(vmiNamespace, vmiName, vmiUID, "Started", "VM started")

		resp, err := notifyServer.HandleK8SEvent(context.Background(), &notifyv1.K8SEventRequest{
			EventJSON: eventJSON,
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Success).To(BeTrue())

		Eventually(fakeRecorder.Events).Should(Receive(ContainSubstring("Started")))
	})

	It("should return InvalidArgument for invalid EventJSON", func() {
		resp, err := notifyServer.HandleK8SEvent(context.Background(), &notifyv1.K8SEventRequest{
			EventJSON: []byte("not valid json"),
		})
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		st, ok := status.FromError(err)
		Expect(ok).To(BeTrue())
		Expect(st.Code()).To(Equal(codes.InvalidArgument))
		Expect(st.Message()).To(ContainSubstring("unmarshal k8s event"))
	})

	It("should return NotFound when VMI does not exist in store", func() {
		eventJSON := makeEventJSON(vmiNamespace, "nonexistent-vmi", vmiUID, "Started", "VM started")

		resp, err := notifyServer.HandleK8SEvent(context.Background(), &notifyv1.K8SEventRequest{
			EventJSON: eventJSON,
		})
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		st, ok := status.FromError(err)
		Expect(ok).To(BeTrue())
		Expect(st.Code()).To(Equal(codes.NotFound))
		Expect(st.Message()).To(ContainSubstring("VMI not found"))
	})

	It("should return NotFound when VMI UID does not match", func() {
		eventJSON := makeEventJSON(vmiNamespace, vmiName, "wrong-uid", "Started", "VM started")

		resp, err := notifyServer.HandleK8SEvent(context.Background(), &notifyv1.K8SEventRequest{
			EventJSON: eventJSON,
		})
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		st, ok := status.FromError(err)
		Expect(ok).To(BeTrue())
		Expect(st.Code()).To(Equal(codes.NotFound))
		Expect(st.Message()).To(ContainSubstring("VMI not found"))
	})

	It("should return Internal when store returns an error", func() {
		notifyServer.vmiStore = &errorStore{
			Store: cache.NewStore(cache.MetaNamespaceKeyFunc),
		}
		eventJSON := makeEventJSON(vmiNamespace, vmiName, vmiUID, "Started", "VM started")

		resp, err := notifyServer.HandleK8SEvent(context.Background(), &notifyv1.K8SEventRequest{
			EventJSON: eventJSON,
		})
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		st, ok := status.FromError(err)
		Expect(ok).To(BeTrue())
		Expect(st.Code()).To(Equal(codes.Internal))
		Expect(st.Message()).To(ContainSubstring("failed to get VMI"))
	})
})
