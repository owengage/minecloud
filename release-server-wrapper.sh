#!/bin/bash

ACCOUNT=$(aws sts get-caller-identity | jq -r .Account)
REGION=eu-west-2
TAG=latest

aws ecr get-login-password --region $REGION | \
    docker login --username AWS --password-stdin $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/minecloud/server-wrapper

docker build -f server-wrapper.Dockerfile -t minecloud/server-wrapper:$TAG .

docker tag minecloud/server-wrapper:$TAG $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/minecloud/server-wrapper:$TAG

docker push $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/minecloud/server-wrapper:$TAG
