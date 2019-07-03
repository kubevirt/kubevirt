package coverage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"sort"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"

	"kubevirt.io/client-go/log"
)

// getSwaggerPath translates request path to generic swagger path, as an example,
// /apis/kubevirt.io/v1alpha3/namespaces/kubevirt-test-default/virtualmachineinstances/vm-name will be translated to
// /apis/kubevirt.io/v1alpha3/namespaces/{namespace}/virtualmachineinstances/{name}
func getSwaggerPath(path string, objectRef *auditv1.ObjectReference) string {
	if namespace := objectRef.Namespace; namespace != "" {
		path = strings.Replace(path, "namespaces/"+namespace, "namespaces/{namespace}", 1)
	}
	if name := objectRef.Name; name != "" {
		path = strings.Replace(path, objectRef.Resource+"/"+name, objectRef.Resource+"/{name}", 1)
	}
	return path
}

// getHTTPMethod translates k8s verbs from audit log into HTTP methods
// NOTE: audit log does not provide information about HTTP methods
func getHTTPMethod(verb string) string {
	switch verb {
	case "get", "list", "watch", "watchList":
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

// matchQueryParams matches query params from request log to stats structure built based on swagger definition,
// as an example, by having requestStats{method: GET, Query: [param1: 0], ParamsHit: 0}
// and request GET /?param1=test1&param2=test2
// - it increments the parameter occurrence number, requestStats{method: GET, Query: [param1: 1], ParamsHit: 1}
// - it logs an error for not documented/invalid parameters, in this example parameter named "test2"
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

// matchBodyParams matches body params from request log to stats structure built based on swagger definition,
// as an example, by having requestStats{method: POST, Body: {}, ParamsHit: 0} and request
// POST / {"param1": {"param2": "test"}}
// - it builds the parameter path and increase its occurrence number, requestStats{method: POST, Body: {param1.param2: 1}, ParamsHit: 1}
// - it returns an error if request provides body params but it is not defined in swagger definition
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

// extractBodyParams builds a body parameter path from JSON structure and increase its occurence number, as an example,
// {param1: {param2: {param3a: value1, param3b: value2}}} will be extracted into paths:
// - param1.param2.param3a: 1
// - param1.param2.param3b: 1
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
		if level == 1 {
			path = k
		} else {
			path += "." + k
		}

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

// calculateCoverage provides a total REST API and PATH:METHOD coverage number
func calculateCoverage(restAPIStats map[string]map[string]*RequestStats) map[string]float64 {
	result := map[string]float64{}
	paramsNum, paramsHit := 0, 0

	for path, req := range restAPIStats {
		for method, stats := range req {
			// count path hits
			if stats.MethodCalled {
				stats.ParamsHit++
				stats.ParamsNum++
			}

			// sometimes hit number is bigger than params number
			// it is because of missing spec definition in swagger
			// as example v1.Patch has only description without listed params
			// TODO: check how v1.Patch was generated
			if stats.ParamsHit > stats.ParamsNum {
				stats.ParamsHit = stats.ParamsNum
			}

			if stats.ParamsNum > 0 {
				paramsNum += stats.ParamsNum
				paramsHit += stats.ParamsHit
				result[path+":"+method] = float64(stats.ParamsHit) * 100 / float64(stats.ParamsNum)
			} else {
				result[path+":"+method] = 0
			}
		}
	}

	if paramsNum > 0 {
		result["total"] = float64(paramsHit) * 100 / float64(paramsNum)
	} else {
		result["total"] = 0
	}

	return result
}

// printReport sends a generated report to stdout,
// with a detailed option, it will show coverage number for each resource
func printReport(report map[string]float64, detailed bool) error {
	fmt.Printf("\nREST API coverage report:\n")
	if detailed {
		keys := []string{}
		for k := range report {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		prevPath := ""
		for _, k := range keys {
			if k == "total" {
				continue
			}
			pm := strings.Split(k, ":")
			if len(pm) != 2 {
				return fmt.Errorf("Invalid path:method pair: %s", pm)
			}
			path, method := pm[0], pm[1]
			if path != prevPath {
				fmt.Printf("\n%s:\n", path)
			}
			prevPath = path
			fmt.Printf("\t%s %s: %.2f%%\n", path, method, report[k])
		}
	}
	fmt.Printf("\nTotal coverage: %.2f%%\n\n", report["total"])
	return nil
}

// dumpReport saves a generated report into a file in JSON format
func dumpReport(path string, report map[string]float64) error {
	jsonReport, err := json.Marshal(report)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, jsonReport, 0644)
}

// GenerateReport provides a full REST API coverage report based on k8s audit log and swagger definition,
// by passing params:
// - "filter" you can limit the report to specific resources, as an example, "/apis/kubevirt.io/v1alpha3/" limits to kubevirt v1alpha3; "" no limit
// - "storeInFilePath" instead of sending the report to stdout it is possible to keep it as a file in JSON format (always detailed)
// - "detailed" whether a printed report should contain coverage information for each resource, if not it will show only the total coverage number
func GenerateReport(auditLogs string, swaggerPath string, filter string, storeInFilePath string, detailed bool) error {
	log.InitializeLogging("rest-api-coverage")

	start := time.Now()
	defer log.Log.Infof("REST API coverage execution time: %s", time.Since(start))

	restAPIStats, err := getRESTApiStats(swaggerPath, filter)
	if err != nil {
		return err
	}

	scanner := bufio.NewReader(strings.NewReader(auditLogs))
	for {
		var event auditv1.Event
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
		if _, ok := restAPIStats[path]; !ok {
			log.Log.Errorf("Path '%s' not found in swagger", path)
			continue
		}

		method := getHTTPMethod(event.Verb)
		if method == "" {
			log.Log.Errorf("Method '%s' not found for '%s' path", method, path)
			continue
		}

		if _, ok := restAPIStats[path][method]; !ok {
			log.Log.Errorf("Method '%s' not found for '%s' path", method, path)
			continue
		}

		restAPIStats[path][method].MethodCalled = true
		matchQueryParams(uri.Query(), restAPIStats[path][method])
		err = matchBodyParams(event.RequestObject, restAPIStats[path][method])
		if err != nil {
			log.Log.Errorf("%s", err)
		}
	}

	report := calculateCoverage(restAPIStats)
	if storeInFilePath != "" {
		return dumpReport(storeInFilePath, report)
	}
	return printReport(report, detailed)
}
