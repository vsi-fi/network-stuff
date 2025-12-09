#!/usr/bin/perl
#collect some config information for nvidia support
use strict;
use warnings FATAL => 'all';

my @pcids;
foreach(`lsmod`) {
	if(/(^.*?(?:mlx|roce|rdma).*?)\s{2,}/) {
		print("\n\nKernel module: $1\n");
		system("modinfo $1");;
	}
}
unlink("/tmp/mellanox_info.txt");
foreach(`lspci`) {
	if(/^(.*?)\s+.*?(?:nvidia|mellanox)/i) {
		system("echo mlxconfig -d $1  q >> /tmp/mellanox_info.txt");	
		system("mlxconfig -d $1 q | tee -a /tmp/mellanox_info.txt");
	}
}
foreach my $c ("mlxfwmanager", "cat /proc/net/bonding/bond2", "cat /proc/cpuinfo", "lspci -vvv", "uname -a", "ibdev2netdev", "ifconfig -a") {
	system("echo $c >> /tmp/mellanox_info.txt");
	`$c | tee -a /tmp/mellanox_info.txt`;$? and die "Failed while trying to execute: $c :: $! $?\n";
}
