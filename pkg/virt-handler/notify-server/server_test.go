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

package eventsserver_test

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	grpcstatus "google.golang.org/grpc/status"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"

	notifyv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/v1"
	eventsserver "kubevirt.io/kubevirt/pkg/virt-handler/notify-server"
	"kubevirt.io/kubevirt/pkg/virt-handler/notify-server/auth"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

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
			errChan <- eventsserver.RunServer(virtShareDir, stopChan, eventChan, recorder, vmiStore, testWatchInterval)
		}()
		// Wait for socket file to be created before returning
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

var _ = Describe("Client identity", func() {
	const (
		rpcTimeout = 5 * time.Second
		settling   = 200 * time.Millisecond
	)

	var (
		virtShareDir string
		eventChan    chan watch.Event
		recorder     *record.FakeRecorder
		vmi          *v1.VirtualMachineInstance
	)

	BeforeEach(func() {
		virtShareDir = GinkgoT().TempDir()
		eventChan = make(chan watch.Event, 10)
		recorder = record.NewFakeRecorder(10)

		vmi = v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "testvmi", "1234")
		vmiStore := cache.NewStore(cache.MetaNamespaceKeyFunc)
		Expect(vmiStore.Add(vmi)).To(Succeed())
		// Another VMI known to virt-handler, events about it must be rejected because of the identity alone
		Expect(vmiStore.Add(v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "othervmi", "5678"))).To(Succeed())

		stopChan := make(chan struct{})
		errChan := make(chan error, 1)
		go func() {
			errChan <- eventsserver.RunServer(virtShareDir, stopChan, eventChan, recorder, vmiStore)
		}()
		sockFile := filepath.Join(virtShareDir, "domain-notify.sock")
		Eventually(sockFile).Should(BeAnExistingFile())
		DeferCleanup(func() {
			close(stopChan)
			Eventually(errChan).WithTimeout(15 * time.Second).Should(Receive(BeNil()))
		})
	})

	// dial connects to the notify server, preamble is written ahead of any gRPC traffic like the pipes do
	dial := func(preamble func(net.Conn) error) notifyv1.NotifyClient {
		sockFile := filepath.Join(virtShareDir, "domain-notify.sock")
		conn, err := grpc.NewClient("passthrough:///domain-notify",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
				c, err := (&net.Dialer{}).DialContext(ctx, "unix", sockFile)
				if err != nil {
					return nil, err
				}
				if err := preamble(c); err != nil {
					c.Close()
					return nil, err
				}
				return c, nil
			}),
		)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(conn.Close)
		return notifyv1.NewNotifyClient(conn)
	}

	dialAsVMI := func() notifyv1.NotifyClient {
		return dial(func(conn net.Conn) error {
			return auth.WriteIdentity(conn, auth.IdentityFromVMI(vmi))
		})
	}

	newDomain := func() *api.Domain {
		domain := api.NewMinimalDomainWithNS(vmi.Namespace, vmi.Name)
		domain.UID = vmi.UID
		domain.Spec.Metadata.KubeVirt.UID = vmi.UID
		return domain
	}

	sendDomain := func(client notifyv1.NotifyClient, eventType watch.EventType, domain *api.Domain) error {
		domainJSON, err := json.Marshal(domain)
		Expect(err).ToNot(HaveOccurred())
		ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
		defer cancel()
		_, err = client.HandleDomainEvent(ctx, &notifyv1.DomainEventRequest{
			DomainJSON: domainJSON,
			EventType:  string(eventType),
		})
		return err
	}

	vmiRef := func() k8sv1.ObjectReference {
		return k8sv1.ObjectReference{
			Kind:      "VirtualMachineInstance",
			Namespace: vmi.Namespace,
			Name:      vmi.Name,
			UID:       vmi.UID,
		}
	}

	sendK8sEvent := func(client notifyv1.NotifyClient, involvedObject k8sv1.ObjectReference) (*notifyv1.Response, error) {
		eventJSON, err := json.Marshal(&k8sv1.Event{
			InvolvedObject: involvedObject,
			Type:           k8sv1.EventTypeNormal,
			Reason:         "TestReason",
			Message:        "test message",
		})
		Expect(err).ToNot(HaveOccurred())
		ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
		defer cancel()
		return client.HandleK8SEvent(ctx, &notifyv1.K8SEventRequest{EventJSON: eventJSON})
	}

	DescribeTable("should forward domain events about the client VMI", func(eventType watch.EventType) {
		Expect(sendDomain(dialAsVMI(), eventType, newDomain())).To(Succeed())

		var event watch.Event
		Eventually(eventChan).Should(Receive(&event))
		Expect(event.Type).To(Equal(eventType))
		domain, ok := event.Object.(*api.Domain)
		Expect(ok).To(BeTrue())
		Expect(domain.Namespace).To(Equal(vmi.Namespace))
		Expect(domain.Name).To(Equal(vmi.Name))
		Expect(domain.UID).To(Equal(vmi.UID))
		Expect(domain.Spec.Metadata.KubeVirt.UID).To(Equal(vmi.UID))
	},
		Entry("when added", watch.Added),
		Entry("when modified", watch.Modified),
		Entry("when deleted", watch.Deleted),
	)

	DescribeTable("should reject domain events about another VMI", func(eventType watch.EventType, mutate func(*api.Domain)) {
		domain := newDomain()
		mutate(domain)

		err := sendDomain(dialAsVMI(), eventType, domain)
		Expect(grpcstatus.Code(err)).To(Equal(codes.PermissionDenied))
		Consistently(eventChan, settling).ShouldNot(Receive())
	},
		Entry("in another namespace", watch.Modified, func(d *api.Domain) { d.Namespace = "other" }),
		Entry("with another name", watch.Modified, func(d *api.Domain) { d.Name = "othervmi" }),
		Entry("with another UID", watch.Modified, func(d *api.Domain) { d.UID = "5678" }),
		Entry("with another KubeVirt metadata UID", watch.Modified, func(d *api.Domain) { d.Spec.Metadata.KubeVirt.UID = "5678" }),
		Entry("without UID", watch.Modified, func(d *api.Domain) {
			d.UID = ""
			d.Spec.Metadata.KubeVirt.UID = ""
		}),
		Entry("when added", watch.Added, func(d *api.Domain) { d.Name = "othervmi" }),
		Entry("when deleted", watch.Deleted, func(d *api.Domain) { d.Name = "othervmi" }),
	)

	It("should record k8s events about the client VMI", func() {
		response, err := sendK8sEvent(dialAsVMI(), vmiRef())
		Expect(err).ToNot(HaveOccurred())
		Expect(response.Success).To(BeTrue())
		Eventually(recorder.Events).Should(Receive(Equal("Normal TestReason test message")))
	})

	DescribeTable("should reject k8s events about another VMI", func(mutate func(*k8sv1.ObjectReference)) {
		involvedObject := vmiRef()
		mutate(&involvedObject)

		_, err := sendK8sEvent(dialAsVMI(), involvedObject)
		Expect(grpcstatus.Code(err)).To(Equal(codes.PermissionDenied))
		Consistently(recorder.Events, settling).ShouldNot(Receive())
	},
		Entry("known to virt-handler", func(ref *k8sv1.ObjectReference) {
			ref.Name = "othervmi"
			ref.UID = "5678"
		}),
		Entry("in another namespace", func(ref *k8sv1.ObjectReference) { ref.Namespace = "other" }),
		Entry("with another UID", func(ref *k8sv1.ObjectReference) { ref.UID = "5678" }),
		Entry("without UID", func(ref *k8sv1.ObjectReference) { ref.UID = "" }),
	)

	It("should refuse connections without client identity", func() {
		client := dial(func(net.Conn) error { return nil })

		err := sendDomain(client, watch.Modified, newDomain())
		Expect(grpcstatus.Code(err)).To(Equal(codes.Unavailable))
		Consistently(eventChan, settling).ShouldNot(Receive())
	})
})
