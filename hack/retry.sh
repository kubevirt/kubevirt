#!/usr/bin/env bash

if [ $# -ne 3 ]; then
    echo 'Usage: retry <num retries> <wait retry secs> "<command>"'
    exit 1
fi

retries=$1
wait_retry=$2
command=$3

for i in `seq 1 $retries`; do
    echo "$command"
    bash -c "$command"
    ret_value=$?
    [ $ret_value -eq 0 ] && break
    echo "> failed with exit code $ret_value, waiting $wait_retry seconds to retry..."
    sleep $wait_retry
done

exit $ret_value