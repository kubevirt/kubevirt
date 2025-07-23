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
	"crypto/tls"
	"net"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/certificates"
	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("MigrationProxy", func() {
	var tlsConfig *tls.Config
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "migrationproxytest")
		Expect(err).ToNot(HaveOccurred())
		os.MkdirAll(tmpDir, 0755)
		store, err := certificates.GenerateSelfSignedCert(tmpDir, "test", "test")

		ephemeraldiskutils.MockDefaultOwnershipManager()
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
			GetCertificate: func(info *tls.ClientHelloInfo) (certificate *tls.Certificate, e error) {
				return store.Current()
			},
		}
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Describe("migration proxy", func() {
		Context("verify proxy connections work", func() {
			It("by verifying source proxy works", func() {
				sourceSock := filepath.Join(tmpDir, "source-sock")

				listener, err := tls.Listen("tcp", "127.0.0.1:12345", tlsConfig)
				Expect(err).ShouldNot(HaveOccurred())

				defer listener.Close()

				sourceProxy := NewSourceProxy(sourceSock, "127.0.0.1:12345", tlsConfig, tlsConfig, "123")
				defer sourceProxy.Stop()

				err = sourceProxy.Start()
				Expect(err).ShouldNot(HaveOccurred())

				numBytes := make(chan int)
				go func() {
					var bytes [1024]byte
					fd, err := listener.Accept()
					Expect(err).ShouldNot(HaveOccurred())
					n, err := fd.Read(bytes[0:])
					if err != nil {
						Expect(err).ShouldNot(HaveOccurred())
					} else {
						numBytes <- n
					}
				}()

				conn, err := net.Dial("unix", sourceSock)
				Expect(err).ShouldNot(HaveOccurred())

				message := "some message"
				messageBytes := []byte(message)
				sentLen, err := conn.Write(messageBytes)
				Expect(err).ShouldNot(HaveOccurred())

				Expect(sentLen).To(Equal(len(messageBytes)))

				num := <-numBytes
				Expect(num).To(Equal(sentLen))
			})

			It("by creating both ends and sending a message", func() {
				sourceSock := filepath.Join(tmpDir, "source-sock")
				virtqemudSock := filepath.Join(tmpDir, "virtqemud-sock")
				virtqemudListener, err := net.Listen("unix", virtqemudSock)

				Expect(err).ShouldNot(HaveOccurred())

				defer virtqemudListener.Close()

				targetProxy := NewTargetProxy("0.0.0.0", 12345, tlsConfig, virtqemudSock, "123")
				sourceProxy := NewSourceProxy(sourceSock, "127.0.0.1:12345", tlsConfig, tlsConfig, "123")
				defer targetProxy.Stop()
				defer sourceProxy.Stop()

				err = targetProxy.Start()
				Expect(err).ShouldNot(HaveOccurred())
				err = sourceProxy.Start()
				Expect(err).ShouldNot(HaveOccurred())

				numBytes := make(chan int)
				go func() {
					fd, err := virtqemudListener.Accept()
					Expect(err).ShouldNot(HaveOccurred())

					var bytes [1024]byte
					n, err := fd.Read(bytes[0:])
					Expect(err).ShouldNot(HaveOccurred())
					numBytes <- n
				}()

				conn, err := net.Dial("unix", sourceSock)
				Expect(err).ShouldNot(HaveOccurred())

				message := "some message"
				messageBytes := []byte(message)
				sentLen, err := conn.Write(messageBytes)
				Expect(err).ShouldNot(HaveOccurred())

				Expect(sentLen).To(Equal(len(messageBytes)))

				num := <-numBytes
				Expect(num).To(Equal(sentLen))
			})

			DescribeTable("by creating both ends with a manager and sending a message", func(migrationConfig *v1.MigrationConfiguration) {
				directMigrationPort := "49152"
				virtqemudSock := filepath.Join(tmpDir, "virtqemud-sock")
				virtqemudListener, err := net.Listen("unix", virtqemudSock)
				Expect(err).ShouldNot(HaveOccurred())
				directSock := filepath.Join(tmpDir, "mykey-"+directMigrationPort)
				directListener, err := net.Listen("unix", directSock)

				Expect(err).ShouldNot(HaveOccurred())

				config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
					MigrationConfiguration: migrationConfig,
				})
				manager := NewMigrationProxyManager(tlsConfig, tlsConfig, config)
				manager.StartTargetListener("mykey", []string{virtqemudSock, directSock})
				destSrcPortMap := manager.GetTargetListenerPorts("mykey")
				manager.StartSourceListener("mykey", "127.0.0.1", destSrcPortMap, tmpDir)

				defer manager.StopTargetListener("myKey")
				defer manager.StopSourceListener("myKey")

				libvirtChan := make(chan int)
				directChan := make(chan int)

				msgReader := func(listener net.Listener, numBytes chan int) {
					fd, err := listener.Accept()
					Expect(err).ShouldNot(HaveOccurred())

					var bytes [1024]byte
					n, err := fd.Read(bytes[0:])
					Expect(err).ShouldNot(HaveOccurred())
					numBytes <- n
				}

				msgWriter := func(sockFile string, numBytes chan int, message string) {
					conn, err := net.Dial("unix", sockFile)
					Expect(err).ShouldNot(HaveOccurred())

					messageBytes := []byte(message)
					sentLen, err := conn.Write(messageBytes)
					Expect(err).ShouldNot(HaveOccurred())

					Expect(sentLen).To(Equal(len(messageBytes)))

					num := <-numBytes
					Expect(num).To(Equal(sentLen))
				}

				go msgReader(virtqemudListener, libvirtChan)
				go msgReader(directListener, directChan)

				for _, sockFile := range manager.GetSourceListenerFiles("mykey") {
					if strings.Contains(sockFile, directMigrationPort) {
						msgWriter(sockFile, directChan, "some direct message")
					} else {
						msgWriter(sockFile, libvirtChan, "some libvirt message")
					}
				}
			},
				Entry("with TLS enabled", &v1.MigrationConfiguration{DisableTLS: pointer.P(false)}),
				Entry("with TLS disabled", &v1.MigrationConfiguration{DisableTLS: pointer.P(true)}),
			)

			DescribeTable("by ensuring no new listeners can be created after shutdown", func(migrationConfig *v1.MigrationConfiguration) {

				key1 := "key1"
				key2 := "key2"

				directMigrationPort := "49152"
				virtqemudSock := filepath.Join(tmpDir, "virtqemud-sock")
				virtqemudListener, err := net.Listen("unix", virtqemudSock)
				Expect(err).ShouldNot(HaveOccurred())
				defer virtqemudListener.Close()

				directSock := filepath.Join(tmpDir, key1+"-"+directMigrationPort)
				directListener, err := net.Listen("unix", directSock)
				Expect(err).ShouldNot(HaveOccurred())
				defer directListener.Close()

				config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
					MigrationConfiguration: migrationConfig,
				})
				manager := NewMigrationProxyManager(tlsConfig, tlsConfig, config)
				err = manager.StartTargetListener(key1, []string{virtqemudSock, directSock})
				Expect(err).ShouldNot(HaveOccurred())
				destSrcPortMap := manager.GetTargetListenerPorts(key1)
				err = manager.StartSourceListener(key1, "127.0.0.1", destSrcPortMap, tmpDir)
				Expect(err).ShouldNot(HaveOccurred())

				defer manager.StopTargetListener(key1)
				defer manager.StopSourceListener(key1)

				// now mark manager for shutdown
				manager.InitiateGracefulShutdown()
				count := manager.OpenListenerCount()
				Expect(count).To(Equal(2))

				err = manager.StartTargetListener(key2, []string{virtqemudSock, directSock})
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(Equal("unable to process new migration connections during virt-handler shutdown"))

				err = manager.StartSourceListener(key2, "127.0.0.1", destSrcPortMap, tmpDir)
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(Equal("unable to process new migration connections during virt-handler shutdown"))

			},
				Entry("with TLS enabled", &v1.MigrationConfiguration{DisableTLS: pointer.P(false)}),
				Entry("with TLS disabled", &v1.MigrationConfiguration{DisableTLS: pointer.P(true)}),
			)
		})
	})
})
