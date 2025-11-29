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
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

type LauncherClientInfo struct {
	Client              cmdclient.LauncherClient
	SocketFile          string
	DomainPipeStopChan  chan struct{}
	NotInitializedSince time.Time
	Ready               bool
	closeOnce           sync.Once
}

func (l *LauncherClientInfo) Close() {
	if l == nil {
		return
	}
	l.closeOnce.Do(func() {
		if l.Client != nil {
			l.Client.Close()
		}
		if l.DomainPipeStopChan != nil {
			close(l.DomainPipeStopChan)
		}
	})
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
