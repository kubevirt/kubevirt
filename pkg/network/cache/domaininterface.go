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

import (
	"kubevirt.io/kubevirt/pkg/os/fs"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const virtLauncherCachedPattern = "/proc/%s/root/var/run/kubevirt-private/interface-cache-%s.json"

type domainInterfaceStore struct {
	pid     string
	pattern string
	fs      fs.Fs
}

func (d domainInterfaceStore) Read(ifaceName string) (*api.Interface, error) {
	iface := &api.Interface{}
	err := readFromCachedFile(d.fs, iface, getInterfaceCacheFile(d.pattern, d.pid, ifaceName))
	return iface, err
}

func (d domainInterfaceStore) Write(ifaceName string, cacheInterface *api.Interface) (err error) {
	err = writeToCachedFile(d.fs, cacheInterface, getInterfaceCacheFile(d.pattern, d.pid, ifaceName))
	return
}
