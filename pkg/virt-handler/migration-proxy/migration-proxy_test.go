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

package migrationproxy

import (
	"crypto/tls"
	"io/ioutil"
	"net"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/certificates"
)

var _ = Describe("MigrationProxy", func() {
	var tlsConfig *tls.Config
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = ioutil.TempDir("", "migrationproxytest")
		Expect(err).ToNot(HaveOccurred())
		os.MkdirAll(tmpDir, 0755)
		store, err := certificates.GenerateSelfSignedCert(tmpDir, "test", "test")

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
				sourceSock := tmpDir + "/source-sock"

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
				sourceSock := tmpDir + "/source-sock"
				libvirtdSock := tmpDir + "/libvirtd-sock"
				libvirtdListener, err := net.Listen("unix", libvirtdSock)

				Expect(err).ShouldNot(HaveOccurred())

				defer libvirtdListener.Close()

				targetProxy := NewTargetProxy("0.0.0.0", 12345, tlsConfig, tlsConfig, libvirtdSock, "123")
				sourceProxy := NewSourceProxy(sourceSock, "127.0.0.1:12345", tlsConfig, tlsConfig, "123")
				defer targetProxy.Stop()
				defer sourceProxy.Stop()

				err = targetProxy.Start()
				Expect(err).ShouldNot(HaveOccurred())
				err = sourceProxy.Start()
				Expect(err).ShouldNot(HaveOccurred())

				numBytes := make(chan int)
				go func() {
					fd, err := libvirtdListener.Accept()
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

			It("by creating both ends with a manager and sending a message", func() {
				directMigrationPort := "49152"
				libvirtdSock := tmpDir + "/libvirtd-sock"
				libvirtdListener, err := net.Listen("unix", libvirtdSock)
				Expect(err).ShouldNot(HaveOccurred())
				directSock := tmpDir + "/mykey-" + directMigrationPort
				directListener, err := net.Listen("unix", directSock)

				Expect(err).ShouldNot(HaveOccurred())

				manager := NewMigrationProxyManager(tlsConfig, tlsConfig)
				manager.StartTargetListener("mykey", []string{libvirtdSock, directSock})
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

				go msgReader(libvirtdListener, libvirtChan)
				go msgReader(directListener, directChan)

				for _, sockFile := range manager.GetSourceListenerFiles("mykey") {
					if strings.Contains(sockFile, directMigrationPort) {
						msgWriter(sockFile, directChan, "some direct message")
					} else {
						msgWriter(sockFile, libvirtChan, "some libvirt message")
					}
				}
			})

			It("by ensuring no new listeners can be created after shutdown", func() {

				key1 := "key1"
				key2 := "key2"

				directMigrationPort := "49152"
				libvirtdSock := tmpDir + "/libvirtd-sock"
				libvirtdListener, err := net.Listen("unix", libvirtdSock)
				defer libvirtdListener.Close()
				Expect(err).ShouldNot(HaveOccurred())
				directSock := tmpDir + "/" + key1 + "-" + directMigrationPort
				directListener, err := net.Listen("unix", directSock)
				defer directListener.Close()

				Expect(err).ShouldNot(HaveOccurred())

				manager := NewMigrationProxyManager(tlsConfig, tlsConfig)
				err = manager.StartTargetListener(key1, []string{libvirtdSock, directSock})
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

				err = manager.StartTargetListener(key2, []string{libvirtdSock, directSock})
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(Equal("unable to process new migration connections during virt-handler shutdown"))

				err = manager.StartSourceListener(key2, "127.0.0.1", destSrcPortMap, tmpDir)
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(Equal("unable to process new migration connections during virt-handler shutdown"))

			})
		})
	})
})
