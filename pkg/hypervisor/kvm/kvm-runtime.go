package kvm

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"kubevirt.io/client-go/log"

	"github.com/mitchellh/go-ps"
	"golang.org/x/sys/unix"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var (
	// parse thread comm value expression
	kvmVcpuRegex = regexp.MustCompile(`^CPU (\d+)/KVM\n$`) // These threads follow this naming pattern as their command value (/proc/{pid}/task/{taskid}/comm)
	// QEMU uses threads to represent vCPUs.

)

type KvmVirtRuntime struct {
	podIsolationDetector isolation.PodIsolationDetector
	logger               *log.FilteredLogger
	KvmLauncherResourceRenderer
}

func NewKvmVirtRuntime(podIsoDetector isolation.PodIsolationDetector, logger *log.FilteredLogger) *KvmVirtRuntime {
	return &KvmVirtRuntime{
		podIsolationDetector: podIsoDetector,
		logger:               logger,
	}
}

func (k *KvmVirtRuntime) HandleHousekeeping(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager, domain *api.Domain) error {
	if vmi.IsCPUDedicated() && vmi.Spec.Domain.CPU.IsolateEmulatorThread {
		err := k.configureHousekeepingCgroup(vmi, cgroupManager, domain)
		if err != nil {
			return err
		}
	}

	// Configure vcpu scheduler for realtime workloads and affine PIT thread for dedicated CPU
	if vmi.IsRealtimeEnabled() && !vmi.IsRunning() && !vmi.IsFinal() {
		k.logger.Object(vmi).Info("Configuring vcpus for real time workloads")
		if err := k.configureVCPUScheduler(vmi); err != nil {
			return err
		}
	}
	if vmi.IsCPUDedicated() && !vmi.IsRunning() && !vmi.IsFinal() {
		k.logger.V(3).Object(vmi).Info("Affining PIT thread")
		if err := k.affinePitThread(vmi); err != nil {
			return err
		}
	}
	return nil
}

func (k *KvmVirtRuntime) GetQEMUProcess(podIsoDetector isolation.PodIsolationDetector, vmi *v1.VirtualMachineInstance) (ps.Process, error) {
	res, err := podIsoDetector.Detect(vmi)
	if err != nil {
		return nil, err
	}
	processes, err := ps.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to get all processes: %v", err)
	}
	qemuProcess, err := findIsolatedQemuProcess(processes, res.PPid())
	if err != nil {
		return nil, err
	}
	return qemuProcess, nil
}

func (k *KvmVirtRuntime) GetVirtqemudProcess(podIsoDetector isolation.PodIsolationDetector, vmi *v1.VirtualMachineInstance) (ps.Process, error) {
	res, err := podIsoDetector.Detect(vmi)
	if err != nil {
		return nil, err
	}
	processes, err := ps.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to get all processes: %v", err)
	}
	launcherPid := res.Pid()

	for _, process := range processes {
		// consider all processes that are virt-launcher children
		if process.PPid() != launcherPid {
			continue
		}

		// virtqemud process sets the memory lock limit before fork/exec-ing into qemu
		if process.Executable() != "virtqemud" {
			continue
		}

		return process, nil
	}

	return nil, nil
}

// findIsolatedQemuProcess Returns the first occurrence of the QEMU process whose parent is PID"
func findIsolatedQemuProcess(processes []ps.Process, pid int) (ps.Process, error) {
	var qemuProcessExecutablePrefixes = []string{"qemu-system", "qemu-kvm"}
	processes = childProcesses(processes, pid)
	for _, execPrefix := range qemuProcessExecutablePrefixes {
		if qemuProcess := lookupProcessByExecutablePrefix(processes, execPrefix); qemuProcess != nil {
			return qemuProcess, nil
		}
	}

	return nil, fmt.Errorf("no QEMU process found under process %d child processes", pid)
}

