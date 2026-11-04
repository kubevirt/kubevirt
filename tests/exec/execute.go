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
 * Copyright The KubeVirt Authors.
 *
 */

package exec

import (
	"bytes"
	"context"
	"fmt"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/streaming/pkg/httpstream"

	"kubevirt.io/client-go/kubecli"
)

func ExecuteCommandOnPod(pod *k8sv1.Pod, containerName string, command []string) (string, error) {
	return ExecuteCommandOnPodWithContext(context.Background(), pod, containerName, command)
}

// ExecuteCommandOnPodWithContext bounds both websocket and SPDY execution by ctx.
func ExecuteCommandOnPodWithContext(ctx context.Context, pod *k8sv1.Pod, containerName string, command []string) (string, error) {
	stdout, stderr, err := executeCommandOnPodWithResults(ctx, pod, containerName, command)

	if err != nil {
		return "", fmt.Errorf("failed executing command on pod: %w: stderr %v: stdout: %v", err, stderr, stdout)
	}

	if len(stderr) > 0 {
		return "", fmt.Errorf("stderr: %v", stderr)
	}

	return stdout, nil
}

func ExecuteCommandOnPodWithResults(pod *k8sv1.Pod, containerName string, command []string) (stdout, stderr string, err error) {
	return executeCommandOnPodWithResults(context.Background(), pod, containerName, command)
}

func executeCommandOnPodWithResults(ctx context.Context, pod *k8sv1.Pod, containerName string, command []string) (stdout, stderr string, err error) {
	var (
		stdoutBuf bytes.Buffer
		stderrBuf bytes.Buffer
	)
	options := remotecommand.StreamOptions{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Tty:    false,
	}
	err = executeCommandOnPodWithOptions(ctx, pod, containerName, command, options)
	return stdoutBuf.String(), stderrBuf.String(), err
}

func ExecuteCommandOnPodWithOptions(pod *k8sv1.Pod, containerName string, command []string, options remotecommand.StreamOptions) error {
	return executeCommandOnPodWithOptions(context.Background(), pod, containerName, command, options)
}

func executeCommandOnPodWithOptions(ctx context.Context, pod *k8sv1.Pod, containerName string, command []string, options remotecommand.StreamOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	req := kubevirt.Client().CoreV1().RESTClient().Post().
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

	var spdyExecutor remotecommand.Executor
	if ctx.Done() == nil {
		spdyExecutor, err = remotecommand.NewSPDYExecutor(virtConfig, "POST", req.URL())
	} else {
		spdyExecutor, err = newCancellableSPDYExecutor(virtConfig, "POST", req.URL())
	}
	if err != nil {
		return err
	}
	wsExecutor, err := remotecommand.NewWebSocketExecutor(virtConfig, "POST", req.URL().String())
	if err != nil {
		return err
	}
	executor, err := remotecommand.NewFallbackExecutor(wsExecutor, spdyExecutor, httpstream.IsUpgradeFailure)
	if err != nil {
		return err
	}

	err = executor.StreamWithContext(ctx, options)
	if err != nil && ctx.Err() != nil {
		return fmt.Errorf("%w: %v", ctx.Err(), err)
	}
	return err
}
