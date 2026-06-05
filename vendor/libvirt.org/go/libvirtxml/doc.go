/*
 * This file is part of the libvirt-go-xml-module project
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 * Copyright (C) 2017 Red Hat, Inc.
 *
 */

// Package libvirt-go-xml-module defines structs for parsing libvirt XML schemas
//
// The libvirt API uses XML schemas/documents to describe the configuration
// of many of its managed objects. Thus when using the libvirt-go package,
// it is often neccessary to either parse or format XML documents. This
// package defines a set of Go structs which have been annotated for use
// with the encoding/xml API to manage libvirt XML documents.
//
// Example creating a domain XML document from configuration:
//
//	package main
//
//	import (
//	 "libvirt.org/go/libvirtxml"
//	)
//
//	func main() {
//	  domcfg := &libvirtxml.Domain{Type: "kvm", Name: "demo",
//	                               UUID: "8f99e332-06c4-463a-9099-330fb244e1b3",
//	                               ....}
//	  xmldoc, err := domcfg.Marshal()
//	}
//
// Example parsing a domainXML document, in combination with libvirt-go
//
//	package main
//
//	import (
//	  "libvirt.org/go/libvirt"
//	  "libvirt.org/go/libvirtxml"
//	  "fmt"
//	)
//
//	func main() {
//	  conn, err := libvirt.NewConnect("qemu:///system")
//	  dom, err := conn.LookupDomainByName("demo")
//	  xmldoc, err := dom.GetXMLDesc(0)
//
//	  domcfg := &libvirtxml.Domain{}
//	  err = domcfg.Unmarshal(xmldoc)
//
//	  fmt.Printf("Virt type %s\n", domcfg.Type)
//	}
package libvirtxml
