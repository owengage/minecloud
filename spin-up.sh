#!/bin/bash
set -xe

instance_id=$(aws ec2 run-instances \
    --launch-template LaunchTemplateName=minecraft-server | jq -r .Instances[0].InstanceId)

eip_id=$(python get-eip.py | jq -r .AllocationId)
eip=$(python get-eip.py | jq -r .PublicIp)

# TODO query AWS instead of just pausing.
echo "Sleeping a bit so we can associate the elastic IP with the EC2 instance"
sleep 10

aws ec2 associate-address \
    --instance-id "$instance_id" \
    --allocation-id "$eip_id"

aws ec2 associate-iam-instance-profile \
    --instance-id "$instance_id" \
    --iam-instance-profile Name=minecraft-s3-role


echo $eip > .eip
echo $eip_id > .eip_id
echo $instance_id > .instance_id

# Remove old host key, we know it's changed since we just made the new
# instance.
ssh-keygen -R "$eip"

echo Sleeping for SSH to be available
sleep 15

./remote-bootstrap.sh
