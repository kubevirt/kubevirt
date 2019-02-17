#!/bin/bash -e
set -x

main() {
    # This can cause issues if standard-CI impelmentation is not safe
    # (see https://ovirt-jira.atlassian.net/browse/OVIRT-992)
    rm -rf exported-artifacts
    mkdir exported-artifacts

    local netns_dir="/var/run/netns"

    local before_ns_c=0
    before_ns_c=$(find "$netns_dir"/ -type f | wc -l)
    echo "Seeing $before_ns_c network namespaces before starting container"

    docker run -d --rm --name ns_test centos:7 sleep inf
    if docker ps -q | grep -E '.+'; then
        echo "Container is UP"
        local res=0
        local during_ns_c=0

        during_ns_c=$(find "$netns_dir"/ -type f | wc -l)
        echo "Seeing $during_ns_c network namespaces with running container"
        if (( during_ns_c <= before_ns_c )); then
            echo Number of namespaces did not increase
            let ++res
        else
            echo Number of namespaces increased as expected
        fi
        docker kill ns_test
        if docker ps -q | grep -E '.+'; then
            echo Container not dead after kill command
            let ++res
        else
            local after_ns_c

            sleep 60 # wait a while to let NS get deleted
            after_ns_c=$(find "$netns_dir"/ -type f | wc -l)
            echo "Seeing $after_ns_c network namespaces when no container"
            if (( after_ns_c != before_ns_c )); then
                echo Number of namespaces did not decrease
                let ++res
            else
                echo Number of namespaces decreased as expected
            fi
        fi
        return $res
    else
        echo Container did not start
        return 1
    fi
}

main "$@"
