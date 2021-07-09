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
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
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
	var podsDir string
	var socketsDir string
	var ghostCacheDir string
	var informer cache.SharedInformer
	var stopChan chan struct{}
	var wg *sync.WaitGroup
	var ctrl *gomock.Controller
	var domainManager *virtwrap.MockDomainManager
	var socketPath string
	var resyncPeriod int

	podUID := "1234"

	BeforeEach(func() {
		resyncPeriod = 5
		stopChan = make(chan struct{})
		wg = &sync.WaitGroup{}

		shareDir, err = ioutil.TempDir("", "kubevirt-share")
		Expect(err).ToNot(HaveOccurred())

		podsDir, err = ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())

		ghostCacheDir, err = ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())

		InitializeGhostRecordCache(ghostCacheDir)

		cmdclient.SetLegacyBaseDir(shareDir)
		cmdclient.SetPodsBaseDir(podsDir)

		socketsDir = filepath.Join(shareDir, "sockets")
		os.Mkdir(socketsDir, 0755)
		os.Mkdir(filepath.Join(socketsDir, "1234"), 0755)

		socketPath = cmdclient.SocketFilePathOnHost(podUID)
		os.MkdirAll(filepath.Dir(socketPath), 0755)

		informer, err = NewSharedInformer(shareDir, 10, nil, nil, time.Duration(resyncPeriod)*time.Second)
		Expect(err).ToNot(HaveOccurred())

		ctrl = gomock.NewController(GinkgoT())
		domainManager = virtwrap.NewMockDomainManager(ctrl)
	})

	AfterEach(func() {
		close(stopChan)
		wg.Wait()
		os.RemoveAll(shareDir)
		os.RemoveAll(podsDir)
		os.RemoveAll(ghostCacheDir)
		DeleteGhostRecord("test", "test")
		ctrl.Finish()
	})

	verifyObj := func(key string, domain *api.Domain) {
		obj, exists, err := informer.GetStore().GetByKey(key)
		Expect(err).To(BeNil())

		if domain != nil {
			Expect(exists).To(BeTrue())

			eventDomain := obj.(*api.Domain)
			eventDomain.Spec.XMLName = xml.Name{}
			Expect(reflect.DeepEqual(&domain.Spec, &eventDomain.Spec)).To(BeTrue())
		} else {

			Expect(exists).To(BeFalse())
		}
	}

	Context("with ghost record cache", func() {
		It("Should be able to retrieve uid", func() {
			err := AddGhostRecord("test1-namespace", "test1", "somefile1", "1234-1")
			Expect(err).ToNot(HaveOccurred())

			uid := LastKnownUIDFromGhostRecordCache("test1-namespace/test1")
			Expect(string(uid)).To(Equal("1234-1"))

		})

		It("Should find ghost record by socket ", func() {
			err := AddGhostRecord("test1-namespace", "test1", "somefile1", "1234-1")
			Expect(err).ToNot(HaveOccurred())

			record, exists := findGhostRecordBySocket("somefile1")
			Expect(exists).To(BeTrue())
			Expect(record.Name).To(Equal("test1"))

			record, exists = findGhostRecordBySocket("does-not-exist")
			Expect(exists).To(BeFalse())
		})

		It("Should initialize cache from disk", func() {
			err := AddGhostRecord("test1-namespace", "test1", "somefile1", "1234-1")
			Expect(err).ToNot(HaveOccurred())
			err = AddGhostRecord("test2-namespace", "test2", "somefile2", "1234-2")
			Expect(err).ToNot(HaveOccurred())

			clearGhostRecordCache()

			_, exists := ghostRecordGlobalCache["test1-namespace/test1"]
			Expect(exists).To(BeFalse())

			err = InitializeGhostRecordCache(ghostCacheDir)
			Expect(err).ToNot(HaveOccurred())

			record, exists := ghostRecordGlobalCache["test1-namespace/test1"]
			Expect(exists).To(BeTrue())
			Expect(string(record.UID)).To(Equal("1234-1"))
			Expect(string(record.SocketFile)).To(Equal("somefile1"))

			record, exists = ghostRecordGlobalCache["test2-namespace/test2"]
			Expect(exists).To(BeTrue())
			Expect(string(record.UID)).To(Equal("1234-2"))
			Expect(string(record.SocketFile)).To(Equal("somefile2"))
		})

		It("Should delete ghost record from cache and disk", func() {
			err := AddGhostRecord("test1-namespace", "test1", "somefile1", "1234-1")
			Expect(err).ToNot(HaveOccurred())

			_, exists := ghostRecordGlobalCache["test1-namespace/test1"]
			Expect(exists).To(BeTrue())

			exists, err = diskutils.FileExists(filepath.Join(ghostRecordDir, "1234-1"))
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())

			err = DeleteGhostRecord("test1-namespace", "test1")
			Expect(err).ToNot(HaveOccurred())

			_, exists = ghostRecordGlobalCache["test1-namespace/test1"]
			Expect(exists).To(BeFalse())

			exists, err = diskutils.FileExists(filepath.Join(ghostRecordDir, "1234-1"))
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())

		})

		It("Should reject adding a ghost record with missing data", func() {
			err := AddGhostRecord("", "test1", "somefile1", "1234-1")
			Expect(err).To(HaveOccurred())

			err = AddGhostRecord("test1-namespace", "", "somefile1", "1234-1")
			Expect(err).To(HaveOccurred())

			err = AddGhostRecord("test1-namespace", "test1", "", "1234-1")
			Expect(err).To(HaveOccurred())

			err = AddGhostRecord("test1-namespace", "test1", "somefile1", "")
			Expect(err).To(HaveOccurred())

		})
	})

	Context("with notification server", func() {
		It("should list current domains.", func() {
			var list []*api.Domain

			list = append(list, api.NewMinimalDomain("testvmi1"))

			domainManager.EXPECT().ListAllDomains().Return(list, nil)
			domainManager.EXPECT().GetGuestOSInfo().Return(&api.GuestOSInfo{})
			domainManager.EXPECT().InterfacesStatus(list[0].Spec.Devices.Interfaces).Return([]api.InterfaceStatus{})

			runCMDServer(wg, socketPath, domainManager, stopChan, nil)

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

		It("should list current domains including inactive domains with ghost record", func() {
			var list []*api.Domain

			list = append(list, api.NewMinimalDomain("testvmi1"))

			domainManager.EXPECT().ListAllDomains().Return(list, nil)
			domainManager.EXPECT().GetGuestOSInfo().Return(&api.GuestOSInfo{})
			domainManager.EXPECT().InterfacesStatus(list[0].Spec.Devices.Interfaces).Return([]api.InterfaceStatus{})

			err := AddGhostRecord("test1-namespace", "test1", "somefile1", "1234-1")
			Expect(err).ToNot(HaveOccurred())
			runCMDServer(wg, socketPath, domainManager, stopChan, nil)

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

			// includes both the domain with an active socket and the ghost record with deleted socket
			Expect(len(listResults)).To(Equal(2))
		})
		It("should detect active domains at startup.", func() {
			var list []*api.Domain

			domain := api.NewMinimalDomain("test")
			list = append(list, domain)

			domainManager.EXPECT().ListAllDomains().Return(list, nil)
			domainManager.EXPECT().GetGuestOSInfo().Return(&api.GuestOSInfo{})
			domainManager.EXPECT().InterfacesStatus(list[0].Spec.Devices.Interfaces).Return([]api.InterfaceStatus{})

			runCMDServer(wg, socketPath, domainManager, stopChan, nil)

			// ensure we can connect to the server first.
			client, err := cmdclient.NewClient(socketPath)
			Expect(err).ToNot(HaveOccurred())
			client.Close()

			runInformer(wg, stopChan, informer)
			cache.WaitForCacheSync(stopChan, informer.HasSynced)

			verifyObj("default/test", domain)
		})

		It("should resync active domains after resync period.", func() {

			domain := api.NewMinimalDomain("test")
			domainManager.EXPECT().ListAllDomains().Return([]*api.Domain{domain}, nil)
			domainManager.EXPECT().GetGuestOSInfo().Return(&api.GuestOSInfo{})
			domainManager.EXPECT().InterfacesStatus(domain.Spec.Devices.Interfaces).Return([]api.InterfaceStatus{})
			// now prove if we make a change, like adding a label, that the resync
			// will pick that change up automatically
			newDomain := domain.DeepCopy()
			newDomain.ObjectMeta.Labels = make(map[string]string)
			newDomain.ObjectMeta.Labels["some-label"] = "some-value"
			domainManager.EXPECT().ListAllDomains().Return([]*api.Domain{newDomain}, nil)
			domainManager.EXPECT().GetGuestOSInfo().Return(nil)
			domainManager.EXPECT().InterfacesStatus(newDomain.Spec.Devices.Interfaces).Return(nil)

			runCMDServer(wg, socketPath, domainManager, stopChan, nil)

			// ensure we can connect to the server first.
			client, err := cmdclient.NewClient(socketPath)
			Expect(err).ToNot(HaveOccurred())
			client.Close()

			runInformer(wg, stopChan, informer)
			cache.WaitForCacheSync(stopChan, informer.HasSynced)

			verifyObj("default/test", domain)

			time.Sleep(time.Duration(resyncPeriod+1) * time.Second)

			obj, exists, err := informer.GetStore().GetByKey("default/test")
			Expect(err).To(BeNil())
			Expect(exists).To(BeTrue())

			eventDomain := obj.(*api.Domain)
			val, ok := eventDomain.ObjectMeta.Labels["some-label"]

			Expect(ok).To(BeTrue())
			Expect(val).To(Equal("some-value"))
		})

		It("should detect expired legacy watchdog file.", func() {
			f, err := os.Create(socketPath)
			Expect(err).ToNot(HaveOccurred())
			f.Close()

			d := &DomainWatcher{
				backgroundWatcherStarted: false,
				virtShareDir:             shareDir,
				watchdogTimeout:          1,
				unresponsiveSockets:      make(map[string]int64),
				resyncPeriod:             time.Duration(1) * time.Hour,
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
				Expect(event.Object.(*api.Domain).ObjectMeta.DeletionTimestamp).ToNot(BeNil())
				Expect(event.Type).To(Equal(watch.Modified))
			case <-timeout:
				timedOut = true
			}

			Expect(timedOut).To(BeFalse())

		}, 5)

		It("should detect unresponsive sockets.", func() {

			f, err := os.Create(socketPath)
			Expect(err).ToNot(HaveOccurred())
			f.Close()

			AddGhostRecord("test", "test", socketPath, "1234")

			d := &DomainWatcher{
				backgroundWatcherStarted: false,
				virtShareDir:             shareDir,
				watchdogTimeout:          1,
				unresponsiveSockets:      make(map[string]int64),
				resyncPeriod:             time.Duration(1) * time.Hour,
			}

			err = d.startBackground()
			Expect(err).ToNot(HaveOccurred())
			defer d.Stop()

			timedOut := false
			timeout := time.After(5 * time.Second)
			select {
			case event := <-d.eventChan:
				Expect(event.Type).To(Equal(watch.Modified))
				Expect(event.Object.(*api.Domain).ObjectMeta.DeletionTimestamp).ToNot(BeNil())
			case <-timeout:
				timedOut = true
			}

			Expect(timedOut).To(BeFalse())

		}, 6)

		It("should detect responsive sockets and not mark for deletion.", func() {

			l, err := net.Listen("unix", socketPath)
			Expect(err).ToNot(HaveOccurred())
			defer l.Close()

			go func() {
				for {
					conn, err := l.Accept()
					if err != nil {
						// closes when socket listener is closed
						return
					}
					conn.Close()
				}
			}()

			err = AddGhostRecord("test", "test", socketPath, "1234")
			Expect(err).ToNot(HaveOccurred())

			d := &DomainWatcher{
				backgroundWatcherStarted: false,
				virtShareDir:             shareDir,
				watchdogTimeout:          1,
				unresponsiveSockets:      make(map[string]int64),
				resyncPeriod:             time.Duration(1) * time.Hour,
			}

			err = d.startBackground()
			Expect(err).ToNot(HaveOccurred())
			defer d.Stop()

			timedOut := false
			timeout := time.After(5 * time.Second)
			select {
			case _ = <-d.eventChan:
				// fall through
			case <-timeout:
				timedOut = true
			}

			Expect(timedOut).To(BeTrue())
		}, 6)

		It("should not return errors when encountering disconnected clients at startup.", func() {
			var list []*api.Domain

			domain := api.NewMinimalDomain("test")
			list = append(list, domain)

			domainManager.EXPECT().ListAllDomains().Return(list, nil)
			domainManager.EXPECT().GetGuestOSInfo().Return(&api.GuestOSInfo{})
			domainManager.EXPECT().InterfacesStatus(list[0].Spec.Devices.Interfaces).Return([]api.InterfaceStatus{})

			// This file doesn't have a unix sock server behind it
			// verify list still completes regardless
			f, err := os.Create(filepath.Join(socketsDir, "default_fakevm_sock"))
			Expect(err).ToNot(HaveOccurred())
			f.Close()
			runCMDServer(wg, socketPath, domainManager, stopChan, nil)
			// ensure we can connect to the server first.
			client, err := cmdclient.NewClient(socketPath)
			Expect(err).ToNot(HaveOccurred())
			client.Close()

			runInformer(wg, stopChan, informer)
			cache.WaitForCacheSync(stopChan, informer.HasSynced)

			verifyObj("default/test", domain)
		})
		It("should watch for domain events.", func() {
			domain := api.NewMinimalDomain("test")

			runInformer(wg, stopChan, informer)
			cache.WaitForCacheSync(stopChan, informer.HasSynced)

			client := notifyclient.NewNotifier(shareDir)

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

func runCMDServer(wg *sync.WaitGroup, socketPath string,
	domainManager virtwrap.DomainManager,
	stopChan chan struct{},
	options *cmdserver.ServerOptions) {
	wg.Add(1)
	done, _ := cmdserver.RunServer(socketPath, domainManager, stopChan, options)
	go func() {
		<-done
		wg.Done()
	}()
}

func runInformer(wg *sync.WaitGroup, stopChan chan struct{}, informer cache.SharedInformer) {
	wg.Add(1)
	go func() { informer.Run(stopChan); wg.Done() }()
}
