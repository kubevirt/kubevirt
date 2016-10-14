package services

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	"io"
	"kubevirt/core/pkg/precond"
	"os/exec"
)

type KubeService interface {
	CreatePod(manifest io.Reader) error
	DeletePodByLabel(key string, value string) error
	DeletePodByName(name string) error
}

type kubeService struct {
	logger    levels.Levels
	ApiServer string
}

func (k *kubeService) CreatePod(manifest io.Reader) error {
	precond.MustNotBeNil(manifest)
	cmd := exec.Command("kubectl", "-s", k.ApiServer, "create", "-f", "-")
	cmd.Stdin = manifest
	return runCMD(cmd)
}

func (k *kubeService) DeletePodByLabel(key string, value string) error {
	precond.MustNotBeEmpty(key)
	precond.MustNotBeEmpty(value)
	cmd := exec.Command("kubectl", "-s", k.ApiServer, "delete", "pods", "-l", fmt.Sprintf("%s=%s", key, value))
	return runCMD(cmd)
}

func (k *kubeService) DeletePodByName(name string) error {
	precond.MustNotBeEmpty(name)
	cmd := exec.Command("kubectl", "-s", k.ApiServer, "delete", "pods", name)
	return runCMD(cmd)
}

func NewKubeService(logger log.Logger, apiServer string) KubeService {
	svc := kubeService{
		logger:    levels.New(logger).With("component", "KubeService"),
		ApiServer: apiServer,
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
