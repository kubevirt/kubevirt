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
	"errors"
	"os"
	"path/filepath"

	"kubevirt.io/kubevirt/pkg/util"
)

const launcherPidFileName = ".launcher-pid"

type LauncherPidCache struct {
	cache *Cache
}

type LauncherPidData struct {
	PID int `json:"pid"`
}

func NewLauncherPidCache(creator cacheCreator, uid string) LauncherPidCache {
	return LauncherPidCache{
		creator.New(filepath.Join(util.VirtPrivateDir, podIfaceCacheDirName, uid, launcherPidFileName)),
	}
}

func (l LauncherPidCache) Read() (int, error) {
	data := &LauncherPidData{}
	if _, err := l.cache.Read(data); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	return data.PID, nil
}

func (l LauncherPidCache) Write(pid int) error {
	return l.cache.Write(&LauncherPidData{PID: pid})
}

func (l LauncherPidCache) Remove() error {
	return l.cache.Delete()
}
