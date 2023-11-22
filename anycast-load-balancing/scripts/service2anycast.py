#!/usr/bin/python3
#Trivial thing that checks if a dns resolver is showing signs of life.
#if not, this tries to disable the protocol advertisements using birdc
#
#example usage:
#__file__ $service_address $some_a_record $name_of_the_bgp_session_in_bird.conf
#this could be adjusted to work with some monitoring process to run periodically
#vsi@kapsi.fi


import os, sys, time
import subprocess, re 

server = sys.argv[1]
target = sys.argv[2]
protocol = sys.argv[3]

def dns_check(server, target):
    try:
        res = subprocess.run(['nslookup', '-timeout=1', target, server], capture_output=False, check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, shell=False)

        #If nslookup didn't die whilst bgp session is disabled, assume it is safe to enable BGP 
        if get_protocol_state(protocol) == 1:
            protocol_action(protocol, "enable")
    except:
        protocol_action(protocol, "disable") 


def protocol_action(protocol,action):
    res = subprocess.run(['birdc', action, protocol], capture_output=False, check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, shell=False)

def get_protocol_state(protocol):
    res = subprocess.run(['birdc', 'show', 'protocols', protocol], stdout=subprocess.PIPE)
    match = "^"+protocol+"\s+BGP\s+"
    for r in res.stdout.decode('utf-8').split('\n'):
        if re.match(match, r):
            if re.split('\s+', r)[3] != "up":
                return(1)
            else:
                return(0)
while(1):
    dns_check(server, target)
    #for testing, run every three seconds
    time.sleep(3)

