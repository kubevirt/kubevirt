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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package tests_test

import (
	"flag"
	"fmt"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("User Access", func() {

	flag.Parse()

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Describe("With default kubevirt service accounts", func() {
		table.DescribeTable("should verify permissions are correct for view, edit, and admin", func(resource string) {
			tests.SkipIfNoKubectl()

			view := tests.ViewServiceAccountName
			edit := tests.EditServiceAccountName
			admin := tests.AdminServiceAccountName

			viewVerbs := make(map[string]string)
			editVerbs := make(map[string]string)
			adminVerbs := make(map[string]string)

			// GET
			viewVerbs["get"] = "yes"
			editVerbs["get"] = "yes"
			adminVerbs["get"] = "yes"

			// List
			viewVerbs["list"] = "yes"
			editVerbs["list"] = "yes"
			adminVerbs["list"] = "yes"

			// WATCH
			viewVerbs["watch"] = "yes"
			editVerbs["watch"] = "yes"
			adminVerbs["watch"] = "yes"

			// DELETE
			viewVerbs["delete"] = "no"
			editVerbs["delete"] = "yes"
			adminVerbs["delete"] = "yes"

			// CREATE
			viewVerbs["create"] = "no"
			editVerbs["create"] = "yes"
			adminVerbs["create"] = "yes"

			// UPDATE
			viewVerbs["update"] = "no"
			editVerbs["update"] = "yes"
			adminVerbs["update"] = "yes"

			// PATCH
			viewVerbs["patch"] = "no"
			editVerbs["patch"] = "yes"
			adminVerbs["patch"] = "yes"

			// DELETE COllECTION
			viewVerbs["deleteCollection"] = "no"
			editVerbs["deleteCollection"] = "no"
			adminVerbs["deleteCollection"] = "yes"

			namespace := tests.NamespaceTestDefault
			verbs := []string{"get", "list", "watch", "delete", "create", "update", "patch", "deletecollection"}

			for _, verb := range verbs {
				// VIEW
				By(fmt.Sprintf("verifying VIEW sa for verb %s", verb))
				expectedRes, _ := viewVerbs[verb]
				as := fmt.Sprintf("system:serviceaccount:%s:%s", namespace, view)
				result, err := tests.RunKubectlCommand("auth", "can-i", "--as", as, verb, resource)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(ContainSubstring(expectedRes))

				// EDIT
				By(fmt.Sprintf("verifying EDIT sa for verb %s", verb))
				expectedRes, _ = editVerbs[verb]
				as = fmt.Sprintf("system:serviceaccount:%s:%s", namespace, edit)
				result, err = tests.RunKubectlCommand("auth", "can-i", "--as", as, verb, resource)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(ContainSubstring(expectedRes))

				// ADMIN
				By(fmt.Sprintf("verifying ADMIN sa for verb %s", verb))
				expectedRes, _ = adminVerbs[verb]
				as = fmt.Sprintf("system:serviceaccount:%s:%s", namespace, admin)
				result, err = tests.RunKubectlCommand("auth", "can-i", "--as", as, verb, resource)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(ContainSubstring(expectedRes))

				// DEFAULT - the default should always return 'no' for ever verb.
				// This is primarily a sanity check.
				By(fmt.Sprintf("verifying DEFAULT sa for verb %s", verb))
				expectedRes = "no"
				as = fmt.Sprintf("system:serviceaccount:%s:default", namespace)
				result, err = tests.RunKubectlCommand("auth", "can-i", "--as", as, verb, resource)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(ContainSubstring(expectedRes))
			}
		},
			table.Entry("given a vmi", "virtualmachineinstances"),
			table.Entry("given an vm", "virtualmachines"),
			table.Entry("given a vmi preset", "virtualmachineinstancepresets"),
			table.Entry("given a vmi replica set", "virtualmachineinstancereplicasets"),
		)
	})
})
