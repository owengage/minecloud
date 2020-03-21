#!/bin/bash
set -e

function upload_failed {
    echo upload failed
    echo not terminating EC2 instance.
    exit 1
}

instance_id=$(cat .instance_id)
eip=$(cat .eip)
eip_id=$(cat .eip_id)

cat <<EOF |  ssh -i minecraft-key-pair.pem ec2-user@$eip || upload_failed
set -e
echo killing
pkill -f "server.jar" || true
echo reuploading 
./reupload.sh
EOF

echo terminating EC2 instance
aws ec2 terminate-instances --instance-id "$instance_id"
