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
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"kubevirt.io/kubevirt/tests/flags"
)

func TestGetListOfManifestsUsesConfiguredRenderedDirectory(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "custom-deployment.yaml")
	if err := os.WriteFile(manifest, []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: helper\n  namespace: custom-infra\n"), 0600); err != nil {
		t.Fatal(err)
	}
	previous := flags.PathToTestingInfrastrucureManifests
	flags.PathToTestingInfrastrucureManifests = dir
	t.Cleanup(func() { flags.PathToTestingInfrastrucureManifests = previous })
	if got := GetListOfManifests(); !reflect.DeepEqual(got, []string{manifest}) {
		t.Fatalf("manifest list = %v, want [%s]", got, manifest)
	}

	flags.PathToTestingInfrastrucureManifests = ""
	previousTestingPath := flags.TestingManifestPath
	flags.TestingManifestPath = dir
	t.Cleanup(func() { flags.TestingManifestPath = previousTestingPath })
	if got := GetListOfManifests(); !reflect.DeepEqual(got, []string{manifest}) {
		t.Fatalf("manifest list via -testing-manifest-path = %v, want [%s]", got, manifest)
	}

	flags.PathToTestingInfrastrucureManifests = dir
	flags.TestingManifestPath = t.TempDir()
	if got := TestingInfrastructureManifestsDir(); got != dir {
		t.Fatalf("resolved manifests directory = %q, want custom directory %q", got, dir)
	}
}

func TestTestingInfrastructureActionsUseLoadedManifestObjects(t *testing.T) {
	gomega.RegisterTestingT(t)

	dir := t.TempDir()
	manifest := filepath.Join(dir, "test-service.yaml")
	writeManifest := func(name string) {
		t.Helper()
		content := "apiVersion: v1\nkind: Service\nmetadata:\n  name: " + name + "\n  namespace: test-infra\n"
		if err := os.WriteFile(manifest, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	writeManifest("original")

	previous := flags.PathToTestingInfrastrucureManifests
	flags.PathToTestingInfrastrucureManifests = dir
	t.Cleanup(func() { flags.PathToTestingInfrastrucureManifests = previous })
	objects := testingInfrastructureObjects()
	if err := validateTestingInfrastructureObjects(objects); err != nil {
		t.Fatal(err)
	}

	// A later manifest regeneration must not change the objects used by an
	// already-started suite.
	writeManifest("replacement")
	var actedOn []string
	deployOrWipeTestingInfrastrucure(objects, func(object unstructured.Unstructured) error {
		actedOn = append(actedOn, object.GetName())
		return nil
	})
	if !reflect.DeepEqual(actedOn, []string{"original"}) {
		t.Fatalf("acted on %v after manifest changed, want [original]", actedOn)
	}
}

func TestTestingInfrastructureNotReadyOnlyChecksManifestWorkloads(t *testing.T) {
	client := fake.NewSimpleClientset(
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: "disks-images-provider", Namespace: "test-infra", Generation: 1},
			Status: appsv1.DaemonSetStatus{
				ObservedGeneration: 1, DesiredNumberScheduled: 2, NumberReady: 2,
			},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: "unrelated-daemonset", Namespace: "test-infra", Generation: 1},
			Status:     appsv1.DaemonSetStatus{ObservedGeneration: 1, DesiredNumberScheduled: 1},
		},
		&k8sv1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "datadog-operator-manager", Namespace: "monitoring"}},
	)
	objects := []unstructured.Unstructured{
		manifestObject("apps/v1", "DaemonSet", "test-infra", "disks-images-provider"),
		manifestObject("v1", "Service", "cdi", "cdi-uploadproxy-nodeport"),
	}

	unready, err := testingInfrastructureNotReady(context.Background(), client, objects)
	if err != nil {
		t.Fatal(err)
	}
	if len(unready) != 0 {
		t.Fatalf("unrelated workloads blocked startup: %v", unready)
	}
	for _, action := range client.Actions() {
		if action.GetVerb() == "list" {
			t.Fatalf("readiness scanned %s instead of fetching a declared workload", action.GetResource().Resource)
		}
	}

	ds, err := client.AppsV1().DaemonSets("test-infra").Get(context.Background(), "disks-images-provider", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	ds.Status.NumberReady = 1
	if _, err := client.AppsV1().DaemonSets("test-infra").Update(context.Background(), ds, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}

	unready, err = testingInfrastructureNotReady(context.Background(), client, objects)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"DaemonSet test-infra/disks-images-provider"}
	if !reflect.DeepEqual(unready, want) {
		t.Fatalf("unready workloads = %v, want %v", unready, want)
	}
}

