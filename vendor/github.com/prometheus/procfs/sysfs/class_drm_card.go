// Copyright 2024 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux
// +build linux

package sysfs

import (
	"fmt"
	"path/filepath"

	"github.com/prometheus/procfs/internal/util"
)

const drmClassPath = "class/drm"

// DRMCard contains info from files in /sys/class/drm for a
// single DRM Card device.
type DRMCard struct {
	Name   string
	Driver string
	Ports  map[string]DRMCardPort
}

// DRMCardPort contains info from files in
// /sys/class/drm/<Card>/<Card>-<Name>
// for a single port of one DRMCard device.
type DRMCardPort struct {
	Name    string
	Status  string
	DPMS    string
	Enabled string
}

// DRMCardClass is a collection of every Card device in
// /sys/class/drm.
//
// The map keys are the names of the InfiniBand devices.
type DRMCardClass map[string]DRMCard

// DRMCardClass returns infos for all DRM devices read from
// /sys/class/drm.
func (fs FS) DRMCardClass() (DRMCardClass, error) {

	cards, err := filepath.Glob(fs.sys.Path("class/drm/card[0-9]"))

	if err != nil {
		return nil, fmt.Errorf("failed to list DRM card ports at %q: %w", cards, err)
	}

	drmCardClass := make(DRMCardClass, len(cards))
	for _, c := range cards {
		card, err := fs.parseDRMCard(filepath.Base(c))
		if err != nil {
			return nil, err
		}

		drmCardClass[card.Name] = *card
	}

	return drmCardClass, nil
}

// Parse one DRMCard.
func (fs FS) parseDRMCard(name string) (*DRMCard, error) {
	path := fs.sys.Path(drmClassPath, name)
	card := DRMCard{Name: name}

	// Read the kernel module of the card
	cardDriverPath, err := filepath.EvalSymlinks(filepath.Join(path, "device/driver"))
	if err != nil {
		return nil, fmt.Errorf("failed to read driver: %w", err)
	}
	card.Driver = filepath.Base(cardDriverPath)

	portsPath, err := filepath.Glob(filepath.Join(path, filepath.Base(path)+"-*-*"))

	if err != nil {
		return nil, fmt.Errorf("failed to list DRM card ports at %q: %w", portsPath, err)
	}

	card.Ports = make(map[string]DRMCardPort, len(portsPath))
	for _, d := range portsPath {
		port, err := parseDRMCardPort(d)
		if err != nil {
			return nil, err
		}

		card.Ports[port.Name] = *port
	}

	return &card, nil
}

func parseDRMCardPort(port string) (*DRMCardPort, error) {
	portStatus, err := util.SysReadFile(filepath.Join(port, "status"))
	if err != nil {
		return nil, err
	}

	drmCardPort := DRMCardPort{Name: filepath.Base(port), Status: portStatus}

	portDPMS, err := util.SysReadFile(filepath.Join(port, "dpms"))
	if err != nil {
		return nil, err
	}

	drmCardPort.DPMS = portDPMS

	portEnabled, err := util.SysReadFile(filepath.Join(port, "enabled"))
	if err != nil {
		return nil, err
	}
	drmCardPort.Enabled = portEnabled

	return &drmCardPort, nil
}
