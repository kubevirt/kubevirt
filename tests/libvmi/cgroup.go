package libvmi

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"kubevirt.io/client-go/log"

	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/testsuite"

	hw_utils "kubevirt.io/kubevirt/pkg/util/hardware"

	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/tests/libnode"

	k8sv1 "k8s.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"
	"kubevirt.io/kubevirt/tests/exec"
)

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

	return "", fmt.Errorf("unknown error occurred: %v", err)
}

// subsystem is ignored if empty
func SearchCgroupByName(nodeName, basePath, subsystem, cgroupName string) (cgroupPath string, err error) {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return "", err
	}

	if subsystem != "" {
		basePath = filepath.Join(basePath, subsystem)
	}

	findCmd := []string{"bash", "-c", fmt.Sprintf("find %s -regex .*%s.* -type d", basePath, cgroupName)}

	cgroupPath, stderr, err := libnode.ExecuteCommandOnNodeThroughVirtHandler(virtClient, nodeName, findCmd)
	if err != nil || stderr != "" {
		err = fmt.Errorf("cannot find dedicated cgroup path. err: %v. stderr: %s", err, stderr)
		return
	}

	cgroupPath = strings.TrimSpace(cgroupPath)

	if cgroupPath == "" {
		err = fmt.Errorf("couldn't find %s directory", services.DedicatedCpusContainerName)
		return
	}

	return
}

func GetCgroupByContainer(pod *k8sv1.Pod, containerName, subsystem string) (cgroupPath string, err error) {
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

	cgroupDirName := "crio-" + strings.TrimPrefix(containerStatus.ContainerID, "cri-o://")

	return SearchCgroupByName(pod.Spec.NodeName, cgroup.HostCgroupBasePath, subsystem, cgroupDirName)
}

func GetCpuset(cgroupPath, nodeName string, cgroupVersion cgroup.CgroupVersion) (cpuset []int, err error) {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return
	}

	var targetFile string
	if cgroupVersion == cgroup.V1 {
		targetFile = "cpuset.cpus"
	} else {
		targetFile = "cpuset.cpus.effective"
	}

	cmd := []string{"bash", "-c", "cat " + filepath.Join(cgroupPath, targetFile)}
	stdout, stderr, err := libnode.ExecuteCommandOnNodeThroughVirtHandler(virtClient, nodeName, cmd)
	if err != nil || stderr != "" {
		err = fmt.Errorf("err: %v. stderr: %s", err, stderr)
		return
	}

	if stdout == "" {
		err = fmt.Errorf("stdout is empty")
		return
	}

	stdout = strings.TrimSpace(stdout)
	stdout = strings.ReplaceAll(stdout, "\n", "")

	return hw_utils.ParseCPUSetLine(stdout, 5000)
}

func GetComputeCpuset(pod *k8sv1.Pod) (cpuset []int, err error) {
	cgroupVersion, err := GetPodCgroupVersion(pod)
	if err != nil {
		return nil, err
	}

	subsystem := ""
	if cgroupVersion == cgroup.V1 {
		subsystem = cgroup.CgroupSubsystemCpuset
	}

	cgroupPath, err := GetCgroupByContainer(pod, "compute", subsystem)
	if err != nil {
		return nil, err
	}

	return GetCpuset(cgroupPath, pod.Spec.NodeName, cgroupVersion)
}

func ListCgroupThreads(pod *k8sv1.Pod, taskType cgroup.TaskType, subsystem string) (ids []int, err error) {
	return ListCgroupThreadsFromContainer(pod, "compute", taskType, subsystem)
}

// taskType is ignoed for cgroup v1
func ListCgroupThreadsFromContainer(pod *k8sv1.Pod, containerName string, taskType cgroup.TaskType, subsystem string) (ids []int, err error) {
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
		targetFile = filepath.Join(cgroup.BasePath, subsystem, "tasks")
	} else {
		switch taskType {
		case cgroup.Thread:
			targetFile = filepath.Join(cgroup.BasePath, "cgroup.threads")
		case cgroup.Process:
			targetFile = filepath.Join(cgroup.BasePath, "cgroup.procs")
		default:
			return nil, fmt.Errorf("illegal task type: %v", taskType)
		}
	}

	output, err := exec.ExecuteCommandOnPod(
		virtClient,
		pod,
		containerName,
		[]string{"cat", targetFile},
	)
	if err != nil {
		return nil, err
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
