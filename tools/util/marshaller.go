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
 * Copyright 2018 Red Hat, Inc.
 *
 */
package util

import (
	"encoding/json"
	"io"
	"strings"

	"kubevirt.io/kubevirt/pkg/api/v1"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func MarshallObject(obj interface{}, writer io.Writer) error {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	var r unstructured.Unstructured
	if err := json.Unmarshal(jsonBytes, &r.Object); err != nil {
		return err
	}

	// remove status and metadata.creationTimestamp
	unstructured.RemoveNestedField(r.Object, "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(r.Object, "spec", "template", "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(r.Object, "status")

	// remove dataSource from PVCs if empty
	templates, exists, err := unstructured.NestedSlice(r.Object, "spec", "dataVolumeTemplates")
	if exists {
		for _, tmpl := range templates {
			template := tmpl.(map[string]interface{})
			_, exists, err = unstructured.NestedString(template, "spec", "pvc", "dataSource")
			if !exists {
				unstructured.RemoveNestedField(template, "spec", "pvc", "dataSource")
			}
		}
		unstructured.SetNestedSlice(r.Object, templates, "spec", "dataVolumeTemplates")
	}
	objects, exists, err := unstructured.NestedSlice(r.Object, "objects")
	if exists {
		for _, obj := range objects {
			object := obj.(map[string]interface{})
			kind, exists, _ := unstructured.NestedString(object, "kind")
			if exists && kind == "PersistentVolumeClaim" {
				_, exists, err = unstructured.NestedString(object, "spec", "dataSource")
				if !exists {
					unstructured.RemoveNestedField(object, "spec", "dataSource")
				}
			}
		}
		unstructured.SetNestedSlice(r.Object, objects, "objects")
	}

	// remove "managed by operator" label...
	labels, exists, err := unstructured.NestedMap(r.Object, "metadata", "labels")
	if exists {
		delete(labels, v1.ManagedByLabel)
		unstructured.SetNestedMap(r.Object, labels, "metadata", "labels")
	}

	jsonBytes, err = json.Marshal(r.Object)
	if err != nil {
		return err
	}

	yamlBytes, err := yaml.JSONToYAML(jsonBytes)
	if err != nil {
		return err
	}

	// fix templates by removing quotes...
	s := string(yamlBytes)
	s = strings.Replace(s, "'{{", "{{", -1)
	s = strings.Replace(s, "}}'", "}}", -1)
	yamlBytes = []byte(s)

	_, err = writer.Write([]byte("---\n"))
	if err != nil {
		return err
	}

	_, err = writer.Write(yamlBytes)
	if err != nil {
		return err
	}

	return nil
}
