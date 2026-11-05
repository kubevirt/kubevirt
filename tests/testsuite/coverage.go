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
	"os"
	"os/exec"
	"path/filepath"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"kubevirt.io/client-go/kubecli"
	klog "kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/coveragemap"
	"kubevirt.io/kubevirt/tests/flags"
)

// daemonLabels maps each control-plane daemon to the label selector used to
// find its pod(s) in the KubeVirt install namespace.
var daemonLabels = map[string]string{
	"virt-api":        "kubevirt.io=virt-api",
	"virt-controller": "kubevirt.io=virt-controller",
	"virt-handler":    "kubevirt.io=virt-handler",
	"virt-operator":   "kubevirt.io=virt-operator",
}

const (
	coveragePort = 6061
	// coverageRunID is the shared run identifier used by daemon flushes and
	// virt-launcher uploads. Must match what SetupCoverage passes to
	// coveragemap.Install so the collector merges all data under one run.
	coverageRunID = "functest"
)

// FlushDaemonCoverage calls POST /coverage/flush on every control-plane daemon
// pod via the Kubernetes pod proxy. Best-effort: errors are logged but do not
// fail the suite (coverage is a reference signal, not a hard requirement).
func FlushDaemonCoverage(ctx context.Context, client kubecli.KubevirtClient) {
	ns := flags.KubeVirtInstallNamespace

	for component, selector := range daemonLabels {
		pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			klog.Log.Reason(err).Warningf("coverage: could not list %s pods", component)
			continue
		}
		for _, pod := range pods.Items {
			if err := flushPod(ctx, client, flags.KubeVirtInstallNamespace, pod.Name); err != nil {
				klog.Log.Reason(err).Warningf("coverage: flush failed for %s/%s", component, pod.Name)
			} else {
				klog.Log.Infof("coverage: flushed %s/%s", component, pod.Name)
			}
		}
	}
}

// flushPod calls POST /coverage/flush on a single pod via the pod proxy.
func flushPod(ctx context.Context, client kubecli.KubevirtClient, ns, podName string) error {
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s:%d/proxy/coverage/flush",
		ns, podName, coveragePort)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp := client.CoreV1().RESTClient().Post().
		AbsPath(path).
		Do(ctx)

	return resp.Error()
}

// CollectCoverageArtifacts downloads the merged covdata from the collector via
// the Kubernetes API server service proxy (so the test binary can reach it
// without in-cluster DNS), writes it to $ARTIFACTS/coverage/, and generates
// text + HTML reports using "go tool covdata".
func CollectCoverageArtifacts(ctx context.Context, client kubecli.KubevirtClient, artifactsDir string) {
	covDir := filepath.Join(artifactsDir, "coverage")
	rawDir := filepath.Join(covDir, "merged.covdata")
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		klog.Log.Reason(err).Warning("coverage: could not create artifacts dir")
		return
	}

	// Use the API server proxy so the test binary (running outside the cluster)
	// can reach the in-cluster collector service.
	proxyPath := fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%d/proxy/coverage/raw?run=%s",
		coverageNS, collectorServiceName, collectorPort, coverageRunID)
	rawBytes, err := client.CoreV1().RESTClient().Get().AbsPath(proxyPath).DoRaw(ctx)
	if err != nil {
		klog.Log.Reason(err).Warningf("coverage: GET %s failed", proxyPath)
		return
	}

	tarPath := filepath.Join(covDir, "merged.tar")
	if err := os.WriteFile(tarPath, rawBytes, 0o644); err != nil {
		klog.Log.Reason(err).Warning("coverage: could not write tar file")
		return
	}

	// Untar into rawDir.
	if out, err := exec.CommandContext(ctx, "tar", "-xf", tarPath, "-C", rawDir).CombinedOutput(); err != nil {
		klog.Log.Reason(err).Warningf("coverage: untar failed: %s", out)
		return
	}

	goBin := "go"
	if gb := os.Getenv("GO_BINARY"); gb != "" {
		goBin = gb
	}

	// Text format → coverage.txt
	txtPath := filepath.Join(covDir, "coverage.txt")
	if out, err := exec.CommandContext(ctx, goBin, "tool", "covdata", "textfmt",
		"-i="+rawDir, "-o="+txtPath).CombinedOutput(); err != nil {
		klog.Log.Reason(err).Warningf("coverage: textfmt failed: %s", out)
		return
	}

	// HTML report → coverage.html
	htmlPath := filepath.Join(covDir, "coverage.html")
	if out, err := exec.CommandContext(ctx, goBin, "tool", "cover",
		"-html="+txtPath, "-o="+htmlPath).CombinedOutput(); err != nil {
		klog.Log.Reason(err).Warningf("coverage: html report failed: %s", out)
		return
	}

	// Print percent summary to stdout so it appears in CI logs.
	if out, err := exec.CommandContext(ctx, goBin, "tool", "covdata", "percent",
		"-i="+rawDir).CombinedOutput(); err == nil {
		fmt.Printf("\n=== E2E Coverage Summary ===\n%s\n", out)
	}

	klog.Log.Infof("coverage: artifacts written to %s", covDir)
}

const (
	coverageNS           = "kubevirt-coverage"
	collectorDeployName  = "kv-coverage-collector"
	collectorServiceName = "kv-coverage-collector-svc"
	collectorPort        = 80
	collectorTargetPort  = 8080
)

