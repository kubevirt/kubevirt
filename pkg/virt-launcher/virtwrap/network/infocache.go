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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package network

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/util"
)

const networkInfoDir = util.VirtPrivateDir + "/network-info-cache"
const VirtHandlerCachePattern = networkInfoDir + "/%s/%s"

var virtLauncherCachedPattern = "/proc/%s/root/var/run/kubevirt-private/interface-cache-%s.json"

func CreateVirtHandlerNetworkInfoCache() error {
	return os.MkdirAll(networkInfoDir, 0755)
}

func CreateVirtHandlerCacheDir(vmiuid types.UID) error {
	return os.MkdirAll(filepath.Join(networkInfoDir, string(vmiuid)), 0755)
}

func RemoveVirtHandlerCacheDir(vmiuid types.UID) error {
	return os.RemoveAll(filepath.Join(networkInfoDir, string(vmiuid)))
}

func writeToCachedFile(obj interface{}, fileName string) error {
	buf, err := json.MarshalIndent(&obj, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling cached object: %v", err)
	}

	err = ioutil.WriteFile(fileName, buf, 0644)
	if err != nil {
		return fmt.Errorf("error writing cached object: %v", err)
	}
	return nil
}

func readFromCachedFile(obj interface{}, fileName string) error {
	buf, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = json.Unmarshal(buf, &obj)
	if err != nil {
		return fmt.Errorf("error unmarshaling cached object: %v", err)
	}
	return nil
}

func readFromVirtLauncherCachedFile(obj interface{}, pid, ifaceName string) error {
	fileName := getInterfaceCacheFile(virtLauncherCachedPattern, pid, ifaceName)
	return readFromCachedFile(obj, fileName)
}

func writeToVirtLauncherCachedFile(obj interface{}, pid, ifaceName string) error {
	fileName := getInterfaceCacheFile(virtLauncherCachedPattern, pid, ifaceName)
	return writeToCachedFile(obj, fileName)
}

func ReadFromVirtHandlerCachedFile(obj interface{}, vmiuid types.UID, ifaceName string) error {
	fileName := getInterfaceCacheFile(VirtHandlerCachePattern, string(vmiuid), ifaceName)
	return readFromCachedFile(obj, fileName)
}

func writeToVirtHandlerCachedFile(obj interface{}, vmiuid types.UID, ifaceName string) error {
	fileName := getInterfaceCacheFile(VirtHandlerCachePattern, string(vmiuid), ifaceName)
	return writeToCachedFile(obj, fileName)
}

func getInterfaceCacheFile(pattern, id, name string) string {
	return fmt.Sprintf(pattern, id, name)
}
