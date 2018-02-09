#!/bin/bash
set -e

wget -q https://storage.googleapis.com/golang/go1.6.2.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.6.2.linux-amd64.tar.gz

apt-get update
apt-get -y install git mercurial apache2-utils

echo 'export PATH=$PATH:/usr/local/go/bin:/go/bin
export GOPATH=/go' >> /home/vagrant/.profile

source /home/vagrant/.profile

go get golang.org/x/tools/cmd/goimports
go get github.com/golang/lint/golint
go get github.com/smartystreets/goconvey/convey
go get github.com/cactus/go-statsd-client/statsd
go get github.com/rcrowley/go-metrics
go get github.com/DataDog/datadog-go/statsd

chown -R vagrant:vagrant /go
