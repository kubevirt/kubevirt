package kubecli

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
* Copyright 2021 Red Hat, Inc.
*
 */

import (
	"context"
	"net/http"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	v1alpha12 "kubevirt.io/api/migrations/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	testPolicyName = "testpolicy"
	proxyPath      = "/proxy/path"
)

var _ = Describe("Kubevirt MigrationPolicy VirtClient", func() {

	var server *ghttp.Server
	var basePath, policyPath string

	BeforeEach(func() {
		basePath = "/apis/migrations.kubevirt.io/v1alpha1/migrationpolicies"
		policyPath = path.Join(basePath, testPolicyName)

		server = ghttp.NewServer()
	})

	expectPoliciesAreEqual := func(actual, expect *v1alpha12.MigrationPolicy) {
		// TODO: Workaround until I figure out what's wrong here
		actual.Kind = expect.Kind
		actual.APIVersion = expect.APIVersion
		Expect(actual).To(Equal(expect))
	}

	DescribeTable("should fetch a MigrationPolicy", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		policy := NewMinimalMigrationPolicy(testPolicyName)
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, policyPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, policy),
		))
		fetchedMigrationPolicy, err := client.MigrationPolicy().Get(context.Background(), testPolicyName, k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		expectPoliciesAreEqual(fetchedMigrationPolicy, policy)
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should detect non existent Migration Policy", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, policyPath)),
			ghttp.RespondWithJSONEncoded(http.StatusNotFound, errors.NewNotFound(schema.GroupResource{}, testPolicyName)),
		))
		_, err = client.MigrationPolicy().Get(context.Background(), testPolicyName, k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).To(HaveOccurred())
		Expect(errors.IsNotFound(err)).To(BeTrue())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should fetch a MigrationPolicy list", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		policy := NewMinimalMigrationPolicy(testPolicyName)
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, basePath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, NewMinimalMigrationPolicyList(*policy)),
		))
		fetchedMigrationPolicy, err := client.MigrationPolicy().List(context.Background(), k8smetav1.ListOptions{})

		Expect(err).ToNot(HaveOccurred())
		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(fetchedMigrationPolicy.Items).To(HaveLen(1))
		expectPoliciesAreEqual(&fetchedMigrationPolicy.Items[0], policy)
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should create a MigrationPolicy", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		policy := NewMinimalMigrationPolicy(testPolicyName)
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", path.Join(proxyPath, basePath)),
			ghttp.RespondWithJSONEncoded(http.StatusCreated, policy),
		))
		createdMigrationPolicy, err := client.MigrationPolicy().Create(context.Background(), policy, k8smetav1.CreateOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		expectPoliciesAreEqual(createdMigrationPolicy, policy)
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should update a MigrationPolicy", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		policy := NewMinimalMigrationPolicy(testPolicyName)
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", path.Join(proxyPath, policyPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, policy),
		))
		updatedMigrationPolicy, err := client.MigrationPolicy().Update(context.Background(), policy, k8smetav1.UpdateOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		expectPoliciesAreEqual(updatedMigrationPolicy, policy)
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should delete a MigrationPolicy", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("DELETE", path.Join(proxyPath, policyPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err = client.MigrationPolicy().Delete(context.Background(), testPolicyName, k8smetav1.DeleteOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	AfterEach(func() {
		server.Close()
	})
})
