#!/bin/bash
for file in $(find vendor/ -name "*_test.go"); do rm ${file}; done
rm -rf vendor/github.com/golang/glog
mkdir -p vendor/github.com/golang/
ln -s ../../../pkg/staging/glog/ vendor/github.com/golang/glog
