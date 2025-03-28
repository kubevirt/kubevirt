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
 * Copyright 2021 IBM, Inc.
 *
 */

package object

import (
	"bytes"
	"fmt"
	"html/template"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	kvv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tools/perfscale-load-generator/config"
)

const (
	Replica         = "replica"
	Iteration       = "iteration"
	UUID            = "uuid"
	Namespace       = "namespace"
	containerPrefix = "containerPrefix"
	containerTag    = "containerTag"
)

var codecs serializer.CodecFactory

func init() {
	scheme := runtime.NewScheme()
	// GroupName is the group name use in this package
	const GroupName = "kubevirt.io"
	// GroupVersions is group version list used to register these objects
	// The preferred group version is the first item in the list.
	groupVersion := schema.GroupVersion{Group: GroupName, Version: "v1"}
	scheme.AddKnownTypes(groupVersion,
		&kvv1.VirtualMachineInstance{},
		&kvv1.VirtualMachineInstanceReplicaSet{},
		&kvv1.VirtualMachineInstancePreset{},
		&kvv1.VirtualMachineInstanceMigration{},
		&kvv1.VirtualMachine{},
	)
	codecs = serializer.NewCodecFactory(scheme)
}

// RenderObject creates a Kubernetes Unstructured object from a template
func RenderObject(templateData map[string]interface{}, objectSpec []byte) (*unstructured.Unstructured, error) {
	var renderedObj bytes.Buffer

	var t *template.Template
	var err error
	if t, err = template.New("").Parse(string(objectSpec)); err != nil {
		return nil, fmt.Errorf("template parsing error: %s", err)
	}
	if err = t.Execute(&renderedObj, templateData); err != nil {
		return nil, fmt.Errorf("object rendering error: %s", err)
	}

	newObject := &unstructured.Unstructured{}
	if _, _, err = codecs.UniversalDeserializer().Decode(renderedObj.Bytes(), nil, newObject); err != nil {
		return nil, fmt.Errorf("error decoding YAML: %s", err)
	}
	return newObject, nil
}

func GenerateObjectTemplateData(obj *config.ObjectSpec, replica int) map[string]interface{} {
	templateData := map[string]interface{}{
		config.Replica: replica,
	}

	for k, v := range obj.InputVars {
		// Verify if the containerPrefix and containerTag are defined in the template, otherwise use the default values
		if k == containerPrefix {
			if v == "" {
				v = config.ContainerPrefix
			}
		}
		if k == containerTag {
			if v == "" {
				v = config.ContainerTag
			}
		}

		templateData[k] = v
	}
	return templateData
}
