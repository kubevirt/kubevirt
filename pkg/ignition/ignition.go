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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package ignition

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/precond"
)

var ignitionLocalDir = "/var/run/libvirt/ignition-dir"

const IgnitionFile = "data.ign"

func GetIgnitionSource(vmi *v1.VirtualMachineInstance) string {
	precond.MustNotBeNil(vmi)
	return vmi.Annotations[v1.IgnitionAnnotation]
}

func SetLocalDirectory(dir string) error {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to initialize Ignition local cache directory (%s). %v", dir, err))
	}

	exists, err := diskutils.FileExists(dir)
	if err != nil {
		return errors.New(fmt.Sprintf("Ignition local cache directory (%s) does not exist or is inaccessible. %v", dir, err))
	} else if exists == false {
		return errors.New(fmt.Sprintf("Ignition local cache directory (%s) does not exist or is inaccessible.", dir))
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
	err := os.MkdirAll(domainBasePath, 0755)
	if err != nil {
		log.Log.V(2).Reason(err).Errorf("unable to create Ignition base path %s", domainBasePath)
		return err
	}

	ignitionFile := fmt.Sprintf("%s/%s", domainBasePath, "data.ign")
	ignitionData := []byte(vmi.Annotations[v1.IgnitionAnnotation])
	err = ioutil.WriteFile(ignitionFile, ignitionData, 0644)
	if err != nil {
		return err
	}

	log.Log.V(2).Infof("generated Ignition file %s/data.ign", domainBasePath)
	return nil
}
