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
	"fmt"
	"runtime"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("[sig-compute]Version", decorators.SigCompute, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Describe("Check that version parameters where loaded by ldflags in build time", func() {
		It("[test_id:555]Should return a good version information struct", func() {
			info, err := virtClient.ServerVersion().Get()
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Compiler).To(Equal(runtime.Compiler))
			Expect(info.Platform).To(Equal(fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)))
			Expect(info.GitVersion).To(Not(SatisfyAny(Equal("v0.0.0-master+$Format:%h$"), Equal("{gitVersion}"))))
			Expect(info.GitCommit).To(Not(SatisfyAny(Equal("$Format:%H$"), Equal("{gitCommit}"))))
			Expect(info.BuildDate).To(Not(SatisfyAny(Equal("1970-01-01T00:00:00Z"), Equal("{buildDate}"))))
		})
	})
})
