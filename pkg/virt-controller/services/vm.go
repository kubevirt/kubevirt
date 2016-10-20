package services

import (
	"bytes"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	"github.com/satori/go.uuid"
	"kubevirt/core/pkg/api"
	"kubevirt/core/pkg/kubecli"
	"kubevirt/core/pkg/precond"
	"regexp"
)

type VMService interface {
	StartVMRaw(*api.VM, []byte) error
	DeleteVM(*api.VM) error
}

type vmService struct {
	logger          levels.Levels
	KubeCli         kubecli.KubeCli `inject:""`
	TemplateService TemplateService `inject:""`
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

	templateBuffer := new(bytes.Buffer)
	if err := v.TemplateService.RenderLaunchManifest(vm, rawXML, templateBuffer); err != nil {
		return err
	}

	if err := v.KubeCli.CreatePod(templateBuffer); err != nil {
		return err
	}
	v.logger.Info().Log("action", "StartVMRaw", "object", "VM", "UUID", vm.UUID, "name", vm.Name)
	return nil
}

func (v *vmService) DeleteVM(vm *api.VM) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeEmpty(vm.Name)

	if err := v.KubeCli.DeletePodsByLabel("domain", vm.Name); err != nil {
		return err
	}
	v.logger.Info().Log("action", "DeleteVM", "object", "VM", "UUID", vm.UUID, "name", vm.Name)
	return nil
}

func NewVMService(logger log.Logger) VMService {
	precond.MustNotBeNil(logger)

	svc := vmService{logger: levels.New(logger).With("component", "VMService")}
	return &svc
}
