package services

import (
	"bytes"
	"errors"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	"kubevirt/core/pkg/entities"
	"os/exec"
)

type KubeService interface {
	StartPod(vm *entities.VM, domainXML []byte) error
}

type kubeService struct {
	logger          levels.Levels
	TemplateService TemplateService `inject:""`
	ApiServer       string
}

func (k *kubeService) StartPod(vm *entities.VM, domainXML []byte) error {
	stdinBuffer := new(bytes.Buffer)
	stderrBuffer := new(bytes.Buffer)
	k.TemplateService.RenderManifest(vm, domainXML, stdinBuffer)
	cmd := exec.Command("kubectl", "-s", k.ApiServer, "create", "-f", "-")
	cmd.Stdin = stdinBuffer
	cmd.Stderr = stderrBuffer
	if err := cmd.Run(); err != nil {
		stderr := stderrBuffer.String()
		if stderr != "" {
			return errors.New(stderr)
		} else {
			return err
		}
	}
	k.logger.Info().Log("action", "StartPod", "object", "VM", "UUID", vm.UUID, "name", vm.Name)
	return nil
}

func NewKubeService(logger log.Logger, apiServer string) KubeService {
	svc := kubeService{
		logger:    levels.New(logger).With("component", "KubeService"),
		ApiServer: apiServer,
	}
	return &svc
}
