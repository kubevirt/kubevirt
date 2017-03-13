package services

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	batchv1 "k8s.io/client-go/pkg/apis/batch/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/rest"
	corev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/middleware"
	"kubevirt.io/kubevirt/pkg/precond"
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

type VMService interface {
	StartVMPod(*corev1.VM) error
	DeleteVMPod(*corev1.VM) error
	GetRunningVMPods(*corev1.VM) (*v1.PodList, error)
	DeleteMigration(*corev1.Migration) error
	GetRunningMigrationPods(*corev1.Migration) (*v1.PodList, error)
	SetupMigration(migration *corev1.Migration, vm *corev1.VM) error
	UpdateMigration(migration *corev1.Migration) error
	FetchVM(vmName string) (*corev1.VM, error)
	StartMigration(migration *corev1.Migration, vm *corev1.VM, sourceNode *v1.Node, targetNode *v1.Node) error
	GetMigrationJob(vm *corev1.VM) (*batchv1.Job, bool, error)
}

type vmService struct {
	KubeCli         *kubernetes.Clientset `inject:""`
	RestClient      *rest.RESTClient      `inject:""`
	TemplateService TemplateService       `inject:""`
}

func (v *vmService) StartVMPod(vm *corev1.VM) error {

	precond.MustNotBeNil(vm)
	precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	precond.MustNotBeEmpty(string(vm.GetObjectMeta().GetUID()))

	podList, err := v.GetRunningVMPods(vm)
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

func (v *vmService) DeleteVMPod(vm *corev1.VM) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

	if err := v.KubeCli.CoreV1().Pods(v1.NamespaceDefault).DeleteCollection(nil, UnfinishedVMPodSelector(vm)); err != nil {
		return err
	}
	return nil
}

func (v *vmService) GetRunningVMPods(vm *corev1.VM) (*v1.PodList, error) {
	podList, err := v.KubeCli.CoreV1().Pods(v1.NamespaceDefault).List(UnfinishedVMPodSelector(vm))

	if err != nil {
		return nil, err
	}
	return podList, nil
}

func (v *vmService) UpdateMigration(migration *corev1.Migration) error {
	migrationName := migration.ObjectMeta.Name
	_, err := v.RestClient.Put().Namespace(v1.NamespaceDefault).Resource("migrations").Body(migration).Name(migrationName).Do().Get()
	return err
}

func (v *vmService) FetchVM(vmName string) (*corev1.VM, error) {
	resp, err := v.RestClient.Get().Namespace(v1.NamespaceDefault).Resource("vms").Name(vmName).Do().Get()
	if err != nil {
		return nil, err
	}
	vm := resp.(*corev1.VM)
	return vm, nil
}

func NewVMService() VMService {
	svc := vmService{}
	return &svc
}

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

func (v *vmService) SetupMigration(migration *corev1.Migration, vm *corev1.VM) error {
	pod, err := v.TemplateService.RenderLaunchManifest(vm)
	pod.ObjectMeta.Labels[corev1.MigrationLabel] = migration.GetObjectMeta().GetName()
	pod.ObjectMeta.Labels[corev1.MigrationUIDLabel] = string(migration.GetObjectMeta().GetUID())
	corev1.SetAntiAffinityToPod(pod, corev1.AntiAffinityFromVMNode(vm))
	if err == nil {
		_, err = v.KubeCli.CoreV1().Pods(v1.NamespaceDefault).Create(pod)
	}
	if err == nil {
		migration.Status.Phase = corev1.MigrationInProgress
	} else {
		migration.Status.Phase = corev1.MigrationFailed
	}

	err2 := v.UpdateMigration(migration)
	if err2 != nil {
		err = err2
	}
	return err
}

func (v *vmService) DeleteMigration(migration *corev1.Migration) error {
	precond.MustNotBeNil(migration)
	precond.MustNotBeEmpty(migration.GetObjectMeta().GetName())

	if err := v.KubeCli.CoreV1().Pods(v1.NamespaceDefault).DeleteCollection(nil, unfinishedMigrationPodSelector(migration)); err != nil {
		return err
	}
	return nil
}

func (v *vmService) GetRunningMigrationPods(migration *corev1.Migration) (*v1.PodList, error) {
	podList, err := v.KubeCli.CoreV1().Pods(v1.NamespaceDefault).List(unfinishedMigrationPodSelector(migration))
	if err != nil {
		return nil, err
	}
	return podList, nil
}

func (v *vmService) StartMigration(migration *corev1.Migration, vm *corev1.VM, sourceNode *v1.Node, targetNode *v1.Node) error {
	job, err := v.TemplateService.RenderMigrationJob(vm, sourceNode, targetNode)
	job.ObjectMeta.Labels[corev1.MigrationLabel] = migration.GetObjectMeta().GetName()
	job.ObjectMeta.Labels[corev1.MigrationUIDLabel] = string(migration.GetObjectMeta().GetUID())
	if err != nil {
		return err
	}
	return v.KubeCli.CoreV1().RESTClient().Post().AbsPath("/apis/batch/v1/namespaces/default/jobs").Body(job).Do().Error()
}

func (v *vmService) GetMigrationJob(vm *corev1.VM) (*batchv1.Job, bool, error) {
	selector, err := labels.Parse(corev1.DomainLabel)
	if err != nil {
		return nil, false, err
	}
	jobList, err := v.KubeCli.CoreV1().RESTClient().Get().AbsPath("/apis/batch/v1/namespaces/default/jobs").LabelsSelectorParam(selector).Do().Get()
	if err != nil {
		return nil, false, err
	}
	for _, job := range jobList.(*batchv1.JobList).Items {
		if job.Status.CompletionTime != nil {
			return &job, true, nil
		}
	}
	return nil, false, nil
}

func unfinishedMigrationPodSelector(migration *corev1.Migration) v1.ListOptions {
	fieldSelector := fields.ParseSelectorOrDie(
		"status.phase!=" + string(v1.PodFailed) +
			",status.phase!=" + string(v1.PodSucceeded))
	labelSelector, err := labels.Parse(fmt.Sprintf(corev1.DomainLabel+" in (%s)", migration.GetObjectMeta().GetName()))
	if err != nil {
		panic(err)
	}
	return v1.ListOptions{FieldSelector: fieldSelector.String(), LabelSelector: labelSelector.String()}
}
