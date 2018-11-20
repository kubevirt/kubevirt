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
	"sync"
	"time"

	"k8s.io/client-go/tools/record"

	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/log"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	notifyserver "kubevirt.io/kubevirt/pkg/virt-handler/notify-server"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/watchdog"
)

func newListWatchFromNotify(virtShareDir string, watchdogTimeout int, recorder record.EventRecorder, vmiStore cache.Store) cache.ListerWatcher {
	d := &DomainWatcher{
		backgroundWatcherStarted: false,
		virtShareDir:             virtShareDir,
		watchdogTimeout:          watchdogTimeout,
		recorder:                 recorder,
		vmiStore:                 vmiStore,
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

		expiredWatchdogTicker := time.NewTicker(time.Duration(d.watchdogTimeout) * time.Second).C
		srvErr := make(chan error)
		go func() {
			defer close(srvErr)
			err := notifyserver.RunServer(d.virtShareDir, d.stopChan, d.eventChan, d.recorder, d.vmiStore)
			srvErr <- err
		}()

		for {
			select {
			case <-expiredWatchdogTicker:
				d.handleStaleWatchdogFiles()
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

func (d *DomainWatcher) handleStaleWatchdogFiles() error {
	domains, err := watchdog.GetExpiredDomains(d.watchdogTimeout, d.virtShareDir)
	if err != nil {
		log.Log.Reason(err).Error("failed to detect expired watchdog files in domain informer")
		return err
	}

	for _, domain := range domains {
		d.eventChan <- watch.Event{Type: watch.Deleted, Object: domain}
	}
	return nil
}

func (d *DomainWatcher) listAllKnownDomains() ([]*api.Domain, error) {
	var domains []*api.Domain

	socketFiles, err := cmdclient.ListAllSockets(d.virtShareDir)
	if err != nil {
		return nil, err
	}
	for _, socketFile := range socketFiles {
		log.Log.V(3).Infof("List domains from sock %s", socketFile)
		client, err := cmdclient.GetClient(socketFile)
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

func (d *DomainWatcher) List(options k8sv1.ListOptions) (runtime.Object, error) {

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

func (d *DomainWatcher) Watch(options k8sv1.ListOptions) (watch.Interface, error) {
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

func NewSharedInformer(virtShareDir string, watchdogTimeout int, recorder record.EventRecorder, vmiStore cache.Store) (cache.SharedInformer, error) {
	lw := newListWatchFromNotify(virtShareDir, watchdogTimeout, recorder, vmiStore)
	informer := cache.NewSharedInformer(lw, &api.Domain{}, 0)
	return informer, nil
}
