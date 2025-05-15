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
package cache

import (
	"net"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/client-go/log"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	notifyserver "kubevirt.io/kubevirt/pkg/virt-handler/notify-server"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const socketDialTimeout = 5

type domainWatcher struct {
	lock                     sync.Mutex
	wg                       sync.WaitGroup
	stopChan                 chan struct{}
	eventChan                chan watch.Event
	backgroundWatcherStarted bool
	virtShareDir             string
	watchdogTimeout          int
	recorder                 record.EventRecorder
	vmiStore                 cache.Store
	resyncPeriod             time.Duration

	watchDogLock        sync.Mutex
	unresponsiveSockets map[string]int64
}

func newListWatchFromNotify(virtShareDir string, watchdogTimeout int, recorder record.EventRecorder, vmiStore cache.Store, resyncPeriod time.Duration) cache.ListerWatcher {
	d := &domainWatcher{
		backgroundWatcherStarted: false,
		virtShareDir:             virtShareDir,
		watchdogTimeout:          watchdogTimeout,
		recorder:                 recorder,
		vmiStore:                 vmiStore,
		unresponsiveSockets:      make(map[string]int64),
		resyncPeriod:             resyncPeriod,
	}

	return d
}

func (d *domainWatcher) worker() {
	defer d.wg.Done()

	resyncTicker := time.NewTicker(d.resyncPeriod)
	resyncTickerChan := resyncTicker.C
	defer resyncTicker.Stop()

	// Divide the watchdogTimeout by 3 for our ticker.
	// This ensures we always have at least 2 response failures
	// in a row before we mark the socket as unavailable (which results in shutdown of VMI)
	expiredWatchdogTicker := time.NewTicker(time.Duration((d.watchdogTimeout/3)+1) * time.Second)
	defer expiredWatchdogTicker.Stop()

	expiredWatchdogTickerChan := expiredWatchdogTicker.C

	srvErr := make(chan error)
	go func() {
		defer close(srvErr)
		err := notifyserver.RunServer(d.virtShareDir, d.stopChan, d.eventChan, d.recorder, d.vmiStore)
		srvErr <- err
	}()

	for {
		select {
		case <-resyncTickerChan:
			d.handleResync()
		case <-expiredWatchdogTickerChan:
			d.handleStaleSocketConnections()
		case err := <-srvErr:
			if err != nil {
				log.Log.Reason(err).Errorf("Unexpected err encountered with Domain Notify aggregation server")
			}

			// server exitted so this goroutine is done.
			return
		}
	}
}

func (d *domainWatcher) startBackground() error {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.backgroundWatcherStarted {
		return nil
	}

	d.stopChan = make(chan struct{}, 1)
	d.eventChan = make(chan watch.Event, 100)

	d.wg.Add(1)
	go d.worker()

	d.backgroundWatcherStarted = true
	return nil
}

func (d *domainWatcher) handleResync() {
	socketFiles, err := listSockets(&GhostRecordGlobalStore)
	if err != nil {
		log.Log.Reason(err).Error("failed to list sockets")
		return
	}

	log.Log.Infof("resyncing virt-launcher domains")
	for _, socket := range socketFiles {
		client, err := cmdclient.NewClient(socket)
		if err != nil {
			log.Log.Reason(err).Error("failed to connect to cmd client socket during resync")
			// Ignore failure to connect to client.
			// These are all local connections via unix socket.
			// A failure to connect means there's nothing on the other
			// end listening.
			continue
		}
		defer client.Close()

		domain, exists, err := client.GetDomain()
		if err != nil {
			// this resync is best effort only.
			log.Log.Reason(err).Errorf("unable to retrieve domain at socket %s during resync", socket)
			continue
		} else if !exists {
			// nothing to sync if it doesn't exist
			continue
		}

		d.eventChan <- watch.Event{Type: watch.Modified, Object: domain}
	}
}

func (d *domainWatcher) handleStaleSocketConnections() error {
	var unresponsive []string

	socketFiles, err := listSockets(&GhostRecordGlobalStore)
	if err != nil {
		log.Log.Reason(err).Error("failed to list sockets")
		return err
	}

	for _, socket := range socketFiles {
		sock, err := net.DialTimeout("unix", socket, time.Duration(socketDialTimeout)*time.Second)
		if err == nil {
			// socket is alive still
			sock.Close()
			continue
		}
		unresponsive = append(unresponsive, socket)
	}

	d.watchDogLock.Lock()
	defer d.watchDogLock.Unlock()

	now := time.Now().UTC().Unix()

	// Add new unresponsive sockets
	for _, socket := range unresponsive {
		_, ok := d.unresponsiveSockets[socket]
		if !ok {
			d.unresponsiveSockets[socket] = now
		}
	}

	for key, timeStamp := range d.unresponsiveSockets {
		found := false
		for _, socket := range unresponsive {
			if socket == key {
				found = true
				break
			}
		}
		// reap old unresponsive sockets
		// remove from unresponsive list if not found unresponsive this iteration
		if !found {
			delete(d.unresponsiveSockets, key)
			break
		}

		diff := now - timeStamp

		if diff > int64(d.watchdogTimeout) {

			record, exists := GhostRecordGlobalStore.findBySocket(key)

			if !exists {
				// ignore if info file doesn't exist
				// this is possible with legacy VMIs that haven't
				// been updated. The watchdog file will catch these.
			} else {
				domain := api.NewMinimalDomainWithNS(record.Namespace, record.Name)
				domain.ObjectMeta.UID = record.UID
				domain.Spec.Metadata.KubeVirt.UID = record.UID
				now := metav1.Now()
				domain.ObjectMeta.DeletionTimestamp = &now
				log.Log.Object(domain).Warningf("detected unresponsive virt-launcher command socket (%s) for domain", key)
				d.eventChan <- watch.Event{Type: watch.Modified, Object: domain}

				err := cmdclient.MarkSocketUnresponsive(key)
				if err != nil {
					log.Log.Reason(err).Errorf("Unable to mark vmi as unresponsive socket %s", key)
				}
			}
		}
	}

	return nil
}

func (d *domainWatcher) listAllKnownDomains() ([]*api.Domain, error) {
	var domains []*api.Domain

	socketFiles, err := listSockets(&GhostRecordGlobalStore)
	if err != nil {
		return nil, err
	}
	for _, socketFile := range socketFiles {

		exists, err := diskutils.FileExists(socketFile)
		if err != nil {
			log.Log.Reason(err).Error("failed access cmd client socket")
			continue
		}

		if !exists {
			record, recordExists := GhostRecordGlobalStore.findBySocket(socketFile)
			if recordExists {
				domain := api.NewMinimalDomainWithNS(record.Namespace, record.Name)
				domain.ObjectMeta.UID = record.UID
				now := metav1.Now()
				domain.ObjectMeta.DeletionTimestamp = &now
				log.Log.Object(domain).Warning("detected stale domain from ghost record")
				domains = append(domains, domain)
			}
			continue
		}

		log.Log.V(3).Infof("List domains from sock %s", socketFile)
		client, err := cmdclient.NewClient(socketFile)
		if err != nil {
			log.Log.Reason(err).Error("failed to connect to cmd client socket")
			// Ignore failure to connect to client.
			// These are all local connections via unix socket.
			// A failure to connect means there's nothing on the other
			// end listening.
			continue
		}
		defer client.Close()

		domain, exists, err := client.GetDomain()
		if err != nil {
			log.Log.Reason(err).Error("failed to list domains on cmd client socket")
			// Failure to get domain list means that client
			// was unable to contact libvirt. As soon as the connection
			// is restored on the client's end, a domain notification will
			// be sent.
			continue
		}
		if exists {
			domains = append(domains, domain)
		}
	}
	return domains, nil
}

func (d *domainWatcher) List(_ metav1.ListOptions) (runtime.Object, error) {

	log.Log.V(3).Info("Synchronizing domains")
	err := d.startBackground()
	if err != nil {
		return nil, err
	}

	domains, err := d.listAllKnownDomains()
	if err != nil {
		return nil, err
	}

	list := api.DomainList{
		Items: []api.Domain{},
	}

	for _, domain := range domains {
		list.Items = append(list.Items, *domain)
	}
	return &list, nil
}

func (d *domainWatcher) Watch(_ metav1.ListOptions) (watch.Interface, error) {
	return d, nil
}

func (d *domainWatcher) Stop() {
	d.lock.Lock()
	defer d.lock.Unlock()

	if !d.backgroundWatcherStarted {
		return
	}
	close(d.stopChan)
	d.wg.Wait()
	d.backgroundWatcherStarted = false
	close(d.eventChan)
}

func (d *domainWatcher) ResultChan() <-chan watch.Event {
	return d.eventChan
}

func listSockets(grs *GhostRecordStore) ([]string, error) {
	var sockets []string

	knownSocketFiles, err := cmdclient.ListAllSockets()
	if err != nil {
		return sockets, err
	}

	ghostRecords := grs.list()

	sockets = append(sockets, knownSocketFiles...)

	for _, record := range ghostRecords {
		exists := false
		for _, socket := range knownSocketFiles {
			if record.SocketFile == socket {
				exists = true
				break
			}
		}
		if !exists {
			sockets = append(sockets, record.SocketFile)
		}
	}

	return sockets, nil
}
