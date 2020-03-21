#!/bin/bash

# Install our dependencies.
sudo yum install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm
sudo yum-config-manager --enable epel
sudo yum install -y java-1.8.0-openjdk daemonize

aws s3 cp s3://ogage-minecraft/run.sh .
aws s3 cp s3://ogage-minecraft/reupload.sh .
aws s3 cp s3://ogage-minecraft/cliff-side-server.tar.gz .

chmod +x run.sh reupload.sh

echo "extracting server"
tar xf cliff-side-server.tar.gz
rm cliff-side-server.tar.gz

