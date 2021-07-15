package cgroup

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/configs"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	virtutil "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

// ihol3 make private?
const (
	ProcMountPoint   = "/proc"
	CgroupMountPoint = "/sys/fs/cgroup"
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

// DEFINE INTERFACE HERE

var (
	isolationDetector *isolation.PodIsolationDetector
)

const (
	ProcMountPointNew = "/proc/1/root" // ihol3
	hostBasePath      = ProcMountPointNew + "/sys/fs/cgroup"
)

// ihol3 Change name?
type Manager interface {
	//DeviceManager
	//cpuManager

	Set(r *configs.Resources) error
	cgroups.Manager

	// ihol3 doc!
	// ihol3 add validation for controller name (save Paths() keys once?)
	GetBasePathToHostController(controller string) (string, error)

	// GetControllersAndPaths ... returns key: controller, value: path.
	//GetControllersAndPaths(pid int) (map[string]string, error)

	// GetControllerPath ...
	//GetControllerPath(controller string) string
}

func NewManagerFromPid(pid int) (Manager, error) {
	const isRootless = true

	//controllerPath, err := getBasePathToHostController("devices")
	//if err != nil {
	//	return nil, err
	//}

	cgroupBasePath := getCgroupBasePath(pid)
	controllerPaths, err := cgroups.ParseCgroupFile(cgroupBasePath)
	if err != nil {
		return nil, err
	}

	config := &configs.Cgroup{
		Path:      hostBasePath,
		Paths:     controllerPaths,
		Resources: &configs.Resources{},
	}

	if cgroups.IsCgroup2UnifiedMode() {
		slicePath := controllerPaths[""] // ihol3 is it different than cgroupBasePath?...
		log.Log.Infof("cgroupBasePath: %s", cgroupBasePath)
		log.Log.Infof("slicePath: %s", slicePath)

		return newV2Manager(config, slicePath, isRootless)
	} else {

		return newV1Manager(config, controllerPaths, isRootless)
	}
}

func NewManagerFromVM(vmi *v1.VirtualMachineInstance) (Manager, error) {
	isolationRes, err := detectVMIsolation(vmi, "")
	if err != nil {
		return nil, err
	}

	return NewManagerFromPid(isolationRes.Pid())
}

// NewManagerFromVMAndSocket is similar to NewManagerFromVM but is faster since there is no need
// to search for the socket.
func NewManagerFromVMAndSocket(vmi *v1.VirtualMachineInstance, socket string) (Manager, error) {
	if socket == "" {
		return nil, fmt.Errorf("socket has to be a non-empty string")
	}

	isolationRes, err := detectVMIsolation(vmi, socket)
	if err != nil {
		return nil, err
	}

	return NewManagerFromPid(isolationRes.Pid())
}

func getCgroupBasePath(pid int) string {
	return filepath.Join(ProcMountPoint, strconv.Itoa(pid), "cgroup")
}

func initIsolationDetectorIfNil() {
	if isolationDetector != nil {
		return
	}

	detector := isolation.NewSocketBasedIsolationDetector(virtutil.VirtShareDir)
	isolationDetector = &detector
}

// detectVMIsolation detects VM's isolation. Socket is optional and makes the execution faster
func detectVMIsolation(vm *v1.VirtualMachineInstance, socket string) (isolationRes isolation.IsolationResult, err error) {
	const detectionErrFormat = "cannot detect vm \"%s\", err: %v"
	initIsolationDetectorIfNil()

	if socket == "" {
		isolationRes, err = (*isolationDetector).Detect(vm)
	} else {
		isolationRes, err = (*isolationDetector).DetectForSocket(vm, socket)
	}

	if err != nil {
		return nil, fmt.Errorf(detectionErrFormat, vm.Name, err)
	}

	return isolationRes, nil
}

func getBasePathToHostController(controller string) (string, error) {
	// ihol3
	// if controller not supported -> error?

	if cgroups.IsCgroup2UnifiedMode() {
		return hostBasePath, nil
	}
	return filepath.Join(hostBasePath, controller), nil
}
