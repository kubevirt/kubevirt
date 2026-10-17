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
	"fmt"

	"kubevirt.io/client-go/log"

	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"
)

// runHooks runs hooks in order via run, owning the semantics shared by every launcher hook
// point: a canceled parent context aborts the whole run, a failed hook aborts the run unless
// its failure strategy is Ignore, and a run error caused by cancellation is reported as such
// regardless of failure strategy.
func runHooks(parentCtx context.Context, hookPoint pluginv1alpha1.LauncherHookPoint, hooks []resolvedHook, run func(ctx context.Context, hook resolvedHook) error) error {
	for _, hook := range hooks {
		if err := parentCtx.Err(); err != nil {
			return fmt.Errorf("launcher hook pipeline canceled: %w", err)
		}

		switch err := run(parentCtx, hook); {
		case err == nil:
		case parentCtx.Err() != nil:
			return fmt.Errorf("plugin %s hook %d at %s canceled: %w (hook error: %w)", hook.pluginName, hook.index, hookPoint, parentCtx.Err(), err)
		case hook.failureStrategy == pluginv1alpha1.FailureStrategyIgnore:
			log.Log.Warningf("Plugin %s hook %d at %s failed (ignored): %v", hook.pluginName, hook.index, hookPoint, err)
		default:
			return fmt.Errorf("plugin %s hook %d at %s failed: %w", hook.pluginName, hook.index, hookPoint, err)
		}
	}

	return nil
}
