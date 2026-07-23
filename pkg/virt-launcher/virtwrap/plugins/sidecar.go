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

	grpcstatus "google.golang.org/grpc/status"

	virtwait "kubevirt.io/kubevirt/pkg/apimachinery/wait"
	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"

	pluginsv1alpha1 "kubevirt.io/kubevirt/pkg/hooks/plugins/v1alpha1"
)

const (
	pluginSocketBaseDir       = "/var/run/kubevirt-plugin"
	sidecarReadinessTimeout   = 30 * time.Second
	sidecarDialTimeoutSeconds = 5
	domainTypeLibvirt         = "libvirt"
	defaultSidecarCallTimeout = 30 * time.Second
)

func callSidecarHook(socketPath, pluginName string, domainXML, vmiJSON []byte, invocationContext string, timeout time.Duration) ([]byte, error) {

	if err := validateSocketPath(socketPath, pluginName); err != nil {
		return nil, fmt.Errorf("invalid socket path: %w", err)
	}

	conn, err := grpcutil.DialSocketWithTimeout(socketPath, sidecarDialTimeoutSeconds)
	if err != nil {
		return nil, fmt.Errorf("dialing sidecar socket %s: %w", socketPath, err)
	}
	defer conn.Close()

	client := pluginsv1alpha1.NewDomainHookServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := client.MutateDomain(ctx, &pluginsv1alpha1.MutateDomainRequest{
		DomainType: domainTypeLibvirt,
		Domain:     domainXML,
		Vmi:        vmiJSON,
		SidecarContext: &pluginsv1alpha1.SidecarContext{
			InvocationContext: invocationContext,
		},
	})
	if err != nil {
		if st, ok := grpcstatus.FromError(err); ok {
			return nil, fmt.Errorf("MutateDomain RPC to %s failed with %s: %s", pluginName, st.Code(), st.Message())
		}
		return nil, fmt.Errorf("MutateDomain RPC to %s failed: %w", pluginName, err)
	}
	return resp.Domain, nil
}

func validateSocketPath(socketPath, pluginName string) error {
	pluginDir := pluginSocketBaseDir + "/" + pluginName + "/"
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

func waitForSidecarSocket(socketPath string, deadline time.Time) error {
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return fmt.Errorf("sidecar socket %s not ready after %v", socketPath, sidecarReadinessTimeout)
	}
	if err := virtwait.PollImmediately(500*time.Millisecond, remaining, func(_ context.Context) (bool, error) {
		if _, err := os.Stat(socketPath); err == nil {
			return true, nil
		} else if errors.Is(err, os.ErrNotExist) {
			return false, nil
		} else {
			return false, fmt.Errorf("checking socket %s: %w", socketPath, err)
		}
	}); err != nil {
		return fmt.Errorf("sidecar socket %s not ready after %v: %w", socketPath, sidecarReadinessTimeout, err)
	}
	return nil
}
