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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package device_manager

import (
	"container/ring"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"kubevirt.io/client-go/log"
)

// Not a const for static test purposes
var mdevClassBusPath = "/sys/class/mdev_bus"

type MDEVTypesManager struct {
	availableMdevTypesMap   map[string][]string
	unconfiguredParentsMap  map[string]struct{}
	mdevsConfigurationMutex sync.Mutex
}

func NewMDEVTypesManager() *MDEVTypesManager {
	initHandler()
	return &MDEVTypesManager{
		availableMdevTypesMap: make(map[string][]string),
	}
}

func (m *MDEVTypesManager) getAlreadyConfiguredMdevParents() (map[string]struct{}, error) {
	configuredPCICards := make(map[string]struct{})
	files, err := filepath.Glob("/sys/bus/mdev/devices/*")
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		originFile, err := os.Readlink(file)
		if err != nil {
			return nil, err
		}

		filePathParts := strings.Split(originFile, string(os.PathSeparator))
		// originFile is the path to the UUID directory under the device path. Example:
		// /sys/devices/pci0000:e0/0000:e0:03.1/0000:e2:01.2/09f7ea8a-b325-4945-8a15-1892bfd22dd2
		// In that example, parentID would be 0000:e2:01.2
		// The smallest split imaginable would have a length of 5:
		// [ "", "sys", "devices", <parentID>, <UUID> ]
		if len(filePathParts) < 5 {
			return nil, fmt.Errorf("invalid device path: %s", originFile)
		}
		parentID := filePathParts[len(filePathParts)-2]
		configuredPCICards[parentID] = struct{}{}
	}
	return configuredPCICards, nil
}

func (m *MDEVTypesManager) updateMDEVTypesConfiguration(desiredTypesList []string, externallyProvidedTypesMap map[string]struct{}) (bool, error) {
	m.mdevsConfigurationMutex.Lock()
	defer m.mdevsConfigurationMutex.Unlock()

	// create a map of types that should not be removed
	typesToKeepMap := make(map[string]struct{})
	for key, val := range externallyProvidedTypesMap {
		typesToKeepMap[key] = val
	}

	// construct a map of desired types for lookup
	desiredTypesMap := make(map[string]struct{})
	for _, mdevType := range desiredTypesList {
		desiredTypesMap[mdevType] = struct{}{}
		typesToKeepMap[mdevType] = struct{}{}
	}

	// the following will remove all configured types that have not been
	// created by an external provider and are not in the desiredTypesMap
	removeUndesiredMDEVs(typesToKeepMap)

	err := m.discoverConfigurableMDEVTypes(desiredTypesMap)
	if err != nil {
		log.Log.Reason(err).Error("failed to discover which mdev types are available for configuration")
		return false, err
	}

	if len(desiredTypesMap) > 0 {
		m.configureDesiredMDEVTypes()
	}

	return true, nil
}

// discoverConfigurableMDEVTypes will create an intersection of desired and configurable available mdev types
func (m *MDEVTypesManager) discoverConfigurableMDEVTypes(desiredTypesMap map[string]struct{}) error {
	// initialize unconfigured parents map
	m.unconfiguredParentsMap = make(map[string]struct{})

	// a map of mdev providers that already have configured mdevs
	existingMdevProviders, err := m.getAlreadyConfiguredMdevParents()
	if err != nil {
		return err
	}

	files, err := filepath.Glob(mdevClassBusPath + "/**/mdev_supported_types/*")
	if err != nil {
		return err
	}

	for _, file := range files {

		filePathParts := strings.Split(file, string(os.PathSeparator))
		if len(filePathParts) < 5 {
			return fmt.Errorf("invalid device path: %s", file)
		}
		parentID := filePathParts[len(filePathParts)-3]

		//find the type's name
		rawName, err := os.ReadFile(filepath.Join(file, "name"))
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}

		// The name usually contain spaces which should be replaced with _
		typeNameStr := strings.Replace(string(rawName), " ", "_", -1)
		typeNameStr = strings.TrimSpace(typeNameStr)

		// get this type's ID
		typeID := filepath.Base(file)

		// find out if type was requested by name
		_, typeNameExist := desiredTypesMap[typeNameStr]
		_, typeIDExist := desiredTypesMap[typeID]
		if typeNameExist || typeIDExist {
			ar, exist := m.availableMdevTypesMap[typeID]
			if !exist {
				ar = []string{}
			}

			if _, exist := existingMdevProviders[parentID]; !exist {
				ar = append(ar, parentID)
				m.availableMdevTypesMap[typeID] = ar
				m.unconfiguredParentsMap[parentID] = struct{}{}
			}
		}
	}
	return nil
}

