package installstrategy

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sv1 "kubevirt.io/client-go/api/v1"
)

func getCADuration(config *k8sv1.KubeVirtSelfSignConfiguration) *metav1.Duration {
	if config == nil || config.CARotateInterval == nil {
		return &metav1.Duration{Duration: Duration7d}
	}

	return config.CARotateInterval
}

func getCAOverlapTime(config *k8sv1.KubeVirtSelfSignConfiguration) *metav1.Duration {
	if config == nil || config.CAOverlapInterval == nil {
		return &metav1.Duration{Duration: Duration1d}
	}

	return config.CAOverlapInterval
}

func getCertDuration(config *k8sv1.KubeVirtSelfSignConfiguration) *metav1.Duration {
	if config == nil || config.CertRotateInterval == nil {
		return &metav1.Duration{Duration: Duration1d}
	}

	return config.CertRotateInterval
}
