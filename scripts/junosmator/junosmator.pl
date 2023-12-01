#!/usr/bin/perl

#Junosmator - Run commands to multiple junos boxes 
# vsimola / CERN 2021

use Data::Dumper;
use POSIX;
use strict;
use warnings FATAL => 'all';

#If this file exists this program should die instead of forking executors
my $killfile = "/tmp/no_fork_for_junosmate";

#parse arguments
my $c = 0;
my $verbose = 0;
my %params;
while($c < @ARGV) {
    if($ARGV[$c] =~ /^(?:-i|--inventory)$/ && -r $ARGV[$c+1]) {
         $params{'inventory'} = $ARGV[$c+1];
    }
    if($ARGV[$c] =~ /^(?:-t|--template)$/) {
        if(-r $ARGV[$c+1]) {
            $params{'template'} = $ARGV[$c+1];
        }
        else {
            usage("Offered inventory file ($ARGV[$c+1]) was not found or it is not readable");
        }
    }
    if($ARGV[$c] eq "-v") { $verbose = 1; }
    if($ARGV[$c] eq "-noping") { $params{'noping'} = 1; }
    if($ARGV[$c] =~ /^(?:-f|--filter)$/ and $ARGV[$c+1]) {
         $params{'filter'} = $ARGV[$c+1];
    }
    if($ARGV[$c] =~ /^(?:-m|--mode)$/) {
        if($ARGV[$c+1] =~ /^(?:dry_run|real_deal)$/) {
            $params{'type'} = $ARGV[$c+1];
        }
        else {
            usage("Offered mode \"$ARGV[$c+1]\" was not recognized. I can only deal with \"dry_run\" and \"real_deal\"");
        }
    } 
    if($ARGV[$c] =~ /^(?:-c|--cmd)$/ and $ARGV[$c+1]) {
        $params{'cmd'} = $ARGV[$c+1];
    } 
    if($ARGV[$c] =~ /^(?:-sh|--single-host)$/ and $ARGV[$c+1]) {
        $params{'single_host'} = $ARGV[$c+1];
    } 
    if($ARGV[$c] =~ /^(?:-sh|--show-hosts)$/) {
        $params{'show_targets_only'} = 1;
    } 
    $c++;
}

#If no command or template is given, do nothing
if(!$params{'template'} && !$params{'cmd'} && !$params{'show_targets_only'}) {
    usage("Both template (-t) and command (-c) are missing.");
    exit(1);
}

#Check to see if ssh-agent is setup
if(!$ENV{SSH_AUTH_SOCK} or !$ENV{SSH_AGENT_PID}) {
    print"It seems that ssh-agent is not configured. This script needs it.\n\nIssue: ssh-agent to get the commands to set it up.\n";
    print"Also, remember to add your keys using ssh-add -command\n";
    exit(1);
}

#If no filter is given take it as a sign that we do not want to do anything
if(!$params{'filter'} and !$params{'single_host'}) {
    $params{'filter'} = "dsaldjaskdSDs8a9d8dash";
}
if(!$params{'filter'} and $params{'single_host'}) {
    $params{'filter'} = "";
}

my @mandatory_args = ("inventory");
if(!$params{'inventory'}) {
    usage("Inventory file missing");
}
if($params{'show_targets_only'}) {
    show_targets($params{'filter'});
    exit(0);
}

sub vprint {
    if($verbose > 0) {
        print"$_[0]\n";
    }
}

