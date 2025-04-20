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
 */

package clientconfig_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/spf13/pflag"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
)

var _ = Describe("client", func() {
	It("NewContext should store a clientConfig and make it retrievable with FromContext", func() {
		By("creating a clientConfig")
		clientConfig := kubecli.DefaultClientConfig(&pflag.FlagSet{})

		By("creating a client from the clientConfig")
		client, err := kubecli.GetKubevirtClientFromClientConfig(clientConfig)
		Expect(err).ToNot(HaveOccurred())

		By("creating a client from context")
		contextClient, _, _, err := clientconfig.ClientAndNamespaceFromContext(
			clientconfig.NewContext(context.Background(), clientConfig),
		)
		Expect(err).ToNot(HaveOccurred())

		By("verifying the clients have the same config")
		Expect(contextClient.Config()).To(Equal(client.Config()))
	})

	It("ClientAndNamespaceFromContext should fail when clientConfig is missing from context", func() {
		_, _, _, err := clientconfig.ClientAndNamespaceFromContext(context.Background())
		Expect(err).To(MatchError("unable to get client config from context"))
	})
})
