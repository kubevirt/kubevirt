package main

import (
	"strings"

	"github.com/Sirupsen/logrus"
	restful "github.com/emicklei/go-restful"
	"github.com/go-openapi/spec"
)

func enrichSwaggerObject(swo *spec.Swagger) {
	swo.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "Example",
			Description: "Resource for doing example things",
			Contact: &spec.ContactInfo{
				Name:  "dkiser",
				Email: "domingo.kiser@gmail.com",
				URL:   "domingo.space",
			},
			License: &spec.License{
				Name: "MIT",
				URL:  "http://mit.org",
			},
			Version: "1.0.0",
		},
	}
	swo.Tags = []spec.Tag{spec.Tag{TagProps: spec.TagProps{
		Name:        "example",
		Description: "Exampling and stuff"}}}

	// setup security definitions
	swo.SecurityDefinitions = map[string]*spec.SecurityScheme{
		"jwt": spec.APIKeyAuth("Authorization", "header"),
	}

	// map routes to security definitions
	enrichSwaggeerObjectSecurity(swo)
}

func enrichSwaggeerObjectSecurity(swo *spec.Swagger) {

	// loop through all registerd web services
	for _, ws := range restful.RegisteredWebServices() {
		for _, route := range ws.Routes() {

			// grab route metadata for a SecurityDefinition
			secdefn, ok := route.Metadata[SecurityDefinitionKey]
			if !ok {
				continue
			}

			// grab pechelper.OAISecurity from the stored interface{}
			var sEntry OAISecurity
			switch v := secdefn.(type) {
			case *OAISecurity:
				sEntry = *v
			case OAISecurity:
				sEntry = v
			default:
				// not valid type
				logrus.Warningf("skipping Security openapi spec for %s:%s, invalid metadata type %v", route.Method, route.Path, v)
				continue
			}

			if _, ok := swo.SecurityDefinitions[sEntry.Name]; !ok {
				logrus.Warningf("skipping Security openapi spec for %s:%s, '%s' not found in SecurityDefinitions", route.Method, route.Path, sEntry.Name)
				continue
			}

			// grab path and path item in openapi spec
			path, err := swo.Paths.JSONLookup(route.Path)
			if err != nil {
				logrus.Warning("skipping Security openapi spec for %s:%s, %s", route.Method, route.Path, err.Error())
				continue
			}
			pItem := path.(*spec.PathItem)

			// Update respective path Option based on method
			var pOption *spec.Operation
			switch method := strings.ToLower(route.Method); method {
			case "get":
				pOption = pItem.Get
			case "post":
				pOption = pItem.Post
			case "patch":
				pOption = pItem.Patch
			case "delete":
				pOption = pItem.Delete
			case "put":
				pOption = pItem.Put
			case "head":
				pOption = pItem.Head
			case "options":
				pOption = pItem.Options
			default:
				// unsupported method
				logrus.Warningf("skipping Security openapi spec for %s:%s, unsupported method '%s'", route.Method, route.Path, route.Method)
				continue
			}

			// update the pOption with security entry
			pOption.SecuredWith(sEntry.Name, sEntry.Scopes...)
		}
	}

}