sub usage {
    if($_[0]) {
        print"\nFAIL: $_[0]\n\n";
    }
    print"Prequisites:\nTo use this tool in a sane fashion you need to have your ssh-key distributed to the Junipers.\nIf this is ok, you then need to use ssh-agent + ssh-add to authenticate to the devices\n";
    print"e.g. execute: \"ssh-agent\" - copy-paste the resulted output to your terminal and then add your key \"ssh-add\"\n\n";
    print"\nActual usage:\n";
    print"To run a random command on a set of devices: $0 -i|--inventory \$file_containing_inventory_of_devices -c|--cmd \"\$junos_command; \$followed_by_maybe_another_one\"\n"; 
    print"To run a random command on a set of devices: $0 -i|--inventory \$file_containing_inventory_of_devices -f|--filter \$string_to_filter_target_groups_for -c|--cmd \"\$junos_command; \$followed_by_maybe_another_one\"\n"; 
    print"To get a list of target devices and bail out: $0 -i|--inventory \$file_containing_inventory_of_devices -sh|--show-hosts -f .\n"; 
    print"To run a set of set-cmds in dry run mode: $0 -i|--inventory \$file_containing_inventory_of_devices -m|--mode dry_run -t|--template \$file_containing_the_commands\n"; 
    print"To run a set of set-cmds and actually commit: $0 -i|--inventory \$file_containing_inventory_of_devices -t|--type real_deal -t|--template \$file_containing_the_commands\n"; 
    print"Example: To execute commands in file \"deactivate_netadmin\", display changes and rollback: $0 -i inventory -f ctrl -m dry_run -t deactivate_netadmin\n";
    print"Example: To check version of control switches: $0 -i inventory -c 'show version | match \\\"EX  Software Suite\\\"' -f \$device_category_filter\n";  
    print"Example: To check version of a single switche: $0 -i inventory -sh sw-leaf-01 -c 'show version'\n";  
    print"Example: To commit a set of of commands to a single host: perl $0 -i inventory -sh sw-stuff-73 -t sw-stuff-template-2021-06-02 -m real_deal\n";
    print"Example: To change something on a single switch: perl $0 -i inventory -sh sw-access-01 -m dry_run -c \"configure; set system domain-name someotherdomain.org; show | compare; rollback;\"\n";
    exit(1);
}

sub show_targets {
  my $targets = parse_inventory($params{'inventory'});
  print"\n#!#!- The targets listed below (if any) were selected #!#!\n\n";
  if(!$params{'filter'}) { $params{'filter'} = ""; }
  while(my($k,$v) = each(%{$targets})) {
      if($k =~ /$params{'filter'}/i) {
          foreach(@{$v}) {
              print"$_\n";
          }
      }
    }
}


my $set_file = $params{'template'};
my $filter = $params{'filter'};
my $mode = $params{'type'};
my $random_cmd = $params{'cmd'};

my ($scp,$ssh) = ("/usr/bin/scp -oStrictHostKeyChecking=no","/usr/bin/ssh -oStrictHostKeyChecking=no -oConnectTimeout=5");
#command files should end up here on the junipers
my $dst_path = "/var/tmp";

#generate the log prefix
my $log_prefix = "";

if($verbose > 0) {
    print"Verbose mode set to on\n";
    sleep(2);
}


my $inventory = parse_inventory($params{'inventory'});
my @targets = @{select_hosts()};
my (%uniq_targets, @uniq_t);
foreach(@targets) {
    if(!$uniq_targets{$_}) {
        $uniq_targets{$_} = 1;
        push(@uniq_t, $_);
    }
}
@targets = @uniq_t;
show_targets($params{'filter'});
if(!$params{'noping'}) {
    print"Testing with ping if the hosts are reachable - you can skip this with -noping\n";
    foreach(@targets) {
        test_ping($_);
    }
}
sleep(3);


my @children;
my @started;

foreach my $dev (@targets) {
    if(-e $killfile) {
        die("Killfile ($killfile) was found - bailing out now\n");
    }
    if(grep(/^$dev$/,@started)) {
        warn("I dont like to attempt several times against same node ( $dev - only one job will run against $dev )\n"); 
        next;
    }
    push(@started,$dev);
    my $pid = fork();
    if($pid) {
        push(@children,$pid);
    }
    elsif($pid == 0) {
        load_cfg($mode,$dev,$set_file,$random_cmd);
        exit(0); 
    }
}
foreach(@children) {
        waitpid($_,0);
        }
