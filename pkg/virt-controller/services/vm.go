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
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/precond"
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

type VMService interface {
	StartVMPod(*corev1.VirtualMachine) error
	DeleteVMPod(*corev1.VirtualMachine) error
	GetRunningVMPods(*corev1.VirtualMachine) (*v1.PodList, error)
	FetchVM(namespace string, vmName string) (*corev1.VirtualMachine, bool, error)
	PutVm(vm *corev1.VirtualMachine) (*corev1.VirtualMachine, error)
}

type vmService struct {
	KubeCli         kubecli.KubevirtClient
	RestClient      *rest.RESTClient
	TemplateService TemplateService
}

func (v *vmService) StartVMPod(vm *corev1.VirtualMachine) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	precond.MustNotBeEmpty(string(vm.GetObjectMeta().GetUID()))

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
func (v *vmService) PutVm(vm *corev1.VirtualMachine) (*corev1.VirtualMachine, error) {
	logger := log.Log.Object(vm)
	obj, err := v.RestClient.Put().Resource("virtualmachines").Body(vm).Name(vm.ObjectMeta.Name).Namespace(vm.ObjectMeta.Namespace).Do().Get()
	if err != nil {
		logger.Reason(err).Error("Setting the VM state failed.")
		return nil, err
	}
	return obj.(*corev1.VirtualMachine), nil
}

func (v *vmService) DeleteVMPod(vm *corev1.VirtualMachine) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())

	if err := v.KubeCli.CoreV1().Pods(vm.ObjectMeta.Namespace).DeleteCollection(nil, UnfinishedVMPodSelector(vm)); err != nil {
		return err
	}

	return nil
}

func (v *vmService) GetRunningVMPods(vm *corev1.VirtualMachine) (*v1.PodList, error) {
	podList, err := v.KubeCli.CoreV1().Pods(vm.ObjectMeta.Namespace).List(UnfinishedVMPodSelector(vm))

	if err != nil {
		return nil, err
	}
	return podList, nil
}

func (v *vmService) FetchVM(namespace string, vmName string) (*corev1.VirtualMachine, bool, error) {
	resp, err := v.RestClient.Get().Namespace(namespace).Resource("virtualmachines").Name(vmName).Do().Get()
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	vm := resp.(*corev1.VirtualMachine)
	return vm, true, nil
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

func UnfinishedVMPodSelector(vm *corev1.VirtualMachine) metav1.ListOptions {
	fieldSelector := fields.ParseSelectorOrDie(
		"status.phase!=" + string(v1.PodFailed) +
			",status.phase!=" + string(v1.PodSucceeded))
	labelSelector, err := labels.Parse(fmt.Sprintf(corev1.AppLabel+"=virt-launcher,"+corev1.DomainLabel+" in (%s)", vm.GetObjectMeta().GetName()))
	if err != nil {
		panic(err)
	}
	return metav1.ListOptions{FieldSelector: fieldSelector.String(), LabelSelector: labelSelector.String()}
}
