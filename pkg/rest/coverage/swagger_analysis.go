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

type Request struct {
	Methods map[string]*RequestStats
}

func getRESTApi(swaggerPath string, filter string) (map[string]Request, error) {
	restAPI := make(map[string]Request)

	document, err := loads.JSONSpec(swaggerPath)
	if err != nil {
		return nil, err
	}

	for _, mp := range document.Analyzer.OperationMethodPaths() {
		v := strings.Split(mp, " ")
		if len(v) != 2 {
			return nil, fmt.Errorf("Invalid method-path pair '%s'", mp)
		}
		method, path := v[0], v[1]

		// filter requests uri
		if !strings.HasPrefix(path, filter) {
			continue
		}

		if _, ok := restAPI[path]; !ok {
			restAPI[path] = Request{
				Methods: make(map[string]*RequestStats),
			}
		}

		if _, ok := restAPI[path].Methods[method]; !ok {
			restAPI[path].Methods[method] = &RequestStats{
				Query:  make(map[string]int),
				Path:   path,
				Method: method,
			}
		}

		addSwaggerParams(method, path, document, restAPI)
	}

	return restAPI, nil
}

func addSwaggerParams(method string, path string, document *loads.Document, restAPI map[string]Request) {
	swagger := document.Spec()
	params := document.Analyzer.ParamsFor(method, path)

	for _, param := range params {
		switch param.In {
		case "body":
			restAPI[path].Methods[method].Body = make(map[string]int)
		case "query":
			restAPI[path].Methods[method].Query[param.Name] = 0
		default:
			continue
		}

		if param.Schema != nil {
			restAPI[path].Methods[method].ParamsNum += countRefParams(param.Schema, swagger)
		} else {
			restAPI[path].Methods[method].ParamsNum++
		}
	}
}

func countRefParams(schema *spec.Schema, swagger *spec.Swagger) int {
	var tokens []string
	ptr := schema.Ref.GetPointer()
	pCnt := 0

	if tokens = ptr.DecodedTokens(); len(tokens) < 2 {
		return 0
	}

	if tokens[0] != "definitions" {
		return 0
	}

	def, ok := swagger.Definitions[tokens[1]]
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
				pCnt += countRefParams(&p, swagger)
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
