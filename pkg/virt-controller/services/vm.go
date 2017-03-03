package services

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
	corev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/middleware"
	"kubevirt.io/kubevirt/pkg/precond"
)

type VMService interface {
	StartVM(*corev1.VM) error
	DeleteVM(*corev1.VM) error
	PrepareMigration(*corev1.VM) error
	GetRunningPods(*corev1.VM) (*v1.PodList, error)
}

type vmService struct {
	KubeCli         *kubernetes.Clientset `inject:""`
	TemplateService TemplateService       `inject:""`
}

func (v *vmService) StartVM(vm *corev1.VM) error {
	precond.MustNotBeNil(vm)
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

	pod, err := v.TemplateService.RenderLaunchManifest(vm)
	if err != nil {
		return err
	}

	if _, err := v.KubeCli.Core().Pods(v1.NamespaceDefault).Create(pod); err != nil {
		return err
	}
	return nil
}

func (v *vmService) DeleteVM(vm *corev1.VM) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

	if err := v.KubeCli.Core().Pods(v1.NamespaceDefault).DeleteCollection(nil, UnfinishedVMPodSelector(vm)); err != nil {
		return err
	}
	return nil
}

func (v *vmService) GetRunningPods(vm *corev1.VM) (*v1.PodList, error) {
	podList, err := v.KubeCli.Core().Pods(v1.NamespaceDefault).List(UnfinishedVMPodSelector(vm))
	if err != nil {
		return nil, err
	}
	return podList, nil
}

func NewVMService() VMService {
	svc := vmService{}
	return &svc
}

func (v *vmService) PrepareMigration(vm *corev1.VM) error {
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

	pod, err := v.TemplateService.RenderLaunchManifest(vm)
	if err != nil {
		return err
	}
	if _, err := v.KubeCli.Core().Pods(v1.NamespaceDefault).Create(pod); err != nil {
		return err
	}
	return nil
}

// Visible for tests
func UnfinishedVMPodSelector(vm *corev1.VM) v1.ListOptions {
	fieldSelector := fields.ParseSelectorOrDie(
		"status.phase!=" + string(v1.PodFailed) +
			",status.phase!=" + string(v1.PodSucceeded))
	labelSelector, err := labels.Parse(fmt.Sprintf(corev1.DomainLabel+" in (%s)", vm.GetObjectMeta().GetName()))
	if err != nil {
		panic(err)
	}
	return v1.ListOptions{FieldSelector: fieldSelector.String(), LabelSelector: labelSelector.String()}
}
