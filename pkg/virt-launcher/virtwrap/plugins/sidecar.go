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

package plugins

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc"
	grpcstatus "google.golang.org/grpc/status"

	"k8s.io/apimachinery/pkg/util/wait"

	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
)

const (
	pluginSocketBaseDir            = "/var/run/kubevirt-plugin"
	sidecarDialTimeoutSeconds      = 5
	domainTypeLibvirt              = "libvirt"
	defaultSidecarCallTimeout      = 30 * time.Second
	defaultSidecarReadinessTimeout = 30 * time.Second
)

type readinessBudget struct {
	deadline time.Time
	total    time.Duration
}

// sidecarCallOptions configures how dialSidecar reaches a hook's socket: where plugin sockets
// are rooted, the readiness budget shared across the hooks in one invocation, and the
// socket-readiness probe (overridden in tests).
type sidecarCallOptions struct {
	socketBaseDir string
	readiness     readinessBudget
	socketReady   func(path string) (bool, error)
}

// defaultSidecarCallOptions returns the production sidecarCallOptions: the real plugin socket
// directory, os.Stat-based readiness, and a readiness budget starting now. All sidecars are
// containers of the same virt-launcher pod and start at roughly the same time, so the budget is
// shared across every hook in one invocation rather than recomputed per hook; otherwise N
// sidecars could delay the run by N readiness timeouts.
func defaultSidecarCallOptions() sidecarCallOptions {
	return sidecarCallOptions{
		socketBaseDir: pluginSocketBaseDir,
		readiness:     readinessBudget{deadline: time.Now().Add(defaultSidecarReadinessTimeout), total: defaultSidecarReadinessTimeout},
		socketReady:   osStatSocketReady,
	}
}

// dialSidecar waits for hook's socket to become ready and connects to it. The caller closes
// the returned connection.
func dialSidecar(parentCtx context.Context, hook resolvedHook, opts sidecarCallOptions) (*grpc.ClientConn, error) {
	socketPath := hook.hook.Sidecar.SocketPath
	if err := waitForSidecarSocket(parentCtx, socketPath, opts.readiness, opts.socketReady); err != nil {
		return nil, err
	}
	if err := validateSocketPath(socketPath, hook.pluginName, opts.socketBaseDir); err != nil {
		return nil, fmt.Errorf("invalid socket path: %w", err)
	}

	conn, err := grpcutil.DialSocketWithTimeout(socketPath, sidecarDialTimeoutSeconds)
	if err != nil {
		return nil, fmt.Errorf("dialing sidecar socket %s: %w", socketPath, err)
	}
	return conn, nil
}

// sidecarRPCError reports a failed launcher hook RPC by its gRPC status when it has one.
func sidecarRPCError(err error) error {
	if st, ok := grpcstatus.FromError(err); ok {
		return fmt.Errorf("launcher hook RPC failed with %s: %s", st.Code(), st.Message())
	}
	return err
}

func validateSocketPath(socketPath, pluginName, socketBaseDir string) error {
	pluginDir := filepath.Join(socketBaseDir, pluginName) + string(os.PathSeparator)
	info, err := os.Lstat(socketPath)
	if err != nil {
		return fmt.Errorf("stat socket path %q: %w", socketPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("socket path %q is a symlink, which is not allowed", socketPath)
	}
	cleaned := filepath.Clean(socketPath)
	if !strings.HasPrefix(cleaned, pluginDir) {
		return fmt.Errorf("socket path %q is outside %s", socketPath, pluginDir)
	}
	return nil
}

func osStatSocketReady(socketPath string) (bool, error) {
	_, err := os.Stat(socketPath)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func waitForSidecarSocket(parentCtx context.Context, socketPath string, budget readinessBudget, socketReady func(path string) (bool, error)) error {
	if err := parentCtx.Err(); err != nil {
		return fmt.Errorf("checking socket %s: %w", socketPath, err)
	}
	ready, err := socketReady(socketPath)
	if err != nil {
		return fmt.Errorf("checking socket %s: %w", socketPath, err)
	}
	if ready {
		return nil
	}

	remaining := time.Until(budget.deadline)
	if remaining <= 0 {
		return fmt.Errorf("sidecar socket %s not ready after %v", socketPath, budget.total)
	}
	if err := wait.PollUntilContextTimeout(parentCtx, 500*time.Millisecond, remaining, true, func(_ context.Context) (bool, error) {
		ready, err := socketReady(socketPath)
		if err != nil {
			return false, fmt.Errorf("checking socket %s: %w", socketPath, err)
		}
		return ready, nil
	}); err != nil {
		return fmt.Errorf("sidecar socket %s not ready after %v: %w", socketPath, budget.total, err)
	}
	return nil
}
