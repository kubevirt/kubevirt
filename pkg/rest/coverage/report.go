package coverage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	AuditV1 "k8s.io/apiserver/pkg/apis/audit/v1"

	"kubevirt.io/kubevirt/pkg/log"
)

func getSwaggerPath(path string, objectRef *AuditV1.ObjectReference) string {
	if namespace := objectRef.Namespace; namespace != "" {
		path = strings.Replace(path, "namespaces/"+namespace, "namespaces/{namespace}", 1)
	}
	if name := objectRef.Name; name != "" {
		path = strings.Replace(path, objectRef.Resource+"/"+name, objectRef.Resource+"/{name}", 1)
	}
	return path
}

func getHTTPMethod(verb string) string {
	// audit log does not contain info about HTTP methods

	switch verb {
	case "get", "list":
		return "GET"
	case "create":
		return "POST"
	case "delete", "deletecollection":
		return "DELETE"
	case "update":
		return "PUT"
	case "patch":
		return "PATCH"
	default:
		return ""
	}
}

func matchQueryParams(values url.Values, requestStats *RequestStats) {
	for k := range values {
		if hits, ok := requestStats.Query[k]; ok {
			if hits < 1 {
				// get only unique hits
				requestStats.ParamsHit++
			}
			requestStats.Query[k]++
		} else {
			log.Log.Errorf("Invalid query param: '%s' for '%s %s'", k, requestStats.Method, requestStats.Path)
		}
	}
}

func matchBodyParams(requestObject *runtime.Unknown, requestStats *RequestStats) error {
	if requestObject != nil && requestStats.Body != nil {
		var req interface{}
		err := json.Unmarshal(requestObject.Raw, &req)
		if err != nil {
			return err
		}

		switch r := req.(type) {
		case []interface{}:
			for _, v := range r {
				err = extractBodyParams(v, "", requestStats.Body, &requestStats.ParamsHit, 0)
				if err != nil {
					return fmt.Errorf("Invalid requestObject '%s' for '%s %s'", err, requestStats.Method, requestStats.Path)
				}
			}
		default:
			err = extractBodyParams(r, "", requestStats.Body, &requestStats.ParamsHit, 0)
			if err != nil {
				return fmt.Errorf("Invalid requestObject '%s' for '%s %s'", err, requestStats.Method, requestStats.Path)
			}
		}

	} else if requestObject != nil {
		log.Log.Warningf("Request '%s %s' should not contain body params", requestStats.Method, requestStats.Path)
	}

	return nil
}

func extractBodyParams(params interface{}, path string, body map[string]int, counter *int, level int) error {
	p, ok := params.(map[string]interface{})
	if !ok && level == 0 {
		return fmt.Errorf("%v", p)
	} else if !ok {
		return nil
	}
	level++

	pathCopy := path
	for k, v := range p {
		path += "." + k

		switch obj := v.(type) {
		case map[string]interface{}:
			extractBodyParams(obj, path, body, counter, level)
		case []interface{}:
			for _, v := range obj {
				extractBodyParams(v, path, body, counter, level)
			}
		default:
			if _, ok := body[path]; !ok {
				*counter++
			}
			body[path]++
		}
		path = pathCopy
	}
	return nil
}

func calculateCoverage(restAPI map[string]Request) {
	paramsNum, paramsHit := 0, 0

	fmt.Printf("\nREST API coverage report:\n\n")
	for path, req := range restAPI {
		fmt.Println("\t", path)

		for _, stats := range req.Methods {
			// count path hit
			if stats.MethodCalled {
				stats.ParamsHit++
				stats.ParamsNum++
			}

			// sometimes hit number is bigger than params number
			// it is because of missing spec definition in swagger json
			// as example v1.Patch has only description without listed params
			// TODO: check how v1.Patch was generated
			if stats.ParamsHit > stats.ParamsNum {
				stats.ParamsHit = stats.ParamsNum
			}
			paramsNum += stats.ParamsNum
			paramsHit += stats.ParamsHit

			fmt.Printf("\t %s %.2f%%\n", stats.Method, float64(stats.ParamsHit)*100/float64(stats.ParamsNum))
		}
		fmt.Println()
	}
	fmt.Printf("Total coverage: %.2f%%\n\n", float64(paramsHit)*100/float64(paramsNum))
}

func GenerateReport(auditLogs string, swaggerPath string, filter string) error {
	log.InitializeLogging("rest-api-coverage")

	start := time.Now()
	defer log.Log.Infof("REST API coverage execution time: %s", time.Since(start))

	var event AuditV1.Event

	restAPI, err := getRESTApi(swaggerPath, filter)
	if err != nil {
		return err
	}

	scanner := bufio.NewReader(strings.NewReader(auditLogs))
	for {
		b, err := scanner.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		err = json.Unmarshal(b, &event)
		if err != nil {
			return err
		}

		uri, err := url.Parse(event.RequestURI)
		if err != nil {
			return err
		}

		path := getSwaggerPath(uri.Path, event.ObjectRef)
		if _, ok := restAPI[path]; !ok {
			log.Log.Errorf("Path '%s' not found in swagger", path)
			continue
		}

		method := getHTTPMethod(event.Verb)
		if method == "" {
			log.Log.Errorf("Method '%s' not found for '%s' path", method, path)
			continue
		}

		if _, ok := restAPI[path].Methods[method]; !ok {
			log.Log.Errorf("Method '%s' not found for '%s' path", method, path)
			continue
		}

		restAPI[path].Methods[method].MethodCalled = true
		matchQueryParams(uri.Query(), restAPI[path].Methods[method])
		err = matchBodyParams(event.RequestObject, restAPI[path].Methods[method])
		if err != nil {
			log.Log.Errorf("%s", err)
		}
	}

	calculateCoverage(restAPI)
	return nil
}
