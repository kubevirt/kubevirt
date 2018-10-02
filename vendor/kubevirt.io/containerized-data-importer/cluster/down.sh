#!/bin/bash -e

source ./cluster/gocli.sh

echo "Bringing down cluster and client ..."
$gocli rm

# clean up unused docker volumes
danglingVols="$(docker volume ls -qf dangling=true)"
if [[ $? == 0 && -n "$danglingVols" ]]; then
    echo "Cleaning up docker and dangling volumes ..."
    docker volume rm $danglingVols
fi

