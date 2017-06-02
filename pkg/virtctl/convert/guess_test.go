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

package convert_test

import (
	. "kubevirt.io/kubevirt/pkg/virtctl/convert"

	"bytes"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Guess", func() {
	table.DescribeTable("Should detect stream type", func(data string, detected Type) {
		b := []byte(data)
		reader, t := GuessStreamType(bytes.NewReader(b), 2048)
		Expect(t).To(Equal(detected))
		Expect(ioutil.ReadAll(reader)).To(Equal(b))
	},
		table.Entry("Should detect a XML stream", "    <xml></xml>", XML),
		table.Entry("Should detect a JSON stream", `    {"a": "b"}`, JSON),
		table.Entry("Should fall back to YAML if it is not a JSON or XML stream", `    a: "b"`, YAML),
	)
})
