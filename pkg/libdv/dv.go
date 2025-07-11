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

package libdv

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	instancetypeapi "kubevirt.io/api/instancetype"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	dvRandomNameLength = 12
	defaultVolumeSize  = "512Mi"
)

// dvOption is an option type for the NewDataVolume function
type dvOption func(*v1beta1.DataVolume)

// NewDataVolume Set up a new DataVolume with a random name, a namespace and an optional list of options
func NewDataVolume(options ...dvOption) *v1beta1.DataVolume {
	name := randName()
	dv := &v1beta1.DataVolume{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cdi.kubevirt.io/v1beta1",
			Kind:       "DataVolume",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	for _, option := range options {
		option(dv)
	}

	return dv
}

func WithNamespace(namespace string) dvOption {
	return func(dv *v1beta1.DataVolume) {
		dv.Namespace = namespace
	}
}

func WithName(name string) dvOption {
	return func(dv *v1beta1.DataVolume) {
		dv.ObjectMeta.Name = name
	}
}

func WithAnnotation(ann, val string) dvOption {
	return func(dv *v1beta1.DataVolume) {
		if dv.ObjectMeta.Annotations == nil {
			dv.ObjectMeta.Annotations = make(map[string]string)
		}
		dv.ObjectMeta.Annotations[ann] = val
	}
}

type pvcOption func(*corev1.PersistentVolumeClaimSpec)
type storageOption func(*v1beta1.StorageSpec)

// WithStorage is a dvOption to add a StorageOption spec to the DataVolume
// The function receives an optional list of StorageOption, to override the defaults
//
// The default values are:
// * no storage class
// * access mode from the StorgeProfile
// * volume size of defaultVolumeSize
// * volume mode from the storageProfile
func WithStorage(options ...storageOption) dvOption {
	storage := &v1beta1.StorageSpec{
		Resources: corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{
				"storage": resource.MustParse(defaultVolumeSize),
			},
		},
	}

	for _, opt := range options {
		opt(storage)
	}

	return func(dv *v1beta1.DataVolume) {
		dv.Spec.Storage = storage
	}
}

// WithPVC is a dvOption to add a PVCOption spec to the DataVolume
// The function receives an optional list of pvcOption, to override the defaults
// * access mode of ReadWriteOnce
// * no volume mode. kubernetes default is PersistentVolumeFilesystem
func WithPVC(options ...pvcOption) dvOption {
	pvc := &corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Resources: corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{
				"storage": resource.MustParse(defaultVolumeSize),
			},
		},
	}
	for _, opt := range options {
		opt(pvc)
	}

	return func(dv *v1beta1.DataVolume) {
		dv.Spec.PVC = pvc
	}
}

// withSource is a dvOption to add a DataVolumeSource to the DataVolume
func withSource(s v1beta1.DataVolumeSource) dvOption {
	return func(dv *v1beta1.DataVolume) {
		dv.Spec.Source = &s
	}
}

// WithRegistryURLSource is a dvOption to add a DataVolumeSource to the DataVolume, with a registry and a URL
func WithRegistryURLSource(imageURL string) dvOption {
	return withSource(v1beta1.DataVolumeSource{
		Registry: &v1beta1.DataVolumeSourceRegistry{
			URL: &imageURL,
		},
	})
}

// WithRegistryURLSourceAndPullMethod is a dvOption to add a DataVolumeSource to the DataVolume, with a registry and URL + pull method
func WithRegistryURLSourceAndPullMethod(imageURL string, pullMethod v1beta1.RegistryPullMethod) dvOption {
	return withSource(v1beta1.DataVolumeSource{
		Registry: &v1beta1.DataVolumeSourceRegistry{
			URL:        &imageURL,
			PullMethod: &pullMethod,
		},
	})
}

// WithBlankImageSource is a dvOption to add a blank DataVolumeSource to the DataVolume
func WithBlankImageSource() dvOption {
	return withSource(v1beta1.DataVolumeSource{
		Blank: &v1beta1.DataVolumeBlankImage{},
	})
}

// WithPVCSource is a dvOption to add a DataVolumeSource to the DataVolume, with a PVC source
func WithPVCSource(namespace, name string) dvOption {
	return withSource(v1beta1.DataVolumeSource{
		PVC: &v1beta1.DataVolumeSourcePVC{
			Namespace: namespace,
			Name:      name,
		},
	})
}

