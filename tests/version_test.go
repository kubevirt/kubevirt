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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests_test

import (
	"flag"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"fmt"
	"runtime"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Version", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Describe("Check that version parameters where loaded by ldflags in build time", func() {
		It("Should return a good version information struct", func() {
			info, err := virtClient.ServerVersion().Get()
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Compiler).To(Equal(runtime.Compiler))
			Expect(info.Platform).To(Equal(fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)))
			Expect(info.GitVersion).To(Not(Equal("v0.0.0-master+$Format:%h$")))
			Expect(info.GitCommit).To(Not(Equal("$Format:%H$")))
			Expect(info.BuildDate).To(Not(Equal("1970-01-01T00:00:00Z")))
		})
	})

})
