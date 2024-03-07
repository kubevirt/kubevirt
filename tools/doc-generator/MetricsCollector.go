package main

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"

	domainstats "kubevirt.io/kubevirt/pkg/monitoring/domainstats/prometheus" // import for prometheus metrics
	_ "kubevirt.io/kubevirt/pkg/virt-controller/watch"
)

type metric struct {
	name        string
	description string
	mType       string
}

type metricList []metric

func getMetricsNotIncludeInEndpointByDefault() metricList {
	metrics := metricList{
		{
			name:        domainstats.MigrateVmiDataProcessedMetricName,
			description: "The total Guest OS data processed and migrated to the new VM.",
			mType:       "Gauge",
		},
		{
			name:        domainstats.MigrateVmiDataRemainingMetricName,
			description: "The remaining guest OS data to be migrated to the new VM.",
			mType:       "Gauge",
		},
		{
			name:        domainstats.MigrateVmiDirtyMemoryRateMetricName,
			description: "The rate of memory being dirty in the Guest OS.",
			mType:       "Gauge",
		},
		{
			name:        domainstats.MigrateVmiMemoryTransferRateMetricName,
			description: "The rate at which the memory is being transferred.",
			mType:       "Gauge",
		},
	}

	return metrics
}

// Len implements sort.Interface.Len
func (m metricList) Len() int {
	return len(m)
}

// Less implements sort.Interface.Less
func (m metricList) Less(i, j int) bool {
	return m[i].name < m[j].name
}

// Swap implements sort.Interface.Swap
func (m metricList) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func parseMetricDesc(line string) (string, string) {
	split := strings.Split(line, " ")
	name := split[2]
	split[3] = strings.Title(split[3])
	description := strings.Join(split[3:], " ")
	return name, description
}

func parseMetricType(scan *bufio.Scanner, name string) string {
	for scan.Scan() {
		typeLine := scan.Text()
		if strings.HasPrefix(typeLine, "# TYPE ") {
			split := strings.Split(typeLine, " ")
			if split[2] == name {
				return strings.Title(split[3])
			}
		}
	}
	return ""
}

func parseVirtMetrics(r io.Reader, metrics *metricList) error {
	scan := bufio.NewScanner(r)
	for scan.Scan() {
		helpLine := scan.Text()
		if strings.HasPrefix(helpLine, "# HELP ") {
			if strings.Contains(helpLine, "kubevirt_") {
				metName, metDesc := parseMetricDesc(helpLine)
				metType := parseMetricType(scan, metName)
				*metrics = append(*metrics, metric{name: metName, description: metDesc, mType: metType})
			}
		}
	}

	if scan.Err() != nil {
		return fmt.Errorf("failed to parse metrics from prometheus endpoint, %w", scan.Err())
	}

	sort.Sort(metrics)

	// remove duplicates
	for i := 0; i < len(*metrics)-1; i++ {
		if (*metrics)[i].name == (*metrics)[i+1].name {
			*metrics = append((*metrics)[:i], (*metrics)[i+1:]...)
			i--
		}
	}

	return nil
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
