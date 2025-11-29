package main

import (
	"testing"

	"kubevirt.io/kubevirt/pkg/capabilities"
)

func TestSupportLevelToString(t *testing.T) {
	tests := []struct {
		level    capabilities.SupportLevel
		expected string
	}{
		{capabilities.Unregistered, "Unregistered"},
		{capabilities.Unsupported, "Unsupported"},
		{capabilities.Experimental, "Experimental"},
		{capabilities.Deprecated, "Deprecated"},
	}

	for _, tt := range tests {
		result := supportLevelToString(tt.level)
		if result != tt.expected {
			t.Errorf("supportLevelToString(%d) = %s, want %s", tt.level, result, tt.expected)
		}
	}
}
