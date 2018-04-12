#!/bin/bash
for file in $(find vendor/ -name "*_test.go"); do rm ${file}; done
