#!/bin/bash

hosts=(
    "user1@host1"
    "user2@host2"
    "user3@host3"
    "user4@host4"
    "user5@host5"
    "user6@host6"
    "user7@host7"
    "user8@host8"
)

for host in "${hosts[@]}"; do
    echo "Starting iperf3 server on $host..."
    ssh -o StrictHostKeyChecking=no "$host" 'nohup iperf3 -s > /dev/null 2>&1 &' &
done

wait

echo "iperf3 servers started on all machines."
