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

package cache

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	notifyclient "kubevirt.io/kubevirt/pkg/virt-launcher/notify-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	cmdserver "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cmd-server"
	"kubevirt.io/kubevirt/pkg/watchdog"
)

var _ = Describe("Domain informer", func() {
	var err error
	var shareDir string
	var socketsDir string
	var informer cache.SharedInformer
	var stopChan chan struct{}
	var ctrl *gomock.Controller
	var domainManager *virtwrap.MockDomainManager

	BeforeEach(func() {
		stopChan = make(chan struct{})

		shareDir, err = ioutil.TempDir("", "kubevirt-share")
		Expect(err).ToNot(HaveOccurred())

		socketsDir = filepath.Join(shareDir, "sockets")
		os.Mkdir(socketsDir, 0755)

		informer, err = NewSharedInformer(shareDir, 10, nil, nil)
		Expect(err).ToNot(HaveOccurred())

		ctrl = gomock.NewController(GinkgoT())
		domainManager = virtwrap.NewMockDomainManager(ctrl)
	})

	AfterEach(func() {
		close(stopChan)
		os.RemoveAll(shareDir)
		ctrl.Finish()
	})

	verifyObj := func(key string, domain *api.Domain) {
		obj, exists, err := informer.GetStore().GetByKey(key)
		Expect(err).To(BeNil())

		if domain != nil {
			Expect(exists).To(BeTrue())

			eventDomain := obj.(*api.Domain)
			eventDomain.Spec.XMLName = xml.Name{}
			Expect(reflect.DeepEqual(&domain.Spec, &eventDomain.Spec)).To(Equal(true))
		} else {

			Expect(exists).To(BeFalse())
		}
	}

	Context("with notification server", func() {
		It("should list current domains.", func() {
			var list []*api.Domain

			list = append(list, api.NewMinimalDomain("testvmi1"))

			socketPath := filepath.Join(socketsDir, "default_testvmi1_sock")
			domainManager.EXPECT().ListAllDomains().Return(list, nil)

			cmdserver.RunServer(socketPath, domainManager, stopChan, nil)

			// ensure we can connect to the server first.
			client, err := cmdclient.NewClient(socketPath)
			Expect(err).ToNot(HaveOccurred())
			client.Close()

			d := &DomainWatcher{
				backgroundWatcherStarted: false,
				virtShareDir:             shareDir,
			}

			listResults, err := d.listAllKnownDomains()
			Expect(err).ToNot(HaveOccurred())

			Expect(len(listResults)).To(Equal(1))
		})
		It("should detect active domains at startup.", func() {
			var list []*api.Domain

			domain := api.NewMinimalDomain("test")
			list = append(list, domain)

			socketPath := filepath.Join(socketsDir, "default_test_sock")
			domainManager.EXPECT().ListAllDomains().Return(list, nil)

			cmdserver.RunServer(socketPath, domainManager, stopChan, nil)

			// ensure we can connect to the server first.
			client, err := cmdclient.NewClient(socketPath)
			Expect(err).ToNot(HaveOccurred())
			client.Close()

			go informer.Run(stopChan)
			cache.WaitForCacheSync(stopChan, informer.HasSynced)

			verifyObj("default/test", domain)
		})

		It("should detect expired watchdog file.", func() {
			socketPath := filepath.Join(socketsDir, "default_test_sock")
			f, err := os.Create(socketPath)
			Expect(err).ToNot(HaveOccurred())
			f.Close()

			d := &DomainWatcher{
				backgroundWatcherStarted: false,
				virtShareDir:             shareDir,
				watchdogTimeout:          1,
			}

			watchdogFile := watchdog.WatchdogFileFromNamespaceName(shareDir, "default", "test")
			os.MkdirAll(filepath.Dir(watchdogFile), 0755)
			watchdog.WatchdogFileUpdate(watchdogFile, "somestring")

			err = d.startBackground()
			Expect(err).ToNot(HaveOccurred())
			defer d.Stop()

			timedOut := false
			timeout := time.After(3 * time.Second)
			select {
			case event := <-d.eventChan:
				Expect(event.Type).To(Equal(watch.Deleted))
			case <-timeout:
				timedOut = true
			}

			Expect(timedOut).To(Equal(false))

		}, 5)

		It("should not return errors when encountering disconnected clients at startup.", func() {
			var list []*api.Domain

			domain := api.NewMinimalDomain("test")
			list = append(list, domain)

			socketPath := filepath.Join(socketsDir, "default_test_sock")
			domainManager.EXPECT().ListAllDomains().Return(list, nil)

			// This file doesn't have a unix sock server behind it
			// verify list still completes regardless
			f, err := os.Create(filepath.Join(socketsDir, "default_fakevm_sock"))
			f.Close()
			cmdserver.RunServer(socketPath, domainManager, stopChan, nil)

			// ensure we can connect to the server first.
			client, err := cmdclient.NewClient(socketPath)
			Expect(err).ToNot(HaveOccurred())
			client.Close()

			go informer.Run(stopChan)
			cache.WaitForCacheSync(stopChan, informer.HasSynced)

			verifyObj("default/test", domain)
		})
		It("should watch for domain events.", func() {
			domain := api.NewMinimalDomain("test")

			go informer.Run(stopChan)
			cache.WaitForCacheSync(stopChan, informer.HasSynced)

			client, err := notifyclient.NewNotifier(shareDir)
			Expect(err).ToNot(HaveOccurred())

			// verify add
			err = client.SendDomainEvent(watch.Event{Type: watch.Added, Object: domain})
			Expect(err).ToNot(HaveOccurred())
			cache.WaitForCacheSync(stopChan, informer.HasSynced)
			verifyObj("default/test", domain)

			// verify modify
			domain.Spec.UUID = "fakeuuid"
			err = client.SendDomainEvent(watch.Event{Type: watch.Modified, Object: domain})
			Expect(err).ToNot(HaveOccurred())
			cache.WaitForCacheSync(stopChan, informer.HasSynced)
			verifyObj("default/test", domain)

			// verify modify
			err = client.SendDomainEvent(watch.Event{Type: watch.Deleted, Object: domain})
			Expect(err).ToNot(HaveOccurred())
			cache.WaitForCacheSync(stopChan, informer.HasSynced)
			verifyObj("default/test", nil)
		})
	})
})