func TestTestingInfrastructureNotReadyWaitsForWorkloadsInOtherNamespaces(t *testing.T) {
	client := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test-helper", Namespace: "custom-infra", Generation: 1},
		Status:     appsv1.DeploymentStatus{ObservedGeneration: 1},
	})
	objects := []unstructured.Unstructured{
		manifestObject("apps/v1", "Deployment", "custom-infra", "test-helper"),
		manifestObject("v1", "Pod", "custom-infra", "not-created-yet"),
	}

	unready, err := testingInfrastructureNotReady(context.Background(), client, objects)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Deployment custom-infra/test-helper", "Pod custom-infra/not-created-yet"}
	if !reflect.DeepEqual(unready, want) {
		t.Fatalf("unready workloads = %v, want %v", unready, want)
	}
}

func TestTestingInfrastructureNotReadyWaitsForDaemonSetToSchedule(t *testing.T) {
	client := fake.NewSimpleClientset(&appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "disks-images-provider", Namespace: "test-infra", Generation: 1},
		Status: appsv1.DaemonSetStatus{
			ObservedGeneration: 1,
		},
	})
	objects := []unstructured.Unstructured{
		manifestObject("apps/v1", "DaemonSet", "test-infra", "disks-images-provider"),
	}

	unready, err := testingInfrastructureNotReady(context.Background(), client, objects)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"DaemonSet test-infra/disks-images-provider"}
	if !reflect.DeepEqual(unready, want) {
		t.Fatalf("zero-scheduled DaemonSet readiness = %v, want %v", unready, want)
	}

	daemonSet, err := client.AppsV1().DaemonSets("test-infra").Get(context.Background(), "disks-images-provider", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	daemonSet.Status.DesiredNumberScheduled = 1
	daemonSet.Status.NumberReady = 1
	if _, err := client.AppsV1().DaemonSets("test-infra").Update(context.Background(), daemonSet, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}

	unready, err = testingInfrastructureNotReady(context.Background(), client, objects)
	if err != nil {
		t.Fatal(err)
	}
	if len(unready) != 0 {
		t.Fatalf("scheduled and ready DaemonSet remained unready: %v", unready)
	}
}

func TestTestingInfrastructureReadErrorIsReported(t *testing.T) {
	client := fake.NewSimpleClientset()
	client.PrependReactor("get", "daemonsets", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("API unavailable")
	})
	objects := []unstructured.Unstructured{
		manifestObject("apps/v1", "DaemonSet", "test-infra", "disks-images-provider"),
	}

	_, err := testingInfrastructureNotReady(context.Background(), client, objects)
	if err == nil || !strings.Contains(err.Error(), "disks-images-provider") || !strings.Contains(err.Error(), "API unavailable") {
		t.Fatalf("read error did not identify the target workload: %v", err)
	}
}

func TestValidateTestingInfrastructureObjects(t *testing.T) {
	objects := []unstructured.Unstructured{
		manifestObject("apps/v1", "DaemonSet", "test-infra", "disks-images-provider"),
		manifestObject("v1", "Service", "cdi", "cdi-uploadproxy-nodeport"),
		manifestObject("v1", "ServiceAccount", "test-infra", "test-sa"),
		manifestObject("rbac.authorization.k8s.io/v1", "ClusterRoleBinding", "", "test-binding"),
	}
	if err := validateTestingInfrastructureObjects(objects); err != nil {
		t.Fatalf("default manifest kinds rejected: %v", err)
	}

	objects = append(objects, manifestObject("example.io/v1", "CustomController", "custom-infra", "helper"))
	if err := validateTestingInfrastructureObjects(objects); err == nil {
		t.Fatal("custom workload without readiness rule was silently accepted")
	}
}

func manifestObject(apiVersion, kind, namespace, name string) unstructured.Unstructured {
	return unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata": map[string]interface{}{
			"namespace": namespace,
			"name":      name,
		},
	}}
}
