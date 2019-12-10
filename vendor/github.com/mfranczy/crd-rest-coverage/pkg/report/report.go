package report

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-openapi/loads"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime"
	auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"

	"github.com/mfranczy/crd-rest-coverage/pkg/analysis"
	"github.com/mfranczy/crd-rest-coverage/pkg/stats"
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
		return "get"
	case "create":
		return "post"
	case "delete", "deletecollection":
		return "delete"
	case "update":
		return "put"
	case "patch":
		return "patch"
	default:
		return ""
	}
}

// matchQueryParams matches query params from request log to stats structure which has been built based on swagger definition
func matchQueryParams(values url.Values, endpoint *stats.Endpoint) {
	for k := range values {
		if n := endpoint.Query.Root.GetChild(k); n != nil {
			endpoint.Params.Query.IncreaseHits(n)
		} else {
			glog.Errorf("Invalid query param: '%s' for '%s %s'", k, endpoint.Method, endpoint.Path)
		}
	}
}

// matchBodyParams matches body params from request log to stats structure which has been built based on swagger definition
func matchBodyParams(requestObject *runtime.Unknown, endpoint *stats.Endpoint) error {
	if requestObject != nil {
		var req interface{}
		err := json.Unmarshal(requestObject.Raw, &req)
		if err != nil {
			return err
		}
		switch r := req.(type) {
		case []interface{}:
			for _, v := range r {
				err = extractBodyParams(v, "", endpoint.Params.Body, endpoint.Params.Body.Root)
				if err != nil {
					return fmt.Errorf("Invalid requestObject '%s' for '%s %s'", err, endpoint.Method, endpoint.Path)
				}
			}
		default:
			err = extractBodyParams(r, "", endpoint.Params.Body, endpoint.Params.Body.Root)
			if err != nil {
				return fmt.Errorf("Invalid requestObject '%s' for '%s %s'", err, endpoint.Method, endpoint.Path)
			}
		}

	} else if requestObject != nil {
		glog.Warningf("Request '%s %s' should not contain body params", endpoint.Method, endpoint.Path)
	}

	return nil
}

// extractBodyParams iterates over json and increase hits number in stats.Trie
func extractBodyParams(params interface{}, key string, body *stats.Trie, node *stats.Node) error {
	p, ok := params.(map[string]interface{})
	if !ok && node.Depth == 0 {
		return fmt.Errorf("%v", p)
	} else if !ok {
		return nil
	}

	if len(p) == 0 && node != body.Root {
		// include empty objects
		body.IncreaseHits(node)
		return nil
	}

	for k, v := range p {
		n := node.GetChild(k)
		// if child node does not exist then increase the current node
		// for instance, having a.b.c.d if 'c' does not have child 'd' then increase hits for 'c'
		if n == nil {
			n = node
		}

		switch obj := v.(type) {
		case map[string]interface{}:
			extractBodyParams(obj, k, body, n)
		case []interface{}:
			for _, v := range obj {
				extractBodyParams(v, k, body, n)
			}
		default:
			body.IncreaseHits(n)
		}
	}

	return nil
}

// calculateCoverage provides a total REST API and PATH:METHOD coverage number
func calculateCoverage(coverage *stats.Coverage) {
	for _, es := range coverage.Endpoints {
		for _, e := range es {
			e.UniqueHits = e.Query.UniqueHits + e.Body.UniqueHits

			if e.MethodCalled {
				e.UniqueHits++
			}
			// sometimes hit number is bigger than params number
			// for instance it might be caused by missing models definition
			// users have to make sure that their definitions are complete
			if e.UniqueHits > e.ExpectedUniqueHits {
				e.UniqueHits = e.ExpectedUniqueHits
			}

			if e.ExpectedUniqueHits > 0 {
				coverage.UniqueHits += e.UniqueHits
				e.Percent = float64(e.UniqueHits) * 100 / float64(e.ExpectedUniqueHits)
			} else {
				e.Percent = 0
			}
		}
	}

	if coverage.ExpectedUniqueHits > 0 {
		coverage.Percent = float64(coverage.UniqueHits) * 100 / float64(coverage.ExpectedUniqueHits)
	} else {
		coverage.Percent = 0
	}
}

// Print shows a generated report, if detailed it will show coverage for each endpoint
func Print(coverage *stats.Coverage, detailed bool) error {
	fmt.Printf("\nREST API coverage report:\n\n")
	if detailed {
		for p, es := range coverage.Endpoints {
			fmt.Println(p)
			for _, e := range es {
				fmt.Printf("%s:%.2f%%\t", strings.ToUpper(e.Method), e.Percent)
			}
			fmt.Print("\n\n")
		}
	}
	fmt.Printf("\nTotal coverage: %.2f%%\n\n", coverage.Percent)
	return nil
}

// Dump saves a generated report into a file in JSON format
func Dump(path string, coverage *stats.Coverage) error {
	jsonCov, err := json.Marshal(coverage)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, jsonCov, 0644)
}

// Generate provides a full REST API coverage report based on k8s audit log and swagger definition,
// by passing param "filter" you can limit the report to specific resources, as an example,
// "/apis/kubevirt.io/v1alpha3/" limits to kubevirt v1alpha3; "" no limit
func Generate(auditLogsPath string, swaggerPath string, filter string) (*stats.Coverage, error) {
	start := time.Now()
	defer glog.Infof("REST API coverage execution time: %s", time.Since(start))

	auditLogs, err := os.Open(auditLogsPath)
	if err != nil {
		return nil, err
	}

	sDocument, err := loads.JSONSpec(swaggerPath)
	if err != nil {
		return nil, err
	}

	coverage, err := analysis.AnalyzeSwagger(sDocument, filter)
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(auditLogs)
	for {
		var event auditv1.Event
		b, err := reader.ReadBytes('\n')
		if err == io.EOF {
			break
		}

		err = json.Unmarshal(b, &event)
		if err != nil {
			return nil, err
		}

		uri, err := url.Parse(event.RequestURI)
		if err != nil {
			return nil, err
		}

		path := getSwaggerPath(uri.Path, event.ObjectRef)
		if _, ok := coverage.Endpoints[path]; !ok {
			if filter == "" {
				glog.Errorf("Path '%s' not found in swagger", path)
			}
			continue
		}

		method := getHTTPMethod(event.Verb)
		if method == "" {
			glog.Errorf("Method '%s' not found for '%s' path", method, path)
			continue
		}

		if _, ok := coverage.Endpoints[path][method]; !ok {
			glog.Errorf("Method '%s' not found for '%s' path", method, path)
			continue
		}

		coverage.Endpoints[path][method].MethodCalled = true
		matchQueryParams(uri.Query(), coverage.Endpoints[path][method])
		err = matchBodyParams(event.RequestObject, coverage.Endpoints[path][method])
		if err != nil {
			glog.Errorf("%s", err)
		}
	}

	calculateCoverage(coverage)
	return coverage, nil
}
