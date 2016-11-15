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
	"k8s.io/client-go/1.5/pkg/fields"
	"k8s.io/client-go/1.5/pkg/labels"
	"k8s.io/client-go/1.5/pkg/types"
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
	GetRunningPods(*api.VM) (*v1.PodList, error)
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
	precond.MustNotBeEmpty(string(vm.GetObjectMeta().GetUID()))

	podList, err := v.GetRunningPods(vm)
	if err != nil {
		return err
	}

	// Pod for VM already exists
	if len(podList.Items) > 0 {
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
	if _, err := v.KubeCli.Core().Pods(kubeapi.NamespaceDefault).Create(&pod); err != nil {
		return err
	}
	v.logger.Info().Log("action", "StartVMRaw", "object", "VM", "UUID", vm.GetObjectMeta().GetUID(), "name", vm.GetObjectMeta().GetName())
	return nil
}

func (v *vmService) DeleteVM(vm *api.VM) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

	if err := v.KubeCli.Core().Pods(kubeapi.NamespaceDefault).DeleteCollection(nil, unfinishedVMPodSelector(vm)); err != nil {
		return err
	}
	v.logger.Info().Log("action", "DeleteVM", "object", "VM", "UUID", vm.GetObjectMeta().GetUID(), "name", vm.GetObjectMeta().GetUID())
	return nil
}

func (v *vmService) GetRunningPods(vm *api.VM) (*v1.PodList, error) {
	podList, err := v.KubeCli.Core().Pods(kubeapi.NamespaceDefault).List(unfinishedVMPodSelector(vm))
	if err != nil {
		return nil, err
	}
	return podList, nil
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
	podList, err := v.GetRunningPods(vm)
	if err != nil {
		return err
	}

	// If there are more than one pod in other states than Succeeded or Failed we can't go on
	if len(podList.Items) > 1 {
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
	if _, err := v.KubeCli.Core().Pods(kubeapi.NamespaceDefault).Create(&pod); err != nil {
		return err
	}
	v.logger.Info().Log("action", "PrepareMigration", "object", "VM", "UUID", vm.GetObjectMeta().GetUID(), "name", vm.GetObjectMeta().GetName())
	return nil
}

func unfinishedVMPodSelector(vm *api.VM) kubeapi.ListOptions {
	fieldSelector := fields.ParseSelectorOrDie(
		"status.phase!=" + string(kubeapi.PodFailed) +
			",status.phase!=" + string(kubeapi.PodSucceeded))
	labelSelector, err := labels.Parse(fmt.Sprintf("domain in (%s)", vm.GetObjectMeta().GetName()))
	if err != nil {
		panic(err)
	}
	return kubeapi.ListOptions{FieldSelector: fieldSelector, LabelSelector: labelSelector}
}
