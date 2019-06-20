#!/usr/bin/env bash

rm -rf vendor/github.com/golang/glog
mkdir -p vendor/github.com/golang/
ln -s ../../../pkg/staging/glog/ vendor/github.com/golang/glog

# create symbolic link on client-go package to avoid duplication
rm -rf vendor/kubevirt.io/client-go
mkdir -p vendor/kubevirt.io
ln -s ../../staging/src/kubevirt.io/client-go/ vendor/kubevirt.io/client-go
