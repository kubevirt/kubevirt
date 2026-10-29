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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/checkpoint"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	notifyserver "kubevirt.io/kubevirt/pkg/virt-handler/notify-server"
)

type IterableCheckpointManager interface {
	ListKeys() []string
	checkpoint.CheckpointManager
}

type iterableCheckpointManager struct {
	base string
	checkpoint.CheckpointManager
}

func (icp *iterableCheckpointManager) ListKeys() []string {
	entries, err := os.ReadDir(icp.base)
	if err != nil {
		return []string{}
	}

	var keys []string
	for _, entry := range entries {
		keys = append(keys, entry.Name())
	}
	return keys

}

func NewIterableCheckpointManager(base, tempPath string) IterableCheckpointManager {
	return &iterableCheckpointManager{
		base,
		checkpoint.NewSimpleCheckpointManager(base, tempPath),
	}
}

type ghostRecord struct {
	Name       string    `json:"name"`
	Namespace  string    `json:"namespace"`
	SocketFile string    `json:"socketFile"`
	UID        types.UID `json:"uid"`
}

type GhostRecordStore struct {
	cache             map[string]ghostRecord
	checkpointManager checkpoint.CheckpointManager
	sync.Mutex
}

func InitializeGhostRecordCache(iterableCPManager IterableCheckpointManager) *GhostRecordStore {

	store := &GhostRecordStore{
		cache:             make(map[string]ghostRecord),
		checkpointManager: iterableCPManager,
	}

	keys := iterableCPManager.ListKeys()
	for _, key := range keys {
		ghostRecord := ghostRecord{}
		if err := store.checkpointManager.Get(key, &ghostRecord); err != nil {
			log.Log.Reason(err).Errorf("Unable to read ghost record checkpoint, %s", key)
			continue
		}
		key := ghostRecord.Namespace + "/" + ghostRecord.Name
		store.cache[key] = ghostRecord
		log.Log.Infof("Added ghost record for key %s", key)
	}
	return store
}

func (store *GhostRecordStore) LastKnownUID(key string) types.UID {
	store.Lock()
	defer store.Unlock()

	record, ok := store.cache[key]
	if !ok {
		return ""
	}

	return record.UID
}

func (store *GhostRecordStore) list() []ghostRecord {
	store.Lock()
	defer store.Unlock()

	var records []ghostRecord

	for _, record := range store.cache {
		records = append(records, record)
	}

	return records
}

func (store *GhostRecordStore) findBySocket(socketFile string) (ghostRecord, bool) {
	store.Lock()
	defer store.Unlock()

	for _, record := range store.cache {
		if filepath.Clean(record.SocketFile) == socketFile {
			return record, true
		}
	}

	return ghostRecord{}, false
}

func (store *GhostRecordStore) Exists(namespace string, name string) bool {
	store.Lock()
	defer store.Unlock()

	key := namespace + "/" + name
	_, ok := store.cache[key]

	return ok
}

func (store *GhostRecordStore) Add(namespace string, name string, socketFile string, uid types.UID) (err error) {
	store.Lock()
	defer store.Unlock()
	if name == "" {
		return fmt.Errorf("can not add ghost record when 'name' is not provided")
	} else if namespace == "" {
		return fmt.Errorf("can not add ghost record when 'namespace' is not provided")
	} else if string(uid) == "" {
		return fmt.Errorf("unable to add ghost record with empty UID")
	} else if socketFile == "" {
		return fmt.Errorf("unable to add ghost record without a socketFile")
	}

	key := namespace + "/" + name
	record, ok := store.cache[key]
	if !ok {
		// record doesn't exist, so add new one.
		record := ghostRecord{
			Name:       name,
			Namespace:  namespace,
			SocketFile: socketFile,
			UID:        uid,
		}
		if err := store.checkpointManager.Store(string(uid), &record); err != nil {
			return fmt.Errorf("failed to checkpoint %s, %w", uid, err)
		}
		store.cache[key] = record
	}

	// This protects us from stomping on a previous ghost record
	// that was not cleaned up properly. A ghost record that was
	// not deleted indicates that the VMI shutdown process did not
	// properly handle cleanup of local data.
	if ok && record.UID != uid {
		return fmt.Errorf("can not add ghost record when entry already exists with differing UID")
	}

	if ok && filepath.Clean(record.SocketFile) != socketFile {
		return fmt.Errorf("can not add ghost record when entry already exists with differing socket file location")
	}

	return nil
}

func (store *GhostRecordStore) Delete(namespace string, name string) error {
	store.Lock()
	defer store.Unlock()
	key := namespace + "/" + name
	record, ok := store.cache[key]
	if !ok {
		// already deleted
		return nil
	}

	if string(record.UID) == "" {
		return fmt.Errorf("unable to remove ghost record with empty UID")
	}

	if err := store.checkpointManager.Delete(string(record.UID)); err != nil {
		return fmt.Errorf("failed to delete checkpoint %s, %w", record.UID, err)
	}

	delete(store.cache, key)

	return nil
}

