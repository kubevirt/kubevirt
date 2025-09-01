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
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	notifyclient "kubevirt.io/kubevirt/pkg/virt-launcher/notify-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	cmdserver "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cmd-server"
)

func clearGhostRecordCache() {
	ghostRecordGlobalMutex.Lock()
	defer ghostRecordGlobalMutex.Unlock()
	ghostRecordGlobalCache = make(map[string]ghostRecord)
}

var _ = Describe("Domain informer", func() {
	var ghostCacheDir string

	BeforeEach(func() {
		var err error

		ghostCacheDir, err = os.MkdirTemp("", "")
		Expect(err).ToNot(HaveOccurred())

		InitializeGhostRecordCache(ghostCacheDir)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(ghostCacheDir)).To(Succeed())
	})

	Context("listSockets ", func() {
		It("should return socket list from ghost record cache", func() {
			const podUID = "5678"
			const socketPath = "/path/to/domainsock"

			ghostCacheDir := GinkgoT().TempDir()

			Expect(InitializeGhostRecordCache(ghostCacheDir)).To(Succeed())

			err := AddGhostRecord("test-ns", "test-domain", socketPath, podUID)
			Expect(err).ToNot(HaveOccurred())

			socketFiles, err := listSockets(getGhostRecords())
			Expect(err).ToNot(HaveOccurred())
			Expect(socketFiles).To(HaveLen(1))
			Expect(socketFiles[0]).To(Equal(socketPath))

		})
	})

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

			exists, err = diskutils.FileExists(filepath.Join(ghostCacheDir, "1234-1"))
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())

			err = DeleteGhostRecord("test1-namespace", "test1")
			Expect(err).ToNot(HaveOccurred())

			_, exists = ghostRecordGlobalCache["test1-namespace/test1"]
			Expect(exists).To(BeFalse())

			exists, err = diskutils.FileExists(filepath.Join(ghostCacheDir, "1234-1"))
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

	Context("with cmd server", func() {
		const podUID = "1234"
		const domainName = "testvmi1"
		const domainNamespace = "default"
		const resyncPeriod = 5

		var shareDir string
		var podsDir string
		var informer cache.SharedInformer
		var stopChan chan struct{}
		var wg *sync.WaitGroup
		var ctrl *gomock.Controller
		var domainManager *virtwrap.MockDomainManager
		var socketPath string

		BeforeEach(func() {
			var err error

			stopChan = make(chan struct{})
			wg = &sync.WaitGroup{}

			shareDir, err = os.MkdirTemp("", "kubevirt-share")
			Expect(err).ToNot(HaveOccurred())

			podsDir, err = os.MkdirTemp("", "")
			Expect(err).ToNot(HaveOccurred())

			cmdclient.SetPodsBaseDir(podsDir)

			socketPath = cmdclient.SocketFilePathOnHost(podUID)
			Expect(os.MkdirAll(filepath.Dir(socketPath), 0755)).To(Succeed())
			Expect(AddGhostRecord(domainNamespace, domainName, socketPath, podUID)).To(Succeed())

			informer, err = NewSharedInformer(shareDir, 10, nil, nil, time.Duration(resyncPeriod)*time.Second)
			Expect(err).ToNot(HaveOccurred())

			ctrl = gomock.NewController(GinkgoT())
			domainManager = virtwrap.NewMockDomainManager(ctrl)
		})

		AfterEach(func() {
			Expect(DeleteGhostRecord(domainNamespace, domainName)).To(Succeed())
			close(stopChan)
			wg.Wait()

			Expect(os.RemoveAll(shareDir)).To(Succeed())
			Expect(os.RemoveAll(podsDir)).To(Succeed())
		})

		verifyObj := func(key string, domain *api.Domain, g Gomega) {
			obj, exists, err := informer.GetStore().GetByKey(key)
			g.Expect(err).ToNot(HaveOccurred())

			if domain != nil {
				g.Expect(exists).To(BeTrue())

				eventDomain := obj.(*api.Domain)
				eventDomain.Spec.XMLName = xml.Name{}
				g.Expect(equality.Semantic.DeepEqual(&domain.Spec, &eventDomain.Spec)).To(BeTrue())
			} else {
				g.Expect(exists).To(BeFalse())
			}
		}

		It("should list current domains.", func() {
			var list []*api.Domain

			list = append(list, api.NewMinimalDomain(domainName))
			domainManager.EXPECT().ListAllDomains().Return(list, nil)
			domainManager.EXPECT().GetGuestOSInfo().Return(&api.GuestOSInfo{})
			domainManager.EXPECT().InterfacesStatus().Return([]api.InterfaceStatus{})

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

			Expect(listResults).To(HaveLen(1))
		})

		It("should list current domains including inactive domains with ghost record", func() {
			var list []*api.Domain

			list = append(list, api.NewMinimalDomain(domainName))

			domainManager.EXPECT().ListAllDomains().Return(list, nil)
			domainManager.EXPECT().GetGuestOSInfo().Return(&api.GuestOSInfo{})
			domainManager.EXPECT().InterfacesStatus().Return([]api.InterfaceStatus{})

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
			Expect(listResults).To(HaveLen(2))
		})
		It("should detect active domains at startup.", func() {
			var list []*api.Domain

			domain := api.NewMinimalDomain(domainName)
			list = append(list, domain)

			domainManager.EXPECT().ListAllDomains().Return(list, nil)
			domainManager.EXPECT().GetGuestOSInfo().Return(&api.GuestOSInfo{})
			domainManager.EXPECT().InterfacesStatus().Return([]api.InterfaceStatus{})

			runCMDServer(wg, socketPath, domainManager, stopChan, nil)

			// ensure we can connect to the server first.
			client, err := cmdclient.NewClient(socketPath)
			Expect(err).ToNot(HaveOccurred())
			client.Close()

			runInformer(wg, stopChan, informer)
			cache.WaitForCacheSync(stopChan, informer.HasSynced)

			key := fmt.Sprintf("%s/%s", domainNamespace, domainName)
			verifyObj(key, domain, Default)
		})

		It("should resync active domains after resync period.", func() {
			domain := api.NewMinimalDomain(domainName)
			domainManager.EXPECT().ListAllDomains().Return([]*api.Domain{domain}, nil)
			domainManager.EXPECT().GetGuestOSInfo().Return(&api.GuestOSInfo{})
			domainManager.EXPECT().InterfacesStatus().Return([]api.InterfaceStatus{})
			// now prove if we make a change, like adding a label, that the resync
			// will pick that change up automatically
			newDomain := domain.DeepCopy()
			newDomain.ObjectMeta.Labels = make(map[string]string)
			newDomain.ObjectMeta.Labels["some-label"] = "some-value"
			domainManager.EXPECT().ListAllDomains().Return([]*api.Domain{newDomain}, nil)
			domainManager.EXPECT().GetGuestOSInfo().Return(nil)
			domainManager.EXPECT().InterfacesStatus().Return(nil)

			runCMDServer(wg, socketPath, domainManager, stopChan, nil)

			// ensure we can connect to the server first.
			client, err := cmdclient.NewClient(socketPath)
			Expect(err).ToNot(HaveOccurred())
			client.Close()

			runInformer(wg, stopChan, informer)
			cache.WaitForCacheSync(stopChan, informer.HasSynced)

			key := fmt.Sprintf("%s/%s", domainNamespace, domainName)
			verifyObj(key, domain, Default)

			time.Sleep(time.Duration(resyncPeriod+1) * time.Second)

			obj, exists, err := informer.GetStore().GetByKey(key)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())

			eventDomain := obj.(*api.Domain)
			val, ok := eventDomain.ObjectMeta.Labels["some-label"]

			Expect(ok).To(BeTrue())
			Expect(val).To(Equal("some-value"))
		})

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

			timeout := time.After(5 * time.Second)
			select {
			case event := <-d.eventChan:
				Expect(event.Type).To(Equal(watch.Modified))
				Expect(event.Object.(*api.Domain).ObjectMeta.DeletionTimestamp).ToNot(BeNil())
			case <-timeout:
				Fail("timeout")
			}

		})

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
		})

		It("should not return errors when encountering disconnected clients at startup.", func() {
			var list []*api.Domain

			domain := api.NewMinimalDomain(domainName)
			list = append(list, domain)

			domainManager.EXPECT().ListAllDomains().Return(list, nil)
			domainManager.EXPECT().GetGuestOSInfo().Return(&api.GuestOSInfo{})
			domainManager.EXPECT().InterfacesStatus().Return([]api.InterfaceStatus{})

			runCMDServer(wg, socketPath, domainManager, stopChan, nil)
			// ensure we can connect to the server first.
			client, err := cmdclient.NewClient(socketPath)
			Expect(err).ToNot(HaveOccurred())
			client.Close()

			runInformer(wg, stopChan, informer)
			cache.WaitForCacheSync(stopChan, informer.HasSynced)

			key := fmt.Sprintf("%s/%s", domainNamespace, domainName)
			verifyObj(key, domain, Default)
		})

		It("should watch for domain events.", func() {
			domain := api.NewMinimalDomain("test")

			runInformer(wg, stopChan, informer)
			cache.WaitForCacheSync(stopChan, informer.HasSynced)

			client := notifyclient.NewNotifier(shareDir)

			// verify add
			err := client.SendDomainEvent(watch.Event{Type: watch.Added, Object: domain})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func(g Gomega) { verifyObj("default/test", domain, g) }, time.Second, 200*time.Millisecond).Should(Succeed())

			// verify modify
			domain.Spec.UUID = "fakeuuid"
			err = client.SendDomainEvent(watch.Event{Type: watch.Modified, Object: domain})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func(g Gomega) { verifyObj("default/test", domain, g) }, time.Second, 200*time.Millisecond).Should(Succeed())

			// verify modify
			err = client.SendDomainEvent(watch.Event{Type: watch.Deleted, Object: domain})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func(g Gomega) { verifyObj("default/test", nil, g) }, time.Second, 200*time.Millisecond).Should(Succeed())
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
