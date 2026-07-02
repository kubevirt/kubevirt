package migrations

import (
	"fmt"
	"math"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	v1 "kubevirt.io/api/core/v1"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

// ValidateMigrationConfigurationOptions validates migration configuration fields.
// When oldConfiguration is non-nil (update), only changed fields are validated
// so that evolving validation criteria don't reject updates to unrelated fields.
func ValidateMigrationConfigurationOptions(
	sourceField *k8sfield.Path,
	oldConfiguration *v1.VMIMConfigurationOptions,
	newConfiguration *v1.VMIMConfigurationOptions,
	devConfig *v1.DeveloperConfiguration,
) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if newConfiguration == nil {
		return causes
	}

	if didFieldChange(oldConfiguration, newConfiguration, func(c *v1.VMIMConfigurationOptions) any { return c.CompletionTimeoutPerGiB }) {
		if newConfiguration.CompletionTimeoutPerGiB != nil && *newConfiguration.CompletionTimeoutPerGiB < 0 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "must not be negative",
				Field:   sourceField.Child("completionTimeoutPerGiB").String(),
			})
		}
	}

	if didFieldChange(oldConfiguration, newConfiguration, func(c *v1.VMIMConfigurationOptions) any { return c.ProgressTimeout }) {
		if newConfiguration.ProgressTimeout != nil && *newConfiguration.ProgressTimeout < 0 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "must be greater or equal to zero",
				Field:   sourceField.Child("progressTimeout").String(),
			})
		}
	}

	if didFieldChange(oldConfiguration, newConfiguration, func(c *v1.VMIMConfigurationOptions) any { return c.MaxDowntimeMs }) {
		if newConfiguration.MaxDowntimeMs != nil && (*newConfiguration.MaxDowntimeMs == 0 || *newConfiguration.MaxDowntimeMs > QEMUMaxMigrationDowntimeMS) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("must be in range 1-%d", QEMUMaxMigrationDowntimeMS),
				Field:   sourceField.Child("maxDowntimeMs").String(),
			})
		}

		if newConfiguration.MaxDowntimeMs != nil && !featuregate.IsEnabled(featuregate.MigrationStallDetection, devConfig) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Field:   sourceField.Child("maxDowntimeMs").String(),
				Message: fmt.Sprintf("maxDowntimeMs cannot be set without enabling the %s feature gate", featuregate.MigrationStallDetection),
			})
		}
	}

	if didFieldChange(oldConfiguration, newConfiguration, func(c *v1.VMIMConfigurationOptions) any { return c.BandwidthPerMigration }) {
		if newConfiguration.BandwidthPerMigration != nil {
			quantity, ok := newConfiguration.BandwidthPerMigration.AsInt64()
			if !ok {
				dec := newConfiguration.BandwidthPerMigration.AsDec()
				quantity = int64(dec.Sign())
			}
			if quantity < 0 {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "must not be negative",
					Field:   sourceField.Child("bandwidthPerMigration").String(),
				})
			}
		}
	}

	if didFieldChange(oldConfiguration, newConfiguration, func(c *v1.VMIMConfigurationOptions) any { return c.AllowPostCopy }) ||
		didFieldChange(oldConfiguration, newConfiguration, func(c *v1.VMIMConfigurationOptions) any { return c.AllowWorkloadDisruption }) {
		if newConfiguration.AllowPostCopy != nil && *newConfiguration.AllowPostCopy &&
			(newConfiguration.AllowWorkloadDisruption != nil && !*newConfiguration.AllowWorkloadDisruption) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "AllowWorkloadDisruption must be true when AllowPostCopy is true",
				Field:   sourceField.Child("allowWorkloadDisruption").String(),
			})
		}
	}

	// Note: Stall detector validation applies only when callers populate experimental options
	// (MigrationPolicy today; KubeVirt CR MigrationConfiguration has no experimental field).
	if didFieldChange(oldConfiguration, newConfiguration, stallDetectorAccessor) {
		sdField := sourceField.Child("experimental", "stallDetector")
		if newConfiguration.ExperimentalMigrationOptions != nil &&
			newConfiguration.ExperimentalMigrationOptions.StallDetector != nil {
			sd := newConfiguration.ExperimentalMigrationOptions.StallDetector
			if !featuregate.IsEnabled(featuregate.MigrationStallDetection, devConfig) {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Field:   sdField.String(),
					Message: fmt.Sprintf("experimental.stallDetector cannot be set without enabling the %s feature gate", featuregate.MigrationStallDetection),
				})
			}
			causes = append(causes, validateStallDetectorFactor(sdField.Child("ewmaAlpha"), sd.EwmaAlpha, 0, 1, true)...)
			causes = append(causes, validateStallDetectorFactor(sdField.Child("precopyPossibleFactor"), sd.PrecopyPossibleFactor, 1, math.MaxFloat64, false)...)
			causes = append(causes, validateStallDetectorFactor(sdField.Child("patienceWindowDecayFactor"), sd.PatienceWindowDecayFactor, 0, 1, false)...)
			causes = append(causes, validateStallDetectorFactor(sdField.Child("completionTimeoutFactor"), sd.CompletionTimeoutFactor, 1, math.MaxFloat64, false)...)
		}
	}

	return causes
}

func stallDetectorAccessor(c *v1.VMIMConfigurationOptions) any {
	if c == nil || c.ExperimentalMigrationOptions == nil {
		return nil
	}
	return c.ExperimentalMigrationOptions.StallDetector
}

func didFieldChange(
	oldConfig *v1.VMIMConfigurationOptions,
	newConfig *v1.VMIMConfigurationOptions,
	accessor func(*v1.VMIMConfigurationOptions) any,
) bool {
	if oldConfig == nil {
		return true
	}
	return !equality.Semantic.DeepEqual(accessor(oldConfig), accessor(newConfig))
}

// validateStallDetectorFactor validates a string-encoded floating-point stall
// detector factor field. exclusiveMin=true enforces factor > 0 (ignores min).
func validateStallDetectorFactor(field *k8sfield.Path, value *string, min, max float64, exclusiveMin bool) []metav1.StatusCause {
	if value == nil {
		return nil
	}

	factor, err := virtconfig.ParseFactor(*value, virtconfig.StallDetectorFactorPrecision)
	if err != nil {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: err.Error(),
			Field:   field.String(),
		}}
	}

	if exclusiveMin {
		if factor <= 0 {
			return []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "must be greater than 0",
				Field:   field.String(),
			}}
		}
	} else if factor < min {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("must be greater than or equal to %g", min),
			Field:   field.String(),
		}}
	}

	if factor > max {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("must be less than or equal to %g", max),
			Field:   field.String(),
		}}
	}

	return nil
}
