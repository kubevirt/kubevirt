/*
 * This file is part of the kubevirt project
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

package exec

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/client-go/kubecli"
)

func TestExecuteCommandWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ExecuteCommandOnPodWithContext(ctx, &v1.Pod{}, "virt-handler", []string{"ip", "route", "show"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation before client initialization, got %v", err)
	}
}

func TestExecuteCommandCancelsSPDYFallback(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			http.Error(w, "websocket unavailable", http.StatusBadRequest)
			return
		}
		if !strings.EqualFold(r.Header.Get("Upgrade"), "SPDY/3.1") {
			t.Errorf("expected SPDY fallback, got %q", r.Header.Get("Upgrade"))
		}
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	defer server.CloseClientConnections()
	kubecli.Init()
	configPath := filepath.Join(t.TempDir(), "kubeconfig")
	config := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: %s
contexts:
- name: test
  context:
    cluster: test
current-context: test
`, server.URL)
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	previous := flag.Lookup("kubeconfig").Value.String()
	if err := flag.Set("kubeconfig", configPath); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := flag.Set("kubeconfig", previous); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := ExecuteCommandOnPodWithContext(ctx, &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "handler", Namespace: "kubevirt"}}, "virt-handler", []string{"ip", "route", "show"})
		result <- err
	}()
	select {
	case <-started:
		cancel()
	case <-ctx.Done():
		t.Fatal("exec never reached SPDY transport")
	}
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected transport cancellation, got %v", err)
		}
		if !strings.Contains(err.Error(), "stderr") || !strings.Contains(err.Error(), "stdout") {
			t.Fatalf("missing execution diagnostics: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("exec ignored cancellation")
	}
}
