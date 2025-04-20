/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 */

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
