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
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
	klog "kubevirt.io/client-go/log"
)

// daemonLabels maps each control-plane daemon to the label selector used to
// find its pod(s) in the KubeVirt install namespace.
var daemonLabels = map[string]string{
	"virt-api":        "kubevirt.io=virt-api",
	"virt-controller": "kubevirt.io=virt-controller",
	"virt-handler":    "kubevirt.io=virt-handler",
	"virt-operator":   "kubevirt.io=virt-operator",
}

const coveragePort = 6061

// FlushDaemonCoverage calls POST /coverage/flush on every control-plane daemon
// pod via the Kubernetes pod proxy. Best-effort: errors are logged but do not
// fail the suite (coverage is a reference signal, not a hard requirement).
func FlushDaemonCoverage(ctx context.Context, client kubecli.KubevirtClient) {
	ns := GetTestNamespace(nil)

	for component, selector := range daemonLabels {
		pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			klog.Log.Reason(err).Warningf("coverage: could not list %s pods", component)
			continue
		}
		for _, pod := range pods.Items {
			if err := flushPod(ctx, client, ns, pod.Name); err != nil {
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

// CollectCoverageArtifacts downloads the merged covdata from the collector,
// writes it to $ARTIFACTS/coverage/, and generates text + HTML reports using
// "go tool covdata".
func CollectCoverageArtifacts(ctx context.Context, collectorURL, artifactsDir string) {
	covDir := filepath.Join(artifactsDir, "coverage")
	rawDir := filepath.Join(covDir, "merged.covdata")
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		klog.Log.Reason(err).Warning("coverage: could not create artifacts dir")
		return
	}

	// Download the merged covdata tar from the collector.
	rawURL := collectorURL + "/raw"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		klog.Log.Reason(err).Warning("coverage: could not build raw request")
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		klog.Log.Reason(err).Warningf("coverage: GET %s failed", rawURL)
		return
	}
	defer resp.Body.Close()

	tarPath := filepath.Join(covDir, "merged.tar")
	f, err := os.Create(tarPath)
	if err != nil {
		klog.Log.Reason(err).Warning("coverage: could not create tar file")
		return
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		klog.Log.Reason(err).Warning("coverage: could not write tar file")
		return
	}
	f.Close()

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
