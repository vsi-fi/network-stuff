#!/usr/bin/perl
#Print commands to configure dot1p PFC on nvidia cards
#verify with ip -d link show dev $logical_interface 
use strict;
use warnings;

my $host_prefix = "test-pc-";
my $logical_device = "vlan114";
my $priority = 4;
my $physical_device = "ens1f0np0";
my $ib_device = "mlx5_bond_0";
my $tos = 104;
my $pfc_string = join ',', map{$_ == $priority?1:0}0 .. 7; 
my $via_ssh = 0;
my @cmd = (
	"ip link set dev $logical_device type vlan egress-qos-map 0:$priority 1:$priority 2:$priority 3:$priority 4:$priority 5:$priority 6:$priority 7:$priority",
	"ip link set dev $logical_device type vlan ingress-qos-map 0:$priority 1:$priority 2:$priority 3:$priority 4:$priority 5:$priority 6:$priority 7:$priority",
	"mlnx_qos -i $physical_device --pfc $pfc_string --trust=pcp",
	"cma_roce_tos -d $ib_device -t $tos"
);

my $n = 1;
my $m = 10;
if($via_ssh == 0) { $n=$m;}
while($n <= $m) {
	$n = sprintf("%03d", $n);
	foreach my $c (@cmd) {
		my $ssh = "";
		my $quote = "";
		if($via_ssh == 1) { $ssh = "ssh root\@$host_prefix$n"; $quote = "\"";}
		print("$ssh $quote$c$quote\n");
	}
	$n++;
}
