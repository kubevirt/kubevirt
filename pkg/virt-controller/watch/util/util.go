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
 * Copyright The KubeVirt Authors.
 *
 */

package util

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	typesutil "kubevirt.io/kubevirt/pkg/storage/types"
)

func ProcessWorkItem(queue workqueue.TypedRateLimitingInterface[string], handler func(string) (time.Duration, error)) bool {
	obj, shutdown := queue.Get()
	if shutdown {
		return false
	}

	err := func(key string) error {
		defer queue.Done(obj)

		if requeueAfter, err := handler(key); requeueAfter > 0 || err != nil {
			if requeueAfter > 0 {
				queue.AddAfter(key, requeueAfter)
			} else {
				queue.AddRateLimited(key)
			}

			return err
		}

		queue.Forget(obj)

		return nil

	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func PodsUsingPVCs(podInformer cache.SharedIndexInformer, namespace string, pvcNames sets.String) ([]corev1.Pod, error) {
	var pods []corev1.Pod

	if pvcNames.Len() < 1 {
		return pods, nil
	}

	objs, err := podInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}

	for _, obj := range objs {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return nil, fmt.Errorf("expected Pod, got %T", obj)
		}

		if pod.Status.Phase == corev1.PodSucceeded {
			continue
		}

		for _, volume := range pod.Spec.Volumes {
			if volume.VolumeSource.PersistentVolumeClaim != nil &&
				pvcNames.Has(volume.PersistentVolumeClaim.ClaimName) {
				pods = append(pods, *pod)
			}
		}
	}

	return pods, nil
}

func CreateDataVolumeManifest(clientset kubecli.KubevirtClient, dataVolumeTemplate virtv1.DataVolumeTemplateSpec, vm *virtv1.VirtualMachine) (*cdiv1.DataVolume, error) {
	newDataVolume, err := typesutil.GenerateDataVolumeFromTemplate(clientset, dataVolumeTemplate, vm.Namespace, vm.Spec.Template.Spec.PriorityClassName)
	if err != nil {
		return nil, err
	}

	newDataVolume.ObjectMeta.Labels[virtv1.CreatedByLabel] = string(vm.UID)
	newDataVolume.ObjectMeta.OwnerReferences = []v1.OwnerReference{
		*v1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind),
	}

	return newDataVolume, nil
}
