# Template rendering
The operator will render all files in a directory that end with ".json" or ".yaml". The files will be passed through the [Go templating engine](https://golang.org/pkg/text/template/).

The aim is to mimic the parsing behavior of `kubectl create -f <dir>` as much as reasonably possible.
