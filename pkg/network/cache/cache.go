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

package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	kfs "kubevirt.io/kubevirt/pkg/os/fs"
)

type Cache struct {
	path string
	fs   cacheFS
}

type cacheFS interface {
	Stat(name string) (os.FileInfo, error)
	MkdirAll(path string, perm os.FileMode) error
	RemoveAll(path string) error
	ReadFile(filename string) ([]byte, error)
	WriteFile(filename string, data []byte, perm fs.FileMode) error
}

type CacheCreator struct{}

func (_ CacheCreator) New(filePath string) *Cache {
	return NewCustomCache(filePath, kfs.New())
}

func NewCustomCache(path string, fs cacheFS) *Cache {
	return &Cache{path, fs}
}

func (c Cache) Entry(path string) (Cache, error) {
	fileInfo, err := c.fs.Stat(c.path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return Cache{}, fmt.Errorf("unable to define entry: %v", err)
		}
	} else if !fileInfo.IsDir() {
		return Cache{}, fmt.Errorf("unable to define entry: parent cache has an existing store")
	}

	return Cache{
		path: filepath.Join(c.path, path),
		fs:   c.fs,
	}, nil
}

func (c Cache) Read(data interface{}) (interface{}, error) {
	err := readFromCachedFile(c.fs, data, c.path)
	return data, err
}

func (c Cache) Write(data interface{}) error {
	return writeToCachedFile(c.fs, data, c.path)
}

func (c Cache) Delete() error {
	return c.fs.RemoveAll(c.path)
}

type cacheCreator interface {
	New(filePath string) *Cache
}

func writeToCachedFile(fs cacheFS, obj interface{}, fileName string) error {
	if err := fs.MkdirAll(filepath.Dir(fileName), 0o750); err != nil {
		return err
	}
	buf, err := json.MarshalIndent(&obj, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling cached object: %v", err)
	}

	err = fs.WriteFile(fileName, buf, 0o604)
	if err != nil {
		return fmt.Errorf("error writing cached object: %v", err)
	}
	return dutils.DefaultOwnershipManager.UnsafeSetFileOwnership(fileName)
}

func readFromCachedFile(fs cacheFS, obj interface{}, fileName string) error {
	buf, err := fs.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = json.Unmarshal(buf, &obj)
	if err != nil {
		return fmt.Errorf("error unmarshaling cached object: %v", err)
	}
	return nil
}
