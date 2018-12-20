/*
Copyright 2018 The CDI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package image

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"kubevirt.io/containerized-data-importer/pkg/system"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

const dataTmpDir string = "/data_tmp"
const whFilePrefix string = ".wh."

// SkopeoOperations defines the interface for executing skopeo subprocesses
type SkopeoOperations interface {
	CopyImage(string, string, string, string) error
}

type skopeoOperations struct{}

type manifest struct {
	SchemaVersion int     `json:"schemaVersion"`
	Layers        []layer `json:"layers"`   // schemaVersion v2
	FsLayers      []layer `json:"fsLayers"` // schemaVersion v1
}
type layer struct {
	Digest  string `json:"digest"`  // schemaVersion v2
	BlobSum string `json:"blobSum"` // schemaVersion v1
}

var (
	skopeoExecFunction = system.ExecWithLimits
	processLimits      = &system.ProcessLimitValues{AddressSpaceLimit: maxMemory, CPUTimeLimit: maxCPUSecs}
	// SkopeoInterface the skopeo operations interface
	SkopeoInterface = NewSkopeoOperations()
)

// NewSkopeoOperations returns the default implementation of SkopeoOperations
func NewSkopeoOperations() SkopeoOperations {
	return &skopeoOperations{}
}

func (o *skopeoOperations) CopyImage(url, dest, accessKey, secKey string) error {
	var err error
	if len(accessKey) > 0 && len(secKey) > 0 {
		var creds = "--src-creds=" + accessKey + ":" + secKey
		_, err = skopeoExecFunction(processLimits, nil, "skopeo", "copy", url, dest, creds)
	} else {
		_, err = skopeoExecFunction(processLimits, nil, "skopeo", "copy", url, dest, "--src-tls-verify=false")
	}
	if err != nil {
		return errors.Wrap(err, "could not copy image")
	}
	return nil
}

// CopyRegistryImage download image from registry with skopeo
func CopyRegistryImage(url, dest, accessKey, secKey string) error {
	skopeoDest := "dir:" + dest + dataTmpDir
	err := SkopeoInterface.CopyImage(url, skopeoDest, accessKey, secKey)
	if err != nil {
		os.RemoveAll(dest + dataTmpDir)
		return errors.Wrap(err, "Failed to download from registry")
	}
	err = extractImageLayers(dest)
	if err != nil {
		return errors.Wrap(err, "Failed to extract image layers")
	}

	// Clean temp folder
	os.RemoveAll(dest + dataTmpDir)

	return err
}

var extractImageLayers = func(dest string) error {
	glog.V(1).Infof("extracting image layers to %q\n", dest)
	// Parse manifest file
	manifest, err := getImageManifest(dest + dataTmpDir)
	if err != nil {
		return err
	}

	// Extract layers
	var layers []layer
	if manifest.SchemaVersion == 1 {
		layers = manifest.FsLayers
	} else {
		layers = manifest.Layers
	}
	for _, m := range layers {
		var layerID string
		if manifest.SchemaVersion == 1 {
			layerID = m.BlobSum
		} else {
			layerID = m.Digest
		}
		layer := strings.TrimPrefix(layerID, "sha256:")
		filePath := fmt.Sprintf("%s%s/%s", dest, dataTmpDir, layer)

		if err := util.UnArchiveLocalTar(filePath, dest, "z"); err != nil {
			return errors.Wrap(err, "could not extract layer tar")
		}
		err = cleanWhiteoutFiles(dest)
	}
	return err
}

func getImageManifest(dest string) (*manifest, error) {
	// Open Manifest.json
	manifestFile, err := ioutil.ReadFile(dest + "/manifest.json")
	if err != nil {
		return nil, errors.Wrap(err, "could not read manifest file")
	}

	// Parse json file
	var manifestObj manifest
	json.Unmarshal(manifestFile, &manifestObj)
	return &manifestObj, nil
}

func cleanWhiteoutFiles(dest string) error {
	whFiles, err := getWhiteoutFiles(dest)
	if err != nil {
		return err
	}

	for _, path := range whFiles {
		os.RemoveAll(path)
		os.RemoveAll(strings.Replace(path, whFilePrefix, "", 1))
	}
	return nil
}

func getWhiteoutFiles(dest string) ([]string, error) {
	var whFiles []string
	err := filepath.Walk(dest,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.Wrapf(err, "Failed reading path: %s", path)
			}
			if strings.HasPrefix(info.Name(), whFilePrefix) {
				whFiles = append(whFiles, path)
			}
			return nil
		})

	if err != nil {
		return nil, errors.Wrapf(err, "Failed traversing directory: %s", dest)
	}
	return whFiles, nil
}
