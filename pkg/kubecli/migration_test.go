/*
 * This file is part of the kubevirt project
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

package kubecli

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("Kubevirt Migartion Client", func() {

	var server *ghttp.Server
	var client KubevirtClient
	basePath := "/apis/kubevirt.io/v1alpha1/namespaces/default/migrations"
	migrationPath := basePath + "/testmigration"

	BeforeEach(func() {
		var err error
		server = ghttp.NewServer()
		client, err = GetKubevirtClientFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should fetch a Migration", func() {
		migration := v1.NewMinimalMigration("testmigration", "testmigration")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", migrationPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, migration),
		))
		fetchedMigration, err := client.Migration(k8smetav1.NamespaceDefault).Get("testmigration", k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedMigration).To(Equal(migration))
	})

	It("should detect non existent Migrations", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", migrationPath),
			ghttp.RespondWithJSONEncoded(http.StatusNotFound, errors.NewNotFound(schema.GroupResource{}, "testmigration")),
		))
		_, err := client.Migration(k8smetav1.NamespaceDefault).Get("testmigration", k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).To(HaveOccurred())
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})

	It("should fetch a Migration list", func() {
		migration := v1.NewMinimalMigration("testmigration", "testmigration")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, NewMigrationList(*migration)),
		))
		fetchedMigrationList, err := client.Migration(k8smetav1.NamespaceDefault).List(k8smetav1.ListOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedMigrationList.Items).To(HaveLen(1))
		Expect(fetchedMigrationList.Items[0]).To(Equal(*migration))
	})

	It("should create a Migration", func() {
		migration := v1.NewMinimalMigration("testmigration", "testmigration")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusCreated, migration),
		))
		createdMigration, err := client.Migration(k8smetav1.NamespaceDefault).Create(migration)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(createdMigration).To(Equal(migration))
	})

	It("should update a Migration", func() {
		migration := v1.NewMinimalMigration("testmigration", "testmigration")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, migration),
		))
		updatedMigration, err := client.Migration(k8smetav1.NamespaceDefault).Update(migration)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedMigration).To(Equal(migration))
	})

	It("should delete a Migration", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("DELETE", migrationPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err := client.Migration(k8smetav1.NamespaceDefault).Delete("testmigration", &k8smetav1.DeleteOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})
})

func NewMigrationList(migrations ...v1.Migration) *v1.MigrationList {
	return &v1.MigrationList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "MigrationList"}, Items: migrations}
}
