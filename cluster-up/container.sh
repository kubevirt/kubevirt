#!/usr/bin/env bash

set -e

if [[ $KUBEVIRT_PROVIDER =~ (ocp|okd).* ]]; then
    CONTAINER=$(docker ps | grep kubevirt | grep $KUBEVIRT_PROVIDER | awk '{print $1}')
    if [ -z $CONTAINER ]; then
        echo "container was not found"
        exit 0
    fi
    docker exec $CONTAINER bash -c "if ! grep "/root/install/auth/kubeconfig" ~/.bashrc > /dev/null; \
                                    then echo export KUBECONFIG=/root/install/auth/kubeconfig >> ~/.bashrc; fi"
    docker exec -it $CONTAINER bash
else
    echo "connect is supported only for ocp / okd, (or KUBEVIRT_PROVIDER isnt exported ?)"
fi
