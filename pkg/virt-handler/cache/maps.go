package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

type LauncherClientInfo struct {
	Client               cmdclient.LauncherClient
	SocketFile           string
	DomainPipeCancelFunc context.CancelFunc
	NotInitializedSince  time.Time
	Ready                bool
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
