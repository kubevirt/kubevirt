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

// handleHookError applies the failure semantics shared by every launcher hook point to the
// outcome of one hook. It returns nil to continue with the next hook (the hook succeeded, or
// failed with failure strategy Ignore) and the error that aborts the run otherwise. A failure
// while parentCtx is canceled aborts the run regardless of failure strategy.
//
// A canceled pipeline needs no separate pre-hook check: dialSidecar fails fast when its
// context is already canceled, so the next hook after a cancellation reaches this function
// with an error and is reported as canceled. CEL hooks only run at GuestDefinition, which
// always uses context.Background().
func handleHookError(parentCtx context.Context, hookPoint pluginv1alpha1.LauncherHookPoint, hook resolvedHook, err error) error {
	switch {
	case err == nil:
		return nil
	case parentCtx.Err() != nil:
		return fmt.Errorf("plugin %s hook %d at %s canceled: %w (hook error: %w)", hook.pluginName, hook.index, hookPoint, parentCtx.Err(), err)
	case hook.failureStrategy == pluginv1alpha1.FailureStrategyIgnore:
		log.Log.Warningf("Plugin %s hook %d at %s failed (ignored): %v", hook.pluginName, hook.index, hookPoint, err)
		return nil
	default:
		return fmt.Errorf("plugin %s hook %d at %s failed: %w", hook.pluginName, hook.index, hookPoint, err)
	}
}
