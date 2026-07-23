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

package synchronization

import (
	"context"
	"io"
	"net"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SyncProxyManager", func() {
	var (
		proxyManager   *SyncProxyManager
		migrationIP    string
		crossClusterIP string
	)

	BeforeEach(func() {
		proxyManager = NewSyncProxyManager()
		migrationIP = "127.0.0.1"
		crossClusterIP = "127.0.0.2"
		proxyManager.Initialize(migrationIP, crossClusterIP)
	})

	AfterEach(func() {
		if proxyManager != nil {
			proxyManager.Shutdown()
		}
	})

	Describe("Port allocation", func() {
		It("should allocate unique ports for different migrations", func() {
			migrationUID1 := "migration-1"
			migrationUID2 := "migration-2"
			targetPortMap := map[int]int{9999: 0}

			By("Starting first source proxy")
			portMap1, err := proxyManager.StartSourceProxies(context.Background(), migrationUID1, "127.0.0.1", targetPortMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(portMap1).To(HaveLen(1))

			By("Starting second source proxy")
			portMap2, err := proxyManager.StartSourceProxies(context.Background(), migrationUID2, "127.0.0.1", targetPortMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(portMap2).To(HaveLen(1))

			By("Verifying port maps are different")
			Expect(portMap1).ToNot(Equal(portMap2))
		})

		It("should return same ports for duplicate migration UID", func() {
			migrationUID := "migration-duplicate"
			targetPortMap := map[int]int{9999: 0}

			By("Starting source proxy first time")
			portMap1, err := proxyManager.StartSourceProxies(context.Background(), migrationUID, "127.0.0.1", targetPortMap)
			Expect(err).ToNot(HaveOccurred())

			By("Starting source proxy again with same UID")
			portMap2, err := proxyManager.StartSourceProxies(context.Background(), migrationUID, "127.0.0.1", targetPortMap)
			Expect(err).ToNot(HaveOccurred())

			By("Verifying same port map is returned")
			Expect(portMap1).To(Equal(portMap2))
		})
	})

	DescribeTable("Proxy lifecycle",
		func(startProxy func(string, string, map[int]int) (map[int]int, error), stopProxy func(string), proxyType string) {
			migrationUID := "migration-stop-test"
			targetIP := "127.0.0.1"
			portMap := map[int]int{9999: 0}

			By("Starting " + proxyType + " proxy")
			resultMap, err := startProxy(migrationUID, targetIP, portMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(resultMap).To(HaveLen(1))

			By("Stopping " + proxyType + " proxy")
			stopProxy(migrationUID)

			By("Restarting " + proxyType + " proxy after stop")
			resultMap2, err := startProxy(migrationUID, targetIP, portMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(resultMap2).To(HaveLen(1))
		},
		Entry("source proxy",
			func(uid, ip string, ports map[int]int) (map[int]int, error) {
				return proxyManager.StartSourceProxies(context.Background(), uid, ip, ports)
			},
			func(uid string) { proxyManager.StopSourceProxy(uid) },
			"source",
		),
		Entry("target proxy",
			func(uid, ip string, ports map[int]int) (map[int]int, error) {
				return proxyManager.StartTargetProxies(context.Background(), uid, ip, ports)
			},
			func(uid string) { proxyManager.StopTargetProxy(uid) },
			"target",
		),
	)

	Describe("Proxy lifecycle edge cases", func() {
		It("should handle stopping non-existent proxy gracefully", func() {
			By("Stopping non-existent source proxy")
			proxyManager.StopSourceProxy("non-existent")

			By("Stopping non-existent target proxy")
			proxyManager.StopTargetProxy("non-existent")
		})

		It("should recreate proxies when a listener dies unexpectedly", func() {
			migrationUID := "migration-dead-listener"
			targetPortMap := map[int]int{9999: 0}

			By("Starting source proxy")
			portMap1, err := proxyManager.StartSourceProxies(context.Background(), migrationUID, "127.0.0.1", targetPortMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(portMap1).To(HaveLen(1))

			var proxyPort int
			for port := range portMap1 {
				proxyPort = port
			}
			proxyAddr := net.JoinHostPort(migrationIP, strconv.Itoa(proxyPort))

			By("Forcing the listener to die without calling StopSourceProxy")
			proxyManager.closeProxyListeners(migrationUID, false)

			Eventually(func() map[int]int {
				return proxyManager.GetSourceProxyPorts(migrationUID)
			}).Should(BeNil())

			By("Starting source proxy again should allocate a new healthy listener")
			portMap2, err := proxyManager.StartSourceProxies(context.Background(), migrationUID, "127.0.0.1", targetPortMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(portMap2).To(HaveLen(1))

			var newProxyPort int
			for port := range portMap2 {
				newProxyPort = port
			}
			newProxyAddr := net.JoinHostPort(migrationIP, strconv.Itoa(newProxyPort))

			Eventually(func() error {
				conn, err := net.DialTimeout("tcp", newProxyAddr, 100*time.Millisecond)
				if conn != nil {
					conn.Close()
				}
				return err
			}).Should(Succeed())

			Consistently(func() error {
				conn, err := net.DialTimeout("tcp", proxyAddr, 50*time.Millisecond)
				if conn != nil {
					conn.Close()
				}
				return err
			}).ShouldNot(Succeed())
		})
	})

	Describe("Connection forwarding", func() {
		It("should forward data bidirectionally through proxies", func() {
			By("Setting up echo server as the target")
			targetListener, err := net.Listen("tcp", "127.0.0.1:0")
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(func() {
				targetListener.Close()
			})

			_, targetPortStr, err := net.SplitHostPort(targetListener.Addr().String())
			Expect(err).ToNot(HaveOccurred())

			By("Starting echo server goroutine")
			go func() {
				for {
					conn, err := targetListener.Accept()
					if err != nil {
						return
					}
					go func(c net.Conn) {
						defer c.Close()
						_, err := io.Copy(c, c)
						Expect(err).To(Or(BeNil(), Equal(io.EOF)))
					}(conn)
				}
			}()

			By("Starting source proxies with single port")
			migrationUID := "migration-forward-test"
			targetPort, err := strconv.Atoi(targetPortStr)
			Expect(err).ToNot(HaveOccurred())
			targetPortMap := map[int]int{targetPort: 0}
			sourcePortMap, err := proxyManager.StartSourceProxies(context.Background(), migrationUID, "127.0.0.1", targetPortMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(sourcePortMap).To(HaveLen(1))

			By("Extracting proxy port")
			var proxyPort int
			for port := range sourcePortMap {
				proxyPort = port
				break
			}
			proxyAddr := net.JoinHostPort(migrationIP, strconv.Itoa(proxyPort))

			By("Waiting for proxy to be ready")
			Eventually(func() error {
				conn, err := net.DialTimeout("tcp", proxyAddr, 100*time.Millisecond)
				if conn != nil {
					conn.Close()
				}
				return err
			}).Should(Succeed())

			By("Connecting to proxy")
			conn, err := net.Dial("tcp", proxyAddr)
			Expect(err).ToNot(HaveOccurred())
			defer conn.Close()

			By("Sending test data through proxy")
			testData := []byte("test message")
			_, err = conn.Write(testData)
			Expect(err).ToNot(HaveOccurred())

			By("Reading echoed data from proxy")
			buf := make([]byte, len(testData))
			_, err = io.ReadFull(conn, buf)
			Expect(err).ToNot(HaveOccurred())

			By("Verifying data was correctly forwarded")
			Expect(buf).To(Equal(testData))
		})
	})

	Describe("Manager shutdown", func() {
		It("should prevent new proxies after shutdown", func() {
			By("Shutting down proxy manager")
			proxyManager.Shutdown()

			migrationUID := "migration-after-shutdown"
			targetPortMap := map[int]int{9999: 0}

			By("Attempting to start source proxies after shutdown")
			_, err := proxyManager.StartSourceProxies(context.Background(), migrationUID, "127.0.0.1", targetPortMap)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("shutting down"))

			By("Attempting to start target proxies after shutdown")
			_, err = proxyManager.StartTargetProxies(context.Background(), migrationUID, "127.0.0.1", targetPortMap)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("shutting down"))
		})

		It("should be idempotent when called multiple times", func() {
			migrationUID := "migration-for-shutdown"
			targetPortMap := map[int]int{9999: 0}

			By("Starting some proxies")
			_, err := proxyManager.StartSourceProxies(context.Background(), migrationUID, "127.0.0.1", targetPortMap)
			Expect(err).ToNot(HaveOccurred())

			By("Calling Shutdown first time")
			Expect(func() {
				proxyManager.Shutdown()
			}).ToNot(Panic())

			By("Calling Shutdown second time (should be no-op)")
			Expect(func() {
				proxyManager.Shutdown()
			}).ToNot(Panic())

			By("Calling Shutdown third time (should still be no-op)")
			Expect(func() {
				proxyManager.Shutdown()
			}).ToNot(Panic())
		})
	})

	Describe("Multi-port proxies", func() {
		It("should create source proxies for multiple ports", func() {
			migrationUID := "multi-port-migration"
			targetIP := "127.0.0.1"
			DeferCleanup(func() {
				proxyManager.StopSourceProxy(migrationUID)
			})

			By("Creating target port map with 3 ports (virtqemud, libvirt, block)")
			targetPortMap := map[int]int{
				5001: 0,     // virtqemud socket
				5002: 49152, // libvirt direct
				5003: 49153, // block migration
			}

			By("Starting source proxies")
			sourceProxyPortMap, err := proxyManager.StartSourceProxies(context.Background(), migrationUID, targetIP, targetPortMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(sourceProxyPortMap).To(HaveLen(3))

			By("Verifying protocol port values are preserved")
			for port, protocolPort := range sourceProxyPortMap {
				Expect(port).ToNot(BeZero())
				Expect(protocolPort).To(BeElementOf(0, 49152, 49153))
			}

			By("Verifying idempotency - calling again returns same ports")
			sourceProxyPortMap2, err := proxyManager.StartSourceProxies(context.Background(), migrationUID, targetIP, targetPortMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(sourceProxyPortMap2).To(Equal(sourceProxyPortMap))
		})

		It("should create target proxies for multiple ports", func() {
			migrationUID := "multi-port-target-migration"
			targetVirtHandlerIP := "127.0.0.1"
			DeferCleanup(func() {
				proxyManager.StopTargetProxy(migrationUID)
			})

			By("Creating target virt-handler port map")
			virtHandlerPortMap := map[int]int{
				6001: 0,     // virtqemud socket
				6002: 49152, // libvirt direct
				6003: 49153, // block migration
			}

			By("Starting target proxies")
			targetProxyPortMap, err := proxyManager.StartTargetProxies(context.Background(), migrationUID, targetVirtHandlerIP, virtHandlerPortMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(targetProxyPortMap).To(HaveLen(3))

			By("Verifying protocol port values are preserved")
			for port, protocolPort := range targetProxyPortMap {
				Expect(port).ToNot(BeZero())
				Expect(protocolPort).To(BeElementOf(0, 49152, 49153))
			}

			By("Verifying GetTargetProxyPorts returns the same ports")
			retrievedPorts := proxyManager.GetTargetProxyPorts(migrationUID)
			Expect(retrievedPorts).To(Equal(targetProxyPortMap))

			By("Verifying GetTargetProxyPorts returns nil after stop")
			proxyManager.StopTargetProxy(migrationUID)
			retrievedPorts = proxyManager.GetTargetProxyPorts(migrationUID)
			Expect(retrievedPorts).To(BeNil())
		})

		It("should preserve protocol port mapping through proxy chain", func() {
			migrationUID := "protocol-preservation-test"
			DeferCleanup(func() {
				proxyManager.StopSourceProxy(migrationUID)
			})

			By("Creating initial port map")
			targetPortMap := map[int]int{
				7001: 0,
				7002: 49152,
				7003: 49153,
			}

			By("Starting source proxies")
			sourcePortMap, err := proxyManager.StartSourceProxies(context.Background(), migrationUID, "127.0.0.1", targetPortMap)
			Expect(err).ToNot(HaveOccurred())

			By("Verifying all protocol values are preserved")
			protocolValues := make(map[int]bool)
			for _, protocolPort := range sourcePortMap {
				protocolValues[protocolPort] = true
			}
			Expect(protocolValues).To(HaveKey(0))
			Expect(protocolValues).To(HaveKey(49152))
			Expect(protocolValues).To(HaveKey(49153))
			Expect(protocolValues).To(HaveLen(3))
		})

		DescribeTable("Idempotent cleanup prevents resource leaks",
			func(
				startProxy func(string, map[int]int) (map[int]int, error),
				stopProxy func(string),
				getProxyPorts func(string) map[int]int,
			) {
				migrationUID := "idempotent-cleanup-test"
				portMap := map[int]int{50000: 0}

				By("Starting proxy")
				resultMap, err := startProxy(migrationUID, portMap)
				Expect(err).ToNot(HaveOccurred())
				Expect(resultMap).ToNot(BeEmpty())

				By("Verifying proxy is active")
				activePorts := getProxyPorts(migrationUID)
				Expect(activePorts).ToNot(BeNil())

				By("Stopping proxy first time")
				stopProxy(migrationUID)

				By("Verifying proxy is stopped")
				activePorts = getProxyPorts(migrationUID)
				Expect(activePorts).To(BeNil())

				By("Stopping proxy second time (should be no-op)")
				Expect(func() {
					stopProxy(migrationUID)
				}).ToNot(Panic())

				By("Stopping proxy third time (should still be no-op)")
				Expect(func() {
					stopProxy(migrationUID)
				}).ToNot(Panic())

				By("Verifying proxy remains stopped")
				activePorts = getProxyPorts(migrationUID)
				Expect(activePorts).To(BeNil())
			},
			Entry("source proxy cleanup is idempotent",
				func(uid string, portMap map[int]int) (map[int]int, error) {
					return proxyManager.StartSourceProxies(context.Background(), uid, "127.0.0.1", portMap)
				},
				func(uid string) {
					proxyManager.StopSourceProxy(uid)
				},
				func(uid string) map[int]int {
					return proxyManager.GetSourceProxyPorts(uid)
				},
			),
			Entry("target proxy cleanup is idempotent",
				func(uid string, portMap map[int]int) (map[int]int, error) {
					return proxyManager.StartTargetProxies(context.Background(), uid, "127.0.0.1", portMap)
				},
				func(uid string) {
					proxyManager.StopTargetProxy(uid)
				},
				func(uid string) map[int]int {
					return proxyManager.GetTargetProxyPorts(uid)
				},
			),
		)
	})

	Describe("IPv4 and IPv6 address support", func() {
		DescribeTable("should create proxy listeners on both IPv4 and IPv6 addresses",
			func(listenIP, targetIP string) {
				proxyManager := NewSyncProxyManager()
				proxyManager.Initialize(listenIP, listenIP)

				migrationUID := "migration-iptest"
				targetPortMap := map[int]int{9999: 0}

				By("Starting target proxies on the specified IP")
				portMap, err := proxyManager.StartTargetProxies(context.Background(), migrationUID, targetIP, targetPortMap)
				Expect(err).ToNot(HaveOccurred())
				Expect(portMap).To(HaveLen(1))

				By("Verifying the proxy listener is actually bound")
				// Get the port that was allocated
				var allocatedPort int
				for port := range portMap {
					allocatedPort = port
					break
				}

				// Try to connect to the listener to verify it's actually bound
				listenAddr := net.JoinHostPort(listenIP, strconv.Itoa(allocatedPort))
				conn, err := net.DialTimeout("tcp", listenAddr, 100*time.Millisecond)
				Expect(err).ToNot(HaveOccurred(), "should be able to connect to the proxy listener")
				if conn != nil {
					conn.Close()
				}

				By("Cleaning up")
				proxyManager.StopTargetProxy(migrationUID)
			},
			Entry("IPv4 loopback", "127.0.0.1", "127.0.0.1"),
			Entry("IPv6 loopback", "::1", "::1"),
		)
	})

	Describe("portMapsMatch", func() {
		DescribeTable("should correctly compare requested port maps with existing proxies",
			func(requestedPortMap map[int]int, existingProxies map[int]*migrationProxy, expectedMatch bool) {
				result := portMapsMatch(requestedPortMap, existingProxies)
				Expect(result).To(Equal(expectedMatch))
			},
			Entry("both empty maps",
				map[int]int{},
				map[int]*migrationProxy{},
				true,
			),
			Entry("nil requested map and empty existing proxies",
				nil,
				map[int]*migrationProxy{},
				true,
			),
			Entry("empty requested map and nil existing proxies",
				map[int]int{},
				nil,
				true,
			),
			Entry("same protocol ports with same proxy ports",
				map[int]int{8080: 9999, 8081: 9998},
				map[int]*migrationProxy{
					8080: {protocolPort: 9999},
					8081: {protocolPort: 9998},
				},
				true,
			),
			Entry("same protocol ports with different proxy ports",
				map[int]int{1111: 9999, 2222: 9998},
				map[int]*migrationProxy{
					8080: {protocolPort: 9999},
					8081: {protocolPort: 9998},
				},
				true,
			),
			Entry("different protocol ports",
				map[int]int{8080: 9999},
				map[int]*migrationProxy{
					8080: {protocolPort: 8888},
				},
				false,
			),
			Entry("different number of ports - requested has more",
				map[int]int{8080: 9999, 8081: 9998},
				map[int]*migrationProxy{
					8080: {protocolPort: 9999},
				},
				false,
			),
			Entry("different number of ports - existing has more",
				map[int]int{8080: 9999},
				map[int]*migrationProxy{
					8080: {protocolPort: 9999},
					8081: {protocolPort: 9998},
				},
				false,
			),
			Entry("requested has subset of existing protocol ports",
				map[int]int{8080: 9999},
				map[int]*migrationProxy{
					8080: {protocolPort: 9999},
					8081: {protocolPort: 9998},
				},
				false,
			),
			Entry("existing has subset of requested protocol ports",
				map[int]int{8080: 9999, 8081: 9998},
				map[int]*migrationProxy{
					8080: {protocolPort: 9999},
				},
				false,
			),
		)
	})
})
