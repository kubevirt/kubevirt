// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package luks

import (
	"fmt"
)

const (
	// Magic1 is the first LUKS2 magic.
	Magic1 = "LUKS\xba\xbe"
	// Magic2 is the second LUKS2 magic.
	Magic2 = "SKUL\xba\xbe"
)

// SuperBlock represents luks encoded partition header.
type SuperBlock struct {
	Magic            [6]byte
	Version          uint16
	HeaderSize       uint64
	SeqID            uint64
	Label            [48]byte
	CSumAlg          [32]byte
	Salt             [64]byte
	UUID             [40]byte
	Subsystem        [48]byte
	SuperBlockOffset uint64
	_                [184]byte
	Checksum         [64]byte
	_                [7 * 512]byte
}

// Is implements the SuperBlocker interface.
func (sb *SuperBlock) Is() bool {
	magic := string(sb.Magic[:])

	return magic == Magic1 || magic == Magic2
}

// Offset implements the SuperBlocker interface.
func (sb *SuperBlock) Offset() int64 {
	return 0x0
}

// Type implements the SuperBlocker interface.
func (sb *SuperBlock) Type() string {
	return fmt.Sprintf("luks%d", sb.Version)
}

// Encrypted implements the SuperBlocker interface.
func (sb *SuperBlock) Encrypted() bool {
	return true
}
