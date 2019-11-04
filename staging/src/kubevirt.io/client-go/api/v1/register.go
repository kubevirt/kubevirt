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
 * Copyright 2019 Red Hat, Inc.
 *
 */
package v1

import (
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
)

// GroupName is the group name use in this package
const GroupName = "kubevirt.io"
const SubresourceGroupName = "subresources.kubevirt.io"

var (
	ApiLatestVersion            = "v1alpha3"
	ApiSupportedWebhookVersions = []string{"v1alpha3"}
	ApiStorageVersion           = "v1alpha3"
	ApiSupportedVersions        = []extv1beta1.CustomResourceDefinitionVersion{
		{
			Name:    "v1alpha3",
			Served:  true,
			Storage: true,
		},
	}
)

var (
	// GroupVersion is the latest group version for the KubeVirt api
	GroupVersion = schema.GroupVersion{Group: GroupName, Version: ApiLatestVersion}

	// StorageGroupVersion is the group version our api is persistented internally as
	StorageGroupVersion = schema.GroupVersion{Group: GroupName, Version: ApiStorageVersion}

	// GroupVersions is group version list used to register these objects
	// The preferred group version is the first item in the list.
	GroupVersions = []schema.GroupVersion{GroupVersion}

	// SubresourceGroupVersions is group version list used to register these objects
	// The preferred group version is the first item in the list.
	SubresourceGroupVersions = []schema.GroupVersion{{Group: SubresourceGroupName, Version: "v1alpha3"}}

	// SubresourceStorageGroupVersion is the group version our api is persistented internally as
	SubresourceStorageGroupVersion = schema.GroupVersion{Group: SubresourceGroupName, Version: ApiStorageVersion}
)

var (
	// GroupVersionKind
	VirtualMachineInstanceGroupVersionKind           = schema.GroupVersionKind{Group: GroupName, Version: GroupVersion.Version, Kind: "VirtualMachineInstance"}
	VirtualMachineInstanceReplicaSetGroupVersionKind = schema.GroupVersionKind{Group: GroupName, Version: GroupVersion.Version, Kind: "VirtualMachineInstanceReplicaSet"}
	VirtualMachineInstancePresetGroupVersionKind     = schema.GroupVersionKind{Group: GroupName, Version: GroupVersion.Version, Kind: "VirtualMachineInstancePreset"}
	VirtualMachineGroupVersionKind                   = schema.GroupVersionKind{Group: GroupName, Version: GroupVersion.Version, Kind: "VirtualMachine"}
	VirtualMachineInstanceMigrationGroupVersionKind  = schema.GroupVersionKind{Group: GroupName, Version: GroupVersion.Version, Kind: "VirtualMachineInstanceMigration"}
	KubeVirtGroupVersionKind                         = schema.GroupVersionKind{Group: GroupName, Version: GroupVersion.Version, Kind: "KubeVirt"}
)

var (
	SchemeBuilder  = runtime.NewSchemeBuilder(addKnownTypes)
	Scheme         = runtime.NewScheme()
	AddToScheme    = SchemeBuilder.AddToScheme
	Codecs         = serializer.NewCodecFactory(Scheme)
	ParameterCodec = runtime.NewParameterCodec(Scheme)
)

func init() {
	AddToScheme(Scheme)
	AddToScheme(scheme.Scheme)
}

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {

	for _, groupVersion := range GroupVersions {
		scheme.AddKnownTypes(groupVersion,
			&VirtualMachineInstance{},
			&VirtualMachineInstanceList{},
			&VirtualMachineInstanceReplicaSet{},
			&VirtualMachineInstanceReplicaSetList{},
			&VirtualMachineInstancePreset{},
			&VirtualMachineInstancePresetList{},
			&VirtualMachineInstanceMigration{},
			&VirtualMachineInstanceMigrationList{},
			&VirtualMachine{},
			&VirtualMachineList{},
			&KubeVirt{},
			&KubeVirtList{},
		)
		metav1.AddToGroupVersion(scheme, groupVersion)
	}

	return nil
}
