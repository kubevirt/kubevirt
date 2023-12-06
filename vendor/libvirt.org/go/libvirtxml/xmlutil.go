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

package libvirtxml

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

type element struct {
	XMLNS    string
	Name     string
	Attrs    map[string]string
	Content  string
	Children []*element
}

type elementstack []*element

func (s *elementstack) push(v *element) {
	*s = append(*s, v)
}

func (s *elementstack) pop() *element {
	res := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return res
}

func getNamespaceURI(xmlnsMap map[string]string, xmlns string, name xml.Name) string {
	if name.Space != "" {
		uri, ok := xmlnsMap[name.Space]
		if !ok {
			return "undefined://" + name.Space
		} else {
			return uri
		}
	} else {
		return xmlns
	}
}

func xmlName(xmlns string, name xml.Name) string {
	if xmlns == "" {
		return name.Local
	}
	return name.Local + "(" + xmlns + ")"
}

func loadXML(xmlstr string, ignoreNSDecl bool) (*element, error) {
	xmlnsMap := make(map[string]string)
	xmlr := strings.NewReader(xmlstr)

	d := xml.NewDecoder(xmlr)
	var root *element
	stack := elementstack{}
	for {
		t, err := d.RawToken()
		if err != nil {
			return nil, err
		}

		var parent *element
		if root != nil {
			if len(stack) == 0 {
				return nil, fmt.Errorf("Unexpectedly empty stack")
			}
			parent = stack[len(stack)-1]
		}

		switch t := t.(type) {
		case xml.StartElement:
			xmlns := ""
			if parent != nil {
				xmlns = parent.XMLNS
			}
			for _, a := range t.Attr {
				if a.Name.Space == "xmlns" {
					xmlnsMap[a.Name.Local] = a.Value
				} else if a.Name.Space == "" && a.Name.Local == "xmlns" {
					xmlns = a.Value
				}
			}
			xmlns = getNamespaceURI(xmlnsMap, xmlns, t.Name)
			child := &element{
				XMLNS: xmlns,
				Name:  xmlName(xmlns, t.Name),
				Attrs: make(map[string]string),
			}

			for _, a := range t.Attr {
				if a.Name.Space == "xmlns" {
					continue
				}
				if a.Name.Space == "" && a.Name.Local == "xmlns" {
					continue
				}
				attrNS := getNamespaceURI(xmlnsMap, "", a.Name)
				child.Attrs[xmlName(attrNS, a.Name)] = a.Value
			}
			stack.push(child)
			if root == nil {
				root = child
			} else {
				parent.Children = append(parent.Children, child)
				parent.Content = ""
			}
		case xml.EndElement:
			stack.pop()
		case xml.CharData:
			if parent != nil && len(parent.Children) == 0 {
				val := string(t)
				if strings.TrimSpace(val) != "" {
					parent.Content = val
				}
			}
		}

		if root != nil && len(stack) == 0 {
			break
		}
	}

	return root, nil
}

func testCompareValue(filename, path, key, expected, actual string) error {
	if expected == actual {
		return nil
	}

	i1, err1 := strconv.ParseInt(expected, 0, 64)
	i2, err2 := strconv.ParseInt(actual, 0, 64)
	if err1 == nil && err2 == nil && i1 == i2 {
		return nil
	}
	path = path + "/@" + key
	return fmt.Errorf("%s: %s: attribute actual value '%s' does not match expected value '%s'",
		filename, path, actual, expected)
}

