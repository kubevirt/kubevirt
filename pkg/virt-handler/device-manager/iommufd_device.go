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

package device_manager

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/util"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
	"kubevirt.io/kubevirt/pkg/virt-handler/selinux"
)

const (
	iommuDevicePath   = "/dev/iommu"
	iommufdDeviceName = "iommufd"

	iommufdContainerSocketPath = "/var/run/kubevirt/iommufd.sock"

	// IOMMU_OPTION is the ioctl number for the IOMMUFD OPTION command.
	// Defined in Linux uAPI as _IO(IOMMUFD_TYPE, IOMMUFD_CMD_OPTION)
	// where IOMMUFD_TYPE = ';' (0x3B) and IOMMUFD_CMD_OPTION = 0x87.
	//nolint:revive,stylecheck
	IOMMU_OPTION = 0x3B87

	iommufdSocketAcceptTimeout = 15 * time.Minute
)

var iommufdSocketDir = "/var/run/kubevirt"

// iommuOption mirrors the kernel's struct iommu_option from
// include/uapi/linux/iommufd.h.
type iommuOption struct {
	Size     uint32
	OptionID uint32
	Op       uint16
	Reserved uint16
	ObjectID uint32
	Val64    uint64
}

type IOMMUFDDevicePlugin struct {
	devs         []*pluginapi.Device
	server       *grpc.Server
	socketPath   string
	stop         <-chan struct{}
	health       chan deviceHealth
	deviceName   string
	resourceName string
	done         chan struct{}
	deviceRoot   string
	initialized  bool
	lock         *sync.Mutex
	deregistered chan struct{}
}

func NewIOMMUFDDevicePlugin(maxDevices int) *IOMMUFDDevicePlugin {
	serverSock := SocketPath(iommufdDeviceName)
	dpi := &IOMMUFDDevicePlugin{
		devs:         make([]*pluginapi.Device, 0, maxDevices),
		socketPath:   serverSock,
		health:       make(chan deviceHealth),
		deviceName:   iommufdDeviceName,
		deviceRoot:   util.HostRootMount,
		resourceName: fmt.Sprintf("%s/%s", DeviceNamespace, iommufdDeviceName),
		initialized:  false,
		lock:         &sync.Mutex{},
	}

	for i := 0; i < maxDevices; i++ {
		dpi.devs = append(dpi.devs, &pluginapi.Device{
			ID:     iommufdDeviceName + strconv.Itoa(i),
			Health: pluginapi.Healthy,
		})
	}

	return dpi
}

func (dpi *IOMMUFDDevicePlugin) GetDeviceName() string {
	return dpi.deviceName
}

func (dpi *IOMMUFDDevicePlugin) Start(stop <-chan struct{}) (err error) {
	logger := log.DefaultLogger()
	dpi.stop = stop
	dpi.done = make(chan struct{})
	dpi.deregistered = make(chan struct{})

	if err = dpi.cleanup(); err != nil {
		return err
	}

	sock, err := net.Listen("unix", dpi.socketPath)
	if err != nil {
		return fmt.Errorf("error creating GRPC server socket: %v", err)
	}

	dpi.server = grpc.NewServer([]grpc.ServerOption{}...)
	defer dpi.stopDevicePlugin()

	pluginapi.RegisterDevicePluginServer(dpi.server, dpi)

	errChan := make(chan error, 2)

	go func() {
		errChan <- dpi.server.Serve(sock)
	}()

	if err = waitForGRPCServer(dpi.socketPath, connectionTimeout); err != nil {
		return fmt.Errorf("error starting the GRPC server: %v", err)
	}

	if err = dpi.register(); err != nil {
		return fmt.Errorf("error registering with device plugin manager: %v", err)
	}

	go func() {
		errChan <- dpi.healthCheck()
	}()

	dpi.setInitialized(true)
	logger.Infof("%s device plugin started", dpi.deviceName)
	return <-errChan
}