sub test_ping {
    my $host = $_[0];
    my $ping = "/usr/bin/ping -c 2 -i 0.3 -w 1 "; #knobs used to test if the target is reachable (2 pings, 1 second timeout, interval of 0.3 seconds)
    my $state = 1;
    foreach(`$ping $host`) {
        if(/^.*?\,\s0%\spacket\sloss/) {
            $state = 0;
        }
    }
    if($state == 1) {
        warn("It seems I am not getting a icmp ping reply fromn $host. You might want to ctrl+c now and investigate.\n\n");
        sleep(5);
    }
}


sub parse_inventory {
    open(INVENTORY,"$_[0]") or die "$0::parse_inventory - failed to read the suggested inventory file $_[0]!\n";
    my $target_group;
    my $target_group_members;
  
    if($params{'single_host'}) {
        $target_group_members->{'Single host'} = [];
        while(<INVENTORY>) {
            if(/^$params{'single_host'}$/) {
                push(@{$target_group_members->{'Single host'}},$params{'single_host'});
                return($target_group_members);
            }
        }
    die("Failed to find $params{'single_host'} form the inventory!\n");
    }
    while(<INVENTORY>) {
        if(/\[(.*?)\]/) {
           $target_group = $1;
        }
        if($target_group && $_ =~ /^(\w.*?)$/ && !/^\[/) {
            if(!$target_group_members->{$target_group}) {
                $target_group_members->{$target_group} = [];
            }
            push(@{$target_group_members->{$target_group}},$1);
        }
    }
    return($target_group_members);
}

sub select_hosts {
    while(my($group,$members) = each(%{$inventory})) {
        if($group =~ /$filter/i) {
            foreach(@{$members}) {
                push(@targets,$_);
            }
        }
    }
    return(\@targets);
}


sub load_cfg {
    my ($mode,$dev,$set,$random_cmd) = @_;
    my @confirm;
    if($random_cmd) {
        vprint("EXECUTING $ssh $dev \"$random_cmd\"\n");
        my @output = `$ssh $dev \"$random_cmd\"`; $? and die "Failed while trying to execute: $ssh $dev \"$random_cmd\"";
        print"#!#! - Output from $dev ($random_cmd) - #!#!\n";
        foreach(@output) {
            print"$dev: $_";
            if(/error|fail|problem|bad/i) {
                print"\n\n*** PROBLEM REPORTED BY $dev: $_";
            }
        }
        print"\n";
        return(0);
    }

    print"Trying to apply config to $dev from $set -file in mode $mode\n";
    sleep(3);
    vprint("EXECUTING: $scp $set $dev:$dst_path/.\n");
    print "$scp $set $dev:$dst_path/.\n";
    my @output = `$scp $set $dev:$dst_path/.`; $? and die "Failed while trying to execute: $scp $set $dev:$dst_path/.\n";
    my @set_clean = split(/\//,$set);
    $set = $set_clean[scalar(@set_clean)-1];
    if($mode eq "dry_run") {
        vprint("EXECUTING: $ssh $dev \"configure; load set $dst_path/$set; show | compare; commit check; rollback; exit\"\n");
        print "$ssh $dev \"configure; load set $dst_path/$set; show | compare; commit check; rollback; exit\"\n";
        @output = `$ssh $dev \"configure; load set $dst_path/$set; show | compare; commit check; rollback; exit\"`;
        foreach(@output) {
            print "$dev: $_";
        }
    }
    #This is the real deal
    if($mode eq "real_deal") {
        print"\nTrying to apply config to $dev from $set..\n";
        print "$ssh $dev \"configure; load set $dst_path/$set; commit confirmed and-quit\"\n";
        @output = `$ssh $dev \"configure; load set $dst_path/$set; show|compare; commit confirmed and-quit\"`;
        foreach(@output) { print; }
        push(@confirm,$dev);
    }
    if($confirm[0]) {
        foreach my $switch (@confirm) {
            print"\nI will now attempt to confirm the changes on $switch\n";
            print"ssh $switch \"configure; commit check and-quit\"\n";
            my @confirm_output = `ssh $switch \"configure; commit check and-quit\"`;
            foreach(@confirm_output) { print; }
        }
    }
}