func testCompareElement(filename, expectPath, actualPath string, expect, actual *element, extraExpectNodes, extraActualNodes map[string]bool) error {
	if expect.Name != actual.Name {
		return fmt.Errorf("%s: name '%s' doesn't match '%s'",
			expectPath, expect.Name, actual.Name)
	}

	expectAttr := expect.Attrs
	for key, val := range actual.Attrs {
		expectval, ok := expectAttr[key]
		if !ok {
			attrPath := actualPath + "/@" + key
			if _, ok := extraActualNodes[attrPath]; ok {
				continue
			}
			return fmt.Errorf("%s: %s: attribute in actual XML missing in expected XML",
				filename, attrPath)
		}
		err := testCompareValue(filename, actualPath, key, expectval, val)
		if err != nil {
			return err
		}
		delete(expectAttr, key)
	}
	for key, _ := range expectAttr {
		attrPath := expectPath + "/@" + key
		if _, ok := extraExpectNodes[attrPath]; ok {
			continue
		}
		return fmt.Errorf("%s: %s: attribute '%s'  in expected XML missing in actual XML",
			filename, attrPath, expectAttr[key])
	}

	if expect.Content != actual.Content {
		return fmt.Errorf("%s: %s: actual content '%s' does not match expected '%s'",
			filename, actualPath, actual.Content, expect.Content)
	}

	used := make([]bool, len(actual.Children))
	expectChildIndexes := make(map[string]uint)
	actualChildIndexes := make(map[string]uint)
	for _, expectChild := range expect.Children {
		expectIndex, _ := expectChildIndexes[expectChild.Name]
		expectChildIndexes[expectChild.Name] = expectIndex + 1
		subExpectPath := fmt.Sprintf("%s/%s[%d]", expectPath, expectChild.Name, expectIndex)

		var actualChild *element = nil
		for i := 0; i < len(used); i++ {
			if !used[i] && actual.Children[i].Name == expectChild.Name {
				actualChild = actual.Children[i]
				used[i] = true
				break
			}
		}
		if actualChild == nil {
			if _, ok := extraExpectNodes[subExpectPath]; ok {
				continue
			}
			return fmt.Errorf("%s: %s: element in expected XML missing in actual XML",
				filename, subExpectPath)
		}

		actualIndex, _ := actualChildIndexes[actualChild.Name]
		actualChildIndexes[actualChild.Name] = actualIndex + 1
		subActualPath := fmt.Sprintf("%s/%s[%d]", actualPath, actualChild.Name, actualIndex)

		err := testCompareElement(filename, subExpectPath, subActualPath, expectChild, actualChild, extraExpectNodes, extraActualNodes)
		if err != nil {
			return err
		}
	}

	actualChildIndexes = make(map[string]uint)
	for i, actualChild := range actual.Children {
		actualIndex, _ := actualChildIndexes[actualChild.Name]
		actualChildIndexes[actualChild.Name] = actualIndex + 1
		if used[i] {
			continue
		}
		subActualPath := fmt.Sprintf("%s/%s[%d]", actualPath, actualChild.Name, actualIndex)

		if _, ok := extraActualNodes[subActualPath]; ok {
			continue
		}
		return fmt.Errorf("%s: %s: element in actual XML missing in expected XML",
			filename, subActualPath)
	}

	return nil
}

func makeExtraNodeMap(nodes []string) map[string]bool {
	ret := make(map[string]bool)
	for _, node := range nodes {
		ret[node] = true
	}
	return ret
}

func testCompareXML(filename, expectStr, actualStr string, extraExpectNodes, extraActualNodes []string) error {
	extraExpectNodeMap := makeExtraNodeMap(extraExpectNodes)
	extraActualNodeMap := makeExtraNodeMap(extraActualNodes)

	//fmt.Printf("%s\n", expectedstr)
	expectRoot, err := loadXML(expectStr, true)
	if err != nil {
		return err
	}
	//fmt.Printf("%s\n", actualstr)
	actualRoot, err := loadXML(actualStr, true)
	if err != nil {
		return err
	}

	if expectRoot.Name != actualRoot.Name {
		return fmt.Errorf("%s: /: expected root element '%s' does not match actual '%s'",
			filename, expectRoot.Name, actualRoot.Name)
	}

	err = testCompareElement(filename, "/"+expectRoot.Name+"[0]", "/"+actualRoot.Name+"[0]", expectRoot, actualRoot, extraExpectNodeMap, extraActualNodeMap)
	if err != nil {
		return err
	}

	return nil
}
