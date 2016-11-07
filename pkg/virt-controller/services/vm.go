package services

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	"github.com/satori/go.uuid"
	"k8s.io/client-go/1.5/kubernetes"
	kubeapi "k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/labels"
	"k8s.io/client-go/1.5/pkg/selection"
	"k8s.io/client-go/1.5/pkg/types"
	"k8s.io/client-go/1.5/pkg/util/sets"
	"k8s.io/client-go/1.5/pkg/util/yaml"
	"kubevirt/core/pkg/api"
	"kubevirt/core/pkg/libvirt"
	"kubevirt/core/pkg/middleware"
	"kubevirt/core/pkg/precond"
	"regexp"
)

type VMService interface {
	StartVMRaw(*api.VM, []byte) error
	StartVM(*api.VM) error
	DeleteVM(*api.VM) error
	PrepareMigration(*api.VM) error
}

type vmService struct {
	logger          levels.Levels
	KubeCli         *kubernetes.Clientset `inject:""`
	TemplateService TemplateService       `inject:""`
}

func (v *vmService) StartVM(vm *api.VM) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	rawXML, err := xml.Marshal(libvirt.NewMinimalVM(vm.GetObjectMeta().GetName()))
	if err != nil {
		return err
	}
	return v.StartVMRaw(vm, rawXML)
}

func (v *vmService) StartVMRaw(vm *api.VM, rawXML []byte) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeNil(rawXML)
	precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

	// Broken racy approach to at least not start multiple VMs during demos
	listOptions, err := vmListOptions(vm)
	if err != nil {
		return err
	}
	podList, err := v.KubeCli.Core().Pods("default").List(*listOptions)
	if err != nil {
		return err
	}
	pods := podList.Items

	// Search for accepted and not finished pods
	c := getUnfinishedPods(pods)

	// Pod for VM already exists
	if c > 0 {
		return middleware.NewResourceExistsError("VM", vm.GetObjectMeta().GetName())
	}

	if vm.GetObjectMeta().GetUID() == "" {
		vm.GetObjectMeta().SetUID(types.UID(uuid.NewV4().String()))
		//TODO when we can serialize VMs to XML, we can get rid of this
		r := regexp.MustCompile("</domain[\\s]*>")
		rawXML = r.ReplaceAll(rawXML, []byte(fmt.Sprintf("<uuid>%s</uuid></domain>", vm.GetObjectMeta().GetUID())))
	}

	templateBuffer := new(bytes.Buffer)
	if err := v.TemplateService.RenderLaunchManifest(vm, rawXML, templateBuffer); err != nil {
		return err
	}

	pod := v1.Pod{}
	err = yaml.NewYAMLToJSONDecoder(templateBuffer).Decode(&pod)
	if err != nil {
		return err
	}
	if _, err := v.KubeCli.Core().Pods("default").Create(&pod); err != nil {
		return err
	}
	v.logger.Info().Log("action", "StartVMRaw", "object", "VM", "UUID", vm.GetObjectMeta().GetUID(), "name", vm.GetObjectMeta().GetName())
	return nil
}

func (v *vmService) DeleteVM(vm *api.VM) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	listOptions, err := vmListOptions(vm)
	if err != nil {
		return err
	}
	if err := v.KubeCli.Core().Pods("default").DeleteCollection(nil, *listOptions); err != nil {
		return err
	}
	v.logger.Info().Log("action", "DeleteVM", "object", "VM", "UUID", vm.GetObjectMeta().GetUID(), "name", vm.GetObjectMeta().GetUID())
	return nil
}

func NewVMService(logger log.Logger) VMService {
	precond.MustNotBeNil(logger)

	svc := vmService{logger: levels.New(logger).With("component", "VMService")}
	return &svc
}

func (v *vmService) PrepareMigration(vm *api.VM) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	precond.MustBeTrue(len(vm.Spec.NodeSelector) > 0)
	listOptions, err := vmListOptions(vm)
	if err != nil {
		return err
	}
	// Broken racy approach to at least not start multiple VMs during demos
	podList, err := v.KubeCli.Core().Pods("default").List(*listOptions)
	if err != nil {
		return err
	}
	pods := podList.Items

	// Pod for VM does not exist
	if len(pods) == 0 {
		return middleware.NewResourceNotFoundError("VM", vm.GetObjectMeta().GetName())
	}

	// Search for accepted and not finished pods
	c := getUnfinishedPods(pods)

	// If there are more than one pod in other states than Succeeded or Failed we can't go on
	if c > 1 {
		return middleware.NewResourceConflictError(fmt.Sprintf("VM %s is already migrating", vm.GetObjectMeta().GetName()))
	}

	templateBuffer := new(bytes.Buffer)
	if err := v.TemplateService.RenderMigrationManifest(vm, templateBuffer); err != nil {
		return err
	}
	pod := v1.Pod{}
	err = yaml.NewYAMLToJSONDecoder(templateBuffer).Decode(&pod)
	if err != nil {
		return err
	}
	if _, err := v.KubeCli.Core().Pods("default").Create(&pod); err != nil {
		return err
	}
	v.logger.Info().Log("action", "PrepareMigration", "object", "VM", "UUID", vm.GetObjectMeta().GetUID(), "name", vm.GetObjectMeta().GetName())
	return nil
}

func getUnfinishedPods(pods []v1.Pod) int {
	c := 0
	states := []v1.PodPhase{v1.PodRunning, v1.PodPending, v1.PodUnknown}
	for _, pod := range pods {
		for _, state := range states {
			if pod.Status.Phase == state {
				c++
			}
		}
	}
	return c
}

func vmSelector(vm *api.VM) (labels.Selector, error) {
	requirement, err := labels.NewRequirement("domain", selection.Equals, sets.NewString(vm.GetObjectMeta().GetName()))
	if err != nil {
		return nil, err
	}
	return labels.NewSelector().Add(*requirement), nil
}

func vmListOptions(vm *api.VM) (*kubeapi.ListOptions, error) {
	labelSelector, err := vmSelector(vm)
	if err != nil {
		return nil, err
	}
	return &kubeapi.ListOptions{LabelSelector: labelSelector}, nil
}
