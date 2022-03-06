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
 * Copyright 2022 Nvidia
 *
 */

package utils

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/config"
	objUtil "kubevirt.io/kubevirt/tools/perfscale-load-generator/object"
)

func Create(client kubecli.KubevirtClient, replica int, obj *config.ObjectSpec, uuid string) (*unstructured.Unstructured, error) {
	templateData := objUtil.GenerateObjectTemplateData(obj, replica)
	newObject, err := objUtil.RenderObject(templateData, obj.ObjectTemplate)
	if err != nil {
		log.Log.Errorf("error rendering obj: %v", err)
		return nil, err
	}

	config.AddLabels(newObject, uuid)
	if _, err := objUtil.CreateObject(client, newObject); err != nil {
		log.Log.Errorf("error creating obj %s: %v", newObject.GroupVersionKind().Kind, err)
		return nil, err
	}
	return newObject, nil
}
