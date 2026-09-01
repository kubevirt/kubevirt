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
	"context"
	"fmt"
	"net"
	"os"
	"slices"
	"sync"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/client-go/log"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const socketDialTimeout = 5

type runServerFunc func(ctx context.Context, c chan watch.Event) error

var (
	notifyServerMaxConsecutiveFails = 10
	notifyServerHealthyRunTime      = 1 * time.Minute
)

type domainWatcher struct {
	wg                  sync.WaitGroup
	cancel              context.CancelFunc
	result              chan watch.Event
	recorder            record.EventRecorder
	consecutiveFails    *int
	unresponsiveSockets map[string]int64
}

func newDomainWatcher(ctx context.Context, runNotifyServer runServerFunc, watchdogTimeout int, resyncPeriod time.Duration, recorder record.EventRecorder, consecutiveFails *int) *domainWatcher {
	ctx, cancel := context.WithCancel(ctx)
	d := &domainWatcher{
		recorder:            recorder,
		unresponsiveSockets: make(map[string]int64),
		consecutiveFails:    consecutiveFails,
		result:              make(chan watch.Event, 100),
		cancel:              cancel,
	}
	d.wg.Add(1)
	go d.worker(ctx, runNotifyServer, resyncPeriod, watchdogTimeout)
	return d
}

func (d *domainWatcher) worker(ctx context.Context, runServer runServerFunc, resyncPeriod time.Duration, watchdogTimeout int) {
	defer d.wg.Done()
	defer close(d.result)

	resyncTicker := time.NewTicker(resyncPeriod)
	defer resyncTicker.Stop()

	// Divide the watchdogTimeout by 3 for our ticker.
	// This ensures we always have at least 2 response failures
	// in a row before we mark the socket as unavailable (which results in shutdown of VMI)
	expiredWatchdogTicker := time.NewTicker(time.Duration((watchdogTimeout/3)+1) * time.Second)
	defer expiredWatchdogTicker.Stop()

	startedAt := time.Now()
	srvErr := make(chan error)
	go func() {
		defer close(srvErr)
		srvErr <- runServer(ctx, d.result)
	}()

	for {
		select {
		case <-resyncTicker.C:
			d.handleResync(ctx)
		case <-expiredWatchdogTicker.C:
			d.handleStaleSocketConnections(ctx, watchdogTimeout)
		case err := <-srvErr:
			if err != nil {
				log.Log.Reason(err).Errorf("Domain notify server exited unexpectedly")
				d.panicOnConsecutiveFailures(err, startedAt)
				d.send(ctx, watch.Event{
					Type: watch.Error,
					Object: &metav1.Status{
						Status:  metav1.StatusFailure,
						Message: fmt.Sprintf("domain notify server error: %v", err),
					},
				})
			}
			return
		}
	}
}

// send delivers event on d.result, but gives up once ctx is done. Without
// this, a worker shutting down after the informer has already stopped
// reading ResultChan() would block on this send forever, hanging Stop().
func (d *domainWatcher) send(ctx context.Context, event watch.Event) bool {
	select {
	case d.result <- event:
		return true
	case <-ctx.Done():
		return false
	}
}

func (d *domainWatcher) panicOnConsecutiveFailures(err error, startedAt time.Time) {
	if time.Since(startedAt) >= notifyServerHealthyRunTime {
		*d.consecutiveFails = 0
	}
	*d.consecutiveFails++

	d.recordNotifyServerFailureEvent(err)

	if *d.consecutiveFails >= notifyServerMaxConsecutiveFails {
		log.Log.Reason(err).Criticalf("Domain notify server reached max consecutive failures (%d)",
			notifyServerMaxConsecutiveFails)
		panic(fmt.Sprintf("domain notify server reached max consecutive failures (%d): %v",
			notifyServerMaxConsecutiveFails, err))
	}
}

func (d *domainWatcher) recordNotifyServerFailureEvent(err error) {
	if d.recorder == nil {
		return
	}
	hostname, _ := os.Hostname()
	node := &k8sv1.Node{ObjectMeta: metav1.ObjectMeta{Name: hostname}}
	d.recorder.Eventf(node, k8sv1.EventTypeWarning, "NotifyServerFailure",
		"Domain notify server exited unexpectedly: %v", err)
}

func (d *domainWatcher) handleResync(ctx context.Context) {
	socketFiles := listSockets(GhostRecordGlobalStore.list())

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

		if !d.send(ctx, watch.Event{Type: watch.Modified, Object: domain}) {
			return
		}
	}
}

func (d *domainWatcher) handleStaleSocketConnections(ctx context.Context, watchdogTimeout int) error {
	var unresponsive []string

	socketFiles := listSockets(GhostRecordGlobalStore.list())

	for _, socket := range socketFiles {
		sock, err := net.DialTimeout("unix", socket, time.Duration(socketDialTimeout)*time.Second)
		if err == nil {
			// socket is alive still
			sock.Close()
			continue
		}
		unresponsive = append(unresponsive, socket)
	}

	now := time.Now().UTC().Unix()

	// Add new unresponsive sockets
	for _, socket := range unresponsive {
		_, ok := d.unresponsiveSockets[socket]
		if !ok {
			d.unresponsiveSockets[socket] = now
		}
	}

	for key, timeStamp := range d.unresponsiveSockets {
		if !slices.Contains(unresponsive, key) {
			delete(d.unresponsiveSockets, key)
			continue
		}

		diff := now - timeStamp

		if diff > int64(watchdogTimeout) {

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
				if !d.send(ctx, watch.Event{Type: watch.Modified, Object: domain}) {
					return ctx.Err()
				}

				err := cmdclient.MarkSocketUnresponsive(key)
				if err != nil {
					log.Log.Reason(err).Errorf("Unable to mark vmi as unresponsive socket %s", key)
				}
			}
		}
	}

	return nil
}

func (d *domainWatcher) Stop() {
	d.cancel()
	d.wg.Wait()
}

func (d *domainWatcher) ResultChan() <-chan watch.Event {
	return d.result
}

func listSockets(ghostRecords []ghostRecord) []string {
	var sockets []string

	for _, record := range ghostRecords {
		sockets = append(sockets, record.SocketFile)
	}

	return sockets
}
