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

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/testsuite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	authClientV1 "k8s.io/client-go/kubernetes/typed/authorization/v1"

	"kubevirt.io/api/core"

	v1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	pool "kubevirt.io/api/pool"
	"kubevirt.io/api/snapshot/v1alpha1"
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

var _ = Describe("[rfe_id:500][crit:high][arm64][vendor:cnv-qe@redhat.com][level:component][sig-compute]User Access", decorators.SigCompute, func() {

	var k8sClient string
	var authClient *authClientV1.AuthorizationV1Client

	doSarRequest := func(group string, resource string, subresource string, namespace string, role string, verb string, expected, clusterWide bool) {
		roleToUser := map[string]string{
			"view":              testsuite.ViewServiceAccountName,
			"instancetype:view": testsuite.ViewInstancetypeServiceAccountName,
			"edit":              testsuite.EditServiceAccountName,
			"admin":             testsuite.AdminServiceAccountName,
			"default":           "default",
		}
		userName, exists := roleToUser[role]
		Expect(exists).To(BeTrue(), fmt.Sprintf("role %s is not defined", role))
		user := fmt.Sprintf("system:serviceaccount:%s:%s", namespace, userName)
		sar := &authv1.SubjectAccessReview{}
		sar.Spec = authv1.SubjectAccessReviewSpec{
			User: user,
			ResourceAttributes: &authv1.ResourceAttributes{
				Verb:        verb,
				Group:       group,
				Version:     v1.GroupVersion.Version,
				Resource:    resource,
				Subresource: subresource,
			},
		}
		if !clusterWide {
			sar.Spec.ResourceAttributes.Namespace = namespace
		}

		result, err := authClient.SubjectAccessReviews().Create(context.Background(), sar, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(result.Status.Allowed).To(Equal(expected), fmt.Sprintf("access check for user '%v' on resource '%v' with subresource '%v' for verb '%v' should have returned '%v'. Gave reason %s.",
			user,
			resource,
			subresource,
			verb,
			expected,
			result.Status.Reason,
		))
	}

	BeforeEach(func() {
		k8sClient = clientcmd.GetK8sCmdClient()
		clientcmd.SkipIfNoCmd(k8sClient)
		virtClient := kubevirt.Client()
		var err error
		authClient, err = authClientV1.NewForConfig(virtClient.Config())
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("With default kubevirt service accounts", func() {
		DescribeTable("should verify permissions on resources are correct for view, edit, and admin", func(group string, resource string, clusterWide bool, accessRights ...rights) {
			namespace := testsuite.GetTestNamespace(nil)
			for _, accessRight := range accessRights {
				for _, entry := range accessRight.list() {
					By(fmt.Sprintf("verifying sa %s for verb %s on resource %s", entry.role, entry.verb, resource))
					doSarRequest(group, resource, "", namespace, entry.role, entry.verb, entry.allowed, clusterWide)
				}
			}
		},
			Entry("[test_id:526]given a vmi",
				core.GroupName,
				"virtualmachineinstances",
				false,
				allowAllFor("admin"),
				denyDeleteCollectionFor("edit"),
				denyModificationsFor("view"),
				denyAllFor("instancetype:view"),
				denyAllFor("default")),

			Entry("[test_id:527]given a vm",
				core.GroupName,
				"virtualmachines",
				false,
				allowAllFor("admin"),
				denyDeleteCollectionFor("edit"),
				denyModificationsFor("view"),
				denyAllFor("instancetype:view"),
				denyAllFor("default")),

			Entry("given a vmpool",
				pool.GroupName,
				"virtualmachinepools",
				false,
				allowAllFor("admin"),
				denyDeleteCollectionFor("edit"),
				denyModificationsFor("view"),
				denyAllFor("instancetype:view"),
				denyAllFor("default")),

			Entry("[test_id:528]given a vmi preset",
				core.GroupName,
				"virtualmachineinstancepresets",
				false,
				allowAllFor("admin"),
				denyDeleteCollectionFor("edit"),
				denyModificationsFor("view"),
				denyAllFor("instancetype:view"),
				denyAllFor("default")),

			Entry("[test_id:529][crit:low]given a vmi replica set",
				core.GroupName,
				"virtualmachineinstancereplicasets",
				false,
				allowAllFor("admin"),
				denyDeleteCollectionFor("edit"),
				denyModificationsFor("view"),
				denyAllFor("instancetype:view"),
				denyAllFor("default")),

			Entry("[test_id:3230]given a vmi migration",
				core.GroupName,
				"virtualmachineinstancemigrations",
				false,
				allowAllFor("admin"),
				denyDeleteCollectionFor("edit"),
				denyModificationsFor("view"),
				denyAllFor("instancetype:view"),
				denyAllFor("default")),

			Entry("[test_id:5243]given a vmsnapshot",
				v1alpha1.SchemeGroupVersion.Group,
				"virtualmachinesnapshots",
				false,
				allowAllFor("admin"),
				denyDeleteCollectionFor("edit"),
				denyModificationsFor("view"),
				denyAllFor("instancetype:view"),
				denyAllFor("default")),

			Entry("[test_id:5244]given a vmsnapshotcontent",
				v1alpha1.SchemeGroupVersion.Group,
				"virtualmachinesnapshotcontents",
				false,
				allowAllFor("admin"),
				denyDeleteCollectionFor("edit"),
				denyModificationsFor("view"),
				denyAllFor("instancetype:view"),
				denyAllFor("default")),
			Entry("[test_id:5245]given a vmsrestore",
				v1alpha1.SchemeGroupVersion.Group,
				"virtualmachinerestores",
				false,
				allowAllFor("admin"),
				denyDeleteCollectionFor("edit"),
				denyModificationsFor("view"),
				denyAllFor("instancetype:view"),
				denyAllFor("default")),
			Entry("[test_id:TODO]given a virtualmachineinstancetype",
				instancetypeapi.GroupName,
				instancetypeapi.PluralResourceName,
				false,
				allowAllFor("admin"),
				denyDeleteCollectionFor("edit"),
				denyModificationsFor("view"),
				// instancetype:view only provides access to the cluster-scoped resources
				denyAllFor("instancetype:view"),
				denyAllFor("default")),
			Entry("[test_id:TODO]given a virtualmachinepreference",
				instancetypeapi.GroupName,
				instancetypeapi.PluralPreferenceResourceName,
				false,
				allowAllFor("admin"),
				denyDeleteCollectionFor("edit"),
				denyModificationsFor("view"),
				// instancetype:view only provides access to the cluster-scoped resources
				denyAllFor("instancetype:view"),
				denyAllFor("default")),
			Entry("[test_id:TODO]given a virtualmachineclusterinstancetype",
				instancetypeapi.GroupName,
				instancetypeapi.ClusterPluralResourceName,
				// only ClusterRoles bound with a ClusterRoleBinding should have access
				true,
				denyAllFor("admin"),
				denyAllFor("edit"),
				denyAllFor("view"),
				denyModificationsFor("instancetype:view"),
				denyAllFor("default")),
			Entry("[test_id:TODO]given a virtualmachineclusterpreference",
				instancetypeapi.GroupName,
				instancetypeapi.ClusterPluralResourceName,
				// only ClusterRoles bound with a ClusterRoleBinding should have access
				true,
				denyAllFor("admin"),
				denyAllFor("edit"),
				denyAllFor("view"),
				denyModificationsFor("instancetype:view"),
				denyAllFor("default")),
		)

		DescribeTable("should verify permissions on subresources are correct for view, edit, admin and default", func(resource string, subresource string, accessRights ...rights) {
			namespace := testsuite.GetTestNamespace(nil)
			for _, accessRight := range accessRights {
				for _, entry := range accessRight.list() {
					By(fmt.Sprintf("verifying sa %s for verb %s on resource %s on subresource %s", entry.role, entry.verb, resource, subresource))
					doSarRequest(v1.SubresourceGroupName, resource, subresource, namespace, entry.role, entry.verb, entry.allowed, false)
				}
			}
		},
			Entry("[test_id:3232]on vm start",
				"virtualmachines", "start",
				allowUpdateFor("admin", "edit"),
				denyAllFor("view", "default")),
			Entry("[test_id:3233]on vm stop",
				"virtualmachines", "stop",
				allowUpdateFor("admin", "edit"),
				denyAllFor("view", "default")),
			Entry("[test_id:3234]on vm restart",
				"virtualmachines", "restart",
				allowUpdateFor("admin", "edit"),
				denyAllFor("view", "default")),
			Entry("on vm expand-spec",
				"virtualmachines", "expand-spec",
				allowGetFor("admin", "edit", "view"),
				denyAllFor("default")),
			Entry("on vm portforward",
				"virtualmachines", "portforward",
				allowGetFor("admin", "edit"),
				denyAllFor("view", "default")),
			Entry("on vmi guestosinfo",
				"virtualmachineinstances", "guestosinfo",
				allowGetFor("admin", "edit", "view"),
				denyAllFor("default")),
			Entry("on vmi userlist",
				"virtualmachineinstances", "userlist",
				allowGetFor("admin", "edit", "view"),
				denyAllFor("default")),
			Entry("on vmi filesystemlist",
				"virtualmachineinstances", "filesystemlist",
				allowGetFor("admin", "edit", "view"),
				denyAllFor("default")),
			Entry("on vmi addvolume",
				"virtualmachineinstances", "addvolume",
				allowUpdateFor("admin", "edit"),
				denyAllFor("view", "default")),
			Entry("on vmi removevolume",
				"virtualmachineinstances", "removevolume",
				allowUpdateFor("admin", "edit"),
				denyAllFor("view", "default")),
			Entry("on vmi freeze",
				"virtualmachineinstances", "freeze",
				allowUpdateFor("admin", "edit"),
				denyAllFor("view", "default")),
			Entry("on vmi unfreeze",
				"virtualmachineinstances", "unfreeze",
				allowUpdateFor("admin", "edit"),
				denyAllFor("view", "default")),
			Entry("on vmi softreboot",
				"virtualmachineinstances", "softreboot",
				allowUpdateFor("admin", "edit"),
				denyAllFor("view", "default")),
			Entry("on vmi portforward",
				"virtualmachineinstances", "portforward",
				allowGetFor("admin", "edit"),
				denyAllFor("view", "default")),
			Entry("on vmi vsock",
				"virtualmachineinstances", "vsock",
				denyAllFor("admin", "edit", "view", "default")),
			Entry("on expand-vm-spec",
				"expand-vm-spec", "",
				allowUpdateFor("admin", "edit", "view"),
				denyAllFor("default")),
			Entry("on vmi sev/fetchcertchain",
				"virtualmachineinstances", "sev/fetchcertchain",
				allowGetFor("admin", "edit", "view"),
				denyAllFor("default")),
			Entry("on vmi sev/querylaunchmeasurement",
				"virtualmachineinstances", "sev/querylaunchmeasurement",
				allowGetFor("admin", "edit", "view"),
				denyAllFor("default")),
			Entry("on vmi sev/setupsession",
				"virtualmachineinstances", "sev/setupsession",
				allowUpdateFor("admin", "edit"),
				denyAllFor("default")),
			Entry("on vmi sev/injectlaunchsecret",
				"virtualmachineinstances", "sev/injectlaunchsecret",
				allowUpdateFor("admin", "edit"),
				denyAllFor("default")),
		)
	})

	Describe("[rfe_id:2919][crit:high][vendor:cnv-qe@redhat.com][level:component] With regular OpenShift user", func() {

		var testUser string

		BeforeEach(func() {
			// Generate unique usernames based on the test namespace which is unique per ginkgo node
			testUser = "testuser-" + testsuite.GetTestNamespace(nil)
			clientcmd.SkipIfNoCmd("oc")
			if !checks.IsOpenShift() {
				Skip("Skip tests which require an openshift managed test user if not running on openshift")
			}
			By("Ensuring the cluster has new test user")
			stdOut, stdErr, err := clientcmd.RunCommandWithNS("", k8sClient, "create", "user", testUser)
			Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)
		})

		testAction := func(resource, verb string, right string) {
			// AS A TEST USER
			By(fmt.Sprintf("verifying user rights for verb %s", verb))
			result, _, _ := clientcmd.RunCommand(k8sClient, "auth", "can-i", "--as", testUser, verb, resource)
			Expect(result).To(ContainSubstring(right), fmt.Sprintf("unexpected permission for %s to %s a %s", testUser, verb, resource))
		}

		testRights := func(resource, right string) {
			verbsList := []string{"get", "list", "watch", "delete", "create", "update", "patch", "deletecollection"}

			for _, verb := range verbsList {
				testAction(resource, verb, right)
			}
		}

		Context("should fail without admin rights for the project", func() {
			BeforeEach(func() {
				stdOut, stdErr, err := clientcmd.RunCommandWithNS("", k8sClient, "project", testsuite.GetTestNamespace(nil))
				Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)
			})

			AfterEach(func() {
				stdOut, stdErr, err := clientcmd.RunCommandWithNS("", k8sClient, "delete", "user", testUser)
				Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)
			})

			DescribeTable("should verify permissions on resources are correct for view, edit, and admin", func(resource string) {
				testRights(resource, "no")
			},
				Entry("[test_id:2921]given a vmi", "virtualmachineinstances"),
				Entry("[test_id:2915]given a vm", "virtualmachines"),
				Entry("given a vmpool", "virtualmachinepools"),
				Entry("[test_id:2917]given a vmi preset", "virtualmachineinstancepresets"),
				Entry("[test_id:2919]given a vmi replica set", "virtualmachineinstancereplicasets"),
				Entry("[test_id:3235]given a vmi migration", "virtualmachineinstancemigrations"),
				Entry("[test_id:5246]given a vmsnapshot", "virtualmachinesnapshots"),
				Entry("[test_id:5247]given a vmsnapshotcontent", "virtualmachinesnapshotcontents"),
				Entry("[test_id:5248]given a vmsrestore", "virtualmachinerestores"),
			)

			DescribeTable("should verify permissions on resources are correct for subresources", func(resource string, action string) {
				testAction(resource, action, "no")
			},
				Entry("[test_id:2921]given a vmi (pause)", "virtualmachineinstances/pause", "update"),
				Entry("[test_id:2921]given a vmi (unpause)", "virtualmachineinstances/unpause", "update"),
				Entry("[test_id:2921]given a vmi (softreboot)", "virtualmachineinstances/softreboot", "update"),
				Entry("[test_id:2921]given a vmi (console)", "virtualmachineinstances/console", "get"),
				Entry("[test_id:2921]given a vmi (vnc)", "virtualmachineinstances/vnc", "get"),
				Entry("[test_id:2921]given a vmi (vnc/screenshot)", "virtualmachineinstances/vnc/screenshot", "get"),
			)
		})

		Context("should succeed with admin rights for the project", func() {
			BeforeEach(func() {
				By("Ensuring user has the admin rights for the test namespace project")
				// This is ussually done in backgroung when creating new user with login and by creating new project by that user
				stdOut, stdErr, err := clientcmd.RunCommandWithNS("", k8sClient, "adm", "policy", "add-role-to-user", "-n", testsuite.GetTestNamespace(nil), "admin", testUser)
				Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)

				stdOut, stdErr, err = clientcmd.RunCommandWithNS("", k8sClient, "project", testsuite.GetTestNamespace(nil))
				Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)
			})

			AfterEach(func() {
				stdOut, stdErr, err := clientcmd.RunCommandWithNS("", k8sClient, "delete", "user", testUser)
				Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)
			})

			DescribeTable("should verify permissions on resources are correct the test user", func(resource string) {
				testRights(resource, "yes")
			},
				Entry("[test_id:2920]given a vmi", "virtualmachineinstances"),
				Entry("[test_id:2831]given a vm", "virtualmachines"),
				Entry("given a vmpool", "virtualmachinepools"),
				Entry("[test_id:2916]given a vmi preset", "virtualmachineinstancepresets"),
				Entry("[test_id:2918][crit:low]given a vmi replica set", "virtualmachineinstancereplicasets"),
				Entry("[test_id:2837]given a vmi migration", "virtualmachineinstancemigrations"),
				Entry("[test_id:5249]given a vmsnapshot", "virtualmachinesnapshots"),
				Entry("[test_id:5250]given a vmsnapshotcontent", "virtualmachinesnapshotcontents"),
				Entry("[test_id:5251]given a vmsrestore", "virtualmachinerestores"),
			)

			DescribeTable("should verify permissions on resources are correct for subresources", func(resource string, action string) {
				testAction(resource, action, "yes")
			},
				Entry("[test_id:2921]given a vmi (pause)", "virtualmachineinstances/pause", "update"),
				Entry("[test_id:2921]given a vmi (unpause)", "virtualmachineinstances/unpause", "update"),
				Entry("[test_id:2921]given a vmi (softreboot)", "virtualmachineinstances/softreboot", "update"),
				Entry("[test_id:2921]given a vmi (console)", "virtualmachineinstances/console", "get"),
				Entry("[test_id:2921]given a vmi (vnc)", "virtualmachineinstances/vnc", "get"),
				Entry("[test_id:2921]given a vmi (vnc/screenshot)", "virtualmachineinstances/vnc/screenshot", "get"),
				Entry("[test_id:2921]given a vmi (guestosinfo)", "virtualmachineinstances/guestosinfo", "get"),
				Entry("[test_id:2921]given a vmi (sev/fetchcertchain)", "virtualmachineinstances/sev/fetchcertchain", "get"),
				Entry("[test_id:2921]given a vmi (sev/querylaunchmeasurement)", "virtualmachineinstances/sev/querylaunchmeasurement", "get"),
				Entry("[test_id:2921]given a vmi (sev/setupsession)", "virtualmachineinstances/sev/setupsession", "update"),
				Entry("[test_id:2921]given a vmi (sev/injectlaunchsecret)", "virtualmachineinstances/sev/injectlaunchsecret", "update"),
			)
		})
	})
})
