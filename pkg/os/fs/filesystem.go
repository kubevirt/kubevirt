/*
Copyright 2017 The Kubernetes Authors.
Copyright 2021 The KubeVirt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

This file was originally copied from https://github.com/kubernetes/kubernetes/blob/e0a22acaa0c62f3e6f9dd37ab2a4e7d960528edc/pkg/util/filesystem/filesystem.go
*/
package fs

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Filesystem is an interface that we can use to mock various filesystem operations
type Fs interface {
	// from "os"
	Stat(name string) (os.FileInfo, error)
	Create(name string) (File, error)
	Rename(oldpath, newpath string) error
	MkdirAll(path string, perm os.FileMode) error
	Chtimes(name string, atime time.Time, mtime time.Time) error
	RemoveAll(path string) error
	Remove(name string) error

	// from "io/ioutil"
	ReadFile(filename string) ([]byte, error)
	WriteFile(filename string, data []byte, perm fs.FileMode) error
	TempDir(dir, prefix string) (string, error)
	TempFile(dir, prefix string) (File, error)
	ReadDir(dirname string) ([]os.FileInfo, error)
	Walk(root string, walkFn filepath.WalkFunc) error
}

// File is an interface that we can use to mock various filesystem operations typically
// accessed through the File object from the "os" package
type File interface {
	// for now, the only os.File methods used are those below, add more as necessary
	Name() string
	Write(b []byte) (n int, err error)
	Sync() error
	Close() error
}
