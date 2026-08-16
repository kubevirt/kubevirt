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
 * Copyright The KubeVirt Authors
 *
 */

package subresources

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/compute"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	// virt-api is deployed with --secure-port=8443
	virtAPISecurePort = "8443"

	authPortForwardTimeout = 10 * time.Second
)

// These tests talk to a virt-api pod directly through a port-forward instead of
// going through the kube-apiserver aggregator. A request routed by the
// aggregator is authenticated and authorized by the kube-apiserver first, so it
// would tell us little about virt-api's own authentication and authorization.
// Connecting to the pod is what makes virt-api the component under test.
var _ = Describe(compute.SIG("virt-api authentication and authorization", func() {
	var baseURL string

	BeforeEach(func() {
		pod, err := libpod.GetRunningPodByLabel("virt-api", v1.AppLabel, flags.KubeVirtInstallNamespace, "")
		Expect(err).ToNot(HaveOccurred())

		stop := make(chan struct{})
		DeferCleanup(func() {
			close(stop)
		})

		localPort, err := libpod.ForwardPorts(
			pod, []string{"0:" + virtAPISecurePort}, stop, authPortForwardTimeout,
		)
		Expect(err).ToNot(HaveOccurred())

		baseURL = fmt.Sprintf("https://localhost:%d", localPort)
	})

	Context("without credentials", func() {
		// An anonymous request belongs to system:unauthenticated, so none of the
		// ClusterRoles KubeVirt ships apply to it. These endpoints can therefore
		// only be served if virt-api exempts them from authorization, which is
		// what it did by exact URL match before moving to the aggregated API
		// server and must keep doing.
		DescribeTable("should serve the always-allow cluster level endpoint", func(resource string) {
			for _, gv := range v1.SubresourceGroupVersions {
				url := baseURL + groupVersionRoot(gv) + "/" + resource
				Expect(getStatusFromVirtAPI(http.MethodGet, url, "")).To(Equal(http.StatusOK), url)
			}
		},
			Entry("version", "version"),
			Entry("guestfs", "guestfs"),
			Entry("healthz", "healthz"),
		)

		It("should serve the process level healthz endpoint", func() {
			Expect(getStatusFromVirtAPI(http.MethodGet, baseURL+"/healthz", "")).To(Equal(http.StatusOK))
		})

		It("should not serve a named subresource", func() {
			// Rejected by authentication or by authorization depending on whether
			// virt-api accepts unauthenticated requests as anonymous, so accept
			// either. What matters is that the subresource is not served.
			Expect(getStatusFromVirtAPI(http.MethodPut, startSubresourceURL(baseURL), "")).
				To(BeElementOf(http.StatusUnauthorized, http.StatusForbidden))
		})
	})

	Context("with a service account token that has no subresource permissions", func() {
		var token string

		BeforeEach(func() {
			token = readServiceAccountToken(testsuite.SubresourceUnprivilegedServiceAccountName)
		})

		It("should not serve a named subresource", func() {
			Expect(getStatusFromVirtAPI(http.MethodPut, startSubresourceURL(baseURL), token)).
				To(BeElementOf(http.StatusUnauthorized, http.StatusForbidden))
		})

		It("should serve the version endpoint", func() {
			url := baseURL + groupVersionRoot(v1.SubresourceGroupVersions[0]) + "/version"
			Expect(getStatusFromVirtAPI(http.MethodGet, url, token)).To(Equal(http.StatusOK))
		})
	})

	Context("with an invalid token", func() {
		// Presenting a token that cannot be reviewed is an authentication
		// failure, so it must be reported as such instead of falling back to an
		// anonymous identity.
		It("should reject a named subresource as unauthorized", func() {
			Expect(getStatusFromVirtAPI(http.MethodPut, startSubresourceURL(baseURL), "not-a-valid-token")).
				To(Equal(http.StatusUnauthorized))
		})
	})
}))

func groupVersionRoot(gv schema.GroupVersion) string {
	return "/apis/" + gv.Group + "/" + gv.Version
}

// startSubresourceURL addresses a subresource that is never exempt from
// authorization. The virtual machine does not need to exist, because
// authorization is decided before the request reaches the storage.
func startSubresourceURL(baseURL string) string {
	return fmt.Sprintf("%s%s/namespaces/%s/virtualmachines/%s/start",
		baseURL,
		groupVersionRoot(v1.SubresourceGroupVersions[0]),
		testsuite.GetTestNamespace(nil),
		"authz-probe-does-not-exist",
	)
}

func readServiceAccountToken(saName string) string {
	secret, err := kubevirt.Client().CoreV1().
		Secrets(testsuite.GetTestNamespace(nil)).
		Get(context.Background(), saName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	token, ok := secret.Data["token"]
	Expect(ok).To(BeTrue())

	return string(token)
}

func getStatusFromVirtAPI(method, url, token string) int {
	client := &http.Client{
		Transport: &http.Transport{
			//nolint:gosec
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequestWithContext(context.Background(), method, url, http.NoBody)
	Expect(err).ToNot(HaveOccurred())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	Expect(err).ToNot(HaveOccurred())
	defer func() {
		Expect(resp.Body.Close()).To(Succeed())
	}()

	return resp.StatusCode
}
