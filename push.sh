#!/bin/bash

registry_port=$(./cluster-up/cli.sh ports registry | tr -d '\r')
registry=localhost:$registry_port
docker build -t $registry/usb-disk-hook123:devel .
docker push $registry/usb-disk-hook123:devel

