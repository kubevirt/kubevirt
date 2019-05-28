#!/bin/bash -e

source ./cluster/gocli.sh

echo 'Bringing down cluster and client ...'
$gocli rm

# clean up unused docker volumes
dangling_volumes="$(docker volume ls -qf dangling=true)"
if [[ $? == 0 && -n "$dangling_volumes" ]]; then
    echo 'Cleaning up docker and dangling volumes ...'
    docker volume rm $dangling_volumes
fi
