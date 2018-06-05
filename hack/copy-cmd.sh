#!/bin/bash

# this script is only used if non-containered build is performed using Makefile.nocontainer

source hack/config-default.sh

# loop over all sub-directories of cmd
for f in ${binaries}; do
    x=$(basename $f)
    # copy all binaries from the GOPATH/bin directory
    mkdir -p _out/cmd/${x}
    cp ${GOPATH}/bin/${x} _out/cmd/${x}
    # copy all other (non-code) content for "make docker"
    rsync -avzq --exclude "**/*.md" --exclude "**/*.go" --exclude "**/.*" ${f}/ _out/cmd/${x}
done