func (dpi *IOMMUFDDevicePlugin) stopDevicePlugin() error {
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

func (dpi *IOMMUFDDevicePlugin) register() error {
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
	return err
}

func (dpi *IOMMUFDDevicePlugin) ListAndWatch(_ *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	dpi.updateDeviceHealth()
	s.Send(&pluginapi.ListAndWatchResponse{Devices: dpi.devs})

	done := false
	for {
		select {
		case <-dpi.health:
			dpi.updateDeviceHealth()
			if err := s.Send(&pluginapi.ListAndWatchResponse{Devices: dpi.devs}); err != nil {
				log.DefaultLogger().Reason(err).Errorf("failed to send updated device list")
			}
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
		log.DefaultLogger().Reason(err).Infof("%s device plugin failed to deregister", dpi.deviceName)
	}
	close(dpi.deregistered)
	return nil
}

func (dpi *IOMMUFDDevicePlugin) updateDeviceHealth() {
	health := pluginapi.Healthy
	if !supportsIOMMUFD() {
		health = pluginapi.Unhealthy
	}
	for _, dev := range dpi.devs {
		dev.Health = health
	}
}

func (dpi *IOMMUFDDevicePlugin) Allocate(_ context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	logger := log.DefaultLogger()
	logger.Infof("IOMMUFD Allocate: %d container request(s)", len(r.ContainerRequests))

	response := pluginapi.AllocateResponse{}

	for range r.ContainerRequests {
		containerResponse := new(pluginapi.ContainerAllocateResponse)

		if !supportsIOMMUFD() {
			return nil, fmt.Errorf("IOMMUFD not supported on this node (%s not found)", iommuDevicePath)
		}

		containerResponse.Devices = []*pluginapi.DeviceSpec{
			{
				HostPath:      iommuDevicePath,
				ContainerPath: iommuDevicePath,
				Permissions:   "mrw",
			},
		}

		iommuFD, err := openAndConfigureIOMMUFDFunc(iommufdSocketDir)
		if err != nil {
			return nil, fmt.Errorf("failed to open/configure IOMMUFD: %w", err)
		}

		hostSocketPath, err := createIOMMUFDSocket(iommuFD, iommufdSocketDir)
		if err != nil {
			unix.Close(iommuFD)
			return nil, fmt.Errorf("failed to create IOMMUFD socket: %w", err)
		}

		containerResponse.Mounts = []*pluginapi.Mount{
			{
				HostPath:      hostSocketPath,
				ContainerPath: iommufdContainerSocketPath,
				ReadOnly:      false,
			},
		}

		response.ContainerResponses = append(response.ContainerResponses, containerResponse)
	}

	return &response, nil
}

func (dpi *IOMMUFDDevicePlugin) GetDevicePluginOptions(_ context.Context, _ *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{PreStartRequired: false}, nil
}

func (dpi *IOMMUFDDevicePlugin) PreStartContainer(_ context.Context, _ *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (dpi *IOMMUFDDevicePlugin) healthCheck() error {
	logger := log.DefaultLogger()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create fsnotify watcher: %v", err)
	}
	defer func() { _ = watcher.Close() }()

	devicePath := filepath.Join(dpi.deviceRoot, iommuDevicePath)
	dirName := filepath.Dir(devicePath)
	if err = watcher.Add(dirName); err != nil {
		return fmt.Errorf("failed to add the device root path to the watcher: %v", err)
	}

	if _, err = os.Stat(devicePath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("could not stat the device: %v", err)
		}
		logger.Infof("device '%s' is not present, devices will be reported as unhealthy", iommuDevicePath)
	} else {
		logger.Infof("device '%s' is present", iommuDevicePath)
	}

	// This allows us to detect when kubelet removes the socket on restart (event.Name == dp.socketPath),
	// so we can exit and trigger re-registration.
	if dpi.socketPath == "" {
		return fmt.Errorf("socket path is empty, kubelet restart detection will not work")
	}

	dirName = filepath.Dir(dpi.socketPath)
	if err = watcher.Add(dirName); err != nil {
		return fmt.Errorf("failed to add the device-plugin kubelet path to the watcher: %v", err)
	}
	if _, err = os.Stat(dpi.socketPath); err != nil {
		return fmt.Errorf("failed to stat the device-plugin socket: %v", err)
	}

	for {
		select {
		case <-dpi.stop:
			return nil
		case err := <-watcher.Errors:
			logger.Reason(err).Errorf("error watching devices and device plugin directory")
		case event := <-watcher.Events:
			logger.V(4).Infof("health Event: %v", event)
			if event.Name == devicePath {
				if event.Op == fsnotify.Create {
					logger.Infof("/dev/iommu appeared")
					dpi.health <- deviceHealth{}
				} else if event.Op == fsnotify.Remove || event.Op == fsnotify.Rename {
					logger.Infof("/dev/iommu disappeared")
					dpi.health <- deviceHealth{}
				}
			} else if event.Name == dpi.socketPath && event.Op == fsnotify.Remove {
				logger.Infof("device socket file for device %s was removed, kubelet probably restarted", dpi.deviceName)
				return nil
			}
		}
	}
}

func (dpi *IOMMUFDDevicePlugin) cleanup() error {
	if err := os.Remove(dpi.socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (dpi *IOMMUFDDevicePlugin) GetInitialized() bool {
	dpi.lock.Lock()
	defer dpi.lock.Unlock()
	return dpi.initialized
}

func (dpi *IOMMUFDDevicePlugin) setInitialized(initialized bool) {
	dpi.lock.Lock()
	defer dpi.lock.Unlock()
	dpi.initialized = initialized
}

var iommuDeviceCheckPath = filepath.Join(util.HostRootMount, iommuDevicePath)

func supportsIOMMUFD() bool {
	_, err := os.Stat(iommuDeviceCheckPath)
	return err == nil
}

// openAndConfigureIOMMUFD opens /dev/iommu via a SELinux-relabeled temporary
// device node and sets IOMMU_OPTION_RLIMIT_MODE. This replicates libvirt's
// virIOMMUFDOpenDevice + virIOMMUFDSetRLimitMode.
func openAndConfigureIOMMUFD(socketDir string) (int, error) {
	fd, err := openUnprivilegedIOMMUFD(socketDir)
	if err != nil {
		return -1, err
	}

	option := iommuOption{
		Size:     uint32(unsafe.Sizeof(iommuOption{})),
		OptionID: 0, // IOMMU_OPTION_RLIMIT_MODE
		Op:       0, // IOMMU_OPTION_OP_SET
		Val64:    1, // enable rlimit mode
	}

	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(IOMMU_OPTION),
		uintptr(unsafe.Pointer(&option)),
	)
	if errno != 0 {
		unix.Close(fd)
		return -1, fmt.Errorf("IOMMU_OPTION ioctl failed: %v", errno)
	}

	log.DefaultLogger().Infof("Opened and configured IOMMUFD (fd=%d, rlimit_mode=true)", fd)
	return fd, nil
}

// openUnprivilegedIOMMUFD creates a temporary device node for /dev/iommu,
// relabels it with a container-friendly SELinux context, and returns an FD
// that virt-launcher is allowed to receive via SCM_RIGHTS.
// This directly uses /dev/iommu directly because it needs to stat + mknod + open
// the device from the container's namespace;
// the FD must originate from within the container's mount namespace to be usable.
func openUnprivilegedIOMMUFD(socketDir string) (int, error) {
	var stat unix.Stat_t
	if err := unix.Stat(iommuDevicePath, &stat); err != nil {
		return -1, fmt.Errorf("failed to stat %s: %w", iommuDevicePath, err)
	}

	tmpNodePath := filepath.Join(socketDir, fmt.Sprintf("iommu-%s.dev", uuid.New().String()))

	if err := unix.Mknod(tmpNodePath, unix.S_IFCHR|0600, int(stat.Rdev)); err != nil {
		return -1, fmt.Errorf("mknod failed for temporary iommu node: %w", err)
	}
	defer os.Remove(tmpNodePath)

	if err := relabelPathFunc(tmpNodePath); err != nil {
		return -1, fmt.Errorf("failed to relabel temporary iommu node: %w", err)
	}

	f, err := os.OpenFile(tmpNodePath, os.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, fmt.Errorf("failed to open relabeled iommu node: %w", err)
	}

	fd, err := unix.Dup(int(f.Fd()))
	f.Close()
	if err != nil {
		return -1, fmt.Errorf("dup failed: %w", err)
	}

	return fd, nil
}

// createIOMMUFDSocket creates a one-shot Unix domain socket that transfers
// the IOMMUFD file descriptor to a connecting client via SCM_RIGHTS.
func createIOMMUFDSocket(iommuFD int, socketDir string) (string, error) {
	if err := ensureDirWithRelabel(socketDir); err != nil {
		return "", err
	}

	hostSocketPath := filepath.Join(socketDir, fmt.Sprintf("iommufd-%s.sock", uuid.New().String()))
	os.Remove(hostSocketPath)

	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: hostSocketPath, Net: "unix"})
	if err != nil {
		return "", fmt.Errorf("failed to listen on %s: %w", hostSocketPath, err)
	}

	if err := os.Chmod(hostSocketPath, 0666); err != nil {
		listener.Close()
		os.Remove(hostSocketPath)
		return "", fmt.Errorf("failed to chmod socket %s: %w", hostSocketPath, err)
	}

	if err := relabelPathFunc(hostSocketPath); err != nil {
		listener.Close()
		os.Remove(hostSocketPath)
		return "", fmt.Errorf("failed to relabel socket: %w", err)
	}

	logger := log.DefaultLogger()
	logger.Infof("IOMMUFD socket created at %s, waiting for connection", hostSocketPath)

	go func() {
		defer listener.Close()
		defer os.Remove(hostSocketPath)
		defer unix.Close(iommuFD)

		if err := listener.SetDeadline(time.Now().Add(iommufdSocketAcceptTimeout)); err != nil {
			logger.Reason(err).Errorf("failed to set deadline on IOMMUFD socket")
			return
		}

		conn, err := listener.AcceptUnix()
		if err != nil {
			logger.Reason(err).Errorf("IOMMUFD socket accept failed")
			return
		}
		defer conn.Close()

		logger.Infof("IOMMUFD connection accepted, sending FD %d", iommuFD)

		rights := unix.UnixRights(iommuFD)
		if _, _, err := conn.WriteMsgUnix([]byte{0}, rights, nil); err != nil {
			logger.Reason(err).Errorf("IOMMUFD WriteMsgUnix failed")
			return
		}

		ack := make([]byte, 1)
		if _, err := conn.Read(ack); err != nil {
			logger.Reason(err).Warning("IOMMUFD ACK read failed (non-fatal)")
		} else {
			logger.Infof("IOMMUFD FD successfully passed and ACK received (fd=%d)", iommuFD)
		}
	}()

	return hostSocketPath, nil
}

var openAndConfigureIOMMUFDFunc = openAndConfigureIOMMUFD

var relabelPathFunc = defaultRelabelPath

func defaultRelabelPath(filePath string) error {
	se, present, err := selinux.NewSELinux()
	if err != nil {
		return fmt.Errorf("failed to detect SELinux: %w", err)
	}
	if !present {
		log.DefaultLogger().V(4).Infof("SELinux not present, skipping relabel of %s", filePath)
		return nil
	}

	safePath, err := safepath.JoinAndResolveWithRelativeRoot("/", filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve path %s for relabeling: %w", filePath, err)
	}

	return selinux.RelabelFilesUnprivileged(se.IsPermissive(), safePath)
}

func ensureDirWithRelabel(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return relabelPathFunc(dir)
}
