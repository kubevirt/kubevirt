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
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	"kubevirt.io/client-go/log"
)

const authType = "kubevirt-notify-identity"

// AuthInfo is attached to every connection accepted with NewServerCredentials.
type AuthInfo struct {
	credentials.CommonAuthInfo
	Identity Identity
}

func (AuthInfo) AuthType() string {
	return authType
}

// NewServerCredentials returns gRPC server credentials which accept only connections made by the
// current process, i.e. the virt-handler pipes, and read the identity of the client from them.
// They don't provide any transport security.
func NewServerCredentials() credentials.TransportCredentials {
	return &serverCredentials{pid: os.Getpid()}
}

type serverCredentials struct {
	// pid of the only process allowed to connect
	pid int
}

func (c *serverCredentials) ServerHandshake(conn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	if err := c.verifyPeer(conn); err != nil {
		log.Log.Reason(err).Warning("Rejected notify connection from unexpected peer")
		return nil, nil, err
	}
	id, err := ReadIdentity(conn)
	if err != nil {
		log.Log.Reason(err).Warning("Rejected notify connection without a valid client identity")
		return nil, nil, err
	}
	return conn, AuthInfo{
		CommonAuthInfo: credentials.CommonAuthInfo{SecurityLevel: credentials.NoSecurity},
		Identity:       id,
	}, nil
}

// verifyPeer ensures the connection was made by the current process. Anyone else able to reach
// domain-notify.sock could claim any identity.
func (c *serverCredentials) verifyPeer(conn net.Conn) error {
	sysConn, ok := conn.(syscall.Conn)
	if !ok {
		return fmt.Errorf("cannot verify the peer of a %T connection", conn)
	}
	rawConn, err := sysConn.SyscallConn()
	if err != nil {
		return err
	}
	var cred *unix.Ucred
	var credErr error
	err = rawConn.Control(func(fd uintptr) {
		cred, credErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	})
	if err != nil {
		return err
	}
	if credErr != nil {
		return fmt.Errorf("failed to read peer credentials: %w", credErr)
	}
	if int(cred.Pid) != c.pid {
		return fmt.Errorf("connection made by pid %d, only pid %d is allowed to connect", cred.Pid, c.pid)
	}
	return nil
}

func (*serverCredentials) ClientHandshake(context.Context, string, net.Conn) (net.Conn, credentials.AuthInfo, error) {
	return nil, nil, errors.New("notify identity credentials can only be used by the server")
}

func (*serverCredentials) Info() credentials.ProtocolInfo {
	return credentials.ProtocolInfo{SecurityProtocol: authType}
}

func (c *serverCredentials) Clone() credentials.TransportCredentials {
	clone := *c
	return &clone
}

func (*serverCredentials) OverrideServerName(string) error {
	return nil
}

// IdentityFromContext returns the identity of the client issuing the RPC.
func IdentityFromContext(ctx context.Context) (Identity, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return Identity{}, errors.New("no peer found in context")
	}
	info, ok := p.AuthInfo.(AuthInfo)
	if !ok {
		return Identity{}, fmt.Errorf("unexpected peer auth info %T", p.AuthInfo)
	}
	return info.Identity, nil
}
