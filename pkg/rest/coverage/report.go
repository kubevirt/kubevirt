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
)

func getSwaggerPath(path string, objectRef *AuditV1.ObjectReference) string {
	if namespace := objectRef.Namespace; namespace != "" {
		path = strings.Replace(path, namespace, "{namespace}", 1)
	}
	if name := objectRef.Name; name != "" {
		path = strings.Replace(path, name, "{name}", 1)
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
			fmt.Printf("Invalid query param: '%s'", k)
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

		// TODO: refactor from 70-84, move type conditions to 'addBodyParams' func
		if r, ok := req.([]interface{}); ok {
			for _, v := range r {
				if mv, ok := v.(map[string]interface{}); ok {
					extractBodyParams(mv, "", requestStats.Body, &requestStats.ParamsHit)
				} else {
					// this should never happen but it is better to log if this will occur
					fmt.Println("Skipping invalid requestObject", req)
				}
			}
		} else if mv, ok := req.(map[string]interface{}); ok {
			extractBodyParams(mv, "", requestStats.Body, &requestStats.ParamsHit)
		} else {
			// this should never happen but it is better to log if this will occur
			fmt.Println("Skipping invalid requestObject", req)
		}
	} else if requestStats != nil {
		// TODO: log warning about passed body param for request which doesn't have it in definition
	}

	return nil
}

func extractBodyParams(req map[string]interface{}, path string, body map[string]int, counter *int) {
	pathCopy := path
	for k, v := range req {
		path += "." + k
		if mv, ok := v.(map[string]interface{}); ok {
			extractBodyParams(mv, path, body, counter)
		} else if mv, ok := v.([]interface{}); ok {
			for _, v := range mv {
				if mv, ok := v.(map[string]interface{}); ok {
					extractBodyParams(mv, path, body, counter)
				} else {
					// here only value should be received (which is useless)
					// TODO: log info about it(?)
				}
			}
		} else {
			if _, ok := body[path]; !ok {
				// count unique hits only
				*counter++
			}
			body[path]++
		}
		path = pathCopy
	}
}

// TODO: make this more detailed like (path:method percentage)
func calculateCoverage(restAPI map[string]Request) float64 {
	paramsNum, paramsHit := 0, 0
	for _, req := range restAPI {
		for _, stats := range req.Methods {
			paramsNum += stats.ParamsNum

			// sometimes hit number is bigger than params number
			// it is because of missing spec definition in swagger json
			// as example v1.Patch has only description without listed params
			// TODO: check how v1.Patch was generated
			if stats.ParamsHit > paramsNum {
				paramsHit += stats.ParamsNum
			} else {
				paramsHit += stats.ParamsHit
			}
		}
	}
	return float64(paramsHit) * 100 / float64(paramsNum)
}

func GenerateReport(auditLogs string, swaggerPath string, filter string) error {
	start := time.Now()
	defer fmt.Printf(", execution time: %s", time.Since(start))

	var event AuditV1.Event

	restAPI, err := getRESTApi(swaggerPath, filter)
	if err != nil {
		return err
	}

	scanner := bufio.NewReader(strings.NewReader(auditLogs))
	for {
		str, err := scanner.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		err = json.Unmarshal(str, &event)
		if err != nil {
			return err
		}

		uri, err := url.Parse(event.RequestURI)
		if err != nil {
			return err
		}

		path := getSwaggerPath(uri.Path, event.ObjectRef)
		if _, ok := restAPI[path]; !ok {
			fmt.Printf("Path '%s' not found in swagger\n", path)
			continue
		}

		method := getHTTPMethod(event.Verb)
		if method == "" {
			fmt.Printf("Method '%s' not found for '%s' path\n", method, path)
			continue
		}

		if _, ok := restAPI[path].Methods[method]; !ok {
			fmt.Printf("Method '%s' not found for '%s' path\n", method, path)
			continue
		}
		restAPI[path].Methods[method].MethodCalled = true
		if restAPI[path].Methods[method].ParamsNum < 1 {
			// count requests without query and body parameters
			restAPI[path].Methods[method].ParamsHit++
		}

		matchQueryParams(uri.Query(), restAPI[path].Methods[method])
		err = matchBodyParams(event.RequestObject, restAPI[path].Methods[method])
		if err != nil {
			fmt.Println(err)
		}
	}

	fmt.Printf("Rest API coverage: %.2f%%", calculateCoverage(restAPI))
	return nil
}
