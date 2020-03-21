#!/bin/bash
set -ex

instance_id=$(cat .instance_id)
eip=$(cat .eip)
eip_id=$(cat .eip_id)

cat <<EOF |  ssh -i minecraft-key-pair.pem ec2-user@$eip
pkill -f "server.jar"
./reupload.sh
EOF


#aws ec2 terminate-instances --instance-id "$instance_id"
