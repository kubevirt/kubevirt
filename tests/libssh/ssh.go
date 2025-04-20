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

package libssh

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"golang.org/x/crypto/ssh"

	"kubevirt.io/kubevirt/tests/errorhandling"
)

func NewKeyPair() (*ecdsa.PrivateKey, ssh.PublicKey, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, pub, nil
}

func DumpPrivateKey(privateKey *ecdsa.PrivateKey, file string) error {
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return err
	}
	privateKeyBlock := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateKeyBytes,
	}
	const filePermissions = 0o600
	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, filePermissions)
	if err != nil {
		return err
	}
	defer errorhandling.SafelyCloseFile(f)
	if err = pem.Encode(f, privateKeyBlock); err != nil {
		return fmt.Errorf("error when encode private pem: %s", err)
	}
	return nil
}

func RenderUserDataWithKey(key ssh.PublicKey) string {
	return fmt.Sprintf(`#!/bin/sh
mkdir -p /root/.ssh/
echo "%s" > /root/.ssh/authorized_keys
chown -R root:root /root/.ssh
`, string(ssh.MarshalAuthorizedKey(key)))
}

// DisableSSHAgent allows disabling the SSH agent to not influence test results
func DisableSSHAgent() {
	const sshAuthSock = "SSH_AUTH_SOCK"
	val, present := os.LookupEnv(sshAuthSock)
	if present {
		Expect(os.Unsetenv(sshAuthSock)).To(Succeed())
		DeferCleanup(func() {
			Expect(os.Setenv(sshAuthSock, val)).To(Succeed())
		})
	}
}
