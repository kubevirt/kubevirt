# editorconfig

[![GoDoc](https://godoc.org/mvdan.cc/editorconfig?status.svg)](https://godoc.org/mvdan.cc/editorconfig)

A small package to parse and use [EditorConfig][1] files. Currently passes all
of the official [test cases][2], which are run via `go test`.

```go
props, err := editorconfig.Find("path/to/file.go")
if err != nil { ... }

// Print all the properties
fmt.Println(props)

// Query specific ones
fmt.Println(props.Get("indent_style"))
fmt.Println(props.IndentSize())
```

Note that an [official library][3] exists for Go. This alternative
implementation started with a different design:

* Specialised INI parser, for full compatibility with the spec
* Ability to cache parsing files and compiling pattern matches
* Storing and querying all properties equally
* Minimising pointers and maps to store data

[1]: https://editorconfig.org/
[2]: https://github.com/editorconfig/editorconfig-core-test
[3]: https://github.com/editorconfig/editorconfig-core-go
