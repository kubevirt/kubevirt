package analysis

import (
	"fmt"
	"strings"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"

	"github.com/mfranczy/crd-rest-coverage/pkg/stats"
)

// AnalyzeSwagger initializes a stats structure based on swagger definition with total params number for each available endpoint
func AnalyzeSwagger(document *loads.Document, filter string) (*stats.Coverage, error) {
	coverage := stats.Coverage{
		Endpoints: make(map[string]map[string]*stats.Endpoint),
	}

	for _, mp := range document.Analyzer.OperationMethodPaths() {
		v := strings.Split(mp, " ")
		if len(v) != 2 {
			return nil, fmt.Errorf("Invalid method:path pair '%s'", mp)
		}
		method, path := strings.ToLower(v[0]), strings.ToLower(v[1])

		// filter requests uri
		if !strings.HasPrefix(path, filter) {
			continue
		}

		if _, ok := coverage.Endpoints[path]; !ok {
			coverage.Endpoints[path] = make(map[string]*stats.Endpoint)
		}

		if _, ok := coverage.Endpoints[path][method]; !ok {
			coverage.Endpoints[path][method] = &stats.Endpoint{
				Params: stats.Params{
					Query: stats.NewTrie(),
					Body:  stats.NewTrie(),
				},
				Path:               path,
				Method:             method,
				ExpectedUniqueHits: 1, // count endpoint calls
			}
			coverage.ExpectedUniqueHits++
		}

		addSwaggerParams(coverage.Endpoints[path][method], document.Analyzer.ParamsFor(method, path), document.Spec().Definitions)
	}

	// caclulate number of expected unique hits
	for path, method := range coverage.Endpoints {
		for name, endpoint := range method {
			expectedUniqueHits := endpoint.Params.Body.ExpectedUniqueHits + endpoint.Params.Query.ExpectedUniqueHits
			coverage.ExpectedUniqueHits += expectedUniqueHits
			coverage.Endpoints[path][name].ExpectedUniqueHits += expectedUniqueHits
		}
	}

	return &coverage, nil
}

// addSwaggerParams adds parameters from swagger definition into coverage structure
func addSwaggerParams(endpoint *stats.Endpoint, params map[string]spec.Parameter, definitions spec.Definitions) {
	for _, param := range params {
		switch param.In {
		case "body":
			if param.Schema != nil {
				extractBodyParams(param.Schema, definitions, endpoint.Body, endpoint.Body.Root)
			} else {
				endpoint.Params.Body.Add(param.Name, endpoint.Body.Root, true)
			}
		case "query":
			endpoint.Params.Query.Add(param.Name, endpoint.Query.Root, true)
		default:
			continue
		}
	}
}

// extractBodyParams builds a stats trie
func extractBodyParams(schema *spec.Schema, definitions spec.Definitions, body *stats.Trie, node *stats.Node) {

	var tokens []string
	ptr := schema.Ref.GetPointer()

	// TODO: replace by ptr.Get() func
	if tokens = ptr.DecodedTokens(); len(tokens) < 2 {
		return
	}
	refType, refName := tokens[0], tokens[1]

	if refType != "definitions" {
		return
	}

	def, ok := definitions[refName]
	// did not find swagger definition
	if !ok {
		return
	}

	if len(def.Properties) > 0 {
		for k, s := range def.Properties {
			if r := s.Ref.GetPointer(); r != nil && len(r.DecodedTokens()) > 0 {
				n := body.Add(k, node, false)
				extractBodyParams(&s, definitions, body, n)
			} else if s.Items != nil && s.Items.Schema != nil {
				// type array can have its own reference
				// !multiple Schemas are not supported so far!
				n := body.Add(k, node, false)
				extractBodyParams(s.Items.Schema, definitions, body, n)
			} else {
				body.Add(k, node, true)
			}
		}
	} else {
		// reference exists but definition is an empty object{}
		node.IsLeaf = true
		body.ExpectedUniqueHits++
	}
}
