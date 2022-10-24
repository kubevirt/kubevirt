package virthandler

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"kubevirt.io/kubevirt/pkg/util"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"
)

func logErr(err error, msg string) {
	if err != nil {
		log.Log.Infof("ihol3 ERROR %s: %v", msg, err)
	}
}

func findSleepPid(vmiAnnotations map[string]string) int {
	sleepPid := -1
	bashCmd := fmt.Sprintf(`ps -A -o pid,args | sed 's@  *@ @g'`)

	// For easier debugging, allow custom command
	if vmiAnnotations != nil {
		customCmd, exists := vmiAnnotations["customCmd"]
		if exists {
			bashCmd = customCmd
		}
	}

	log.Log.Infof("ihol3 command: %s", bashCmd)
	out, err := exec.Command("bash", "-c", bashCmd).CombinedOutput()
	if err != nil {
		logErr(err, "ps")
		return sleepPid
	}
	log.Log.Infof("ihol3 ps output: %s", string(out))

	for _, psOutputLine := range strings.Split(string(out), "\n") {
		if strings.Contains(psOutputLine, "sleep "+services.SleepMagicNumber) {
			psOutputLine = strings.TrimSpace(psOutputLine)
			log.Log.Infof("ihol3 chosen process line: %s", psOutputLine)

			sleepPid, err = strconv.Atoi(strings.Split(psOutputLine, " ")[0])
			if err != nil {
				logErr(err, "ps atoi")
				return sleepPid
			}
		}
	}

	if sleepPid < 0 {
		err := fmt.Errorf("cannot find sleep %s process ", services.SleepMagicNumber)
		logErr(err, "find sleep")
		return sleepPid
	}

	return sleepPid
}

func findCgroupPaths(origVMI *v1.VirtualMachineInstance, sleepPid int) (computeSlicePath, dedicatedCpuSlicePath string) {
	if sleepPid < 0 {
		return
	}

	cgroupManager, err := cgroup.NewManagerFromVM(origVMI)
	if err != nil {
		return
	}

	computeSlicePath, err = cgroupManager.GetBasePathToHostSubsystem("cpuset")
	if err != nil {
		logErr(err, "find cpuset path")
		return
	}
	log.Log.Infof("ihol3 computeCpuPath: %s", computeSlicePath)

	podSlicePath := filepath.Join(computeSlicePath, "..")
	dirs, err := os.ReadDir(podSlicePath)
	if err != nil {
		logErr(err, "read dit")
		return
	}

	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		procsPath := filepath.Join(podSlicePath, dir.Name(), "cgroup.procs")
		procsContents, err := ioutil.ReadFile(procsPath)
		if err != nil {
			logErr(err, fmt.Sprintf("error reading file %s", procsPath))
		}

		if strings.Contains(string(procsContents), strconv.Itoa(sleepPid)) {
			dedicatedCpuSlicePath = filepath.Join(podSlicePath, dir.Name())
			break
		}
	}

	if dedicatedCpuSlicePath == "" {
		err := fmt.Errorf("dedicated CPU slice wasn't found")
		logErr(err, "find dedicated cpu slice")
		return
	}

	log.Log.Infof("ihol3 dedicated CPU slice found: %s", dedicatedCpuSlicePath)
	return
}

func getVcpuThreads(computeSlicePath string) (vCpuTids []int) {
	if computeSlicePath == "" {
		return
	}

	// Find vCPU threads
	const bashCmd = "ps -TA | sed 's@  *@ @g'"
	vCpuTids = make([]int, 0, 3)

	log.Log.Infof("ihol3 command: %s", bashCmd)
	out, err := exec.Command("bash", "-c", bashCmd).CombinedOutput()
	if err != nil {
		logErr(err, "ps threads")
		return
	}
	log.Log.Infof("ihol3 ps threads output: %s", string(out))

	for _, psOutputLine := range strings.Split(string(out), "\n") {
		if strings.Contains(psOutputLine, "CPU ") || strings.Contains(psOutputLine, "KVM") {
			psOutputLine = strings.TrimSpace(psOutputLine)
			log.Log.Infof("ihol3 chosen process line: %s", psOutputLine)

			tid, err := strconv.Atoi(strings.Split(psOutputLine, " ")[1])
			if err != nil {
				logErr(err, "ps thread atoi")
				return
			}

			vCpuTids = append(vCpuTids, tid)
		}
	}

	log.Log.Infof("ihol3 VCPU Tids: %+v", vCpuTids)

	return
}

func createThreadedChildCgroupForV2(slicePath string) error {
	newGroupPath := filepath.Join(slicePath, "vcpu-threaded")
	if _, err := os.Stat(newGroupPath); !errors.Is(err, os.ErrNotExist) {
		return nil
	}

	// Write "+subsystem" to cgroup.subtree_control
	const childCgroupSubSystem = "cpuset"
	wVal := "+" + childCgroupSubSystem
	err := cgroups.WriteFile(slicePath, "cgroup.subtree_control", wVal)
	if err != nil {
		logErr(err, "edit cgroup.subtree_control")
		return err
	}

	// Create new cgroup directory
	err = util.MkdirAllWithNosec(newGroupPath)
	if err != nil {
		log.Log.Infof("mkdir %s failed", newGroupPath)
		return err
	}

	// Enable threaded cgroup controller
	err = cgroups.WriteFile(newGroupPath, "cgroup.type", "threaded")
	if err != nil {
		return err
	}

	// Write "+subsystem" to newcgroup/cgroup.subtree_control
	wVal = "+" + childCgroupSubSystem
	err = cgroups.WriteFile(newGroupPath, "cgroup.subtree_control", wVal)
	if err != nil {
		return err
	}
	return nil
}

func attachVcpusToCgroup(dedicatedCpuSlicePath string, vCpuTids []int) {
	if dedicatedCpuSlicePath == "" || len(vCpuTids) == 0 {
		return
	}

	threadsFile := "tasks"
	if cgroups.IsCgroup2UnifiedMode() {
		threadsFile = "cgroup.threads"
	}

	for _, vcpuTid := range vCpuTids {
		err := cgroups.WriteFile(dedicatedCpuSlicePath, threadsFile, strconv.Itoa(vcpuTid))
		if err != nil {
			logErr(err, "move vCPU")
			return
		}
	}

	log.Log.Infof("ihol3 DONE SUCCESSFULLY!!!!!!!!!!!!!!")
}

func (d *VirtualMachineController) configureDedicatedCPUCgroup(origVMI *v1.VirtualMachineInstance) {

	// Here's an overview of how the attachment of the vCPUs to the dedicated-cpu cgroup is performed.
	// You're welcome to look inside the helper functions to see the implementation.
	//
	// BE AWARE that this is a POC that doesn't fit production use!
	//
	// 1. The dedicated-cpu container runs a "sleep <SleepMagicNumber>" process. First, we need to find this process' PID.
	// 2. We will iterate on each of the Pod's sub-cgroup hierarchy and will check if the found PID exists in this cgroup.
	//	  If the PID exists, this is the dedicated-cpu cgroup.
	// 3. In the compute cgroup, we'll locate the vCPU threads and their TID (=thread ID)
	// 4. The thread IDs will be added to the dedicated-cpu cgroup.

	sleepPid := findSleepPid(origVMI.Annotations)

	computeSlicePath, dedicatedCpuSlicePath := findCgroupPaths(origVMI, sleepPid)

	vCpuTids := getVcpuThreads(computeSlicePath)

	attachVcpusToCgroup(dedicatedCpuSlicePath, vCpuTids)
}
