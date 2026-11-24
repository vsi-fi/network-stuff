#!/usr/bin/perl
#Check for arp fluctuations
use strict;
use warnings FATAL => 'all';

my (@target,%r);
my $v = 0;

my $c = 0;
while($c < scalar(@ARGV)) {
	if($ARGV[$c] =~ /^-v$/) {
		$v = 1;
		vprint("Setting verbose mode to 1\n");
	}
	if($ARGV[$c] eq "-t") {
		if(!$ARGV[$c+1] || $ARGV[$c+1] !~ /\d+\./) {
			print("I need a list,of,ip-addresses after -t\n");
			exit(1);
		}
		else {
			@target = split(/\,/, $ARGV[$c+1]);
		}
	}
	$c++;
}
sub vprint {
	my $m = $_[0];
	if($v > 0) {
		print($m);
	}
}

if(!@target) {
	print("Usage: $0 [-v] -t list,of,ip-address,to,arp\n");
	print("Example: $0 -v -t 10.193.186.240,10.193.186.241\n");
	exit(1);
}

while(1) {
	foreach my $t (@target) {
		#print("sudo arp -d $t\n");
		`sudo arp -d $t`; $? and die "Failed to delete arp for $t\n";
		if($v > 0) {
			foreach(`arp $t`) {vprint($_);}
		}
		if(int(rand(2)) ) {
			vprint("ping -w 1 -c 1 -t $t\n");
			foreach(`ping -c 1 $t`) {
				vprint($_);
			}	
		}
		else {
			vprint("nc $t 9123\n");
			foreach(`nc $t 9123`) { vprint($_); }
		}	
		foreach(`arp -an`) {
			if(/^.*?$t.*?at\s(.*?)\s/) {
				if(!$r{$t}) {
					$r{$t} = $1;
					print("Set initial arp for $t to $1\n");
				}
				else {
					if($r{$t} ne $1) {
						print"!!! ARP response changed from $r{$t} to $1\n";
					}
					else {
						print("Same ARP response from $1 for $t\n");
					}
				}
			}
		}
	}
	sleep(1);
}
