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

package auth

import (
	"context"
	"io"
	"net"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

type foreignAuthInfo struct{}

func (foreignAuthInfo) AuthType() string { return "foreign" }

var _ = Describe("Server credentials", func() {
	identity := Identity{Namespace: "default", Name: "testvmi", UID: "1234"}

	// unixConnPair returns both ends of a unix socket connection made by this process
	unixConnPair := func() (server, client net.Conn) {
		listener, err := net.Listen("unix", filepath.Join(GinkgoT().TempDir(), "test.sock"))
		Expect(err).ToNot(HaveOccurred())
		defer listener.Close()

		client, err = net.Dial("unix", listener.Addr().String())
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(client.Close)

		server, err = listener.Accept()
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(server.Close)
		return server, client
	}

	It("should authenticate the identity sent by the current process", func() {
		server, client := unixConnPair()
		Expect(WriteIdentity(client, identity)).To(Succeed())
		_, err := client.Write([]byte("Hello"))
		Expect(err).ToNot(HaveOccurred())

		conn, authInfo, err := NewServerCredentials().ServerHandshake(server)
		Expect(err).ToNot(HaveOccurred())
		Expect(authInfo).To(Equal(AuthInfo{
			CommonAuthInfo: credentials.CommonAuthInfo{SecurityLevel: credentials.NoSecurity},
			Identity:       identity,
		}))

		By("handing over the client data following the identity")
		msg := make([]byte, 5)
		_, err = io.ReadFull(conn, msg)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(msg)).To(Equal("Hello"))
	})

	It("should reject connections made by another process", func() {
		server, client := unixConnPair()
		Expect(WriteIdentity(client, identity)).To(Succeed())

		creds := &serverCredentials{pid: os.Getpid() + 1}
		_, _, err := creds.ServerHandshake(server)
		Expect(err).To(MatchError(ContainSubstring("only pid")))
	})

	It("should reject connections whose peer cannot be verified", func() {
		server, client := net.Pipe()
		DeferCleanup(server.Close)
		DeferCleanup(client.Close)

		_, _, err := NewServerCredentials().ServerHandshake(server)
		Expect(err).To(MatchError(ContainSubstring("cannot verify the peer")))
	})

	It("should reject connections without identity", func() {
		server, client := unixConnPair()
		_, err := client.Write([]byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"))
		Expect(err).ToNot(HaveOccurred())

		_, _, err = NewServerCredentials().ServerHandshake(server)
		Expect(err).To(MatchError(ContainSubstring("exceeds")))
	})

	It("should not be usable by clients", func() {
		_, _, err := NewServerCredentials().ClientHandshake(context.Background(), "", nil)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("IdentityFromContext", func() {
	It("should return the identity of the peer", func() {
		identity := Identity{Namespace: "default", Name: "testvmi", UID: "1234"}
		ctx := peer.NewContext(context.Background(), &peer.Peer{AuthInfo: AuthInfo{Identity: identity}})

		Expect(IdentityFromContext(ctx)).To(Equal(identity))
	})

	It("should fail without peer", func() {
		_, err := IdentityFromContext(context.Background())
		Expect(err).To(HaveOccurred())
	})

	It("should fail when the peer was not authenticated by the server credentials", func() {
		ctx := peer.NewContext(context.Background(), &peer.Peer{AuthInfo: foreignAuthInfo{}})

		_, err := IdentityFromContext(ctx)
		Expect(err).To(HaveOccurred())
	})
})
