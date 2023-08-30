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

package kubecli

import (
	"net/http"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Kubevirt Migration Client", func() {
	var server *ghttp.Server
	basePath := "/apis/kubevirt.io/v1/namespaces/default/virtualmachineinstancemigrations"
	migrationPath := path.Join(basePath, "testmigration")
	proxyPath := "/proxy/path"

	BeforeEach(func() {
		server = ghttp.NewServer()
	})

	DescribeTable("should fetch a MigrationMigration", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		migration := NewMinimalMigration("testmigration")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, migrationPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, migration),
		))
		fetchedMigration, err := client.VirtualMachineInstanceMigration(k8sv1.NamespaceDefault).Get("testmigration", &k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedMigration).To(Equal(migration))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should detect non existent Migrations", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, migrationPath)),
			ghttp.RespondWithJSONEncoded(http.StatusNotFound, errors.NewNotFound(schema.GroupResource{}, "testmigration")),
		))
		_, err = client.VirtualMachineInstanceMigration(k8sv1.NamespaceDefault).Get("testmigration", &k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).To(HaveOccurred())
		Expect(errors.IsNotFound(err)).To(BeTrue())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should fetch a Migration list", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		migration := NewMinimalMigration("testmigration")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, basePath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, NewMigrationList(*migration)),
		))
		fetchedMigrationList, err := client.VirtualMachineInstanceMigration(k8sv1.NamespaceDefault).List(&k8smetav1.ListOptions{})
		apiVersion, kind := v1.VirtualMachineInstanceMigrationGroupVersionKind.ToAPIVersionAndKind()

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedMigrationList.Items).To(HaveLen(1))
		Expect(fetchedMigrationList.Items[0].APIVersion).To(Equal(apiVersion))
		Expect(fetchedMigrationList.Items[0].Kind).To(Equal(kind))
		Expect(fetchedMigrationList.Items[0]).To(Equal(*migration))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should create a Migration", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		migration := NewMinimalMigration("testmigration")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", path.Join(proxyPath, basePath)),
			ghttp.RespondWithJSONEncoded(http.StatusCreated, migration),
		))
		createdMigration, err := client.VirtualMachineInstanceMigration(k8sv1.NamespaceDefault).Create(migration, &k8smetav1.CreateOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(createdMigration).To(Equal(migration))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should update a Migration", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		migration := NewMinimalMigration("testmigration")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", path.Join(proxyPath, migrationPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, migration),
		))
		updatedMigration, err := client.VirtualMachineInstanceMigration(k8sv1.NamespaceDefault).Update(migration)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedMigration).To(Equal(migration))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should patch a Migration", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		migration := NewMinimalMigration("testmigration")
		migration.Spec.VMIName = "somethingelse"

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PATCH", path.Join(proxyPath, migrationPath)),
			ghttp.VerifyBody([]byte("{\"spec\":{\"vmiName\":something}}")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, migration),
		))

		_, err = client.VirtualMachineInstanceMigration(k8sv1.NamespaceDefault).Patch(migration.Name, types.MergePatchType,
			[]byte("{\"spec\":{\"vmiName\":something}}"))

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should delete a Migration", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("DELETE", path.Join(proxyPath, migrationPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err = client.VirtualMachineInstanceMigration(k8sv1.NamespaceDefault).Delete("testmigration", &k8smetav1.DeleteOptions{})

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
