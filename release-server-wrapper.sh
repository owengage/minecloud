#!/bin/bash

ACCOUNT=$(aws sts get-caller-identity | jq -r .Account)
REGION=$(aws configure get region)

aws ecr get-login-password --region eu-west-2 | \
    docker login --username AWS --password-stdin $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/minecloud/server-wrapper

docker build -f server-wrapper.Dockerfile -t minecloud/server-wrapper .

docker tag minecloud/server-wrapper:latest $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/minecloud/server-wrapper:latest

docker push $ACCOUNT.dkr.ecr.eu-west-2.amazonaws.com/minecloud/server-wrapper:latest
