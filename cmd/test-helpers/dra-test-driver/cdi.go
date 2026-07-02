/*
 * Copyright The Kubernetes Authors.
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
 */

package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"k8s.io/klog/v2"
	cdiapi "tags.cncf.io/container-device-interface/pkg/cdi"
	cdiparser "tags.cncf.io/container-device-interface/pkg/parser"
	cdispec "tags.cncf.io/container-device-interface/specs-go"
)

const cdiCommonDeviceName = "common"

var nonWord = regexp.MustCompile(`[^a-zA-Z0-9]+`)

type CDIHandler struct {
	cache      *cdiapi.Cache
	driverName string
	class      string
}

func NewCDIHandler(root string, driverName, class string) (*CDIHandler, error) {
	cache, err := cdiapi.NewCache(
		cdiapi.WithSpecDirs(root),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create a new CDI cache: %w", err)
	}
	handler := &CDIHandler{
		cache:      cache,
		driverName: driverName,
		class:      class,
	}

	return handler, nil
}

func (cdi *CDIHandler) CreateCommonSpecFile() error {
	spec := &cdispec.Spec{
		Kind: cdi.kind(),
		Devices: []cdispec.Device{
			{
				Name: cdiCommonDeviceName,
				ContainerEdits: cdispec.ContainerEdits{
					Env: []string{
						fmt.Sprintf("KUBERNETES_NODE_NAME=%s", os.Getenv("NODE_NAME")),
						fmt.Sprintf("DRA_RESOURCE_DRIVER_NAME=%s", cdi.driverName),
					},
				},
			},
		},
	}

	minVersion, err := cdispec.MinimumRequiredVersion(spec)
	if err != nil {
		return fmt.Errorf("failed to get minimum required CDI spec version: %v", err)
	}
	spec.Version = minVersion

	specName, err := cdiapi.GenerateNameForTransientSpec(spec, cdiCommonDeviceName)
	if err != nil {
		return fmt.Errorf("failed to generate Spec name: %w", err)
	}

	return cdi.cache.WriteSpec(spec, specName)
}

func (cdi *CDIHandler) CreateClaimSpecFile(claimUID string, devices PreparedDevices) error {
	specName := cdiapi.GenerateTransientSpecName(cdi.vendor(), cdi.class, claimUID)
	klog.Infof("Creating CDI spec file for claim %s with name %s", claimUID, specName)

	spec := &cdispec.Spec{
		Kind:    cdi.kind(),
		Devices: []cdispec.Device{},
	}

	for _, device := range devices {
		deviceEnvKey := strings.ToUpper(nonWord.ReplaceAllString(device.DeviceName, "_"))
		claimEdits := cdiapi.ContainerEdits{
			ContainerEdits: &cdispec.ContainerEdits{
				Env: []string{
					fmt.Sprintf("%s_DEVICE_%s_RESOURCE_CLAIM=%s", strings.ToUpper(cdi.class), deviceEnvKey, claimUID),
					fmt.Sprintf("DRA_ADMIN_ACCESS=%t", device.AdminAccess),
				},
			},
		}

		// If this device has admin access, then here is where to inject host hardware information

		claimEdits.Append(device.ContainerEdits)

		cdiDevice := cdispec.Device{
			Name:           fmt.Sprintf("%s-%s", claimUID, device.DeviceName),
			ContainerEdits: *claimEdits.ContainerEdits,
		}

		spec.Devices = append(spec.Devices, cdiDevice)
	}

	minVersion, err := cdispec.MinimumRequiredVersion(spec)
	if err != nil {
		return fmt.Errorf("failed to get minimum required CDI spec version: %v", err)
	}
	spec.Version = minVersion

	klog.Infof("Writing CDI spec for claim %s, devices: %d", claimUID, len(spec.Devices))
	if err := cdi.cache.WriteSpec(spec, specName); err != nil {
		klog.Errorf("Failed to write CDI spec for claim %s: %v", claimUID, err)
		return err
	}
	klog.Infof("Successfully wrote CDI spec file for claim %s", claimUID)
	return nil
}

func (cdi *CDIHandler) DeleteClaimSpecFile(claimUID string) error {
	specName := cdiapi.GenerateTransientSpecName(cdi.vendor(), cdi.class, claimUID)
	return cdi.cache.RemoveSpec(specName)
}

func (cdi *CDIHandler) GetClaimDevices(claimUID string, devices []string) []string {
	cdiDevices := []string{
		cdiparser.QualifiedName(cdi.vendor(), cdi.class, cdiCommonDeviceName),
	}

	for _, device := range devices {
		cdiDevice := cdiparser.QualifiedName(cdi.vendor(), cdi.class, fmt.Sprintf("%s-%s", claimUID, device))
		cdiDevices = append(cdiDevices, cdiDevice)
	}

	return cdiDevices
}

func (cdi *CDIHandler) kind() string {
	return cdi.vendor() + "/" + cdi.class
}

func (cdi *CDIHandler) vendor() string {
	return "k8s." + cdi.driverName
}
