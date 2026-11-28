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
 *
 */

package auth

import (
	"bytes"
	"encoding/binary"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Identity", func() {
	identity := Identity{Namespace: "default", Name: "testvmi", UID: "1234"}

	frame := func(payload string) *bytes.Buffer {
		buf := &bytes.Buffer{}
		Expect(binary.Write(buf, binary.BigEndian, uint32(len(payload)))).To(Succeed())
		buf.WriteString(payload)
		return buf
	}

	It("should be derived from the VMI", func() {
		vmi := v1.NewVMIReferenceWithUUID("default", "testvmi", "1234")
		Expect(IdentityFromVMI(vmi)).To(Equal(identity))
	})

	It("should be read back as written and leave the following bytes untouched", func() {
		buf := &bytes.Buffer{}
		Expect(WriteIdentity(buf, identity)).To(Succeed())
		buf.WriteString("Hello")

		Expect(ReadIdentity(buf)).To(Equal(identity))
		Expect(buf.String()).To(Equal("Hello"))
	})

	DescribeTable("should refuse to write an incomplete identity", func(id Identity) {
		buf := &bytes.Buffer{}
		Expect(WriteIdentity(buf, id)).To(MatchError(ContainSubstring("incomplete identity")))
		Expect(buf.Len()).To(BeZero())
	},
		Entry("without namespace", Identity{Name: "testvmi", UID: "1234"}),
		Entry("without name", Identity{Namespace: "default", UID: "1234"}),
		Entry("without UID", Identity{Namespace: "default", Name: "testvmi"}),
	)

	DescribeTable("should fail to read", func(input *bytes.Buffer, expectedErr string) {
		_, err := ReadIdentity(input)
		Expect(err).To(MatchError(ContainSubstring(expectedErr)))
	},
		Entry("an empty stream", &bytes.Buffer{}, "failed to read identity length"),
		Entry("a truncated identity", bytes.NewBuffer(frame(`{"namespace":"default"}`).Bytes()[:10]), "failed to read identity"),
		Entry("an oversized identity", frame(strings.Repeat("a", maxIdentitySize+1)), "exceeds"),
		Entry("a gRPC client preface instead of an identity", bytes.NewBufferString("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"), "exceeds"),
		Entry("a malformed identity", frame("{"), "failed to decode identity"),
		Entry("an incomplete identity", frame(`{"namespace":"default","name":"testvmi"}`), "incomplete identity"),
	)

	It("should verify a matching object", func() {
		Expect(identity.Verify("default", "testvmi", "1234")).To(Succeed())
	})

	DescribeTable("should reject an object", func(namespace, name string, uid types.UID) {
		Expect(identity.Verify(namespace, name, uid)).To(MatchError(ContainSubstring("does not belong to client")))
	},
		Entry("in another namespace", "other", "testvmi", types.UID("1234")),
		Entry("with another name", "default", "other", types.UID("1234")),
		Entry("with another UID", "default", "testvmi", types.UID("5678")),
		Entry("without UID", "default", "testvmi", types.UID("")),
	)
})
