#!/usr/bin/env bash

function usage {
    cat <<EOF
usage: $0 [-h|--help] show help
       $0 [num_shards [shard_index]]

       runs a shard of the e2e tests as in 'repeated_test.sh'. Shard to run is automatically determined by time of day. Default is four shards, where
       the first shard would be run from 0-5AM, 2nd would be run from 6-11AM ....

       Note: this script is calling 'repeated_test.sh' to run the tests.
EOF
}

function find_tests {
    find tests/ -name '*.go' | grep -vE '^tests/(assert|framework|utils.go|lib|performance|realtime)|_suite(_test)\.go'
}

function main {
    if [[ $# -gt 0 ]] && [[ $1 =~ -h ]]; then
        usage
        exit 0
    fi

    set -x

    num_shards=${1-4}
    hours_per_shard=$(( 24 / num_shards ))
    if [[ $# -lt 2 ]]; then
        shard_index=$(( $(date +%H) / hours_per_shard ))
    else
        shard_index=${2}
        if [[ $shard_index -ge $num_shards ]]; then
            echo "num_shards=$num_shards <= $shard_index!"
            exit 1
        fi
    fi

    num_lines=$(find_tests | wc -l)
    if [[ ${num_lines} -eq 0 ]]; then
        echo "No tests"
        exit 0
    fi
    num_lines_per_segment=$(( num_lines / num_shards ))
    start_line=$(( shard_index * num_lines_per_segment ))
    end_line=$(( (shard_index + 1) * num_lines_per_segment ))
    line_index=0
    shard_tests=()
    for line in $(find_tests); do
        if [[ $line_index -ge $start_line ]] && [[ $line_index -lt $end_line ]]; then
            shard_tests+=("$line")
        fi
        (( line_index++ ))
    done
    echo "Test files to consume: ${#shard_tests[@]}"
    NEW_TESTS=$(echo "${shard_tests[*]}" | tr ' ' '|')
    NEW_TESTS=${NEW_TESTS} automation/repeated_test.sh --force
}

main "$@"
