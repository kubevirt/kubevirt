#!/usr/bin/env bash

set -e

# use time to record the elapsed time for the step
shift # get rid of the '-c' supplied by make.

if [[ -n ${KUBEVIRT_PROFILE_MAKE} ]]; then
    date
fi
/bin/bash -c "$*"
