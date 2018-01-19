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
	"fmt"
	"sync"

	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cmd-server/client"
	notifyserver "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/notify-server"
)

func newListWatchFromNotify(virtShareDir string) cache.ListerWatcher {
	d := &DomainWatcher{
		backgroundWatcherStarted: false,
		virtShareDir:             virtShareDir,
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

		srvErr := make(chan error)
		go func() {
			defer close(srvErr)
			err := notifyserver.RunServer(d.virtShareDir, d.stopChan, d.eventChan)
			srvErr <- err
		}()

		// wait for server to exit.
		select {
		case err := <-srvErr:
			if err != nil {
				log.Log.Reason(err).Errorf("Unexpeted err encountered with Domain Notify aggregation server")
			}
		}
	}()

	d.backgroundWatcherStarted = true
	return nil
}

func (d *DomainWatcher) List(options k8sv1.ListOptions) (runtime.Object, error) {

	log.Log.V(3).Info("Synchronizing domains")
	err := d.startBackground()
	if err != nil {
		return nil, err
	}

	list := api.DomainList{
		Items: []api.Domain{},
	}

	socketFiles, err := cmdclient.ListAllSockets(d.virtShareDir)
	if err != nil {
		return nil, err
	}
	for _, socketFile := range socketFiles {
		client, err := cmdclient.GetClient(socketFile)
		if err != nil {
			// Ignore failure to connect to client.
			// These are all local connections via unix socket.
			// A failure to connect means there's nothing on the other
			// end listening.
			continue
		}
		defer client.Close()

		domains, err := client.ListDomains()
		if err != nil {
			// Failure to get domain list means that client
			// was unable to contact libvirt. As soon as the connection
			// is restored on the client's end, a domain notification will
			// be sent.
			continue
		}
		for _, domain := range domains {
			list.Items = append(list.Items, *domain)
		}
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
	close(d.eventChan)
	close(d.stopChan)
	d.wg.Wait()
	d.backgroundWatcherStarted = false
}

func (d *DomainWatcher) ResultChan() <-chan watch.Event {
	return d.eventChan
}

// VMNamespaceKeyFunc constructs the domain name with a namespace prefix i.g.
// namespace_name.
func VMNamespaceKeyFunc(vm *v1.VirtualMachine) string {
	domName := fmt.Sprintf("%s_%s", vm.GetObjectMeta().GetNamespace(), vm.GetObjectMeta().GetName())
	return domName
}

func NewSharedInformer(virtShareDir string) (cache.SharedInformer, error) {
	lw := newListWatchFromNotify(virtShareDir)
	informer := cache.NewSharedInformer(lw, &api.Domain{}, 0)
	return informer, nil
}
