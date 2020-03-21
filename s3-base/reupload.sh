#!/bin/bash

# Package server
tar czvf cliff-side-server.tar.gz cliff-side-server

# Upload
aws s3 cp cliff-side-server.tar.gz s3://ogage-minecraft/cliff-side-server.tar.gz

