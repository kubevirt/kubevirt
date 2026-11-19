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
 * Copyright The KubeVirt Authors.
 *
 */

package testsuite

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/onsi/gomega/format"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest/fake"
)

func TestApplyRawManifest(t *testing.T) {
	for _, kind := range []string{"Secret", "ConfigMap", "Pod", "ClusterRole"} {
		for _, statusCode := range []int{http.StatusForbidden, http.StatusUnprocessableEntity, http.StatusCreated} {
			t.Run(fmt.Sprintf("%s/%d", kind, statusCode), func(t *testing.T) {
				const sensitive = "synthetic-sensitive-manifest-value"
				object := unstructured.Unstructured{Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       kind,
					"metadata": map[string]interface{}{
						"name":        "test-resource",
						"namespace":   "test-namespace",
						"annotations": map[string]interface{}{"sensitive": sensitive},
					},
				}}
				switch kind {
				case "Secret":
					object.Object["data"] = map[string]interface{}{"password": "c3ludGhldGljLXNlbnNpdGl2ZQ=="}
					object.Object["stringData"] = map[string]interface{}{"password": sensitive}
				case "ConfigMap":
					object.Object["data"] = map[string]interface{}{"password": sensitive}
				case "Pod":
					object.Object["spec"] = map[string]interface{}{"containers": []interface{}{
						map[string]interface{}{"name": "app", "image": "test-image", "env": []interface{}{
							map[string]interface{}{"name": "PASSWORD", "value": sensitive},
						}},
					}}
				}
				if kind == "ClusterRole" {
					object.SetAPIVersion("rbac.authorization.k8s.io/v1")
					object.SetNamespace("")
				}
				var requests int
				client := &fake.RESTClient{
					GroupVersion: schema.GroupVersion{Version: "v1"},
					Client: fake.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
						requests++
						if req.Method != http.MethodPost || req.URL.Path != composeResourceURI(object) {
							t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
						}
						body, err := io.ReadAll(req.Body)
						if err != nil {
							t.Fatal(err)
						}
						if !bytes.Contains(body, []byte(sensitive)) {
							t.Error("request did not contain the synthetic sensitive value")
						}
						response := "synthetic API rejection"
						if statusCode == http.StatusCreated {
							response = `{"apiVersion":"v1","kind":"Secret"}`
						}
						return &http.Response{
							StatusCode: statusCode,
							Header:     http.Header{"Content-Type": []string{"text/plain"}},
							Body:       io.NopCloser(strings.NewReader(response)),
						}, nil
					}),
				}
				// These subtests run serially because stdout is process-wide.
				originalStdout := os.Stdout
				reader, writer, err := os.Pipe()
				if err != nil {
					t.Fatal(err)
				}
				defer reader.Close()
				defer writer.Close()
				defer func() { os.Stdout = originalStdout }()
				os.Stdout = writer
				applyErr := applyRawManifest(client, object)
				writer.Close()
				os.Stdout = originalStdout
				output, err := io.ReadAll(reader)
				if err != nil {
					t.Fatal(err)
				}
				if requests != 1 {
					t.Errorf("got %d requests, want 1", requests)
				}
				diagnostic := append(output, []byte(format.Object(applyErr, 0))...)
				for _, value := range []string{sensitive, "c3ludGhldGljLXNlbnNpdGl2ZQ==", "stringData", "annotations"} {
					if bytes.Contains(diagnostic, []byte(value)) {
						t.Error("apply failure output contains manifest contents")
						break
					}
				}
				if statusCode == http.StatusCreated {
					if applyErr != nil {
						t.Errorf("successful apply returned an error: %v", applyErr)
					}
					if len(output) != 0 {
						t.Error("successful apply produced failure output")
					}
					return
				}
				if applyErr == nil {
					t.Fatal("failed apply did not return an error")
				}
				for _, metadata := range []string{kind, object.GetName(), fmt.Sprintf("namespace %q", object.GetNamespace())} {
					if !strings.Contains(applyErr.Error(), metadata) {
						t.Errorf("returned error is missing resource metadata %q", metadata)
					}
				}
				var statusErr *k8serrors.StatusError
				if !errors.As(applyErr, &statusErr) {
					t.Error("wrapped API error was lost")
				}
				if len(output) != 0 {
					t.Error("failed apply produced duplicate failure output")
				}
				if statusCode == http.StatusForbidden && !k8serrors.IsForbidden(applyErr) {
					t.Error("API error classification was lost")
				}
			})
		}
	}
}
