#!/bin/bash -e
source hack/config.sh

cd tests
go test -master=http://$master_ip:$master_port "$@"
