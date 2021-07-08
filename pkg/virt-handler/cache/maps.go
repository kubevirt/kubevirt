package cache

import (
	"fmt"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/types"

	netcache "kubevirt.io/kubevirt/pkg/network/cache"
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

func syncMapLen(m *sync.Map) int {
	mapLen := 0
	m.Range(func(k, v interface{}) bool {
		mapLen += 1
		return true
	})
	return mapLen
}
