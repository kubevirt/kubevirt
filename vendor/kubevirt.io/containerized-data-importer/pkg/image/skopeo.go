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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/klog"
	"kubevirt.io/containerized-data-importer/pkg/system"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

const dataTmpDir string = "/data_tmp"
const whFilePrefix string = ".wh."

// SkopeoOperations defines the interface for executing skopeo subprocesses
type SkopeoOperations interface {
	CopyImage(string, string, string, string, string, bool) error
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
	// SkopeoInterface the skopeo operations interface
	SkopeoInterface = NewSkopeoOperations()
)

// NewSkopeoOperations returns the default implementation of SkopeoOperations
func NewSkopeoOperations() SkopeoOperations {
	return &skopeoOperations{}
}

func (o *skopeoOperations) CopyImage(url, dest, accessKey, secKey, certDir string, insecureRegistry bool) error {
	var err error
	args := []string{"copy", url, dest}
	if accessKey != "" && secKey != "" {
		creds := "--src-creds=" + accessKey + ":" + secKey
		args = append(args, creds)
	}
	if certDir != "" {
		klog.Infof("Using user specified TLS certs at %s", certDir)
		args = append(args, "--src-cert-dir="+certDir)
	} else if insecureRegistry {
		klog.Infof("Disabling TLS verification for URL %s", url)
		args = append(args, "--src-tls-verify=false")
	}
	_, err = skopeoExecFunction(nil, nil, "skopeo", args...)
	if err != nil {
		return errors.Wrap(err, "could not copy image")
	}
	return nil
}

// CopyRegistryImage download image from registry with skopeo
// url: source registry url.
// dest: the scratch space destination.
// accessKey: accessKey for the registry described in url.
// secKey: secretKey for the registry decribed in url.
// certDir: directory public CA keys are stored for registry identity verification
// insecureRegistry: boolean if true will allow insecure registries.
func CopyRegistryImage(url, dest, destFile, accessKey, secKey, certDir string, insecureRegistry bool) error {
	skopeoDest := "dir:" + filepath.Join(dest, dataTmpDir)

	// Copy to scratch space
	err := SkopeoInterface.CopyImage(url, skopeoDest, accessKey, secKey, certDir, insecureRegistry)
	if err != nil {
		os.RemoveAll(filepath.Join(dest, dataTmpDir))
		return errors.Wrap(err, "Failed to download from registry")
	}
	// Extract image layers to target space.
	err = extractImageLayers(dest, destFile)
	if err != nil {
		return errors.Wrap(err, "Failed to extract image layers")
	}

	//If a specifc file was requested verify it exists, if not - fail
	if len(destFile) > 0 {
		if _, err = os.Stat(filepath.Join(dest, destFile)); err != nil {
			klog.Errorf("Failed to find VM disk image file in the container image")
			err = errors.New("Failed to find VM disk image file in the container image")
		}
	}
	// Clean scratch space
	os.RemoveAll(filepath.Join(dest, dataTmpDir))

	return err
}

var extractImageLayers = func(dest string, arg ...string) error {
	klog.V(1).Infof("extracting image layers to %q\n", dest)
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
		filePath := filepath.Join(dest, dataTmpDir, layer)

		//prepend z option to the beggining of untar arguments
		args := append([]string{"z"}, arg...)

		if err := util.UnArchiveLocalTar(filePath, dest, args...); err != nil {
			//ignore errors if specific file extract was requested - we validate whether the file was extracted at the end of the sequence
			if len(arg) == 0 {
				return errors.Wrap(err, "could not extract layer tar")
			}
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
	err = json.Unmarshal(manifestFile, &manifestObj)
	if err != nil {
		return nil, errors.Wrap(err, "could not read manifest file")
	}
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
