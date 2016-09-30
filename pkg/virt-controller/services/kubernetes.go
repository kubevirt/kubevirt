package services

import (
	"bytes"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	"kubevirt/core/pkg/virt-controller/entities"
	"os/exec"
)

type KubeService interface {
	StartPod(vm *entities.VM) error
}

type kubeService struct {
	logger          levels.Levels
	TemplateService TemplateService `inject:""`
	ApiServer       string
}

func (k *kubeService) StartPod(vm *entities.VM) error {
	buffer := new(bytes.Buffer)
	k.TemplateService.RenderManifest(vm, buffer)
	cmd := exec.Command("kubectl", "create", "-f", "-", "-s", k.ApiServer)
	cmd.Stdin = buffer
	if err := cmd.Run(); err != nil {
		return err
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
