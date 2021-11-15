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

	"kubevirt.io/api/core"

	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

type rightsEntry struct {
	allowed bool
	verb    string
	role    string
}

func denyModificationsFor(roles ...string) rights {
	return rights{
		Roles: roles,
		Get:   true,
		List:  true,
		Watch: true,
	}
}

func allowAllFor(roles ...string) rights {
	return rights{
		Roles:            roles,
		Get:              true,
		List:             true,
		Watch:            true,
		Delete:           true,
		Create:           true,
		Update:           true,
		Patch:            true,
		DeleteCollection: true,
	}
}

func allowUpdateFor(roles ...string) rights {
	return rights{
		Roles:  roles,
		Update: true,
	}
}

func allowGetFor(roles ...string) rights {
	return rights{
		Roles: roles,
		Get:   true,
	}
}

func denyAllFor(roles ...string) rights {
	return rights{
		Roles: roles,
	}
}

func denyDeleteCollectionFor(roles ...string) rights {
	r := allowAllFor(roles...)
	r.DeleteCollection = false
	return r
}

type rights struct {
	Roles            []string
	Get              bool
	List             bool
	Watch            bool
	Delete           bool
	Create           bool
	Update           bool
	Patch            bool
	DeleteCollection bool
}

func (r rights) list() (e []rightsEntry) {

	for _, role := range r.Roles {
		e = append(e,
			rightsEntry{
				allowed: r.Get,
				verb:    "get",
				role:    role,
			},
			rightsEntry{
				allowed: r.List,
				verb:    "list",
				role:    role,
			},
			rightsEntry{
				allowed: r.Watch,
				verb:    "watch",
				role:    role,
			},
			rightsEntry{
				allowed: r.Delete,
				verb:    "delete",
				role:    role,
			},
			rightsEntry{
				allowed: r.Create,
				verb:    "create",
				role:    role,
			},
			rightsEntry{
				allowed: r.Update,
				verb:    "update",
				role:    role,
			},
			rightsEntry{
				allowed: r.Patch,
				verb:    "patch",
				role:    role,
			},
			rightsEntry{
				allowed: r.DeleteCollection,
				verb:    "deletecollection",
				role:    role,
			},
		)
	}
	return
}

