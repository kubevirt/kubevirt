package coverage

import (
	"fmt"
	"strings"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
)

var refCache = make(map[string]int)

type RequestStats struct {
	Body         map[string]int
	Query        map[string]int
	ParamsHit    int
	ParamsNum    int
	MethodCalled bool
	Path         string
	Method       string
}

func getRESTApiStats(swaggerPath string, filter string) (map[string]map[string]*RequestStats, error) {
	restAPIStats := make(map[string]map[string]*RequestStats)

	document, err := loads.JSONSpec(swaggerPath)
	if err != nil {
		return nil, err
	}

	for _, mp := range document.Analyzer.OperationMethodPaths() {
		v := strings.Split(mp, " ")
		if len(v) != 2 {
			return nil, fmt.Errorf("Invalid method:path pair '%s'", mp)
		}
		method, path := v[0], v[1]

		// filter requests uri
		if !strings.HasPrefix(path, filter) {
			continue
		}

		if _, ok := restAPIStats[path]; !ok {
			restAPIStats[path] = make(map[string]*RequestStats)
		}

		if _, ok := restAPIStats[path][method]; !ok {
			restAPIStats[path][method] = &RequestStats{
				Query:  make(map[string]int),
				Path:   path,
				Method: method,
			}
		}

		addSwaggerParams(restAPIStats[path][method], document.Analyzer.ParamsFor(method, path), document.Spec().Definitions)
	}

	return restAPIStats, nil
}

func addSwaggerParams(requestStats *RequestStats, params map[string]spec.Parameter, definitions spec.Definitions) {
	for _, param := range params {
		switch param.In {
		case "body":
			requestStats.Body = make(map[string]int)
		case "query":
			requestStats.Query[param.Name] = 0
		default:
			continue
		}

		if param.Schema != nil {
			requestStats.ParamsNum += countRefParams(param.Schema, definitions)
		} else {
			requestStats.ParamsNum++
		}
	}
}

func countRefParams(schema *spec.Schema, definitions spec.Definitions) int {
	var tokens []string
	ptr := schema.Ref.GetPointer()
	pCnt := 0

	if tokens = ptr.DecodedTokens(); len(tokens) < 2 {
		return 0
	}

	if tokens[0] != "definitions" {
		return 0
	}

	def, ok := definitions[tokens[1]]
	// did not find swagger definition
	if !ok {
		return 0
	}

	// if it is possible, get data from map to avoid calculation
	if val, ok := refCache[tokens[1]]; ok {
		return val
	}

	if len(def.Properties) > 0 {
		for _, p := range def.Properties {
			if r := p.Ref.GetPointer(); r != nil && len(r.DecodedTokens()) > 0 {
				pCnt += countRefParams(&p, definitions)
			} else {
				pCnt++
			}
		}
	} else {
		pCnt++
	}

	refCache[tokens[1]] = pCnt
	return pCnt
}
