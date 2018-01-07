#!/bin/bash

PROVIDER=${PROVIDER:-vagrant-kubernetes}
source cluster/$PROVIDER/provider.sh
up
