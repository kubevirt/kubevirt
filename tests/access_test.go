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
	"fmt"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	authv1 "k8s.io/api/authorization/v1"
	authClientV1 "k8s.io/client-go/kubernetes/typed/authorization/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("[rfe_id:500][crit:high][vendor:cnv-qe@redhat.com][level:component]User Access", func() {

	tests.FlagParse()

	view := tests.ViewServiceAccountName
	edit := tests.EditServiceAccountName
	admin := tests.AdminServiceAccountName

	const VIEW_VERBS = "view"
	const EDIT_VERBS = "edit"
	const ADMIN_VERBS = "admin"

	const namespace = tests.NamespaceTestDefault
	var verbs = []string{"get", "list", "watch", "delete", "create", "update", "patch", "deletecollection"}

	var k8sClient string

	BeforeEach(func() {
		k8sClient = tests.GetK8sCmdClient()
		tests.SkipIfNoCmd(k8sClient)

		tests.BeforeTestCleanup()
	})

	Describe("With default kubevirt service accounts", func() {

		Describe("should verify permissions on resources", func() {

			getEmptyPermissions := func() (viewRole map[string]bool, editRole map[string]bool, adminRole map[string]bool) {
				viewRole = make(map[string]bool)
				editRole = make(map[string]bool)
				adminRole = make(map[string]bool)
				return
			}

			getResourcePermissions := func() []map[string]bool {
				viewRole, editRole, adminRole := getEmptyPermissions()

				// get, list, watch
				viewRole[VIEW_VERBS] = true
				editRole[VIEW_VERBS] = true
				adminRole[VIEW_VERBS] = true

				// delete, create, update, watch
				editRole[EDIT_VERBS] = true
				adminRole[EDIT_VERBS] = true

				// deletecollection
				adminRole[ADMIN_VERBS] = true

				// everything else is false and forbidden

				return []map[string]bool{viewRole, editRole, adminRole}
			}

			getMigrationResourcePermissions := func() []map[string]bool {
				viewRole, editRole, adminRole := getEmptyPermissions()

				// get, list, watch
				viewRole[VIEW_VERBS] = true
				editRole[VIEW_VERBS] = true
				adminRole[VIEW_VERBS] = true

				// everything else is false and forbidden

				return []map[string]bool{viewRole, editRole, adminRole}
			}

			table.DescribeTable("are correct for view, edit, and admin", func(resource string, permissions []map[string]bool) {

				viewRole := permissions[0]
				editRole := permissions[1]
				adminRole := permissions[2]

				toYesOrNo := func(in bool) string {
					if in {
						return "yes"
					}
					return "no"
				}

				for _, verb := range verbs {

					var verbGroup string
					switch verb {
					case "get", "list", "watch":
						verbGroup = VIEW_VERBS
					case "delete", "create", "update", "patch":
						verbGroup = EDIT_VERBS
					case "deletecollection":
						verbGroup = ADMIN_VERBS
					}

					// VIEW
					By(fmt.Sprintf("verifying VIEW sa for verb %s", verb))
					expectedRes, _ := viewRole[verbGroup]
					as := fmt.Sprintf("system:serviceaccount:%s:%s", namespace, view)
					result, _, _ := tests.RunCommand(k8sClient, "auth", "can-i", "--as", as, verb, resource)
					Expect(result).To(ContainSubstring(toYesOrNo(expectedRes)))

					// EDIT
					By(fmt.Sprintf("verifying EDIT sa for verb %s", verb))
					expectedRes, _ = editRole[verbGroup]
					as = fmt.Sprintf("system:serviceaccount:%s:%s", namespace, edit)
					result, _, _ = tests.RunCommand(k8sClient, "auth", "can-i", "--as", as, verb, resource)
					Expect(result).To(ContainSubstring(toYesOrNo(expectedRes)))

					// ADMIN
					By(fmt.Sprintf("verifying ADMIN sa for verb %s", verb))
					expectedRes, _ = adminRole[verbGroup]
					as = fmt.Sprintf("system:serviceaccount:%s:%s", namespace, admin)
					result, _, _ = tests.RunCommand(k8sClient, "auth", "can-i", "--as", as, verb, resource)
					Expect(result).To(ContainSubstring(toYesOrNo(expectedRes)))

					// DEFAULT - the default should always return 'no' for ever verb.
					// This is primarily a sanity check.
					By(fmt.Sprintf("verifying DEFAULT sa for verb %s", verb))
					expectedRes = false
					as = fmt.Sprintf("system:serviceaccount:%s:default", namespace)
					result, _, _ = tests.RunCommand(k8sClient, "auth", "can-i", "--as", as, verb, resource)
					Expect(result).To(ContainSubstring(toYesOrNo(expectedRes)))

				}

			},
				table.Entry("[test_id:526]given a vmi", "virtualmachineinstances", getResourcePermissions()),
				table.Entry("[test_id:527]given a vm", "virtualmachines", getResourcePermissions()),
				table.Entry("[test_id:528]given a vmi preset", "virtualmachineinstancepresets", getResourcePermissions()),
				table.Entry("[test_id:529][crit:low]given a vmi replica set", "virtualmachineinstancereplicasets", getResourcePermissions()),
				table.Entry("given a vmi migration", "virtualmachineinstancemigrations", getMigrationResourcePermissions()),
			)

		})

		Describe("should verify permissions on subresources", func() {

			var authClient *authClientV1.AuthorizationV1Client
			It("Prepare auth client", func() {
				virtClient, err := kubecli.GetKubevirtClient()
				Expect(err).ToNot(HaveOccurred())
				authClient, err = authClientV1.NewForConfig(virtClient.Config())
				Expect(err).ToNot(HaveOccurred())
			})

			getEmptyAllowed := func() map[string]map[string]bool {
				// map from role to verb to isAllowed
				allowed := make(map[string]map[string]bool)
				allowed["view"] = make(map[string]bool)
				allowed["edit"] = make(map[string]bool)
				allowed["admin"] = make(map[string]bool)
				return allowed
			}

			getAllowed := func() map[string]map[string]bool {
				allowed := getEmptyAllowed()
				allowed["edit"]["update"] = true
				allowed["admin"]["update"] = true
				// everything else is false and forbidden
				return allowed
			}

			table.DescribeTable("are correct for view, edit, and admin", func(resource string, subresource string, allowed map[string]map[string]bool) {

				// kubectl / oc auth can-i does not seem to work for subresources defined by aggregated apiservers
				// so we use a SubjectAccessReview request
				doSarRequest := func(resource string, subresource string, user string, verb string, expected bool) {
					sar := &authv1.SubjectAccessReview{}
					sar.Spec = authv1.SubjectAccessReviewSpec{
						User: user,
						ResourceAttributes: &authv1.ResourceAttributes{
							Namespace:   namespace,
							Verb:        verb,
							Group:       v1.SubresourceGroupName,
							Version:     v1.GroupVersion.Version,
							Resource:    resource,
							Subresource: subresource,
						},
					}
					result, err := authClient.SubjectAccessReviews().Create(sar)
					Expect(err).ToNot(HaveOccurred())
					Expect(result.Status.Allowed).To(Equal(expected))
				}

				for _, verb := range verbs {
					// VIEW
					By(fmt.Sprintf("verifying VIEW sa for verb %s", verb))
					expectedRes := allowed["view"][verb]
					user := fmt.Sprintf("system:serviceaccount:%s:%s", namespace, view)
					doSarRequest(resource, subresource, user, verb, expectedRes)

					// EDIT
					By(fmt.Sprintf("verifying EDIT sa for verb %s", verb))
					expectedRes = allowed["edit"][verb]
					user = fmt.Sprintf("system:serviceaccount:%s:%s", namespace, edit)
					doSarRequest(resource, subresource, user, verb, expectedRes)

					// ADMIN
					By(fmt.Sprintf("verifying ADMIN sa for verb %s", verb))
					expectedRes = allowed["admin"][verb]
					user = fmt.Sprintf("system:serviceaccount:%s:%s", namespace, admin)
					doSarRequest(resource, subresource, user, verb, expectedRes)

					// DEFAULT - the default should always return 'no' for ever verb.
					// This is primarily a sanity check.
					By(fmt.Sprintf("verifying DEFAULT sa for verb %s", verb))
					expectedRes = false
					user = fmt.Sprintf("system:serviceaccount:%s:default", namespace)
					doSarRequest(resource, subresource, user, verb, expectedRes)
				}

			},
				table.Entry("on vm start", "virtualmachines", "start", getAllowed()),
				table.Entry("on vm stop", "virtualmachines", "stop", getAllowed()),
				table.Entry("on vm restart", "virtualmachines", "restart", getAllowed()),
				table.Entry("on vm migrate", "virtualmachines", "migrate", getEmptyAllowed()),
			)

		})

	})

	Describe("With regular OpenShift user", func() {
		BeforeEach(func() {
			tests.SkipIfNoCmd("oc")
		})

		const testUser = "testuser"

		testRights := func(resource, right string) {
			verbsList := []string{"get", "list", "watch", "delete", "create", "update", "patch", "deletecollection"}

			for _, verb := range verbsList {
				// AS A TEST USER
				By(fmt.Sprintf("verifying user rights for verb %s", verb))
				result, _, _ := tests.RunCommand(k8sClient, "auth", "can-i", "--as", testUser, verb, resource)
				Expect(result).To(ContainSubstring(right))
			}
		}

		Context("should fail without admin rights for the project", func() {
			BeforeEach(func() {
				By("Ensuring the cluster has new test user")
				stdOut, stdErr, err := tests.RunCommandWithNS("", k8sClient, "create", "user", testUser)
				Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)

				stdOut, stdErr, err = tests.RunCommandWithNS("", k8sClient, "project", tests.NamespaceTestDefault)
				Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)
			})

			AfterEach(func() {
				stdOut, stdErr, err := tests.RunCommandWithNS("", k8sClient, "delete", "user", testUser)
				Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)
			})

			table.DescribeTable("should verify permissions on resources are correct for view, edit, and admin", func(resource string) {
				testRights(resource, "no")
			},
				table.Entry("[test_id:2921]given a vmi", "virtualmachineinstances"),
				table.Entry("[test_id:2915]given a vm", "virtualmachines"),
				table.Entry("[test_id:2917]given a vmi preset", "virtualmachineinstancepresets"),
				table.Entry("[test_id:2919]given a vmi replica set", "virtualmachineinstancereplicasets"),
			)
		})

		Context("should succeed with admin rights for the project", func() {
			BeforeEach(func() {
				By("Ensuring the cluster has new test user")
				stdOut, stdErr, err := tests.RunCommandWithNS("", k8sClient, "create", "user", testUser)
				Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)

				By("Ensuring user has the admin rights for the test namespace project")
				// This is ussually done in backgroung when creating new user with login and by creating new project by that user
				stdOut, stdErr, err = tests.RunCommandWithNS("", k8sClient, "adm", "policy", "add-role-to-user", "-n", tests.NamespaceTestDefault, "admin", testUser)
				Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)

				stdOut, stdErr, err = tests.RunCommandWithNS("", k8sClient, "project", tests.NamespaceTestDefault)
				Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)
			})

			AfterEach(func() {
				stdOut, stdErr, err := tests.RunCommandWithNS("", k8sClient, "delete", "user", testUser)
				Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)
			})

			table.DescribeTable("should verify permissions on resources are correct the test user", func(resource string) {
				testRights(resource, "yes")
			},
				table.Entry("[test_id:2920]given a vmi", "virtualmachineinstances"),
				table.Entry("[test_id:2831]given a vm", "virtualmachines"),
				table.Entry("[test_id:2916]given a vmi preset", "virtualmachineinstancepresets"),
				table.Entry("[test_id:2918][crit:low]given a vmi replica set", "virtualmachineinstancereplicasets"),
			)
		})
	})
})
