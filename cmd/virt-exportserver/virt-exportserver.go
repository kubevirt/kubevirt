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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package main

import (
	"crypto/tls"
	"encoding/json"
	"os"
	"strconv"
	"time"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/service"

	"kubevirt.io/kubevirt/pkg/storage/export/export"
	exportServer "kubevirt.io/kubevirt/pkg/storage/export/virt-exportserver"
)

const (
	listenAddr = ":8443"
)

func main() {
	log.InitializeLogging("virt-exportserver-" + os.Getenv("POD_NAME"))
	log.Log.Info("Starting export server")

	certFile, keyFile := getCert()
	config := exportServer.ExportServerConfig{
		CertFile:        certFile,
		KeyFile:         keyFile,
		Deadline:        getDeadline(),
		ListenAddr:      getListenAddr(),
		TokenFile:       getTokenFile(),
		Paths:           export.CreateServerPaths(export.EnvironToMap()),
		TLSMinVersion:   getTLSMinVersion(),
		TLSCipherSuites: getTLSCipherSuites(),
	}
	if len(config.Paths.Backups) > 0 {
		config.BackupUID = getBackupUID()
		config.BackupType = getBackupType()
		config.BackupCheckpoint = getBackupCheckpoint()
		config.BackupCACert = getBackupCACert()
	}
	server := exportServer.NewExportServer(config)
	service.Setup(server)
	server.Run()
}

func getTokenFile() string {
	tokenFile := os.Getenv("TOKEN_FILE")
	if tokenFile == "" {
		panic("no token file set")
	}
	return tokenFile
}

func getCert() (certFile, keyFile string) {
	certFile = os.Getenv("CERT_FILE")
	keyFile = os.Getenv("KEY_FILE")
	if certFile == "" || keyFile == "" {
		panic("TLS config incomplete")
	}
	return
}

func getListenAddr() string {
	addr := os.Getenv("LISTEN_ADDR")
	if addr != "" {
		return addr
	}
	return listenAddr
}

func getDeadline() (result time.Time) {
	dl := os.Getenv("DEADLINE")
	if dl != "" {
		var err error
		result, err = time.Parse(time.RFC3339, dl)
		if err != nil {
			panic("Invalid Deadline")
		}
	}
	return
}

func getBackupUID() string {
	backupUID := os.Getenv("BACKUP_UID")
	if backupUID == "" {
		panic("backup export but not backup UID provided")
	}
	return backupUID
}

func getBackupType() string {
	backupType := os.Getenv("BACKUP_TYPE")
	if backupType == "" {
		panic("backup export but no backup type provided")
	}
	return backupType
}

func getBackupCheckpoint() string {
	checkpointName := os.Getenv("BACKUP_CHECKPOINT")
	return checkpointName
}

func getBackupCACert() []byte {
	caCert := os.Getenv("BACKUP_CACERT")
	if caCert == "" {
		panic("backup export but no backup CA provided")
	}
	return []byte(caCert)
}

func getTLSMinVersion() uint16 {
	env := os.Getenv("TLS_MIN_VERSION")
	if env == "" {
		return tls.VersionTLS12
	}
	v, err := strconv.ParseUint(env, 10, 16)
	if err != nil {
		log.Log.Warningf("Failed to parse TLS_MIN_VERSION %q: %v", env, err)
		return tls.VersionTLS12
	}
	return uint16(v)
}

func getTLSCipherSuites() []uint16 {
	env := os.Getenv("TLS_CIPHER_SUITES")
	if env == "" {
		return nil
	}
	var ids []uint16
	if err := json.Unmarshal([]byte(env), &ids); err != nil {
		log.Log.Warningf("Failed to parse TLS_CIPHER_SUITES: %v", err)
		return nil
	}
	return ids
}
