/*
 * Copyright The Kubernetes Authors.
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
 */

package main

import (
	"context"
	"errors"
	"fmt"

	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/dynamic-resource-allocation/kubeletplugin"
	"k8s.io/klog/v2"
)

type driver struct {
	helper    *kubeletplugin.Helper
	state     *DeviceState
	cancelCtx func(error)
}

func NewDriver(ctx context.Context, config *Config) (*driver, error) {
	klog.Infof("NewDriver: Starting with registrar path=%s, plugin path=%s", kubeletplugin.KubeletRegistryDir, config.DriverPluginPath())

	draDriver := &driver{
		cancelCtx: config.cancelMainCtx,
	}

	state, err := NewDeviceState(config)
	if err != nil {
		return nil, err
	}
	draDriver.state = state

	helper, err := kubeletplugin.Start(ctx, draDriver,
		kubeletplugin.KubeClient(config.coreclient),
		kubeletplugin.NodeName(config.nodeName),
		kubeletplugin.DriverName(config.driverName),
		kubeletplugin.RegistrarDirectoryPath(kubeletplugin.KubeletRegistryDir),
		kubeletplugin.PluginDataDirectoryPath(config.DriverPluginPath()),
	)
	klog.Infof("NewDriver: kubeletplugin.Start returned, err=%v", err)
	if err != nil {
		return nil, err
	}
	draDriver.helper = helper

	if err = helper.PublishResources(ctx, state.driverResources); err != nil {
		return nil, err
	}

	return draDriver, nil
}

func (d *driver) Shutdown(logger klog.Logger) error {
	d.helper.Stop()
	return nil
}

func (d *driver) PrepareResourceClaims(ctx context.Context, claims []*resourceapi.ResourceClaim) (map[types.UID]kubeletplugin.PrepareResult, error) {
	logger := klog.FromContext(ctx)
	logger.Info("PrepareResourceClaims is called", "numClaims", len(claims))
	result := make(map[types.UID]kubeletplugin.PrepareResult)

	for _, claim := range claims {
		result[claim.UID] = d.prepareResourceClaim(ctx, claim)
	}

	return result, nil
}

func (d *driver) prepareResourceClaim(ctx context.Context, claim *resourceapi.ResourceClaim) kubeletplugin.PrepareResult {
	logger := klog.FromContext(ctx)
	logger.Info("Preparing claim", "uid", claim.UID, "namespace", claim.Namespace, "name", claim.Name)
	preparedPBs, err := d.state.Prepare(ctx, claim)
	if err != nil {
		logger.Error(err, "Error preparing devices for claim", "uid", claim.UID)
		return kubeletplugin.PrepareResult{
			Err: fmt.Errorf("error preparing devices for claim %v: %w", claim.UID, err),
		}
	}
	var prepared []kubeletplugin.Device
	for _, preparedPB := range preparedPBs {
		prepared = append(prepared, kubeletplugin.Device{
			Requests:     preparedPB.GetRequestNames(),
			PoolName:     preparedPB.GetPoolName(),
			DeviceName:   preparedPB.GetDeviceName(),
			CDIDeviceIDs: preparedPB.GetCDIDeviceIDs(),
		})
	}

	logger.Info("Returning newly prepared devices for claim", "uid", claim.UID, "devices", prepared)
	return kubeletplugin.PrepareResult{Devices: prepared}
}

func (d *driver) UnprepareResourceClaims(ctx context.Context, claims []kubeletplugin.NamespacedObject) (map[types.UID]error, error) {
	logger := klog.FromContext(ctx)
	logger.Info("UnprepareResourceClaims is called", "numClaims", len(claims))
	result := make(map[types.UID]error)

	for _, claim := range claims {
		result[claim.UID] = d.unprepareResourceClaim(ctx, claim)
	}

	return result, nil
}

func (d *driver) unprepareResourceClaim(ctx context.Context, claim kubeletplugin.NamespacedObject) error {
	if err := d.state.Unprepare(ctx, claim.UID); err != nil {
		return fmt.Errorf("error unpreparing devices for claim %v: %w", claim.UID, err)
	}

	return nil
}

func (d *driver) HandleError(ctx context.Context, err error, msg string) {
	utilruntime.HandleErrorWithContext(ctx, err, msg)
	if !errors.Is(err, kubeletplugin.ErrRecoverable) && d.cancelCtx != nil {
		d.cancelCtx(fmt.Errorf("fatal background error: %w", err))
	}
}
