#!/bin/bash
set -ex

# Stop CLI using pager (which requires user input)
export AWS_PAGER=""
S3_ARGS='--s3-bucket ogage-minecraft --s3-key lambda-singleton.zip'

go build lambdas/singleton/main.go
zip lambda-singleton.zip main

aws s3 cp lambda-singleton.zip s3://ogage-minecraft/lambda-singleton.zip
aws lambda update-function-code --function-name MinecloudSingleton $S3_ARGS
aws lambda update-function-code --function-name MinecloudAlphaBananaUp $S3_ARGS
aws lambda update-function-code --function-name MinecloudAlphaBananaDown $S3_ARGS

rm lambda-singleton.zip
rm main
