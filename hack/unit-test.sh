#!/bin/bash

set -e

go get github.com/mattn/goveralls
go get -v github.com/onsi/ginkgo/ginkgo
go get -v github.com/onsi/gomega
go get -v -t ./...
go get -u github.com/evanphx/json-patch 
export PATH=$PATH:$HOME/gopath/bin

ginkgo -r -cover
