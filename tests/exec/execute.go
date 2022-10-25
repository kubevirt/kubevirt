/*
 * This file is part of the kubevirt project
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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package exec

import (
	"bytes"
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"kubevirt.io/client-go/kubecli"
)

func ExecuteCommandOnPod(virtCli kubecli.KubevirtClient, pod *k8sv1.Pod, containerName string, command []string) (string, error) {
	stdout, stderr, err := ExecuteCommandOnPodWithResults(virtCli, pod, containerName, command)

	if err != nil {
		return "", fmt.Errorf("failed executing command on pod: %v: stderr %v: stdout: %v", err, stderr, stdout)
	}

	if len(stderr) > 0 {
		return "", fmt.Errorf("stderr: %v", stderr)
	}

	return stdout, nil
}

func ExecuteCommandOnPodWithResults(virtCli kubecli.KubevirtClient, pod *k8sv1.Pod, containerName string, command []string) (stdout, stderr string, err error) {
	var (
		stdoutBuf bytes.Buffer
		stderrBuf bytes.Buffer
	)
	options := remotecommand.StreamOptions{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Tty:    false,
	}
	err = ExecuteCommandOnPodWithOptions(virtCli, pod, containerName, command, options)
	return stdoutBuf.String(), stderrBuf.String(), err
}

func ExecuteCommandOnPodWithOptions(virtCli kubecli.KubevirtClient, pod *k8sv1.Pod, containerName string, command []string, options remotecommand.StreamOptions) error {
	req := virtCli.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		Param("container", containerName)

	req.VersionedParams(&k8sv1.PodExecOptions{
		Container: containerName,
		Command:   command,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, scheme.ParameterCodec)

	virtConfig, err := kubecli.GetKubevirtClientConfig()
	if err != nil {
		return err
	}

	executor, err := remotecommand.NewSPDYExecutor(virtConfig, "POST", req.URL())
	if err != nil {
		return err
	}

	return executor.Stream(options)
}
