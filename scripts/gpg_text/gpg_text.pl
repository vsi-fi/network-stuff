#!/usr/bin/perl
#Thingy to encrypt/decrypt short messages using preshared keyfile
#keys should be stored in a file in a "$id_number $key_string" -format. Use $0 -mode genkeys to generate such key file
#To encrypt a string: $0 -mode encrypt -key_file keyfile3 -keyid 20 -message ThisIsAmazing
#To decrypt a string encrypted as above: $0 -mode decrypt -key_file keyfile3 -keyid 20 -message $output_from_above
#vsi@kapsi.fi/2025
use strict;
use warnings FATAL => 'all';

my @mandatory = ("mode", "key_file", "message", "keyid");
my %args;
my %keys;
my $ac = 0;
while($ac < scalar(@ARGV)) {
	foreach(@mandatory) {
		if($ARGV[$ac] =~ /-$_/ and $ARGV[$ac+1]) {
			$args{$_} = $ARGV[$ac+1];
		}
	}
	$ac++;
}
if($args{'mode'} and $args{'mode'} ne "genkeys") {
	foreach(@mandatory) {
		if(!$args{$_}) {
			die("Missing argument: \"$_\"\n");
		}
	}
}
if($args{'mode'} !~ /^(?:en|de)crypt$|^genkeys$/) {
	die "-mode has to be (en|de)crypt|genkeys";
}

if($args{'mode'} eq "genkeys") {
	my $keys = 0;
	while($keys <= 1024) {
		my $key = "";
		print"$keys ",generate_filename(),"\n";
		$keys++;
	}
	exit(0);
}
open(KEY_FILE, $args{"key_file"}) or die "Failed to open key file \"$args{'key_file'}\" for reading!\nYou can generate suitable file with $0 -mode genkeys > keyfile\n\n";
while(<KEY_FILE>) {
	if(/^(\d+)\s(.*?)$/) {
		$keys{$1} = $2;
	}
}
close(KEY_FILE);
if(scalar(%keys) < 1) {
	die("Seems that i failed to parse any preshared keys from \"$args{'key_file'}\". I am expecting format ^(\\d+)\\s(.*?)\$ where the first match is the key id and second is the key itself\n");
}
secrets($args{'mode'}, $args{'message'}, $args{'keyid'});
sub generate_filename {
	my @chars = ('a'..'z', 'A'..'Z');
	my $filename = "";
	$filename .= $chars[rand @chars] for 1..32;
	return($filename);
}
sub secrets {
	my ($d,$input,$id) = @_;
	my $output;
	my $filename = generate_filename();
	open(KEY,">/tmp/$filename");
	chmod(0600, "/tmp/$filename");
	print(KEY $keys{$id});
	if($d eq "encrypt") {
		$output = `echo "$input" | gpg --symmetric --cipher-algo AES256 --batch --yes --passphrase-file /tmp/$filename --pinentry-mode loopback --output - | base64 -w 0`;
	}
	else {
		$output = `echo "$input" | base64 -d | gpg --decrypt --batch --yes --passphrase-file /tmp/$filename --pinentry-mode loopback --output -`;
	}
	close(KEY);
	unlink("/tmp/$filename");
	print("Message after $args{'mode'}: $output\n");
}
