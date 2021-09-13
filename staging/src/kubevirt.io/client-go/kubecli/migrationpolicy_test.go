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
	"net/http"

	"github.com/onsi/gomega/ghttp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	testPolicyName = "testpolicy"
)

var _ = Describe("Kubevirt MigrationPolicy Client", func() {

	var server *ghttp.Server
	var client KubevirtClient
	var basePath, policyPath string

	BeforeEach(func() {
		basePath = "/apis/kubevirt.io/v1alpha3/namespaces/default/migrationpolicies"
		policyPath = basePath + "/" + testPolicyName

		var err error
		server = ghttp.NewServer()
		client, err = GetKubevirtClientFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should fetch a MigrationPolicy", func() {
		policy := NewMinimalMigrationPolicy(testPolicyName, k8sv1.NamespaceDefault)
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", policyPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, policy),
		))
		fetchedMigrationPolicy, err := client.MigrationPolicy(k8sv1.NamespaceDefault).Get(testPolicyName, &k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedMigrationPolicy).To(Equal(policy))
	})

	It("should detect non existent Migration Policy", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", policyPath),
			ghttp.RespondWithJSONEncoded(http.StatusNotFound, errors.NewNotFound(schema.GroupResource{}, testPolicyName)),
		))
		_, err := client.MigrationPolicy(k8sv1.NamespaceDefault).Get(testPolicyName, &k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).To(HaveOccurred())
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})

	It("should fetch a MigrationPolicy list", func() {
		policy := NewMinimalMigrationPolicy(testPolicyName, k8sv1.NamespaceDefault)
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, NewMinimalMigrationPolicyList(*policy)),
		))
		fetchedMigrationPolicy, err := client.MigrationPolicy(k8sv1.NamespaceDefault).List(&k8smetav1.ListOptions{})

		Expect(err).ToNot(HaveOccurred())
		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(fetchedMigrationPolicy.Items).To(HaveLen(1))
		Expect(fetchedMigrationPolicy.Items[0]).To(Equal(*policy))
	})

	It("should create a MigrationPolicy", func() {
		policy := NewMinimalMigrationPolicy(testPolicyName, k8sv1.NamespaceDefault)
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusCreated, policy),
		))
		createdMigrationPolicy, err := client.MigrationPolicy(k8sv1.NamespaceDefault).Create(policy)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(createdMigrationPolicy).To(Equal(policy))
	})

	It("should update a MigrationPolicy", func() {
		policy := NewMinimalMigrationPolicy(testPolicyName, k8sv1.NamespaceDefault)
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", policyPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, policy),
		))
		updatedMigrationPolicy, err := client.MigrationPolicy(k8sv1.NamespaceDefault).Update(policy)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedMigrationPolicy).To(Equal(policy))
	})

	It("should delete a MigrationPolicy", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("DELETE", policyPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err := client.MigrationPolicy(k8sv1.NamespaceDefault).Delete(testPolicyName, &k8smetav1.DeleteOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})
})