// SetupCoverage deploys the coverage collector and MAP on the cluster (node 1
// only, idempotent) then resets all daemon counters. Called from
// SynchronizedBeforeTestSetup when --cov-report is set.
func SetupCoverage() {
	ctx := context.Background()
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		klog.Log.Reason(err).Warning("coverage: cannot get kubevirt client, skipping setup")
		return
	}

	// 1. Namespace.
	_, err = virtClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: coverageNS},
	}, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		klog.Log.Reason(err).Warningf("coverage: create namespace %s", coverageNS)
	}

	// 2. Collector Deployment (idempotent).
	replicas := int32(1)
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: collectorDeployName, Namespace: coverageNS},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": collectorDeployName}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": collectorDeployName}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "collector",
						Image:           flags.CovCollectorImage,
						ImagePullPolicy: corev1.PullAlways,
						Env: []corev1.EnvVar{
							{Name: "LISTEN_ADDR", Value: fmt.Sprintf(":%d", collectorTargetPort)},
							{Name: "COVERAGE_DIR", Value: "/coverage-data"},
						},
						Ports:        []corev1.ContainerPort{{ContainerPort: int32(collectorTargetPort)}},
						VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/coverage-data"}},
					}},
					Volumes: []corev1.Volume{{
						Name: "data",
						VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
					}},
				},
			},
		},
	}
	_, err = virtClient.AppsV1().Deployments(coverageNS).Create(ctx, deploy, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		klog.Log.Reason(err).Warning("coverage: create collector deployment")
	}

	// 3. Collector Service.
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: collectorServiceName, Namespace: coverageNS},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": collectorDeployName},
			Ports: []corev1.ServicePort{{
				Port:       int32(collectorPort),
				TargetPort: intstr.FromInt(collectorTargetPort),
			}},
		},
	}
	_, err = virtClient.CoreV1().Services(coverageNS).Create(ctx, svc, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		klog.Log.Reason(err).Warning("coverage: create collector service")
	}

	// 4. Wait for the collector to be ready (up to 2 min).
	// Use the Kubernetes API server service proxy so the test binary can reach
	// the collector without in-cluster DNS (the test binary runs outside the cluster).
	collectorProxyURL := fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%d/proxy/coverage",
		coverageNS, collectorServiceName, collectorPort)
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		result := virtClient.CoreV1().RESTClient().Get().AbsPath(collectorProxyURL + "/healthz").Do(ctx)
		if result.Error() == nil {
			klog.Log.Info("coverage: collector is ready")
			break
		}
		time.Sleep(5 * time.Second)
	}

	// 5. Inject COVERAGE_COLLECTOR_URL + COVERAGE_RUN_ID into daemon deployments
	// so their coveragehttp.flush() knows where to send data.
	collectorEnvURL := fmt.Sprintf("http://%s.%s/coverage", collectorServiceName, coverageNS)
	daemonDeployments := []string{"virt-api", "virt-controller"}
	for _, name := range daemonDeployments {
		patch := fmt.Sprintf(`{"spec":{"template":{"spec":{"containers":[{"name":%q,"env":[
			{"name":"COVERAGE_COLLECTOR_URL","value":%q},
			{"name":"COVERAGE_RUN_ID","value":%q},
			{"name":"POD_NAME","valueFrom":{"fieldRef":{"fieldPath":"metadata.name"}}}
		]}]}}}}`, name, collectorEnvURL, coverageRunID)
		if _, err := virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).
			Patch(ctx, name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{}); err != nil {
			klog.Log.Reason(err).Warningf("coverage: could not patch %s with collector env", name)
		}
	}
	// virt-handler is a DaemonSet
	handlerPatch := fmt.Sprintf(`{"spec":{"template":{"spec":{"containers":[{"name":"virt-handler","env":[
		{"name":"COVERAGE_COLLECTOR_URL","value":%q},
		{"name":"COVERAGE_RUN_ID","value":%q},
		{"name":"POD_NAME","valueFrom":{"fieldRef":{"fieldPath":"metadata.name"}}}
	]}]}}}}`, collectorEnvURL, coverageRunID)
	if _, err := virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).
		Patch(ctx, "virt-handler", types.StrategicMergePatchType, []byte(handlerPatch), metav1.PatchOptions{}); err != nil {
		klog.Log.Reason(err).Warning("coverage: could not patch virt-handler with collector env")
	}

	// 6. Probe virt-api :6061 — fail fast if images aren't instrumented.
	pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(
		ctx, metav1.ListOptions{LabelSelector: "kubevirt.io=virt-api"})
	if err == nil && len(pods.Items) > 0 {
		if err := flushPod(ctx, virtClient, flags.KubeVirtInstallNamespace, pods.Items[0].Name); err != nil {
			klog.Log.Reason(err).Warning("coverage: virt-api :6061 not reachable — are images built with --build-cover?")
		}
	}

	// 6. Install MAP + binding for virt-launcher injection.
	if flags.CovUploaderImage != "" {
		if err := coveragemap.Install(ctx, virtClient,
			flags.CovUploaderImage,
			fmt.Sprintf("http://%s.%s/coverage", collectorServiceName, coverageNS),
			coverageRunID); err != nil {
			klog.Log.Reason(err).Warning("coverage: MAP install failed")
		} else {
			klog.Log.Info("coverage: MAP installed for virt-launcher injection")
		}
	}

	// 7. Reset all daemon counters for a clean baseline.
	FlushDaemonCoverage(ctx, virtClient)
	klog.Log.Info("coverage: setup complete")
}
