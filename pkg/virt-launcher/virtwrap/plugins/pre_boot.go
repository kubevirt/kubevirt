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
	"encoding/json"
	"fmt"

	v1 "kubevirt.io/api/core/v1"
	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"

	pluginsv1alpha1 "kubevirt.io/kubevirt/pkg/hooks/plugins/v1alpha1"
)

// ApplyPreBootHooks signals that generated artifacts are ready for modification.
// Calls are sequential and may repeat after a failed start. Ignore cannot undo
// filesystem changes made by a failed hook.
func ApplyPreBootHooks(plugins []pluginv1alpha1.Plugin, vmi *v1.VirtualMachineInstance, preBootArtifactPaths func(vmi *v1.VirtualMachineInstance) map[string]string) error {
	if len(plugins) == 0 {
		return nil
	}

	hooks, err := selectHooks(plugins, vmi, pluginv1alpha1.LauncherHookPreBoot)
	if err != nil {
		return err
	}
	if len(hooks) == 0 {
		return nil
	}

	vmiJSON, err := json.Marshal(vmi)
	if err != nil {
		return fmt.Errorf("marshal VMI: %w", err)
	}

	opts := defaultSidecarCallOptions()
	artifactPaths := preBootArtifactPaths(vmi)

	return runHooks(context.Background(), pluginv1alpha1.LauncherHookPreBoot, hooks, func(ctx context.Context, hook resolvedHook) error {
		// PreBoot is sidecar-only, so selectHooks never returns a CEL hook here.
		return callSidecar(ctx, hook, opts, func(ctx context.Context, client pluginsv1alpha1.LauncherHookServiceClient) error {
			_, err := client.PreBoot(ctx, &pluginsv1alpha1.PreBootRequest{
				Vmi:            vmiJSON,
				PreBootContext: &pluginsv1alpha1.PreBootContext{ArtifactPaths: artifactPaths},
			})
			return err
		})
	})
}