// AdjustQemuProcessMemoryLimits adjusts QEMU process MEMLOCK rlimits that runs inside
// virt-launcher pod on the given VMI according to its spec.
// Only VMI's with VFIO devices (e.g: SRIOV, GPU), SEV or RealTime workloads require QEMU process MEMLOCK adjustment.
// For VMI's that are not running yet, we need to adjust the memlock limits of the virtqemud process
// which will later fork/exec into the QEMU process.
// For VMI's that are already running, we need to adjust the memlock limits of the QEMU process itself.
func (k *KvmVirtRuntime) adjustQemuProcessMemoryLimits(podIsoDetector isolation.PodIsolationDetector, vmi *v1.VirtualMachineInstance, additionalOverheadRatio *string) error {
	if !util.IsVFIOVMI(vmi) && !vmi.IsRealtimeEnabled() && !util.IsSEVVMI(vmi) {
		return nil
	}

	var targetProcess ps.Process
	var err error
	if vmi.IsRunning() {
		targetProcess, err = k.GetQEMUProcess(podIsoDetector, vmi)
		if err != nil {
			return err
		}
	} else {
		targetProcess, err = k.GetVirtqemudProcess(podIsoDetector, vmi)
		if err != nil {
			return err
		}
		if targetProcess == nil {
			// TODO L1VH: Return quietly. Check if this is the right behavior.
			return nil
		}
	}

	qemuProcessID := targetProcess.Pid()
	// make the best estimate for memory required by libvirt
	memlockSize := k.GetMemoryOverhead(vmi, runtime.GOARCH, additionalOverheadRatio)
	// Add max memory assigned to the VM
	var vmiBaseMemory *resource.Quantity

	switch {
	case vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.MaxGuest != nil:
		vmiBaseMemory = vmi.Spec.Domain.Memory.MaxGuest
	case vmi.Spec.Domain.Resources.Requests.Memory() != nil:
		vmiBaseMemory = vmi.Spec.Domain.Resources.Requests.Memory()
	case vmi.Spec.Domain.Memory != nil:
		vmiBaseMemory = vmi.Spec.Domain.Memory.Guest
	}

	memlockSize.Add(*resource.NewScaledQuantity(vmiBaseMemory.ScaledValue(resource.Kilo), resource.Kilo))

	if err := setProcessMemoryLockRLimit(qemuProcessID, memlockSize.Value()); err != nil {
		return fmt.Errorf("failed to set process %d memlock rlimit to %d: %v", qemuProcessID, memlockSize.Value(), err)
	}
	log.Log.V(5).Object(vmi).Infof("set process %+v memlock rlimits to: Cur: %[2]d Max:%[2]d",
		targetProcess, memlockSize.Value())

	return nil
}

func (k *KvmVirtRuntime) AdjustResources(podIsoDetector isolation.PodIsolationDetector, vmi *v1.VirtualMachineInstance, config *v1.KubeVirtConfiguration) error {
	err := k.adjustQemuProcessMemoryLimits(podIsoDetector, vmi, config.AdditionalGuestMemoryOverheadRatio)
	if err != nil {
		return fmt.Errorf("Unable to adjust qemu process memory limits for VMI %s: %w", vmi.Name, err)
	}
	return nil
}

func (k *KvmVirtRuntime) configureHousekeepingCgroup(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager, domain *api.Domain) error {
	if err := cgroupManager.CreateChildCgroup("housekeeping", "cpuset"); err != nil {
		k.logger.Reason(err).Error("CreateChildCgroup ")
		return err
	}

	// bail out if domain does not exist
	if domain == nil {
		return nil
	}

	if domain.Spec.CPUTune == nil || domain.Spec.CPUTune.EmulatorPin == nil {
		return nil
	}

	hkcpus, err := hardware.ParseCPUSetLine(domain.Spec.CPUTune.EmulatorPin.CPUSet, 100)
	if err != nil {
		return err
	}

	k.logger.V(3).Object(vmi).Infof("housekeeping cpu: %v", hkcpus)

	err = cgroupManager.SetCpuSet("housekeeping", hkcpus)
	if err != nil {
		return err
	}

	tids, err := cgroupManager.GetCgroupThreads()
	if err != nil {
		return err
	}
	hktids := make([]int, 0, 10)

	for _, tid := range tids {
		proc, err := ps.FindProcess(tid)
		if err != nil {
			k.logger.Object(vmi).Errorf("Failure to find process: %s", err.Error())
			return err
		}
		if proc == nil {
			return fmt.Errorf("failed to find process with tid: %d", tid)
		}
		comm := proc.Executable()
		if strings.Contains(comm, "CPU ") && strings.Contains(comm, "KVM") {
			continue
		}
		hktids = append(hktids, tid)
	}

	k.logger.V(3).Object(vmi).Infof("hk thread ids: %v", hktids)
	for _, tid := range hktids {
		err = cgroupManager.AttachTID("cpuset", "housekeeping", tid)
		if err != nil {
			k.logger.Object(vmi).Errorf("Error attaching tid %d: %v", tid, err.Error())
			return err
		}
	}

	return nil
}

