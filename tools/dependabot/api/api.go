package api

import (
	"strings"
	"time"
)

type Alert struct {
	Number     int    `json:"number"`
	State      string `json:"state"`
	Dependency struct {
		Package struct {
			Ecosystem string `json:"ecosystem"`
			Name      string `json:"name"`
		} `json:"package"`
		ManifestPath string `json:"manifest_path"`
		Scope        string `json:"scope"`
	} `json:"dependency"`
	SecurityAdvisory struct {
		GhsaID      string `json:"ghsa_id"`
		CveID       string `json:"cve_id"`
		Summary     string `json:"summary"`
		Description string `json:"description"`
		Severity    string `json:"severity"`
		Identifiers []struct {
			Value string `json:"value"`
			Type  string `json:"type"`
		} `json:"identifiers"`
		References []struct {
			URL string `json:"url"`
		} `json:"references"`
		PublishedAt     time.Time   `json:"published_at"`
		UpdatedAt       time.Time   `json:"updated_at"`
		WithdrawnAt     interface{} `json:"withdrawn_at"`
		Vulnerabilities []struct {
			Package struct {
				Ecosystem string `json:"ecosystem"`
				Name      string `json:"name"`
			} `json:"package"`
			Severity               string `json:"severity"`
			VulnerableVersionRange string `json:"vulnerable_version_range"`
			FirstPatchedVersion    struct {
				Identifier string `json:"identifier"`
			} `json:"first_patched_version"`
		} `json:"vulnerabilities"`
		Cvss struct {
			VectorString string  `json:"vector_string"`
			Score        float64 `json:"score"`
		} `json:"cvss"`
		Cwes []struct {
			CweID string `json:"cwe_id"`
			Name  string `json:"name"`
		} `json:"cwes"`
	} `json:"security_advisory"`
	SecurityVulnerability struct {
		Package struct {
			Ecosystem string `json:"ecosystem"`
			Name      string `json:"name"`
		} `json:"package"`
		Severity               string `json:"severity"`
		VulnerableVersionRange string `json:"vulnerable_version_range"`
		FirstPatchedVersion    struct {
			Identifier string `json:"identifier"`
		} `json:"first_patched_version"`
	} `json:"security_vulnerability"`
	URL              string      `json:"url"`
	HTMLURL          string      `json:"html_url"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
	DismissedAt      interface{} `json:"dismissed_at"`
	DismissedBy      interface{} `json:"dismissed_by"`
	DismissedReason  interface{} `json:"dismissed_reason"`
	DismissedComment interface{} `json:"dismissed_comment"`
	FixedAt          time.Time   `json:"fixed_at"`
}

type CVE struct {
	PackageName         string
	FixedPackageVersion string
	CVE                 string
	GoMod               string
}

func GetOpenGolangCVEs(alerts []Alert) []CVE {
	var cves []CVE
	for _, alert := range alerts {
		if alert.State != "open" {
			continue
		}
		if alert.SecurityVulnerability.Package.Ecosystem != "go" {
			continue
		}
		cves = append(cves, CVE{
			PackageName:         alert.SecurityVulnerability.Package.Name,
			FixedPackageVersion: "v" + strings.TrimPrefix(alert.SecurityVulnerability.FirstPatchedVersion.Identifier, "v"),
			CVE:                 alert.SecurityAdvisory.CveID,
			GoMod:               strings.TrimSuffix(alert.Dependency.ManifestPath, "go.sum") + "go.mod",
		})
	}
	return cves
}
