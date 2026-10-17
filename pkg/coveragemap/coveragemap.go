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

// Package coveragemap creates and deletes the MutatingAdmissionPolicy and
// binding that inject the coverage-uploader native sidecar into virt-launcher
// pods during E2E coverage runs (--cov-report). The MAP pattern follows
// tests/plugin_test.go exactly.
package coveragemap

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"kubevirt.io/client-go/kubecli"
)

const (
	PolicyName  = "kv-coverage-launcher"
	BindingName = PolicyName + "-binding"

	// CoverageDir is the path inside the compute container and the uploader
	// sidecar where virt-launcher dumps its coverage data.
	CoverageDir = "/var/run/kubevirt/coverage"

	// VolumeName is the emptyDir volume shared between compute and the sidecar.
	VolumeName = "coverage-data"
)

var (
	// MutatingAdmissionPolicy is v1alpha1 in k8s 1.32 (GA expected in 1.34+).
	// Verified against our kubeadm 1.32.13 cluster: kubectl api-resources
	// shows admissionregistration.k8s.io/v1alpha1 for mutatingadmissionpolicies.
	mapGVR = schema.GroupVersionResource{
		Group:    "admissionregistration.k8s.io",
		Version:  "v1alpha1",
		Resource: "mutatingadmissionpolicies",
	}
	bindingGVR = schema.GroupVersionResource{
		Group:    "admissionregistration.k8s.io",
		Version:  "v1alpha1",
		Resource: "mutatingadmissionpolicybindings",
	}
)

// Install creates the MutatingAdmissionPolicy and its binding that inject
// the coverage-uploader sidecar into virt-launcher pods.
//
//   - uploaderImage is the fully-qualified image for the coverage-uploader.
//   - collectorURL is the URL of the kv-coverage-collector service, e.g.
//     "http://kv-coverage-collector-svc.kubevirt-coverage/coverage".
//   - runID is the suite-wide identifier generated in SynchronizedBeforeSuite.
func Install(ctx context.Context, client kubecli.KubevirtClient, uploaderImage, collectorURL, runID string) error {
	policy := buildPolicy(uploaderImage, collectorURL, runID)
	if _, err := client.DynamicClient().Resource(mapGVR).Create(ctx, policy, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("create MutatingAdmissionPolicy %s: %w", PolicyName, err)
	}
	binding := buildBinding()
	if _, err := client.DynamicClient().Resource(bindingGVR).Create(ctx, binding, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("create MutatingAdmissionPolicyBinding %s: %w", BindingName, err)
	}
	return nil
}

// Uninstall deletes the MAP and its binding. Errors are ignored so cleanup
// does not mask test failures.
func Uninstall(ctx context.Context, client kubecli.KubevirtClient) {
	_ = client.DynamicClient().Resource(mapGVR).Delete(ctx, PolicyName, metav1.DeleteOptions{})
	_ = client.DynamicClient().Resource(bindingGVR).Delete(ctx, BindingName, metav1.DeleteOptions{})
}

// buildPolicy returns the MAP object that injects the coverage-uploader sidecar
// and the coverage-data emptyDir volume into every virt-launcher pod.
func buildPolicy(uploaderImage, collectorURL, runID string) *unstructured.Unstructured {
	// JSONPatch expression (CEL). It:
	//   1. Adds an emptyDir volume "coverage-data".
	//   2. Mounts it at CoverageDir on the "compute" container.
	//   3. Sets GOCOVERDIR and COVERAGE_RUN_ID on the "compute" container.
	//   4. Sets terminationGracePeriodSeconds >= 60 so the sidecar has time.
	//   5. Appends the coverage-uploader native sidecar.
	patch := fmt.Sprintf(`[
  JSONPatch{op: "add", path: "/spec/volumes/-", value: Object.spec.volumes{
    name: %q,
    emptyDir: Object.spec.volumes.emptyDir{}
  }},
  JSONPatch{op: "add",
    path: "/spec/initContainers/-",
    value: Object.spec.initContainers{
      name: "coverage-uploader",
      image: %q,
      restartPolicy: "Always",
      env: [
        Object.spec.initContainers.env{name: "GOCOVERDIR",             value: %q},
        Object.spec.initContainers.env{name: "COVERAGE_RUN_ID",        value: %q},
        Object.spec.initContainers.env{name: "COVERAGE_COLLECTOR_URL", value: %q},
        Object.spec.initContainers.env{name: "POD_NAME",
          valueFrom: Object.spec.initContainers.env.valueFrom{
            fieldRef: Object.spec.initContainers.env.valueFrom.fieldRef{fieldPath: "metadata.name"}
          }
        }
      ],
      volumeMounts: [Object.spec.initContainers.volumeMounts{
        name: %q, mountPath: %q
      }],
      securityContext: Object.spec.initContainers.securityContext{
        allowPrivilegeEscalation: false,
        runAsNonRoot: true,
        seccompProfile: Object.spec.initContainers.securityContext.seccompProfile{type: "RuntimeDefault"},
        capabilities: Object.spec.initContainers.securityContext.capabilities{drop: ["ALL"]}
      }
    }
  }
]`,
		VolumeName,
		uploaderImage,
		CoverageDir,
		runID,
		collectorURL,
		VolumeName, CoverageDir,
	)

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "admissionregistration.k8s.io/v1alpha1",
			"kind":       "MutatingAdmissionPolicy",
			"metadata":   map[string]interface{}{"name": PolicyName},
			"spec": map[string]interface{}{
				"failurePolicy":      "Fail",
				"reinvocationPolicy": "Never",
				"matchConstraints": map[string]interface{}{
					"resourceRules": []interface{}{
						map[string]interface{}{
							"operations":  []interface{}{"CREATE"},
							"apiGroups":   []interface{}{""},
							"apiVersions": []interface{}{"v1"},
							"resources":   []interface{}{"pods"},
						},
					},
				},
				"matchConditions": []interface{}{
					map[string]interface{}{
						"name":       "is-virt-launcher",
						"expression": `has(object.metadata.labels) && "kubevirt.io" in object.metadata.labels && object.metadata.labels["kubevirt.io"] == "virt-launcher"`,
					},
				},
				"mutations": []interface{}{
					map[string]interface{}{
						"patchType": "JSONPatch",
						"jsonPatch": map[string]interface{}{
							"expression": patch,
						},
					},
				},
			},
		},
	}
}

func buildBinding() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "admissionregistration.k8s.io/v1alpha1",
			"kind":       "MutatingAdmissionPolicyBinding",
			"metadata":   map[string]interface{}{"name": BindingName},
			"spec": map[string]interface{}{
				"policyName": PolicyName,
			},
		},
	}
}
