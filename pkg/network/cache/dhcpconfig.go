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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package cache

import "kubevirt.io/kubevirt/pkg/os/fs"

const dhcpConfigCachedPattern = "/proc/%s/root/var/run/kubevirt-private/vif-cache-%s.json"

type dhcpConfigCacheStore struct {
	pid     string
	pattern string
	fs      fs.Fs
}

func (d dhcpConfigCacheStore) Read(ifaceName string) (*DHCPConfig, error) {
	cachedIface := &DHCPConfig{}
	err := readFromCachedFile(d.fs, cachedIface, d.getInterfaceCacheFile(ifaceName))
	return cachedIface, err
}

func (d dhcpConfigCacheStore) Write(ifaceName string, ifaceToCache *DHCPConfig) error {
	return writeToCachedFile(d.fs, ifaceToCache, d.getInterfaceCacheFile(ifaceName))
}

func (d dhcpConfigCacheStore) getInterfaceCacheFile(ifaceName string) string {
	return getInterfaceCacheFile(d.pattern, d.pid, ifaceName)
}