// WithForceBindAnnotation adds the "cdi.kubevirt.io/storage.bind.immediate.requested" annotation to the DV,
// with the value of "true"
func WithForceBindAnnotation() dvOption {
	return func(dv *v1beta1.DataVolume) {
		if dv.Annotations == nil {
			dv.Annotations = make(map[string]string)
		}
		dv.Annotations["cdi.kubevirt.io/storage.bind.immediate.requested"] = "true"
	}
}

func WithDefaultInstancetype(kind, name string) dvOption {
	return func(dv *v1beta1.DataVolume) {
		if dv.Labels == nil {
			dv.Labels = map[string]string{}
		}
		dv.Labels[instancetypeapi.DefaultInstancetypeLabel] = name
		dv.Labels[instancetypeapi.DefaultInstancetypeKindLabel] = kind
	}
}

func WithDefaultPreference(kind, name string) dvOption {
	return func(dv *v1beta1.DataVolume) {
		if dv.Labels == nil {
			dv.Labels = map[string]string{}
		}
		dv.Labels[instancetypeapi.DefaultPreferenceLabel] = name
		dv.Labels[instancetypeapi.DefaultPreferenceKindLabel] = kind
	}
}

func WithDataVolumeSourceRef(kind, namespace, name string) dvOption {
	return func(dv *v1beta1.DataVolume) {
		dv.Spec.SourceRef = &v1beta1.DataVolumeSourceRef{
			Kind:      kind,
			Namespace: pointer.P(namespace),
			Name:      name,
		}
	}
}

func randName() string {
	return "test-datavolume-" + rand.String(dvRandomNameLength)
}

// PVC Options
// PVCWithStorageClass add the sc storage class name to the DV
func PVCWithStorageClass(sc string) pvcOption {
	return func(pvc *corev1.PersistentVolumeClaimSpec) {
		if pvc == nil {
			// TODO: Fail here instead? This is programmer error
			return
		}

		pvc.StorageClassName = &sc
	}
}

// Storage Options
// StorageWithStorageClass add the sc storage class name to the DV
func StorageWithStorageClass(sc string) storageOption {
	return func(storage *v1beta1.StorageSpec) {
		if storage == nil {
			// TODO: Fail here instead? This is programmer error
			return
		}

		storage.StorageClassName = &sc
	}
}

// StorageWithVolumeSize overrides the default volume size (defaultVolumeSize), with the size parameter
// The size parameter must be in parsable valid quantity string.
func StorageWithVolumeSize(size string) storageOption {
	return func(storage *v1beta1.StorageSpec) {
		if storage == nil {
			// TODO: Fail here instead? This is programmer error
			return
		}

		storage.Resources.Requests = corev1.ResourceList{"storage": resource.MustParse(size)}
	}
}

// StorageWithoutVolumeSize removes the default volume size, useful for clones
func StorageWithoutVolumeSize() storageOption {
	return func(storage *v1beta1.StorageSpec) {
		if storage != nil {
			storage.Resources.Requests = nil
		}
	}
}

// StorageWithVolumeMode adds the volume mode to the DV
func StorageWithVolumeMode(volumeMode corev1.PersistentVolumeMode) storageOption {
	return func(storage *v1beta1.StorageSpec) {
		if storage == nil {
			// TODO: Fail here instead? This is programmer error
			return
		}

		storage.VolumeMode = &volumeMode
	}
}

// StorageWithBlockVolumeMode adds the PersistentVolumeBlock volume mode to the DV
func StorageWithBlockVolumeMode() storageOption {
	return StorageWithVolumeMode(corev1.PersistentVolumeBlock)
}

// StorageWithFilesystemVolumeMode adds the PersistentVolumeBlock volume mode to the DV
func StorageWithFilesystemVolumeMode() storageOption {
	return StorageWithVolumeMode(corev1.PersistentVolumeFilesystem)
}

// StorageWithAccessMode overrides the DV default access mode (ReadWriteOnce) with the accessMode parameter
func StorageWithAccessMode(accessMode corev1.PersistentVolumeAccessMode) storageOption {
	return func(storage *v1beta1.StorageSpec) {
		if storage == nil {
			// TODO: Fail here instead? This is programmer error
			return
		}

		storage.AccessModes = []corev1.PersistentVolumeAccessMode{accessMode}
	}
}

// StorageWithReadWriteManyAccessMode set the DV access mode to ReadWriteMany
func StorageWithReadWriteManyAccessMode() storageOption {
	return StorageWithAccessMode(corev1.ReadWriteMany)
}
