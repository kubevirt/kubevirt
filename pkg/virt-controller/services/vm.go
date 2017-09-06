/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package services

import (
	"fmt"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"

	corev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/precond"
	registrydisk "kubevirt.io/kubevirt/pkg/registry-disk"
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

type VMService interface {
	StartVMPod(*corev1.VM) error
	DeleteVMPod(*corev1.VM) error
	GetRunningVMPods(*corev1.VM) (*v1.PodList, error)
	DeleteMigrationTargetPods(*corev1.Migration) error
	GetRunningMigrationPods(*corev1.Migration) (*v1.PodList, error)
	CreateMigrationTargetPod(migration *corev1.Migration, vm *corev1.VM) error
	UpdateMigration(migration *corev1.Migration) error
	FetchVM(namespace string, vmName string) (*corev1.VM, bool, error)
	FetchMigration(namespace string, migrationName string) (*corev1.Migration, bool, error)
	StartMigration(migration *corev1.Migration, vm *corev1.VM, sourceNode *v1.Node, targetNode *v1.Node, targetPod *v1.Pod) error
	GetMigrationJob(migration *corev1.Migration) (*v1.Pod, bool, error)
	PutVm(vm *corev1.VM) (*corev1.VM, error)
}

type vmService struct {
	KubeCli         kubecli.KubevirtClient
	RestClient      *rest.RESTClient
	TemplateService TemplateService
}

func (v *vmService) StartVMPod(vm *corev1.VM) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	precond.MustNotBeEmpty(string(vm.GetObjectMeta().GetUID()))

	err := registrydisk.Initialize(vm, v.KubeCli)
	if err != nil {
		logger := logging.DefaultLogger().Object(vm)
		logger.Error().Reason(err).Msg("Initializing container registry disks failed.")
		return err
	}

	pod, err := v.TemplateService.RenderLaunchManifest(vm)
	if err != nil {
		return err
	}

	if _, err := v.KubeCli.CoreV1().Pods(vm.GetObjectMeta().GetNamespace()).Create(pod); err != nil {
		return err
	}
	return nil
}

// synchronously put updated VM object to API server.
func (v *vmService) PutVm(vm *corev1.VM) (*corev1.VM, error) {
	logger := logging.DefaultLogger().Object(vm)
	obj, err := v.RestClient.Put().Resource("vms").Body(vm).Name(vm.ObjectMeta.Name).Namespace(vm.ObjectMeta.Namespace).Do().Get()
	if err != nil {
		logger.Error().Reason(err).Msg("Setting the VM state failed.")
		return nil, err
	}
	return obj.(*corev1.VM), nil
}

func (v *vmService) DeleteVMPod(vm *corev1.VM) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())

	if err := v.KubeCli.CoreV1().Pods(vm.ObjectMeta.Namespace).DeleteCollection(nil, UnfinishedVMPodSelector(vm)); err != nil {
		return err
	}

	registrydisk.CleanUp(vm, v.KubeCli)

	return nil
}

func (v *vmService) GetRunningVMPods(vm *corev1.VM) (*v1.PodList, error) {
	podList, err := v.KubeCli.CoreV1().Pods(vm.ObjectMeta.Namespace).List(UnfinishedVMPodSelector(vm))

	if err != nil {
		return nil, err
	}
	return podList, nil
}

func (v *vmService) UpdateMigration(migration *corev1.Migration) error {
	migrationName := migration.ObjectMeta.Name
	_, err := v.RestClient.Put().Namespace(migration.ObjectMeta.Namespace).Resource("migrations").Body(migration).Name(migrationName).Do().Get()
	return err
}

func (v *vmService) FetchVM(namespace string, vmName string) (*corev1.VM, bool, error) {
	resp, err := v.RestClient.Get().Namespace(namespace).Resource("vms").Name(vmName).Do().Get()
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	vm := resp.(*corev1.VM)
	return vm, true, nil
}

func (v *vmService) FetchMigration(namespace string, migrationName string) (*corev1.Migration, bool, error) {
	resp, err := v.RestClient.Get().Namespace(namespace).Resource("migrations").Name(migrationName).Do().Get()
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	migration := resp.(*corev1.Migration)
	return migration, true, nil
}

func NewVMService(KubeCli kubecli.KubevirtClient,
	RestClient *rest.RESTClient,
	TemplateService TemplateService) VMService {
	svc := vmService{
		KubeCli:         KubeCli,
		RestClient:      RestClient,
		TemplateService: TemplateService,
	}
	return &svc
}

