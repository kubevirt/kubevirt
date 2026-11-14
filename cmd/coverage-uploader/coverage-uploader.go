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

// Command coverage-uploader is the native sidecar injected into virt-launcher
// pods by the MutatingAdmissionPolicy when --cov-report is set. It watches
// GOCOVERDIR for covmeta/covcounters files written by the instrumented
// virt-launcher and virt-launcher-monitor binaries and POSTs each file to
// the kv-coverage-collector. It runs as a native sidecar (initContainer with
// restartPolicy: Always) so it is signalled only after the compute container
// exits, giving it time to upload any exit-time dumps before the pod is
// deleted.
package main

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	klog "kubevirt.io/client-go/log"
)

const (
	pollInterval  = 2 * time.Second
	uploadTimeout = 30 * time.Second
)

func main() {
	klog.InitializeLogging("coverage-uploader")

	coverDir := os.Getenv("GOCOVERDIR")
	if coverDir == "" {
		klog.Log.Critical("GOCOVERDIR not set")
		os.Exit(1)
	}
	collector := os.Getenv("COVERAGE_COLLECTOR_URL")
	if collector == "" {
		klog.Log.Critical("COVERAGE_COLLECTOR_URL not set")
		os.Exit(1)
	}
	runID := os.Getenv("COVERAGE_RUN_ID")
	pod := os.Getenv("POD_NAME")
	component := "virt-launcher"

	klog.Log.Infof("coverage-uploader: watching %s, uploading to %s (run-id=%s pod=%s)",
		coverDir, collector, runID, pod)

	uploaded := map[string]bool{}

	for {
		entries, err := os.ReadDir(coverDir)
		if err != nil {
			// Dir may not exist yet if virt-launcher hasn't started; keep polling.
			time.Sleep(pollInterval)
			continue
		}

		for _, e := range entries {
			if !e.Type().IsRegular() {
				continue
			}
			name := e.Name()
			if !isCoverageFile(name) || uploaded[name] {
				continue
			}

			info, ferr := e.Info()
			if ferr != nil {
				continue
			}
			// Wait until the file hasn't grown for one poll cycle (simple
			// write-completion heuristic; covdata files are written atomically
			// by the Go runtime so this is usually instant).
			if !isStable(filepath.Join(coverDir, name), info) {
				continue
			}

			if err := upload(collector, component, pod, runID, filepath.Join(coverDir, name), name); err != nil {
				klog.Log.Reason(err).Warningf("coverage-uploader: failed to upload %s, will retry", name)
				continue
			}
			uploaded[name] = true
			klog.Log.Infof("coverage-uploader: uploaded %s", name)
		}

		time.Sleep(pollInterval)
	}
}

func isCoverageFile(name string) bool {
	return strings.HasPrefix(name, "covmeta.") || strings.HasPrefix(name, "covcounters.")
}

// isStable checks that the file size hasn't changed since info was captured.
// Since info is captured in the same ReadDir pass and covdata files are written
// atomically, this is effectively always true — but guards against the unlikely
// case of a partially written file.
func isStable(path string, prev fs.FileInfo) bool {
	cur, err := os.Stat(path)
	if err != nil {
		return false
	}
	return cur.Size() == prev.Size() && cur.Size() > 0
}

func upload(collector, component, pod, runID, path, filename string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), uploadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, collector, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Coverage-Component", component)
	req.Header.Set("X-Coverage-Pod", pod)
	req.Header.Set("X-Coverage-Run-Id", runID)
	req.Header.Set("X-Coverage-Filename", filename)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("collector returned %s", resp.Status)
	}
	return nil
}
