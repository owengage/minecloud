#!/bin/bash

eip=$(cat .eip)

cat bootstrap.sh | ssh \
    -oStrictHostKeyChecking=accept-new \
    -i minecraft-key-pair.pem ec2-user@$eip 'bash -'