var _ = Describe("[rfe_id:500][crit:high][arm64][vendor:cnv-qe@redhat.com][level:component][sig-compute]User Access", func() {

	var k8sClient string
	var authClient *authClientV1.AuthorizationV1Client

	doSarRequest := func(group string, resource string, subresource string, namespace string, role string, verb string, expected bool) {
		roleToUser := map[string]string{
			"view":    tests.ViewServiceAccountName,
			"edit":    tests.EditServiceAccountName,
			"admin":   tests.AdminServiceAccountName,
			"default": "default",
		}
		userName, exists := roleToUser[role]
		Expect(exists).To(BeTrue(), fmt.Sprintf("role %s is not defined", role))
		user := fmt.Sprintf("system:serviceaccount:%s:%s", namespace, userName)
		sar := &authv1.SubjectAccessReview{}
		sar.Spec = authv1.SubjectAccessReviewSpec{
			User: user,
			ResourceAttributes: &authv1.ResourceAttributes{
				Namespace:   namespace,
				Verb:        verb,
				Group:       group,
				Version:     v1.GroupVersion.Version,
				Resource:    resource,
				Subresource: subresource,
			},
		}
		result, err := authClient.SubjectAccessReviews().Create(context.Background(), sar, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(result.Status.Allowed).To(Equal(expected), fmt.Sprintf("access check for user '%v' on resource '%v' with subresource '%v' for verb '%v' should have returned '%v'.",
			user,
			resource,
			subresource,
			verb,
			expected,
		))
	}

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
		table.DescribeTable("should verify permissions on resources are correct for view, edit, and admin", func(group string, resource string, accessRights ...rights) {
			namespace := util.NamespaceTestDefault
			for _, accessRight := range accessRights {
				for _, entry := range accessRight.list() {
					By(fmt.Sprintf("verifying sa %s for verb %s on resource %s", entry.role, entry.verb, resource))
					doSarRequest(group, resource, "", namespace, entry.role, entry.verb, entry.allowed)
				}
			}
		},
			table.Entry("[test_id:526]given a vmi",
				core.GroupName,
				"virtualmachineinstances",
				allowAllFor("admin"),
				denyDeleteCollectionFor("edit"),
				denyModificationsFor("view"),
				denyAllFor("default")),

			table.Entry("[test_id:527]given a vm",
				core.GroupName,
				"virtualmachines",
				allowAllFor("admin"),
				denyDeleteCollectionFor("edit"),
				denyModificationsFor("view"),
				denyAllFor("default")),

			table.Entry("[test_id:528]given a vmi preset",
				core.GroupName,
				"virtualmachineinstancepresets",
				allowAllFor("admin"),
				denyDeleteCollectionFor("edit"),
				denyModificationsFor("view"),
				denyAllFor("default")),

			table.Entry("[test_id:529][crit:low]given a vmi replica set",
				core.GroupName,
				"virtualmachineinstancereplicasets",
				allowAllFor("admin"),
				denyDeleteCollectionFor("edit"),
				denyModificationsFor("view"),
				denyAllFor("default")),

			table.Entry("[test_id:3230]given a vmi migration",
				core.GroupName,
				"virtualmachineinstancemigrations",
				allowAllFor("admin"),
				denyDeleteCollectionFor("edit"),
				denyModificationsFor("view"),
				denyAllFor("default")),

			table.Entry("[test_id:5243]given a vmsnapshot",
				v1alpha1.SchemeGroupVersion.Group,
				"virtualmachinesnapshots",
				allowAllFor("admin"),
				denyDeleteCollectionFor("edit"),
				denyModificationsFor("view"),
				denyAllFor("default")),

			table.Entry("[test_id:5244]given a vmsnapshotcontent",
				v1alpha1.SchemeGroupVersion.Group,
				"virtualmachinesnapshotcontents",
				allowAllFor("admin"),
				denyDeleteCollectionFor("edit"),
				denyModificationsFor("view"),
				denyAllFor("default")),
			table.Entry("[test_id:5245]given a vmsrestore",
				v1alpha1.SchemeGroupVersion.Group,
				"virtualmachinerestores",
				allowAllFor("admin"),
				denyDeleteCollectionFor("edit"),
				denyModificationsFor("view"),
				denyAllFor("default")),
		)

		table.DescribeTable("should verify permissions on subresources are correct for view, edit, admin and default", func(resource string, subresource string, accessRights ...rights) {
			namespace := util.NamespaceTestDefault
			for _, accessRight := range accessRights {
				for _, entry := range accessRight.list() {
					By(fmt.Sprintf("verifying sa %s for verb %s on resource %s on subresource %s", entry.role, entry.verb, resource, subresource))
					doSarRequest(v1.SubresourceGroupName, resource, subresource, namespace, entry.role, entry.verb, entry.allowed)
				}
			}
		},
			table.Entry("[test_id:3232]on vm start",
				"virtualmachines", "start",
				allowUpdateFor("admin", "edit"),
				denyAllFor("view", "default")),
			table.Entry("[test_id:3233]on vm stop",
				"virtualmachines", "stop",
				allowUpdateFor("admin", "edit"),
				denyAllFor("view", "default")),
			table.Entry("[test_id:3234]on vm restart",
				"virtualmachines", "restart",
				allowUpdateFor("admin", "edit"),
				denyAllFor("view", "default")),
			table.Entry("on vmi guestosinfo",
				"virtualmachineinstances", "guestosinfo",
				allowGetFor("admin", "edit", "view"),
				denyAllFor("default")),
			table.Entry("on vmi userlist",
				"virtualmachineinstances", "userlist",
				allowGetFor("admin", "edit", "view"),
				denyAllFor("default")),

			table.Entry("on vmi filesystemlist",
				"virtualmachineinstances", "filesystemlist",
				allowGetFor("admin", "edit", "view"),
				denyAllFor("default")),
			table.Entry("on vmi addvolume",
				"virtualmachineinstances", "addvolume",
				allowUpdateFor("admin", "edit"),
				denyAllFor("view", "default")),
			table.Entry("on vmi removevolume",
				"virtualmachineinstances", "removevolume",
				allowUpdateFor("admin", "edit"),
				denyAllFor("view", "default")),
			table.Entry("on vmi freeze",
				"virtualmachineinstances", "freeze",
				allowUpdateFor("admin", "edit"),
				denyAllFor("view", "default")),
			table.Entry("on vmi unfreeze",
				"virtualmachineinstances", "unfreeze",
				allowUpdateFor("admin", "edit"),
				denyAllFor("view", "default")),
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

				stdOut, stdErr, err = tests.RunCommandWithNS("", k8sClient, "project", util.NamespaceTestDefault)
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
				stdOut, stdErr, err = tests.RunCommandWithNS("", k8sClient, "adm", "policy", "add-role-to-user", "-n", util.NamespaceTestDefault, "admin", testUser)
				Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)

				stdOut, stdErr, err = tests.RunCommandWithNS("", k8sClient, "project", util.NamespaceTestDefault)
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
				table.Entry("[test_id:2921]given a vmi", "virtualmachineinstances/guestosinfo", "get"),
			)
		})
	})
})