func UnfinishedVMPodSelector(vm *corev1.VM) metav1.ListOptions {
	fieldSelector := fields.ParseSelectorOrDie(
		"status.phase!=" + string(v1.PodFailed) +
			",status.phase!=" + string(v1.PodSucceeded))
	labelSelector, err := labels.Parse(fmt.Sprintf(corev1.AppLabel+"=virt-launcher,"+corev1.DomainLabel+" in (%s)", vm.GetObjectMeta().GetName()))
	if err != nil {
		panic(err)
	}
	return metav1.ListOptions{FieldSelector: fieldSelector.String(), LabelSelector: labelSelector.String()}
}

func (v *vmService) CreateMigrationTargetPod(migration *corev1.Migration, vm *corev1.VM) error {
	pod, err := v.TemplateService.RenderLaunchManifest(vm)
	migrationLabel := migration.GetObjectMeta().GetName()

	pod.ObjectMeta.Labels[corev1.MigrationLabel] = migrationLabel
	pod.ObjectMeta.Labels[corev1.MigrationUIDLabel] = string(migration.GetObjectMeta().GetUID())
	pod.Spec.Affinity = corev1.AntiAffinityFromVMNode(vm)
	if err == nil {
		_, err = v.KubeCli.CoreV1().Pods(migration.GetObjectMeta().GetNamespace()).Create(pod)
	}
	return err
}

func (v *vmService) DeleteMigrationTargetPods(migration *corev1.Migration) error {
	precond.MustNotBeNil(migration)
	precond.MustNotBeEmpty(migration.GetObjectMeta().GetName())

	if err := v.KubeCli.CoreV1().Pods(migration.GetObjectMeta().GetNamespace()).DeleteCollection(nil, unfinishedMigrationTargetPodSelector(migration)); err != nil {
		return err
	}
	return nil
}

func (v *vmService) GetRunningMigrationPods(migration *corev1.Migration) (*v1.PodList, error) {
	podList, err := v.KubeCli.CoreV1().Pods(migration.GetObjectMeta().GetNamespace()).List(unfinishedMigrationTargetPodSelector(migration))
	if err != nil {
		return nil, err
	}
	return podList, nil
}

func (v *vmService) StartMigration(migration *corev1.Migration, vm *corev1.VM, sourceNode *v1.Node, targetNode *v1.Node, targetPod *v1.Pod) error {

	// Look up node migration details
	nodeDetails, err := kubecli.NewVirtHandlerClient(v.KubeCli).ForNode(targetNode.Name).NodeMigrationDetails(vm)
	if err != nil {
		return err
	}

	job, err := v.TemplateService.RenderMigrationJob(vm, sourceNode, targetNode, targetPod, nodeDetails)
	job.ObjectMeta.Labels[corev1.MigrationLabel] = migration.GetObjectMeta().GetName()
	job.ObjectMeta.Labels[corev1.MigrationUIDLabel] = string(migration.GetObjectMeta().GetUID())
	if err != nil {
		return err
	}
	_, err = v.KubeCli.CoreV1().Pods(migration.GetObjectMeta().GetNamespace()).Create(job)
	return err
}

func (v *vmService) GetMigrationJob(migration *corev1.Migration) (*v1.Pod, bool, error) {
	selector := migrationJobSelector(migration)
	podList, err := v.KubeCli.CoreV1().Pods(migration.ObjectMeta.Namespace).List(selector)
	if err != nil {
		return nil, false, err
	}
	if len(podList.Items) == 0 {
		return nil, false, nil
	}

	return &podList.Items[0], true, nil
}

func unfinishedMigrationTargetPodSelector(migration *corev1.Migration) metav1.ListOptions {
	fieldSelector := fields.ParseSelectorOrDie(
		"status.phase!=" + string(v1.PodFailed) +
			",status.phase!=" + string(v1.PodSucceeded))
	labelSelector, err := labels.Parse(
		fmt.Sprintf(corev1.AppLabel+"=virt-launcher,"+corev1.DomainLabel+","+corev1.MigrationUIDLabel+" in (%s)", migration.GetObjectMeta().GetUID()))
	if err != nil {
		panic(err)
	}
	return metav1.ListOptions{FieldSelector: fieldSelector.String(), LabelSelector: labelSelector.String()}
}

func migrationJobSelector(migration *corev1.Migration) metav1.ListOptions {
	labelSelector, err := labels.Parse(corev1.DomainLabel + "," + corev1.AppLabel + "=migration" +
		"," + corev1.MigrationUIDLabel + "=" + string(migration.GetObjectMeta().GetUID()),
	)
	if err != nil {
		panic(err)
	}
	return metav1.ListOptions{LabelSelector: labelSelector.String()}
}
