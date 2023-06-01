package vmispec

import "strings"

const (
	InfoSourceDomain       string = "domain"
	InfoSourceGuestAgent   string = "guest-agent"
	InfoSourceMultusStatus string = "multus-status"
	InfoSourceDomainAndGA  string = InfoSourceDomain + ", " + InfoSourceGuestAgent

	seperator = ", "
)

func AddInfoSource(infoSourceData, name string) string {
	var infoSources []string
	if infoSourceData != "" {
		infoSources = strings.Split(infoSourceData, seperator)
	}
	for _, infoSourceName := range infoSources {
		if infoSourceName == name {
			return infoSourceData
		}
	}
	infoSources = append(infoSources, name)
	return NewInfoSource(infoSources...)
}

func RemoveInfoSource(infoSourceData, name string) string {
	var newInfoSources []string
	infoSources := strings.Split(infoSourceData, seperator)
	for _, infoSourceName := range infoSources {
		if infoSourceName != name {
			newInfoSources = append(newInfoSources, infoSourceName)
		}
	}
	return NewInfoSource(newInfoSources...)
}

func ContainsInfoSource(infoSourceData, name string) bool {
	infoSources := strings.Split(infoSourceData, seperator)
	for _, infoSourceName := range infoSources {
		if infoSourceName == name {
			return true
		}
	}
	return false
}

func NewInfoSource(names ...string) string {
	return strings.Join(names, seperator)
}
