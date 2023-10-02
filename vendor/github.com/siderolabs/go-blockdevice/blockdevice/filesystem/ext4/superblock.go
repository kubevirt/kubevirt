// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package ext4

import (
	"unsafe"
)

const (
	// Magic is the ext magic signature.
	Magic = 0xef53
)

// SuperBlock represents the ext filesystem super block.
//
// See https://ext4.wiki.kernel.org/index.php/Ext4_Disk_Layout#The_Super_Block for the reference.
type SuperBlock struct {
	_     [0x38]uint8
	Magic [2]uint8
	_     [0x3e]uint8
	Label [0x10]uint8
	_     [0x378]uint8
}

// Is implements the SuperBlocker interface.
func (sb *SuperBlock) Is() bool {
	return sb.Magic[1] == Magic>>8 && sb.Magic[0] == Magic&0xff
}

// Offset implements the SuperBlocker interface.
func (sb *SuperBlock) Offset() int64 {
	return 0x400
}

// Type implements the SuperBlocker interface.
func (sb *SuperBlock) Type() string {
	return "ext4"
}

// Encrypted implements the SuperBlocker interface.
func (sb *SuperBlock) Encrypted() bool {
	return false
}

func init() {
	if unsafe.Sizeof(SuperBlock{}) != 0x400 {
		panic("ext4: SuperBlock size is not 0x400")
	}
}
