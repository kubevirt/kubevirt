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

package apply

import (
	"fmt"

	"kubevirt.io/kubevirt/pkg/controller"

	"k8s.io/client-go/tools/cache"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	kubevirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

var _ = Describe("RBAC test", func() {

	var (
		clientset                  *kubecli.MockKubevirtClient
		ctrl                       *gomock.Controller
		k8sClient                  *fake.Clientset
		kv                         *kubevirtv1.KubeVirt
		stores                     util.Stores
		expectations               *util.Expectations
		version, imageRegistry, id string
	)

	const (
		roleType               string = "roles"
		clusterRoleType        string = "clusterroles"
		roleBindingType        string = "rolebindings"
		clusterRoleBindingType string = "clusterrolebindings"
	)

	getTypeName := func(object runtime.Object) (objType string) {
		switch object.(type) {
		case *rbacv1.Role:
			objType = roleType
		case *rbacv1.ClusterRole:
			objType = clusterRoleType
		case *rbacv1.RoleBinding:
			objType = roleBindingType
		case *rbacv1.ClusterRoleBinding:
			objType = clusterRoleBindingType
		default:
			Expect(false).To(BeTrue(), "such type is unknown")
		}
		return objType
	}
	expectEqual := func(object1, object2 runtime.Object) {
		Expect(getTypeName(object1)).To(Equal(getTypeName(object2)))
		switch object1.(type) {
		case *rbacv1.Role:
			obj1Casted := object1.(*rbacv1.Role)
			obj2Casted := object2.(*rbacv1.Role)
			Expect(obj1Casted).To(Equal(obj2Casted))
		case *rbacv1.ClusterRole:
			obj1Casted := object1.(*rbacv1.ClusterRole)
			obj2Casted := object2.(*rbacv1.ClusterRole)
			Expect(obj1Casted).To(Equal(obj2Casted))
		case *rbacv1.RoleBinding:
			obj1Casted := object1.(*rbacv1.RoleBinding)
			obj2Casted := object2.(*rbacv1.RoleBinding)
			Expect(obj1Casted).To(Equal(obj2Casted))
		case *rbacv1.ClusterRoleBinding:
			obj1Casted := object1.(*rbacv1.ClusterRoleBinding)
			obj2Casted := object2.(*rbacv1.ClusterRoleBinding)
			Expect(obj1Casted).To(Equal(obj2Casted))
		}
	}
	expectRbacUpdate := func(object runtime.Object) {
		k8sClient.Fake.PrependReactor("update", getTypeName(object), func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			update, ok := action.(testing.UpdateActionImpl)
			Expect(ok).To(BeTrue())

			expectEqual(update.GetObject(), object)
			return true, update.GetObject(), nil
		})
	}
	newFakePolicyRules := func(fakeNames ...string) []rbacv1.PolicyRule {
		rules := make([]rbacv1.PolicyRule, len(fakeNames))

		for idx, fakeName := range fakeNames {
			rules[idx] = rbacv1.PolicyRule{
				Verbs:           []string{fmt.Sprintf("fakeVerb-%s", fakeName)},
				APIGroups:       []string{fmt.Sprintf("apiGroups-%s", fakeName)},
				Resources:       []string{fmt.Sprintf("Resources-%s", fakeName)},
				ResourceNames:   []string{fmt.Sprintf("ResourceNames-%s", fakeName)},
				NonResourceURLs: []string{fmt.Sprintf("NonResourceURLs-%s", fakeName)},
			}
		}

		return rules
	}
	newFakeSubjects := func(fakeNames ...string) []rbacv1.Subject {
		subjects := make([]rbacv1.Subject, len(fakeNames))

		for idx, fakeName := range fakeNames {
			subjects[idx] = rbacv1.Subject{
				Kind:      fmt.Sprintf("fakeKind-%s", fakeName),
				APIGroup:  fmt.Sprintf("APIGroup-%s", fakeName),
				Name:      fmt.Sprintf("Name-%s", fakeName),
				Namespace: fmt.Sprintf("Namespace-%s", fakeName),
			}
		}

		return subjects
	}
	newFakeRoleRef := func(fakeName string) rbacv1.RoleRef {
		return rbacv1.RoleRef{
			Kind:     fmt.Sprintf("fakeKind-%s", fakeName),
			APIGroup: rbacv1.GroupName,
			Name:     fmt.Sprintf("Name-%s", fakeName),
		}
	}
	newEmptyResource := func(resourceType string) (object runtime.Object) {
		By("Initializing object")

		switch resourceType {
		case roleType:
			object = &rbacv1.Role{}
		case clusterRoleType:
			object = &rbacv1.ClusterRole{}
		case roleBindingType:
			object = &rbacv1.RoleBinding{}
		case clusterRoleBindingType:
			object = &rbacv1.ClusterRoleBinding{}
		default:
			Expect(false).To(BeTrue(), "unknown type")
		}
		return
	}

	assignRulesToRoles := func(rules []rbacv1.PolicyRule, objects ...runtime.Object) {
		By("Assigning rules to role resources")

		for _, object := range objects {
			switch object.(type) {
			case *rbacv1.Role:
				role := object.(*rbacv1.Role)
				role.Rules = rules
			case *rbacv1.ClusterRole:
				clusterRole := object.(*rbacv1.ClusterRole)
				clusterRole.Rules = rules
			default:
				Expect(false).To(BeTrue(), "Rule assignment is valid only for role / cluster role")
			}
		}
	}
	assignSubjectsToBinding := func(subjects []rbacv1.Subject, objects ...runtime.Object) {
		By("Assigning Subjects to binding resources")

		for _, object := range objects {
			switch object.(type) {
			case *rbacv1.RoleBinding:
				roleBinding := object.(*rbacv1.RoleBinding)
				roleBinding.Subjects = subjects
			case *rbacv1.ClusterRoleBinding:
				clusterRoleBinding := object.(*rbacv1.ClusterRoleBinding)
				clusterRoleBinding.Subjects = subjects
			default:
				Expect(false).To(BeTrue(), "Subject assignment is valid only for roleBinding binding / cluster roleBinding binding")
			}
		}
	}
	assignRoleRefToBinding := func(roleRef rbacv1.RoleRef, objects ...runtime.Object) {
		By("Assigning RoleRef to binding resources")

		for _, object := range objects {
			switch object.(type) {
			case *rbacv1.RoleBinding:
				roleBinding := object.(*rbacv1.RoleBinding)
				roleBinding.RoleRef = roleRef
			case *rbacv1.ClusterRoleBinding:
				clusterRoleBinding := object.(*rbacv1.ClusterRoleBinding)
				clusterRoleBinding.RoleRef = roleRef
			default:
				Expect(false).To(BeTrue(), "RoleRef assignment is valid only for roleBinding binding / cluster roleBinding binding")
			}
		}
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		k8sClient = fake.NewSimpleClientset()

		k8sClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil(), "expect no updates by default")
			return true, nil, nil
		})

		kv = &kubevirtv1.KubeVirt{}
		stores = util.Stores{}
		stores.RoleCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.RoleBindingCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.ClusterRoleCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.ClusterRoleBindingCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)

		expectations = &util.Expectations{}
		expectations.Role = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("Role"))
		expectations.RoleBinding = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("RoleBinding"))
		expectations.ClusterRole = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ClusterRole"))
		expectations.ClusterRoleBinding = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ClusterRoleBinding"))

		clientset = kubecli.NewMockKubevirtClient(ctrl)

		version, imageRegistry, id = getTargetVersionRegistryID(kv)
	})

	Context("when reconciling", func() {

		var reconciler Reconciler

		updateResource := func(required runtime.Object) error {
			By("Updating resource")

			switch required.(type) {
			case *rbacv1.Role:
				return reconciler.createOrUpdateRole(required.(*rbacv1.Role), version, imageRegistry, id)
			case *rbacv1.ClusterRole:
				return reconciler.createOrUpdateClusterRole(required.(*rbacv1.ClusterRole), version, imageRegistry, id)
			case *rbacv1.RoleBinding:
				return reconciler.createOrUpdateRoleBinding(required.(*rbacv1.RoleBinding), version, imageRegistry, id)
			case *rbacv1.ClusterRoleBinding:
				return reconciler.createOrUpdateClusterRoleBinding(required.(*rbacv1.ClusterRoleBinding), version, imageRegistry, id)
			default:
				Expect(false).To(BeTrue(), "such type is unknown")
				return nil
			}
		}
		addToCache := func(object runtime.Object) {
			By("Adding object to cache")
			var err error

			injectOperatorMetadata(kv, getRbacMetaObject(object), version, imageRegistry, id, true)
			err = getRbacCache(&reconciler, object).Add(object)

			kind := object.GetObjectKind().GroupVersionKind().Kind
			Expect(err).ShouldNot(HaveOccurred(), "error while adding %s object to cache: %v", kind, err)
		}

		BeforeEach(func() {
			By("initialize reconciler")
			reconciler = Reconciler{
				kv:             kv,
				targetStrategy: nil,
				stores:         stores,
				virtClientset:  clientset,
				k8sClientset:   k8sClient,
				expectations:   expectations,
			}
		})

		DescribeTable("Check reconciliation of PolocyRules for", func(resourceType string, changeExisting bool) {

			Expect(resourceType).To(Or(Equal(roleType), Equal(clusterRoleType)))
			existing := newEmptyResource(resourceType)
			required := newEmptyResource(resourceType)

			assignRulesToRoles(newFakePolicyRules("policy1"), existing, required)
			getRbacMetaObject(required).OwnerReferences = []metav1.OwnerReference{}
			addToCache(existing)

			if changeExisting {
				assignRulesToRoles(newFakePolicyRules("policy2"), required)
				expectRbacUpdate(required)
			}

			err := updateResource(required)
			Expect(err).ShouldNot(HaveOccurred())
		},
			Entry("Role resource where resource had changed", roleType, true),
			Entry("Role resource where resource had not changed", roleType, false),
			Entry("ClusterRole resource where resource had changed", clusterRoleType, true),
			Entry("ClusterRole resource where resource had not changed", clusterRoleType, false),
		)

		DescribeTable("Check reconciliation of Subjects and RoleRef for", func(resourceType string, changeExistingSubjects, changeExistingRoleRef bool) {

			Expect(resourceType).To(Or(Equal(roleBindingType), Equal(clusterRoleBindingType)))
			existing := newEmptyResource(resourceType)
			required := newEmptyResource(resourceType)

			assignSubjectsToBinding(newFakeSubjects("policy1"), existing, required)
			assignRoleRefToBinding(newFakeRoleRef("policy1"), existing, required)
			getRbacMetaObject(required).OwnerReferences = []metav1.OwnerReference{}
			addToCache(existing)

			if changeExistingSubjects {
				assignSubjectsToBinding(newFakeSubjects("policy2"), required)
				expectRbacUpdate(required)
			}
			if changeExistingRoleRef {
				assignRoleRefToBinding(newFakeRoleRef("policy2"), required)
				expectRbacUpdate(required)
			}

			err := updateResource(required)
			Expect(err).ShouldNot(HaveOccurred())
		},
			Entry("RoleBinding resource where resource had changed Subjects and RoleRef", roleBindingType, true, true),
			Entry("RoleBinding resource where resource had changed Subjects", roleBindingType, true, false),
			Entry("RoleBinding resource where resource had changed RoleRef", roleBindingType, false, true),
			Entry("RoleBinding resource where resource had not changed", roleBindingType, false, false),

			Entry("ClusterRoleBinding resource where resource had changed Subjects and RoleRef", clusterRoleBindingType, true, true),
			Entry("ClusterRoleBinding resource where resource had changed Subjects", clusterRoleBindingType, true, false),
			Entry("ClusterRoleBinding resource where resource had changed RoleRef", clusterRoleBindingType, false, true),
			Entry("ClusterRoleBinding resource where resource had not changed", clusterRoleBindingType, false, false),
		)

		DescribeTable("when subjects are same but in different order, expect no update for", func(resourceType string) {
			Expect(resourceType).To(Or(Equal(roleBindingType), Equal(clusterRoleBindingType)))
			existing := newEmptyResource(resourceType)
			required := newEmptyResource(resourceType)
			const policy1Name, policy2Name = "policy1", "policy2"

			By("Assigning same subjects but in different order")
			assignSubjectsToBinding(newFakeSubjects(policy1Name, policy2Name), existing, required)
			assignSubjectsToBinding(newFakeSubjects(policy2Name, policy1Name), existing, required)
			addToCache(existing)

			err := updateResource(required)
			Expect(err).ShouldNot(HaveOccurred())
		},
			Entry("RoleBindings", roleBindingType),
			Entry("ClusterRoleBinding", clusterRoleBindingType),
		)

	})
})