func (m *MDEVTypesManager) initMDEVTypesRing() *ring.Ring {
	// Create a new ring of size of availableMdevTypesMap
	r := ring.New(len(m.availableMdevTypesMap))

	// Initialize the ring with some integer values
	for desiredType := range m.availableMdevTypesMap {
		r.Value = desiredType
		r = r.Next()
	}
	return r
}

func (m *MDEVTypesManager) getNextAvailableParentToConfigure(parents []string) (string, []string) {
	for idx := 0; idx < len(parents); idx++ {
		parent := parents[idx]
		if _, exist := m.unconfiguredParentsMap[parent]; exist {
			return parent, parents[idx+1:]
		}
	}
	return "", []string{}
}

func (m *MDEVTypesManager) configureDesiredMDEVTypes() {
	r := m.initMDEVTypesRing()

	if r.Len() == 0 {
		return
	}

	// Iterate over the ring and configure the relevant mdev types
	for {
		mdevTypeToConfigure := r.Value.(string)
		if parents, exist := m.availableMdevTypesMap[mdevTypeToConfigure]; exist {
			if len(parents) > 0 {
				// Currently, we can configure only one mdev type per card.
				// Find the next available parent to congigure and remove the
				// configured parents from the list.
				parent, remainingParents := m.getNextAvailableParentToConfigure(parents)
				parents = remainingParents
				if parent != "" {
					if err := createMdevTypes(mdevTypeToConfigure, parent); err == nil {
						m.availableMdevTypesMap[mdevTypeToConfigure] = remainingParents
						// remove the already configured parent
						delete(m.unconfiguredParentsMap, parent)
					}
				}
			}
			if len(parents) == 0 {
				delete(m.availableMdevTypesMap, mdevTypeToConfigure)
			}
		}

		// all requested mdev types has been configured. We can exist now.
		if len(m.availableMdevTypesMap) == 0 || len(m.unconfiguredParentsMap) == 0 {
			break
		}
		r = r.Next()
	}
}

func createMdevTypes(mdevType string, parentID string) error {
	instances, err := Handler.ReadMDEVAvailableInstances(mdevType, parentID)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create mdevs of type %s, failed to obtain number of instances", mdevType)
		return err
	}
	// create mdevs for all available instances
	for i := 0; i < instances; i++ {
		err := Handler.CreateMDEVType(mdevType, parentID)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to create mdevs of type %s", mdevType)
			return err
		}
	}
	return nil
}

func shouldRemoveMDEV(mdevUUID string, desiredTypesMap map[string]struct{}) bool {

	if rawName, err := os.ReadFile(filepath.Join(mdevBasePath, mdevUUID, "mdev_type/name")); err == nil {
		typeNameStr := strings.Replace(string(rawName), " ", "_", -1)
		typeNameStr = strings.TrimSpace(typeNameStr)
		if _, exist := desiredTypesMap[typeNameStr]; exist {
			return false
		}
	}

	originFile, err := os.Readlink(filepath.Join(mdevBasePath, mdevUUID, "mdev_type"))
	if err != nil {
		return false
	}
	rawName := []byte(filepath.Base(originFile))

	// The name usually contain spaces which should be replaced with _
	typeNameStr := strings.Replace(string(rawName), " ", "_", -1)
	typeNameStr = strings.TrimSpace(typeNameStr)
	if _, exist := desiredTypesMap[typeNameStr]; exist {
		return false
	}
	return true
}

func removeUndesiredMDEVs(desiredTypesMap map[string]struct{}) {
	files, err := os.ReadDir(mdevBasePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Log.Reason(err).Errorf("failed to remove mdev types: failed to read the content of %s directory", mdevBasePath)
		} else {
			log.Log.Reason(err).V(4).Infof("failed to remove mdev types: failed to read the content of %s directory. This most likely means that no mdev cleanup is necessary", mdevBasePath)
		}
		return
	}
	for _, file := range files {
		if shouldRemoveMDEV(file.Name(), desiredTypesMap) {
			err = Handler.RemoveMDEVType(file.Name())
			log.Log.Reason(err).Warningf("failed to remove mdev type: %s", file.Name())
		}
	}
}
