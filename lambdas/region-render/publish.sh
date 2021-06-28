#!/bin/bash
set -ex
./build-to-zip.sh

aws s3 cp lambda.zip s3://ogage-minecraft/anvil-tiles.zip