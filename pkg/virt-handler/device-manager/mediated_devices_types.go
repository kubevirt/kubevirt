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

package device_manager

import (
	"container/ring"
	"encoding/json"
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

// managedMdevTypesStateFile records the mdev types virt-handler last configured
// on this node. It survives virt-handler restarts (hostPath /var/run/kubevirt)
// so KubeVirt-created types can still be garbage-collected after a crash, but
// is lost on node reboot - which is fine, because sysfs mdevs are too.
// Not a const for static test purposes.
var managedMdevTypesStateFile = "/var/run/kubevirt/managed-mdev-types"

type MDEVTypesManager struct {
	availableMdevTypesMap   map[string][]string
	unconfiguredParentsMap  map[string]struct{}
	mdevsConfigurationMutex sync.Mutex
	// previouslyManagedTypes is the set of mdev types this manager most
	// recently configured. Removal is limited to this set so that mdevs
	// created by an external provider (DRA, GPU operator, vfio-ap, ...)
	// are not destroyed.
	previouslyManagedTypes map[string]struct{}
}

func NewMDEVTypesManager() *MDEVTypesManager {
	m := &MDEVTypesManager{
		availableMdevTypesMap:  make(map[string][]string),
		previouslyManagedTypes: make(map[string]struct{}),
	}
	m.loadManagedTypes()
	return m
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

	typesToKeepMap := make(map[string]struct{})
	for key := range externallyProvidedTypesMap {
		addMdevTypeKey(typesToKeepMap, key)
	}

	desiredTypesMap := make(map[string]struct{})
	for _, mdevType := range desiredTypesList {
		addMdevTypeKey(desiredTypesMap, mdevType)
		addMdevTypeKey(typesToKeepMap, mdevType)
	}

	// Remove only types this manager previously configured. Unknown types
	// (for example vfio-ap mdevs created by a DRA driver) are left alone.
	removeUndesiredMDEVs(typesToKeepMap, m.previouslyManagedTypes)

	err := m.discoverConfigurableMDEVTypes(desiredTypesMap)
	if err != nil {
		log.Log.Reason(err).Error("failed to discover which mdev types are available for configuration")
		m.setPreviouslyManagedTypes(mergeManagedTypes(desiredTypesMap, remainingManagedTypes(m.previouslyManagedTypes)))
		return false, err
	}

	if len(desiredTypesMap) > 0 {
		m.configureDesiredMDEVTypes()
	}

	m.setPreviouslyManagedTypes(mergeManagedTypes(desiredTypesMap, remainingManagedTypes(m.previouslyManagedTypes)))
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
		typeNameStr := strings.ReplaceAll(string(rawName), " ", "_")
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
	instances, err := handler.ReadMDEVAvailableInstances(mdevType, parentID)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create mdevs of type %s, failed to obtain number of instances", mdevType)
		return err
	}
	// create mdevs for all available instances
	for i := 0; i < instances; i++ {
		err := handler.CreateMDEVType(mdevType, parentID)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to create mdevs of type %s", mdevType)
			return err
		}
	}
	return nil
}

func mergeManagedTypes(sets ...map[string]struct{}) map[string]struct{} {
	merged := make(map[string]struct{})
	for _, set := range sets {
		for key, val := range set {
			merged[key] = val
		}
	}
	return merged
}

func remainingManagedTypes(previouslyManaged map[string]struct{}) map[string]struct{} {
	// Keep types that we failed to remove so the next reconcile retries,
	// without ever adopting unknown/external mdevs.
	remaining := make(map[string]struct{})
	if len(previouslyManaged) == 0 {
		return remaining
	}
	files, err := os.ReadDir(mdevBasePath)
	if err != nil {
		for key, val := range previouslyManaged {
			remaining[key] = val
		}
		return remaining
	}
	for _, file := range files {
		typeName, typeID := mdevTypeNameAndID(file.Name())
		if mdevTypeInSet(typeName, typeID, previouslyManaged) {
			addMdevTypeKey(remaining, typeName)
			addMdevTypeKey(remaining, typeID)
		}
	}
	return remaining
}

func addMdevTypeKey(types map[string]struct{}, mdevType string) {
	if mdevType == "" {
		return
	}
	types[mdevType] = struct{}{}
	if normalized := removeSelectorSpaces(mdevType); normalized != mdevType {
		types[normalized] = struct{}{}
	}
}