// configureRealTimeVCPUs parses the realtime mask value and configured the selected vcpus
// for real time workloads by setting the scheduler to FIFO and process priority equal to 1.
func (k *KvmVirtRuntime) configureVCPUScheduler(vmi *v1.VirtualMachineInstance) error {
	qemuProcess, err := k.GetQEMUProcess(k.podIsolationDetector, vmi)
	if err != nil {
		return err
	}
	vcpus, err := getVCPUThreadIDs(qemuProcess.Pid())
	if err != nil {
		return err
	}
	mask, err := parseCPUMask(vmi.Spec.Domain.CPU.Realtime.Mask)
	if err != nil {
		return err
	}
	for vcpuID, threadID := range vcpus {
		if mask.isEnabled(vcpuID) {
			param := schedParam{priority: 1}
			tid, err := strconv.Atoi(threadID)
			if err != nil {
				return err
			}
			err = schedSetScheduler(tid, schedFIFO, param)
			if err != nil {
				return fmt.Errorf("failed to set FIFO scheduling and priority 1 for thread %d: %w", tid, err)
			}
		}
	}
	return nil
}

func (k *KvmVirtRuntime) KvmPitPid(vmi *v1.VirtualMachineInstance) (int, error) {
	qemuprocess, err := k.GetQEMUProcess(k.podIsolationDetector, vmi)
	if err != nil {
		return -1, err
	}
	processes, _ := ps.Processes()
	nspid, err := GetNspid(qemuprocess.Pid())
	if err != nil || nspid == -1 {
		return -1, err
	}
	pitstr := "kvm-pit/" + strconv.Itoa(nspid)

	for _, process := range processes {
		if process.Executable() == pitstr {
			return process.Pid(), nil
		}
	}
	return -1, nil
}

func (k *KvmVirtRuntime) affinePitThread(vmi *v1.VirtualMachineInstance) error {
	var Mask unix.CPUSet
	Mask.Zero()
	qemuprocess, err := k.GetQEMUProcess(k.podIsolationDetector, vmi)
	if err != nil {
		return err
	}
	qemupid := qemuprocess.Pid()
	if qemupid == -1 {
		return nil
	}

	pitpid, err := k.KvmPitPid(vmi)
	if err != nil {
		return err
	}
	if pitpid == -1 {
		return nil
	}
	if vmi.IsRealtimeEnabled() {
		param := schedParam{priority: 2}
		err = schedSetScheduler(pitpid, schedFIFO, param)
		if err != nil {
			return fmt.Errorf("failed to set FIFO scheduling and priority 2 for thread %d: %w", pitpid, err)
		}
	}
	vcpus, err := getVCPUThreadIDs(qemupid)
	if err != nil {
		return err
	}
	vpid, ok := vcpus["0"]
	if ok == false {
		return nil
	}
	vcpupid, err := strconv.Atoi(vpid)
	if err != nil {
		return err
	}
	err = unix.SchedGetaffinity(vcpupid, &Mask)
	if err != nil {
		return err
	}
	return unix.SchedSetaffinity(pitpid, &Mask)
}

func isVCPU(comm []byte) (string, bool) {
	if !kvmVcpuRegex.MatchString(string(comm)) {
		return "", false
	}
	v := kvmVcpuRegex.FindSubmatch(comm)
	return string(v[1]), true
}

func getVCPUThreadIDs(pid int) (map[string]string, error) {
	p := filepath.Join(string(os.PathSeparator), "proc", strconv.Itoa(pid), "task")
	d, err := os.ReadDir(p)
	if err != nil {
		return nil, err
	}
	ret := map[string]string{}
	for _, f := range d {
		if f.IsDir() {
			c, err := os.ReadFile(filepath.Join(p, f.Name(), "comm"))
			if err != nil {
				return nil, err
			}
			if v, ok := isVCPU(c); ok {
				ret[v] = f.Name()
			}
		}
	}
	return ret, nil
}
