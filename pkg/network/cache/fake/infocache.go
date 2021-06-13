package fake

import (
	"os"
	"sync"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func NewFakeInMemoryNetworkCacheFactory() cache.InterfaceCacheFactory {
	return &fakeInterfaceCacheFactory{
		vmiCacheStores:    map[types.UID]*fakePodInterfaceCacheStore{},
		domainCacheStores: map[string]*fakeDomainInterfaceStore{},
		dhcpConfigStores:  map[string]*fakeDHCPConfigCacheStore{},
		lock:              &sync.Mutex{},
	}
}

type fakeInterfaceCacheFactory struct {
	vmiCacheStores    map[types.UID]*fakePodInterfaceCacheStore
	domainCacheStores map[string]*fakeDomainInterfaceStore
	dhcpConfigStores  map[string]*fakeDHCPConfigCacheStore
	lock              *sync.Mutex
}

func (f *fakeInterfaceCacheFactory) CacheForVMI(vmi *v1.VirtualMachineInstance) cache.PodInterfaceCacheStore {
	f.lock.Lock()
	defer f.lock.Unlock()
	if store, exists := f.vmiCacheStores[vmi.UID]; exists {
		return store
	}
	f.vmiCacheStores[vmi.UID] = &fakePodInterfaceCacheStore{
		store: map[string]*cache.PodCacheInterface{},
		lock:  &sync.Mutex{},
	}
	return f.vmiCacheStores[vmi.UID]
}

func (f *fakeInterfaceCacheFactory) CacheDomainInterfaceForPID(pid string) cache.DomainInterfaceStore {
	f.lock.Lock()
	defer f.lock.Unlock()
	if store, exists := f.domainCacheStores[pid]; exists {
		return store
	}
	f.domainCacheStores[pid] = &fakeDomainInterfaceStore{
		store: map[string]*api.Interface{},
		lock:  &sync.Mutex{},
	}
	return f.domainCacheStores[pid]
}

func (f *fakeInterfaceCacheFactory) CacheDHCPConfigForPid(pid string) cache.DHCPConfigStore {
	f.lock.Lock()
	defer f.lock.Unlock()
	if store, exists := f.dhcpConfigStores[pid]; exists {
		return store
	}
	f.dhcpConfigStores[pid] = &fakeDHCPConfigCacheStore{
		store: map[string]*cache.DHCPConfig{},
		lock:  &sync.Mutex{},
	}
	return f.dhcpConfigStores[pid]
}

type fakePodInterfaceCacheStore struct {
	lock  *sync.Mutex
	store map[string]*cache.PodCacheInterface
}

func (f *fakePodInterfaceCacheStore) Read(ifaceName string) (*cache.PodCacheInterface, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if val, exists := f.store[ifaceName]; exists {
		return val, nil
	}
	return nil, os.ErrNotExist
}

func (f *fakePodInterfaceCacheStore) Write(ifaceName string, cacheInterface *cache.PodCacheInterface) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.store[ifaceName] = cacheInterface
	return nil
}

func (f *fakePodInterfaceCacheStore) Remove() error {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.store = map[string]*cache.PodCacheInterface{}
	return nil
}

type fakeDomainInterfaceStore struct {
	lock  *sync.Mutex
	store map[string]*api.Interface
}

func (f *fakeDomainInterfaceStore) Read(ifaceName string) (*api.Interface, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if val, exists := f.store[ifaceName]; exists {
		return val, nil
	}
	return nil, os.ErrNotExist
}

func (f *fakeDomainInterfaceStore) Write(ifaceName string, cacheInterface *api.Interface) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.store[ifaceName] = cacheInterface
	return nil
}

type fakeDHCPConfigCacheStore struct {
	lock  *sync.Mutex
	store map[string]*cache.DHCPConfig
}

func (f *fakeDHCPConfigCacheStore) Read(ifaceName string) (*cache.DHCPConfig, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if val, exists := f.store[ifaceName]; exists {
		return val, nil
	}
	return nil, os.ErrNotExist
}

func (f *fakeDHCPConfigCacheStore) Write(ifaceName string, vifToCache *cache.DHCPConfig) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.store[ifaceName] = vifToCache
	return nil
}