// domainListWatch wraps a cache.ListWatch to explicitly opt out of the
// WatchList streaming-list semantics. The WatchListClient feature gate is
// enabled by default since client-go 1.35, which makes the reflector try a
// watch-list first and expect a "k8s.io/initial-events-end" bookmark. The
// domain informer is a local, non-apiserver informer backed by the
// virt-launcher notify server, which never emits that bookmark, so the
// reflector would fall back and perform an extra list pass. Reporting
// unsupported semantics keeps the reflector on the plain list-then-watch path.
type domainListWatch struct {
	*cache.ListWatch
}

func (domainListWatch) IsWatchListSemanticsUnSupported() bool { return true }

func NewSharedInformer(virtShareDir string, watchdogTimeoutSeconds int, recorder record.EventRecorder, vmiStore cache.Store, resyncPeriod time.Duration, ghostRecordStore *GhostRecordStore) cache.SharedInformer {
	return NewSharedInformerWithAuth(virtShareDir, watchdogTimeoutSeconds, recorder, vmiStore, resyncPeriod, ghostRecordStore, nil)
}

// NewSharedInformerWithAuth creates a domain SharedInformer that
// authenticates virt-launcher sockets during the initial List and during
// resync using projected SA tokens and the TokenReview API.
// authClient may be nil to skip authentication (backwards compatible).
func NewSharedInformerWithAuth(virtShareDir string, watchdogTimeoutSeconds int, recorder record.EventRecorder, vmiStore cache.Store, resyncPeriod time.Duration, ghostRecordStore *GhostRecordStore, authClient kubernetes.Interface) cache.SharedInformer {
	consecutiveFails := new(int)
	runServer := func(ctx context.Context, c chan watch.Event) error {
		return notifyserver.RunServer(virtShareDir, ctx.Done(), c, recorder, vmiStore)
	}
	listFunc := func(ctx context.Context, _ metav1.ListOptions) (runtime.Object, error) {
		return ghostRecordStore.DomainList(ctx, authClient, vmiStore)
	}
	lw := domainListWatch{&cache.ListWatch{
		ListWithContextFunc: listFunc,
		WatchFuncWithContext: func(ctx context.Context, _ metav1.ListOptions) (watch.Interface, error) {
			return newDomainWatcher(ctx, runServer, watchdogTimeoutSeconds, resyncPeriod, recorder, consecutiveFails, ghostRecordStore, authClient, vmiStore), nil
		},
	}}
	return cache.NewSharedInformer(lw, &api.Domain{}, 0)
}

func (store *GhostRecordStore) List(ctx context.Context, _ metav1.ListOptions) (runtime.Object, error) {
	return store.DomainList(ctx, nil, nil)
}

func (store *GhostRecordStore) DomainList(ctx context.Context, authClient kubernetes.Interface, vmiStore cache.Store) (runtime.Object, error) {
	log.Log.V(3).Info("Synchronizing domains")
	// An auth client means the feature gate is on, which is also when
	// discovery moves off ghost records and onto the VMI store.
	var domains []*api.Domain
	if authClient != nil {
		domains = listDomainsFromVMIStore(ctx, vmiStore, authClient)
	} else {
		domains = store.listKnownDomains()
	}
	list := api.DomainList{
		Items: []api.Domain{},
	}
	for _, domain := range domains {
		list.Items = append(list.Items, *domain)
	}
	return &list, nil
}

func (store *GhostRecordStore) listAllKnownDomains() []*api.Domain {
	return store.listKnownDomains()
}

// listKnownDomains enumerates ghost records. It is only reached when the
// LauncherSocketAuthentication gate is off, which is also the only time
// ghost records are maintained, so nothing here authenticates.
func (store *GhostRecordStore) listKnownDomains() []*api.Domain {
	var domains []*api.Domain
	for _, record := range store.list() {
		if domain := getDomainFromRecord(record); domain != nil {
			domains = append(domains, domain)
		}
	}
	return domains
}

// socketEntry pairs a VMI with the launcher socket it owns on this node.
type socketEntry struct {
	vmi  *v1.VirtualMachineInstance
	path string
}

