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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"
)

var _ = Describe("handleHookError", func() {
	hook := func(strategy pluginv1alpha1.FailureStrategy) resolvedHook {
		return resolvedHook{pluginName: "a", index: 3, failureStrategy: strategy}
	}

	canceledContext := func() context.Context {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}

	DescribeTable("applies the failure semantics shared by every launcher hook point",
		func(parentCtx context.Context, strategy pluginv1alpha1.FailureStrategy, err error, verify func(error)) {
			verify(handleHookError(parentCtx, pluginv1alpha1.LauncherHookGuestDefinition, hook(strategy), err))
		},
		Entry("succeeds when the hook succeeded", context.Background(), pluginv1alpha1.FailureStrategyFail, nil, func(err error) {
			Expect(err).To(Succeed())
		}),
		Entry("aborts when a Fail-strategy hook failed", context.Background(), pluginv1alpha1.FailureStrategyFail, fmt.Errorf("boom"), func(err error) {
			Expect(err).To(MatchError(ContainSubstring("plugin a hook 3")))
			Expect(err).To(MatchError(ContainSubstring(string(pluginv1alpha1.LauncherHookGuestDefinition))))
			Expect(err).To(MatchError(ContainSubstring("boom")))
		}),
		Entry("continues when an Ignore-strategy hook failed", context.Background(), pluginv1alpha1.FailureStrategyIgnore, fmt.Errorf("boom"), func(err error) {
			Expect(err).To(Succeed())
		}),
		Entry("aborts on cancellation even when the failure strategy is Ignore", canceledContext(), pluginv1alpha1.FailureStrategyIgnore, fmt.Errorf("boom"), func(err error) {
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, context.Canceled)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("boom"), "the hook's own error must be reported alongside the cancellation cause")
		}),
		Entry("aborts on cancellation with a Fail-strategy hook", canceledContext(), pluginv1alpha1.FailureStrategyFail, fmt.Errorf("checking socket: %w", context.Canceled), func(err error) {
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, context.Canceled)).To(BeTrue())
		}),
	)
})
