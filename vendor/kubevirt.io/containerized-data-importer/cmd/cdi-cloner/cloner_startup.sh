#!/bin/sh

#Copyright 2018 The CDI Authors.
#
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.

set -euo pipefail

if [ $# != 2 ]; then
    echo "cloner: 2 args are supported: source|target and socket name"
    exit 1
fi
obj="$1"      # source|target
rand_dir="$2" # part of socket path

pipe_dir="/tmp/clone/socket/$rand_dir/pipe"
image_dir="/tmp/clone/image"
retries=0
max_retries=20
sleep_time=3

if [ "$obj" == "source" ]; then
    echo "cloner: Starting clone source"
    echo "cloner: creating fifo pipe"
    mkfifo $pipe_dir
    echo "cloner: creating tarball of the image and redirecting it to $pipe_dir"
    pushd $image_dir
	#figure out the size of content in the directory
    size=$(du -sb . | cut -f1)
    echo $size
    #Write the size to the pipe so the other end can read it.
    echo "$size" >$pipe_dir
    tar cv --sparse . >$pipe_dir
    popd
    echo "cloner: finished writing image to $pipe_dir"
    exit 0
fi

if [ "$obj" == "target" ]; then
    echo "cloner: Starting clone target"
    while true; do
        echo "cloner: check if the fifo pipe was created by the cloning source pod"
        if [ -e "$pipe_dir" ]; then
            pushd $image_dir
            echo "cloner: extract the image from $pipe_dir into $image_dir directory"
            /usr/bin/cdi-cloner -pipedir $pipe_dir -alsologtostderr -v=3
            popd
        	if [ "$?" != "0" ]; then
        		echo "cloner: failed with exit code $?"
        		exit 1
        	fi
            echo "cloner: finished cloning image from $pipe_dir to $image_dir"
            exit 0
        fi
        if ((retries == max_retries)); then
            echo "cloner: failed after $retries retries to clone image"
            exit 1
        fi
        echo "cloner: $retries: fifo pipe has not been created by the source pod. Waiting $sleep_time seconds before checking again..."
        sleep $sleep_time
        let retries+=1
    done
fi

echo "cloner: argument \"$obj\" is wrong; expect 'source' or 'target'"
exit 1
