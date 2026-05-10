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
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	notifyv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/v1"
	eventsserver "kubevirt.io/kubevirt/pkg/virt-handler/notify-server"
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

var _ = Describe("HandleDomainEvent", func() {
	var (
		eventChan chan watch.Event
		notify    *eventsserver.Notify
	)

	BeforeEach(func() {
		eventChan = make(chan watch.Event, 1)
		notify = &eventsserver.Notify{
			EventChan: eventChan,
		}
	})

	It("should not translate domain error events to informer watch.Error event", func() {
		statusJSON, err := json.Marshal(&metav1.Status{
			Status:  metav1.StatusFailure,
			Message: "domain error",
		})
		Expect(err).NotTo(HaveOccurred())

		response, err := notify.HandleDomainEvent(context.Background(), &notifyv1.DomainEventRequest{
			EventType:  string(watch.Error),
			StatusJSON: statusJSON,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(response.Success).To(BeTrue())

		Consistently(eventChan).WithTimeout(100 * time.Millisecond).ShouldNot(Receive())
	})
})
