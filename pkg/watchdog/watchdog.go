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

package watchdog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func WatchdogFileDirectory(baseDir string) string {
	return filepath.Join(baseDir, "watchdog-files")
}

func WatchdogFileFromNamespaceName(baseDir string, namespace string, name string) string {
	watchdogFile := namespace + "_" + name
	return filepath.Join(baseDir, "watchdog-files", watchdogFile)
}

// attempts to retrieve vmi uid from watchdog file if it exists
func WatchdogFileGetUID(baseDir string, vmi *v1.VirtualMachineInstance) string {
	namespace := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetName())

	filePath := WatchdogFileFromNamespaceName(baseDir, namespace, domain)
	// #nosec No risk for path injection. Using static path and base path of "virtShareDir"
	b, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}

	return string(b)
}

func WatchdogFileRemove(baseDir string, vmi *v1.VirtualMachineInstance) error {
	namespace := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetName())

	file := WatchdogFileFromNamespaceName(baseDir, namespace, domain)

	log.Log.V(3).Infof("Remove watchdog file %s", file)
	return diskutils.RemoveFilesIfExist(file)
}

func WatchdogFileUpdate(watchdogFile string, uid string) error {
	f, err := os.Create(watchdogFile)
	if err != nil {
		return err
	}
	_, err = f.WriteString(uid)
	if err != nil {
		return err
	}
	f.Close()

	return nil
}

func WatchdogFileExists(baseDir string, vmi *v1.VirtualMachineInstance) (bool, error) {
	namespace := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetName())

	filePath := WatchdogFileFromNamespaceName(baseDir, namespace, domain)
	exists, err := diskutils.FileExists(filePath)
	if err != nil {
		log.Log.Reason(err).Errorf("Error encountered while attempting to verify if watchdog file at path %s exists.", filePath)

		return false, err
	}
	return exists, nil
}

func WatchdogFileIsExpired(timeoutSeconds int, baseDir string, vmi *v1.VirtualMachineInstance) (bool, error) {
	return watchdogFileIsExpired(timeoutSeconds, baseDir, vmi, time.Now())
}

func watchdogFileIsExpired(timeoutSeconds int, baseDir string, vmi *v1.VirtualMachineInstance, timeNow time.Time) (bool, error) {
	namespace := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetName())

	filePath := WatchdogFileFromNamespaceName(baseDir, namespace, domain)

	exists, err := diskutils.FileExists(filePath)
	if err != nil {
		return false, err
	}

	if exists == false {
		return true, nil
	}

	stat, err := os.Stat(filePath)
	if err != nil {
		return false, err
	}

	now := timeNow.UTC().Unix()

	return isExpired(now, timeoutSeconds, stat), nil
}

func isExpired(now int64, timeoutSeconds int, stat os.FileInfo) bool {
	mod := stat.ModTime().UTC().Unix()
	diff := now - mod

	if diff > int64(timeoutSeconds) {
		return true
	}
	return false
}

func GetExpiredDomains(timeoutSeconds int, virtShareDir string) ([]*api.Domain, error) {
	return getExpiredDomains(timeoutSeconds, virtShareDir, time.Now())
}

func getExpiredDomains(timeoutSeconds int, virtShareDir string, timeNow time.Time) ([]*api.Domain, error) {

	var domains []*api.Domain

	fileDir := WatchdogFileDirectory(virtShareDir)

	exists, _ := diskutils.FileExists(fileDir)
	if !exists {
		return domains, nil
	}

	files, err := os.ReadDir(fileDir)
	if err != nil {
		return nil, err
	}
	now := timeNow.UTC().Unix()
	for _, file := range files {
		fileInfo, err := file.Info()
		if err != nil {
			return nil, err
		}

		if isExpired(now, timeoutSeconds, fileInfo) == true {
			key := file.Name()
			namespace, name, err := splitFileNamespaceName(key)
			if err != nil {
				log.Log.Reason(err).Errorf("Invalid key (%s) detected during watchdog tick, ignoring and continuing.", key)
				continue
			}
			domain := api.NewMinimalDomainWithNS(namespace, name)
			domains = append(domains, domain)
		}
	}

	return domains, nil
}

func splitFileNamespaceName(fullPath string) (namespace string, domain string, err error) {
	fileName := filepath.Base(fullPath)
	namespaceName := strings.Split(fileName, "_")
	if len(namespaceName) != 2 {
		return "", "", fmt.Errorf("Invalid file path: %s", fullPath)
	}

	namespace = namespaceName[0]
	domain = namespaceName[1]
	return namespace, domain, nil
}
