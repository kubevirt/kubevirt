set -ex
eval "$(curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | GIMME_GO_VERSION=1.7.4 bash)"
pip install j2cli

export GOPATH=$PWD/go
export GOBIN=$PWD/go/bin
export PATH=$GOPATH/bin:$PATH
cd $GOPATH/src/kubevirt.io/kubevirt
go get -u github.com/kardianos/govendor
make test
