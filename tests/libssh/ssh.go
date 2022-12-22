package libssh

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/clientcmd"
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
	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer errorhandling.SafelyCloseFile(f)
	if err = pem.Encode(f, privateKeyBlock); err != nil {
		return fmt.Errorf("error when encode private pem: %s", err)

	}
	return nil
}

func GenerateKeyPair(tmpDir string) (privateKeyPath string, publicKey ssh.PublicKey, err error) {
	priv, pub, err := NewKeyPair()
	if err != nil {
		return "", nil, err
	}
	path := filepath.Join(tmpDir, "private.key")
	if err := DumpPrivateKey(priv, path); err != nil {
		return "", nil, err
	}
	return path, pub, nil
}

func RenderUserDataWithKey(key ssh.PublicKey) string {
	return fmt.Sprintf(`#!/bin/sh
mkdir -p /root/.ssh/
echo "%s" > /root/.ssh/authorized_keys
chown -R root:root /root/.ssh
`, string(ssh.MarshalAuthorizedKey(key)))
}

func SCPToVMI(vmi *v1.VirtualMachineInstance, keyFile, srcFile, targetFile string) error {
	args := []string{
		"scp",
		"--namespace", vmi.Namespace,
		"--username", "root",
		"--identity-file", keyFile,
		"--known-hosts=",
	}

	args = append(args, srcFile, fmt.Sprintf("%s:%s", vmi.Name, targetFile))

	return clientcmd.NewRepeatableVirtctlCommand(args...)()
}
