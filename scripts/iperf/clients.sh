#!/bin/bash

servers=(
    "host1"
    "host2"
    "host3"
    "host4"
    "host5"
    "host6"
    "host7"
    "host8"
)

duration=10
p=3
port=5201

for server in "${servers[@]}"; do
    echo "Starting iperf3 client to $server..."
    iperf3 -c "$server" -t $duration -P 3 -p $p > "iperf_client_$server.log" 2>&1 &
done

wait

echo "iperf3 client tests completed for all servers."
