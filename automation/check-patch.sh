#
# This file is part of the kubevirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2017 Red Hat, Inc.
#

set -ex
eval "$(curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | GIMME_GO_VERSION=1.7.4 bash)"
pip install j2cli

export GOPATH=$PWD/go
export GOBIN=$PWD/go/bin
export PATH=$GOPATH/bin:$PATH
cd $GOPATH/src/kubevirt.io/kubevirt
go get -u github.com/kardianos/govendor
make test
