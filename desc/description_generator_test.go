package main

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

//Address doc
type Address struct {
	//Country doc
	Country string `json:"country,omitempty"`
	//PostCode doc
	PostCode int `json:"postcode,omitempty"`
}

//Person doc
//with multiline
type Person struct {
	//FirstName doc
	FirstName string `json:"firstName,omitempty"`
	//LastName doc
	LastName string `json:",omitempty"`
	//MiddleName doc
	MiddleName string `json:"middleName"`

	//Field without tag and
	// multiline comment and something to escape \ "
	HeightInPounds int
}

type UndocumentedStruct struct {
	UndocumentedField string
}

type StructWithEmbeddedStruct struct {
	Person
	Address
}

var expected = strings.TrimSpace(`
package main

func (Address) SwaggerDoc() map[string]string {
	return map[string]string{
		"": "Address doc",
		"country": "Country doc",
		"postcode": "PostCode doc",
	}
}

func (Person) SwaggerDoc() map[string]string {
	return map[string]string{
		"": "Person doc\nwith multiline",
		"firstName": "FirstName doc",
		"LastName": "LastName doc",
		"middleName": "MiddleName doc",
		"HeightInPounds": "Field without tag and\nmultiline comment and something to escape \\ \"",
	}
}

func (UndocumentedStruct) SwaggerDoc() map[string]string {
	return map[string]string{
	}
}

func (StructWithEmbeddedStruct) SwaggerDoc() map[string]string {
	return map[string]string{
	}
}
`)

func TestSwaggerDocGeneration(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "description_generator_test.go", nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	c := Parse(f)

	buf := ""
	for line := range c {
		buf = buf + line + "\n"
	}
	buf = strings.TrimSpace(buf)
	if buf != expected {
		t.Fatalf("Expected: %s, got: %s", expected, buf)
	}
}
