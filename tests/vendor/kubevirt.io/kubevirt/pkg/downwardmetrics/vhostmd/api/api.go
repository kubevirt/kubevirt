package api

import "encoding/xml"

type MetricContext string
type MetricType string

const (
	MetricContextHost MetricContext = "host"
	MetricContextVM   MetricContext = "vm"
)

const (
	MetricTypeReal64 MetricType = "real64"
	MetricTypeReal32 MetricType = "real32"
	MetricTypeInt64  MetricType = "int64"
	MetricTypeInt32  MetricType = "int32"
	MetricTypeUInt64 MetricType = "uint64"
	MetricTypeUInt32 MetricType = "uint32"
	MetricTypeString MetricType = "string"
)

type Metrics struct {
	XMLName xml.Name `xml:"metrics"`
	Text    string   `xml:",chardata"`
	Metrics []Metric `xml:"metric"`
}

type Metric struct {
	Text    string        `xml:",chardata"`
	Type    MetricType    `xml:"type,attr"`
	Context MetricContext `xml:"context,attr"`
	Name    string        `xml:"name"`
	Value   string        `xml:"value"`
	Unit    string        `xml:"unit,attr,omitempty"`
}
