#!/usr/bin/env bash

if [ $# -lt 3 ] || [ $# -gt 4 ]; then
    echo 'Usage: retry <num retries> <wait retry secs> "<command>"'
    exit 1
fi

retries=$1
wait_retry=$2
command=$3
debug_cmd=$4

for i in `seq 1 $retries`; do
    echo "$command"
    bash -c "$command"
    ret_value=$?
    [ $ret_value -eq 0 ] && break
    echo "> failed with exit code $ret_value, waiting $wait_retry seconds to retry..."
    sleep $wait_retry
done

if [[ ${ret_value} -ne 0 ]]; then
  if [[ -n ${debug_cmd} ]]; then
    echo "${debug_cmd}"
    bash -c "${debug_cmd}"
  fi
fi

exit $ret_value