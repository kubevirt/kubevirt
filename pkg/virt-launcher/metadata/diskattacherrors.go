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

package metadata

import (
	"cmp"
	"slices"
	"sync"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// DiskAttachErrors tracks the last hotplug attach error per volume.
// Unlike SafeData, it holds a map, which is not comparable.
type DiskAttachErrors struct {
	m           sync.Mutex
	dirtyChanel chan<- struct{}
	errors      map[string]string
}

// Set records the attach error of the given volume.
// In case a notification channel exists and the error changed, a signal is sent.
func (d *DiskAttachErrors) Set(volumeName, attachError string) {
	d.m.Lock()
	defer d.m.Unlock()
	if current, exists := d.errors[volumeName]; exists && current == attachError {
		return
	}
	if d.errors == nil {
		d.errors = map[string]string{}
	}
	d.errors[volumeName] = attachError
	notify(d.dirtyChanel)
}

// Retain drops the errors of all volumes for which keep returns false.
// In case a notification channel exists and an error got dropped, a signal is sent.
func (d *DiskAttachErrors) Retain(keep func(volumeName string) bool) {
	d.m.Lock()
	defer d.m.Unlock()
	changed := false
	for volumeName := range d.errors {
		if !keep(volumeName) {
			delete(d.errors, volumeName)
			changed = true
		}
	}
	if changed {
		notify(d.dirtyChanel)
	}
}

// Load returns the recorded attach errors, sorted by volume name.
func (d *DiskAttachErrors) Load() []api.DiskStatus {
	d.m.Lock()
	defer d.m.Unlock()
	if len(d.errors) == 0 {
		return nil
	}
	statuses := make([]api.DiskStatus, 0, len(d.errors))
	for volumeName, attachError := range d.errors {
		statuses = append(statuses, api.DiskStatus{Name: volumeName, AttachError: attachError})
	}
	slices.SortFunc(statuses, func(a, b api.DiskStatus) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return statuses
}
