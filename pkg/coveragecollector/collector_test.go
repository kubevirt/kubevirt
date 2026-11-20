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

package coveragecollector

import (
	"archive/tar"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime/coverage"
	"strings"
	"testing"
)

func TestSanitizeRunID(t *testing.T) {
	cases := map[string]string{
		"":                 defaultRunID,
		"   ":              defaultRunID,
		"functest-123":     "functest-123",
		"../../etc/passwd": "------etc-passwd",
		"a/b":              "a-b",
	}
	for in, want := range cases {
		if got := sanitizeRunID(in); got != want {
			t.Errorf("sanitizeRunID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsCoverageFile(t *testing.T) {
	ok := []string{"covmeta.abc", "covcounters.abc.1.2"}
	bad := []string{"", ".", "..", "passwd", "cov.txt", "meta.covmeta"}
	for _, n := range ok {
		if !isCoverageFile(n) {
			t.Errorf("isCoverageFile(%q) = false, want true", n)
		}
	}
	for _, n := range bad {
		if isCoverageFile(n) {
			t.Errorf("isCoverageFile(%q) = true, want false", n)
		}
	}
}

func TestUploadRejectsBadFilename(t *testing.T) {
	c := New(t.TempDir())
	srv := httptest.NewServer(c.Handler())
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/coverage", strings.NewReader("data"))
	req.Header.Set("X-Coverage-Filename", "../escape")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("got status %d, want 400", resp.StatusCode)
	}
}

// TestEndToEnd uploads this test binary's own coverage data, then checks the
// report and raw endpoints. It only runs when the test binary is coverage
// instrumented (go test -cover), which is how it produces real covdata files.
func TestEndToEnd(t *testing.T) {
	src := t.TempDir()
	if err := coverage.WriteMetaDir(src); err != nil {
		t.Skipf("test not built with -cover, skipping end-to-end: %v", err)
	}
	if err := coverage.WriteCountersDir(src); err != nil {
		t.Fatalf("WriteCountersDir: %v", err)
	}

	c := New(t.TempDir())
	srv := httptest.NewServer(c.Handler())
	defer srv.Close()

	// Upload every covmeta/covcounters file the test binary produced.
	files, _ := os.ReadDir(src)
	uploaded := 0
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(src, f.Name()))
		if err != nil {
			t.Fatal(err)
		}
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/coverage", bytes.NewReader(data))
		req.Header.Set("X-Coverage-Filename", f.Name())
		req.Header.Set("X-Coverage-Run-Id", "functest-test")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("upload %s: status %d", f.Name(), resp.StatusCode)
		}
		uploaded++
	}
	if uploaded == 0 {
		t.Fatal("no coverage files were produced/uploaded")
	}

	// Report endpoint should return a covdata percent summary.
	report := httpGet(t, srv.URL+"/coverage/report?run=functest-test")
	if !strings.Contains(report, "coverage:") {
		t.Fatalf("report missing coverage summary, got:\n%s", report)
	}

	// Raw endpoint should return a tar containing a covmeta file.
	rawTar := httpGetBytes(t, srv.URL+"/coverage/raw?run=functest-test")
	if !tarHasPrefix(t, rawTar, "covmeta.") {
		t.Fatal("raw tar did not contain a covmeta file")
	}
}

func httpGet(t *testing.T, url string) string {
	t.Helper()
	return string(httpGetBytes(t, url))
}

func httpGetBytes(t *testing.T, url string) []byte {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status %d\n%s", url, resp.StatusCode, body)
	}
	return body
}

func tarHasPrefix(t *testing.T, data []byte, prefix string) bool {
	t.Helper()
	tr := tar.NewReader(bytes.NewReader(data))
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return false
		}
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(hdr.Name, prefix) {
			return true
		}
	}
}
