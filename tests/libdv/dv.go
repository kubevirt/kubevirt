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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package libdv

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

const (
	dvRandomNameLength = 12
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

type pvcOption func(*corev1.PersistentVolumeClaimSpec)

// WithPVC is a dvOption to add a PVCOption spec to the DataVolume
// The function receives an optional list of pvcOption, to override the defaults
//
// The default values are:
// * no storage class
// * access mode of ReadWriteOnce
// * volume size of cd.CirrosVolumeSize
// * no volume mode. kubernetes default is PersistentVolumeFilesystem
func WithPVC(options ...pvcOption) dvOption {
	pvc := &corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Resources: corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{
				"storage": resource.MustParse(cd.CirrosVolumeSize),
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

func randName() string {
	return "test-datavolume-" + rand.String(dvRandomNameLength)
}

// PVC Options

// PVCWithStorageClass add the sc storage class name to the DV
func PVCWithStorageClass(sc string) pvcOption {
	return func(pvc *corev1.PersistentVolumeClaimSpec) {
		if pvc == nil {
			return
		}

		pvc.StorageClassName = &sc
	}
}

// PVCWithVolumeSize overrides the default volume size (cd.CirrosVolumeSize), with the size parameter
// The size parameter must be in parsable valid quantity string.
func PVCWithVolumeSize(size string) pvcOption {
	return func(pvc *corev1.PersistentVolumeClaimSpec) {
		if pvc == nil {
			return
		}

		pvc.Resources.Requests = corev1.ResourceList{"storage": resource.MustParse(size)}
	}
}

// PVCWithVolumeMode adds the volume mode to the DV
func PVCWithVolumeMode(volumeMode corev1.PersistentVolumeMode) pvcOption {
	return func(pvc *corev1.PersistentVolumeClaimSpec) {
		if pvc == nil {
			return
		}

		pvc.VolumeMode = &volumeMode
	}
}

// PVCWithBlockVolumeMode adds the PersistentVolumeBlock volume mode to the DV
func PVCWithBlockVolumeMode() pvcOption {
	return PVCWithVolumeMode(corev1.PersistentVolumeBlock)
}

// PVCWithAccessMode overrides the DV default access mode (ReadWriteOnce) with the accessMode parameter
func PVCWithAccessMode(accessMode corev1.PersistentVolumeAccessMode) pvcOption {
	return func(pvc *corev1.PersistentVolumeClaimSpec) {
		if pvc == nil {
			return
		}

		pvc.AccessModes = []corev1.PersistentVolumeAccessMode{accessMode}
	}
}

// PVCWithReadWriteManyAccessMode set the DV access mode to ReadWriteMany
func PVCWithReadWriteManyAccessMode() pvcOption {
	return PVCWithAccessMode(corev1.ReadWriteMany)
}
