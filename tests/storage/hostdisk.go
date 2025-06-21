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

package storage

import (
	"fmt"
	"path/filepath"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
)

func CreateHostDisk(diskPath string) *k8sv1.Pod {
	hostPathType := k8sv1.HostPathDirectoryOrCreate
	dir := filepath.Dir(diskPath)

	command := fmt.Sprintf(`dd if=/dev/zero of=%s bs=1 count=0 seek=1G && ls -l %s`, diskPath, dir)
	if !checks.IsFeatureEnabled(featuregate.Root) {
		command = command + fmt.Sprintf(" && chown 107:107 %s", diskPath)
	}

	args := []string{command}
	pod := libpod.RenderHostPathPod("hostdisk-create-job", dir, hostPathType, k8sv1.MountPropagationNone, []string{"/bin/bash", "-c"}, args)
	return pod
}

func RemoveHostDisk(diskPath string, nodeName string) error {
	virtClient := kubevirt.Client()
	procPath := filepath.Join("/proc/1/root", diskPath)

	virtHandlerPod, err := libnode.GetVirtHandlerPod(virtClient, nodeName)
	if err != nil {
		return err
	}

	_, _, err = exec.ExecuteCommandOnPodWithResults(virtHandlerPod, "virt-handler", []string{"rm", "-rf", procPath})
	return err
}

func RandHostDiskDir() string {
	const tmpPath = "/var/provision/kubevirt.io/tests"
	return filepath.Join(tmpPath, rand.String(10))
}
