// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package msdos

import (
	"bytes"
)

const (
	// Magic12 is the FAT12 magic signature.
	Magic12 = "FAT12"
	// Magic16 is the FAT16 magic signature.
	Magic16 = "FAT16"
)

// SuperBlock represents the vfat super block.
//
// See https://en.wikipedia.org/wiki/Design_of_the_FAT_file_system#Extended_BIOS_Parameter_Block for the reference.
type SuperBlock struct {
	_     [0x2b]uint8
	Label [11]uint8
	Magic [8]uint8
	_     [0x1cd]uint8
}

// Is implements the SuperBlocker interface.
func (sb *SuperBlock) Is() bool {
	trimmed := bytes.Trim(sb.Magic[:], " ")

	return bytes.Equal(trimmed, []byte(Magic12)) || bytes.Equal(trimmed, []byte(Magic16))
}

// Offset implements the SuperBlocker interface.
func (sb *SuperBlock) Offset() int64 {
	return 0x0
}

// Type implements the SuperBlocker interface.
func (sb *SuperBlock) Type() string {
	// using `vfat` here, as it's the filesystem type in Linux
	return "vfat"
}

// Encrypted implements the SuperBlocker interface.
func (sb *SuperBlock) Encrypted() bool {
	return false
}
