package cache

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"

	netcache "kubevirt.io/kubevirt/pkg/network/cache"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

type PodInterfaceByVMIAndName struct {
	syncMap sync.Map
}

func (p *PodInterfaceByVMIAndName) DeleteAllForVMI(vmiUID types.UID) {
	// Clean Pod interface cache from map and files
	p.syncMap.Range(func(key, value interface{}) bool {
		if strings.Contains(key.(string), string(vmiUID)) {
			p.syncMap.Delete(key)
		}
		return true
	})
}

func (p *PodInterfaceByVMIAndName) Load(vmiUID types.UID, interfaceName string) (*netcache.PodCacheInterface, bool) {
	result, exists := p.syncMap.Load(p.key(vmiUID, interfaceName))

	if !exists {
		return nil, false
	}
	return p.cast(result), true
}

func (p *PodInterfaceByVMIAndName) Store(vmiUID types.UID, interfaceName string, podCacheInterface *netcache.PodCacheInterface) {
	p.syncMap.Store(p.key(vmiUID, interfaceName), podCacheInterface)
}

func (*PodInterfaceByVMIAndName) cast(result interface{}) *netcache.PodCacheInterface {
	podCacheInterface, ok := result.(*netcache.PodCacheInterface)
	if !ok {
		panic(fmt.Sprintf("failed casting %+v to *PodCacheInterface", result))
	}
	return podCacheInterface
}

func (*PodInterfaceByVMIAndName) key(vmiUID types.UID, interfaceName string) string {
	return fmt.Sprintf("%s/%s", vmiUID, interfaceName)
}

func (p *PodInterfaceByVMIAndName) Size() int {
	return syncMapLen(&p.syncMap)
}

type LauncherClientInfo struct {
	Client              cmdclient.LauncherClient
	SocketFile          string
	DomainPipeStopChan  chan struct{}
	NotInitializedSince time.Time
	Ready               bool
}

type LauncherClientInfoByVMI struct {
	syncMap sync.Map
}

func (l *LauncherClientInfoByVMI) Delete(vmiUID types.UID) {
	l.syncMap.Delete(vmiUID)
}

func (l *LauncherClientInfoByVMI) Store(vmiUID types.UID, launcherClientInfo *LauncherClientInfo) {
	l.syncMap.Store(vmiUID, launcherClientInfo)
}

func (l *LauncherClientInfoByVMI) Load(vmiUID types.UID) (*LauncherClientInfo, bool) {
	result, exists := l.syncMap.Load(vmiUID)
	if !exists {
		return nil, exists
	}
	return l.cast(result), exists
}

func (*LauncherClientInfoByVMI) cast(result interface{}) *LauncherClientInfo {
	launcherClientInfo, ok := result.(*LauncherClientInfo)
	if !ok {
		panic(fmt.Sprintf("failed casting %+v to *LauncherClientInfo", result))
	}
	return launcherClientInfo
}

func syncMapLen(m *sync.Map) int {
	mapLen := 0
	m.Range(func(k, v interface{}) bool {
		mapLen += 1
		return true
	})
	return mapLen
}
