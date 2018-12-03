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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package util

import (
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

const (
	KubeVirtFinalizer string = "foregroundDeleteKubeVirt"
)

func UpdatePhase(kv *v1.KubeVirt, phase v1.KubeVirtPhase, clientset kubecli.KubevirtClient) error {
	patchStr := fmt.Sprintf("{\"status\":{\"phase\":\"%s\"}}", phase)
	_, err := clientset.KubeVirt(kv.Namespace).Patch(kv.Name, types.MergePatchType, []byte(patchStr))
	return err
}

func AddFinalizer(kv *v1.KubeVirt, clientset kubecli.KubevirtClient) error {
	if !HasFinalizer(kv) {
		kv.Finalizers = append(kv.Finalizers, KubeVirtFinalizer)
		return patchFinalizer(kv, clientset)
	}
	return nil
}

func RemoveFinalizer(kv *v1.KubeVirt, clientset kubecli.KubevirtClient) error {
	kv.SetFinalizers([]string{})
	return patchFinalizer(kv, clientset)
}

func HasFinalizer(kv *v1.KubeVirt) bool {
	for _, f := range kv.GetFinalizers() {
		if f == KubeVirtFinalizer {
			return true
		}
	}
	return false
}

func patchFinalizer(kv *v1.KubeVirt, clientset kubecli.KubevirtClient) error {
	var finalizers string
	//if len(kv.Finalizers) > 0 {
	bytes, err := json.Marshal(kv.Finalizers)
	if err != nil {
		return err
	}
	finalizers = string(bytes)
	//} else {
	//	finalizers = "\"[]\""
	//}
	patchStr := fmt.Sprintf(`{"metadata":{"finalizers":%s}}`, finalizers)
	kv, err = clientset.KubeVirt(kv.Namespace).Patch(kv.Name, types.MergePatchType, []byte(patchStr))
	return err
}
