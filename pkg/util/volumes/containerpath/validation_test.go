package containerpath

import (
	"strings"
	"testing"
)

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
		expected    string
		errorMsg    string
	}{
		{
			name:        "valid absolute path",
			path:        "/var/run/secrets/tokens",
			expectError: false,
			expected:    "/var/run/secrets/tokens",
		},
		{
			name:        "valid path with trailing slash",
			path:        "/var/run/secrets/",
			expectError: false,
			expected:    "/var/run/secrets",
		},
		{
			name:        "path with double slashes normalizes",
			path:        "/var//run///secrets",
			expectError: false,
			expected:    "/var/run/secrets",
		},
		{
			name:        "path with dot segments normalizes",
			path:        "/var/./run/./secrets",
			expectError: false,
			expected:    "/var/run/secrets",
		},
		{
			name:        "empty path fails",
			path:        "",
			expectError: true,
			errorMsg:    "path cannot be empty",
		},
		{
			name:        "relative path fails",
			path:        "var/run/secrets",
			expectError: true,
			errorMsg:    "path must be absolute",
		},
		{
			name:        "path traversal with .. fails (caught by dangerous path check)",
			path:        "/var/run/../../etc/passwd",
			expectError: true,
			errorMsg:    "not allowed for security reasons",
		},
		{
			name:        "path with .. in middle fails after normalization",
			path:        "/var/../etc/passwd",
			expectError: true,
			errorMsg:    "not allowed for security reasons",
		},
		{
			name:        "dangerous path /proc fails",
			path:        "/proc",
			expectError: true,
			errorMsg:    "not allowed for security reasons",
		},
		{
			name:        "dangerous subpath /proc/sys fails",
			path:        "/proc/sys",
			expectError: true,
			errorMsg:    "not allowed for security reasons",
		},
		{
			name:        "dangerous path /sys fails",
			path:        "/sys",
			expectError: true,
			errorMsg:    "not allowed for security reasons",
		},
		{
			name:        "dangerous path /dev fails",
			path:        "/dev",
			expectError: true,
			errorMsg:    "not allowed for security reasons",
		},
		{
			name:        "dangerous path /etc/passwd fails",
			path:        "/etc/passwd",
			expectError: true,
			errorMsg:    "not allowed for security reasons",
		},
		{
			name:        "dangerous path /etc/shadow fails",
			path:        "/etc/shadow",
			expectError: true,
			errorMsg:    "not allowed for security reasons",
		},
		{
			name:        "dangerous path /root fails",
			path:        "/root",
			expectError: true,
			errorMsg:    "not allowed for security reasons",
		},
		{
			name:        "/etc directory itself is allowed (only specific files blocked)",
			path:        "/etc/config",
			expectError: false,
			expected:    "/etc/config",
		},
		{
			name:        "AWS IRSA token path",
			path:        "/var/run/secrets/eks.amazonaws.com/serviceaccount",
			expectError: false,
			expected:    "/var/run/secrets/eks.amazonaws.com/serviceaccount",
		},
		{
			name:        "GCP Workload Identity token path",
			path:        "/var/run/secrets/gke.io/workload-identity",
			expectError: false,
			expected:    "/var/run/secrets/gke.io/workload-identity",
		},
		{
			name:        "Kubernetes service account token path",
			path:        "/var/run/secrets/kubernetes.io/serviceaccount",
			expectError: false,
			expected:    "/var/run/secrets/kubernetes.io/serviceaccount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidatePath(tt.path)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q but got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if result != tt.expected {
					t.Errorf("expected normalized path %q but got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestValidatePathMatchesVolumeMount(t *testing.T) {
	volumeMounts := []string{
		"/var/run/secrets/eks.amazonaws.com/serviceaccount",
		"/var/run/secrets/kubernetes.io/serviceaccount",
		"/etc/config",
	}

	tests := []struct {
		name        string
		path        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "exact match",
			path:        "/var/run/secrets/eks.amazonaws.com/serviceaccount",
			expectError: false,
		},
		{
			name:        "subpath of mount",
			path:        "/var/run/secrets/eks.amazonaws.com/serviceaccount/token",
			expectError: false,
		},
		{
			name:        "deep subpath of mount",
			path:        "/var/run/secrets/eks.amazonaws.com/serviceaccount/data/token",
			expectError: false,
		},
		{
			name:        "different mount exact match",
			path:        "/etc/config",
			expectError: false,
		},
		{
			name:        "subpath of different mount",
			path:        "/etc/config/app.conf",
			expectError: false,
		},
		{
			name:        "no match - completely different path",
			path:        "/tmp/random/path",
			expectError: true,
			errorMsg:    "does not correspond to any volumeMount",
		},
		{
			name:        "no match - similar prefix but not subpath",
			path:        "/var/run/secrets/other",
			expectError: true,
			errorMsg:    "does not correspond to any volumeMount",
		},
		{
			name:        "no match - parent of mount path",
			path:        "/var/run/secrets",
			expectError: true,
			errorMsg:    "does not correspond to any volumeMount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePathMatchesVolumeMount(tt.path, volumeMounts)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q but got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidatePathMatchesVolumeMount_EmptyMounts(t *testing.T) {
	err := ValidatePathMatchesVolumeMount("/any/path", []string{})
	if err == nil {
		t.Errorf("expected error for empty volumeMounts but got none")
	}
	if !strings.Contains(err.Error(), "no volumeMounts available") {
		t.Errorf("expected error about no volumeMounts but got: %v", err)
	}
}

func TestGetDefaultReadOnly(t *testing.T) {
	if !GetDefaultReadOnly() {
		t.Errorf("expected default ReadOnly to be true, got false")
	}
}

func TestApplyReadOnlyDefault(t *testing.T) {
	tests := []struct {
		name     string
		input    *bool
		expected bool
	}{
		{
			name:     "nil input uses default (true)",
			input:    nil,
			expected: true,
		},
		{
			name:     "explicit true",
			input:    boolPtr(true),
			expected: true,
		},
		{
			name:     "explicit false",
			input:    boolPtr(false),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyReadOnlyDefault(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v but got %v", tt.expected, result)
			}
		})
	}
}

// Helper function to create bool pointers
func boolPtr(b bool) *bool {
	return &b
}