func mdevTypeInSet(typeName, typeID string, types map[string]struct{}) bool {
	if typeName != "" {
		if _, exist := types[typeName]; exist {
			return true
		}
	}
	if typeID != "" {
		if _, exist := types[typeID]; exist {
			return true
		}
	}
	return false
}

func mdevTypeNameAndID(mdevUUID string) (typeName, typeID string) {
	if rawName, err := os.ReadFile(filepath.Join(mdevBasePath, mdevUUID, "mdev_type/name")); err == nil {
		typeName = strings.TrimSpace(strings.ReplaceAll(string(rawName), " ", "_"))
	}

	originFile, err := os.Readlink(filepath.Join(mdevBasePath, mdevUUID, "mdev_type"))
	if err != nil {
		return typeName, typeID
	}
	typeID = strings.TrimSpace(strings.ReplaceAll(filepath.Base(originFile), " ", "_"))
	return typeName, typeID
}

// shouldRemoveMDEV reports whether an existing mdev should be destroyed.
// Types currently desired or marked as externally provided are kept.
// Types this manager previously configured and that are no longer desired
// are removed. Any other type is assumed to belong to an external owner
// and is left untouched.
func shouldRemoveMDEV(mdevUUID string, typesToKeep, previouslyManaged map[string]struct{}) bool {
	typeName, typeID := mdevTypeNameAndID(mdevUUID)
	if typeName == "" && typeID == "" {
		return false
	}
	if mdevTypeInSet(typeName, typeID, typesToKeep) {
		return false
	}
	return mdevTypeInSet(typeName, typeID, previouslyManaged)
}

func removeUndesiredMDEVs(typesToKeep, previouslyManaged map[string]struct{}) {
	if len(previouslyManaged) == 0 {
		return
	}
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
		if !shouldRemoveMDEV(file.Name(), typesToKeep, previouslyManaged) {
			continue
		}
		if err := handler.RemoveMDEVType(file.Name()); err != nil {
			log.Log.Reason(err).Warningf("failed to remove mdev type: %s", file.Name())
		}
	}
}

func equalStringSets(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for key := range a {
		if _, exist := b[key]; !exist {
			return false
		}
	}
	return true
}

func (m *MDEVTypesManager) setPreviouslyManagedTypes(next map[string]struct{}) {
	if equalStringSets(m.previouslyManagedTypes, next) {
		return
	}
	m.previouslyManagedTypes = make(map[string]struct{}, len(next))
	for key, val := range next {
		m.previouslyManagedTypes[key] = val
	}
	m.persistManagedTypes()
}

func (m *MDEVTypesManager) loadManagedTypes() {
	if managedMdevTypesStateFile == "" {
		return
	}
	data, err := os.ReadFile(managedMdevTypesStateFile)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Log.Reason(err).Warningf("failed to read managed mdev types state file %s", managedMdevTypesStateFile)
		}
		return
	}
	var types []string
	if err := json.Unmarshal(data, &types); err != nil {
		log.Log.Reason(err).Warningf("failed to parse managed mdev types state file %s", managedMdevTypesStateFile)
		return
	}
	for _, mdevType := range types {
		addMdevTypeKey(m.previouslyManagedTypes, mdevType)
	}
}

func (m *MDEVTypesManager) persistManagedTypes() {
	if managedMdevTypesStateFile == "" {
		return
	}
	types := make([]string, 0, len(m.previouslyManagedTypes))
	for mdevType := range m.previouslyManagedTypes {
		types = append(types, mdevType)
	}
	data, err := json.Marshal(types)
	if err != nil {
		log.Log.Reason(err).Warning("failed to marshal managed mdev types")
		return
	}
	// Do not create the parent directory. virt-handler's share dir is a
	// volume mount; skipping when it is absent also keeps unit tests from
	// writing under /var/run/kubevirt.
	if _, err := os.Stat(filepath.Dir(managedMdevTypesStateFile)); err != nil {
		return
	}
	tmp := managedMdevTypesStateFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		log.Log.Reason(err).Warningf("failed to write managed mdev types state file %s", tmp)
		return
	}
	if err := os.Rename(tmp, managedMdevTypesStateFile); err != nil {
		log.Log.Reason(err).Warningf("failed to persist managed mdev types state file %s", managedMdevTypesStateFile)
		_ = os.Remove(tmp)
	}
}
