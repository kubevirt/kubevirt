#!/bin/bash
#
# This is meant to very minimally mimic the Mariner RPMS SPEC file.
#

set -x

outdir=$1
cmds=${@:2}

if [ -z "$outdir" ] || [ -z "$cmds" ]; then
    echo "Usage: $0 <build-output-directory> <cmd1> <cmd2> ..."
    exit 1
fi

export GOFLAGS+=" -buildmode=pie"
./hack/build-go.sh install $cmds

mkdir -p $outdir
cp -a _out $outdir/
cp -a cmd $outdir/
