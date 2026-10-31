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
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	authv1 "k8s.io/api/authentication/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	cmdauth "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/auth"
	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const socketDialTimeoutSeconds = 5

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
	authClient          kubernetes.Interface
	vmiStore            cache.Store
	// useVMIDiscovery is true when the LauncherSocketAuthentication
	// feature gate is enabled. Socket discovery uses VMI store
	// iteration via ActivePods instead of ghost records.
	useVMIDiscovery bool
}

func newDomainWatcher(ctx context.Context, runNotifyServer runServerFunc, watchdogTimeout int, resyncPeriod time.Duration, recorder record.EventRecorder, consecutiveFails *int, authClient kubernetes.Interface, vmiStore cache.Store) *domainWatcher {
	ctx, cancel := context.WithCancel(ctx)
	d := &domainWatcher{
		recorder:            recorder,
		unresponsiveSockets: make(map[string]int64),
		consecutiveFails:    consecutiveFails,
		result:              make(chan watch.Event, 100),
		cancel:              cancel,
		authClient:          authClient,
		vmiStore:            vmiStore,
		useVMIDiscovery:     authClient != nil,
	}
	d.wg.Add(1)
	go d.worker(ctx, runNotifyServer, resyncPeriod, watchdogTimeout)
	return d
}

func (d *domainWatcher) worker(ctx context.Context, runServer runServerFunc, resyncPeriod time.Duration, watchdogTimeoutSeconds int) {
	defer d.wg.Done()
	defer close(d.result)

	resyncTicker := time.NewTicker(resyncPeriod)
	defer resyncTicker.Stop()

	// Divide the watchdogTimeout by 3 for our ticker.
	// This ensures we always have at least 2 response failures
	// in a row before we mark the socket as unavailable (which results in shutdown of VMI)
	expiredWatchdogTicker := time.NewTicker(time.Duration((watchdogTimeoutSeconds/3)+1) * time.Second)
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
			d.handleStaleSocketConnections(ctx, watchdogTimeoutSeconds)
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
	log.Log.Infof("resyncing virt-launcher domains")

	var entries []socketEntry

	if d.useVMIDiscovery {
		entries = d.socketsFromVMIStore()
	} else {
		for _, rec := range GhostRecordGlobalStore.list() {
			entries = append(entries, socketEntry{
				path:       rec.SocketFile,
				activePods: d.activePodsForSocket(rec.SocketFile),
			})
		}
	}

	for _, entry := range entries {
		if err := AuthenticateSocket(ctx, d.authClient, entry.path, entry.activePods); err != nil {
			log.Log.Reason(err).Errorf("launcher authentication failed for socket %s during resync, skipping", entry.path)
			continue
		}

		client, err := cmdclient.NewClient(entry.path)
		if err != nil {
			log.Log.Reason(err).Error("failed to connect to cmd client socket during resync")
			continue
		}
		defer client.Close()

		domain, exists, err := client.GetDomain()
		if err != nil {
			log.Log.Reason(err).Errorf("unable to retrieve domain at socket %s during resync", entry.path)
			continue
		} else if !exists {
			continue
		}

		if !d.send(ctx, watch.Event{Type: watch.Modified, Object: domain}) {
			return
		}
	}
}

// socketsFromVMIStore iterates all VMIs in the VMI store and derives
// socket paths from ActivePods. This replaces ghost record iteration
// when the LauncherSocketAuthentication feature gate is enabled.
func (d *domainWatcher) socketsFromVMIStore() []socketEntry {
	if d.vmiStore == nil {
		return nil
	}
	var entries []socketEntry
	for _, obj := range d.vmiStore.List() {
		vmi, ok := obj.(*v1.VirtualMachineInstance)
		if !ok {
			continue
		}
		for podUID := range vmi.Status.ActivePods {
			socket := cmdclient.SocketFilePathOnHost(string(podUID))
			if _, err := os.Stat(socket); err != nil {
				continue
			}
			entries = append(entries, socketEntry{
				path:       socket,
				activePods: vmi.Status.ActivePods,
			})
		}
	}
	return entries
}

type socketEntry struct {
	path       string
	activePods map[types.UID]string
}

func (d *domainWatcher) activePodsForSocket(socketPath string) map[types.UID]string {
	if d.vmiStore == nil {
		return nil
	}
	record, found := GhostRecordGlobalStore.findBySocket(socketPath)
	if !found {
		return nil
	}
	key := record.Namespace + "/" + record.Name
	obj, exists, err := d.vmiStore.GetByKey(key)
	if err != nil || !exists {
		return nil
	}
	vmi, ok := obj.(*v1.VirtualMachineInstance)
	if !ok {
		return nil
	}
	return vmi.Status.ActivePods
}

