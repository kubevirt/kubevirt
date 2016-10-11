package services

import (
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	"github.com/satori/go.uuid"
	"kubevirt/core/pkg/api"
	"kubevirt/core/pkg/precond"
	"regexp"
)

type VMService interface {
	StartVMRaw(*api.VM, []byte) error
}

type vmService struct {
	logger      levels.Levels
	KubeService KubeService `inject:""`
}

func (v *vmService) StartVMRaw(vm *api.VM, rawXML []byte) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeNil(rawXML)
	precond.MustNotBeEmpty(vm.Name)

	if vm.UUID == uuid.Nil {
		vm.UUID = uuid.NewV4()
		//TODO when we can serialize VMs to XML, we can get rid of this
		r := regexp.MustCompile("</domain[\\s]*>")
		rawXML = r.ReplaceAll(rawXML, []byte(fmt.Sprintf("<uuid>%s</uuid></domain>", vm.UUID.String())))
	}
	if err := v.KubeService.StartPod(vm, rawXML); err != nil {
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
