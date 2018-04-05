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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"

	corev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/precond"
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

type VMService interface {
	StartVMPod(*corev1.VirtualMachine) (*v1.Pod, error)
	DeleteVMPod(*corev1.VirtualMachine) error
	GetRunningVMPods(*corev1.VirtualMachine) (*v1.PodList, error)
}

type vmService struct {
	KubeCli         kubecli.KubevirtClient
	TemplateService TemplateService
}

func (v *vmService) StartVMPod(vm *corev1.VirtualMachine) (*v1.Pod, error) {
	precond.MustNotBeNil(vm)
	precond.MustNotBeEmpty(vm.GetName())
	precond.MustNotBeEmpty(vm.GetNamespace())
	precond.MustNotBeEmpty(string(vm.GetUID()))

	pod, err := v.TemplateService.RenderLaunchManifest(vm)
	if err != nil {
		return nil, err
	}

	if pod, err = v.KubeCli.CoreV1().Pods(vm.GetNamespace()).Create(pod); err != nil {
		return nil, err
	}
	return pod, nil
}

func (v *vmService) DeleteVMPod(vm *corev1.VirtualMachine) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeEmpty(vm.GetName())
	precond.MustNotBeEmpty(vm.GetNamespace())

	if err := v.KubeCli.CoreV1().Pods(vm.Namespace).DeleteCollection(nil, UnfinishedVMPodSelector(vm)); err != nil {
		return err
	}

	return nil
}

func (v *vmService) GetRunningVMPods(vm *corev1.VirtualMachine) (*v1.PodList, error) {
	podList, err := v.KubeCli.CoreV1().Pods(vm.Namespace).List(UnfinishedVMPodSelector(vm))

	if err != nil {
		return nil, err
	}
	return podList, nil
}

func NewVMService(KubeCli kubecli.KubevirtClient,
	TemplateService TemplateService) VMService {
	svc := vmService{
		KubeCli:         KubeCli,
		TemplateService: TemplateService,
	}
	return &svc
}

func UnfinishedVMPodSelector(vm *corev1.VirtualMachine) metav1.ListOptions {
	fieldSelector := fields.ParseSelectorOrDie(
		"status.phase!=" + string(v1.PodFailed) +
			",status.phase!=" + string(v1.PodSucceeded))
	labelSelector, err := labels.Parse(fmt.Sprintf(corev1.AppLabel+"=virt-launcher,"+corev1.DomainLabel+" in (%s)", vm.GetName()))
	if err != nil {
		panic(err)
	}
	return metav1.ListOptions{FieldSelector: fieldSelector.String(), LabelSelector: labelSelector.String()}
}
