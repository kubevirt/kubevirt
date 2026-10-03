/*
 * This file is part of the kubevirt project
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

package ignition

import (
	"fmt"
	"os"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
)

var ignitionLocalDir = "/var/run/libvirt/ignition-dir"

const IgnitionFile = "data.ign"

func GetIgnitionSource(vmi *v1.VirtualMachineInstance) string {
	precond.MustNotBeNil(vmi)
	return vmi.Annotations[v1.IgnitionAnnotation]
}

func SetLocalDirectory(dir string) error {
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return fmt.Errorf("Unable to initialize Ignition local cache directory (%s): %w", dir, err)
	}
	if err := chownQemu(dir); err != nil {
		return fmt.Errorf("Unable to initialize Ignition local cache directory (%s): %w", dir, err)
	}

	exists, err := diskutils.FileExists(dir)
	if err != nil {
		return fmt.Errorf("Ignition local cache directory (%s) does not exist or is inaccessible: %w", dir, err)
	} else if exists == false {
		return fmt.Errorf("Ignition local cache directory (%s) does not exist or is inaccessible", dir)
	}

	ignitionLocalDir = dir
	return nil
}

func GetDomainBasePath(domain string, namespace string) string {
	return fmt.Sprintf("%s/%s/%s", ignitionLocalDir, namespace, domain)
}

func GenerateIgnitionLocalData(vmi *v1.VirtualMachineInstance, namespace string) error {
	precond.MustNotBeEmpty(vmi.Name)
	precond.MustNotBeNil(vmi.Annotations[v1.IgnitionAnnotation])

	domainBasePath := GetDomainBasePath(vmi.Name, namespace)
	err := os.MkdirAll(domainBasePath, 0700)
	if err != nil {
		log.Log.Reason(err).Errorf("unable to create Ignition base path %s", domainBasePath)
		return err
	}
	if err := chownQemu(domainBasePath); err != nil {
		return err
	}

	ignitionFile := fmt.Sprintf("%s/%s", domainBasePath, IgnitionFile)
	ignitionData := []byte(vmi.Annotations[v1.IgnitionAnnotation])
	err = os.WriteFile(ignitionFile, ignitionData, 0600)
	if err != nil {
		return err
	}
	if err := chownQemu(ignitionFile); err != nil {
		return err
	}

	log.Log.V(2).Infof("generated Ignition file %s", ignitionFile)
	return nil
}

// chownQemu makes path readable by the qemu user that runs the VM, for the
// case where virt-launcher runs as root and creates it owned by root.
// drop this when the Root feature gate is dropped.
func chownQemu(path string) error {
	const qemuUid, qemuGid = 107, 107
	if os.Geteuid() != 0 {
		return nil
	}
	return os.Chown(path, qemuUid, qemuGid)
}
