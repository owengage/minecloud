#!/bin/bash
set -e

go build lambdas/command/main.go
zip lambda-command.zip main
aws lambda update-function-code --function-name MinecraftCommand --zip-file fileb://lambda-command.zip
rm lambda-command.zip
rm main

