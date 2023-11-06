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

package testsuite

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/gomega"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"

	"kubevirt.io/client-go/log"

	putil "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/tests/util"
)

func GetListOfManifests(pathToManifestsDir string) []string {
	var manifests []string
	matchFileName := func(pattern, filename string) bool {
		match, err := filepath.Match(pattern, filename)
		if err != nil {
			panic(err)
		}
		return match
	}
	err := filepath.Walk(pathToManifestsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("ERROR: Can not access a path %q: %v\n", path, err)
			return err
		}
		if !info.IsDir() && matchFileName("*.yaml", info.Name()) {
			manifests = append(manifests, path)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("ERROR: Walking the path %q: %v\n", pathToManifestsDir, err)
		panic(err)
	}
	return manifests
}

func ReadManifestYamlFile(pathToManifest string) []unstructured.Unstructured {
	var objects []unstructured.Unstructured
	stream, err := os.Open(pathToManifest)
	util.PanicOnError(err)
	defer putil.CloseIOAndCheckErr(stream, nil)
	decoder := yaml.NewYAMLOrJSONDecoder(stream, 1024)
	for {
		obj := map[string]interface{}{}
		err := decoder.Decode(&obj)
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		if len(obj) == 0 {
			continue
		}
		objects = append(objects, unstructured.Unstructured{Object: obj})
	}
	return objects
}

func ApplyRawManifest(object unstructured.Unstructured) error {
	virtCli := kubevirt.Client()

	uri := composeResourceURI(object)
	jsonbody, err := object.MarshalJSON()
	util.PanicOnError(err)
	b, err := virtCli.CoreV1().RESTClient().Post().RequestURI(uri).Body(jsonbody).DoRaw(context.Background())
	if err != nil {
		fmt.Printf(fmt.Sprintf("ERROR: Can not apply %s\n, err: %#v", object, err))
		panic(err)
	}
	status := unstructured.Unstructured{}
	return json.Unmarshal(b, &status)
}

func DeleteRawManifest(object unstructured.Unstructured) error {
	virtCli := kubevirt.Client()

	uri := composeResourceURI(object)
	uri = path.Join(uri, object.GetName())

	policy := metav1.DeletePropagationForeground
	options := &metav1.DeleteOptions{PropagationPolicy: &policy}

	log.DefaultLogger().Infof("Calling DELETE on testing manifest: %s", uri)

	EventuallyWithOffset(2, func() error {
		result := virtCli.CoreV1().RESTClient().Delete().RequestURI(uri).Body(options).Do(context.Background())
		return result.Error()
	}, 30*time.Second, 1*time.Second).Should(
		SatisfyAll(HaveOccurred(), WithTransform(k8serrors.IsNotFound, BeTrue())),
		fmt.Sprintf("%s failed to be cleaned up", uri),
	)

	return nil
}

func composeResourceURI(object unstructured.Unstructured) string {
	uri := "/api"
	if object.GetAPIVersion() != "v1" {
		uri += "s"
	}
	uri = path.Join(uri, object.GetAPIVersion())
	if object.GetNamespace() != "" && isNamespaceScoped(object.GroupVersionKind()) {
		uri = path.Join(uri, "namespaces", object.GetNamespace())
	}
	uri = path.Join(uri, strings.ToLower(object.GetKind()))
	if !strings.HasSuffix(object.GetKind(), "s") {
		uri += "s"
	}
	return uri
}

func isNamespaceScoped(kind schema.GroupVersionKind) bool {
	switch kind.Kind {
	case "ClusterRole", "ClusterRoleBinding":
		return false
	}
	return true
}
