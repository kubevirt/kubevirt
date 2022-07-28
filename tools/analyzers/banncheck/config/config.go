// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// TODO: Temporary fork of banncheck from https://github.com/google/go-safeweb/tree/eb79df54b8bc1910ac3bc3fc3328da7c0fb016e1/cmd/bancheck
package config

import (
	"encoding/json"
	"errors"
	"io/fs"
)

// Config represents a configuration passed to the linter.
type Config struct {
	Imports   []BannedAPI `json:"imports"`
	Functions []BannedAPI `json:"functions"`
}

// BannedAPI represents an identifier (e.g. import, function call) that should not be used.
type BannedAPI struct {
	Name       string      `json:"name"` // fully qualified identifier name
	Msg        string      `json:"msg"`  // additional information e.g. rationale for banning
	Exemptions []Exemption `json:"exemptions"`
}

// Exemption represents a location that should be exempted from checking for banned APIs.
type Exemption struct {
	Justification string `json:"justification"`
	AllowedPkg    string `json:"allowedPkg"` // Uses Go RegExp https://golang.org/pkg/regexp/syntax
}

// ReadConfigs reads banned APIs from all files.
func ReadConfigs(configFS fs.FS, files []string) (*Config, error) {
	var imports []BannedAPI
	var fns []BannedAPI

	for _, file := range files {
		config, err := readCfg(configFS, file)
		if err != nil {
			return nil, err
		}

		imports = append(imports, config.Imports...)
		fns = append(fns, config.Functions...)
	}

	return &Config{Imports: imports, Functions: fns}, nil
}

func readCfg(configFS fs.FS, filename string) (*Config, error) {
	f, err := openFile(configFS, filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return decodeCfg(f)
}

func openFile(configFS fs.FS, filename string) (fs.File, error) {
	file, err := configFS.Open(filename)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, errors.New("file is a directory")
	}
	return file, nil
}

func decodeCfg(f fs.File) (*Config, error) {
	var cfg Config
	err := json.NewDecoder(f).Decode(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