// AuthenticateSocket connects to a virt-launcher socket, requests its
// projected ServiceAccount token via the CmdAuth gRPC service, and
// validates it through the Kubernetes TokenReview API. If k8sClient is
// nil, authentication is skipped (returns nil).
//
// activePods is the VMI.Status.ActivePods map; when non-nil, the
// token's pod UID must appear in this map to prove the pod was created
// by virt-controller as a legitimate launcher for this VMI.
func AuthenticateSocket(ctx context.Context, k8sClient kubernetes.Interface, socketPath string, activePods map[types.UID]string) error {
	if k8sClient == nil {
		return nil
	}

	conn, err := grpcutil.DialSocket(socketPath)
	if err != nil {
		return fmt.Errorf("dialing socket for auth: %w", err)
	}
	defer conn.Close()

	authClient := cmdauth.NewCmdAuthClient(conn)
	resp, err := authClient.Authenticate(ctx, &cmdauth.AuthRequest{})
	if err != nil {
		return fmt.Errorf("calling Authenticate RPC: %w", err)
	}

	review := &authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token:     resp.GetToken(),
			Audiences: []string{cmdauth.Audience},
		},
	}
	result, err := k8sClient.AuthenticationV1().TokenReviews().Create(ctx, review, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("TokenReview API call: %w", err)
	}
	if !result.Status.Authenticated {
		return fmt.Errorf("token not authenticated: %s", result.Status.Error)
	}

	tokenPodUID, err := podUIDFromTokenReview(result)
	if err != nil {
		return fmt.Errorf("token missing pod identity: %w", err)
	}
	socketPodUID, err := podUIDFromSocketPath(socketPath)
	if err != nil {
		return fmt.Errorf("cannot extract pod UID from socket path %q: %w", socketPath, err)
	}
	if tokenPodUID != socketPodUID {
		return fmt.Errorf("pod UID mismatch: token belongs to %q but socket belongs to %q", tokenPodUID, socketPodUID)
	}

	if activePods != nil {
		if _, ok := activePods[types.UID(tokenPodUID)]; !ok {
			return fmt.Errorf("pod %q is not in VMI ActivePods: not a legitimate virt-launcher", tokenPodUID)
		}
	}

	log.Log.Infof("authenticated launcher at %s: user=%s pod=%s", socketPath, result.Status.User.Username, tokenPodUID)
	return nil
}

const tokenReviewPodUIDKey = "authentication.kubernetes.io/pod-uid"

func podUIDFromTokenReview(result *authv1.TokenReview) (string, error) {
	values, ok := result.Status.User.Extra[tokenReviewPodUIDKey]
	if !ok || len(values) == 0 {
		return "", fmt.Errorf("%s not present in TokenReview response", tokenReviewPodUIDKey)
	}
	uid := string(values[0])
	if uid == "" {
		return "", fmt.Errorf("%s is empty in TokenReview response", tokenReviewPodUIDKey)
	}
	return uid, nil
}

// podUIDFromSocketPath extracts the pod UID from a launcher socket
// path of the form .../<podUID>/volumes/.../launcher-sock.
func podUIDFromSocketPath(socketPath string) (string, error) {
	idx := strings.Index(socketPath, "/volumes/")
	if idx <= 0 {
		return "", fmt.Errorf("path does not contain /volumes/ segment")
	}
	return filepath.Base(socketPath[:idx]), nil
}

func (d *domainWatcher) handleStaleSocketConnections(ctx context.Context, watchdogTimeoutSeconds int) error {
	var unresponsive []string
	var socketFiles []string

	if d.useVMIDiscovery {
		for _, entry := range d.socketsFromVMIStore() {
			socketFiles = append(socketFiles, entry.path)
		}
	} else {
		socketFiles = listSockets(GhostRecordGlobalStore.list())
	}

	for _, socket := range socketFiles {
		sock, err := net.DialTimeout("unix", socket, time.Duration(socketDialTimeoutSeconds)*time.Second)
		if err == nil {
			sock.Close()
			continue
		}
		unresponsive = append(unresponsive, socket)
	}

	now := time.Now().UTC().Unix()

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

		if diff > int64(watchdogTimeoutSeconds) {
			domain := d.domainForStaleSocket(key)
			if domain == nil {
				continue
			}
			deletionTime := metav1.Now()
			domain.ObjectMeta.DeletionTimestamp = &deletionTime
			log.Log.Object(domain).Warningf("detected unresponsive virt-launcher command socket (%s) for domain", key)
			if !d.send(ctx, watch.Event{Type: watch.Modified, Object: domain}) {
				return ctx.Err()
			}

			if err := cmdclient.MarkSocketUnresponsive(key); err != nil {
				log.Log.Reason(err).Errorf("Unable to mark vmi as unresponsive socket %s", key)
			}
		}
	}

	return nil
}

// domainForStaleSocket builds a minimal domain object for a stale socket.
// When VMI-based discovery is active, it looks up the VMI by matching the
// socket's pod UID against ActivePods. Otherwise, it uses the ghost record.
func (d *domainWatcher) domainForStaleSocket(socketPath string) *api.Domain {
	if d.useVMIDiscovery {
		return d.domainFromVMIStoreBySocket(socketPath)
	}
	record, exists := GhostRecordGlobalStore.findBySocket(socketPath)
	if !exists {
		return nil
	}
	return newDomainFromGhostRecord(record, api.DomainStatus{})
}

// domainFromVMIStoreBySocket extracts the pod UID from the socket path
// and finds the VMI whose ActivePods contains that UID.
func (d *domainWatcher) domainFromVMIStoreBySocket(socketPath string) *api.Domain {
	podUID, err := podUIDFromSocketPath(socketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("cannot extract pod UID from socket path %s", socketPath)
		return nil
	}
	if d.vmiStore == nil {
		return nil
	}
	for _, obj := range d.vmiStore.List() {
		vmi, ok := obj.(*v1.VirtualMachineInstance)
		if !ok {
			continue
		}
		if _, found := vmi.Status.ActivePods[types.UID(podUID)]; found {
			domain := api.NewMinimalDomainWithNS(vmi.Namespace, vmi.Name)
			domain.ObjectMeta.UID = vmi.UID
			domain.Spec.Metadata.KubeVirt.UID = vmi.UID
			return domain
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
