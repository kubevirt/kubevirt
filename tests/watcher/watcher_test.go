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

package watcher

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("ObjectEventWatcher", func() {
	It("should not fail when the context is canceled before the watch is established", func() {
		// the canceled context fails the watch request before the server is contacted
		Expect(kubecli.FlagSet().Parse([]string{"--master=http://127.0.0.1:1"})).To(Succeed())
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default", ResourceVersion: "1"}}
		Expect(func() {
			New(pod).SinceWatchedObjectResourceVersion().Watch(ctx, func(*v1.Event) bool { return false }, "")
		}).NotTo(Panic())
	})
})
