package webhooks

import (
	"crypto/tls"

	ocpv1 "github.com/openshift/api/config/v1"
)

var (
	ciphers         = tls.CipherSuites()
	insecureCiphers = tls.InsecureCipherSuites()
)

func SelectCipherSuitesAndMinTLSVersion(profile *ocpv1.TLSSecurityProfile) (cipherSuiteIDs []uint16, minTLSVersion uint16) {
	if profile == nil {
		profile = &ocpv1.TLSSecurityProfile{
			Type:         ocpv1.TLSProfileIntermediateType,
			Intermediate: &ocpv1.IntermediateTLSProfile{},
		}
	}
	if profile.Custom != nil {
		cipherSuiteIDs, minTLSVersion = CipherSuitesIDs(profile.Custom.TLSProfileSpec.Ciphers), TlsVersion(profile.Custom.TLSProfileSpec.MinTLSVersion)
		return
	}

	cipherSuiteIDs, minTLSVersion = CipherSuitesIDs(ocpv1.TLSProfiles[profile.Type].Ciphers), TlsVersion(ocpv1.TLSProfiles[profile.Type].MinTLSVersion)
	return
}

func CipherSuitesIDs(names []string) []uint16 {
	// ref: https://www.iana.org/assignments/tls-parameters/tls-parameters.xml
	// ref: https://testssl.sh/openssl-iana.mapping.html
	var idByName = map[string]uint16{
		// TLS 1.2
		"ECDHE-ECDSA-AES128-GCM-SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"ECDHE-RSA-AES128-GCM-SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"ECDHE-ECDSA-AES256-GCM-SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"ECDHE-RSA-AES256-GCM-SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"ECDHE-ECDSA-CHACHA20-POLY1305": tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		"ECDHE-RSA-CHACHA20-POLY1305":   tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		"ECDHE-ECDSA-AES128-SHA256":     tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
		"ECDHE-RSA-AES128-SHA256":       tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		"AES128-GCM-SHA256":             tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		"AES256-GCM-SHA384":             tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		"AES128-SHA256":                 tls.TLS_RSA_WITH_AES_128_CBC_SHA256,

		// TLS 1
		"ECDHE-ECDSA-AES128-SHA": tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"ECDHE-RSA-AES128-SHA":   tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"ECDHE-ECDSA-AES256-SHA": tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"ECDHE-RSA-AES256-SHA":   tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,

		// SSL 3
		"AES128-SHA":   tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"AES256-SHA":   tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		"DES-CBC3-SHA": tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
	}
	for _, cipherSuite := range tls.CipherSuites() {
		idByName[cipherSuite.Name] = cipherSuite.ID
	}

	ids := []uint16{}
	for _, name := range names {
		if id, ok := idByName[name]; ok {
			ids = append(ids, id)
		}
	}
	return ids
}

// TlsVersion converts from human-readable TLS version (for example "1.1")
// to the values accepted by tls.Config (for example 0x301).
func TlsVersion(version ocpv1.TLSProtocolVersion) uint16 {
	switch version {
	// default is previous behavior
	case ocpv1.VersionTLS10:
		return tls.VersionTLS10
	case ocpv1.VersionTLS11:
		return tls.VersionTLS11
	case ocpv1.VersionTLS12:
		return tls.VersionTLS12
	case ocpv1.VersionTLS13:
		return tls.VersionTLS13
	default:
		return tls.VersionTLS10
	}
}
