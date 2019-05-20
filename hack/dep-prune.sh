#!/usr/bin/env bash
for file in $(find vendor/ -name "*_test.go"); do rm ${file}; done
rm -rf vendor/github.com/golang/glog
mkdir -p vendor/github.com/golang/
ln -s ../../../pkg/staging/glog/ vendor/github.com/golang/glog
#create link of client-go from staging
ln -s ../../staging/src/kubevirt.io/client-go vendor/kubevirt.io/client-go
