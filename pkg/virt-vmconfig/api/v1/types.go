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

package v1

//go:generate swagger-doc

/*
 ATTENTION: Rerun code generators when comments on structs or fields are modified.
*/

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	kubevirtv1 "kubevirt.io/kubevirt/pkg/api/v1"
)

var VMConfigGroupVersionKind = schema.GroupVersionKind{Group: kubevirtv1.GroupName, Version: kubevirtv1.GroupVersion.Version, Kind: "VMConfig"}

// VMConfig is a persistent representation of a VMSpec and VMFeatures.
type VMConfig struct {
	metav1.TypeMeta `json:",inline"`
	ObjectMeta      metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec            VMConfigSpec      `json:"spec,omitempty" valid:"required"`
}

type VMConfigSpec struct {
	Template *VMConfigTemplate `json:"template,omitempty"`
}

type VMConfigTemplate struct {
	Spec     *kubevirtv1.DomainSpec `json:"spec,omitempty"`
	Features *VMConfigFeatures      `json:"features,omitempty"`
}

type VMConfigFeatures struct {
	OS string `json:"os,omitempty"`
}

// Required to satisfy Object interface
func (vmc *VMConfig) GetObjectKind() schema.ObjectKind {
	return &vmc.TypeMeta
}

// VMConfigList is a list of VMConfigs.
type VMConfigList struct {
	metav1.TypeMeta `json:",inline"`
	ListMeta        metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VMConfig      `json:"items"`
}

// Required to satisfy Object interface
func (vmcl *VMConfigList) GetObjectKind() schema.ObjectKind {
	return &vmcl.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (vmcl *VMConfigList) GetListMeta() metav1.List {
	return &vmcl.ListMeta
}
