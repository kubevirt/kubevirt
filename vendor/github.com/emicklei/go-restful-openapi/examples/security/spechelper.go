package main

import "fmt"

const (
	SecurityDefinitionKey = "OAPI_SECURITY_DEFINITION"
)

type OAISecurity struct {
	Name   string   // SecurityDefinition name
	Scopes []string // Scopes for oauth2
}

func (s *OAISecurity) Valid() error {
	switch s.Name {
	case "oauth2":
		return nil
	case "openIdConnect":
		return nil
	default:
		if len(s.Scopes) > 0 {
			return fmt.Errorf("oai Security scopes for scheme '%s' should be empty", s.Name)
		}
	}

	return nil
}
