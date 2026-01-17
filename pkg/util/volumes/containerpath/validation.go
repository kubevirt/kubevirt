package containerpath

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidationError represents a validation failure for ContainerPath volumes
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidatePath validates and normalizes a container path.
// It returns the normalized path or an error if the path is invalid.
//
// Validation rules:
// - Path must not be empty
// - Path must be absolute (start with /)
// - Path must not contain '..' after normalization
// - Path must not point to dangerous system directories
func ValidatePath(path string) (string, error) {
	// Check for empty path
	if path == "" {
		return "", &ValidationError{
			Field:   "path",
			Message: "path cannot be empty",
		}
	}

	// Must be absolute
	if !filepath.IsAbs(path) {
		return "", &ValidationError{
			Field:   "path",
			Message: fmt.Sprintf("path must be absolute, got: %s", path),
		}
	}

	// Normalize path (resolve .., ., remove double slashes)
	normalized := filepath.Clean(path)

	// Check for path traversal attempts after normalization
	// If the normalized path still contains "..", it's trying to escape
	if strings.Contains(normalized, "..") {
		return "", &ValidationError{
			Field:   "path",
			Message: "path must not contain '..' components",
		}
	}

	// Blacklist dangerous paths that should never be exposed to VMs
	dangerousPaths := []string{
		"/proc",
		"/sys",
		"/dev",
		"/etc/passwd",
		"/etc/shadow",
		"/etc/sudoers",
		"/root",
	}

	for _, dangerous := range dangerousPaths {
		if normalized == dangerous || strings.HasPrefix(normalized, dangerous+"/") {
			return "", &ValidationError{
				Field:   "path",
				Message: fmt.Sprintf("path %s is not allowed for security reasons", dangerous),
			}
		}
	}

	return normalized, nil
}

// ValidatePathMatchesVolumeMount validates that the path corresponds to a
// volumeMount in the virt-launcher pod. This should be called at runtime
// when the actual pod spec is available.
//
// A path is valid if it either:
// - Exactly matches a volumeMount path, OR
// - Is a subpath of a volumeMount path
func ValidatePathMatchesVolumeMount(path string, volumeMounts []string) error {
	if len(volumeMounts) == 0 {
		return &ValidationError{
			Field:   "path",
			Message: "no volumeMounts available in virt-launcher pod",
		}
	}

	for _, mount := range volumeMounts {
		// Exact match
		if path == mount {
			return nil
		}
		// Subpath of mount
		if strings.HasPrefix(path, mount+"/") {
			return nil
		}
	}

	return &ValidationError{
		Field:   "path",
		Message: fmt.Sprintf("path %s does not correspond to any volumeMount in virt-launcher pod (available mounts: %v)", path, volumeMounts),
	}
}

// GetDefaultReadOnly returns the default value for the ReadOnly field.
// This should be used consistently across admission, mutation, and conversion logic.
func GetDefaultReadOnly() bool {
	return true
}

// ApplyReadOnlyDefault applies the default ReadOnly value if not explicitly set.
// Returns the effective ReadOnly value.
func ApplyReadOnlyDefault(readOnly *bool) bool {
	if readOnly == nil {
		return GetDefaultReadOnly()
	}
	return *readOnly
}
