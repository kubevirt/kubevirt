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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package main

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[tools]test-validator", func() {

	It("should return the same test names on duplicate It with By", func() {
		Expect(expandTestNames("", []*ginkgoNode{
			{
				ginkgoMetadata: ginkgoMetadata{
					Name: "It",
					Text: "[test_id:4641]should be shut down when the watchdog expires",
					Spec: true,
				},
				Nodes: []*ginkgoNode{
					{
						ginkgoMetadata: ginkgoMetadata{
							Name: "By",
							Text: "Expecting the VirtualMachineInstance console",
							Spec: false,
						},
						Nodes: []*ginkgoNode{},
					},
				},
			},
			{
				ginkgoMetadata: ginkgoMetadata{
					Name: "It",
					Text: "[test_id:4641]should be shut down when the watchdog expires",
					Spec: true,
				},
				Nodes: []*ginkgoNode{
					{
						ginkgoMetadata: ginkgoMetadata{
							Name: "By",
							Text: "Expecting the VirtualMachineInstance console",
							Spec: false,
						},
						Nodes: []*ginkgoNode{},
					},
				},
			},
		})).To(BeEquivalentTo([]string{
			"[test_id:4641]should be shut down when the watchdog expires",
			"[test_id:4641]should be shut down when the watchdog expires",
		}))
	})

	It("should return the expanded test description", func() {
		Expect(expandTestNames("", []*ginkgoNode{
			{
				ginkgoMetadata: ginkgoMetadata{
					Name: "Describe",
					Text: "[sig-compute] parent description",
				},
				Nodes: []*ginkgoNode{
					{
						ginkgoMetadata: ginkgoMetadata{
							Name: "Context",
							Text: "[ref_id:4217] the reference context",
						},
						Nodes: []*ginkgoNode{
							{
								ginkgoMetadata: ginkgoMetadata{
									Name: "It",
									Text: "[test_id:1742] what makes this so special?",
									Spec: true,
								},
								Nodes: nil,
							},
						},
					},
				}}})).To(BeEquivalentTo([]string{
			"[sig-compute] parent description [ref_id:4217] the reference context [test_id:1742] what makes this so special?",
		}))
	})

	It("should not return the expanded test description including the By, but should return something", func() {
		Expect(expandTestNames("", []*ginkgoNode{
			{
				ginkgoMetadata: ginkgoMetadata{
					Name: "Describe",
					Text: "[sig-compute] parent description",
				},
				Nodes: []*ginkgoNode{
					{
						ginkgoMetadata: ginkgoMetadata{
							Name: "Context",
							Text: "[ref_id:4217] the reference context",
						},
						Nodes: []*ginkgoNode{
							{
								ginkgoMetadata: ginkgoMetadata{
									Name: "It",
									Text: "[test_id:1742] what makes this so special?",
									Spec: true,
								},
								Nodes: []*ginkgoNode{
									{
										ginkgoMetadata: ginkgoMetadata{
											Name: "By",
											Text: "Expecting somthing to happen after this",
										},
										Nodes: []*ginkgoNode{},
									},
								},
							},
						},
					},
				}}})).To(BeEquivalentTo([]string{
			"[sig-compute] parent description [ref_id:4217] the reference context [test_id:1742] what makes this so special?",
		}))
	})

	It("should return the same test names on duplicate It with By", func() {
		Expect(expandTestNames("", []*ginkgoNode{
			{
				ginkgoMetadata: ginkgoMetadata{
					Name: "It",
					Text: "[test_id:1742] what makes this so special?",
					Spec: true,
				},
				Nodes: nil,
			},
			{
				ginkgoMetadata: ginkgoMetadata{
					Name: "It",
					Text: "[test_id:1742] what makes this so special?",
					Spec: true,
				},
				Nodes: nil,
			},
		})).To(BeEquivalentTo([]string{
			"[test_id:1742] what makes this so special?",
			"[test_id:1742] what makes this so special?",
		}))
	})

	It("don't return entries with description 'undefined' on tables (i.e. where the description is referencing a const)", func() {
		Expect(expandTestNames("", []*ginkgoNode{
			{
				ginkgoMetadata: ginkgoMetadata{
					Name: "DescribeTable",
					Text: "[sig-compute] table description",
				},
				Nodes: []*ginkgoNode{
					{
						ginkgoMetadata: ginkgoMetadata{
							Name: "Entry",
							Text: "[test_id:1234] first entry",
							Spec: true,
						},
						Nodes: nil,
					},
					{
						ginkgoMetadata: ginkgoMetadata{
							Name: "Entry",
							Text: "undefined",
							Spec: true,
						},
						Nodes: nil,
					},
					{
						ginkgoMetadata: ginkgoMetadata{
							Name: "Entry",
							Text: "[test_id:1234] third entry",
							Spec: true,
						},
						Nodes: nil,
					},
				}}})).To(BeEquivalentTo([]string{
			"[sig-compute] table description [test_id:1234] first entry",
			"[sig-compute] table description [test_id:1234] third entry",
		}))
	})

})

func TestTestValidator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TestTestValidator suite")
}
