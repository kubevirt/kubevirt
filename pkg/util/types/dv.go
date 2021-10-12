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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package types

import (
	"fmt"

	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/client-go/api/v1"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
)

type CloneSource struct {
	Namespace string
	Name      string
}

func GetCloneSourceFromCache(vm *virtv1.VirtualMachine, dvSpec *cdiv1.DataVolumeSpec, dataSourceCache cache.Store) (*CloneSource, error) {
	var cloneSource *CloneSource
	if dvSpec.Source != nil && dvSpec.Source.PVC != nil {
		cloneSource = &CloneSource{
			Namespace: dvSpec.Source.PVC.Namespace,
			Name:      dvSpec.Source.PVC.Name,
		}

		if cloneSource.Namespace == "" {
			cloneSource.Namespace = vm.Namespace
		}
	} else if dvSpec.SourceRef != nil && dvSpec.SourceRef.Kind == "DataSource" {
		ns := vm.Namespace
		if dvSpec.SourceRef.Namespace != nil {
			ns = *dvSpec.SourceRef.Namespace
		}

		key := fmt.Sprintf("%v/%v", ns, dvSpec.SourceRef.Name)
		obj, exists, err := dataSourceCache.GetByKey(key)
		if err != nil {
			return nil, err
		} else if !exists {
			return nil, fmt.Errorf("DataSource %s/%s does not exist", ns, dvSpec.SourceRef.Name)
		}

		ds := obj.(*cdiv1.DataSource)

		if ds.Spec.Source.PVC != nil {
			cloneSource = &CloneSource{
				Namespace: ds.Spec.Source.PVC.Namespace,
				Name:      ds.Spec.Source.PVC.Name,
			}

			if cloneSource.Namespace == "" {
				cloneSource.Namespace = ns
			}
		}
	}

	return cloneSource, nil
}
