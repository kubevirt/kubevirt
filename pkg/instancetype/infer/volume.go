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
 * Copyright The KubeVirt Authors
 *
 */
package infer

import (
	"context"
	"errors"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	virtv1 "kubevirt.io/api/core/v1"
)

const (
	unsupportedVolumeTypeFmt          = "unable to infer defaults from volume %s as type is not supported"
	missingLabelFmt                   = "unable to find required %s label on the volume"
	unsupportedDataVolumeSource       = "unable to infer defaults from DataVolumeSpec as DataVolumeSource is not supported"
	missingDataVolumeSourcePVC        = "unable to infer defaults from DataSource that doesn't provide DataVolumeSourcePVC"
	unsupportedDataVolumeSourceRefFmt = "unable to infer defaults from DataVolumeSourceRef as Kind %s is not supported"
)

/*
Defaults will be inferred from the following combinations of DataVolumeSources, DataVolumeTemplates, DataSources and PVCs:

Volume -> PersistentVolumeClaimVolumeSource -> PersistentVolumeClaim
Volume -> DataVolumeSource -> DataVolume
Volume -> DataVolumeSource -> DataVolumeSourcePVC -> PersistentVolumeClaim
Volume -> DataVolumeSource -> DataVolumeSourceRef -> DataSource
Volume -> DataVolumeSource -> DataVolumeSourceRef -> DataSource -> PersistentVolumeClaim
Volume -> DataVolumeSource -> DataVolumeTemplate -> DataVolumeSourcePVC -> PersistentVolumeClaim
Volume -> DataVolumeSource -> DataVolumeTemplate -> DataVolumeSourceRef -> DataSource
Volume -> DataVolumeSource -> DataVolumeTemplate -> DataVolumeSourceRef -> DataSource -> PersistentVolumeClaim
*/
func (h *handler) fromVolumes(
	vm *virtv1.VirtualMachine, inferFromVolumeName, defaultNameLabel, defaultKindLabel string,
) (defaultName, defaultKind string, err error) {
	for _, volume := range vm.Spec.Template.Spec.Volumes {
		if volume.Name != inferFromVolumeName {
			continue
		}
		if volume.PersistentVolumeClaim != nil {
			return h.fromPVC(volume.PersistentVolumeClaim.ClaimName, vm.Namespace, defaultNameLabel, defaultKindLabel)
		}
		if volume.DataVolume != nil {
			return h.fromDataVolume(vm, volume.DataVolume.Name, defaultNameLabel, defaultKindLabel)
		}
		return "", "", NewIgnoreableInferenceError(fmt.Errorf(unsupportedVolumeTypeFmt, inferFromVolumeName))
	}
	return "", "", fmt.Errorf("unable to find volume %s to infer defaults", inferFromVolumeName)
}

func fromLabels(labels map[string]string, defaultNameLabel, defaultKindLabel string) (defaultName, defaultKind string, err error) {
	defaultName, hasLabel := labels[defaultNameLabel]
	if !hasLabel {
		return "", "", NewIgnoreableInferenceError(fmt.Errorf(missingLabelFmt, defaultNameLabel))
	}
	return defaultName, labels[defaultKindLabel], nil
}

func (h *handler) fromPVC(pvcName, pvcNamespace, defaultNameLabel, defaultKindLabel string) (defaultName, defaultKind string, err error) {
	pvc, err := h.virtClient.CoreV1().PersistentVolumeClaims(pvcNamespace).Get(context.Background(), pvcName, metav1.GetOptions{})
	if err != nil {
		return "", "", err
	}
	return fromLabels(pvc.Labels, defaultNameLabel, defaultKindLabel)
}

func (h *handler) fromDataVolume(
	vm *virtv1.VirtualMachine, dvName, defaultNameLabel, defaultKindLabel string,
) (defaultName, defaultKind string, err error) {
	if len(vm.Spec.DataVolumeTemplates) > 0 {
		for _, dvt := range vm.Spec.DataVolumeTemplates {
			if dvt.Name != dvName {
				continue
			}
			dvtSpec := dvt.Spec
			return h.fromDataVolumeSpec(&dvtSpec, defaultNameLabel, defaultKindLabel, vm.Namespace)
		}
	}
	dv, err := h.virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Get(context.Background(), dvName, metav1.GetOptions{})
	if err != nil {
		// Handle garbage collected DataVolumes by attempting to lookup the PVC using the name of the DataVolume in the VM namespace
		if k8serrors.IsNotFound(err) {
			return h.fromPVC(dvName, vm.Namespace, defaultNameLabel, defaultKindLabel)
		}
		return "", "", err
	}
	// Check the DataVolume for any labels before checking the underlying PVC
	defaultName, defaultKind, err = fromLabels(dv.Labels, defaultNameLabel, defaultKindLabel)
	if err == nil {
		return defaultName, defaultKind, nil
	}
	return h.fromDataVolumeSpec(&dv.Spec, defaultNameLabel, defaultKindLabel, vm.Namespace)
}

func (h *handler) fromDataVolumeSpec(
	dataVolumeSpec *cdiv1beta1.DataVolumeSpec, defaultNameLabel, defaultKindLabel, vmNameSpace string,
) (defaultName, defaultKind string, err error) {
	if dataVolumeSpec != nil && dataVolumeSpec.Source != nil && dataVolumeSpec.Source.PVC != nil {
		return h.fromPVC(dataVolumeSpec.Source.PVC.Name, dataVolumeSpec.Source.PVC.Namespace, defaultNameLabel, defaultKindLabel)
	}
	if dataVolumeSpec != nil && dataVolumeSpec.SourceRef != nil {
		return h.fromDataVolumeSourceRef(dataVolumeSpec.SourceRef, defaultNameLabel, defaultKindLabel, vmNameSpace)
	}
	return "", "", NewIgnoreableInferenceError(errors.New(unsupportedDataVolumeSource))
}

func (h *handler) fromDataSource(
	dataSourceName, dataSourceNamespace, defaultNameLabel, defaultKindLabel string,
) (defaultName, defaultKind string, err error) {
	ds, err := h.virtClient.CdiClient().CdiV1beta1().DataSources(dataSourceNamespace).Get(
		context.Background(), dataSourceName, metav1.GetOptions{})
	if err != nil {
		return "", "", err
	}
	// Check the DataSource for any labels before checking the underlying PVC
	defaultName, defaultKind, err = fromLabels(ds.Labels, defaultNameLabel, defaultKindLabel)
	if err == nil {
		return defaultName, defaultKind, nil
	}
	if ds.Spec.Source.PVC != nil {
		return h.fromPVC(ds.Spec.Source.PVC.Name, ds.Spec.Source.PVC.Namespace, defaultNameLabel, defaultKindLabel)
	}
	return "", "", NewIgnoreableInferenceError(errors.New(missingDataVolumeSourcePVC))
}

func (h *handler) fromDataVolumeSourceRef(
	sourceRef *cdiv1beta1.DataVolumeSourceRef, defaultNameLabel, defaultKindLabel, vmNameSpace string,
) (defaultName, defaultKind string, err error) {
	if sourceRef.Kind == "DataSource" {
		// The namespace can be left blank here with the assumption that the DataSource is in the same namespace as the VM
		namespace := vmNameSpace
		if sourceRef.Namespace != nil {
			namespace = *sourceRef.Namespace
		}
		return h.fromDataSource(sourceRef.Name, namespace, defaultNameLabel, defaultKindLabel)
	}
	return "", "", NewIgnoreableInferenceError(fmt.Errorf(unsupportedDataVolumeSourceRefFmt, sourceRef.Kind))
}
