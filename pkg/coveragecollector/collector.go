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

// Package coveragecollector implements the kv-coverage-collector server used
// during E2E coverage runs. Instrumented KubeVirt components (via
// pkg/coveragehttp or the virt-launcher uploader sidecar) POST their raw
// covmeta/covcounters files here; the collector stores them per run-id as a
// covdata directory and can merge and summarise them with "go tool covdata".
package coveragecollector

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	// maxUploadBytes caps a single coverage file upload (defensive; covmeta
	// files for large binaries are a few hundred KB).
	maxUploadBytes = 256 << 20 // 256 MiB

	defaultRunID = "default"
)

// Collector stores uploaded coverage files and produces reports from them.
type Collector struct {
	// baseDir is where per-run covdata directories are written.
	baseDir string
	// goBin is the Go binary used for "go tool covdata"; overridable for tests.
	goBin string
}

// New returns a Collector that stores data under baseDir.
func New(baseDir string) *Collector {
	goBin := os.Getenv("GO_BINARY")
	if goBin == "" {
		goBin = "go"
	}
	return &Collector{baseDir: baseDir, goBin: goBin}
}

// Handler returns the collector's HTTP routes.
func (c *Collector) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/coverage", c.handleUpload)        // POST: store one file
	mux.HandleFunc("/coverage/report", c.handleReport) // GET: covdata percent
	mux.HandleFunc("/coverage/raw", c.handleRaw)       // GET: merged covdata tar
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return mux
}

// runDir returns (and creates) the covdata directory for a run-id.
func (c *Collector) runDir(runID string) (string, error) {
	dir := filepath.Join(c.baseDir, sanitizeRunID(runID))
	return dir, os.MkdirAll(dir, 0o755)
}

func (c *Collector) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	filename := filepath.Base(r.Header.Get("X-Coverage-Filename"))
	if !isCoverageFile(filename) {
		http.Error(w, "X-Coverage-Filename must be a covmeta.* or covcounters.* name", http.StatusBadRequest)
		return
	}

	dir, err := c.runDir(r.Header.Get("X-Coverage-Run-Id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := io.ReadAll(io.LimitReader(r.Body, maxUploadBytes))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := writeUnique(dir, filename, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (c *Collector) handleReport(w http.ResponseWriter, r *http.Request) {
	dir, err := c.runDir(r.URL.Query().Get("run"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out, err := c.covdata("percent", "-i="+dir)
	if err != nil {
		http.Error(w, fmt.Sprintf("covdata percent failed: %v\n%s", err, out), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write(out)
}

func (c *Collector) handleRaw(w http.ResponseWriter, r *http.Request) {
	dir, err := c.runDir(r.URL.Query().Get("run"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	merged, err := os.MkdirTemp("", "covmerged-")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(merged)

	if out, err := c.covdata("merge", "-i="+dir, "-o="+merged); err != nil {
		http.Error(w, fmt.Sprintf("covdata merge failed: %v\n%s", err, out), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-tar")
	if err := tarDir(merged, w); err != nil {
		// Response is likely partially written; log-style error only.
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// covdata runs "go tool covdata <args...>" and returns its combined output.
func (c *Collector) covdata(args ...string) ([]byte, error) {
	full := append([]string{"tool", "covdata"}, args...)
	return exec.Command(c.goBin, full...).CombinedOutput()
}

// --- helpers ---

func isCoverageFile(name string) bool {
	return name != "" && name != "." && name != ".." &&
		(strings.HasPrefix(name, "covmeta.") || strings.HasPrefix(name, "covcounters."))
}

// sanitizeRunID keeps only safe characters so a run-id cannot escape baseDir.
func sanitizeRunID(runID string) string {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return defaultRunID
	}
	clean := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, runID)
	if clean == "" {
		return defaultRunID
	}
	return clean
}

// writeUnique writes data to dir/name. A covmeta file is keyed by its content
// hash (covmeta.<hash>), so if one already exists it is identical and we skip
// the write. covcounters files from different pods can legitimately share a
// name, so those get a numeric suffix to keep every counter set.
func writeUnique(dir, name string, data []byte) error {
	dest := filepath.Join(dir, name)

	if strings.HasPrefix(name, "covmeta.") {
		if _, err := os.Stat(dest); err == nil {
			return nil // identical meta already stored
		}
		return os.WriteFile(dest, data, 0o644)
	}

	for i := 1; ; i++ {
		if _, err := os.Stat(dest); errors.Is(err, os.ErrNotExist) {
			break
		}
		dest = filepath.Join(dir, fmt.Sprintf("%s.%d", name, i))
	}
	return os.WriteFile(dest, data, 0o644)
}

// tarDir streams the regular files in dir as a tar archive to w.
func tarDir(dir string, w io.Writer) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = e.Name()
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		f, err := os.Open(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, f); err != nil {
			f.Close()
			return err
		}
		f.Close()
	}
	return nil
}
