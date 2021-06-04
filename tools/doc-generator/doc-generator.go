package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"

	domainstats "kubevirt.io/kubevirt/pkg/monitoring/domainstats/prometheus" // import for prometheus metrics
	fake "kubevirt.io/kubevirt/pkg/monitoring/domainstats/prometheus"
	_ "kubevirt.io/kubevirt/pkg/virt-controller/watch"
)

func main() {
	req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
	checkError(err)

	recorder := httptest.NewRecorder()

	handler := domainstats.Handler(1)

	fake.RegisterFakeCollector()

	handler.ServeHTTP(recorder, req)

	if status := recorder.Code; status == http.StatusOK {
		linesWithHelp := []string{"kubevirt_", "_virt_controller"}
		metrics := parseVirtMetrics(recorder.Body, linesWithHelp)
		writeToFile(metrics)
	} else {
		panic(recorder.Code)
	}
}

func parseVirtMetrics(r io.Reader, contains []string) map[string]string {
	metrics := map[string]string{}
	scan := bufio.NewScanner(r)
	for scan.Scan() {
		for _, contain := range contains {
			help := scan.Text()
			if strings.Contains(help, contain) && strings.HasPrefix(help, "# HELP ") {
				name := strings.Split(help, " ")[2]

				metrics[name] = help

			}
		}
	}
	if scan.Err() != nil {
		panic(fmt.Errorf("Failed to parse metrics from prometheus endpoint, %v", scan.Err()))
	}
	return metrics
}

func writeToFile(metrics map[string]string) {
	old, err := os.Open("metrics.md")
	checkError(err)
	defer old.Close()

	new, err := os.Create("newmetrics.md")
	checkError(err)
	defer new.Close()

	var write string

	format := "###%s"

	buf := bufio.NewWriter(new)
	scan := bufio.NewScanner(old)
	for scan.Scan() {
		line := scan.Text()
		if write != "" {
			if strings.Contains(line, "HELP") {
				if line != write {
					buf.WriteString(write)
					line = ""
				}

			} else {
				buf.WriteString(write)
				buf.WriteString("\n")
			}
			write = ""

		}

		buf.WriteString(line + "\n")
		for k, v := range metrics {
			if strings.Contains(line, k) {
				write = fmt.Sprintf(format, v)
				delete(metrics, k)
				break
			}
		}
	}
	if scan.Err() != nil {
		panic(fmt.Errorf("Failed to update metric doc, %v", scan.Err()))
	}

	if len(metrics) > 0 {
		buf.WriteString("\n # Other Metrics \n")
	}

	var keys []string
	for k := range metrics {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	for _, k := range keys {
		buf.WriteString("## " + k + "\n")
		buf.WriteString("###" + metrics[k] + "\n")
	}
	buf.Flush()
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
