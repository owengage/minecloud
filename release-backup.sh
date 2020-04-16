#!/bin/bash
set -e

go build lambdas/backup/main.go
zip lambda-backup.zip main
aws lambda update-function-code --function-name MinecloudBackup --zip-file fileb://lambda-backup.zip
rm lambda-backup.zip
rm main

