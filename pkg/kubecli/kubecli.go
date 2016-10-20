package kubecli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"kubevirt/core/pkg/kubecli/v1.2"
	"kubevirt/core/pkg/precond"
	"os/exec"
)

type runHelper func(cmd *exec.Cmd) error

type KubeCli interface {
	CreatePod(manifest io.Reader) error
	GetPod(name string) (*v1_2.Pod, error)
	GetPodsByLabel(key string, value string) ([]v1_2.Pod, error)
	DeletePodsByLabel(key string, value string) error
	DeletePod(name string) error
}

type kubeCli struct {
	ApiServer string
	runHelper runHelper
}

func (k *kubeCli) CreatePod(manifest io.Reader) error {
	precond.MustNotBeNil(manifest)
	cmd := exec.Command("kubectl", "-s", k.ApiServer, "create", "-f", "-")
	cmd.Stdin = manifest
	return runCMD(cmd)
}

func (k *kubeCli) DeletePodsByLabel(key string, value string) error {
	precond.MustNotBeEmpty(key)
	precond.MustNotBeEmpty(value)
	cmd := exec.Command("kubectl", "-s", k.ApiServer, "delete", "pods", "-l", fmt.Sprintf("%s=%s", key, value))
	return runCMD(cmd)
}

func (k *kubeCli) DeletePod(name string) error {
	precond.MustNotBeEmpty(name)
	cmd := exec.Command("kubectl", "-s", k.ApiServer, "delete", "pods", name)
	return runCMD(cmd)
}

func (k *kubeCli) GetPod(podName string) (*v1_2.Pod, error) {
	cmd := exec.Command("kubectl", "-s", k.ApiServer, "get", "pods", podName, "-o", "json")
	stdout := new(bytes.Buffer)
	cmd.Stdout = stdout
	err := runCMD(cmd)
	if err != nil {
		return nil, err
	}
	pod := v1_2.Pod{}
	if err := json.NewDecoder(stdout).Decode(&pod); err == io.EOF {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &pod, nil
}

func (k *kubeCli) GetPodsByLabel(key string, value string) ([]v1_2.Pod, error) {
	cmd := exec.Command("kubectl", "-s", k.ApiServer, "get", "pods", "-l", fmt.Sprintf("%s=%s", key, value), "-o", "json")
	stdout := new(bytes.Buffer)
	cmd.Stdout = stdout
	err := runCMD(cmd)
	if err != nil {
		return nil, err
	}
	pods := v1_2.PodList{}
	if err := json.NewDecoder(stdout).Decode(&pods); err == io.EOF {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

func NewKubeCli(apiServer string) KubeCli {
	svc := kubeCli{
		ApiServer: apiServer,
		runHelper: runCMD,
	}
	return &svc
}

func runCMD(cmd *exec.Cmd) error {
	stderrBuffer := new(bytes.Buffer)
	cmd.Stderr = stderrBuffer
	if err := cmd.Run(); err != nil {
		stderr := stderrBuffer.String()
		if stderr != "" {
			return errors.New(stderr)
		} else {
			return err
		}
	}
	return nil
}
