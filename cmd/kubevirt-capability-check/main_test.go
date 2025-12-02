package main

import (
	"testing"

	core_capabilities "kubevirt.io/kubevirt/pkg/capabilities/core"
)

func TestSupportLevelToString(t *testing.T) {
	tests := []struct {
		level    core_capabilities.SupportLevel
		expected string
	}{
		{core_capabilities.Unregistered, "Unregistered"},
		{core_capabilities.Unsupported, "Unsupported"},
		{core_capabilities.Experimental, "Experimental"},
		{core_capabilities.Deprecated, "Deprecated"},
	}

	for _, tt := range tests {
		result := supportLevelToString(tt.level)
		if result != tt.expected {
			t.Errorf("supportLevelToString(%d) = %s, want %s", tt.level, result, tt.expected)
		}
	}
}
