#!/bin/bash

ls -iZ /usr/lib /var/lib /opt /lib /lib64
export LD_SHOW_AUXV=1
$@