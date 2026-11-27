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

// Package auth lets virt-handler's notify server tell which VMI a client connection belongs to.
//
// virt-launcher never connects to the notify server directly. virt-handler accepts the launcher
// connection on a pipe socket it created for one particular VMI and proxies it to
// domain-notify.sock. Before proxying any byte of the launcher, the pipe writes the Identity of
// that VMI. The notify server reads the identity back during the gRPC handshake, accepting it
// only from connections made by the virt-handler process itself, and exposes it to the RPC
// handlers through IdentityFromContext.
package auth

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
)

const (
	// lengthPrefixSize is the size of the big-endian length preceding the encoded identity.
	lengthPrefixSize = 4
	// maxIdentitySize bounds the encoded identity, the longest valid namespace, name and UID
	// fit comfortably.
	maxIdentitySize = 4096
)

// Identity is the VMI a client of the notify server acts for.
type Identity struct {
	Namespace string    `json:"namespace"`
	Name      string    `json:"name"`
	UID       types.UID `json:"uid"`
}

func IdentityFromVMI(vmi *v1.VirtualMachineInstance) Identity {
	return Identity{
		Namespace: vmi.Namespace,
		Name:      vmi.Name,
		UID:       vmi.UID,
	}
}

func (id Identity) String() string {
	return fmt.Sprintf("%s/%s (uid %q)", id.Namespace, id.Name, id.UID)
}

// Verify returns an error unless namespace, name and uid all match the identity.
func (id Identity) Verify(namespace, name string, uid types.UID) error {
	if namespace != id.Namespace || name != id.Name || uid != id.UID {
		return fmt.Errorf("%s/%s (uid %q) does not belong to client %s", namespace, name, uid, id)
	}
	return nil
}

func (id Identity) validate() error {
	if id.Namespace == "" || id.Name == "" || id.UID == "" {
		return fmt.Errorf("incomplete identity %s", id)
	}
	return nil
}

// WriteIdentity writes id to w as a 4 byte big-endian length followed by the JSON encoded identity.
func WriteIdentity(w io.Writer, id Identity) error {
	if err := id.validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(id)
	if err != nil {
		return err
	}
	if len(payload) > maxIdentitySize {
		return fmt.Errorf("encoded identity exceeds %d bytes", maxIdentitySize)
	}
	msg := make([]byte, lengthPrefixSize, lengthPrefixSize+len(payload))
	binary.BigEndian.PutUint32(msg, uint32(len(payload)))
	_, err = w.Write(append(msg, payload...))
	return err
}

// ReadIdentity reads an identity written by WriteIdentity.
// It never reads past the identity, whatever follows is left in r.
func ReadIdentity(r io.Reader) (Identity, error) {
	var prefix [lengthPrefixSize]byte
	if _, err := io.ReadFull(r, prefix[:]); err != nil {
		return Identity{}, fmt.Errorf("failed to read identity length: %w", err)
	}
	size := binary.BigEndian.Uint32(prefix[:])
	if size > maxIdentitySize {
		return Identity{}, fmt.Errorf("identity length %d exceeds %d bytes", size, maxIdentitySize)
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return Identity{}, fmt.Errorf("failed to read identity: %w", err)
	}
	var id Identity
	if err := json.Unmarshal(payload, &id); err != nil {
		return Identity{}, fmt.Errorf("failed to decode identity: %w", err)
	}
	if err := id.validate(); err != nil {
		return Identity{}, err
	}
	return id, nil
}
