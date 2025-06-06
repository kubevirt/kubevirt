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

package passt_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/network/passt"
)

var _ = Describe("passt-repair", func() {

	Context("successful runner", func() {
		var tmpDir, fileToClean string

		BeforeEach(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "dummy-dir")
			Expect(err).ToNot(HaveOccurred())
			fileToClean = filepath.Join(tmpDir, "file-to-clean")
			err = os.WriteFile(fileToClean, []byte("content"), 0666)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(tmpDir)
		})

		It("goroutine should exit and run cleanup", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			passtRepairCommand := passt.NewFakeCommand(ctx, false, false)
			runner := passt.NewPasstRepairRunner(passtRepairCommand)
			wg := &sync.WaitGroup{}
			wg.Add(1)
			cleanupFunc := func() {
				os.RemoveAll(fileToClean)
				//hack to make the test wait for the goroutine to exit, since cleanup happens last.
				wg.Done()
			}
			Expect(fileToClean).To(BeAnExistingFile())
			go runner.RunContextual(ctx, cleanupFunc)
			wg.Wait()
			Expect(passtRepairCommand.IsStartCalled()).To(BeTrue())
			Expect(passtRepairCommand.IsWaitCalled()).To(BeTrue())
			Expect(fileToClean).ShouldNot(BeAnExistingFile())
		})
	})
	Context("erred runner", func() {
		DescribeTable("should identify errors correctly",
			func(timeout time.Duration, isStartError, isWaitError bool, expectedError error) {

				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				defer cancel()
				passtRepairCommand := passt.NewFakeCommand(ctx, isStartError, isWaitError)
				runner := passt.NewPasstRepairRunner(passtRepairCommand)
				//execution is synchronous to identify retuned error
				err := runner.RunContextual(ctx, func() {})
				Expect(errors.Is(err, expectedError)).To(BeTrue())

			},
			Entry("successful run", 2*time.Second, false, false, nil),
			Entry("timeout", 10*time.Millisecond, false, false, context.DeadlineExceeded),
			Entry("command start error", 2*time.Second, true, false, passt.StartError{}),
			Entry("error while waiting for completion", 2*time.Second, false, true, passt.WaitError{}),
		)
	})
})
