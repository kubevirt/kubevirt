#!/bin/bash

PROVIDER=${PROVIDER:-vagrant}
source cluster/$PROVIDER/provider.sh
down
