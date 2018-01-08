#!/bin/bash

goveralls -service=travis-ci -package=./pkg/... -ignore=$(find -regextype posix-egrep -regex ".*generated_mock.*\.go|.*swagger_generated\.go" -printf "%P\n" | paste -d, -s) -v
