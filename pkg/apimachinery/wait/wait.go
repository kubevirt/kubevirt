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

package wait

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// PollImmediately polls at fixed intervals without backoff, passing a context to the condition function.
// The condition function is invoked immediately before waiting and guarantees at least one invocation,
// even if the context is already cancelled. This is useful when the condition needs immediate evaluation
// and access to the context (e.g., for cancellation).
func PollImmediately(interval, timeout time.Duration, condition wait.ConditionWithContextFunc) error {
	return wait.PollUntilContextTimeout(context.Background(), interval, timeout, true, condition)
}
