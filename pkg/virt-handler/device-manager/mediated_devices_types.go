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
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
    "k8s.io/apimachinery/pkg/util/uuid"

	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/util"
)

type MDEVTypesManager struct {
    availableMdevTypesMap  map[string][]string
	mdevsConfigurationMutex sync.Mutex
    configuredMdevTypes    []string
}

func NewMDEVTypesManager() *MDEVTypesManager {
    return &MDEVTypesManager{
        availableMdevTypesMap: make(map[string][]string)
    }
}
// Not a const for static test purposes
var mdevBasePath string = "/sys/bus/mdev/devices"



func (m *MDEVTypesManager) updateMDEVTypesConfiguration(desiredTypesList  []string) error {
    if bytes.Compare(m.configuredMdevTypes, desiredTypesList) != 0 {
        // construct a map of desired types
        desiredTypesMap := make(map[string]struct{})
        for _, mdevType := range desiredTypesList {
            desiredTypesMap[mdevType] = struct{}{}
        }

        c.mdevsConfigurationMutex.Lock()
        defer c.mdevsConfigurationMutex.Unlock()

        removeUndesiredMDEVs(desiredTypesMap)
        err := m.discoverConfigurableMDEVTypes(desiredTypesMap)
        if err != nil {
            log.Log.Reason(err).Error("failed to discover which mdev types are available for configuration")
            return err
        }
        m.configureDesiredMDEVTypes()
        // store the configured list of types
        m.configuredMdevTypes = desiredTypesList
    }
}

func (m *MDEVTypesManager) discoverConfigurableMDEVTypes(desiredTypesMap  map[string]struct{}) error {
	files, err := filepath.Glob("/sys/class/mdev_bus/**/mdev_supported_types/*")

	if err != nil {
		return err
	}

	for _, file := range files {

		filePathParts := strings.Split(file, string(os.PathSeparator))
        parentID := filePathParts[len(filePathParts)-3]
		rawName, err := ioutil.ReadFile(filepath.Join(file, "name"))
		if err != nil {
			if !os.IsNotExist(err) {
                return err
			}
		}

		// The name usually contain spaces which should be replaced with _
		typeNameStr := strings.Replace(string(rawName), " ", "_", -1)
		typeNameStr = strings.TrimSpace(typeNameStr)
		typeID := filepath.Base(file)
		 _, typeNameExist := desiredTypesMap[typeNameStr]
		 _, typeIDExist := desiredTypesMap[typeID]
		if  typeNameExist || typeIDExist {
			ar, exist := m.availableMdevTypesMap[typeID]
			if !exist {
				ar = []string{}
			}
			ar = append(ar, parentID)
			m.availableMdevTypesMap[typeID] = ar
        }
	}
    return nil
}

func (m *MDEVTypesManager) initMDEVTypesRing() *Ring {
	// create a ring out of intersection of desired and available types.
	// Create a new ring of size of availableMdevTypesMap
	r := ring.New(len(m.availableMdevTypesMap))

	// Initialize the ring with some integer values
	for desiredType, _ := range m.availableMdevTypesMap {
		r.Value = desiredType
		r = r.Next()
	}
    return r
}

func (m *MDEVTypesManager) configureDesiredMDEVTypes() {
    r := m.initMDEVTypesRing()

	// Iterate through the ring and configure the relevant types
	for {
		if parents, exist := m.availableMdevTypesMap[r.Value.(string)]; exist {

			parent := parents[0]
            createMdevType(r.Value.(string), parent)
            // figure out what to do with errors here
			// debug log fmt.Println("Configuring: ", r.Value.(string), " - parent: ", parent)
			if len(parents) > 0 {
				parents = append(parents[:0], parents[1:]...)
				m.availableMdevTypesMap[r.Value.(string)] = parents
				if len(parents) == 0 {
					delete(m.availableMdevTypesMap, r.Value.(string))
				}
			}
		}

		if len(m.availableMdevTypesMap) == 0 {
			break
		}
		r = r.Next()
	}
}

func createMdevType(mdevType string, parentID string) error {
    uid := uuid.NewUUID()

    path := filepath.Join(pciBasePath, parentID,"mdev_supported_types", mdevType, "create")

    f, err := os.OpenFile(path, os.O_WRONLY, 0200)
    if err != nil {
        //log 
        return err
    }

    defer f.Close()

    if _, err = f.WriteString(uid); err != nil {
        //log
        return err
    }
}

func removeMdevsByType(mdevType) error {
	filepath.Walk(mdevBasePath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		fmt.Println("file : ", info.Name())
		path1 := filepath.Join(mdevBasePath, info.Name(), "remove")

		f, err := os.OpenFile(path1, os.O_WRONLY, 0200)
		if err != nil {
            // log
			return err
		}

		defer f.Close()

		if _, err = f.WriteString("1"); err != nil {
            //log
			return err
		}
		return nil
	})
	return nil

}

shouldRemoveMDEV(mdevUUID string, desiredTypesMap  map[string]struct{}) bool {

	if rawName, err := ioutil.ReadFile(filepath.Join(mdevBasePath, mdevUUID, "mdev_type/name")); err == nil {
        typeNameStr := strings.Replace(string(rawName), " ", "_", -1)
        if _, exist := desiredTypesMap[typeNameStr]; exist {
            return false
        }
    }

    originFile, err := os.Readlink(filepath.Join(mdevBasePath, mdevUUID, "mdev_type"))
    if err != nil {
        // log
        return false
    }
    rawName = []byte(filepath.Base(originFile))

	// The name usually contain spaces which should be replaced with _
	typeNameStr := strings.Replace(string(rawName), " ", "_", -1)
    if _, exist := desiredTypesMap[typeNameStr]; exist {
        return false
    }
	return true
}

removeUndesiredMDEVs(desiredTypesMap  map[string]struct{}) {
	filepath.Walk(mdevBasePath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
        if shouldRemoveMDEV(info.Name()) {
		    removePath := filepath.Join(mdevBasePath, info.Name(), "remove")

            f, err := os.OpenFile(removePath, os.O_WRONLY, 0200)
            if err != nil {
                // log
                return err
            }

            defer f.Close()

            if _, err = f.WriteString("1"); err != nil {
                //log
                return err
            }
        }
		return nil
	})
	return nil
}