// socketsFromVMIStore derives at most one launcher socket per VMI from the
// VMI store. This replaces ghost record enumeration when the
// LauncherSocketAuthentication feature gate is enabled.
//
// cmdclient.FindSocket applies exactly the filters this needs, and is what
// GetLauncherClient uses to pick its socket: it reads the node name from
// NODE_NAME (which the virt-handler DaemonSet sets) and skips ActivePods
// entries for other nodes without touching the filesystem, keeps only
// sockets that exist, and refuses to guess when a VMI has more than one
// local socket. Sharing it keeps the domain-watcher and the VM controller
// on the same peer.
func socketsFromVMIStore(vmiStore cache.Store) []socketEntry {
	if vmiStore == nil {
		return nil
	}
	var entries []socketEntry
	for _, obj := range vmiStore.List() {
		vmi, ok := obj.(*v1.VirtualMachineInstance)
		if !ok {
			continue
		}
		socket, err := cmdclient.FindSocket(vmi)
		if err != nil {
			// No local socket is the normal case: the VMI runs on another
			// node, or its launcher has not created the socket yet.
			if !errors.Is(err, cmdclient.ErrNoSocketFound) {
				log.Log.Object(vmi).Reason(err).Warning("skipping VMI during launcher socket discovery")
			}
			continue
		}
		entries = append(entries, socketEntry{vmi: vmi, path: socket})
	}
	return entries
}

// listDomainsFromVMIStore builds the domain list from VMI-derived sockets.
// A socket that exists must authenticate before GetDomain, and a failure
// omits it. handleResync cannot evict a domain this List already added.
func listDomainsFromVMIStore(ctx context.Context, vmiStore cache.Store, authClient kubernetes.Interface) []*api.Domain {
	var domains []*api.Domain
	for _, entry := range socketsFromVMIStore(vmiStore) {
		if err := AuthenticateSocket(ctx, authClient, entry.path, entry.vmi.Status.ActivePods); err != nil {
			log.Log.Reason(err).Errorf("launcher authentication failed for socket %s during domain list, skipping", entry.path)
			continue
		}
		if domain := getDomainFromSocket(entry.vmi, entry.path); domain != nil {
			domains = append(domains, domain)
		}
	}
	return domains
}

// getDomainFromSocket connects to a launcher socket and retrieves the
// domain. The socket is known to exist: FindSocket only returns paths it
// has already resolved on disk.
func getDomainFromSocket(vmi *v1.VirtualMachineInstance, socketFile string) *api.Domain {
	log.Log.V(3).Infof("List domains from sock %s", socketFile)
	client, err := cmdclient.NewClient(socketFile)
	if err != nil {
		log.Log.Reason(err).Warningf("failed to connect to cmd client socket %s, preserving domain with Unknown status", socketFile)
		return newDomainFromVMI(vmi, api.DomainStatus{Status: api.Unknown})
	}
	defer client.Close()

	domain, domainExists, err := client.GetDomain()
	if err != nil {
		log.Log.Reason(err).Warningf("failed to list domains on cmd client socket %s, preserving domain with Unknown status", socketFile)
		return newDomainFromVMI(vmi, api.DomainStatus{Status: api.Unknown})
	}
	if !domainExists {
		return nil
	}
	return domain
}

func newDomainFromVMI(vmi *v1.VirtualMachineInstance, status api.DomainStatus) *api.Domain {
	domain := api.NewMinimalDomainWithNS(vmi.Namespace, vmi.Name)
	domain.ObjectMeta.UID = vmi.UID
	domain.Spec.Metadata.KubeVirt.UID = vmi.UID
	domain.Status = status
	return domain
}

func newDomainFromGhostRecord(record ghostRecord, status api.DomainStatus) *api.Domain {
	domain := api.NewMinimalDomainWithNS(record.Namespace, record.Name)
	domain.ObjectMeta.UID = record.UID
	domain.Spec.Metadata.KubeVirt.UID = record.UID
	domain.Status = status

	return domain
}

func getDomainFromRecord(record ghostRecord) *api.Domain {
	socketFile := record.SocketFile

	exists, err := diskutils.FileExists(socketFile)
	if err != nil {
		log.Log.Reason(err).Error("failed access cmd client socket")
		return nil
	}

	if !exists {
		domain := newDomainFromGhostRecord(record, api.DomainStatus{})
		now := metav1.Now()
		domain.DeletionTimestamp = &now
		log.Log.Object(domain).Warning("detected stale domain from ghost record")
		return domain
	}

	log.Log.V(3).Infof("List domains from sock %s", socketFile)
	client, err := cmdclient.NewClient(socketFile)
	if err != nil {
		log.Log.Reason(err).Warningf("failed to connect to cmd client socket %s, preserving domain with Unknown status", socketFile)
		return newDomainFromGhostRecord(record, api.DomainStatus{Status: api.Unknown})
	}
	defer client.Close()

	domain, exists, err := client.GetDomain()
	if err != nil {
		log.Log.Reason(err).Warningf("failed to list domains on cmd client socket %s, preserving domain with Unknown status", socketFile)
		return newDomainFromGhostRecord(record, api.DomainStatus{Status: api.Unknown})
	}
	if !exists {
		return nil
	}
	return domain
}
