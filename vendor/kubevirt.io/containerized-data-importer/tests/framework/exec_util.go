/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// ExecOptions passed to ExecWithOptions
type ExecOptions struct {
	Command []string

	Namespace     string
	PodName       string
	ContainerName string

	Stdin         io.Reader
	CaptureStdout bool
	CaptureStderr bool
	// If false, whitespace in std{err,out} will be removed.
	PreserveWhitespace bool
}

// ExecWithOptions executes a command in the specified container,
// returning stdout, stderr and error. `options` allowed for
// additional parameters to be passed.
func (f *Framework) ExecWithOptions(options ExecOptions) (string, string, error) {
	config := f.RestConfig

	req := f.K8sClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(options.PodName).
		Namespace(options.Namespace).
		SubResource("exec").
		Param("container", options.ContainerName)
	req.VersionedParams(&v1.PodExecOptions{
		Container: options.ContainerName,
		Command:   options.Command,
		Stdin:     options.Stdin != nil,
		Stdout:    options.CaptureStdout,
		Stderr:    options.CaptureStderr,
		TTY:       false,
	}, scheme.ParameterCodec)

	var stdout, stderr bytes.Buffer
	err := execute("POST", req.URL(), config, options.Stdin, &stdout, &stderr, false)

	if options.PreserveWhitespace {
		return stdout.String(), stderr.String(), err
	}
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

// ExecCommandInContainerWithFullOutput executes a command in the
// specified container and return stdout, stderr and error
func (f *Framework) ExecCommandInContainerWithFullOutput(namespace, podName, containerName string, cmd ...string) (string, string, error) {
	return f.ExecWithOptions(ExecOptions{
		Command:       cmd,
		Namespace:     namespace,
		PodName:       podName,
		ContainerName: containerName,

		Stdin:              nil,
		CaptureStdout:      true,
		CaptureStderr:      true,
		PreserveWhitespace: false,
	})
}

// ExecCommandInContainer executes a command in the specified container.
func (f *Framework) ExecCommandInContainer(namespace, podName, containerName string, cmd ...string) string {
	stdout, _, err := f.ExecCommandInContainerWithFullOutput(namespace, podName, containerName, cmd...)
	if err != nil {
		fmt.Fprintf(ginkgo.GinkgoWriter, "[WARN] error executing command %q, error: %s\n", cmd, err.Error())
	}
	return stdout
}

// ExecShellInContainer provides a function to execute a shell cmd for the specified running container in a pod
func (f *Framework) ExecShellInContainer(podName, containerName string, cmd string) string {
	return f.ExecCommandInContainer(podName, containerName, "/bin/sh", "-c", cmd)
}

// ExecCommandInPod provides a function to execute a command on a running pod
func (f *Framework) ExecCommandInPod(podName, namespace string, cmd ...string) string {
	pod, err := f.K8sClient.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to get pod")
	gomega.Expect(pod.Spec.Containers).NotTo(gomega.BeEmpty())
	return f.ExecCommandInContainer(namespace, podName, pod.Spec.Containers[0].Name, cmd...)
}

// ExecCommandInPodWithFullOutput provides a function to execute a command in a running pod and to capture its output
func (f *Framework) ExecCommandInPodWithFullOutput(namespace, podName string, cmd ...string) (string, string, error) {
	pod, err := f.K8sClient.CoreV1().Pods(f.Namespace.GetName()).Get(podName, metav1.GetOptions{})
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to get pod")
	gomega.Expect(pod.Spec.Containers).NotTo(gomega.BeEmpty())
	return f.ExecCommandInContainerWithFullOutput(namespace, podName, pod.Spec.Containers[0].Name, cmd...)
}

// ExecShellInPod provides a function to execute a shell cmd in the specified pod
func (f *Framework) ExecShellInPod(podName, namespace string, cmd string) string {
	return f.ExecCommandInPod(podName, namespace, "/bin/sh", "-c", cmd)
}

// ExecShellInPodWithFullOutput provides a function to execute a shell cmd in a running pod and to capture its output
func (f *Framework) ExecShellInPodWithFullOutput(podName string, cmd string) (string, string, error) {
	return f.ExecCommandInPodWithFullOutput(podName, "/bin/sh", "-c", cmd)
}

func execute(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}
	return exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	})
}
