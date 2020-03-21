#!/bin/bash

eip=$(cat .eip)

echo "cat stdout.log" | ssh \
    -i minecraft-key-pair.pem ec2-user@$eip 'bash -'

