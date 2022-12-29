package libvmi

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"kubevirt.io/kubevirt/tests/libnode"

	"kubevirt.io/client-go/log"

	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/testsuite"

	hw_utils "kubevirt.io/kubevirt/pkg/util/hardware"

	k8sv1 "k8s.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"
	"kubevirt.io/kubevirt/tests/exec"
)

// subsystem is ignored if empty
func SearchCgroupByName(nodeName, basePath, subsystem, cgroupName string) (cgroupPath string, err error) {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return "", err
	}

	if subsystem != "" {
		basePath = filepath.Join(basePath, subsystem)
	}

	findCmd := []string{"bash", "-c", fmt.Sprintf("find %s -regex .*%s$ -type d", basePath, cgroupName)}

	cgroupPath, stderr, err := libnode.ExecuteCommandOnNodeThroughVirtHandler(virtClient, nodeName, findCmd)
	if err != nil || stderr != "" {
		err = fmt.Errorf("cannot find dedicated cgroup path. err: %v. stderr: %s", err, stderr)
		return
	}

	cgroupPath = strings.TrimSpace(cgroupPath)

	if cgroupPath == "" {
		err = fmt.Errorf("couldn't find %s directory", cgroupName)
		return
	}

	return
}

func GetCgroupByContainer(pod *k8sv1.Pod, containerName, subsystem string, subCgroups ...string) (cgroupPath string, err error) {
	var containerStatus k8sv1.ContainerStatus
	for _, c := range pod.Status.ContainerStatuses {
		if c.Name == containerName {
			containerStatus = c
		}
	}

	if containerStatus.Name == "" {
		err = fmt.Errorf("couldn't find container %s for pod %s", containerName, pod.Name)
		return
	}

	cgroupVersion, err := GetPodCgroupVersion(pod)
	if err != nil {
		return
	}

	cgroupDirName := "crio-" + strings.TrimPrefix(containerStatus.ContainerID, "cri-o://")
	if cgroupVersion == cgroup.V1 {
		cgroupDirName += ".scope"
	}

	for _, subCgroup := range subCgroups {
		cgroupDirName = filepath.Join(cgroupDirName, subCgroup)
	}

	return SearchCgroupByName(pod.Spec.NodeName, cgroup.HostCgroupBasePath, subsystem, cgroupDirName)
}

func GetVmiCgroupVersion(vmi *k6tv1.VirtualMachineInstance) (cgroup.CgroupVersion, error) {
	pod, err := GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
	if err != nil {
		return "", err
	}

	return GetPodCgroupVersion(pod)
}

func GetPodCgroupVersion(pod *k8sv1.Pod) (cgroup.CgroupVersion, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return "", err
	}

	containerName := ""
	for _, container := range pod.Spec.Containers {
		if container.Name == "compute" {
			containerName = container.Name
			break
		}
	}

	if containerName == "" {
		containerName = pod.Spec.Containers[0].Name
	}

	const cgroupV2OnlyFile = "/sys/fs/cgroup/cgroup.controllers"

	command := []string{"bash", "-c", "cat " + cgroupV2OnlyFile}

	_, err = exec.ExecuteCommandOnPod(virtClient, pod, containerName, command)
	if err == nil {
		return cgroup.V2, nil
	}

	if strings.Contains(err.Error(), "No such file or directory") {
		return cgroup.V1, nil
	}

	return "", fmt.Errorf("unknown error occured: %v", err)
}

func GetCpusetWithSubPath(pod *k8sv1.Pod, containerName, subCgroupPath string) (cpuset []int, err error) {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return
	}

	cgroupVersion, err := GetPodCgroupVersion(pod)
	if err != nil {
		return nil, err
	}

	var targetFile string
	if cgroupVersion == cgroup.V1 {
		targetFile = "cpuset.cpus"
	} else {
		targetFile = "cpuset.cpus.effective"
	}

	subsystem := ""
	if cgroupVersion == cgroup.V1 {
		subsystem = cgroup.CgroupSubsystemCpuset
	}

	output, err := exec.ExecuteCommandOnPod(
		virtClient,
		pod,
		containerName,
		[]string{"cat", filepath.Join(cgroup.BasePath, subsystem, subCgroupPath, targetFile)},
	)

	if output == "" {
		err = fmt.Errorf("stdout is empty")
		return
	}

	output = strings.TrimSpace(output)
	output = strings.ReplaceAll(output, "\n", "")

	return hw_utils.ParseCPUSetLine(output, 5000)
}

func GetCpuset(pod *k8sv1.Pod, containerName string) (cpuset []int, err error) {
	return GetCpusetWithSubPath(pod, containerName, "")
}

func GetComputeCpuset(pod *k8sv1.Pod) (cpuset []int, err error) {
	return GetCpuset(pod, "compute")
}

func ListComputeCgroupThreads(pod *k8sv1.Pod, taskType cgroup.TaskType, subsystem string) (ids []int, err error) {
	return ListCgroupThreadsFromContainer(pod, "compute", taskType, subsystem)
}

// taskType is ignoed for cgroup v1
func ListCgroupThreadsFromContainer(pod *k8sv1.Pod, containerName string, taskType cgroup.TaskType, subsystem string, subCgroups ...string) (ids []int, err error) {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return
	}

	cgroupVersion, err := GetPodCgroupVersion(pod)
	if err != nil {
		return nil, err
	}

	var targetFile string

	if taskType == cgroup.Process {
		targetFile = "cgroup.procs"
	} else {
		if cgroupVersion == cgroup.V1 {
			targetFile = "tasks"
		} else {
			targetFile = "cgroup.threads"
		}
	}

	if cgroupVersion == cgroup.V2 {
		// subsystem is ignored on v2
		subsystem = ""
	}

	cgroupPath, err := GetCgroupByContainer(pod, containerName, subsystem, subCgroups...)
	if err != nil {
		return nil, err
	}

	output, stderr, err := libnode.ExecuteCommandOnNodeThroughVirtHandler(virtClient, pod.Spec.NodeName, []string{"cat", filepath.Join(cgroupPath, targetFile)})
	if err != nil || stderr != "" {
		err = fmt.Errorf("err: %v. stderr: %s", err, stderr)
		return
	}

	var pids []int
	output = strings.TrimSpace(output)
	for _, pidStr := range strings.Split(output, "\n") {
		pidInt, err := strconv.Atoi(pidStr)
		if err != nil {
			log.Log.Warningf("cannot convert string PID to int: %s", pidStr)
			continue
		}

		pids = append(pids, pidInt)
	}

	return pids, nil
}
