#!/bin/bash
set -e

go build lambdas/singleton/main.go
zip lambda-singleton.zip main
aws lambda update-function-code --function-name MinecloudSingleton --zip-file fileb://lambda-singleton.zip
rm lambda-singleton.zip
rm main
