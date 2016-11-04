package services

import (
	"bytes"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	"github.com/satori/go.uuid"
	"k8s.io/client-go/1.5/pkg/types"
	"kubevirt/core/pkg/api"
	"kubevirt/core/pkg/kubecli"
	"kubevirt/core/pkg/kubecli/v1.2"
	"kubevirt/core/pkg/middleware"
	"kubevirt/core/pkg/precond"
	"regexp"
)

type VMService interface {
	StartVMRaw(*api.VM, []byte) error
	DeleteVM(*api.VM) error
	PrepareMigration(*api.VM) error
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

	// Broken racy approach to at least not start multiple VMs during demos
	pods, err := v.KubeCli.GetPodsByLabel("domain", vm.Name)
	if err != nil {
		return err
	}

	// Search for accepted and not finished pods
	c := getUnfinishedPods(pods)

	// Pod for VM already exists
	if c > 0 {
		return middleware.NewResourceExistsError("VM", vm.Name)
	}

	if vm.UID == "" {
		vm.UID = types.UID(uuid.NewV4().String())
		//TODO when we can serialize VMs to XML, we can get rid of this
		r := regexp.MustCompile("</domain[\\s]*>")
		rawXML = r.ReplaceAll(rawXML, []byte(fmt.Sprintf("<uuid>%s</uuid></domain>", vm.UID)))
	}

	templateBuffer := new(bytes.Buffer)
	if err := v.TemplateService.RenderLaunchManifest(vm, rawXML, templateBuffer); err != nil {
		return err
	}

	if err := v.KubeCli.CreatePod(templateBuffer); err != nil {
		return err
	}
	v.logger.Info().Log("action", "StartVMRaw", "object", "VM", "UUID", vm.UID, "name", vm.Name)
	return nil
}

func (v *vmService) DeleteVM(vm *api.VM) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeEmpty(vm.Name)

	if err := v.KubeCli.DeletePodsByLabel("domain", vm.Name); err != nil {
		return err
	}
	v.logger.Info().Log("action", "DeleteVM", "object", "VM", "UUID", vm.UID, "name", vm.Name)
	return nil
}

func NewVMService(logger log.Logger) VMService {
	precond.MustNotBeNil(logger)

	svc := vmService{logger: levels.New(logger).With("component", "VMService")}
	return &svc
}

func (v *vmService) PrepareMigration(vm *api.VM) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeEmpty(vm.Name)
	precond.MustBeTrue(len(vm.Spec.NodeSelector) > 0)

	// Broken racy approach to at least not start multiple VMs during demos
	pods, err := v.KubeCli.GetPodsByLabel("domain", vm.Name)
	if err != nil {
		return err
	}

	// Pod for VM does not exist
	if len(pods) == 0 {
		return middleware.NewResourceNotFoundError("VM", vm.Name)
	}

	// Search for accepted and not finished pods
	c := getUnfinishedPods(pods)

	// If there are more than one pod in other states than Succeeded or Failed we can't go on
	if c > 1 {
		return middleware.NewResourceConflictError(fmt.Sprintf("VM %s is already migrating", vm.Name))
	}

	templateBuffer := new(bytes.Buffer)
	if err := v.TemplateService.RenderMigrationManifest(vm, templateBuffer); err != nil {
		return err
	}
	if err := v.KubeCli.CreatePod(templateBuffer); err != nil {
		return err
	}
	v.logger.Info().Log("action", "PrepareMigration", "object", "VM", "UUID", vm.UID, "name", vm.Name)
	return nil
}

func getUnfinishedPods(pods []v1_2.Pod) int {
	c := 0
	states := []string{"Running", "Pending", "Unknown"}
	for _, pod := range pods {
		for _, state := range states {
			if pod.Status.Phase == state {
				c++
			}
		}
	}
	return c
}
