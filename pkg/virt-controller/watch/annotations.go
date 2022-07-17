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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2022 Red Hat, Inc.
 *
 */

package watch

import (
	"errors"
	"fmt"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	k8sv1 "k8s.io/api/core/v1"

	virtv1 "kubevirt.io/api/core/v1"
)

func copyMultusAnnotationsFromPodToVmi(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) error {
	annotationsKeyList := []string{networkv1.NetworkStatusAnnot}
	for _, annotationKey := range annotationsKeyList {
		err := copyPodAnnotationToVmi(vmi, pod, annotationKey)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *VMIController) multusAnnotationsCopiedToVmi(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) error {
	annotationsKeyList := []string{networkv1.NetworkStatusAnnot}

	executorWithTimeout := c.sriovVmiCopyAnnotExecPool.LoadOrStore(vmi.UID)
	return executorWithTimeout.Exec(func() error {
		for _, annotationKey := range annotationsKeyList {
			if !annotationCopiedToVmi(vmi, pod, annotationKey) {
				return fmt.Errorf("multus annotation %s in vmi %s does not exist/equal to same annotation in pod", annotationKey, vmi.Name)
			}
		}
		return nil
	})
}

func annotationCopiedToVmi(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod, annotationKey string) bool {
	var podAnnotation, vmiAnnotation string
	var ok bool
	if podAnnotation, ok = pod.Annotations[annotationKey]; !ok {
		return false
	}
	if vmiAnnotation, ok = vmi.Annotations[annotationKey]; !ok {
		return false
	}
	if podAnnotation != vmiAnnotation {
		return false
	}

	return true
}

func copyPodAnnotationToVmi(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod, annotationKey string) error {
	annotation, ok := pod.Annotations[annotationKey]
	if !ok {
		return errors.New(fmt.Sprintf("annotation not found on pod: %s", annotationKey))
	}

	vmi.Annotations[annotationKey] = annotation
	return nil
}
