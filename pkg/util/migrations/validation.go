package migrations

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	v1 "kubevirt.io/api/core/v1"
)

func ValidateVMMigrationConfiguration(
	sourceField *k8sfield.Path,
	configuration v1.VMMigrationConfiguration,
) []metav1.StatusCause {
	return ValidateLegacyVMMigrationConfiguration(sourceField, configuration.LegacyVMMigrationConfiguration)
}

func ValidateLegacyVMMigrationConfiguration(
	sourceField *k8sfield.Path,
	configuration v1.LegacyVMMigrationConfiguration,
) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if configuration.CompletionTimeoutPerGiB != nil && *configuration.CompletionTimeoutPerGiB < 0 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "must not be negative",
			Field:   sourceField.Child("completionTimeoutPerGiB").String(),
		})
	}
	if configuration.ProgressTimeout != nil && *configuration.ProgressTimeout < 0 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "must be greater or equal to zero",
			Field:   sourceField.Child("progressTimeout").String(),
		})
	}
	if configuration.MaxDowntime != nil && (*configuration.MaxDowntime <= 0 || *configuration.MaxDowntime > QEMUMaxMigrationDowntimeMS) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("must be in range 1-%d", QEMUMaxMigrationDowntimeMS),
			Field:   sourceField.Child("maxDowntime").String(),
		})
	}

	if configuration.BandwidthPerMigration != nil {
		quantity, ok := configuration.BandwidthPerMigration.AsInt64()
		if !ok {
			dec := configuration.BandwidthPerMigration.AsDec()
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

	if configuration.AllowPostCopy != nil && *configuration.AllowPostCopy &&
		configuration.AllowWorkloadDisruption != nil && !*configuration.AllowWorkloadDisruption {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "AllowWorkloadDisruption must be true if AllowPostCopy is true",
			Field:   sourceField.Child("allowWorkloadDisruption").String(),
		})
	}

	return causes
}

func ValidateClusterMigrationConfiguration(
	_ *k8sfield.Path,
	_ v1.ClusterMigrationConfiguration,
) []metav1.StatusCause {
	var causes []metav1.StatusCause
	return causes
}
