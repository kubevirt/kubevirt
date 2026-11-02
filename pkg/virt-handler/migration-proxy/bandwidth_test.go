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

package migrationproxy

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/time/rate"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "kubevirt.io/api/core/v1"

	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("MigrationProxy bandwidth limiter", func() {
	DescribeTable("converts bandwidth quantities to bytes per second",
		func(quantity string, expectedBytesPerSecond int64) {
			Expect(bandwidthBytesPerSecond(resource.MustParse(quantity))).To(Equal(expectedBytesPerSecond))
		},
		Entry("zero", "0", int64(0)),
		Entry("one byte rounds up to one MiB", "1", int64(1024*1024)),
		Entry("decimal SI is rounded consistently with libvirt migration bandwidth", "100M", int64(95*1024*1024)),
		Entry("binary SI is converted directly", "2Mi", int64(2*1024*1024)),
		Entry("negative quantities disable limiter creation", "-1Mi", int64(0)),
	)

	It("creates limiters using bytes per second and caps burst at the copy buffer size", func() {
		limiter := newBandwidthLimiter(resource.MustParse("64Mi"))

		Expect(limiter).ToNot(BeNil())
		Expect(limiter.Limit()).To(Equal(rate.Limit(64 * 1024 * 1024)))
		Expect(limiter.Burst()).To(Equal(copyBufferSize))
		Expect(newBandwidthLimiter(resource.MustParse("0"))).To(BeNil())
	})

	It("limits only the direct migration source proxy", func() {
		ephemeraldiskutils.MockDefaultOwnershipManager()
		tmpDir, err := os.MkdirTemp("", "migrationproxy-bandwidth")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(tmpDir)

		Expect(os.MkdirAll(filepath.Join(tmpDir, "run/kubevirt"), 0755)).To(Succeed())
		mountRoot, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir)
		Expect(err).ToNot(HaveOccurred())

		config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
		manager := NewMigrationProxyManager(nil, nil, config)
		defer manager.StopSourceListener("vmi")

		ports := map[string]int{
			"12345": LibvirtDirectMigrationPort,
			"12346": LibvirtBlockMigrationPort,
			"12347": 0,
		}
		Expect(manager.StartSourceListener("vmi", "127.0.0.1", ports, mountRoot, resource.MustParse("64Mi"))).To(Succeed())

		proxies := manager.sourceProxies["vmi"]
		Expect(proxies).To(HaveLen(3))
		var directProxy *migrationProxy
		for _, proxy := range proxies {
			if strings.Contains(proxy.unixSocketPath, "-49152-source.sock") {
				directProxy = proxy
				continue
			}
			Expect(proxy.limiter).To(BeNil())
			Expect(proxy.bandwidthBytes).To(Equal(int64(64 * 1024 * 1024)))
		}
		Expect(directProxy).ToNot(BeNil())
		Expect(directProxy.limiter).ToNot(BeNil())
		Expect(directProxy.limiter.Limit()).To(Equal(rate.Limit(64 * 1024 * 1024)))
		Expect(directProxy.bandwidthBytes).To(Equal(int64(64 * 1024 * 1024)))
	})

	It("returns context errors before writing partially read bytes", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		src := strings.NewReader("aa")
		dst := &strings.Builder{}
		limiter := rate.NewLimiter(rate.Limit(1), 1)

		n, err := copyRateLimited(ctx, dst, src, limiter)

		Expect(err).To(MatchError(context.Canceled))
		Expect(n).To(Equal(int64(0)))
		Expect(dst.String()).To(BeEmpty())
	})

	It("throttles copies with the configured limiter", func() {
		payload := bytes.Repeat([]byte("abcd"), 512*1024) // 2 MiB payload
		src := bytes.NewReader(payload)
		dst := &bytes.Buffer{}
		limiter := newBandwidthLimiter(resource.MustParse("1Mi"))

		started := time.Now()
		n, err := copyRateLimited(context.Background(), dst, src, limiter)

		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(int64(len(payload))))
		Expect(dst.Bytes()).To(Equal(payload))
		Expect(time.Since(started)).To(BeNumerically("~", 2*time.Second, 250*time.Millisecond))
	})

	It("stops limiter waits when the proxy stops", func() {
		proxy := &migrationProxy{stopChan: make(chan struct{})}
		ctx, cancel := proxy.connectionContext()
		defer cancel()

		limiter := rate.NewLimiter(rate.Limit(1), 1)
		Expect(limiter.Allow()).To(BeTrue())
		result := make(chan error, 1)
		go func() {
			result <- limiter.WaitN(ctx, 1)
		}()

		close(proxy.stopChan)
		Eventually(result, time.Second).Should(Receive(MatchError(context.Canceled)))
	})
})
