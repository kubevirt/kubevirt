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

package driver

import (
	"context"
	"errors"

	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/dynamic-resource-allocation/kubeletplugin"

	"kubevirt.io/client-go/log"
)

const DriverName = "vhostuser.net.kubevirt.io"

type Driver struct {
	cancel context.CancelFunc
}

// New returns a Driver that cancels the given context on non-recoverable errors,
// letting the caller run its regular shutdown path.
func New(cancel context.CancelFunc) *Driver {
	return &Driver{cancel: cancel}
}

func (d *Driver) PrepareResourceClaims(ctx context.Context, claims []*resourceapi.ResourceClaim) (map[types.UID]kubeletplugin.PrepareResult, error) {
	results := make(map[types.UID]kubeletplugin.PrepareResult)
	for _, claim := range claims {
		cdiDeviceID, err := prepareHostpath(claim.UID)
		if err != nil {
			results[claim.UID] = kubeletplugin.PrepareResult{
				Err: fmt.Errorf("failed to prepare claim %s: %w", claim.Name, err),
			}
			continue
		}

		var devices []kubeletplugin.Device
		for _, result := range claim.Status.Allocation.Devices.Results {
			devices = append(devices, kubeletplugin.Device{
				PoolName:     result.Pool,
				DeviceName:   result.Device,
				CDIDeviceIDs: []string{cdiDeviceID},
			})
		}
		results[claim.UID] = kubeletplugin.PrepareResult{Devices: devices}
	}
	return results, nil
}

func (d *Driver) UnprepareResourceClaims(ctx context.Context, claims []kubeletplugin.NamespacedObject) (map[types.UID]error, error) {
	results := make(map[types.UID]error)
	for _, claim := range claims {
		unprepareHostpath(claim.UID)
		results[claim.UID] = nil
	}
	return results, nil
}

func (d *Driver) HandleError(ctx context.Context, err error, msg string) {
	log.Log.Reason(err).Errorf("DRA plugin error: %s", msg)
	if errors.Is(err, kubeletplugin.ErrRecoverable) {
		return
	}
	d.cancel()
}
