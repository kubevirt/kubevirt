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

// WithPVC is a dvOption to add a PVCOption spec to the DataVolume
func WithPVC(storageClass string, size string, accessMode corev1.PersistentVolumeAccessMode, volumeMode corev1.PersistentVolumeMode) dvOption {
	pvc := &corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{accessMode},
		VolumeMode:  &volumeMode,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				"storage": resource.MustParse(size),
			},
		},
		StorageClassName: &storageClass,
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
