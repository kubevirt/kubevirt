package apply

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sv1 "kubevirt.io/api/core/v1"
)

func GetCADuration(config *k8sv1.KubeVirtSelfSignConfiguration) *metav1.Duration {
	defaultDuration := &metav1.Duration{Duration: Duration7d}

	if config == nil {
		return defaultDuration
	}

	// deprecated, but takes priority to provide a smooth upgrade path
	if config.CARotateInterval != nil {
		return config.CARotateInterval
	}
	if config.CA != nil && config.CA.Duration != nil {
		return config.CA.Duration
	}

	return defaultDuration
}

func GetCARenewBefore(config *k8sv1.KubeVirtSelfSignConfiguration) *metav1.Duration {
	caDuration := GetCADuration(config)
	defaultDuration := &metav1.Duration{Duration: time.Duration(float64(caDuration.Duration) * 0.2)}

	if config == nil {
		return defaultDuration
	}

	// deprecated, but takes priority to provide a smooth upgrade path
	if config.CAOverlapInterval != nil {
		return config.CAOverlapInterval
	}

	if config.CA != nil && config.CA.RenewBefore != nil {
		return config.CA.RenewBefore
	}

	return defaultDuration
}

func GetCertDuration(config *k8sv1.KubeVirtSelfSignConfiguration) *metav1.Duration {
	defaultDuration := &metav1.Duration{Duration: Duration1d}

	if config == nil {
		return defaultDuration
	}

	// deprecated, but takes priority to provide a smooth upgrade path
	if config.CertRotateInterval != nil {
		return config.CertRotateInterval
	}
	if config.Server != nil && config.Server.Duration != nil {
		return config.Server.Duration
	}

	return defaultDuration
}

func GetCertRenewBefore(config *k8sv1.KubeVirtSelfSignConfiguration) *metav1.Duration {
	certDuration := GetCertDuration(config)
	defaultDuration := &metav1.Duration{Duration: time.Duration(float64(certDuration.Duration) * 0.2)}

	if config == nil {
		return defaultDuration
	}

	if config.Server != nil && config.Server.RenewBefore != nil {
		return config.Server.RenewBefore
	}

	return defaultDuration
}
