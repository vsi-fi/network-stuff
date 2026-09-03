#!/usr/bin/perl
#Hack to run all-to-all ib_send_bw
#<vsimola@cern.ch> 08-2027
use strict;
use warnings FATAL => 'all';
use Data::Dumper;


my @nodes= (
        "hostname-01",
        "hostname-02",
        "hostname-03",
        "hostname-04",
        "hostname-05",
        "hostname-06",
        "hostname-07",
        "hostname-08",
        "hostname-09",
        "hostname-10",
        "hostname-11",
        "hostname-12",
        "hostname-13",
        "hostname-14",
        "hostname-15",
        "hostname-16",
        "hostname-17",
        "hostname-18",
        "hostname-19",
        "hostname-20",
        "hostname-21",
        "hostname-22",
        "hostname-23",
        "hostname-24",
        "hostname-25",
        "hostname-26",
        "hostname-27",
        "hostname-28",
        "hostname-29",
        "hostname-30",
);

my $sleep = scalar(@nodes/10*5);

my $ib_device = "mlx5_bond_0";
my $host_suffix = "-mon0"; #We have a dedicated ip on the network being tested
my $duration = 300;
my $timeout = $duration + 10; #This is passed as part of the command, as an attempt to make sure ib_send_bw goes away at some point
my $ssh = "/usr/bin/ssh -q -o ConnectTimeout=1 -o StrictHostKeyChecking=no";
my $tos = 136; #This should match the network classifiers (See cma_roce_tos etc)

#Group the commands per node to keep the number of ssh sessions at more reasonable level
my %h_srv_cmds;
my %h_clt_cmds;
foreach my $n (@nodes) {
    my $port = 9000;
    for my $c (0 .. $#nodes) {
        next if $nodes[$c] eq $n;
        # Server runs on $n
        $h_srv_cmds{$n} .= "timeout $timeout ib_send_bw -d $ib_device --tclass=$tos -D $duration -p $port & ";
        $h_clt_cmds{$nodes[$c]} .= "timeout $timeout ib_send_bw -d $ib_device --tclass=$tos -D $duration -p $port $n$host_suffix &";
        ++$port;
    }
}

#Print the commands and quit
if(grep(/-so/, @ARGV)) {
        print(Dumper(%h_srv_cmds));
        print(Dumper(%h_clt_cmds));
        exit(0);
}
print("Starting servers\n");
my @pids;
while (my($node, $cmd) = each (%h_srv_cmds)) {
        my $pid = fork();
        die("Failed to spawn server: $!") unless defined $pid;
        if($pid == 0) {
                exec("$ssh $node \"$cmd\" > /dev/null 2>&1");
                die("Execution failed: $!\n");
        }
        push(@pids, $pid);
}
sleep($sleep);
print("Starting clients\n");
while (my($node, $cmd) = each (%h_clt_cmds)) {
        my $c_pid = fork();
        die("Failed to spawn client: $!") unless defined $c_pid;
        if($c_pid == 0) {
                exec("$ssh $node \"$cmd\" > /dev/null 2>&1");
                die("Execution failed: $!\n");
        }
        push(@pids, $c_pid);
}
for my $pid (@pids) {
        waitpid($pid, 0);
}
