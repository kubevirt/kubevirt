package red

// Common capabilities
const (
	CapabilityAuthSelection uint32 = 0
	CapabilityAuthSpice     uint32 = 1
	CapabilityAuthSASL      uint32 = 2
	CapabilityMiniHeader    uint32 = 3
)

// Main Channel capabilities
const (
	CapabilityMainSemiSeamlessMigrate  uint32 = 0
	CapabilityMainNameAndUUID          uint32 = 1
	CapabilityMainAgentConnectedTokens uint32 = 2
	CapabilityMainSeamlessMigrate      uint32 = 3
)

// Playback Channel capabilities
const (
	CapabilityPlaybackCELT051 uint32 = 0
	CapabilityPlaybackVolume  uint32 = 1
	CapabilityPlaybackLatency uint32 = 2
	CapabilityPlaybackOpus    uint32 = 3
)

// Record Channel capabilities
const (
	CapabilityRecordCELT051 uint32 = 0
	CapabilityRecordVolume  uint32 = 1
	CapabilityRecordOpus    uint32 = 2
)

// Display Channel capabilities
const (
	CapabilityDisplaySizedStream     uint32 = 0
	CapabilityDisplayMonitorsConfig  uint32 = 1
	CapabilityDisplayComposite       uint32 = 2
	CapabilityDisplayA8Surface       uint32 = 3
	CapabilityDisplayStreamReport    uint32 = 4
	CapabilityDisplayLZ4Compression  uint32 = 5
	CapabilityDisplayPREFCompression uint32 = 6
	CapabilityDisplayGLScanout       uint32 = 7
	CapabilityDisplayMMultiCodec     uint32 = 8
	CapabilityDisplayCodecMJPEG      uint32 = 9
	CapabilityDisplayCodecVP8        uint32 = 10
	CapabilityDisplayCodecH264       uint32 = 11
)

// Input Channel capabilities
const (
	CapabilityInputKeyScancode uint32 = 0
)

// Capability is a bitwise capability set
type Capability uint32

// Test whether bit i is set.
func (c *Capability) Test(i uint32) bool {
	if i >= 32 {
		return false
	}
	return *c&(1<<i) > 0
}

// Set bit i to 1
func (c *Capability) Set(i uint32) *Capability {
	if i >= 32 {
		return c
	}
	*c |= 1 << i
	return c
}

// Clear bit i to 0
func (c *Capability) Clear(i uint32) *Capability {
	if i >= 32 {
		return c
	}
	*c &^= 1 << i
	return c
}

// SetTo sets bit i to value
func (c *Capability) SetTo(i uint32, value bool) *Capability {
	if value {
		return c.Set(i)
	}
	return c.Clear(i)
}

// Flip bit at i
func (c *Capability) Flip(i uint32) *Capability {
	if i >= 32 {
		return c
	}
	*c ^= 1 << i
	return c
}
