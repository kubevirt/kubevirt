/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package device_manager

import (
	"context"
	"errors"
	"os"
	"path"
	"sync"
	"time"

	"google.golang.org/grpc"
	"kubevirt.io/client-go/log"

	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

type DevicePluginBase struct {
	devs         []*pluginapi.Device
	server       *grpc.Server
	socketPath   string
	stop         <-chan struct{}
	health       chan deviceHealth
	resourceName string
	done         chan struct{}
	initialized  bool
	lock         *sync.Mutex
	deregistered chan struct{}
	devicePath   string
	deviceRoot   string
	deviceName   string
}

func (dpi *DevicePluginBase) GetDeviceName() string {
	return dpi.resourceName
}

func (dpi *DevicePluginBase) ListAndWatch(_ *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	s.Send(&pluginapi.ListAndWatchResponse{Devices: dpi.devs})

	done := false
	for {
		select {
		case devHealth := <-dpi.health:
			for _, dev := range dpi.devs {
				// If the devHealth.DevId is empty, it was not set by the device plugin, so we update all devices
				if devHealth.DevId == dev.ID || devHealth.DevId == "" {
					dev.Health = devHealth.Health
				}
			}
			s.Send(&pluginapi.ListAndWatchResponse{Devices: dpi.devs})
		case <-dpi.stop:
			done = true
		case <-dpi.done:
			done = true
		}
		if done {
			break
		}
	}
	emptyList := []*pluginapi.Device{}
	if err := s.Send(&pluginapi.ListAndWatchResponse{Devices: emptyList}); err != nil {
		log.DefaultLogger().Reason(err).Infof("%s device plugin failed to deregister", dpi.resourceName)
	}
	close(dpi.deregistered)
	return nil
}

func (dpi *DevicePluginBase) PreStartContainer(_ context.Context, _ *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	res := &pluginapi.PreStartContainerResponse{}
	return res, nil
}

func (dpi *DevicePluginBase) GetDevicePluginOptions(_ context.Context, _ *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	options := &pluginapi.DevicePluginOptions{
		PreStartRequired: false,
	}
	return options, nil
}

func (dpi *DevicePluginBase) Allocate(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	log.DefaultLogger().Infof("Generic Allocate: resourceName: %s", dpi.deviceName)
	log.DefaultLogger().Infof("Generic Allocate: request: %v", r.ContainerRequests)
	response := pluginapi.AllocateResponse{}
	containerResponse := new(pluginapi.ContainerAllocateResponse)

	dev := new(pluginapi.DeviceSpec)
	dev.HostPath = dpi.devicePath
	dev.ContainerPath = dpi.devicePath
	containerResponse.Devices = []*pluginapi.DeviceSpec{dev}

	response.ContainerResponses = []*pluginapi.ContainerAllocateResponse{containerResponse}

	return &response, nil
}

func (dpi *DevicePluginBase) stopDevicePlugin() error {
	defer func() {
		if !IsChanClosed(dpi.done) {
			close(dpi.done)
		}
	}()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	select {
	case <-dpi.deregistered:
	case <-ticker.C:
	}

	dpi.server.Stop()
	dpi.setInitialized(false)
	return dpi.cleanup()
}

func (dpi *DevicePluginBase) cleanup() error {
	if err := os.Remove(dpi.socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (dpi *DevicePluginBase) GetInitialized() bool {
	dpi.lock.Lock()
	defer dpi.lock.Unlock()
	return dpi.initialized
}

func (dpi *DevicePluginBase) setInitialized(initialized bool) {
	dpi.lock.Lock()
	dpi.initialized = initialized
	dpi.lock.Unlock()
}

func (dpi *DevicePluginBase) register() error {
	conn, err := gRPCConnect(pluginapi.KubeletSocket, connectionTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     path.Base(dpi.socketPath),
		ResourceName: dpi.resourceName,
	}

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		return err
	}
	return nil
}
