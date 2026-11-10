// Copyright The Prometheus Authors
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

package sysfs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/prometheus/procfs/internal/util"
)

// ClassThermalZoneStats contains info from files in /sys/class/thermal/thermal_zone<zone>
// for a single <zone>.
// https://www.kernel.org/doc/Documentation/thermal/sysfs-api.txt
type ClassThermalZoneStats struct {
	Name    string  // The name of the zone from the directory structure.
	Type    string  // The type of thermal zone.
	Temp    int64   // Temperature in millidegree Celsius.
	Policy  string  // One of the various thermal governors used for a particular zone.
	Mode    *bool   // Optional: One of the predefined values in [enabled, disabled].
	Passive *uint64 // Optional: millidegrees Celsius. (0 for disabled, > 1000 for enabled+value)

	// ReadError contains any errors returned when gathering data.
	ReadErrors error
}

// ClassThermalZoneStats returns Thermal Zone metrics for all zones.
func (fs FS) ClassThermalZoneStats() ([]ClassThermalZoneStats, error) {
	zones, err := filepath.Glob(fs.sys.Path("class/thermal/thermal_zone[0-9]*"))
	if err != nil {
		return nil, err
	}

	stats := make([]ClassThermalZoneStats, 0, len(zones))
	for _, zone := range zones {
		zoneStats := parseClassThermalZone(zone)
		zoneStats.Name = strings.TrimPrefix(filepath.Base(zone), "thermal_zone")
		stats = append(stats, zoneStats)
	}
	return stats, nil
}

func parseClassThermalZone(zone string) ClassThermalZoneStats {
	var errs []error
	// Required attributes.
	zoneType, err := util.SysReadFile(filepath.Join(zone, "type"))
	if err != nil {
		errs = append(errs, fmt.Errorf("error reading type: %w", err))
	}
	zonePolicy, err := util.SysReadFile(filepath.Join(zone, "policy"))
	if err != nil {
		errs = append(errs, fmt.Errorf("error reading policy: %w", err))
	}
	zoneTemp, err := util.SysReadIntFromFile(filepath.Join(zone, "temp"))
	if err != nil && !errors.Is(err, os.ErrInvalid) {
		errs = append(errs, fmt.Errorf("error reading temp: %w", err))
	}

	// Optional attributes.
	mode, err := util.SysReadFile(filepath.Join(zone, "mode"))
	if err != nil && !os.IsNotExist(err) && !os.IsPermission(err) {
		errs = append(errs, fmt.Errorf("error reading mode: %w", err))
	}
	zoneMode := util.ParseBool(mode)

	var zonePassive *uint64
	passive, err := util.SysReadUintFromFile(filepath.Join(zone, "passive"))
	switch {
	case os.IsNotExist(err), os.IsPermission(err):
		zonePassive = nil
	case err != nil:
		errs = append(errs, fmt.Errorf("error reading passive: %w", err))
	default:
		zonePassive = &passive
	}

	return ClassThermalZoneStats{
		Type:       zoneType,
		Policy:     zonePolicy,
		Temp:       zoneTemp,
		Mode:       zoneMode,
		Passive:    zonePassive,
		ReadErrors: errors.Join(errs...),
	}
}
