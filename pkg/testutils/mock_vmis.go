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

package testutils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
)

// NewVMIReferenceWithNodeName returns a new VMI with object meta labels
func NewVMIReferenceWithLabels(name string) *v1.VirtualMachineInstance {
	labels := make(map[string]string)
	vmi := &v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
	vmi.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   v1.GroupVersion.Group,
		Kind:    "VirtualMachineInstance",
		Version: v1.GroupVersion.Version})
	return vmi
}
