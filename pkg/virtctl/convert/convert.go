/*
 * This file is part of the kubevirt project
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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package convert

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"fmt"

	flag "github.com/spf13/pflag"
)

func NewConvertCommand() *Convert {
	return &Convert{stdin: os.Stdin, stdout: os.Stdout}
}

const (
	RunSucceeded int = 0
	RunFailed    int = 1
)

type Convert struct {
	stdin  io.ReadCloser
	stdout io.Writer
}

func (c *Convert) FlagSet() *flag.FlagSet {
	cf := flag.NewFlagSet("convert", flag.ExitOnError)
	cf.StringP("filename", "f", "", "Filename, directory, or URL to files identifying the resource to get from a server.")
	cf.StringP("output", "o", "json", "Output format. One of: json|yaml|xml.")
	cf.String("output-version", "kubevirt.io/v1alpha1", "Output the formatted object with the given group version")
	return cf
}

func (c *Convert) Usage() string {
	usage := "Convert between KubeVirt and Libvirt VM representations:\n\n"
	usage += "Examples:\n"
	usage += "# Convert Domain xml to yaml via stdin\n"
	usage += "virsh dumpxml testvm | virtctl convert-spec -f - -o yaml\n\n"
	usage += "# Convert yaml into json from a http resource\n"
	usage += "virtctl convert-spec -f http://127.0.0.1:4012/vm.yaml -o json\n\n"
	usage += "# Convert VM specification from a yaml file into a Libvirt Domain XML\n"
	usage += "virtctl convert-spec -f vm.yaml -o xml > dom.xml\n\n"
	usage += "Options:\n"
	usage += c.FlagSet().FlagUsages()
	return usage
}

func (c *Convert) Run(flags *flag.FlagSet) int {
	sourceName, _ := flags.GetString("filename")

	outputFlag, _ := flags.GetString("output")
	outputFormat := Type(outputFlag)

	if sourceName == "" {
		log.Println("No source specified")
		return RunFailed
	}

	rawSource, err := c.open(sourceName)
	if err != nil {
		log.Println("Failed to open source", err)
		return RunFailed
	}
	defer rawSource.Close()

	source, inputFormat := GuessStreamType(rawSource, 2048)

	var encoder Encoder
	var decoder Decoder

	if outputFormat == UNSPECIFIED {
		outputFormat = JSON
	}

	switch inputFormat {
	case XML:
		decoder = fromXML
	case JSON, YAML:
		decoder = fromYAMLOrJSON
	default:
		log.Printf("Unsupported input format '%s'\n", inputFormat)
		return RunFailed
	}

	switch outputFormat {
	case XML:
		encoder = toXML
	case JSON:
		encoder = toJSON
	case YAML:
		encoder = toYAML
	default:
		log.Printf("Unsupported output format '%s'\n", outputFormat)
		return RunFailed
	}

	vm, err := decoder(source)
	if err != nil {
		log.Println("Failed to decode struct", err)
		return RunFailed
	}
	err = encoder(vm, c.stdout)
	if err != nil {
		log.Println("Failed to encode struct", err)
		return RunFailed
	}
	fmt.Fprint(c.stdout, "\n")
	return RunSucceeded
}

func (c *Convert) open(sourceName string) (io.ReadCloser, error) {
	if sourceName == "-" {
		return c.stdin, nil
	} else if strings.HasPrefix(sourceName, "http") {
		resp, err := http.Get(sourceName)
		if err != nil {
			return nil, err
		}
		return resp.Body, nil

	} else {
		return os.Open(sourceName)
	}
}
