#!/bin/bash

ls -alZ /etc/hosts 
ls -alZ /usr/bin/virt-launcher-cap
mount 
strace $@