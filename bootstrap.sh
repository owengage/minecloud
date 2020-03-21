#!/bin/bash
aws s3 cp s3://ogage-minecraft/init.sh .
chmod +x init.sh
./init.sh
./run.sh
