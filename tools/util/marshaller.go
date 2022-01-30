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
	"regexp"
	"strings"

	v1 "kubevirt.io/api/core/v1"

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
	unstructured.RemoveNestedField(r.Object, "template", "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(r.Object, "spec", "template", "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(r.Object, "status")

	// remove dataSource from PVCs if empty
	templates, exists, err := unstructured.NestedSlice(r.Object, "spec", "dataVolumeTemplates")
	if err != nil {
		return err
	}
	if exists {
		for _, tmpl := range templates {
			template := tmpl.(map[string]interface{})
			_, exists, err = unstructured.NestedString(template, "spec", "pvc", "dataSource")
			if err != nil {
				return err
			}
			if !exists {
				unstructured.RemoveNestedField(template, "spec", "pvc", "dataSource")
			}
		}
		unstructured.SetNestedSlice(r.Object, templates, "spec", "dataVolumeTemplates")
	}
	objects, exists, err := unstructured.NestedSlice(r.Object, "objects")
	if err != nil {
		return err
	}
	if exists {
		for _, obj := range objects {
			object := obj.(map[string]interface{})
			kind, exists, _ := unstructured.NestedString(object, "kind")
			if exists && kind == "PersistentVolumeClaim" {
				_, exists, err = unstructured.NestedString(object, "spec", "dataSource")
				if err != nil {
					return err
				}
				if !exists {
					unstructured.RemoveNestedField(object, "spec", "dataSource")
				}
			}
			unstructured.RemoveNestedField(object, "status", "startFailure")
		}
		unstructured.SetNestedSlice(r.Object, objects, "objects")
	}

	deployments, exists, err := unstructured.NestedSlice(r.Object, "spec", "install", "spec", "deployments")
	if err != nil {
		return err
	}
	if exists {
		for _, obj := range deployments {
			deployment := obj.(map[string]interface{})
			unstructured.RemoveNestedField(deployment, "metadata", "creationTimestamp")
			unstructured.RemoveNestedField(deployment, "spec", "template", "metadata", "creationTimestamp")
			unstructured.RemoveNestedField(deployment, "status")
		}
		unstructured.SetNestedSlice(r.Object, deployments, "spec", "install", "spec", "deployments")
	}

	// remove "managed by operator" label...
	labels, exists, err := unstructured.NestedMap(r.Object, "metadata", "labels")
	if err != nil {
		return err
	}
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

	// fix templates by removing unneeded single quotes...
	s := string(yamlBytes)
	s = strings.Replace(s, "'{{", "{{", -1)
	s = strings.Replace(s, "}}'", "}}", -1)

	// fix double quoted strings by removing unneeded single quotes...
	s = strings.Replace(s, " '\"", " \"", -1)
	s = strings.Replace(s, "\"'\n", "\"\n", -1)

	// The current function is sometimes used on yaml templates, and manipulates them as json/yaml above.
	// However, this only works for simple templates dealing with simple strings.
	// For list values, we need template code to iterate over the slice, and that code is not valid yaml.
	// To work around that, the variable name for the featureGates slice was treated as the first and only list item until now.
	// Therefore, if we're currently handling a template, the featureGates section looks like:
	//      featureGates:
	//      - {{.FeatureGates}}
	// however we want to treat the variable (".FeatureGates" here) as a slice and iterate over it (with a special case for empty list):
	//      featureGates:{{if .FeatureGates}}
	//      {{- range .FeatureGates}}
	//      - {{.}}
	//      {{- end}}{{else}} []{{end}}
	// The replace call below will transform the former into the latter, keeping the variable name ($2) and intendation ($1)
	featureGates, exists, err := unstructured.NestedStringSlice(r.Object, "spec", "configuration", "developerConfiguration", "featureGates")
	if err == nil && exists && len(featureGates) == 1 && strings.HasPrefix(featureGates[0], `{{`) {
		re := regexp.MustCompile(`(?m)featureGates:\n([ \t]+)- \{\{(.*)\}\}`)
		s = re.ReplaceAllString(s, `featureGates:{{if $2}}
$1{{- range $2}}
$1- {{.}}
$1{{- end}}{{else}} []{{end}}`)
	}

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
