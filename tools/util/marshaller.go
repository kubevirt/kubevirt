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

// this file directly copy/pasted from KubeVirt project here:
// https://github.com/kubevirt/kubevirt/blob/master/tools/util/marshaller.go

package util

import (
	"encoding/json"
	"io"
	"regexp"
	"strings"

	corev1 "kubevirt.io/client-go/apis/core/v1"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func MarshallObject(obj interface{}, writer io.Writer) error {
	r, err := unmarshalToUnstructured(obj)
	if err != nil {
		return err
	}

	cleanupNonSpecFields(r)
	yamlBytes, err := objectToByteArray(r)
	if err != nil {
		return err
	}

	yamlBytes = fixQuoteIssues(yamlBytes)
	return writeOutputWithYamlSeparator(writer, yamlBytes)
}

func unmarshalToUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return &unstructured.Unstructured{}, err
	}

	var r unstructured.Unstructured
	if err := json.Unmarshal(jsonBytes, &r.Object); err != nil {
		return &unstructured.Unstructured{}, err
	}
	return &r, nil
}

func objectToByteArray(r *unstructured.Unstructured) ([]byte, error) {
	jsonBytes2, err := json.Marshal(r.Object)
	if err != nil {
		return nil, err
	}

	yamlBytes, err := yaml.JSONToYAML(jsonBytes2)
	if err != nil {
		return nil, err
	}
	return yamlBytes, nil
}

func fixQuoteIssues(yamlBytes []byte) []byte {
	// fix templates by removing unneeded single quotes...
	s := string(yamlBytes)
	re := regexp.MustCompile(`'({{.*?}})'`)
	s = re.ReplaceAllString(s, "$1")

	// fix double quoted strings by removing unneeded single quotes...
	s = strings.Replace(s, " '\"", " \"", -1)
	s = strings.Replace(s, "\"'\n", "\"\n", -1)

	// fix quoted empty square brackets by removing unneeded single quotes...
	s = strings.Replace(s, " '[]'", " []", -1)

	yamlBytes = []byte(s)
	return yamlBytes
}

func writeOutputWithYamlSeparator(writer io.Writer, yamlBytes []byte) error {
	_, err := writer.Write([]byte("---\n"))
	if err != nil {
		return err
	}

	_, err = writer.Write(yamlBytes)
	if err != nil {
		return err
	}
	return nil
}

func cleanupNonSpecFields(r *unstructured.Unstructured) {
	cleanupNonSpecFieldsFromMainObject(r)
	cleanupDataSourceFromTemplates(r)
	cleanupDataSourceFromPVC(r)
	cleanupNonSpecFieldsFromDeployments(r)
	cleanupLabels(r)
}

func cleanupNonSpecFieldsFromMainObject(r *unstructured.Unstructured) {
	// remove status and metadata.creationTimestamp
	unstructured.RemoveNestedField(r.Object, "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(r.Object, "template", "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(r.Object, "spec", "template", "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(r.Object, "status")
}

func cleanupDataSourceFromTemplates(r *unstructured.Unstructured) {
	// remove dataSource from PVCs if empty
	templates, exists, _ := unstructured.NestedSlice(r.Object, "spec", "dataVolumeTemplates")
	if exists {
		for _, tmpl := range templates {
			template := tmpl.(map[string]interface{})
			_, exists, _ = unstructured.NestedString(template, "spec", "pvc", "dataSource")
			if !exists {
				unstructured.RemoveNestedField(template, "spec", "pvc", "dataSource")
			}
		}
		_ = unstructured.SetNestedSlice(r.Object, templates, "spec", "dataVolumeTemplates")
	}
}

func cleanupDataSourceFromPVC(r *unstructured.Unstructured) {
	objects, exists, _ := unstructured.NestedSlice(r.Object, "objects")
	if exists {
		for _, obj := range objects {
			object := obj.(map[string]interface{})
			kind, exists, _ := unstructured.NestedString(object, "kind")
			if exists && kind == "PersistentVolumeClaim" {
				_, exists, _ = unstructured.NestedString(object, "spec", "dataSource")
				if !exists {
					unstructured.RemoveNestedField(object, "spec", "dataSource")
				}
			}
		}
		_ = unstructured.SetNestedSlice(r.Object, objects, "objects")
	}
}

func cleanupNonSpecFieldsFromDeployments(r *unstructured.Unstructured) {
	deployments, exists, _ := unstructured.NestedSlice(r.Object, "spec", "install", "spec", "deployments")
	if exists {
		for _, obj := range deployments {
			deployment := obj.(map[string]interface{})
			unstructured.RemoveNestedField(deployment, "metadata", "creationTimestamp")
			unstructured.RemoveNestedField(deployment, "spec", "template", "metadata", "creationTimestamp")
			unstructured.RemoveNestedField(deployment, "status")
		}
		_ = unstructured.SetNestedSlice(r.Object, deployments, "spec", "install", "spec", "deployments")
	}
}

func cleanupLabels(r *unstructured.Unstructured) {
	// remove "managed by operator" label...
	labels, exists, _ := unstructured.NestedMap(r.Object, "metadata", "labels")
	if exists {
		delete(labels, corev1.ManagedByLabel)
		_ = unstructured.SetNestedMap(r.Object, labels, "metadata", "labels")
	}
}
