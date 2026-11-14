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

//go:build coverage_e2e

// Package coveragehttp exposes an in-process HTTP endpoint that lets a
// long-running, coverage-instrumented KubeVirt daemon dump its Go coverage
// counters on demand (via runtime/coverage) instead of only at process exit.
//
// It is compiled only into binaries built with the "coverage_e2e" build tag
// (i.e. images built with hack/bazel-build-images.sh --build-cover). In all
// other builds the no-op Start in coveragehttp_stub.go is used, so production
// binaries carry no coverage HTTP server.
//
// Endpoints (served on :6061), reached by tests through the Kubernetes pod
// proxy:
//
//	POST /coverage/reset  -> runtime/coverage.ClearCounters() (requires atomic
//	                         covermode, which rules_go enables for -cover builds)
//	POST /coverage/flush  -> upload meta (first call only) + counters to the
//	                         collector
//
// A SIGTERM handler performs the same flush before the process shuts down, so
// coverage survives pods that are terminated rather than exiting normally.
package coveragehttp

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/coverage"
	"sync"
	"syscall"
	"time"

	klog "kubevirt.io/client-go/log"
)

const (
	// listenAddr is where every instrumented daemon exposes the coverage API.
	// Tests reach it through the Kubernetes pod proxy, so it binds all
	// interfaces on a fixed port.
	listenAddr = "0.0.0.0:6061"

	// Environment variables used to label uploads and locate the collector.
	// These are injected into the pod (daemon manifests / MAP) at test time.
	envPod       = "POD_NAME"                // metadata.name of this pod
	envRunID     = "COVERAGE_RUN_ID"         // shared id for one functest suite
	envCollector = "COVERAGE_COLLECTOR_URL"  // e.g. http://kv-coverage-collector-svc.kubevirt-coverage/coverage

	uploadTimeout = 30 * time.Second
)

// server holds the per-process coverage upload state.
type server struct {
	component string
	pod       string
	runID     string
	collector string

	mu          sync.Mutex
	metaWritten bool // meta-data is invariant per binary, upload it only once
}

// Start launches the coverage HTTP server and installs a SIGTERM flush handler.
// component is the KubeVirt component name (e.g. "virt-api") used to label the
// uploaded profiles. It returns immediately; the server runs in a goroutine.
func Start(component string) {
	s := &server{
		component: component,
		pod:       os.Getenv(envPod),
		runID:     os.Getenv(envRunID),
		collector: os.Getenv(envCollector),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/coverage/reset", s.handleReset)
	mux.HandleFunc("/coverage/flush", s.handleFlush)

	httpServer := &http.Server{Addr: listenAddr, Handler: mux}

	go func() {
		klog.Log.Infof("coverage: serving coverage API on %s for component %q", listenAddr, component)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			klog.Log.Reason(err).Warning("coverage: HTTP server stopped")
		}
	}()

	s.installSignalHandler()
}

// handleReset clears all coverage counters. This is what enables a per-spec
// coverage picture in serial/sequential runs. It requires the binary to have
// been built in atomic covermode (rules_go does this for -cover builds);
// ClearCounters returns an error otherwise.
func (s *server) handleReset(w http.ResponseWriter, _ *http.Request) {
	if err := coverage.ClearCounters(); err != nil {
		http.Error(w, fmt.Sprintf("ClearCounters failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleFlush snapshots current coverage and uploads it to the collector.
func (s *server) handleFlush(w http.ResponseWriter, _ *http.Request) {
	if err := s.flush(); err != nil {
		http.Error(w, fmt.Sprintf("flush failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// flush writes the current coverage to a temporary directory using the
// runtime/coverage *Dir helpers -- which produce correctly named covmeta.<hash>
// and covcounters.<hash>.<pid>.<time> files -- then uploads each file to the
// collector under its original name. Preserving those names lets the collector
// feed them straight to "go tool covdata", which pairs counters to their meta
// by the hash in the filename. Meta-data is invariant per binary, so it is only
// emitted on the first flush.
func (s *server) flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tmp, err := os.MkdirTemp("", "coverage-flush-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	if !s.metaWritten {
		if err := coverage.WriteMetaDir(tmp); err != nil {
			return fmt.Errorf("WriteMetaDir: %w", err)
		}
	}
	if err := coverage.WriteCountersDir(tmp); err != nil {
		return fmt.Errorf("WriteCountersDir: %w", err)
	}

	entries, err := os.ReadDir(tmp)
	if err != nil {
		return err
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(tmp, e.Name()))
		if err != nil {
			return err
		}
		if err := s.upload(e.Name(), data); err != nil {
			return fmt.Errorf("upload %s: %w", e.Name(), err)
		}
	}
	s.metaWritten = true
	return nil
}

// upload POSTs a single coverage data file to the collector, labelling it via
// headers (including its original covmeta/covcounters filename) so the collector
// can lay it out as a covdata directory.
func (s *server) upload(filename string, data []byte) error {
	if s.collector == "" {
		return fmt.Errorf("%s not set", envCollector)
	}

	ctx, cancel := context.WithTimeout(context.Background(), uploadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.collector, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Coverage-Component", s.component)
	req.Header.Set("X-Coverage-Pod", s.pod)
	req.Header.Set("X-Coverage-Run-Id", s.runID)
	req.Header.Set("X-Coverage-Filename", filename) // covmeta.* or covcounters.*

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

// installSignalHandler flushes coverage on SIGTERM before the process exits.
// Go delivers SIGTERM to every registered handler, so this coexists with a
// daemon's own graceful-shutdown handler rather than replacing it.
func (s *server) installSignalHandler() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM)
	go func() {
		<-ch
		klog.Log.Info("coverage: SIGTERM received, flushing coverage before shutdown")
		if err := s.flush(); err != nil {
			klog.Log.Reason(err).Warning("coverage: SIGTERM flush failed")
		}
	}()
}
