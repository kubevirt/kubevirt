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
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

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

type TempCacheCreator struct {
	once   sync.Once
	tmpDir string
}

func (c *TempCacheCreator) New(filePath string) *Cache {
	c.once.Do(func() {
		tmpDir, err := ioutil.TempDir("", "temp-cache")
		if err != nil {
			panic("Unable to create temp cache directory")
		}
		c.tmpDir = tmpDir
	})
	return NewCustomCache(filePath, kfs.NewWithRootPath(c.tmpDir))
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
