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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/certificates"
	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/safepath"
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
		Context("source proxy", func() {
			const vmiUID = "my-vmi-uid"

			It("listens under /run/kubevirt and forwards traffic to TCP", func() {
				prepareMigrationProxyDirUnderMountRoot(tmpDir)

				sourceRelativePath := SourceUnixFile(VirtLauncherKubevirtRunDir, ConstructProxyKey(vmiUID, LibvirtDirectMigrationPort))
				sourceSock := hostSocketPath(tmpDir, sourceRelativePath)

				mountRoot, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir)
				Expect(err).ShouldNot(HaveOccurred())

				tcpListener, err := tls.Listen("tcp", "127.0.0.1:12345", tlsConfig)
				Expect(err).ShouldNot(HaveOccurred())
				defer tcpListener.Close()

				sourceProxy := NewSourceProxy(mountRoot, sourceRelativePath, "127.0.0.1:12345", tlsConfig, vmiUID)
				defer sourceProxy.Stop()

				Expect(sourceProxy.Start()).Should(Succeed())

				numBytes := make(chan int, 1)
				go func() {
					defer GinkgoRecover()
					fd, err := tcpListener.Accept()
					Expect(err).ShouldNot(HaveOccurred())
					defer fd.Close()

					var bytes [1024]byte
					n, err := fd.Read(bytes[:])
					Expect(err).ShouldNot(HaveOccurred())
					numBytes <- n
				}()

				conn, err := net.Dial("unix", sourceSock)
				Expect(err).ShouldNot(HaveOccurred())
				defer conn.Close()

				message := "some message"
				sentLen, err := conn.Write([]byte(message))
				Expect(err).ShouldNot(HaveOccurred())
				Expect(sentLen).To(Equal(len(message)))

				Expect(<-numBytes).To(Equal(sentLen))
			})

			It("is reachable through /var/run when the pod uses a ../run symlink", func() {
				podRoot := filepath.Join(tmpDir, "podroot")
				prepareMigrationProxyDirUnderMountRoot(podRoot)
				Expect(os.MkdirAll(filepath.Join(podRoot, "var"), 0755)).To(Succeed())
				Expect(os.Symlink("../run", filepath.Join(podRoot, "var", "run"))).To(Succeed())

				mountRoot, err := safepath.JoinAndResolveWithRelativeRoot(podRoot)
				Expect(err).NotTo(HaveOccurred())

				tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
				Expect(err).NotTo(HaveOccurred())
				defer tcpListener.Close()

				host, port, err := net.SplitHostPort(tcpListener.Addr().String())
				Expect(err).NotTo(HaveOccurred())

				config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
					MigrationConfiguration: &v1.MigrationConfiguration{DisableTLS: pointer.P(true)},
				})
				manager := NewMigrationProxyManager(nil, nil, config)
				err = manager.StartSourceListener(vmiUID, mountRoot, host, map[string]int{port: LibvirtDirectMigrationPort})
				Expect(err).NotTo(HaveOccurred())
				defer manager.StopSourceListener(vmiUID)

				tcpAccepted := make(chan struct{})
				go func() {
					defer GinkgoRecover()
					serverConn, acceptErr := tcpListener.Accept()
					Expect(acceptErr).NotTo(HaveOccurred())
					defer serverConn.Close()
					close(tcpAccepted)
				}()

				qemuSock := qemuSocketPath(
					podRoot,
					SourceUnixFile(VirtLauncherKubevirtRunDir, ConstructProxyKey(vmiUID, LibvirtDirectMigrationPort)),
				)
				conn, err := net.Dial("unix", qemuSock)
				Expect(err).NotTo(HaveOccurred())
				defer conn.Close()

				_, err = conn.Write([]byte("ping"))
				Expect(err).NotTo(HaveOccurred())

				Eventually(tcpAccepted, 5*time.Second).Should(BeClosed())
			})

			It("fails to start when the migrationproxy directory is missing", func() {
				mountRoot, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir)
				Expect(err).ShouldNot(HaveOccurred())

				sourceRelativePath := SourceUnixFile(VirtLauncherKubevirtRunDir, ConstructProxyKey(vmiUID, LibvirtDirectMigrationPort))
				sourceProxy := NewSourceProxy(mountRoot, sourceRelativePath, "127.0.0.1:12345", tlsConfig, vmiUID)
				defer sourceProxy.Stop()

				Expect(sourceProxy.Start()).Should(HaveOccurred())
			})

			It("removes the unix socket when the manager stops the source listener", func() {
				prepareMigrationProxyDirUnderMountRoot(tmpDir)

				sourceRelativePath := SourceUnixFile(VirtLauncherKubevirtRunDir, ConstructProxyKey(vmiUID, LibvirtDirectMigrationPort))
				sourceSock := hostSocketPath(tmpDir, sourceRelativePath)

				mountRoot, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir)
				Expect(err).ShouldNot(HaveOccurred())

				tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
				Expect(err).NotTo(HaveOccurred())
				defer tcpListener.Close()

				host, port, err := net.SplitHostPort(tcpListener.Addr().String())
				Expect(err).NotTo(HaveOccurred())

				config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
					MigrationConfiguration: &v1.MigrationConfiguration{DisableTLS: pointer.P(true)},
				})
				manager := NewMigrationProxyManager(nil, nil, config)
				err = manager.StartSourceListener(vmiUID, mountRoot, host, map[string]int{port: LibvirtDirectMigrationPort})
				Expect(err).NotTo(HaveOccurred())

				Expect(sourceSock).To(BeAnExistingFile())

				manager.StopSourceListener(vmiUID)

				Eventually(sourceSock, 5*time.Second).ShouldNot(BeAnExistingFile())
			})

			DescribeTable("manager forwards traffic through source and target proxies", func(migrationConfig *v1.MigrationConfiguration) {
				const (
					virtqemudRelativePath = "/virtqemud-sock"
					directRelativePath    = "/mykey-49152"
				)

				directMigrationPort := "49152"
				virtqemudSock := filepath.Join(tmpDir, "virtqemud-sock")
				virtqemudListener, err := net.Listen("unix", virtqemudSock)
				Expect(err).ShouldNot(HaveOccurred())
				directSock := filepath.Join(tmpDir, "mykey-"+directMigrationPort)
				directListener, err := net.Listen("unix", directSock)
				Expect(err).ShouldNot(HaveOccurred())

				mountRoot, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir)
				Expect(err).ShouldNot(HaveOccurred())
				prepareMigrationProxyDirUnderMountRoot(tmpDir)

				config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
					MigrationConfiguration: migrationConfig,
				})
				manager := NewMigrationProxyManager(tlsConfig, tlsConfig, config)
				err = manager.StartTargetListener(
					"mykey",
					mountRoot,
					[]string{virtqemudRelativePath, directRelativePath})
				Expect(err).ShouldNot(HaveOccurred())
				destSrcPortMap := manager.GetTargetListenerPorts("mykey")
				Expect(manager.StartSourceListener("mykey", mountRoot, "127.0.0.1", destSrcPortMap)).Should(Succeed())

				defer manager.StopTargetListener("mykey")
				defer manager.StopSourceListener("mykey")

				sourcePaths := manager.GetSourceListenerFiles("mykey")
				Expect(sourcePaths).To(ConsistOf(
					SourceUnixFile(VirtLauncherKubevirtRunDir, ConstructProxyKey("mykey", 0)),
					SourceUnixFile(VirtLauncherKubevirtRunDir, ConstructProxyKey("mykey", LibvirtDirectMigrationPort)),
				))

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
					defer conn.Close()

					messageBytes := []byte(message)
					sentLen, err := conn.Write(messageBytes)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(sentLen).To(Equal(len(messageBytes)))

					num := <-numBytes
					Expect(num).To(Equal(sentLen))
				}

				go msgReader(virtqemudListener, libvirtChan)
				go msgReader(directListener, directChan)

				for _, relativeSockFile := range sourcePaths {
					sockFile := hostSocketPath(tmpDir, relativeSockFile)
					if strings.Contains(relativeSockFile, ConstructProxyKey("mykey", LibvirtDirectMigrationPort)) {
						msgWriter(sockFile, directChan, "some direct message")
					} else {
						msgWriter(sockFile, libvirtChan, "some libvirt message")
					}
				}
			},
				Entry("with TLS enabled", &v1.MigrationConfiguration{DisableTLS: pointer.P(false)}),
				Entry("with TLS disabled", &v1.MigrationConfiguration{DisableTLS: pointer.P(true)}),
			)
		})

		Context("verify proxy connections work", func() {
			It("by creating both ends and sending a message", func() {
				const (
					virtqemudRelativePath = "/virtqemud-sock"
				)

				prepareMigrationProxyDirUnderMountRoot(tmpDir)
				sourceRelativePath := SourceUnixFile(VirtLauncherKubevirtRunDir, ConstructProxyKey("123", LibvirtDirectMigrationPort))
				sourceSock := hostSocketPath(tmpDir, sourceRelativePath)
				virtqemudSock := filepath.Join(tmpDir, "virtqemud-sock")
				virtqemudListener, err := net.Listen("unix", virtqemudSock)
				Expect(err).ShouldNot(HaveOccurred())

				mountRoot, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir)
				Expect(err).ShouldNot(HaveOccurred())

				defer virtqemudListener.Close()

				targetProxy := NewTargetProxy(
					"0.0.0.0",
					12345,
					tlsConfig,
					mountRoot,
					virtqemudRelativePath,
					"123")
				sourceProxy := NewSourceProxy(mountRoot, sourceRelativePath, "127.0.0.1:12345", tlsConfig, "123")
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

			DescribeTable("by ensuring no new listeners can be created after shutdown", func(migrationConfig *v1.MigrationConfiguration) {
				const (
					virtqemudRelativePath = "/virtqemud-sock"
					directRelativePath    = "/key1-49152"
				)

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

				mountRoot, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir)
				Expect(err).ShouldNot(HaveOccurred())
				prepareMigrationProxyDirUnderMountRoot(tmpDir)

				config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
					MigrationConfiguration: migrationConfig,
				})
				manager := NewMigrationProxyManager(tlsConfig, tlsConfig, config)
				err = manager.StartTargetListener(
					key1,
					mountRoot,
					[]string{virtqemudRelativePath, directRelativePath})
				Expect(err).ShouldNot(HaveOccurred())
				destSrcPortMap := manager.GetTargetListenerPorts(key1)
				err = manager.StartSourceListener(key1, mountRoot, "127.0.0.1", destSrcPortMap)
				Expect(err).ShouldNot(HaveOccurred())

				defer manager.StopTargetListener(key1)
				defer manager.StopSourceListener(key1)

				// now mark manager for shutdown
				manager.InitiateGracefulShutdown()
				count := manager.OpenListenerCount()
				Expect(count).To(Equal(2))

				err = manager.StartTargetListener(
					key2,
					mountRoot,
					[]string{virtqemudRelativePath, directRelativePath})
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(Equal("unable to process new migration connections during virt-handler shutdown"))

				err = manager.StartSourceListener(key2, mountRoot, "127.0.0.1", destSrcPortMap)
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(Equal("unable to process new migration connections during virt-handler shutdown"))

			},
				Entry("with TLS enabled", &v1.MigrationConfiguration{DisableTLS: pointer.P(false)}),
				Entry("with TLS disabled", &v1.MigrationConfiguration{DisableTLS: pointer.P(true)}),
			)
		})

		Context("handleConnection unix target", func() {
			runHandleConnection := func(proxy *migrationProxy) (net.Conn, <-chan struct{}) {
				clientConn, serverConn := net.Pipe()
				done := make(chan struct{})
				go func() {
					proxy.handleConnection(serverConn)
					close(done)
				}()
				return clientConn, done
			}

			expectNoBackendConnection := func(listener net.Listener) {
				unixListener := listener.(*net.UnixListener)
				Expect(unixListener.SetDeadline(time.Now().Add(100 * time.Millisecond))).To(Succeed())
				_, err := listener.Accept()
				Expect(err).To(HaveOccurred())
			}

			It("dials the outbound unix socket through safepath resolution", func() {
				const virtqemudRelativePath = "/virtqemud-sock"

				virtqemudSock := filepath.Join(tmpDir, "virtqemud-sock")
				virtqemudListener, err := net.Listen("unix", virtqemudSock)
				Expect(err).ShouldNot(HaveOccurred())
				defer virtqemudListener.Close()

				mountRoot, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir)
				Expect(err).ShouldNot(HaveOccurred())

				proxy := newTestUnixProxy(mountRoot, virtqemudRelativePath)
				clientConn, done := runHandleConnection(proxy)

				acceptedCh := make(chan net.Conn, 1)
				go func() {
					conn, acceptErr := virtqemudListener.Accept()
					Expect(acceptErr).ShouldNot(HaveOccurred())
					acceptedCh <- conn
				}()

				message := "safepath message"
				_, err = clientConn.Write([]byte(message))
				Expect(err).ShouldNot(HaveOccurred())

				accepted := <-acceptedCh
				buf := make([]byte, len(message))
				n, err := accepted.Read(buf)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(n).To(Equal(len(message)))
				Expect(string(buf)).To(Equal(message))

				Expect(accepted.Close()).To(Succeed())
				Expect(clientConn.Close()).To(Succeed())
				Eventually(done, 5*time.Second).Should(BeClosed())
			})

			It("dials a nested unix socket path through safepath resolution", func() {
				const virtqemudRelativePath = "/run/virtqemud/socket"

				nestedDir := filepath.Join(tmpDir, "run", "virtqemud")
				Expect(os.MkdirAll(nestedDir, 0755)).To(Succeed())
				virtqemudSock := filepath.Join(nestedDir, "socket")
				virtqemudListener, err := net.Listen("unix", virtqemudSock)
				Expect(err).ShouldNot(HaveOccurred())
				defer virtqemudListener.Close()

				mountRoot, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir)
				Expect(err).ShouldNot(HaveOccurred())

				proxy := newTestUnixProxy(mountRoot, virtqemudRelativePath)
				clientConn, done := runHandleConnection(proxy)

				acceptedCh := make(chan net.Conn, 1)
				go func() {
					conn, acceptErr := virtqemudListener.Accept()
					Expect(acceptErr).ShouldNot(HaveOccurred())
					acceptedCh <- conn
				}()

				message := "nested safepath message"
				_, err = clientConn.Write([]byte(message))
				Expect(err).ShouldNot(HaveOccurred())

				accepted := <-acceptedCh
				buf := make([]byte, len(message))
				n, err := accepted.Read(buf)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(n).To(Equal(len(message)))
				Expect(string(buf)).To(Equal(message))

				Expect(accepted.Close()).To(Succeed())
				Expect(clientConn.Close()).To(Succeed())
				Eventually(done, 5*time.Second).Should(BeClosed())
			})

			It("returns early when mount root is unavailable", func() {
				const virtqemudRelativePath = "/virtqemud-sock"

				virtqemudSock := filepath.Join(tmpDir, "virtqemud-sock")
				virtqemudListener, err := net.Listen("unix", virtqemudSock)
				Expect(err).ShouldNot(HaveOccurred())
				defer virtqemudListener.Close()

				proxy := newTestUnixProxy(nil, virtqemudRelativePath)

				clientConn, done := runHandleConnection(proxy)
				Eventually(done, 5*time.Second).Should(BeClosed())
				Expect(clientConn.Close()).To(Succeed())
				expectNoBackendConnection(virtqemudListener)
			})

			It("returns early when the relative path does not exist", func() {
				const missingRelativePath = "/missing.sock"

				mountRoot, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir)
				Expect(err).ShouldNot(HaveOccurred())

				proxy := newTestUnixProxy(mountRoot, missingRelativePath)
				clientConn, done := runHandleConnection(proxy)
				Eventually(done, 5*time.Second).Should(BeClosed())
				Expect(clientConn.Close()).To(Succeed())
			})

			It("returns early when the relative path is a symlink", func() {
				const escapeRelativePath = "/escape.sock"

				externalDir, err := os.MkdirTemp("", "migrationproxy-external")
				Expect(err).ShouldNot(HaveOccurred())
				defer os.RemoveAll(externalDir)

				externalSock := filepath.Join(externalDir, "socket")
				externalListener, err := net.Listen("unix", externalSock)
				Expect(err).ShouldNot(HaveOccurred())
				defer externalListener.Close()

				linkPath := filepath.Join(tmpDir, "escape.sock")
				Expect(os.Symlink(externalSock, linkPath)).To(Succeed())

				mountRoot, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir)
				Expect(err).ShouldNot(HaveOccurred())

				proxy := newTestUnixProxy(mountRoot, escapeRelativePath)
				clientConn, done := runHandleConnection(proxy)
				Eventually(done, 5*time.Second).Should(BeClosed())
				Expect(clientConn.Close()).To(Succeed())
				expectNoBackendConnection(externalListener)
			})
		})

		Context("handleConnection unsupported protocol", func() {
			It("returns early for an unsupported target protocol", func() {
				proxy := &migrationProxy{
					targetProtocol: "udp",
					stopChan:       make(chan struct{}),
					logger:         log.Log,
				}

				clientConn, serverConn := net.Pipe()
				done := make(chan struct{})
				go func() {
					proxy.handleConnection(serverConn)
					close(done)
				}()

				Eventually(done, 5*time.Second).Should(BeClosed())
				Expect(clientConn.Close()).To(Succeed())
			})
		})
	})
})

func newTestUnixProxy(mountRoot *safepath.Path, relativePath string) *migrationProxy {
	return &migrationProxy{
		targetProtocol: "unix",
		mountRoot:      mountRoot,
		relativePath:   relativePath,
		stopChan:       make(chan struct{}),
		logger:         log.Log,
	}
}

func hostSocketPath(mountRootDir, relativePath string) string {
	return filepath.Join(mountRootDir, strings.TrimPrefix(relativePath, "/"))
}

// qemuSocketPath returns the unix socket path QEMU uses inside the pod (/var/run/kubevirt/...).
func qemuSocketPath(podRoot, handlerPath string) string {
	return filepath.Join(podRoot, "var", "run", strings.TrimPrefix(handlerPath, "/run/"))
}

func prepareMigrationProxyDirUnderMountRoot(mountRootDir string) {
	relativeDir := strings.TrimPrefix(MigrationSocketsDir(VirtLauncherKubevirtRunDir), "/")
	Expect(os.MkdirAll(filepath.Join(mountRootDir, relativeDir), 0755)).To(Succeed())
}
