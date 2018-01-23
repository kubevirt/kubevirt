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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package eventsclient

import (
	"io/ioutil"
	"os"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/watch"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	notifyserver "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/notify-server"
)

var _ = Describe("Domain notify", func() {
	var err error
	var shareDir string
	var stop chan struct{}
	var stopped bool
	var eventChan chan watch.Event
	var client *DomainEventClient

	BeforeEach(func() {
		stop = make(chan struct{})
		eventChan = make(chan watch.Event, 100)
		stopped = false
		shareDir, err = ioutil.TempDir("", "kubevirt-share")
		Expect(err).ToNot(HaveOccurred())

		go func() {
			notifyserver.RunServer(shareDir, stop, eventChan)
		}()

		time.Sleep(1 * time.Second)
		client, err = NewDomainEventClient(shareDir)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if stopped == false {
			close(stop)
		}
		os.RemoveAll(shareDir)
	})

	Context("server", func() {
		table.DescribeTable("should accept Domain notify events", func(eventType watch.EventType) {
			domain := api.NewMinimalDomain("testvm")
			err := client.SendDomainEvent(watch.Event{Type: eventType, Object: domain})
			Expect(err).ToNot(HaveOccurred())

			timedOut := false
			timeout := time.After(2 * time.Second)
			select {
			case <-timeout:
				timedOut = true
			case event := <-eventChan:
				newDomain, ok := event.Object.(*api.Domain)
				Expect(ok).To(Equal(true))
				Expect(reflect.DeepEqual(domain, newDomain)).To(Equal(true))
				Expect(event.Type == eventType).To(Equal(true))
			}
			Expect(timedOut).To(Equal(false))
		},
			table.Entry("added", watch.Added),
			table.Entry("deleted", watch.Deleted),
			table.Entry("modified", watch.Modified),
		)
	})
})
