package services

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	"kubevirt/core/pkg/virt-controller/entities"
	"kubevirt/core/pkg/virt-controller/precond"
)

type VMService interface {
	StartVMRaw(*entities.VM, []byte) error
}

type vmService struct {
	logger      levels.Levels
	KubeService KubeService `inject:""`
}

func (v *vmService) StartVMRaw(vm *entities.VM, rawXML []byte) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeNil(rawXML)
	if err := v.KubeService.StartPod(vm); err != nil {
		return err
	}
	v.logger.Info().Log("action", "StartVMRaw", "object", "VM", "UUID", vm.UUID, "name", vm.Name)
	return nil
}

func NewVMService(logger log.Logger) VMService {
	precond.MustNotBeNil(logger)
	svc := vmService{logger: levels.New(logger).With("component", "VMService")}
	return &svc
}
