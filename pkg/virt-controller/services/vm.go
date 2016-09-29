package services

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	"kubevirt/core/pkg/virt-controller/entities"
)

type VMService interface {
	StartVMRaw(*entities.VM, []byte) error
}

type vmService struct {
	logger levels.Levels
}

func (v *vmService) StartVMRaw(vm *entities.VM, rawXML []byte) error {
	v.logger.Info().Log("action", "StartVMRaw", "object", "VM", "UUID", vm.UUID, "name", vm.Name)
	return nil
}

func MakeVMService(logger log.Logger) VMService {
	svc := vmService{logger: levels.New(logger).With("component", "VMService")}
	return &svc
}
