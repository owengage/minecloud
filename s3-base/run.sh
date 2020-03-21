#!/bin/bash

memfree_line=$(cat /proc/meminfo | grep 'MemFree:')
memfree_unit=$(echo "$memfree_line" | awk '{print $3}')
memfree=$(echo "$memfree_line" |awk '{print $2}')

if [[ "$memfree_unit" == "kB" ]]
then
    echo "sanity check: unit for meminfo was kB"
else
    echo "sanity check: unit for meminfo was not kB"
    echo "was \"$memfree_unit\""
    exit 1
fi

let mem_to_use_kb=$memfree*90/100 # use percentage of memory available.
let mem_to_use_formatted=mem_to_use_kb/1024
mem_to_use_formatted=${mem_to_use_formatted}M

echo using $mem_to_use_formatted for JVM

java=$(which java)

# Running script directly via SSH means we need to add the location of
# daemonize to our path.
PATH=$PATH:/usr/sbin

daemonize \
    -a \
    -c $HOME/cliff-side-server \
    -e $HOME/stderr.log \
    -o $HOME/stdout.log \
    -p $HOME/daemon.pid \
    -l $HOME/daemon.lock \
    $java "-Xms${mem_to_use_formatted}" "-Xmx${mem_to_use_formatted}" -jar server.jar
