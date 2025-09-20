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
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"k8s.io/client-go/tools/record"

	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/log"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/util"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	notifyserver "kubevirt.io/kubevirt/pkg/virt-handler/notify-server"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/watchdog"
)

const socketDialTimeout = 5

func newListWatchFromNotify(virtShareDir string, watchdogTimeout int, recorder record.EventRecorder, vmiStore cache.Store, resyncPeriod time.Duration) cache.ListerWatcher {
	d := &DomainWatcher{
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

type DomainWatcher struct {
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

type ghostRecord struct {
	Name       string    `json:"name"`
	Namespace  string    `json:"namespace"`
	SocketFile string    `json:"socketFile"`
	UID        types.UID `json:"uid"`
}

var ghostRecordGlobalCache map[string]ghostRecord
var ghostRecordGlobalMutex sync.Mutex
var ghostRecordDir string

func InitializeGhostRecordCache(directoryPath string) error {
	ghostRecordGlobalMutex.Lock()
	defer ghostRecordGlobalMutex.Unlock()

	ghostRecordGlobalCache = make(map[string]ghostRecord)
	ghostRecordDir = directoryPath
	err := util.MkdirAllWithNosec(ghostRecordDir)
	if err != nil {
		return err
	}

	files, err := os.ReadDir(ghostRecordDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		recordPath := filepath.Join(ghostRecordDir, file.Name())
		// #nosec no risk for path injection. Used only for testing and using static location
		fileBytes, err := os.ReadFile(recordPath)
		if err != nil {
			log.Log.Reason(err).Errorf("Unable to read ghost record file at path %s", recordPath)
			continue
		}

		ghostRecord := ghostRecord{}
		err = json.Unmarshal(fileBytes, &ghostRecord)
		if err != nil {
			log.Log.Reason(err).Errorf("Unable to unmarshal json contents of ghost record file at path %s", recordPath)
			continue
		}

		key := ghostRecord.Namespace + "/" + ghostRecord.Name
		ghostRecordGlobalCache[key] = ghostRecord
		log.Log.Infof("Added ghost record for key %s", key)
	}

	return nil
}

func LastKnownUIDFromGhostRecordCache(key string) types.UID {
	ghostRecordGlobalMutex.Lock()
	defer ghostRecordGlobalMutex.Unlock()

	record, ok := ghostRecordGlobalCache[key]
	if !ok {
		return ""
	}

	return record.UID
}

func getGhostRecords() []ghostRecord {
	ghostRecordGlobalMutex.Lock()
	defer ghostRecordGlobalMutex.Unlock()

	var records []ghostRecord

	for _, record := range ghostRecordGlobalCache {
		records = append(records, record)
	}

	return records
}

func findGhostRecordBySocket(socketFile string) (ghostRecord, bool) {
	ghostRecordGlobalMutex.Lock()
	defer ghostRecordGlobalMutex.Unlock()

	for _, record := range ghostRecordGlobalCache {
		if record.SocketFile == socketFile {
			return record, true
		}
	}

	return ghostRecord{}, false
}

func HasGhostRecord(namespace string, name string) bool {
	ghostRecordGlobalMutex.Lock()
	defer ghostRecordGlobalMutex.Unlock()

	key := namespace + "/" + name
	_, ok := ghostRecordGlobalCache[key]

	return ok
}

func AddGhostRecord(namespace string, name string, socketFile string, uid types.UID) (err error) {
	ghostRecordGlobalMutex.Lock()
	defer ghostRecordGlobalMutex.Unlock()
	if name == "" {
		return fmt.Errorf("can not add ghost record when 'name' is not provided")
	} else if namespace == "" {
		return fmt.Errorf("can not add ghost record when 'namespace' is not provided")
	} else if string(uid) == "" {
		return fmt.Errorf("Unable to add ghost record with empty UID")
	} else if socketFile == "" {
		return fmt.Errorf("Unable to add ghost record without a socketFile")
	}

	key := namespace + "/" + name
	recordPath := filepath.Join(ghostRecordDir, string(uid))

	record, ok := ghostRecordGlobalCache[key]
	if !ok {
		// record doesn't exist, so add new one.
		record := ghostRecord{
			Name:       name,
			Namespace:  namespace,
			SocketFile: socketFile,
			UID:        uid,
		}

		fileBytes, err := json.Marshal(&record)
		if err != nil {
			return err
		}
		f, err := os.Create(recordPath)
		if err != nil {
			return err
		}
		defer util.CloseIOAndCheckErr(f, &err)

		_, err = f.Write(fileBytes)
		if err != nil {
			return err
		}
		ghostRecordGlobalCache[key] = record
	}

	// This protects us from stomping on a previous ghost record
	// that was not cleaned up properly. A ghost record that was
	// not deleted indicates that the VMI shutdown process did not
	// properly handle cleanup of local data.
	if ok && record.UID != uid {
		return fmt.Errorf("can not add ghost record when entry already exists with differing UID")
	}

	if ok && record.SocketFile != socketFile {
		return fmt.Errorf("can not add ghost record when entry already exists with differing socket file location")
	}

	return nil
}

func DeleteGhostRecord(namespace string, name string) error {
	ghostRecordGlobalMutex.Lock()
	defer ghostRecordGlobalMutex.Unlock()
	key := namespace + "/" + name
	record, ok := ghostRecordGlobalCache[key]
	if !ok {
		// already deleted
		return nil
	}

	if string(record.UID) == "" {
		return fmt.Errorf("Unable to remove ghost record with empty UID")
	}

	recordPath := filepath.Join(ghostRecordDir, string(record.UID))
	err := os.RemoveAll(recordPath)
	if err != nil {
		return nil
	}

	delete(ghostRecordGlobalCache, key)

	return nil
}

func (d *DomainWatcher) startBackground() error {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.backgroundWatcherStarted == true {
		return nil
	}

	d.stopChan = make(chan struct{}, 1)
	d.eventChan = make(chan watch.Event, 100)

	d.wg.Add(1)
	go func() {
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
				d.handleStaleWatchdogFiles()
				d.handleStaleSocketConnections()
			case err := <-srvErr:
				if err != nil {
					log.Log.Reason(err).Errorf("Unexpected err encountered with Domain Notify aggregation server")
				}

				// server exitted so this goroutine is done.
				return
			}
		}
	}()

	d.backgroundWatcherStarted = true
	return nil
}

// TODO remove watchdog file usage eventually and only rely on detecting stale socket connections
// for now we have to keep watchdog files around for backwards compatibility with old VMIs
func (d *DomainWatcher) handleStaleWatchdogFiles() error {
	domains, err := watchdog.GetExpiredDomains(d.watchdogTimeout, d.virtShareDir)
	if err != nil {
		log.Log.Reason(err).Error("failed to detect expired watchdog files in domain informer")
		return err
	}

	for _, domain := range domains {
		log.Log.Object(domain).Warning("detected expired watchdog for domain")
		now := k8sv1.Now()
		domain.ObjectMeta.DeletionTimestamp = &now
		d.eventChan <- watch.Event{Type: watch.Modified, Object: domain}
	}
	return nil
}

func (d *DomainWatcher) handleResync() {

	socketFiles, err := listSockets(getGhostRecords())
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

func (d *DomainWatcher) handleStaleSocketConnections() error {
	var unresponsive []string

	socketFiles, err := listSockets(getGhostRecords())
	if err != nil {
		log.Log.Reason(err).Error("failed to list sockets")
		return err
	}

	for _, socket := range socketFiles {
		if !cmdclient.SocketMonitoringEnabled(socket) {
			// don't process legacy sockets here. They still use the
			// old watchdog file method
			continue
		}

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

			record, exists := findGhostRecordBySocket(key)

			if !exists {
				// ignore if info file doesn't exist
				// this is possible with legacy VMIs that haven't
				// been updated. The watchdog file will catch these.
			} else {
				domain := api.NewMinimalDomainWithNS(record.Namespace, record.Name)
				domain.ObjectMeta.UID = record.UID
				domain.Spec.Metadata.KubeVirt.UID = record.UID
				now := k8sv1.Now()
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

func (d *DomainWatcher) listAllKnownDomains() ([]*api.Domain, error) {
	var domains []*api.Domain

	socketFiles, err := listSockets(getGhostRecords())
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
			record, recordExists := findGhostRecordBySocket(socketFile)
			if recordExists {
				domain := api.NewMinimalDomainWithNS(record.Namespace, record.Name)
				domain.ObjectMeta.UID = record.UID
				now := k8sv1.Now()
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
		if exists == true {
			domains = append(domains, domain)
		}
	}
	return domains, nil
}

func (d *DomainWatcher) List(_ k8sv1.ListOptions) (runtime.Object, error) {

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

func (d *DomainWatcher) Watch(_ k8sv1.ListOptions) (watch.Interface, error) {
	return d, nil
}

func (d *DomainWatcher) Stop() {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.backgroundWatcherStarted == false {
		return
	}
	close(d.stopChan)
	d.wg.Wait()
	d.backgroundWatcherStarted = false
	close(d.eventChan)
}

func (d *DomainWatcher) ResultChan() <-chan watch.Event {
	return d.eventChan
}

func NewSharedInformer(virtShareDir string, watchdogTimeout int, recorder record.EventRecorder, vmiStore cache.Store, resyncPeriod time.Duration) (cache.SharedInformer, error) {
	lw := newListWatchFromNotify(virtShareDir, watchdogTimeout, recorder, vmiStore, resyncPeriod)
	informer := cache.NewSharedInformer(lw, &api.Domain{}, 0)
	return informer, nil
}

func listSockets(ghostRecords []ghostRecord) ([]string, error) {
	var sockets []string

	for _, record := range ghostRecords {
		sockets = append(sockets, record.SocketFile)
	}

	return sockets, nil
}
