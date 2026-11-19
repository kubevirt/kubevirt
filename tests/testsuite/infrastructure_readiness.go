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
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	k8sv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
)

// waitForTestingInfrastructureReady only waits for workloads from the manifests
// used by this test run. Their namespaces may differ from KubeVirt's namespace.
func waitForTestingInfrastructureReady(objects []unstructured.Unstructured, timeout time.Duration) {
	client := kubevirt.Client()
	Eventually(func(g Gomega) []string {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		unready, err := testingInfrastructureNotReady(ctx, client, objects)
		g.Expect(err).ToNot(HaveOccurred())
		return unready
	}, timeout, 2*time.Second).Should(BeEmpty(), "Testing infrastructure workloads are not ready.")
}

// validateTestingInfrastructureObjects keeps custom manifests from silently
// bypassing readiness checks when they introduce a new workload kind.
func validateTestingInfrastructureObjects(objects []unstructured.Unstructured) error {
	for _, object := range objects {
		resource := object.GetAPIVersion() + "/" + object.GetKind()
		if object.GetName() == "" {
			return fmt.Errorf("testing infrastructure manifest %s has no name", resource)
		}
		switch resource {
		case "apps/v1/DaemonSet", "apps/v1/Deployment", "apps/v1/StatefulSet", "apps/v1/ReplicaSet",
			"v1/ReplicationController", "v1/Pod", "batch/v1/Job":
			if object.GetNamespace() == "" {
				return fmt.Errorf("testing infrastructure workload %s %s has no namespace", resource, object.GetName())
			}
		case "v1/Service", "v1/ServiceAccount", "v1/ConfigMap", "v1/Secret",
			"rbac.authorization.k8s.io/v1/Role", "rbac.authorization.k8s.io/v1/RoleBinding",
			"rbac.authorization.k8s.io/v1/ClusterRole", "rbac.authorization.k8s.io/v1/ClusterRoleBinding":
			// Applying these objects is enough; they have no workload readiness state.
		default:
			return fmt.Errorf("no readiness rule for testing infrastructure manifest %s %s/%s", resource, object.GetNamespace(), object.GetName())
		}
	}
	return nil
}

func testingInfrastructureNotReady(ctx context.Context, client kubernetes.Interface, objects []unstructured.Unstructured) ([]string, error) {
	var unready []string
	for _, object := range objects {
		ready, checked, err := manifestWorkloadReady(ctx, client, object)
		if err != nil && !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("check %s %s/%s: %w", object.GetKind(), object.GetNamespace(), object.GetName(), err)
		}
		if checked && (!ready || err != nil) {
			unready = append(unready, fmt.Sprintf("%s %s/%s", object.GetKind(), object.GetNamespace(), object.GetName()))
		}
	}
	return unready, nil
}

func manifestWorkloadReady(ctx context.Context, client kubernetes.Interface, object unstructured.Unstructured) (ready bool, checked bool, err error) {
	namespace, name := object.GetNamespace(), object.GetName()
	switch object.GetAPIVersion() + "/" + object.GetKind() {
	case "apps/v1/DaemonSet":
		daemonSet, err := client.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, true, err
		}
		return daemonSet.Status.ObservedGeneration >= daemonSet.Generation &&
			daemonSet.Status.DesiredNumberScheduled > 0 &&
			daemonSet.Status.NumberReady == daemonSet.Status.DesiredNumberScheduled, true, nil
	case "apps/v1/Deployment":
		deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, true, err
		}
		return deployment.Status.ObservedGeneration >= deployment.Generation &&
			deployment.Status.AvailableReplicas >= desiredReplicas(deployment.Spec.Replicas), true, nil
	case "apps/v1/StatefulSet":
		statefulSet, err := client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, true, err
		}
		return statefulSet.Status.ObservedGeneration >= statefulSet.Generation &&
			statefulSet.Status.ReadyReplicas >= desiredReplicas(statefulSet.Spec.Replicas), true, nil
	case "apps/v1/ReplicaSet":
		replicaSet, err := client.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, true, err
		}
		return replicaSet.Status.ObservedGeneration >= replicaSet.Generation &&
			replicaSet.Status.ReadyReplicas >= desiredReplicas(replicaSet.Spec.Replicas), true, nil
	case "v1/ReplicationController":
		controller, err := client.CoreV1().ReplicationControllers(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, true, err
		}
		return controller.Status.ObservedGeneration >= controller.Generation &&
			controller.Status.ReadyReplicas >= desiredReplicas(controller.Spec.Replicas), true, nil
	case "v1/Pod":
		pod, err := client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, true, err
		}
		if pod.Status.Phase == k8sv1.PodSucceeded {
			return true, true, nil
		}
		for _, condition := range pod.Status.Conditions {
			if condition.Type == k8sv1.PodReady && condition.Status == k8sv1.ConditionTrue {
				return true, true, nil
			}
		}
		return false, true, nil
	case "batch/v1/Job":
		job, err := client.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, true, err
		}
		for _, condition := range job.Status.Conditions {
			if condition.Type == batchv1.JobComplete && condition.Status == k8sv1.ConditionTrue {
				return true, true, nil
			}
		}
		return false, true, nil
	default:
		// validateTestingInfrastructureObjects permits only known non-workloads here.
		return true, false, nil
	}
}

func desiredReplicas(replicas *int32) int32 {
	if replicas == nil {
		return 1
	}
	return *replicas
}
