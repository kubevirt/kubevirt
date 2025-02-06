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
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"

	"kubevirt.io/api/core"
)

const SubresourceGroupName = "subresources.kubevirt.io"
const KubeVirtClientGoSchemeRegistrationVersionEnvVar = "KUBEVIRT_CLIENT_GO_SCHEME_REGISTRATION_VERSION"

var (
	ApiLatestVersion            = "v1"
	ApiSupportedWebhookVersions = []string{"v1alpha3", "v1"}
	ApiStorageVersion           = "v1"
	ApiSupportedVersions        = []extv1.CustomResourceDefinitionVersion{
		{
			Name:    "v1",
			Served:  true,
			Storage: true,
		},
		{
			Name:               "v1alpha3",
			Served:             true,
			Storage:            false,
			Deprecated:         true,
			DeprecationWarning: pointer.String("kubevirt.io/v1alpha3 is now deprecated and will be removed in a future release."),
		},
	}
)

var (
	// GroupVersion is the latest group version for the KubeVirt api
	GroupVersion       = schema.GroupVersion{Group: core.GroupName, Version: ApiLatestVersion}
	SchemeGroupVersion = schema.GroupVersion{Group: core.GroupName, Version: ApiLatestVersion}

	// StorageGroupVersion is the group version our api is persistented internally as
	StorageGroupVersion = schema.GroupVersion{Group: core.GroupName, Version: ApiStorageVersion}

	// GroupVersions is group version list used to register these objects
	// The preferred group version is the first item in the list.
	GroupVersions = []schema.GroupVersion{{Group: core.GroupName, Version: "v1"}, {Group: core.GroupName, Version: "v1alpha3"}}

	// SubresourceGroupVersions is group version list used to register these objects
	// The preferred group version is the first item in the list.
	SubresourceGroupVersions = []schema.GroupVersion{{Group: SubresourceGroupName, Version: ApiLatestVersion}, {Group: SubresourceGroupName, Version: "v1alpha3"}}

	// SubresourceStorageGroupVersion is the group version our api is persistented internally as
	SubresourceStorageGroupVersion = schema.GroupVersion{Group: SubresourceGroupName, Version: ApiStorageVersion}
)

var (
	// GroupVersionKind
	VirtualMachineInstanceGroupVersionKind           = schema.GroupVersionKind{Group: core.GroupName, Version: GroupVersion.Version, Kind: "VirtualMachineInstance"}
	VirtualMachineInstanceReplicaSetGroupVersionKind = schema.GroupVersionKind{Group: core.GroupName, Version: GroupVersion.Version, Kind: "VirtualMachineInstanceReplicaSet"}
	VirtualMachineInstancePresetGroupVersionKind     = schema.GroupVersionKind{Group: core.GroupName, Version: GroupVersion.Version, Kind: "VirtualMachineInstancePreset"}
	VirtualMachineGroupVersionKind                   = schema.GroupVersionKind{Group: core.GroupName, Version: GroupVersion.Version, Kind: "VirtualMachine"}
	VirtualMachineInstanceMigrationGroupVersionKind  = schema.GroupVersionKind{Group: core.GroupName, Version: GroupVersion.Version, Kind: "VirtualMachineInstanceMigration"}
	KubeVirtGroupVersionKind                         = schema.GroupVersionKind{Group: core.GroupName, Version: GroupVersion.Version, Kind: "KubeVirt"}
)

var (
	SchemeBuilder = runtime.NewSchemeBuilder(AddKnownTypesGenerator([]schema.GroupVersion{GroupVersion}))
	AddToScheme   = SchemeBuilder.AddToScheme
)

func AddKnownTypesGenerator(groupVersions []schema.GroupVersion) func(scheme *runtime.Scheme) error {

	// Adds the list of known types to api.Scheme.
	return func(scheme *runtime.Scheme) error {

		for _, groupVersion := range groupVersions {
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
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return GroupVersion.WithResource(resource).GroupResource()
}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}
