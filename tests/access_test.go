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
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	authClientV1 "k8s.io/client-go/kubernetes/typed/authorization/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = FDescribe("[rfe_id:500][crit:high][vendor:cnv-qe@redhat.com][level:component]User Access", func() {

	view := tests.ViewServiceAccountName
	edit := tests.EditServiceAccountName
	admin := tests.AdminServiceAccountName

	var k8sClient string
	var authClient *authClientV1.AuthorizationV1Client

	BeforeEach(func() {
		k8sClient = tests.GetK8sCmdClient()
		tests.SkipIfNoCmd(k8sClient)
		virtClient, err := kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
		authClient, err = authClientV1.NewForConfig(virtClient.Config())
		Expect(err).ToNot(HaveOccurred())

		tests.BeforeTestCleanup()
	})

	Describe("With default kubevirt service accounts", func() {
		table.DescribeTable("should verify permissions on resources are correct for view, edit, and admin", func(resource string) {

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
				result, _, _ := tests.RunCommand(k8sClient, "auth", "can-i", "--as", as, verb, resource)
				Expect(result).To(ContainSubstring(expectedRes))

				// EDIT
				By(fmt.Sprintf("verifying EDIT sa for verb %s", verb))
				expectedRes, _ = editVerbs[verb]
				as = fmt.Sprintf("system:serviceaccount:%s:%s", namespace, edit)
				result, _, _ = tests.RunCommand(k8sClient, "auth", "can-i", "--as", as, verb, resource)
				Expect(result).To(ContainSubstring(expectedRes))

				// ADMIN
				By(fmt.Sprintf("verifying ADMIN sa for verb %s", verb))
				expectedRes, _ = adminVerbs[verb]
				as = fmt.Sprintf("system:serviceaccount:%s:%s", namespace, admin)
				result, _, _ = tests.RunCommand(k8sClient, "auth", "can-i", "--as", as, verb, resource)
				Expect(result).To(ContainSubstring(expectedRes))

				// DEFAULT - the default should always return 'no' for ever verb.
				// This is primarily a sanity check.
				By(fmt.Sprintf("verifying DEFAULT sa for verb %s", verb))
				expectedRes = "no"
				as = fmt.Sprintf("system:serviceaccount:%s:default", namespace)
				result, _, _ = tests.RunCommand(k8sClient, "auth", "can-i", "--as", as, verb, resource)
				Expect(result).To(ContainSubstring(expectedRes))
			}
		},
			table.Entry("[test_id:526]given a vmi", "virtualmachineinstances"),
			table.Entry("[test_id:527]given a vm", "virtualmachines"),
			table.Entry("[test_id:528]given a vmi preset", "virtualmachineinstancepresets"),
			table.Entry("[test_id:529][crit:low]given a vmi replica set", "virtualmachineinstancereplicasets"),
			table.Entry("[test_id:3230]given a vmi migration", "virtualmachineinstancemigrations"),
			table.Entry("[test_id:5243]given a vmsnapshot", "virtualmachinesnapshots"),
			table.Entry("[test_id:5244]given a vmsnapshotcontent", "virtualmachinesnapshotcontents"),
			table.Entry("[test_id:5245]given a vmsrestore", "virtualmachinerestores"),
		)

		table.DescribeTable("should verify permissions on subresources are correct for view, edit, and admin", func(resource string, subresource string) {

			viewVerbs := make(map[string]bool)
			editVerbs := make(map[string]bool)
			adminVerbs := make(map[string]bool)

			// GET
			viewVerbs["get"] = false
			editVerbs["get"] = false
			adminVerbs["get"] = false

			// List
			viewVerbs["list"] = false
			editVerbs["list"] = false
			adminVerbs["list"] = false

			// WATCH
			viewVerbs["watch"] = false
			editVerbs["watch"] = false
			adminVerbs["watch"] = false

			// DELETE
			viewVerbs["delete"] = false
			editVerbs["delete"] = false
			adminVerbs["delete"] = false

			// CREATE
			viewVerbs["create"] = false
			editVerbs["create"] = false
			adminVerbs["create"] = false

			// UPDATE
			viewVerbs["update"] = false
			editVerbs["update"] = true
			adminVerbs["update"] = true

			// PATCH
			viewVerbs["patch"] = false
			editVerbs["patch"] = false
			adminVerbs["patch"] = false

			// DELETE COllECTION
			viewVerbs["deleteCollection"] = false
			editVerbs["deleteCollection"] = false
			adminVerbs["deleteCollection"] = false

			namespace := tests.NamespaceTestDefault
			verbs := []string{"get", "list", "watch", "delete", "create", "update", "patch", "deletecollection"}

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
				result, err := authClient.SubjectAccessReviews().Create(context.Background(), sar, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Status.Allowed).To(Equal(expected))
			}

			for _, verb := range verbs {
				// VIEW
				By(fmt.Sprintf("verifying VIEW sa for verb %s", verb))
				expectedRes, _ := viewVerbs[verb]
				user := fmt.Sprintf("system:serviceaccount:%s:%s", namespace, view)
				doSarRequest(resource, subresource, user, verb, expectedRes)

				// EDIT
				By(fmt.Sprintf("verifying EDIT sa for verb %s", verb))
				expectedRes, _ = editVerbs[verb]
				user = fmt.Sprintf("system:serviceaccount:%s:%s", namespace, edit)
				doSarRequest(resource, subresource, user, verb, expectedRes)

				// ADMIN
				By(fmt.Sprintf("verifying ADMIN sa for verb %s", verb))
				expectedRes, _ = adminVerbs[verb]
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
			table.Entry("[test_id:3232]on vm start", "virtualmachines", "start"),
			table.Entry("[test_id:3233]on vm stop", "virtualmachines", "stop"),
			table.Entry("[test_id:3234]on vm restart", "virtualmachines", "restart"),
		)
	})

	Describe("[Serial][rfe_id:2919][crit:high][vendor:cnv-qe@redhat.com][level:component] With regular OpenShift user", func() {
		BeforeEach(func() {
			tests.SkipIfNoCmd("oc")
			if !tests.IsOpenShift() {
				Skip("Skip tests which require an openshift managed test user if not running on openshift")
			}
		})

		const testUser = "testuser"

		testAction := func(resource, verb string, right string) {
			// AS A TEST USER
			By(fmt.Sprintf("verifying user rights for verb %s", verb))
			result, _, _ := tests.RunCommand(k8sClient, "auth", "can-i", "--as", testUser, verb, resource)
			Expect(result).To(ContainSubstring(right))
		}

		testRights := func(resource, right string) {
			verbsList := []string{"get", "list", "watch", "delete", "create", "update", "patch", "deletecollection"}

			for _, verb := range verbsList {
				testAction(resource, verb, right)
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
				table.Entry("[test_id:3235]given a vmi migration", "virtualmachineinstancemigrations"),
				table.Entry("[test_id:5246]given a vmsnapshot", "virtualmachinesnapshots"),
				table.Entry("[test_id:5247]given a vmsnapshotcontent", "virtualmachinesnapshotcontents"),
				table.Entry("[test_id:5248]given a vmsrestore", "virtualmachinerestores"),
			)

			table.DescribeTable("should verify permissions on resources are correct for subresources", func(resource string, action string) {
				testAction(resource, action, "no")
			},
				table.Entry("[test_id:2921]given a vmi", "virtualmachineinstances/pause", "update"),
				table.Entry("[test_id:2921]given a vmi", "virtualmachineinstances/unpause", "update"),
				table.Entry("[test_id:2921]given a vmi", "virtualmachineinstances/console", "get"),
				table.Entry("[test_id:2921]given a vmi", "virtualmachineinstances/vnc", "get"),
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
				table.Entry("[test_id:2837]given a vmi migration", "virtualmachineinstancemigrations"),
				table.Entry("[test_id:5249]given a vmsnapshot", "virtualmachinesnapshots"),
				table.Entry("[test_id:5250]given a vmsnapshotcontent", "virtualmachinesnapshotcontents"),
				table.Entry("[test_id:5251]given a vmsrestore", "virtualmachinerestores"),
			)

			table.DescribeTable("should verify permissions on resources are correct for subresources", func(resource string, action string) {
				testAction(resource, action, "yes")
			},
				table.Entry("[test_id:2921]given a vmi", "virtualmachineinstances/pause", "update"),
				table.Entry("[test_id:2921]given a vmi", "virtualmachineinstances/unpause", "update"),
				table.Entry("[test_id:2921]given a vmi", "virtualmachineinstances/console", "get"),
				table.Entry("[test_id:2921]given a vmi", "virtualmachineinstances/vnc", "get"),
			)
		})
	})
})
